package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/remote"
	"github.com/salar/runnerkit/internal/state"
)

func TestUpNonInteractiveRequiresHost(t *testing.T) {
	_, errOut, err := executeForTest(t, "up", "--repo", "owner/name", "--non-interactive", "--yes", "--no-color")
	if err == nil {
		t.Fatal("expected missing host error")
	}
	if got := ExitCode(err); got != ExitInputRequired {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInputRequired)
	}
	if !strings.Contains(errOut, "Pass --host user@host for BYO setup.") {
		t.Fatalf("missing host remediation: %q", errOut)
	}
}

func TestUpHostKeyMismatchFailsBeforePreflight(t *testing.T) {
	stateDir := t.TempDir()
	store := state.NewStore(stateDir)
	now := time.Date(2026, 4, 29, 1, 0, 0, 0, time.UTC)
	if err := store.SaveRepository(state.RepositoryState{Repo: github.Repo{Owner: "owner", Name: "repo", FullName: "owner/repo", Private: true}, Auth: state.AuthReference{Source: "gh", Reference: "gh"}, Runner: state.RunnerIdentity{Name: "runnerkit-owner-repo-local"}, Machine: state.MachineRef{Kind: "byo-ssh", HostKeyFingerprint: "SHA256:old", HostKeyAcceptedAt: &now}, Provider: state.ProviderRef{Kind: "byo"}, Cleanup: state.CleanupMetadata{ManagedPaths: []string{}, ProviderResourceIDs: []string{}}, Safety: state.SafetyMetadata{Code: "ok", Allowed: true}}, false); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	remoteExec := newFakeRemoteExecutor()
	remoteExec.probe.HostKey = remote.HostKey{Algorithm: "ssh-ed25519", Fingerprint: "SHA256:new"}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: newFakePermittedGitHubService(), RemoteExecutor: remoteExec, Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--replace", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected safety gate mismatch, err=%v stdout=%s stderr=%s", err, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "SSH host key fingerprint changed") || len(remoteExec.runs) != 0 {
		t.Fatalf("mismatch did not fail before preflight; runs=%d stderr=%s", len(remoteExec.runs), errOut.String())
	}
}

func TestUpFailedPreflightDoesNotCreateRegistrationToken(t *testing.T) {
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	remoteExec.probe.Arch = "sparc"
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: service, RemoteExecutor: remoteExec, Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected preflight failure, err=%v stdout=%s stderr=%s", err, out.String(), errOut.String())
	}
	if service.tokenCalls != 0 {
		t.Fatalf("CreateRegistrationToken calls = %d, want 0", service.tokenCalls)
	}
}

func TestUpDuplicateRunnerBlocksBeforeRegistrationToken(t *testing.T) {
	service := newFakePermittedGitHubService()
	service.runners = []github.Runner{{ID: 99, Name: "runnerkit-owner-repo-local", Status: "online"}}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: service, RemoteExecutor: newFakeRemoteExecutor(), Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected duplicate runner conflict, err=%v", err)
	}
	if !strings.Contains(errOut.String(), "already exists") || service.tokenCalls != 0 {
		t.Fatalf("duplicate did not block before token; tokenCalls=%d stderr=%s", service.tokenCalls, errOut.String())
	}
}

func TestUpDryRunDoesNotApplyBootstrap(t *testing.T) {
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: service, RemoteExecutor: remoteExec, Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--host", "alice@example.com", "--dry-run", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("dry-run returned error: %v\n%s", err, errOut.String())
	}
	for _, command := range remoteExec.runs {
		if command.ID == "configure_runner" || command.ID == "install_service" || command.ID == "verify_service" {
			t.Fatalf("dry-run applied bootstrap command: %#v", command)
		}
	}
	if service.tokenCalls != 0 {
		t.Fatalf("dry-run created registration token: %d", service.tokenCalls)
	}
}

func TestUpCompletionHumanJSONAndWorkflowFileUnchanged(t *testing.T) {
	stateDir := t.TempDir()
	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldDir) }()
	workflowPath := filepath.Join(workDir, ".github", "workflows", "existing.yml")
	if err := os.MkdirAll(filepath.Dir(workflowPath), 0755); err != nil {
		t.Fatal(err)
	}
	originalWorkflow := []byte("name: existing\n")
	if err := os.WriteFile(workflowPath, originalWorkflow, 0644); err != nil {
		t.Fatal(err)
	}
	out, errOut, err := executeWithStateDir(t, stateDir, "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("up returned error: %v\nstderr=%s", err, errOut)
	}
	for _, want := range []string{"BYO runner ready", "runnerkit-owner-repo-local", "alice@example.com:22", "runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]", "Add the runs-on snippet above"} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q:\n%s", want, out)
		}
	}
	workflowBytes, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(workflowBytes, originalWorkflow) {
		t.Fatalf("workflow file changed: %q", workflowBytes)
	}

	jsonStateDir := t.TempDir()
	jsonOut, jsonErr, err := executeWithStateDir(t, jsonStateDir, "--json", "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("json up returned error: %v\nstderr=%s", err, jsonErr)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("json output invalid: %v\n%s", err, jsonOut)
	}
	if payload["runner_installed"] != true || payload["redactions_applied"] != true || payload["github_runner_id"].(float64) != 123 {
		t.Fatalf("unexpected json payload: %#v", payload)
	}
}
