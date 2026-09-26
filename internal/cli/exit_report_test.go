package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// A-01 (P1-1): errors nobody rendered are printed by main exactly once;
// errors a command already rendered are not repeated.

func TestReportUnrenderedError_PrintsCauseToStderr(t *testing.T) {
	var out, errOut bytes.Buffer
	err := errors.New("unknown command \"bogus\" for \"runnerkit\"\n\nDid you mean this?\n\tlogs\n")
	ReportUnrenderedError(&out, &errOut, []string{"bogus"}, err, false)
	if !strings.HasPrefix(errOut.String(), "runnerkit: unknown command \"bogus\"") || !strings.Contains(errOut.String(), "Did you mean this?") {
		t.Fatalf("stderr = %q", errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("human mode must not write to stdout: %q", out.String())
	}
}

func TestReportUnrenderedError_SilentWhenAlreadyRendered(t *testing.T) {
	var out, errOut bytes.Buffer
	ReportUnrenderedError(&out, &errOut, []string{"--json", "status"}, NewExitError(ExitInvalidInput, errors.New("x")), true)
	if out.Len() != 0 || errOut.Len() != 0 {
		t.Fatalf("already-rendered error printed again: stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

func TestReportUnrenderedError_JSONEnvelope(t *testing.T) {
	for _, args := range [][]string{{"--json", "bogus"}, {"bogus", "--json=true"}} {
		var out, errOut bytes.Buffer
		ReportUnrenderedError(&out, &errOut, args, NewExitError(ExitInvalidInput, errors.New(`unknown command "bogus" for "runnerkit"`)), false)
		var payload map[string]any
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("%v: stdout not json: %v\n%s", args, err, out.String())
		}
		errObj, _ := payload["error"].(map[string]any)
		if payload["ok"] != false || errObj["code"] != "cli_usage" || !strings.Contains(errObj["message"].(string), "bogus") {
			t.Fatalf("%v: unexpected envelope %#v", args, payload)
		}
		if errOut.Len() == 0 {
			t.Fatalf("%v: stderr must still name the cause", args)
		}
	}
	var out bytes.Buffer
	ReportUnrenderedError(&out, &bytes.Buffer{}, []string{"--json=false", "bogus"}, errors.New("boom"), false)
	if out.Len() != 0 {
		t.Fatalf("--json=false must not emit an envelope: %q", out.String())
	}
}

func TestArgsRequestJSON(t *testing.T) {
	cases := map[string]bool{
		"--json":         true,
		"--json=1":       true,
		"--json=maybe":   false,
		"--json=false":   false,
		"-- --json":      false,
		"status --bogus": false,
	}
	for in, want := range cases {
		if got := argsRequestJSON(strings.Fields(in)); got != want {
			t.Fatalf("argsRequestJSON(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestInvalidFlagIsRendered(t *testing.T) {
	_, errOut, err := executeForTest(t, "status", "--bogus")
	if ExitCode(err) != ExitInvalidInput {
		t.Fatalf("ExitCode = %d, want %d", ExitCode(err), ExitInvalidInput)
	}
	if !strings.Contains(errOut, "invalid_flag: unknown flag: --bogus") || !strings.Contains(errOut, "runnerkit status --help") {
		t.Fatalf("flag error not rendered: %q", errOut)
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	_, _, err := executeForTest(t, "stauts")
	if ExitCode(err) != ExitInvalidInput {
		t.Fatalf("ExitCode = %d, want %d (err=%v)", ExitCode(err), ExitInvalidInput, err)
	}
	if !strings.Contains(err.Error(), "Did you mean this?") || !strings.Contains(err.Error(), "status") {
		t.Fatalf("suggestions lost: %q", err.Error())
	}
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	out, _, err := executeForTest(t, "--version")
	if err != nil {
		t.Fatalf("--version returned error: %v", err)
	}
	if out != "runnerkit test-version\n" {
		t.Fatalf("--version stdout = %q", out)
	}
}

func TestByoPrepareTombstone(t *testing.T) {
	_, errOut, err := executeForTest(t, "byo-prepare", "--host", "alice@example.com", "--anything")
	if ExitCode(err) != ExitInvalidInput {
		t.Fatalf("ExitCode = %d, want %d", ExitCode(err), ExitInvalidInput)
	}
	if !strings.Contains(errOut, "removed in v1.0.8") || !strings.Contains(errOut, "Known issues") {
		t.Fatalf("tombstone message missing: %q", errOut)
	}
	help, _, _ := executeForTest(t, "--help")
	if strings.Contains(help, "byo-prepare") {
		t.Fatalf("byo-prepare must stay hidden from help:\n%s", help)
	}
}
