package bootstrap

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

func ubuntuTestOptions(mode string) Options {
	return Options{
		RunnerName:  "runnerkit-owner-repo-local",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"self-hosted", "runnerkit"},
		ServiceUser: "runnerkit-runner",
		RunnerToken: "registration-token-secret",
		OSReleaseID: "ubuntu",
		Mode:        mode,
		Package:     RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
	}
}

type applyFunc func(context.Context, remote.Executor, remote.Target, Options) (Result, error)

var applyVariants = []struct {
	name string
	mode string
	fn   applyFunc
}{
	{name: "Apply", mode: "", fn: Apply},
	{name: "ApplyEphemeral", mode: "ephemeral", fn: ApplyEphemeral},
}

// TestApplyCreatesRunnerUserBeforeImageSetupOnUbuntu is the P0-2
// regression. setup_runner_image adds the service user to the docker
// group and installs rustup as that user; before v1.3.4 it ran before
// create_runner_user, so on every fresh host both silently did nothing.
// The older order tests never set OSReleaseID, so setup_runner_image
// was not even in the plan they checked.
func TestApplyCreatesRunnerUserBeforeImageSetupOnUbuntu(t *testing.T) {
	for _, variant := range applyVariants {
		t.Run(variant.name, func(t *testing.T) {
			rec := &recordingExecutor{}
			if _, err := variant.fn(context.Background(), rec, remote.Target{User: "alice", Host: "example.com", Port: 22}, ubuntuTestOptions(variant.mode)); err != nil {
				t.Fatalf("%s returned error: %v", variant.name, err)
			}
			index := map[string]int{}
			var ids []string
			for i, command := range rec.commands {
				index[command.ID] = i
				ids = append(ids, command.ID)
			}
			userIdx, okUser := index["create_runner_user"]
			imageIdx, okImage := index["setup_runner_image"]
			if !okUser || !okImage {
				t.Fatalf("plan missing create_runner_user or setup_runner_image: %v", ids)
			}
			if userIdx > imageIdx {
				t.Fatalf("create_runner_user (%d) must precede setup_runner_image (%d): %v", userIdx, imageIdx, ids)
			}
			if index["fix_dependencies"] != 0 {
				t.Fatalf("fix_dependencies must stay first: %v", ids)
			}
		})
	}
}

func TestImageSetupVersionBumpedForRunnerUserReorder(t *testing.T) {
	// "1" markers were written by hosts where the docker group and
	// rustup steps ran before the runner user existed; they must re-run.
	if ImageSetupVersion == "1" {
		t.Fatal("ImageSetupVersion must be bumped past \"1\" so hosts set up before the P0-2 reorder re-run setup_runner_image")
	}
}

// TestApplyDispatchesEveryScriptWithFailTrap asserts every bootstrap
// script starts with the RKFAIL ERR trap (P1-15).
func TestApplyDispatchesEveryScriptWithFailTrap(t *testing.T) {
	for _, variant := range applyVariants {
		t.Run(variant.name, func(t *testing.T) {
			rec := &recordingExecutor{}
			if _, err := variant.fn(context.Background(), rec, remote.Target{User: "alice", Host: "example.com", Port: 22}, ubuntuTestOptions(variant.mode)); err != nil {
				t.Fatalf("%s returned error: %v", variant.name, err)
			}
			if len(rec.commands) == 0 {
				t.Fatal("no commands dispatched")
			}
			for _, command := range rec.commands {
				if !strings.HasPrefix(command.Script, FailTrapLine+"\n") {
					t.Errorf("%s script does not start with the fail trap:\n%s", command.ID, firstLines(command.Script, 3))
				}
				if strings.Count(command.Script, FailTrapLine) != 1 {
					t.Errorf("%s script has the fail trap %d times", command.ID, strings.Count(command.Script, FailTrapLine))
				}
			}
		})
	}
}

// TestFailTrapReportsFailingCommandInBash runs a trapped script under a
// real bash with set -euo pipefail: tolerated failures stay silent, the
// first unguarded failure prints RKFAIL:<unexpanded command>, and the
// exit status is preserved.
func TestFailTrapReportsFailingCommandInBash(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	script := withFailTrap(`set -euo pipefail
SECRET=hunter2
if ! false; then :; fi
false || true
X=$(false | cat; echo ok)
for p in a b; do
  if ! command -v rk-missing-$p >/dev/null 2>&1; then :; fi
done
sh -c 'exit 7' "$SECRET"
echo unreachable
`)
	cmd := exec.Command("bash", "-s")
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
		t.Fatalf("exit = %v, want exit status 7 (stderr=%q)", err, stderr.String())
	}
	if strings.Count(stderr.String(), FailTrapMarker) != 1 {
		t.Fatalf("want exactly one %s line, stderr=%q", FailTrapMarker, stderr.String())
	}
	if got := FailedCommand(stderr.String()); got != `sh -c 'exit 7' "$SECRET"` {
		t.Fatalf("FailedCommand = %q (stderr=%q)", got, stderr.String())
	}
	if strings.Contains(stderr.String(), "hunter2") {
		t.Fatalf("trap expanded a variable into stderr: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "unreachable") {
		t.Fatal("set -e no longer stops the script after the trap fires")
	}
}

// failingStepExecutor fails one command the way SystemExecutor does:
// a non-zero ExitCode together with the raw *exec.ExitError.
type failingStepExecutor struct {
	failID string
	result remote.Result
	err    error
}

func (f *failingStepExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	return remote.ProbeResult{}, nil
}
func (f *failingStepExecutor) Run(_ context.Context, _ remote.Target, command remote.Command) (remote.Result, error) {
	if command.ID == f.failID {
		return f.result, f.err
	}
	return remote.Result{ExitCode: 0}, nil
}

// TestApplyWrapsExecutorErrorsAsRemoteError is the P1-15 regression:
// the raw *exec.ExitError from the system executor used to escape, so
// `up` could not name the failing step.
func TestApplyWrapsExecutorErrorsAsRemoteError(t *testing.T) {
	exitErr := exec.Command("sh", "-c", "exit 4").Run()
	var rawExit *exec.ExitError
	if !errors.As(exitErr, &rawExit) {
		t.Fatalf("setup: want *exec.ExitError, got %T", exitErr)
	}
	cases := []struct {
		name   string
		result remote.Result
		err    error
		want   int
	}{
		{name: "exit error", result: remote.Result{ExitCode: 4, Stderr: "RKFAIL:sudo mkdir -p /etc/apt/keyrings\n"}, err: exitErr, want: 4},
		{name: "transport error without exit code", result: remote.Result{}, err: errors.New("ssh: not found"), want: -1},
		{name: "exit code without error", result: remote.Result{ExitCode: 2}, want: 2},
	}
	for _, variant := range applyVariants {
		for _, tc := range cases {
			t.Run(variant.name+"/"+tc.name, func(t *testing.T) {
				ex := &failingStepExecutor{failID: "setup_runner_image", result: tc.result, err: tc.err}
				_, err := variant.fn(context.Background(), ex, remote.Target{User: "alice", Host: "h", Port: 22}, ubuntuTestOptions(variant.mode))
				var remoteErr remote.RemoteError
				if !errors.As(err, &remoteErr) {
					t.Fatalf("err = %T %v, want remote.RemoteError", err, err)
				}
				if remoteErr.CommandID != "setup_runner_image" || remoteErr.ExitCode != tc.want {
					t.Fatalf("RemoteError = %+v, want setup_runner_image exit %d", remoteErr, tc.want)
				}
				if tc.err != nil && !errors.Is(err, tc.err) {
					t.Fatalf("underlying error not preserved: %v", err)
				}
			})
		}
	}
}

func TestServiceStepFailureKeepsServiceNotActiveErrorAndNamesStep(t *testing.T) {
	ex := &failingStepExecutor{failID: "verify_service", result: remote.Result{ExitCode: 3, Stderr: "inactive"}}
	_, err := Apply(context.Background(), ex, remote.Target{User: "alice", Host: "h", Port: 22}, ubuntuTestOptions(""))
	var serviceErr ServiceNotActiveError
	if !errors.As(err, &serviceErr) || serviceErr.CommandID != "verify_service" || serviceErr.Stderr != "inactive" {
		t.Fatalf("err = %#v, want ServiceNotActiveError for verify_service", err)
	}
	var remoteErr remote.RemoteError
	if !errors.As(serviceErr.Err, &remoteErr) || remoteErr.CommandID != "verify_service" || remoteErr.ExitCode != 3 {
		t.Fatalf("ServiceNotActiveError.Err = %#v, want RemoteError{verify_service, 3}", serviceErr.Err)
	}
	// Every step failure, service steps included, carries a RemoteError.
	remoteErr = remote.RemoteError{}
	if !errors.As(err, &remoteErr) || remoteErr.CommandID != "verify_service" {
		t.Fatalf("errors.As(err, *RemoteError) through ServiceNotActiveError failed: %#v", err)
	}
}

func TestFailedCommandPicksLastMarker(t *testing.T) {
	stderr := "noise\nRKFAIL:first\nmore\n  RKFAIL:sudo apt-get install -y foo  \n"
	if got := FailedCommand(stderr); got != "sudo apt-get install -y foo" {
		t.Fatalf("FailedCommand = %q", got)
	}
	if got := FailedCommand("no marker here"); got != "" {
		t.Fatalf("FailedCommand = %q, want empty", got)
	}
}

// TestFailTrapFoldsMultiLineCommand: for a failing heredoc command
// bash prints the whole heredoc after RKFAIL:. The body must not be
// mistaken for remote stderr, and only the first line names the
// command.
func TestFailTrapFoldsMultiLineCommand(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	target := t.TempDir() + "/missing-dir/finalize.sh"
	script := withFailTrap("set -euo pipefail\necho before-failure >&2\ncat >" + target + " <<'EOSCRIPT'\nheredoc-body-line\nEOSCRIPT\necho unreachable\n")
	cmd := exec.Command("bash", "-s")
	cmd.Stdin = strings.NewReader(script)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("script should fail")
	}
	if !strings.Contains(stderr.String(), "heredoc-body-line") {
		t.Fatalf("setup: expected bash to print the heredoc body after RKFAIL, stderr=%q", stderr.String())
	}
	before, command := SplitFailTrap(stderr.String())
	if want := "cat > " + target + " <<'EOSCRIPT' ..."; command != want {
		t.Fatalf("command = %q, want %q (stderr=%q)", command, want, stderr.String())
	}
	if strings.Contains(before, "heredoc-body-line") || strings.Contains(before, FailTrapMarker) {
		t.Fatalf("before still carries the trap output: %q", before)
	}
	if !strings.Contains(before, "before-failure") {
		t.Fatalf("before lost the script's own stderr: %q", before)
	}
	if got, _ := SplitFailTrap("plain stderr\n"); got != "plain stderr\n" {
		t.Fatalf("SplitFailTrap without marker = %q", got)
	}
}

func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
