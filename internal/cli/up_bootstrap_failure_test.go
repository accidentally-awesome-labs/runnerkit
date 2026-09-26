package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

// realExitError returns a genuine *exec.ExitError, the error type the
// system ssh executor returns when the remote script exits non-zero.
func realExitError(t *testing.T, code string) error {
	t.Helper()
	err := exec.Command("sh", "-c", "exit "+code).Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T %v", err, err)
	}
	return err
}

// TestUp_BootstrapFailed_NamesFailingStepAndCommand is the P1-15
// regression: the system executor returns (*exec.ExitError, result)
// on a non-zero exit, Apply used to return that raw error, and `up`
// rendered the failing step as "(unknown)".
func TestUp_BootstrapFailed_NamesFailingStepAndCommand(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor() // probe reports ID=ubuntu, so setup_runner_image runs
	remoteExec.runResults["setup_runner_image"] = remote.Result{
		ExitCode: 4,
		Stdout:   "Installing Node.js 20.x...\nE: Unable to locate package nodejs",
		Stderr: "Warning: Permanently added '203.0.113.7' (ED25519) to the list of known hosts.\n" +
			"sudo: a terminal is required to read the password; either use the -S option to read from standard input or configure an askpass helper\n" +
			"RKFAIL:sudo mkdir -p /etc/apt/keyrings\n",
	}
	remoteExec.runErrs["setup_runner_image"] = realExitError(t, "4")

	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"--json", "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected bootstrap_failed exit")
	}
	if got := ExitCode(err); got != ExitSafetyGate {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitSafetyGate)
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json output: %v\n%s", err, out.String())
	}
	errorObject, _ := payload["error"].(map[string]any)
	if errorObject["code"] != "bootstrap_failed" {
		t.Fatalf("error code = %v, want bootstrap_failed: %#v", errorObject["code"], payload)
	}
	var remediation []string
	for _, r := range errorObject["remediation"].([]any) {
		remediation = append(remediation, r.(string))
	}
	combined := strings.Join(remediation, "\n")
	for _, want := range []string{
		"(setup_runner_image)",
		"Failed command (exit 4): sudo mkdir -p /etc/apt/keyrings",
		"a terminal is required",
		"E: Unable to locate package nodejs",
	} {
		if !strings.Contains(combined, want) {
			t.Errorf("remediation missing %q:\n%s", want, combined)
		}
	}
	for _, unwanted := range []string{"(unknown)", "Permanently added", "RKFAIL:"} {
		if strings.Contains(combined, unwanted) {
			t.Errorf("remediation contains %q:\n%s", unwanted, combined)
		}
	}
	// A-18: setup_runner_image must run after the runner user exists.
	var ids []string
	for _, run := range remoteExec.runs {
		ids = append(ids, run.ID)
	}
	joined := "," + strings.Join(ids, ",") + ","
	if !strings.Contains(joined, ",create_runner_user,setup_runner_image,") {
		t.Fatalf("bootstrap order = %v, want create_runner_user immediately before setup_runner_image", ids)
	}
}

func TestLastCommandFailureContext(t *testing.T) {
	t.Run("remote error with trap, stdout tail and ssh noise", func(t *testing.T) {
		stdout := make([]string, 0, 40)
		for i := 0; i < 40; i++ {
			stdout = append(stdout, "filler")
		}
		stdout = append(stdout, "E: Package 'foo' has no installation candidate")
		result := bootstrap.Result{Commands: []remote.Result{
			{ExitCode: 0},
			{ExitCode: 100, Stdout: strings.Join(stdout, "\n"), Stderr: "Warning: Permanently added 'h' (ED25519) to the list of known hosts.\nRKFAIL:sudo apt-get install -y foo\n"},
		}}
		id, detail := lastCommandFailureContext(result, remote.RemoteError{CommandID: "fix_dependencies", ExitCode: 100})
		if id != "fix_dependencies" {
			t.Fatalf("id = %q", id)
		}
		if !strings.HasPrefix(detail, "(empty)\n") {
			t.Fatalf("detail should flag empty stderr first:\n%s", detail)
		}
		if !strings.Contains(detail, "Failed command (exit 100): sudo apt-get install -y foo") {
			t.Fatalf("detail missing failed command:\n%s", detail)
		}
		if !strings.Contains(detail, "has no installation candidate") {
			t.Fatalf("detail missing stdout tail:\n%s", detail)
		}
		if got := strings.Count(detail, "filler"); got != bootstrapStdoutTailLines-1 {
			t.Fatalf("stdout tail not bounded: %d filler lines\n%s", got, detail)
		}
		if strings.Contains(detail, "Permanently added") {
			t.Fatalf("ssh known-hosts noise not stripped:\n%s", detail)
		}
	})
	t.Run("plain stderr unchanged", func(t *testing.T) {
		result := bootstrap.Result{Commands: []remote.Result{{ExitCode: 1, Stderr: "  boom  "}}}
		id, detail := lastCommandFailureContext(result, remote.RemoteError{CommandID: "configure_runner", ExitCode: 1})
		if id != "configure_runner" || detail != "boom" {
			t.Fatalf("got (%q, %q)", id, detail)
		}
	})
	t.Run("silent step failure still names the step", func(t *testing.T) {
		result := bootstrap.Result{Commands: []remote.Result{{}, {ExitCode: 137}}}
		id, detail := lastCommandFailureContext(result, remote.RemoteError{CommandID: "setup_runner_image", ExitCode: 137})
		if id != "setup_runner_image" || detail != "(empty)\nExit code: 137" {
			t.Fatalf("got (%q, %q)", id, detail)
		}
		transport := errors.New("exec: \"ssh\": executable file not found in $PATH")
		_, detail = lastCommandFailureContext(bootstrap.Result{Commands: []remote.Result{{ExitCode: -1}}}, remote.RemoteError{CommandID: "fix_dependencies", ExitCode: -1, Err: transport})
		if detail != "(empty)\nExecutor error: "+transport.Error() {
			t.Fatalf("transport detail = %q", detail)
		}
	})
	t.Run("no remote error never renders unknown", func(t *testing.T) {
		result := bootstrap.Result{Commands: []remote.Result{{}, {ExitCode: -1, Stderr: "ssh: connect to host h port 22: Connection refused"}}}
		id, _ := lastCommandFailureContext(result, errors.New("exit status 255"))
		if id == "unknown" || id == "" {
			t.Fatalf("id = %q", id)
		}
	})
}
