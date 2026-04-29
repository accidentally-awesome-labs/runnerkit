package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/ops"
	"github.com/salar/runnerkit/internal/remote"
	"github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/testsupport"
)

type staticCommandRunner struct{ remote string }

func (r staticCommandRunner) LookPath(name string) (string, error) { return name, nil }
func (r staticCommandRunner) Run(context.Context, string, ...string) (string, error) {
	return r.remote, nil
}

func executeStatusForTest(t *testing.T, stateDir string, github *testsupport.GitHubService, remoteExec *testsupport.RemoteExecutor, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: github, RemoteExecutor: remoteExec, CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	return out.String(), errOut.String(), runErr
}

func saveHealthyState(t *testing.T, stateDir string) state.RepositoryState {
	t.Helper()
	repo := testsupport.HealthyRepositoryState()
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	return repo
}

func healthyStatusRemote() *testsupport.RemoteExecutor {
	return &testsupport.RemoteExecutor{ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"}, Results: map[string]remote.Result{ops.CommandStatusSSHReachable: {ExitCode: 0}, ops.CommandStatusSystemdShow: {Stdout: "LoadState=loaded\nActiveState=active\nSubState=running\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0}}}
}

func TestStatusRepoHumanOutputAndReadOnly(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	remoteExec := healthyStatusRemote()
	out, errOut, err := executeStatusForTest(t, stateDir, github, remoteExec, "status", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("status returned error: %v\nstderr=%s", err, errOut)
	}
	for _, want := range []string{"Step 1 of 1: runner status", "Health: ready", "State", "GitHub", "SSH", "Service", "Labels", "runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]", "Do not use runs-on: self-hosted alone for RunnerKit-managed runners."} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output missing %q:\n%s", want, out)
		}
	}
	if github.CreateRegistrationTokenCalls != 0 || github.CreateRemovalTokenCalls != 0 || github.DeleteRunnerCalls != 0 {
		t.Fatalf("status mutated GitHub: %#v", github)
	}
	for _, command := range remoteExec.Commands {
		if command.ID != ops.CommandStatusSSHReachable && command.ID != ops.CommandStatusSystemdShow {
			t.Fatalf("status ran non-status remote command: %#v", command)
		}
	}
}

func TestStatusInferredGitRemoteAllJSONLabelDriftAndHostKeyMismatch(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{{ID: 123, Name: repo.Runner.Name, Status: "online", Labels: []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "gpu"}}}}
	out, _, err := executeStatusForTest(t, stateDir, github, healthyStatusRemote(), "--json", "status", "--no-color")
	if err != nil {
		t.Fatalf("inferred json status returned error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	if payload["repo"] != "owner/repo" || payload["redactions_applied"] != true {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if !strings.Contains(out, "label_drift") || !strings.Contains(out, "persistent") || !strings.Contains(out, "gpu") {
		t.Fatalf("json status missing label drift facts:\n%s", out)
	}

	hostKeyRemote := &testsupport.RemoteExecutor{ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:changed"}}
	out, _, err = executeStatusForTest(t, stateDir, &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}, hostKeyRemote, "status", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("host-key mismatch status should still render: %v", err)
	}
	if !strings.Contains(out, "host key mismatch") || !strings.Contains(out, "broken") {
		t.Fatalf("host-key mismatch not rendered:\n%s", out)
	}

	out, _, err = executeStatusForTest(t, stateDir, &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}, healthyStatusRemote(), "status", "--all", "--no-color")
	if err != nil {
		t.Fatalf("status --all returned error: %v", err)
	}
	if !strings.Contains(out, "owner/repo: ready") {
		t.Fatalf("status --all missing inventory row:\n%s", out)
	}
}

func TestStatusMissingStateEmptyCopy(t *testing.T) {
	out, _, err := executeStatusForTest(t, t.TempDir(), &testsupport.GitHubService{}, healthyStatusRemote(), "status", "--repo", "owner/repo", "--no-color")
	if err != nil {
		t.Fatalf("missing state should render empty status, got %v", err)
	}
	if !strings.Contains(out, "No RunnerKit-managed runner found") || !strings.Contains(out, "Run runnerkit up --repo owner/name --host user@host") || !strings.Contains(out, "pass --all to list saved runners") {
		t.Fatalf("missing state copy not rendered:\n%s", out)
	}
}
