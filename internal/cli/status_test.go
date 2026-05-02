package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/ops"
	"github.com/salar/runnerkit/internal/provider"
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
	for _, want := range []string{"No RunnerKit-managed runner found", "Run runnerkit up --repo owner/name --host user@host", "pass --all to list saved runners", "No RunnerKit-managed cloud runner is saved for `owner/name`.", "Run `runnerkit up --repo owner/name --cloud hetzner` to create one", "`--host user@host` to use an existing machine."} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing state copy missing %q:\n%s", want, out)
		}
	}
}

func TestStatusEphemeralCompletedTreatsMissingGitHubRunnerAsTerminal(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.HealthyRepositoryState()
	repo.Runner.Mode = "ephemeral"
	repo.Runner.Name = "runnerkit-owner-repo-ephemeral-fake1"
	repo.Runner.Labels = []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "ephemeral"}
	repo.Machine.ServiceName = "runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-fake1.service"
	repo.Safety.SafetyProfile = "ephemeral-byo"
	repo.Ephemeral = state.EphemeralMetadata{Enabled: true, TTL: "24h", LogArchivePath: "/var/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-fake1/logs", FinalizerStatus: "completed", CleanupCommand: "runnerkit down --repo owner/repo"}
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	github := &testsupport.GitHubService{Runners: nil}
	exec := &testsupport.RemoteExecutor{ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"}, Results: map[string]remote.Result{
		ops.CommandStatusSSHReachable: {ExitCode: 0},
		ops.CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=inactive\nSubState=dead\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0},
		"status.ephemeral.state":      {Stdout: `{"finalizer_status":"completed"}`, ExitCode: 0},
	}}
	out, _, err := executeStatusForTest(t, stateDir, github, exec, "status", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("ephemeral completed status returned error: %v", err)
	}
	flat := strings.Join(strings.Fields(out), " ")
	if !strings.Contains(flat, "Ephemeral runner completed one job and needs cleanup.") {
		t.Fatalf("expected ephemeral_completed summary in:\n%s", out)
	}
	for _, want := range []string{"Mode: ephemeral", "Safety profile: ephemeral-byo", "Log archive: /var/lib/runnerkit/ephemeral/", "Cleanup after the job: runnerkit down --repo owner/repo"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ephemeral status missing %q (flattened):\n%s", want, out)
		}
	}
	if strings.Contains(flat, "github_runner_missing") {
		t.Fatalf("ephemeral_completed should not surface github_runner_missing:\n%s", out)
	}
}

func TestStatusMissingStateRendersUISpecEmptyCopy(t *testing.T) {
	out, _, err := executeStatusForTest(t, t.TempDir(), &testsupport.GitHubService{}, healthyStatusRemote(), "status", "--repo", "owner/repo", "--no-color")
	if err != nil {
		t.Fatalf("missing state status returned error: %v", err)
	}
	flat := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"No RunnerKit-managed runner is saved for `owner/repo`.",
		"Run `runnerkit up --repo owner/repo --mode ephemeral --cloud hetzner` for a one-job cloud runner, or use `--host user@host` for an existing machine.",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ui-spec empty state missing %q (flattened):\n%s", want, out)
		}
	}
}

// TestStatusEphemeralCompletedNeedsCleanup proves that a cloud ephemeral
// repository with FinalizerStatus=completed (read from the remote
// sentinel by status.ephemeral.state) renders the UI-SPEC summary
// "Ephemeral runner completed one job and needs cleanup.", surfaces the
// cleanup command, prints the saved log archive bullet, and does NOT
// surface the persistent runnerkit recover hint.
func TestStatusEphemeralCompletedNeedsCleanup(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.EphemeralCloudRepositoryState()
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	github := &testsupport.GitHubService{Runners: nil}
	exec := &testsupport.RemoteExecutor{ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"}, Results: map[string]remote.Result{
		ops.CommandStatusSSHReachable: {ExitCode: 0},
		ops.CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=inactive\nSubState=dead\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0},
		"status.ephemeral.state":      {Stdout: `{"finalizer_status":"completed"}`, ExitCode: 0},
	}}
	cloud := &provider.FakeProvider{DescribeOut: provider.ProviderStatus{Kind: "hetzner", Found: true, Status: "running", Region: "fsn1", ServerType: "cpx22", Image: "ubuntu-24.04"}}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: github, RemoteExecutor: exec, Providers: provider.NewRegistry(cloud), CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
	cmd.SetArgs([]string{"status", "--repo", repo.Repo.FullName, "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral completed status returned error: %v", err)
	}
	flat := strings.Join(strings.Fields(out.String()), " ")
	for _, want := range []string{
		"Ephemeral runner completed one job and needs cleanup.",
		"Next: runnerkit destroy --repo owner/repo",
		"Log archive: /var/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-20260501t183000/logs",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ephemeral_completed status missing %q (flattened):\n%s", want, out.String())
		}
	}
	if strings.Contains(flat, "runnerkit recover --repo") {
		t.Fatalf("ephemeral_completed status must not surface runnerkit recover hint:\n%s", out.String())
	}
}

// TestMissingStateRendersRunnerKitEmptyState asserts the UI-SPEC empty
// state heading and remediation copy when no RunnerKit state exists for
// the requested repository.
func TestMissingStateRendersRunnerKitEmptyState(t *testing.T) {
	out, _, err := executeStatusForTest(t, t.TempDir(), &testsupport.GitHubService{}, healthyStatusRemote(), "status", "--repo", "owner/name", "--no-color")
	if err != nil {
		t.Fatalf("missing state status returned error: %v", err)
	}
	flat := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"No RunnerKit-managed runner is saved for `owner/name`.",
		"Run `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner` for a one-job cloud runner, or use `--host user@host` for an existing machine.",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ui-spec empty state missing %q (flattened):\n%s", want, out)
		}
	}
}

// TestStatusEphemeralTTLExpired proves that ExpiresAt in the past plus a
// remote sentinel reporting ttl_expired surfaces the UI-SPEC TTL-expired
// summary plus the `ephemeral_ttl_expired` reason in JSON output.
func TestStatusEphemeralTTLExpired(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.EphemeralCloudRepositoryState()
	expired := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	repo.Ephemeral.ExpiresAt = &expired
	repo.Ephemeral.FinalizerStatus = "ttl_expired"
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	exec := &testsupport.RemoteExecutor{ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"}, Results: map[string]remote.Result{
		ops.CommandStatusSSHReachable: {ExitCode: 0},
		ops.CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=inactive\nSubState=dead\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0},
		"status.ephemeral.state":      {Stdout: `{"finalizer_status":"ttl_expired"}`, ExitCode: 0},
	}}
	cloud := &provider.FakeProvider{DescribeOut: provider.ProviderStatus{Kind: "hetzner", Found: true, Status: "running", Region: "fsn1", ServerType: "cpx22", Image: "ubuntu-24.04"}}
	runStatus := func(args ...string) string {
		var out, errOut bytes.Buffer
		cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: &testsupport.GitHubService{}, RemoteExecutor: exec, Providers: provider.NewRegistry(cloud), CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("ttl-expired status returned error: %v\nstderr=%s", err, errOut.String())
		}
		return out.String()
	}
	humanOut := runStatus("status", "--repo", repo.Repo.FullName, "--no-color")
	flat := strings.Join(strings.Fields(humanOut), " ")
	if !strings.Contains(flat, "Ephemeral runner TTL expired before a job completed. Run cleanup now.") {
		t.Fatalf("expected ttl_expired summary in:\n%s", humanOut)
	}
	jsonOut := runStatus("--json", "status", "--repo", repo.Repo.FullName, "--no-color")
	if !strings.Contains(jsonOut, `"ephemeral_ttl_expired"`) {
		t.Fatalf("expected ephemeral_ttl_expired in JSON reasons:\n%s", jsonOut)
	}
}

func TestStatusCloudProviderJSONAndReadOnly(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.CloudRepositoryState()
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	cloud := &provider.FakeProvider{DescribeOut: provider.ProviderStatus{Kind: "hetzner", Found: true, Status: "running", Region: "fsn1", ServerType: "cpx22", Image: "ubuntu-24.04", PublicHost: "203.0.113.10", BillableResources: []string{"server:srv-123", "ssh_key:key-123", "firewall:fw-123"}}}
	var out, errOut bytes.Buffer
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: github, RemoteExecutor: healthyStatusRemote(), Providers: provider.NewRegistry(cloud), CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
	cmd.SetArgs([]string{"--json", "status", "--repo", repo.Repo.FullName, "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cloud json status returned error: %v\nstderr=%s", err, errOut.String())
	}
	for _, want := range []string{`"sources"`, `"provider"`, `"kind":"hetzner"`, `"found":true`, `"status":"running"`, `"region":"fsn1"`, `"server_type":"cpx22"`, `"image":"ubuntu-24.04"`, `"public_host":"203.0.113.10"`, `"billable_resources"`, `"drift"`, `"error"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("cloud status json missing %q:\n%s", want, out.String())
		}
	}
	if cloud.DescribeCalls != 1 || cloud.ProvisionCalls != 0 || cloud.DestroyCalls != 0 || cloud.VerifyDestroyedCalls != 0 || github.CreateRegistrationTokenCalls != 0 || github.CreateRemovalTokenCalls != 0 {
		t.Fatalf("status mutated dependencies: cloud=%#v github=%#v", cloud, github)
	}
}
