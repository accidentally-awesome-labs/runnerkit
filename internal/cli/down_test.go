package cli

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/ops"
	"github.com/salar/runnerkit/internal/remote"
	"github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/testsupport"
	"github.com/salar/runnerkit/internal/ui"
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
