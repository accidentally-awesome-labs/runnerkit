package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/labels"
	"github.com/salar/runnerkit/internal/provider"
	"github.com/salar/runnerkit/internal/state"
)

// fakeEphemeralGitHubService extends fakePermittedGitHubService so the
// ListRunners second-call response uses the deterministic ephemeral
// runner name produced when shortEphemeralIDFn is stubbed to "fake1".
type fakeEphemeralGitHubService struct {
	*fakePermittedGitHubService
	stub []gh.Runner
}

func (s *fakeEphemeralGitHubService) ListRunners(_ context.Context, repo gh.Repo) ([]gh.Runner, error) {
	s.listCalls++
	if s.runners != nil {
		return s.runners, nil
	}
	if s.listCalls == 1 {
		return nil, nil
	}
	if s.stub != nil {
		return s.stub, nil
	}
	return []gh.Runner{{ID: 222, Name: labels.EphemeralRunnerName(repo, "fake1"), OS: "linux", Status: "online", Labels: []string{"self-hosted", "runnerkit", labels.RepoScopedLabel(repo), "linux", "x64", "ephemeral"}}}, nil
}

func newFakeEphemeralGitHubService() *fakeEphemeralGitHubService {
	return &fakeEphemeralGitHubService{fakePermittedGitHubService: newFakePermittedGitHubService()}
}

// withDeterministicEphemeralID stubs the package-level
// shortEphemeralIDFn for the duration of t so ephemeral runner names
// match the fake's expected value "fake1".
func withDeterministicEphemeralID(t *testing.T) {
	t.Helper()
	prev := shortEphemeralIDFn
	shortEphemeralIDFn = func() string { return "fake1" }
	t.Cleanup(func() { shortEphemeralIDFn = prev })
}

func TestUpEphemeralBYOEndToEndApplyEphemeralAndPersistsState(t *testing.T) {
	withDeterministicEphemeralID(t)
	stateDir := t.TempDir()
	service := newFakeEphemeralGitHubService()
	remoteExec := newFakeRemoteExecutor()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
		Clock:          func() time.Time { return time.Date(2026, 5, 2, 18, 30, 0, 0, time.UTC) },
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "alice@example.com", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral BYO end-to-end returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	stdout := out.String()
	flat := strings.Join(strings.Fields(stdout), " ")
	for _, want := range []string{
		"Ephemeral runner ready",
		"GitHub will assign at most one job to this runner, then automatically deregister it.",
		"TTL safeguard: RunnerKit finalizes the ephemeral runner after 24h if no job completes.",
		"RunnerKit preserves best-effort runner _diag and systemd journal logs before cleanup.",
		"Cleanup after the job: runnerkit down --repo owner/name",
		"Ephemeral mode is not a fleet manager.",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ephemeral BYO completion missing %q (flattened):\n%s", want, stdout)
		}
	}
	ids := []string{}
	for _, command := range remoteExec.runs {
		ids = append(ids, command.ID)
	}
	for _, want := range []string{"configure_ephemeral_runner", "install_ephemeral_finalizer", "install_ephemeral_service", "install_ephemeral_ttl_timer", "verify_ephemeral_service"} {
		found := false
		for _, id := range ids {
			if id == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected ApplyEphemeral command %q in %v", want, ids)
		}
	}
	stateBytes, err := os.ReadFile(state.NewStore(stateDir).Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	for _, want := range []string{
		`"mode": "ephemeral"`,
		`"safety_profile": "ephemeral-byo"`,
		`"log_archive_path": "/var/lib/runnerkit/ephemeral/`,
		`"cleanup_command": "runnerkit down --repo owner/name"`,
		`"enabled": true`,
		`"ttl": "24h"`,
		`"finalizer_status": "pending"`,
	} {
		if !strings.Contains(string(stateBytes), want) {
			t.Fatalf("ephemeral state missing %q:\n%s", want, stateBytes)
		}
	}
}

func TestUpEphemeralBYOReadinessFailureBlocksRegistrationToken(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	remoteExec.probe.Arch = "sparc"
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: service, RemoteExecutor: remoteExec, Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "alice@example.com", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected preflight failure, err=%v", err)
	}
	if service.tokenCalls != 0 {
		t.Fatalf("ephemeral preflight failure must not mint registration token; got %d", service.tokenCalls)
	}
}

func TestUpEphemeralCloudPlanAndStateUseCloudCleanupAndModeEphemeralTag(t *testing.T) {
	withDeterministicEphemeralID(t)
	stateDir := t.TempDir()
	service := newFakeEphemeralGitHubService()
	machine := cloudReadyMachineForTest()
	cloud := &provider.FakeProvider{
		ProvisionOut: provider.ProvisionResult{Machine: machine, CreatedResourceIDs: machine.ResourceIDs, CheckpointRequired: true},
		WaitReadyOut: machine,
	}
	remoteExec := newFakeRemoteExecutor()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Providers:      provider.NewRegistry(cloud),
		Sleep:          noSleep,
		Clock:          func() time.Time { return time.Date(2026, 5, 2, 18, 30, 0, 0, time.UTC) },
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--cloud", "hetzner", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral cloud end-to-end returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	combined := out.String() + errOut.String()
	flat := strings.Join(strings.Fields(combined), " ")
	if !strings.Contains(flat, "Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.") {
		t.Fatalf("ephemeral cloud plan missing exact Hetzner caveat (flattened):\n%s", flat)
	}
	if cloud.ProvisionCalls != 1 || len(cloud.ProvisionInput) != 1 {
		t.Fatalf("expected one provider Provision call, got %d", cloud.ProvisionCalls)
	}
	provisionInput := cloud.ProvisionInput[0]
	if provisionInput.Mode != "ephemeral" {
		t.Fatalf("expected ProvisionInput.Mode=ephemeral, got %q", provisionInput.Mode)
	}
	stateBytes, err := os.ReadFile(state.NewStore(stateDir).Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	for _, want := range []string{
		`"safety_profile": "ephemeral-cloud"`,
		`"cleanup_command": "runnerkit destroy --repo owner/name"`,
		`"mode": "ephemeral"`,
	} {
		if !strings.Contains(string(stateBytes), want) {
			t.Fatalf("ephemeral cloud state missing %q:\n%s", want, stateBytes)
		}
	}
}

func TestUpPersistentDefaultStateStillUsesPersistentMode(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: service, RemoteExecutor: remoteExec, Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--host", "alice@example.com", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("persistent default returned error: %v\nstderr=%s", err, errOut.String())
	}
	stateBytes, _ := os.ReadFile(state.NewStore(stateDir).Path())
	if !strings.Contains(string(stateBytes), `"mode": "persistent"`) || !strings.Contains(string(stateBytes), `"runnerkit-owner-name-local"`) {
		t.Fatalf("persistent default state regressed:\n%s", stateBytes)
	}
	if strings.Contains(string(stateBytes), `"safety_profile": "ephemeral-byo"`) {
		t.Fatalf("persistent default leaked ephemeral safety profile:\n%s", stateBytes)
	}
}

func TestUpEphemeralCloudReadinessFailureBlocksRegistrationToken(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	machine := cloudReadyMachineForTest()
	cloud := &provider.FakeProvider{
		ProvisionOut: provider.ProvisionResult{Machine: machine, CreatedResourceIDs: machine.ResourceIDs, CheckpointRequired: true},
		WaitReadyErr: errors.New("provider still provisioning"),
	}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		GitHub:         service,
		RemoteExecutor: newFakeRemoteExecutor(),
		Providers:      provider.NewRegistry(cloud),
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--cloud", "hetzner", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected ephemeral cloud readiness failure, err=%v", err)
	}
	if service.tokenCalls != 0 {
		t.Fatalf("ephemeral cloud readiness failure must not mint registration token; got %d", service.tokenCalls)
	}
}

func TestUpEphemeralBYOJSONCompletionIncludesEphemeralKeys(t *testing.T) {
	withDeterministicEphemeralID(t)
	stateDir := t.TempDir()
	service := newFakeEphemeralGitHubService()
	remoteExec := newFakeRemoteExecutor()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"--json", "up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "alice@example.com", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral BYO json completion returned error: %v\nstderr=%s", err, errOut.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	for k, want := range map[string]any{
		"mode":               "ephemeral",
		"safety_profile":     "ephemeral-byo",
		"ephemeral":          true,
		"ttl":                "24h0m0s",
		"redactions_applied": true,
	} {
		if payload[k] != want {
			t.Fatalf("payload[%q] = %#v, want %#v", k, payload[k], want)
		}
	}
	if _, ok := payload["log_archive"].(string); !ok {
		t.Fatalf("payload[log_archive] missing/non-string: %#v", payload["log_archive"])
	}
	if cleanup, _ := payload["cleanup_command"].(string); cleanup != "runnerkit down --repo owner/name" {
		t.Fatalf("payload[cleanup_command] = %#v", payload["cleanup_command"])
	}
	if snippet, _ := payload["workflow_snippet"].(string); !strings.Contains(snippet, "ephemeral") {
		t.Fatalf("payload[workflow_snippet] missing ephemeral: %#v", payload["workflow_snippet"])
	}
}
