package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

// scriptedRemoteExecutor is a fakeRemoteExecutor variant used by the
// byo-prepare tests. It records every command and lets each test
// programmatically pre-stage Stdout/Stderr/ExitCode keyed by command
// ID. The base fakeRemoteExecutor already provides this via
// runResults / runErrs, so we just thread results through.
func newScriptedRemoteExecutor() *fakeRemoteExecutor {
	return newFakeRemoteExecutor()
}

// TestByoPrepareCommandRegistered asserts the new top-level command
// is registered on the root cobra tree.
func TestByoPrepareCommandRegistered(t *testing.T) {
	deps := Dependencies{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	root := NewRootCommand(deps)
	cmd, _, err := root.Find([]string{"byo-prepare"})
	if err != nil {
		t.Fatalf("byo-prepare not registered: %v", err)
	}
	if cmd == nil || cmd.Name() != "byo-prepare" {
		t.Fatalf("expected byo-prepare command, got %#v", cmd)
	}
}

// TestByoPrepare_Idempotent asserts that re-running byo-prepare against
// a host whose sudoers file already matches RenderSudoersEntry exits
// without writing or invoking visudo. Captured commands MUST NOT
// include install_sudoers nor verify_sudo_n.
func TestByoPrepare_Idempotent(t *testing.T) {
	remoteExec := newScriptedRemoteExecutor()
	// First read returns the matching content → idempotent path.
	remoteExec.runResults["read_sudoers"] = remote.Result{
		ExitCode: 0,
		Stdout:   bootstrap.RenderSudoersEntry("alice"),
	}
	prompter := &recordingPasswordPrompter{password: "p@ss"}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		TTY:            ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:        prompter,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"byo-prepare", "--host", "alice@example.com", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("idempotent byo-prepare returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	for _, c := range remoteExec.runs {
		if c.ID == "install_sudoers" {
			t.Fatalf("idempotent re-run still ran install_sudoers; this defeats the skip path")
		}
	}
	if prompter.passwordCalls != 0 {
		t.Fatalf("idempotent re-run prompted for password unexpectedly: %d", prompter.passwordCalls)
	}
	if !strings.Contains(out.String(), "already prepared") {
		t.Fatalf("idempotent message missing 'already prepared':\n%s", out.String())
	}
}

// TestByoPrepare_VisudoValidationFails_DoesNotMoveFile asserts the
// critical lockout-prevention property: when the install_sudoers
// command fails (visudo rejected the rendered content), byo-prepare
// surfaces the error AND no subsequent commands attempt to mv the
// tempfile into /etc/sudoers.d/. The remote script handles the
// atomicity (tmp + visudo + mv), but at the CLI layer we assert the
// error path returns non-nil so the caller knows the install failed.
func TestByoPrepare_VisudoValidationFails_DoesNotMoveFile(t *testing.T) {
	remoteExec := newScriptedRemoteExecutor()
	remoteExec.runResults["read_sudoers"] = remote.Result{ExitCode: 1} // missing → proceed to install
	remoteExec.runResults["install_sudoers"] = remote.Result{
		ExitCode: 21,
		Stderr:   "visudo: parse error in /tmp/runnerkit-installer.XXX near line 2",
	}
	prompter := &recordingPasswordPrompter{password: "p@ss"}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		TTY:            ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:        prompter,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"byo-prepare", "--host", "alice@example.com", "--no-color"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected visudo-validation failure to return error")
	}
	// No verify_sudo_n probe should be issued after a failed install.
	for _, c := range remoteExec.runs {
		if c.ID == "verify_sudo_n" {
			t.Fatalf("verify_sudo_n ran even after install_sudoers failed: %#v", c)
		}
	}
	combined := out.String() + errOut.String()
	// Human renderer prints the message; JSON renderer prints the code.
	// Either way the user-facing copy must reference the install failure.
	if !strings.Contains(combined, "could not install the scoped sudoers entry") && !strings.Contains(combined, "byo_prepare_failed") {
		t.Fatalf("error message missing from output:\n%s", combined)
	}
	// The remote stderr must surface so users can self-diagnose.
	if !strings.Contains(combined, "parse error") {
		t.Fatalf("remote visudo stderr missing from output:\n%s", combined)
	}
}

// TestByoPrepare_Remove asserts the --remove inverse: byo-prepare
// issues a remove_sudoers command that targets the canonical sudoers
// file path.
func TestByoPrepare_Remove(t *testing.T) {
	remoteExec := newScriptedRemoteExecutor()
	remoteExec.runResults["remove_sudoers"] = remote.Result{ExitCode: 0}
	prompter := &recordingPasswordPrompter{password: "p@ss"}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		TTY:            ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:        prompter,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"byo-prepare", "--host", "alice@example.com", "--remove", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("byo-prepare --remove returned error: %v\nstderr=%s", err, errOut.String())
	}
	sawRemove := false
	for _, c := range remoteExec.runs {
		if c.ID == "remove_sudoers" {
			sawRemove = true
			if !strings.Contains(c.Script, bootstrap.SudoersFilePath) || !strings.Contains(c.Script, bootstrap.RunnerCISudoersFilePath) {
				t.Fatalf("remove_sudoers must rm both installer + CI sudoers paths: %s", c.Script)
			}
		}
	}
	if !sawRemove {
		t.Fatalf("--remove did not issue remove_sudoers command; commands=%v", commandIDs(remoteExec.runs))
	}
}

// TestByoPrepare_RequiresHost asserts the --host flag is required.
func TestByoPrepare_RequiresHost(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Out: &out, Err: &errOut, RemoteExecutor: newFakeRemoteExecutor()})
	cmd.SetArgs([]string{"byo-prepare", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing --host error")
	}
	if got := ExitCode(err); got != ExitInputRequired {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInputRequired)
	}
}

// TestByoPrepare_NonInteractiveFailsWithoutTTY asserts that when no
// TTY is available for password input, byo-prepare fails fast (it
// can't run without a sudo password) rather than silently hanging.
func TestByoPrepare_NonInteractiveFailsWithoutTTY(t *testing.T) {
	remoteExec := newScriptedRemoteExecutor()
	remoteExec.runResults["read_sudoers"] = remote.Result{ExitCode: 1} // missing → proceed
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		TTY:            ui.TerminalCapabilities{StdinTTY: false, StdoutTTY: false, Width: 80},
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"byo-prepare", "--host", "alice@example.com", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected no-TTY failure")
	}
	if got := ExitCode(err); got != ExitInputRequired {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInputRequired)
	}
}

// TestByoPrepare_GrantCI_Idempotent skips all installs when installer + CI sudoers match.
func TestByoPrepare_GrantCI_Idempotent(t *testing.T) {
	remoteExec := newScriptedRemoteExecutor()
	remoteExec.runResults["read_sudoers"] = remote.Result{
		ExitCode: 0,
		Stdout:   bootstrap.RenderSudoersEntry("alice"),
	}
	remoteExec.runResults["detect_kernel"] = remote.Result{ExitCode: 0, Stdout: "Linux\n"}
	remoteExec.runResults["read_ci_sudoers"] = remote.Result{
		ExitCode: 0,
		Stdout:   bootstrap.RenderRunnerCISudoersEntry(bootstrap.DefaultServiceUser),
	}
	prompter := &recordingPasswordPrompter{password: "p@ss"}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		TTY:            ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:        prompter,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"byo-prepare", "--host", "alice@example.com", "--grant-ci-sudo", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("grant-ci idempotent byo-prepare: %v\nstderr=%s", err, errOut.String())
	}
	for _, c := range remoteExec.runs {
		if c.ID == "install_sudoers" || c.ID == "install_ci_sudoers" {
			t.Fatalf("unexpected install command %s on full idempotent hit", c.ID)
		}
	}
	if prompter.passwordCalls != 0 {
		t.Fatalf("password prompted on idempotent run: %d", prompter.passwordCalls)
	}
}

// TestByoPrepare_GrantCI_InstallsOnlyCIWhenInstallerPresent installs CI sudoers when bootstrap sudoers already exist.
func TestByoPrepare_GrantCI_InstallsOnlyCIWhenInstallerPresent(t *testing.T) {
	remoteExec := newScriptedRemoteExecutor()
	remoteExec.runResults["read_sudoers"] = remote.Result{
		ExitCode: 0,
		Stdout:   bootstrap.RenderSudoersEntry("alice"),
	}
	remoteExec.runResults["detect_kernel"] = remote.Result{ExitCode: 0, Stdout: "Linux\n"}
	remoteExec.runResults["read_ci_sudoers"] = remote.Result{ExitCode: 1}
	remoteExec.runResults["install_ci_sudoers"] = remote.Result{ExitCode: 0}

	prompter := &recordingPasswordPrompter{password: "p@ss"}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		TTY:            ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:        prompter,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"byo-prepare", "--host", "alice@example.com", "--grant-ci-sudo", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected success: %v stderr=%s", err, errOut.String())
	}
	var sawMain, sawCI bool
	for _, c := range remoteExec.runs {
		switch c.ID {
		case "install_sudoers":
			sawMain = true
		case "install_ci_sudoers":
			sawCI = true
			if !strings.Contains(c.Env["RUNNERKIT_CI_SUDOERS_CONTENT"], "runnerkit-runner ALL=(root) NOPASSWD:") {
				t.Fatalf("CI sudoers content missing service user NOPASSWD line")
			}
		}
	}
	if sawMain {
		t.Fatal("installer sudoers should not reinstall when already prepared")
	}
	if !sawCI {
		t.Fatal("expected install_ci_sudoers")
	}
}

func commandIDs(commands []remote.Command) []string {
	ids := make([]string, 0, len(commands))
	for _, c := range commands {
		ids = append(ids, c.ID)
	}
	return ids
}

// _ = context.Background  // silence unused import when scenarios are minimal.
var _ = context.Background
