package ui

import (
	"fmt"
	"strings"
)

// BoxedCommand is structured data for JSON output (machine-parseable).
type BoxedCommand struct {
	Host    string `json:"host"`
	Command string `json:"command"`
	Why     string `json:"why,omitempty"`
}

// RenderBoxed formats a copy-paste command for a remote host.
// useUnicode selects light border characters; otherwise ASCII +/-/| is used.
// width is the terminal content width (from TerminalCapabilities); must be >= 20.
func RenderBoxed(host, cmd, why string, useUnicode bool, width int) string {
	if width < 20 {
		width = 20
	}
	var topL, topR, botL, botR, horiz, vert string
	if useUnicode {
		topL, topR, botL, botR = "┌", "┐", "└", "┘"
		horiz, vert = "─", "│"
	} else {
		topL, topR, botL, botR = "+", "+", "+", "+"
		horiz, vert = "-", "|"
	}
	var b strings.Builder
	if host != "" {
		fmt.Fprintf(&b, "Copy and paste this on %s:\n\n", host)
	} else {
		fmt.Fprintf(&b, "Copy and paste this:\n\n")
	}
	if strings.TrimSpace(why) != "" {
		fmt.Fprintf(&b, "%s\n\n", why)
	}
	innerMax := width - 4 // borders + spaces
	if innerMax < 12 {
		innerMax = 12
	}
	lines := wrapBoxCommand(cmd, innerMax)
	borderW := innerMax + 2
	top := topL + strings.Repeat(horiz, borderW) + topR
	bot := botL + strings.Repeat(horiz, borderW) + botR
	b.WriteString(top)
	b.WriteByte('\n')
	for _, line := range lines {
		pad := innerMax - displayLen(line)
		if pad < 0 {
			pad = 0
		}
		fmt.Fprintf(&b, "%s %s%s %s\n", vert, line, strings.Repeat(" ", pad), vert)
	}
	b.WriteString(bot)
	return b.String()
}

func displayLen(s string) int {
	// Runes approximated as bytes for ASCII commands; good enough for runnerkit CLIs.
	return len(s)
}

func wrapBoxCommand(cmd string, innerMax int) []string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return []string{""}
	}
	var out []string
	words := strings.Fields(cmd)
	if len(words) == 0 {
		return []string{""}
	}
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) <= innerMax {
			cur += " " + w
			continue
		}
		out = append(out, cur)
		cur = w
	}
	out = append(out, cur)
	return out
}
