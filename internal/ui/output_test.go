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

// TestErrorRemediationKeepsExplicitNewlines guards the v1.3.4 A-19
// bootstrap_failed excerpt: its "Remote stderr (<step>)" remediation is
// multi-line (stderr tail, "Failed command (exit N): ...", "Last stdout
// lines:"). The renderer used to reflow every newline into one paragraph,
// burying the failing command mid-line (seen in the local BYO e2e run).
func TestErrorRemediationKeepsExplicitNewlines(t *testing.T) {
	var errOut bytes.Buffer
	r := NewRenderer(&bytes.Buffer{}, &errOut, FormatHuman, TerminalCapabilities{StdoutTTY: false, ASCII: true, Width: 80}, redact.New())
	excerpt := "Remote stderr (setup_runner_image): curl: (22) The requested URL returned error: 403\n\n" +
		"Failed command (exit 22): GO_VERSION=$(curl -fsSL 'https://go.dev/VERSION?m=text' | head -1)\n" +
		"Last stdout lines:\nInstalling Go..."
	if err := r.Error("bootstrap_failed", "RunnerKit could not apply the BYO runner install plan.", []string{excerpt}); err != nil {
		t.Fatalf("Error() error = %v", err)
	}
	got := errOut.String()
	for _, want := range []string{
		"\nNEXT Remote stderr (setup_runner_image): curl: (22) The requested URL returned\n",
		"\n     Failed command (exit 22): GO_VERSION=$(curl -fsSL\n",
		"\n     Last stdout lines:\n     Installing Go...\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\n     \n") {
		t.Fatalf("blank excerpt line should be dropped:\n%s", got)
	}
}
