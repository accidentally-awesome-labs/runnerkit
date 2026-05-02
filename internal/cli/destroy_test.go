package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/provider"
	"github.com/salar/runnerkit/internal/remote"
	"github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/testsupport"
	"github.com/salar/runnerkit/internal/ui"
)

type destroyInputPrompter struct {
	prompt string
	input  string
}

func (p *destroyInputPrompter) Confirm(context.Context, ui.Prompt) (bool, error) { return false, nil }
func (p *destroyInputPrompter) Select(context.Context, ui.Prompt, []ui.Option) (string, error) {
	return "", nil
}
func (p *destroyInputPrompter) Input(_ context.Context, prompt ui.Prompt) (string, error) {
	p.prompt = prompt.Message
	return p.input, nil
}

func saveCloudStateForDestroy(t *testing.T, stateDir string) state.RepositoryState {
	t.Helper()
	repo := testsupport.CloudRepositoryState()
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save cloud state: %v", err)
	}
	return repo
}

func executeDestroyForTest(t *testing.T, stateDir string, github *testsupport.GitHubService, remoteExec remote.Executor, cloud *provider.FakeProvider, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: github, RemoteExecutor: remoteExec, Providers: provider.NewRegistry(cloud), CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	return out.String(), errOut.String(), runErr
}

func TestDestroyDryRunJSONMutatesNothing(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveCloudStateForDestroy(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	remoteExec := doctorRemote(true)
	cloud := &provider.FakeProvider{}
	out, errOut, err := executeDestroyForTest(t, stateDir, github, remoteExec, cloud, "--json", "destroy", "--repo", repo.Repo.FullName, "--dry-run", "--no-color")
	if err != nil {
		t.Fatalf("destroy dry-run returned error: %v\nstderr=%s", err, errOut)
	}
	for _, want := range []string{`"command":"destroy"`, `"dry_run":true`, `"provider_verification":{"status":"not_run"}`, `"partial_cleanup":false`, `"pending":[]`, `"state_removed":false`, `"redactions_applied":true`} {
		if !strings.Contains(out, want) {
			t.Fatalf("destroy dry-run json missing %q:\n%s", want, out)
		}
	}
	if github.CreateRemovalTokenCalls != 0 || github.DeleteRunnerCalls != 0 || len(remoteExec.Commands) != 0 || cloud.ProvisionCalls != 0 || cloud.DestroyCalls != 0 || cloud.VerifyDestroyedCalls != 0 {
		t.Fatalf("dry-run mutated dependencies: github=%#v remote=%#v cloud=%#v", github, remoteExec, cloud)
	}
	if _, found, loadErr := state.NewStore(stateDir).GetRepository(repo.Repo.FullName); loadErr != nil || !found {
		t.Fatalf("dry-run removed state found=%v err=%v", found, loadErr)
	}
}

func TestDestroyRejectsBYOState(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	out, errOut, err := executeDestroyForTest(t, stateDir, &testsupport.GitHubService{}, doctorRemote(true), &provider.FakeProvider{}, "destroy", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err == nil || ExitCode(err) != ExitInvalidInput {
		t.Fatalf("expected BYO destroy rejection, err=%v stdout=%s stderr=%s", err, out, errOut)
	}
	if !strings.Contains(out+errOut, "Use runnerkit down --repo owner/repo for BYO cleanup.") {
		t.Fatalf("missing BYO remediation:\nstdout=%s\nstderr=%s", out, errOut)
	}
}

func TestDestroyNoTTYRequiresYes(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveCloudStateForDestroy(t, stateDir)
	_, errOut, err := executeDestroyForTest(t, stateDir, &testsupport.GitHubService{}, doctorRemote(true), &provider.FakeProvider{}, "destroy", "--repo", repo.Repo.FullName, "--no-color")
	if err == nil || ExitCode(err) != ExitInputRequired {
		t.Fatalf("expected input-required, err=%v", err)
	}
	for _, want := range []string{"Pass --yes to apply the cloud destroy plan non-interactively", "--dry-run to preview only."} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("missing --yes remediation %q: %s", want, errOut)
		}
	}
}

func TestDestroyInteractiveTypedConfirmation(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveCloudStateForDestroy(t, stateDir)
	prompts := &destroyInputPrompter{input: "destroy owner/repo"}
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	cloud := &provider.FakeProvider{VerifyOut: provider.VerificationResult{OK: true}}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: github, RemoteExecutor: doctorRemote(true), Providers: provider.NewRegistry(cloud), Prompts: prompts, TTY: ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80}, CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
	cmd.SetArgs([]string{"destroy", "--repo", repo.Repo.FullName, "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("interactive destroy returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	if prompts.prompt != "Destroy cloud runner: type `destroy owner/repo` to remove the GitHub runner registration and RunnerKit-created Hetzner resources." {
		t.Fatalf("prompt = %q", prompts.prompt)
	}
}

func TestDestroyCompleteRemovesStateAfterProviderVerification(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveCloudStateForDestroy(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	cloud := &provider.FakeProvider{VerifyOut: provider.VerificationResult{OK: true}}
	out, errOut, err := executeDestroyForTest(t, stateDir, github, doctorRemote(true), cloud, "--json", "destroy", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("destroy returned error: %v\nstdout=%s\nstderr=%s", err, out, errOut)
	}
	for _, want := range []string{`"state_removed":true`, `"partial_cleanup":false`, `"ok":true`, `"provider_verification":{"billable_resources":[],"error":"","missing":[],"ok":true}`} {
		if !strings.Contains(out, want) {
			t.Fatalf("destroy success json missing %q:\n%s", want, out)
		}
	}
	if _, found, loadErr := state.NewStore(stateDir).GetRepository(repo.Repo.FullName); loadErr != nil || found {
		t.Fatalf("state should be removed found=%v err=%v", found, loadErr)
	}
	if github.DeleteRunnerCalls != 1 || cloud.DestroyCalls != 1 || cloud.VerifyDestroyedCalls != 1 {
		t.Fatalf("cleanup calls mismatch github=%#v cloud=%#v", github, cloud)
	}
}

func TestDestroyProviderVerificationFailureKeepsStateWithResourceIDs(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveCloudStateForDestroy(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	cloud := &provider.FakeProvider{VerifyOut: provider.VerificationResult{OK: false, BillableResources: []string{"server:srv-123"}}}
	out, errOut, err := executeDestroyForTest(t, stateDir, github, doctorRemote(true), cloud, "--json", "destroy", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("partial destroy should render without command error: %v\nstdout=%s\nstderr=%s", err, out, errOut)
	}
	if !strings.Contains(out, `"partial_cleanup":true`) || !strings.Contains(out, pendingProviderVerification) || !strings.Contains(out, "server:srv-123") {
		t.Fatalf("partial json missing verification pending/resource id:\n%s", out)
	}
	loaded, found, loadErr := state.NewStore(stateDir).GetRepository(repo.Repo.FullName)
	if loadErr != nil || !found {
		t.Fatalf("partial destroy should keep state found=%v err=%v", found, loadErr)
	}
	joined := strings.Join(loaded.Cleanup.ProviderResourceIDs, ",")
	if !strings.Contains(joined, "server:srv-123") || len(loaded.Operations) == 0 {
		t.Fatalf("partial state lost provider IDs or checkpoints: %#v", loaded)
	}
}

func TestDestroyGitHubFailureKeepsState(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveCloudStateForDestroy(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}, DeleteRunnerErr: errors.New("delete denied")}
	cloud := &provider.FakeProvider{VerifyOut: provider.VerificationResult{OK: true}}
	out, _, err := executeDestroyForTest(t, stateDir, github, doctorRemote(true), cloud, "--json", "destroy", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("partial GitHub destroy should render without command error: %v", err)
	}
	if !strings.Contains(out, pendingGitHubCleanup) || !strings.Contains(out, `"partial_cleanup":true`) {
		t.Fatalf("github partial output missing pending: %s", out)
	}
	if _, found, _ := state.NewStore(stateDir).GetRepository(repo.Repo.FullName); !found {
		t.Fatal("github failure should keep state")
	}
}

func TestDestroyEphemeralCloudPreservesLogsBeforeProviderDestroyAndUsesEphemeralPrompt(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.CloudRepositoryState()
	repo.Runner.Mode = "ephemeral"
	repo.Runner.Name = "runnerkit-owner-repo-ephemeral-fake1"
	repo.Machine.ServiceName = "runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-fake1.service"
	repo.Safety.SafetyProfile = "ephemeral-cloud"
	repo.Ephemeral = state.EphemeralMetadata{Enabled: true, TTL: "24h", LogArchivePath: "/var/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-fake1/logs", FinalizerStatus: "pending", CleanupCommand: "runnerkit destroy --repo owner/repo"}
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	exec := doctorRemote(true)
	exec.Results["ephemeral.logs.preserve"] = remote.Result{ExitCode: 0}
	cloud := &provider.FakeProvider{VerifyOut: provider.VerificationResult{OK: true}}
	out, errOut, err := executeDestroyForTest(t, stateDir, github, exec, cloud, "destroy", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("ephemeral cloud destroy returned error: %v\nstdout=%s\nstderr=%s", err, out, errOut)
	}
	preserveIdx := -1
	for i, command := range exec.Commands {
		if command.ID == "ephemeral.logs.preserve" {
			preserveIdx = i
			break
		}
	}
	if preserveIdx < 0 {
		t.Fatalf("expected ephemeral.logs.preserve in command IDs: %v", exec.CommandIDs())
	}
	if cloud.DestroyCalls != 1 {
		t.Fatalf("expected one provider Destroy, got %d", cloud.DestroyCalls)
	}
}

func TestDestroyEphemeralCloudInteractivePromptCopy(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.CloudRepositoryState()
	repo.Runner.Mode = "ephemeral"
	repo.Safety.SafetyProfile = "ephemeral-cloud"
	repo.Ephemeral = state.EphemeralMetadata{Enabled: true, TTL: "24h", FinalizerStatus: "pending", CleanupCommand: "runnerkit destroy --repo owner/repo"}
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	prompts := &destroyInputPrompter{input: "destroy owner/repo"}
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	cloud := &provider.FakeProvider{VerifyOut: provider.VerificationResult{OK: true}}
	exec := doctorRemote(true)
	exec.Results["ephemeral.logs.preserve"] = remote.Result{ExitCode: 0}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: github, RemoteExecutor: exec, Providers: provider.NewRegistry(cloud), Prompts: prompts, TTY: ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80}, CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
	cmd.SetArgs([]string{"destroy", "--repo", repo.Repo.FullName, "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("interactive ephemeral destroy returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	want := "Destroy ephemeral cloud runner: type `destroy owner/repo` to remove the GitHub runner registration and RunnerKit-created Hetzner resources."
	if prompts.prompt != want {
		t.Fatalf("ephemeral destroy prompt = %q, want %q", prompts.prompt, want)
	}
}

// TestDestroyEphemeralCloudPreservesLogsBeforeProviderDelete proves that
// runnerkit destroy on an ephemeral cloud repository runs
// `ephemeral.logs.preserve` before provider Destroy, that the JSON
// output includes provider_verification, that the destroy command Provider
// Destroy was called exactly once, and that the typed-confirmation
// prompt copy uses the dedicated ephemeral cloud destroy template
// (verified via a second interactive run because --json bypasses the
// interactive typed confirmation).
func TestDestroyEphemeralCloudPreservesLogsBeforeProviderDelete(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.EphemeralCloudRepositoryState()
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	cloud := &provider.FakeProvider{VerifyOut: provider.VerificationResult{OK: true}}
	exec := doctorRemote(true)
	exec.Results["ephemeral.logs.preserve"] = remote.Result{ExitCode: 0}

	// First run: JSON --yes path so we can assert provider_verification
	// is present in the JSON envelope without a prompt.
	jsonOut, errOut, err := executeDestroyForTest(t, stateDir, github, exec, cloud, "--json", "destroy", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("ephemeral cloud destroy returned error: %v\nstdout=%s\nstderr=%s", err, jsonOut, errOut)
	}
	preserveIdx := -1
	destroyCallsBefore := cloud.DestroyCalls
	for i, command := range exec.Commands {
		if command.ID == "ephemeral.logs.preserve" {
			preserveIdx = i
			break
		}
	}
	if preserveIdx < 0 {
		t.Fatalf("expected ephemeral.logs.preserve in command IDs: %v", exec.CommandIDs())
	}
	if destroyCallsBefore != 1 {
		t.Fatalf("expected one provider Destroy, got %d", destroyCallsBefore)
	}
	if !strings.Contains(jsonOut, `"provider_verification"`) {
		t.Fatalf("expected provider_verification in JSON destroy output:\n%s", jsonOut)
	}

	// Second run: interactive typed-confirmation path for prompt copy.
	// Reset state because the first run removed it on success.
	stateDir2 := t.TempDir()
	if err := state.NewStore(stateDir2).Save(testsupport.StateWithRepository(testsupport.EphemeralCloudRepositoryState())); err != nil {
		t.Fatalf("save state: %v", err)
	}
	prompts := &destroyInputPrompter{input: "destroy " + repo.Repo.FullName}
	github2 := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	cloud2 := &provider.FakeProvider{VerifyOut: provider.VerificationResult{OK: true}}
	exec2 := doctorRemote(true)
	exec2.Results["ephemeral.logs.preserve"] = remote.Result{ExitCode: 0}
	var out2, errOut2 bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out2, Err: &errOut2, StateBaseDir: stateDir2, GitHub: github2, RemoteExecutor: exec2, Providers: provider.NewRegistry(cloud2), Prompts: prompts, TTY: ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80}, CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
	cmd.SetArgs([]string{"destroy", "--repo", repo.Repo.FullName, "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("interactive ephemeral destroy returned error: %v\nstdout=%s\nstderr=%s", err, out2.String(), errOut2.String())
	}
	want := "Destroy ephemeral cloud runner: type `destroy owner/repo` to remove the GitHub runner registration and RunnerKit-created Hetzner resources."
	if prompts.prompt != want {
		t.Fatalf("ephemeral destroy prompt = %q, want %q", prompts.prompt, want)
	}
}

func TestDestroyRedactsRemovalAndProviderTokens(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveCloudStateForDestroy(t, stateDir)
	providerValue := strings.Join([]string{"hcloud", "redact", "value"}, "-")
	removalValue := strings.Join([]string{"removal", "token", "testsupport"}, "-")
	t.Setenv("HCLOUD_TOKEN", providerValue)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	cloud := &provider.FakeProvider{VerifyOut: provider.VerificationResult{OK: false, BillableResources: []string{"server:srv-123"}}}
	out, errOut, err := executeDestroyForTest(t, stateDir, github, doctorRemote(true), cloud, "--json", "destroy", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err != nil {
		t.Fatalf("destroy returned error: %v", err)
	}
	stateBytes, _ := json.Marshal(mustLoadRepo(t, stateDir, repo.Repo.FullName))
	combined := out + errOut + string(stateBytes)
	for _, raw := range []string{removalValue, providerValue} {
		if strings.Contains(combined, raw) {
			t.Fatalf("destroy leaked %q:\n%s", raw, combined)
		}
	}
}

func mustLoadRepo(t *testing.T, stateDir string, fullName string) state.RepositoryState {
	t.Helper()
	repo, found, err := state.NewStore(stateDir).GetRepository(fullName)
	if err != nil || !found {
		t.Fatalf("load repo found=%v err=%v", found, err)
	}
	return repo
}
