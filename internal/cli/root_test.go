package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func executeForTest(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var out, err bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version: "test-version",
		Out:     &out,
		Err:     &err,
	})
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	return out.String(), err.String(), runErr
}

func TestRootHelpListsRunnerKitAndUp(t *testing.T) {
	out, _, err := executeForTest(t, "--help")
	if err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	if !strings.Contains(out, "RunnerKit") {
		t.Fatalf("help missing RunnerKit: %q", out)
	}
	if !strings.Contains(out, "up") {
		t.Fatalf("help missing up command: %q", out)
	}
}

func TestVersionJSONContract(t *testing.T) {
	out, _, err := executeForTest(t, "--json", "version")
	if err != nil {
		t.Fatalf("version returned error: %v", err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("json output contains ansi: %q", out)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("version output is not json: %v\n%s", err, out)
	}
	if payload["ok"] != true || payload["command"] != "version" || payload["version"] != "test-version" || payload["redactions_applied"] != true {
		t.Fatalf("unexpected version payload: %#v", payload)
	}
}

func TestInvalidFlagMapsToExitCodeTwo(t *testing.T) {
	_, _, err := executeForTest(t, "--definitely-not-a-flag")
	if err == nil {
		t.Fatal("expected invalid flag error")
	}
	if got := ExitCode(err); got != ExitInvalidInput {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInvalidInput)
	}
}
