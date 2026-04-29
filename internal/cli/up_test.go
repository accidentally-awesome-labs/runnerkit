package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUpDryRunDisplaysPhaseOneWizard(t *testing.T) {
	out, errOut, err := executeForTest(t, "up", "--dry-run", "--repo", "owner/name", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("up dry-run returned error: %v\nstderr: %s", err, errOut)
	}
	for _, step := range []string{"Welcome", "Prerequisites", "Repo/auth", "Safety checks", "State preview", "Next steps"} {
		if !strings.Contains(out, step) {
			t.Fatalf("dry-run output missing step %q:\n%s", step, out)
		}
	}
	for _, copy := range []string{"Phase 1 does not install a runner yet", "Will not install a runner in Phase 1"} {
		if !strings.Contains(out, copy) {
			t.Fatalf("dry-run output missing copy %q:\n%s", copy, out)
		}
	}
}

func TestUpNonInteractiveRequiresRepo(t *testing.T) {
	_, errOut, err := executeForTest(t, "up", "--non-interactive", "--no-color")
	if err == nil {
		t.Fatal("expected missing repo error")
	}
	if got := ExitCode(err); got != ExitInputRequired {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInputRequired)
	}
	if !strings.Contains(errOut, "--repo owner/name") {
		t.Fatalf("missing remediation in stderr: %q", errOut)
	}
}

func TestUpJSONDryRunContract(t *testing.T) {
	out, errOut, err := executeForTest(t, "--json", "up", "--dry-run", "--repo", "owner/name", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("up json dry-run returned error: %v\nstderr: %s", err, errOut)
	}
	if strings.Contains(out, "\x1b[") || !strings.HasPrefix(out, "{") {
		t.Fatalf("json output is not machine-only: %q", out)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json output invalid: %v\n%s", err, out)
	}
	if payload["runner_installed"] != false || payload["redactions_applied"] != true {
		t.Fatalf("unexpected up payload: %#v", payload)
	}
}
