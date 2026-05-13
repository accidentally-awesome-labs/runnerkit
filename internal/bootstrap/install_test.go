package bootstrap

import (
	"context"
	"errors"
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
	for _, want := range []string{"sudo curl", "sudo sha256sum -c -", "sudo tar xzf", "runnerkit-shared-bin"} {
		if !strings.Contains(dl.Script, want) {
			t.Fatalf("download_runner script missing %q:\n%s", want, dl.Script)
		}
	}
}

func TestDownloadRunnerUsesRunnerCacheRootWhenSet(t *testing.T) {
	exec := &recordingExecutor{}
	opts := Options{
		RunnerName:      "runnerkit-owner-repo",
		RepoURL:         "https://github.com/owner/repo",
		Labels:          []string{"x"},
		ServiceUser:     "runnerkit-runner",
		RunnerToken:     "registration-token-x",
		Package:         RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc"},
		RunnerCacheRoot: "/var/tmp/runnerkit-cache-override-test",
	}
	if _, err := Apply(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	var dl string
	for _, c := range exec.commands {
		if c.ID == "download_runner" {
			dl = c.Script
			break
		}
	}
	if !strings.Contains(dl, "/var/tmp/runnerkit-cache-override-test") {
		t.Fatalf("download_runner script missing RunnerCacheRoot override:\n%s", dl)
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
	for _, want := range []string{"sudo curl", "sudo sha256sum -c -", "sudo tar xzf", "runnerkit-shared-bin"} {
		if !strings.Contains(dl.Script, want) {
			t.Fatalf("ephemeral download_runner script missing %q:\n%s", want, dl.Script)
		}
	}
}

// TestApply_WithoutSudoPassword_BehaviorUnchangedFromPlan0605 asserts
// that when Options.SudoPassword is empty, none of the rendered
// commands are wrapped with `sudo -S` or carry the
// RUNNERKIT_SUDO_PASSWORD env — preserving Plan 06-05's exact behavior
// for the NOPASSWD-sudo / byo-prepared happy path.
// Bug 10 / Plan 06-09 — gap doc 06-GAP-byo-sudo-handling.md.
//
// Plan 06-07 attempt-7 against salar@mckee-small-desktop got past
// preflight + Path B prompt + Bug 9 fix (configure_runner Sudo: true)
// and aborted with:
//
//   Sorry, try again.
//   sudo: no password was provided
//   sudo: 1 incorrect password attempt
//
// Root cause: wrapSudoCommand wrapped the script in
//   printf '$PW' | { rewritten_script }
// so the brace group's outer stdin was the password. The inner
// pattern `printf 'sum' | sudo -S sha256sum -c -` (from
// RenderInstallScript's checksum-verify step) opens its OWN pipe to
// sudo. Ubuntu's sudo defaults (use_pty + tty-scoped timestamp cache)
// did not cache cred reliably across the SSH session, so sudo -S
// re-prompted; -S read from the inner printf and got the checksum
// string instead of the password ("Sorry, try again"), then EOF
// ("no password was provided").
//
// The byo-prepare flow (internal/cli/byo_prepare.go) does NOT have
// this bug because it uses a different structure — prime cred once
// via a dedicated `printf | sudo -S -v` invocation, then run the
// rewritten script WITHOUT an outer brace-group pipe. Each subsequent
// sudo -S hits the freshly-primed cred and does not read its stdin,
// so inner `printf X | sudo Y` patterns work because sudo lets the
// printf reach Y.
//
// Bug 15 / Plan 06-09 — gap doc 06-GAP-byo-sudo-handling.md.
//
// Plan 06-07 attempt-12 against salar@mckee-small-desktop got past
// install_service (Bug 14 fix) and aborted at verify_service:
//
//   sudo: ./svc.sh: command not found
//
// Root cause: install.go's verify_service Command literal:
//   {ID: "verify_service", Script: "set -euo pipefail\nsudo ./svc.sh status\n", Sudo: true}
// has no `cd` — each remote.Command runs in a fresh SSH session whose
// default cwd is the SSH user's HOME, not installPath. ./svc.sh is
// relative to cwd, so the lookup fails.
//
// install_service does NOT have this bug because RenderServiceScript
// emits an explicit `cd <installPath>` at the top. verify_service was
// written inline as a one-liner and that cd was never added.

func TestApply_VerifyService_CdsIntoInstallPathBeforeSvcSh(t *testing.T) {
	t.Parallel()
	exec := &recordingExecutor{}
	opts := Options{
		RunnerName: "runnerkit-x", RepoURL: "https://github.com/owner/repo",
		Labels: []string{"x"}, ServiceUser: "runnerkit-runner", RunnerToken: "tk",
		InstallPath: "/opt/actions-runner/runnerkit-x",
		Package:     RunnerPackage{Filename: "r.tgz", URL: "https://x.invalid/r.tgz", SHA256: "abc"},
	}
	if _, err := Apply(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	for _, c := range exec.commands {
		if c.ID == "verify_service" {
			if !strings.Contains(c.Script, "cd /opt/actions-runner/runnerkit-x") {
				t.Fatalf("verify_service script must cd into installPath before invoking ./svc.sh (Bug 15):\n%s", c.Script)
			}
			cdIdx := strings.Index(c.Script, "cd ")
			svcIdx := strings.Index(c.Script, "./svc.sh")
			if cdIdx < 0 || svcIdx < 0 || cdIdx >= svcIdx {
				t.Fatalf("cd must precede ./svc.sh; got cd=%d svc=%d\nscript:\n%s", cdIdx, svcIdx, c.Script)
			}
			return
		}
	}
	t.Fatal("verify_service command not recorded")
}

// Bug 12 / Plan 06-09 — gap doc 06-GAP-byo-sudo-handling.md.
//
// Plan 06-07 attempt-9 against salar@mckee-small-desktop got past Bug 11
// (config.sh cwd) and aborted with the generic message:
//
//   ERROR RunnerKit installed the runner but the service is not active.
//   NEXT  Run sudo ./svc.sh status in the runner directory ...
//
// The actual remote stderr (from svc.sh status / install_service)
// was never surfaced — Apply returned ServiceNotActiveError{Err: err}
// without the failing command's stderr. up.go's handler emits a
// generic remediation, leaving the user unable to diagnose root cause.
//
// Bug 12 fix: extend ServiceNotActiveError with CommandID + Stderr
// fields populated by Apply / ApplyEphemeral, and have up.go's
// handlers render them in the user-facing remediation.

func TestServiceNotActiveError_CarriesCommandIDAndStderr(t *testing.T) {
	t.Parallel()
	e := ServiceNotActiveError{CommandID: "verify_service", Stderr: "Failed to start actions.runner.foo.service: Unit not found"}
	if e.CommandID != "verify_service" {
		t.Fatalf("CommandID field missing or wrong: %#v", e)
	}
	if !strings.Contains(e.Stderr, "Unit not found") {
		t.Fatalf("Stderr field missing or wrong: %#v", e)
	}
}

func TestApply_ServiceFailureSurfacesStderrInError(t *testing.T) {
	t.Parallel()
	exec := &serviceFailingExecutor{
		commands: nil,
		stderr:   "Job for actions.runner.foo.service failed because the control process exited with error code.",
		exitCode: 3,
	}
	opts := Options{
		RunnerName: "runnerkit-x", RepoURL: "https://github.com/owner/repo",
		Labels: []string{"x"}, ServiceUser: "runnerkit-runner", RunnerToken: "tk",
		Package: RunnerPackage{Filename: "r.tgz", URL: "https://x.invalid/r.tgz", SHA256: "abc"},
	}
	_, err := Apply(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, opts)
	if err == nil {
		t.Fatal("expected error from Apply")
	}
	var serviceErr ServiceNotActiveError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("err is not ServiceNotActiveError: %T %v", err, err)
	}
	if serviceErr.CommandID == "" {
		t.Fatalf("ServiceNotActiveError.CommandID empty — user can't tell which step failed")
	}
	if !strings.Contains(serviceErr.Stderr, "control process exited") {
		t.Fatalf("ServiceNotActiveError.Stderr does not surface remote stderr; got %q", serviceErr.Stderr)
	}
}

type serviceFailingExecutor struct {
	commands []remote.Command
	stderr   string
	exitCode int
}

func (s *serviceFailingExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	return remote.ProbeResult{}, nil
}
func (s *serviceFailingExecutor) Run(_ context.Context, _ remote.Target, command remote.Command) (remote.Result, error) {
	s.commands = append(s.commands, command)
	if command.ID == "install_service" || command.ID == "verify_service" {
		return remote.Result{ExitCode: s.exitCode, Stderr: s.stderr}, nil
	}
	return remote.Result{ExitCode: 0}, nil
}

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
				t.Fatalf("configure_runner must have Sudo: true so scoped NOPASSWD sudo applies to `sudo curl/sha256sum/chown/su`. Got Sudo=%v", c.Sudo)
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

func TestMergePackagesDeduplicates(t *testing.T) {
	merged := mergePackages([]string{"curl", "tar"}, []string{"tar", "libsecret-1-dev"})
	want := []string{"curl", "tar", "libsecret-1-dev"}
	if strings.Join(merged, ",") != strings.Join(want, ",") {
		t.Fatalf("mergePackages = %v, want %v", merged, want)
	}
}

func TestMergePackagesEmptyExtra(t *testing.T) {
	original := []string{"curl", "tar"}
	merged := mergePackages(original, nil)
	if len(merged) != 2 || merged[0] != "curl" || merged[1] != "tar" {
		t.Fatalf("mergePackages with nil extras = %v, want %v", merged, original)
	}
}

func TestApplyIncludesExtraPackagesInFixDependencies(t *testing.T) {
	exec := &recordingExecutor{}
	opts := Options{
		RunnerName:    "runnerkit-test",
		RepoURL:       "https://github.com/owner/repo",
		Labels:        []string{"self-hosted"},
		Package:       RunnerPackage{Filename: "runner.tar.gz", URL: "https://example.com/runner.tar.gz", SHA256: "abc123", Version: RunnerVersion, Arch: "x64"},
		RunnerToken:   "fake-token",
		MissingTools:  []string{"curl"},
		ExtraPackages: []string{"libsecret-1-dev", "dbus-x11"},
	}
	_, _ = Apply(context.Background(), exec, remote.Target{Host: "example.com", User: "alice", Port: 22}, opts)
	if len(exec.commands) == 0 {
		t.Fatal("no commands executed")
	}
	fixDeps := exec.commands[0]
	if fixDeps.ID != "fix_dependencies" {
		t.Fatalf("first command = %q, want fix_dependencies", fixDeps.ID)
	}
	for _, pkg := range []string{"curl", "libsecret-1-dev", "dbus-x11"} {
		if !strings.Contains(fixDeps.Script, pkg) {
			t.Fatalf("fix_dependencies script missing %q:\n%s", pkg, fixDeps.Script)
		}
	}
}
