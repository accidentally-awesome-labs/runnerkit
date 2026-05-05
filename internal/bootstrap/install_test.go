package bootstrap

import (
	"context"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

type recordingExecutor struct{ commands []remote.Command }

func (r *recordingExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	return remote.ProbeResult{}, nil
}
func (r *recordingExecutor) Run(_ context.Context, _ remote.Target, command remote.Command) (remote.Result, error) {
	r.commands = append(r.commands, command)
	return remote.Result{ExitCode: 0}, nil
}

func TestApplyEphemeralRunsCommandsInOrderRedactsTokenAndAvoidsSvcSh(t *testing.T) {
	token := strings.Join([]string{"registration", "token", "ephemeral", "secret", "12345"}, "-")
	exec := &recordingExecutor{}
	opts := Options{
		RunnerName:  "runnerkit-owner-repo-ephemeral-abc123",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "ephemeral"},
		ServiceUser: "runnerkit-runner",
		RunnerToken: token,
		Mode:        "ephemeral",
		Package:     RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
	}
	if _, err := ApplyEphemeral(context.Background(), exec, remote.Target{User: "alice", Host: "example.com", Port: 22}, opts); err != nil {
		t.Fatalf("ApplyEphemeral returned error: %v", err)
	}
	var ids []string
	for _, command := range exec.commands {
		ids = append(ids, command.ID)
	}
	want := []string{
		"fix_dependencies",
		"create_runner_user",
		"download_runner",
		"configure_ephemeral_runner",
		"install_ephemeral_finalizer",
		"install_ephemeral_service",
		"install_ephemeral_ttl_timer",
		"verify_ephemeral_service",
	}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("ephemeral command IDs = %#v, want %#v", ids, want)
	}
	configure := exec.commands[3]
	if configure.ID != "configure_ephemeral_runner" {
		t.Fatalf("configure step at index 3 = %s", configure.ID)
	}
	if configure.Env["RUNNERKIT_REGISTRATION_TOKEN"] != token {
		t.Fatalf("configure_ephemeral_runner env missing token: %#v", configure.Env)
	}
	if len(configure.RedactArgs) == 0 || configure.RedactArgs[0] != token {
		t.Fatalf("configure_ephemeral_runner redaction args missing token: %#v", configure.RedactArgs)
	}
	if strings.Contains(configure.Script, token) {
		t.Fatalf("configure_ephemeral_runner script leaked token:\n%s", configure.Script)
	}
	for _, command := range exec.commands {
		if strings.Contains(command.Script, "svc.sh install") || strings.Contains(command.Script, "svc.sh start") {
			t.Fatalf("ephemeral apply leaked svc.sh install/start loop in command %s:\n%s", command.ID, command.Script)
		}
	}
}

// TestApplyDownloadRunnerCommandUsesSudoForCurlSha256SumTar asserts the
// fix for Bug 2 of gap doc 06-GAP-byo-sudo-handling.md: the
// download_runner step in Apply must prefix curl, sha256sum -c -, and
// tar xzf with sudo so the install dir owned by serviceUser receives
// the tarball without `Permission denied` failures when the SSH user
// differs from the service user.
func TestApplyDownloadRunnerCommandUsesSudoForCurlSha256SumTar(t *testing.T) {
	exec := &recordingExecutor{}
	opts := Options{RunnerName: "runnerkit-owner-repo", RepoURL: "https://github.com/owner/repo", Labels: []string{"x"}, ServiceUser: "runnerkit-runner", RunnerToken: "registration-token-x", Package: RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc"}}
	if _, err := Apply(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	var dl remote.Command
	for _, c := range exec.commands {
		if c.ID == "download_runner" {
			dl = c
			break
		}
	}
	if dl.ID == "" {
		t.Fatalf("download_runner command not found in recorded commands")
	}
	for _, want := range []string{"sudo curl", "sudo sha256sum -c -", "sudo tar xzf"} {
		if !strings.Contains(dl.Script, want) {
			t.Fatalf("download_runner script missing %q:\n%s", want, dl.Script)
		}
	}
}

// TestApplyEphemeralDownloadRunnerCommandUsesSudoForCurlSha256SumTar
// is the parallel assertion for ApplyEphemeral.
func TestApplyEphemeralDownloadRunnerCommandUsesSudoForCurlSha256SumTar(t *testing.T) {
	exec := &recordingExecutor{}
	opts := Options{RunnerName: "runnerkit-owner-repo-ephemeral", RepoURL: "https://github.com/owner/repo", Labels: []string{"x"}, ServiceUser: "runnerkit-runner", RunnerToken: "registration-token-x", Mode: "ephemeral", Package: RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc"}}
	if _, err := ApplyEphemeral(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, opts); err != nil {
		t.Fatalf("ApplyEphemeral returned error: %v", err)
	}
	var dl remote.Command
	for _, c := range exec.commands {
		if c.ID == "download_runner" {
			dl = c
			break
		}
	}
	if dl.ID == "" {
		t.Fatalf("download_runner command not found in recorded ephemeral commands")
	}
	for _, want := range []string{"sudo curl", "sudo sha256sum -c -", "sudo tar xzf"} {
		if !strings.Contains(dl.Script, want) {
			t.Fatalf("ephemeral download_runner script missing %q:\n%s", want, dl.Script)
		}
	}
}

// TestApply_WithSudoPassword_UsesSudoMinusSPipedFromHeredoc asserts the
// Plan 06-06 Path B contract: when Options.SudoPassword is non-empty,
// each sudo-prefixed command's Script is wrapped so it pipes the
// password from the env-var $RUNNERKIT_SUDO_PASSWORD into `sudo -S`.
// The literal password value MUST flow through Env (not be embedded
// in Script) and MUST be appended to RedactArgs so the executor
// scrubs it from any captured stderr.
func TestApply_WithSudoPassword_UsesSudoMinusSPipedFromHeredoc(t *testing.T) {
	password := "correct horse battery staple"
	exec := &recordingExecutor{}
	opts := Options{
		RunnerName:   "runnerkit-owner-repo",
		RepoURL:      "https://github.com/owner/repo",
		Labels:       []string{"x"},
		ServiceUser:  "runnerkit-runner",
		RunnerToken:  "registration-token-x",
		Package:      RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc"},
		SudoPassword: password,
	}
	if _, err := Apply(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(exec.commands) == 0 {
		t.Fatal("no commands recorded")
	}
	sawSudoMinusS := false
	for _, c := range exec.commands {
		if !c.Sudo {
			continue
		}
		// The wrapped script must reference sudo -S and the env var,
		// but NOT contain the literal password value.
		if strings.Contains(c.Script, password) {
			t.Fatalf("command %q leaked literal password into Script:\n%s", c.ID, c.Script)
		}
		if strings.Contains(c.Script, "sudo -S") && strings.Contains(c.Script, "RUNNERKIT_SUDO_PASSWORD") {
			sawSudoMinusS = true
			if c.Env["RUNNERKIT_SUDO_PASSWORD"] != password {
				t.Fatalf("command %q missing password Env: %#v", c.ID, c.Env)
			}
			redacted := false
			for _, ra := range c.RedactArgs {
				if ra == password {
					redacted = true
					break
				}
			}
			if !redacted {
				t.Fatalf("command %q missing password in RedactArgs: %#v", c.ID, c.RedactArgs)
			}
		}
	}
	if !sawSudoMinusS {
		t.Fatalf("no sudo command was wrapped with sudo -S + RUNNERKIT_SUDO_PASSWORD env in any of %d commands", len(exec.commands))
	}
}

// TestApply_WithoutSudoPassword_BehaviorUnchangedFromPlan0605 asserts
// that when Options.SudoPassword is empty, none of the rendered
// commands are wrapped with `sudo -S` or carry the
// RUNNERKIT_SUDO_PASSWORD env — preserving Plan 06-05's exact behavior
// for the NOPASSWD-sudo / byo-prepared happy path.
func TestApply_WithoutSudoPassword_BehaviorUnchangedFromPlan0605(t *testing.T) {
	exec := &recordingExecutor{}
	opts := Options{
		RunnerName:  "runnerkit-owner-repo",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"x"},
		ServiceUser: "runnerkit-runner",
		RunnerToken: "registration-token-x",
		Package:     RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc"},
	}
	if _, err := Apply(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	for _, c := range exec.commands {
		if strings.Contains(c.Script, "sudo -S") {
			t.Fatalf("command %q unexpectedly wrapped with sudo -S without SudoPassword opt:\n%s", c.ID, c.Script)
		}
		if _, has := c.Env["RUNNERKIT_SUDO_PASSWORD"]; has {
			t.Fatalf("command %q unexpectedly carried RUNNERKIT_SUDO_PASSWORD without SudoPassword opt", c.ID)
		}
	}
}

// Bug 9 / Plan 06-09 — gap doc 06-GAP-byo-sudo-handling.md.
//
// Plan 06-07 attempt-6 against salar@mckee-small-desktop got past
// preflight (Bugs 7+8 closed) and aborted at the configure_runner
// step:
//
//   sudo: a terminal is required to read the password; either use
//   the -S option to read from standard input or configure an
//   askpass helper
//   sudo: a password is required
//
// Root cause: the configure_runner and configure_ephemeral_runner
// remote.Command structs in Apply / ApplyEphemeral omit the
// `Sudo: true` field. wrapSudoCommand bails out early when c.Sudo
// is false (the gating "Path B should never wrap non-sudo commands"
// safety), so the rendered script — which DOES contain `sudo curl`,
// `sudo sha256sum`, `sudo tar`, `sudo chown`, and `sudo su -s` — is
// dispatched verbatim. Each of those raw sudo invocations then tries
// to read the password from /dev/tty over a non-tty SSH session and
// fails.

func TestApplyConfigureRunnerCommand_HasSudoTrue(t *testing.T) {
	t.Parallel()
	exec := &recordingExecutor{}
	opts := Options{
		RunnerName: "runnerkit-owner-repo", RepoURL: "https://github.com/owner/repo",
		Labels: []string{"x"}, ServiceUser: "runnerkit-runner", RunnerToken: "tk",
		Package: RunnerPackage{Filename: "r.tgz", URL: "https://x.invalid/r.tgz", SHA256: "abc"},
	}
	if _, err := Apply(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	for _, c := range exec.commands {
		if c.ID == "configure_runner" {
			if !c.Sudo {
				t.Fatalf("configure_runner must have Sudo: true so wrapSudoCommand wraps `sudo curl/sha256sum/chown/su` for Path B password threading. Got Sudo=%v", c.Sudo)
			}
			return
		}
	}
	t.Fatal("configure_runner command not recorded")
}

func TestApplyEphemeralConfigureCommand_HasSudoTrue(t *testing.T) {
	t.Parallel()
	exec := &recordingExecutor{}
	opts := Options{
		RunnerName: "runnerkit-owner-repo", RepoURL: "https://github.com/owner/repo",
		Labels: []string{"x"}, ServiceUser: "runnerkit-runner", RunnerToken: "tk",
		Package: RunnerPackage{Filename: "r.tgz", URL: "https://x.invalid/r.tgz", SHA256: "abc"},
		Mode:    "ephemeral",
	}
	if _, err := ApplyEphemeral(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, opts); err != nil {
		t.Fatalf("ApplyEphemeral returned error: %v", err)
	}
	for _, c := range exec.commands {
		if c.ID == "configure_ephemeral_runner" {
			if !c.Sudo {
				t.Fatalf("configure_ephemeral_runner must have Sudo: true (same reason as Bug 9 for Apply). Got Sudo=%v", c.Sudo)
			}
			return
		}
	}
	t.Fatal("configure_ephemeral_runner command not recorded")
}

func TestApplyRunsBootstrapCommandsInOrderAndRedactsToken(t *testing.T) {
	token := "registration-token-secret-12345"
	exec := &recordingExecutor{}
	opts := Options{RunnerName: "runnerkit-owner-repo-local", RepoURL: "https://github.com/owner/repo", Labels: []string{"self-hosted", "runnerkit"}, ServiceUser: "runnerkit-runner", RunnerToken: token, Package: RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"}}
	if _, err := Apply(context.Background(), exec, remote.Target{User: "alice", Host: "example.com", Port: 22}, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	var ids []string
	for _, command := range exec.commands {
		ids = append(ids, command.ID)
	}
	want := []string{"fix_dependencies", "create_runner_user", "download_runner", "configure_runner", "install_service", "verify_service"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("command IDs = %#v, want %#v", ids, want)
	}
	configure := exec.commands[3]
	if configure.ID != "configure_runner" {
		t.Fatalf("command before install_service = %s, want configure_runner", configure.ID)
	}
	if configure.Env["RUNNERKIT_REGISTRATION_TOKEN"] != token {
		t.Fatalf("configure token env missing: %#v", configure.Env)
	}
	if len(configure.RedactArgs) == 0 || configure.RedactArgs[0] != token {
		t.Fatalf("configure redaction args missing token: %#v", configure.RedactArgs)
	}
	if strings.Contains(configure.Script, token) {
		t.Fatalf("configure script leaked token:\n%s", configure.Script)
	}
}
