package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestUpgrade_DetectsHomebrew(t *testing.T) {
	cases := []string{
		"/opt/homebrew/Cellar/runnerkit/1.0.0/bin/runnerkit",
		"/usr/local/Cellar/runnerkit/1.0.0/bin/runnerkit",
		"/opt/homebrew/Caskroom/runnerkit/1.0.0/runnerkit",
	}
	for _, p := range cases {
		if got := detectChannel(p); got != "homebrew" {
			t.Fatalf("detectChannel(%q) = %q, want %q", p, got, "homebrew")
		}
	}
}

func TestUpgrade_DetectsBinaryChannel(t *testing.T) {
	cases := []string{
		"/usr/local/bin/runnerkit",
		"/home/user/.local/bin/runnerkit",
	}
	for _, p := range cases {
		if got := detectChannel(p); got != "binary" {
			t.Fatalf("detectChannel(%q) = %q, want %q", p, got, "binary")
		}
	}
}

func TestUpgrade_DetectsUnknownChannel(t *testing.T) {
	if got := detectChannel("/opt/somethingelse/foo"); got != "unknown" {
		t.Fatalf("detectChannel(unknown path) = %q, want %q", got, "unknown")
	}
}

// TestUpgrade_JSONContract: invokes runnerkit upgrade --json and asserts
// the JSON shape includes ok/channel/commands/current/latest. Uses a state
// dir to avoid picking up the user's real ~/.local/state cache.
func TestUpgrade_JSONContract(t *testing.T) {
	stateDir := t.TempDir()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:      "test-version",
		Out:          &out,
		Err:          &errOut,
		StateBaseDir: stateDir,
	})
	cmd.SetArgs([]string{"--json", "upgrade", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("upgrade --json returned error: %v\nstderr=%s", err, errOut.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json output: %v\n%s", err, out.String())
	}
	if payload["ok"] != true {
		t.Fatalf("ok=%v, want true; payload=%#v", payload["ok"], payload)
	}
	if _, ok := payload["channel"].(string); !ok {
		t.Fatalf("channel missing or not string: %#v", payload)
	}
	commands, ok := payload["commands"].([]any)
	if !ok || len(commands) == 0 {
		t.Fatalf("commands missing or empty: %#v", payload)
	}
	if payload["current"] != "test-version" {
		t.Fatalf("current=%v, want test-version", payload["current"])
	}
	if _, ok := payload["latest"]; !ok {
		t.Fatalf("latest key missing: %#v", payload)
	}
}

// TestUpgrade_HumanContract: human output mentions the install channel
// label and at least one upgrade command.
func TestUpgrade_HumanContract(t *testing.T) {
	stateDir := t.TempDir()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:      "test-version",
		Out:          &out,
		Err:          &errOut,
		StateBaseDir: stateDir,
	})
	cmd.SetArgs([]string{"upgrade", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("upgrade returned error: %v\nstderr=%s", err, errOut.String())
	}
	text := out.String()
	if !strings.Contains(text, "test-version") {
		t.Fatalf("upgrade human output missing version: %q", text)
	}
	if !strings.Contains(text, "Upgrade instructions") {
		t.Fatalf("upgrade human output missing 'Upgrade instructions' header: %q", text)
	}
}
