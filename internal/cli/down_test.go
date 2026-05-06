package cli

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

type promptRecorder struct{ messages []string }

func (p *promptRecorder) Confirm(_ context.Context, prompt ui.Prompt) (bool, error) {
	p.messages = append(p.messages, prompt.Message)
	return prompt.Default, nil
}
func (p *promptRecorder) Select(context.Context, ui.Prompt, []ui.Option) (string, error) {
	return "", nil
}

func downRemote(statusExit int) *testsupport.RemoteExecutor {
	return &testsupport.RemoteExecutor{ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"}, Results: map[string]remote.Result{
		ops.CommandStatusSSHReachable: {ExitCode: statusExit},
		ops.CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=active\nSubState=running\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0},
		"down.runner.remove":          {ExitCode: 0},
		"down.service.uninstall":      {ExitCode: 0},
		"down.files.remove":           {ExitCode: 0},
	}}
}

func executeDownForTest(t *testing.T, stateDir string, github *testsupport.GitHubService, remoteExec *testsupport.RemoteExecutor, prompts ui.Prompter, tty bool, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut strings.Builder
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: github, RemoteExecutor: remoteExec, CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Prompts: prompts, TTY: ui.TerminalCapabilities{StdinTTY: tty, StdoutTTY: false, Width: 80}, Sleep: noSleep, Clock: func() time.Time { return time.Date(2026, 4, 30, 1, 0, 0, 0, time.UTC) }})
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	return out.String(), errOut.String(), runErr
}

func TestDownDryRunJSONNoTTYAndInteractiveDefaults(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	remoteExec := downRemote(0)
	out, _, err := executeDownForTest(t, stateDir, github, remoteExec, nil, false, "down", "--repo", repo.Repo.FullName, "--dry-run", "--no-color")
	if err != nil {
		t.Fatalf("down dry-run returned error: %v", err)
	}
	if !strings.Contains(out, "Step 1 of 1: cleanup plan") || !strings.Contains(out, "This will remove RunnerKit-managed runner artifacts for owner/repo.") || !strings.Contains(out, "Next: answer each prompt") {
		t.Fatalf("dry-run missing plan:\n%s", out)
	}
	if github.DeleteRunnerCalls != 0 || len(remoteExec.Commands) != 0 {
		t.Fatalf("dry-run mutated: github=%#v commands=%#v", github, remoteExec.CommandIDs())
	}
	out, _, err = executeDownForTest(t, stateDir, github, downRemote(0), nil, false, "--json", "down", "--repo", repo.Repo.FullName, "--dry-run", "--no-color")
	if err != nil || !strings.Contains(out, `"partial_cleanup":false`) || !strings.Contains(out, `"state_removed":false`) {
		t.Fatalf("json dry-run failed err=%v out=%s", err, out)
	}
	_, _, err = executeDownForTest(t, stateDir, github, downRemote(0), nil, false, "down", "--repo", repo.Repo.FullName, "--no-color")
	if err == nil || ExitCode(err) != ExitInputRequired {
		t.Fatalf("missing --yes ExitCode=%d err=%v", ExitCode(err), err)
	}
	prompts := &promptRecorder{}
	_, _, err = executeDownForTest(t, stateDir, github, downRemote(0), prompts, true, "down", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("interactive default no returned error: %v", err)
	}
	if len(prompts.messages) != 5 || !strings.Contains(strings.Join(prompts.messages, "\n"), "Remove GitHub runner runnerkit-owner-repo-local from owner/repo? [y/N]") {
		t.Fatalf("prompt messages missing: %#v", prompts.messages)
	}
}

func TestDownYesCompleteCleanupDeletesStateAndRedactsToken(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	removalToken := strings.Join([]string{"removal", "token", "down", "secret"}, "-")
	github := &testsupport.GitHubService{RemovalToken: gh.RunnerToken{Token: removalToken, ExpiresAt: time.Now().Add(time.Hour)}, Runners: []gh.Runner{testsupport.HealthyRunner()}}
	remoteExec := downRemote(0)
	out, errOut, err := executeDownForTest(t, stateDir, github, remoteExec, nil, false, "--json", "down", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("down --yes returned error: %v\nstderr=%s", err, errOut)
	}
	for _, want := range []string{"down.runner.remove", "down.service.uninstall", "down.files.remove"} {
		if !commandIDsContain(remoteExec, want) {
			t.Fatalf("down missing command %q in %#v", want, remoteExec.CommandIDs())
		}
	}
	if github.DeleteRunnerCalls != 1 || len(github.DeletedRunnerIDs) != 1 || github.DeletedRunnerIDs[0] != 123 {
		t.Fatalf("expected DeleteRunner ID 123: %#v", github)
	}
	if strings.Contains(out, removalToken) || strings.Contains(errOut, removalToken) {
		t.Fatalf("down leaked removal token stdout=%s stderr=%s", out, errOut)
	}
	if !strings.Contains(out, `"state_removed":true`) || strings.Contains(out, `"partial_cleanup":true`) {
		t.Fatalf("down json missing complete cleanup fields:\n%s", out)
	}
	stateBytes, err := os.ReadFile(state.NewStore(stateDir).Path())
	if err != nil {
		t.Fatalf("read state after cleanup: %v", err)
	}
	if strings.Contains(string(stateBytes), "owner/repo") {
		t.Fatalf("state still contains owner/repo after cleanup:\n%s", stateBytes)
	}
}

// Bug 21 (Plan 06-10, 2026-05-06): runnerkit down --yes against a BYO
// host with password-protected sudo currently fails at the
// `runner_files` cleanup step with `sudo: a terminal is required to
// read the password`. Down's remote cleanup must thread the sudo
// password through `printf | sudo -S` the same way Plan 06-09 Bug 10's
// wrapSudoCommand does for bootstrap. This test:
//   - probes sudo via the canonical `down.sudo.probe` command,
//   - the probe reports password-required (exit code 1, stderr
//     contains "password is required" or "a terminal is required"),
//   - down prompts via the password prompter (interactive TTY),
//   - and the resulting service-uninstall + files-remove commands
//     have their Script prefixed with `printf '%s\n'
//     "$RUNNERKIT_SUDO_PASSWORD" | sudo -S -v` and Env carries
//     RUNNERKIT_SUDO_PASSWORD.
func TestDownThreadsSudoPasswordWhenSudoRequiresPasswordClosesBug21(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	github := &testsupport.GitHubService{
		RemovalToken: gh.RunnerToken{Token: "down-removal-token", ExpiresAt: time.Now().Add(time.Hour)},
		Runners:      []gh.Runner{testsupport.HealthyRunner()},
	}
	exec := &testsupport.RemoteExecutor{
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
		Results: map[string]remote.Result{
			ops.CommandStatusSSHReachable: {ExitCode: 0},
			ops.CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=active\nSubState=running\n", ExitCode: 0},
			"down.sudo.probe":             {ExitCode: 1, Stderr: "sudo: a password is required\n"},
			"down.runner.remove":          {ExitCode: 0},
			"down.service.uninstall":      {ExitCode: 0},
			"down.files.remove":           {ExitCode: 0},
		},
	}
	prompts := &passwordRecorder{password: "hunter2"}
	out, errOut, err := executeDownForTest(t, stateDir, github, exec, prompts, true, "down", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("down with password-protected sudo returned error: %v\nstderr=%s\nstdout=%s", err, errOut, out)
	}
	for _, want := range []string{"down.sudo.probe", "down.service.uninstall", "down.files.remove"} {
		if !commandIDsContain(exec, want) {
			t.Fatalf("down missing command %q in %v", want, exec.CommandIDs())
		}
	}
	for _, command := range exec.Commands {
		if command.ID != "down.service.uninstall" && command.ID != "down.files.remove" {
			continue
		}
		if !strings.Contains(command.Script, `printf '%s\n' "$RUNNERKIT_SUDO_PASSWORD" | sudo -S -v`) {
			t.Fatalf("command %q must thread sudo password via printf|sudo -S -v; got script:\n%s", command.ID, command.Script)
		}
		if command.Env == nil || command.Env["RUNNERKIT_SUDO_PASSWORD"] != "hunter2" {
			t.Fatalf("command %q Env must carry RUNNERKIT_SUDO_PASSWORD=hunter2; got %#v", command.ID, command.Env)
		}
		if !containsString(command.RedactArgs, "hunter2") {
			t.Fatalf("command %q RedactArgs must contain the sudo password literal; got %#v", command.ID, command.RedactArgs)
		}
	}
	if strings.Contains(out, "hunter2") || strings.Contains(errOut, "hunter2") {
		t.Fatalf("sudo password leaked in output: stdout=%s stderr=%s", out, errOut)
	}
}

// When sudo does NOT require a password (NOPASSWD path / Path C
// byo-prepare), down must NOT prompt and must NOT wrap the cleanup
// commands — preserving the existing happy path.
func TestDownDoesNotPromptWhenSudoIsPasswordless(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	github := &testsupport.GitHubService{
		RemovalToken: gh.RunnerToken{Token: "down-removal-token", ExpiresAt: time.Now().Add(time.Hour)},
		Runners:      []gh.Runner{testsupport.HealthyRunner()},
	}
	exec := &testsupport.RemoteExecutor{
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
		Results: map[string]remote.Result{
			ops.CommandStatusSSHReachable: {ExitCode: 0},
			ops.CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=active\nSubState=running\n", ExitCode: 0},
			"down.sudo.probe":             {ExitCode: 0}, // sudo -n true succeeded
			"down.runner.remove":          {ExitCode: 0},
			"down.service.uninstall":      {ExitCode: 0},
			"down.files.remove":           {ExitCode: 0},
		},
	}
	prompts := &passwordRecorder{}
	_, _, err := executeDownForTest(t, stateDir, github, exec, prompts, true, "down", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("down with passwordless sudo returned error: %v", err)
	}
	if prompts.calls != 0 {
		t.Fatalf("must NOT prompt when sudo is passwordless; got prompt calls=%d", prompts.calls)
	}
	for _, command := range exec.Commands {
		if strings.Contains(command.Script, "RUNNERKIT_SUDO_PASSWORD") {
			t.Fatalf("passwordless path must not wrap commands with sudo password; got command %q script:\n%s", command.ID, command.Script)
		}
	}
}

// passwordRecorder is a minimal Prompter + PasswordPrompter test
// double for Bug 21 — Confirm/Select aren't exercised by --yes paths,
// but Password is the load-bearing capability.
type passwordRecorder struct {
	password string
	calls    int
}

func (p *passwordRecorder) Confirm(context.Context, ui.Prompt) (bool, error) { return false, nil }
func (p *passwordRecorder) Select(context.Context, ui.Prompt, []ui.Option) (string, error) {
	return "", nil
}
func (p *passwordRecorder) Password(_ context.Context, _ ui.Prompt) (string, error) {
	p.calls++
	return p.password, nil
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func TestDownPartialAndStaleGitHubOnlyFlows(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	out, _, err := executeDownForTest(t, stateDir, github, downRemote(1), nil, false, "--json", "down", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("ssh-unreachable partial down returned error: %v", err)
	}
	if !strings.Contains(out, `"partial_cleanup":true`) || !strings.Contains(out, "remote_cleanup_pending") || github.DeleteRunnerCalls != 1 {
		t.Fatalf("partial cleanup missing pending/github delete: out=%s github=%#v", out, github)
	}
	loaded, _, err := state.NewStore(stateDir).GetRepository(repo.Repo.FullName)
	if err != nil || len(loaded.Operations) == 0 || loaded.Operations[0].Message != "SSH unreachable during cleanup" {
		t.Fatalf("partial checkpoint not persisted: %#v err=%v", loaded, err)
	}

	staleGitHub := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	out, _, err = executeDownForTest(t, t.TempDir(), staleGitHub, downRemote(0), nil, false, "--json", "down", "--repo", repo.Repo.FullName, "--github-runner-id", "123", "--yes", "--no-color")
	if err != nil || staleGitHub.DeleteRunnerCalls != 1 || staleGitHub.DeletedRunnerIDs[0] != 123 || strings.Contains(out, "remote_cleanup_pending") {
		t.Fatalf("stale GitHub-only deletion failed err=%v out=%s github=%#v", err, out, staleGitHub)
	}

	ambiguous := &testsupport.GitHubService{Runners: []gh.Runner{{ID: 1, Name: "runnerkit-owner-repo-local", Labels: []string{"runnerkit"}}, {ID: 2, Name: "runnerkit-owner-repo-local", Labels: []string{"runnerkit"}}}}
	_, _, err = executeDownForTest(t, t.TempDir(), ambiguous, downRemote(0), nil, false, "down", "--repo", repo.Repo.FullName, "--runner-name", "runnerkit-owner-repo-local", "--yes", "--no-color")
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("ambiguous runner-name should block ExitCode=%d err=%v", ExitCode(err), err)
	}
}

func TestDownEphemeralBYOPreservesLogsBeforeFileRemoval(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.HealthyRepositoryState()
	repo.Runner.Mode = "ephemeral"
	repo.Runner.Name = "runnerkit-owner-repo-ephemeral-fake1"
	repo.Machine.InstallPath = "/opt/actions-runner/runnerkit-owner-repo-ephemeral-fake1"
	repo.Machine.WorkDir = "/var/lib/runnerkit/work/runnerkit-owner-repo-ephemeral-fake1"
	repo.Machine.ServiceName = "runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-fake1.service"
	repo.Safety.SafetyProfile = "ephemeral-byo"
	repo.Ephemeral = state.EphemeralMetadata{Enabled: true, TTL: "24h", LogArchivePath: "/var/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-fake1/logs", FinalizerStatus: "completed", CleanupCommand: "runnerkit down --repo owner/repo"}
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	exec := downRemote(0)
	exec.Results["ephemeral.logs.preserve"] = remote.Result{ExitCode: 0}
	out, errOut, err := executeDownForTest(t, stateDir, github, exec, nil, false, "down", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("ephemeral down returned error: %v\nstderr=%s", err, errOut)
	}
	preserveIdx := -1
	filesIdx := -1
	for i, command := range exec.Commands {
		if command.ID == "ephemeral.logs.preserve" {
			preserveIdx = i
		}
		if command.ID == "down.files.remove" {
			filesIdx = i
		}
	}
	if preserveIdx < 0 {
		t.Fatalf("expected ephemeral.logs.preserve in: %v", exec.CommandIDs())
	}
	if filesIdx < 0 || preserveIdx >= filesIdx {
		t.Fatalf("ephemeral.logs.preserve must run before down.files.remove: preserve=%d files=%d ids=%v", preserveIdx, filesIdx, exec.CommandIDs())
	}
	_ = out
}

// TestDownEphemeralPreservesLogsBeforeRemovingFiles proves that
// runnerkit down on an ephemeral BYO state runs the
// `ephemeral.logs.preserve` remote command before `down.files.remove`
// so finalizer/diag/journal logs survive cleanup.
func TestDownEphemeralPreservesLogsBeforeRemovingFiles(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.EphemeralBYORepositoryState()
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	github := &testsupport.GitHubService{Runners: []gh.Runner{{ID: 999, Name: repo.Runner.Name, OS: "linux", Status: "online", Labels: append([]string(nil), repo.Runner.Labels...)}}}
	exec := downRemote(0)
	exec.Results["ephemeral.logs.preserve"] = remote.Result{ExitCode: 0}
	repo.Cleanup.GitHubRunnerID = 999
	// Re-save with the updated GitHub runner ID so down can match.
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("re-save state: %v", err)
	}
	out, errOut, err := executeDownForTest(t, stateDir, github, exec, nil, false, "down", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("ephemeral down returned error: %v\nstderr=%s", err, errOut)
	}
	preserveIdx := -1
	filesIdx := -1
	for i, command := range exec.Commands {
		if command.ID == "ephemeral.logs.preserve" {
			preserveIdx = i
		}
		if command.ID == "down.files.remove" {
			filesIdx = i
		}
	}
	if preserveIdx < 0 {
		t.Fatalf("expected ephemeral.logs.preserve in: %v", exec.CommandIDs())
	}
	if filesIdx < 0 || preserveIdx >= filesIdx {
		t.Fatalf("ephemeral.logs.preserve must run before down.files.remove: preserve=%d files=%d ids=%v", preserveIdx, filesIdx, exec.CommandIDs())
	}
	_ = out
}

func TestDownGitHubDeleteErrorKeepsPendingState(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}, DeleteRunnerErr: errors.New("delete failed")}
	out, _, err := executeDownForTest(t, stateDir, github, downRemote(0), nil, false, "--json", "down", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("github delete pending should not hard fail: %v", err)
	}
	if !strings.Contains(out, "github_cleanup_pending") || !strings.Contains(out, `"state_removed":false`) {
		t.Fatalf("github pending output missing:\n%s", out)
	}
	loaded, found, err := state.NewStore(stateDir).GetRepository(repo.Repo.FullName)
	if err != nil || !found || len(loaded.Operations) == 0 || loaded.Operations[0].Artifact != "github_runner" {
		t.Fatalf("pending github cleanup not persisted found=%v err=%v state=%#v", found, err, loaded)
	}
}
