package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
)

func TestJSONOutputIsMachineOnlyAndRedacted(t *testing.T) {
	var out, errOut bytes.Buffer
	r := NewRenderer(&out, &errOut, FormatJSON, TerminalCapabilities{StdoutTTY: false, Color: false, Width: 80}, redact.New())
	r.Redactor().Register(redact.GitHubToken, "secret-token")

	if err := r.JSON(map[string]any{"ok": true, "command": "test", "token": "secret-token"}); err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	got := out.String()
	if !strings.HasPrefix(got, "{") {
		t.Fatalf("JSON output should start with object: %q", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("JSON output contains ANSI: %q", got)
	}
	if strings.Contains(got, "secret-token") {
		t.Fatalf("JSON output leaked secret: %q", got)
	}
	if !strings.Contains(got, `"redactions_applied":true`) {
		t.Fatalf("JSON output missing redactions flag: %q", got)
	}
	if errOut.Len() != 0 {
		t.Fatalf("JSON wrote to stderr: %q", errOut.String())
	}
}

func TestHumanStepGlyphsAndASCIIFallbacks(t *testing.T) {
	var unicodeOut bytes.Buffer
	unicodeRenderer := NewRenderer(&unicodeOut, &bytes.Buffer{}, FormatHuman, TerminalCapabilities{StdoutTTY: true, Color: false, Width: 80}, redact.New())
	if err := unicodeRenderer.Step(1, 1, "Welcome", Success("ready"), WarningLine("risk"), ErrorLine("blocked"), PromptLine("question"), Next("fix"), Bullet("item")); err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	for _, want := range []string{"✓", "!", "✗", "?", "→", "•"} {
		if !strings.Contains(unicodeOut.String(), want) {
			t.Fatalf("unicode output missing %q: %s", want, unicodeOut.String())
		}
	}

	var asciiOut bytes.Buffer
	asciiRenderer := NewRenderer(&asciiOut, &bytes.Buffer{}, FormatHuman, TerminalCapabilities{StdoutTTY: false, ASCII: true, Color: false, Width: 80}, redact.New())
	if err := asciiRenderer.Step(1, 1, "Welcome", Success("ready"), WarningLine("risk"), ErrorLine("blocked"), PromptLine("question"), Next("fix"), Bullet("item")); err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	for _, want := range []string{"OK", "WARNING", "ERROR", "PROMPT", "NEXT", "-"} {
		if !strings.Contains(asciiOut.String(), want) {
			t.Fatalf("ascii output missing %q: %s", want, asciiOut.String())
		}
	}
}
