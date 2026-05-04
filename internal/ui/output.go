package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
)

type Format string

const (
	FormatHuman Format = "human"
	FormatJSON  Format = "json"
)

// TerminalCapabilities describes the terminal features available to the CLI.
type TerminalCapabilities struct {
	StdinTTY  bool
	StdoutTTY bool
	Color     bool
	ASCII     bool
	Width     int
}

type LineKind string

const (
	LineSuccess LineKind = "success"
	LineWarning LineKind = "warning"
	LineError   LineKind = "error"
	LinePrompt  LineKind = "prompt"
	LineNext    LineKind = "next"
	LineBullet  LineKind = "bullet"
)

type Line struct {
	Kind LineKind
	Text string
}

type Renderer struct {
	out      io.Writer
	err      io.Writer
	format   Format
	caps     TerminalCapabilities
	redactor *redact.Redactor
}

func NewRenderer(out io.Writer, errOut io.Writer, format Format, caps TerminalCapabilities, redactor *redact.Redactor) *Renderer {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}
	if format == "" {
		format = FormatHuman
	}
	if caps.Width == 0 {
		caps.Width = 80
	}
	if redactor == nil {
		redactor = redact.New()
	}
	return &Renderer{out: out, err: errOut, format: format, caps: caps, redactor: redactor}
}

func (r *Renderer) Redactor() *redact.Redactor { return r.redactor }

func Success(text string) Line     { return Line{Kind: LineSuccess, Text: text} }
func WarningLine(text string) Line { return Line{Kind: LineWarning, Text: text} }
func ErrorLine(text string) Line   { return Line{Kind: LineError, Text: text} }
func PromptLine(text string) Line  { return Line{Kind: LinePrompt, Text: text} }
func Next(text string) Line        { return Line{Kind: LineNext, Text: text} }
func Bullet(text string) Line      { return Line{Kind: LineBullet, Text: text} }

func (r *Renderer) Step(current, total int, title string, lines ...Line) error {
	if r.format == FormatJSON {
		return r.JSON(map[string]any{"ok": true, "step": current, "total_steps": total, "title": title})
	}
	if _, err := fmt.Fprintf(r.out, "Step %d of %d: %s\n", current, total, r.clean(title)); err != nil {
		return err
	}
	for _, line := range lines {
		if err := r.writeWrapped(r.out, r.glyph(line.Kind), line.Text); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(r.out)
	return err
}

func (r *Renderer) Warning(title string, body []string, next string) error {
	if r.format == FormatJSON {
		return r.JSON(map[string]any{"ok": true, "warning": title, "body": body, "next": next})
	}
	if err := r.writeWrapped(r.err, r.glyph(LineWarning), title); err != nil {
		return err
	}
	for _, line := range body {
		if err := r.writeWrapped(r.err, " ", line); err != nil {
			return err
		}
	}
	if next != "" {
		if err := r.writeWrapped(r.err, r.glyph(LineNext), next); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(r.err)
	return err
}

func (r *Renderer) Error(code string, message string, remediation []string) error {
	if r.format == FormatJSON {
		return r.JSON(map[string]any{
			"ok": false,
			"error": map[string]any{
				"code":        code,
				"message":     message,
				"remediation": remediation,
			},
		})
	}
	if err := r.writeWrapped(r.err, r.glyph(LineError), message); err != nil {
		return err
	}
	for _, line := range remediation {
		if err := r.writeWrapped(r.err, r.glyph(LineNext), line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(r.err)
	return err
}

func (r *Renderer) JSON(v any) error {
	payload, err := objectWithRedactionsFlag(v)
	if err != nil {
		return err
	}
	encoded, err := marshalNoEscape(payload)
	if err != nil {
		return err
	}
	encoded = r.redactor.JSONBytes(encoded)
	_, err = r.out.Write(append(encoded, '\n'))
	return err
}

func objectWithRedactionsFlag(v any) (any, error) {
	encoded, err := marshalNoEscape(v)
	if err != nil {
		return nil, err
	}
	var payload any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return nil, err
	}
	if object, ok := payload.(map[string]any); ok {
		object["redactions_applied"] = true
		return object, nil
	}
	return map[string]any{"ok": true, "value": payload, "redactions_applied": true}, nil
}

func (r *Renderer) writeWrapped(w io.Writer, prefix string, text string) error {
	text = r.clean(text)
	width := r.wrapWidth()
	linePrefix := prefix
	if strings.TrimSpace(linePrefix) != "" {
		linePrefix += " "
	}
	if strings.HasPrefix(text, "runs-on:") {
		_, err := fmt.Fprintf(w, "%s%s\n", linePrefix, text)
		return err
	}
	for i, line := range wrapText(text, width-len(linePrefix)) {
		if i == 0 {
			if _, err := fmt.Fprintf(w, "%s%s\n", linePrefix, line); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "%s%s\n", strings.Repeat(" ", len(linePrefix)), line); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) clean(text string) string {
	return r.redactor.String(text)
}

func (r *Renderer) wrapWidth() int {
	width := r.caps.Width
	if width <= 0 {
		width = 80
	}
	if width > 100 {
		width = 100
	}
	if width < 20 {
		width = 20
	}
	return width
}

func (r *Renderer) glyph(kind LineKind) string {
	if r.caps.ASCII {
		switch kind {
		case LineSuccess:
			return "OK"
		case LineWarning:
			return "WARNING"
		case LineError:
			return "ERROR"
		case LinePrompt:
			return "PROMPT"
		case LineNext:
			return "NEXT"
		case LineBullet:
			return "-"
		default:
			return "-"
		}
	}
	switch kind {
	case LineSuccess:
		return "✓"
	case LineWarning:
		return "!"
	case LineError:
		return "✗"
	case LinePrompt:
		return "?"
	case LineNext:
		return "→"
	case LineBullet:
		return "•"
	default:
		return "•"
	}
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		width = 64
	}
	if text == "" {
		return []string{""}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	lines = append(lines, current)
	return lines
}

func marshalNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}
