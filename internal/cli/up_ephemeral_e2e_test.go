package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"context"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/labels"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

// stagedListRunnersGitHub returns nil on the first ListRunners call so
// the duplicate-name check passes, then returns the staged Runners slice
// on subsequent calls so waitForRunnerOnline succeeds. This mirrors the
// behavior of fakeEphemeralGitHubService for the testsupport.GitHubService
// type used by the end-to-end tests below.
type stagedListRunnersGitHub struct {
	*testsupport.GitHubService
	staged []gh.Runner
	calls  int
}

func (s *stagedListRunnersGitHub) ListRunners(_ context.Context, _ gh.Repo) ([]gh.Runner, error) {
	s.calls++
	if s.calls == 1 {
		return nil, nil
	}
	if s.staged != nil {
		out := make([]gh.Runner, len(s.staged))
		copy(out, s.staged)
		return out, nil
	}
	return nil, nil
}

// ephemeralE2EDeps wires fake GitHub, remote, provider, state, and clock
// dependencies for end-to-end runnerkit up tests against trusted/private
// and public/fork scenarios. The clock is fixed at 2026-05-01T18:30:00Z
// and shortEphemeralIDFn is stubbed to "fake1" so the ephemeral runner
// name and persisted state are deterministic. The fake provider returns
// a ready Hetzner machine, and the fake remote executor advertises host
// key SHA256:fakehostfingerprint plus a passing probe so cloud and BYO
// readiness gates succeed.
func ephemeralE2EDeps(t *testing.T, repo gh.Repo) (Dependencies, *stagedListRunnersGitHub, *testsupport.RemoteExecutor, *provider.FakeProvider, string) {
	t.Helper()
	withDeterministicEphemeralID(t)
	stateDir := t.TempDir()
	github := &stagedListRunnersGitHub{GitHubService: &testsupport.GitHubService{Repo: repo}}
	machine := cloudReadyMachineForTest()
	cloud := &provider.FakeProvider{
		ProvisionOut: provider.ProvisionResult{Machine: machine, CreatedResourceIDs: machine.ResourceIDs, CheckpointRequired: true},
		WaitReadyOut: machine,
	}
	remoteExec := &testsupport.RemoteExecutor{
		ProbeResult:        passingProbe(),
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
	}
	deps := Dependencies{
		Version:        "test-version",
		StateBaseDir:   stateDir,
		GitHub:         github,
		RemoteExecutor: remoteExec,
		Providers:      provider.NewRegistry(cloud),
		Sleep:          noSleep,
		Clock:          func() time.Time { return time.Date(2026, 5, 1, 18, 30, 0, 0, time.UTC) },
	}
	return deps, github, remoteExec, cloud, stateDir
}

// privateRepoForE2E returns a private repository fixture for trusted
// E2E paths.
func privateRepoForE2E() gh.Repo {
	return gh.Repo{Host: "github.com", Owner: "owner", Name: "name", FullName: "owner/name", Private: true}
}

// publicRepoForE2E returns a public repository fixture for risky E2E
// paths. The fake GitHub Repository call returns this struct verbatim
// because it stores it as Repo.
func publicRepoForE2E() gh.Repo {
	return gh.Repo{Host: "github.com", Owner: "owner", Name: "name", FullName: "owner/name", Private: false}
}

func TestPersistentPrivateDefaultStillUsesPersistentProfile(t *testing.T) {
	deps, github, remoteExec, cloud, _ := ephemeralE2EDeps(t, privateRepoForE2E())
	var out, errOut bytes.Buffer
	deps.Out = &out
	deps.Err = &errOut
	cmd := NewRootCommand(deps)
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--host", "user@host", "--dry-run", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("persistent private default returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	for _, want := range []string{
		"Default mode: persistent for trusted private repositories.",
		"Safety profile: persistent-trusted",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-name, linux, x64, persistent]",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("persistent private default missing %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "--ephemeral") || strings.Contains(out.String(), "Safety profile: ephemeral-byo") || strings.Contains(out.String(), "Safety profile: ephemeral-cloud") {
		t.Fatalf("persistent default leaked ephemeral copy:\n%s", out.String())
	}
	if cloud.ProvisionCalls != 0 || cloud.ValidateCalls != 0 || cloud.PlanCalls != 0 {
		t.Fatalf("persistent BYO default must not call provider: provision=%d validate=%d plan=%d", cloud.ProvisionCalls, cloud.ValidateCalls, cloud.PlanCalls)
	}
	if github.CreateRegistrationTokenCalls != 0 {
		t.Fatalf("dry-run must not mint registration token; got %d", github.CreateRegistrationTokenCalls)
	}
	_ = remoteExec
}

func TestPublicPersistentBlocksAndRecommendsEphemeralCloud(t *testing.T) {
	deps, github, remoteExec, cloud, stateDir := ephemeralE2EDeps(t, publicRepoForE2E())
	var out, errOut bytes.Buffer
	deps.Out = &out
	deps.Err = &errOut
	cmd := NewRootCommand(deps)
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "persistent", "--host", "user@host", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected ExitSafetyGate, got err=%v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	combined := out.String() + errOut.String()
	flat := strings.Join(strings.Fields(combined), " ")
	for _, want := range []string{
		"Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.",
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("public persistent block missing %q (flattened):\n%s", want, flat)
		}
	}
	// No GitHub auth, runner-management read, registration token,
	// remote command, provider validate/plan/provision, or state save
	// must occur before the safety gate fires.
	if github.VerifyAuthCalls != 0 || github.VerifyRunnerManagementReadCalls != 0 || github.CreateRegistrationTokenCalls != 0 {
		t.Fatalf("public persistent block leaked github side effects: auth=%d read=%d token=%d", github.VerifyAuthCalls, github.VerifyRunnerManagementReadCalls, github.CreateRegistrationTokenCalls)
	}
	if len(remoteExec.Commands) != 0 {
		t.Fatalf("public persistent block leaked remote commands: %v", remoteExec.CommandIDs())
	}
	if cloud.ValidateCalls != 0 || cloud.PlanCalls != 0 || cloud.ProvisionCalls != 0 {
		t.Fatalf("public persistent block leaked provider side effects: %#v", cloud)
	}
	if _, err := os.Stat(state.NewStore(stateDir).Path()); err == nil {
		t.Fatalf("public persistent block must not write state file at %s", state.NewStore(stateDir).Path())
	}
}

func TestEphemeralCloudRecommendedForPublicRepo(t *testing.T) {
	deps, github, _, cloud, stateDir := ephemeralE2EDeps(t, publicRepoForE2E())
	// Wire the second-pass ListRunners response to the deterministic
	// ephemeral runner so waitForRunnerOnline succeeds without a live
	// GitHub call. The first call returns nil so the duplicate-name
	// check passes.
	github.staged = []gh.Runner{{
		ID:     222,
		Name:   labels.EphemeralRunnerName(publicRepoForE2E(), "fake1"),
		OS:     "linux",
		Status: "online",
		Labels: []string{"self-hosted", "runnerkit", labels.RepoScopedLabel(publicRepoForE2E()), "linux", "x64", "ephemeral"},
	}}
	var out, errOut bytes.Buffer
	deps.Out = &out
	deps.Err = &errOut
	cmd := NewRootCommand(deps)
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--cloud", "hetzner", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral cloud public end-to-end returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	combined := out.String() + errOut.String()
	flat := strings.Join(strings.Fields(combined), " ")
	for _, want := range []string{
		"Use ephemeral cloud runner",
		"Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.",
		"Ephemeral runner ready",
		"GitHub will assign at most one job to this runner, then automatically deregister it.",
		"Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online.",
		"Cleanup after the job: runnerkit destroy --repo owner/name",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ephemeral cloud public completion missing %q (flattened):\n%s", want, flat)
		}
	}
	stateBytes, err := os.ReadFile(state.NewStore(stateDir).Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	for _, want := range []string{
		`"mode": "ephemeral"`,
		`"safety_profile": "ephemeral-cloud"`,
		`"ttl": "24h"`,
	} {
		if !strings.Contains(string(stateBytes), want) {
			t.Fatalf("ephemeral cloud state missing %q:\n%s", want, stateBytes)
		}
	}
	if github.CreateRegistrationTokenCalls != 1 {
		t.Fatalf("expected exactly one registration token, got %d", github.CreateRegistrationTokenCalls)
	}
	if cloud.ProvisionCalls != 1 {
		t.Fatalf("expected exactly one provider Provision call, got %d", cloud.ProvisionCalls)
	}
	if len(cloud.ProvisionInput) == 0 || cloud.ProvisionInput[0].Mode != "ephemeral" {
		t.Fatalf("expected ProvisionInput.Mode=ephemeral, got %#v", cloud.ProvisionInput)
	}
	// Sanity check provisioning tag set includes mode=ephemeral via
	// HetznerOwnershipTags (already exercised by provider tests; here we
	// only confirm Mode plumbing).
}

func TestEphemeralBYOPublicRequiresAcknowledgement(t *testing.T) {
	deps, github, remoteExec, _, _ := ephemeralE2EDeps(t, publicRepoForE2E())
	var out, errOut bytes.Buffer
	deps.Out = &out
	deps.Err = &errOut
	cmd := NewRootCommand(deps)
	// No --allow-ephemeral-byo-risk: the safety gate must block.
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "user@host", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected ExitSafetyGate for ephemeral BYO public, got err=%v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	combined := out.String() + errOut.String()
	flat := strings.Join(strings.Fields(combined), " ")
	if !strings.Contains(flat, "BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine.") {
		t.Fatalf("missing BYO clean-VM caveat in error (flattened):\n%s", flat)
	}
	if github.CreateRegistrationTokenCalls != 0 || len(remoteExec.Commands) != 0 {
		t.Fatalf("ephemeral BYO public gate leaked side effects: token=%d remote=%v", github.CreateRegistrationTokenCalls, remoteExec.CommandIDs())
	}
}

func TestEphemeralBYOTrustedPrivateUsesDownCleanup(t *testing.T) {
	deps, github, _, cloud, stateDir := ephemeralE2EDeps(t, privateRepoForE2E())
	github.staged = []gh.Runner{{
		ID:     333,
		Name:   labels.EphemeralRunnerName(privateRepoForE2E(), "fake1"),
		OS:     "linux",
		Status: "online",
		Labels: []string{"self-hosted", "runnerkit", labels.RepoScopedLabel(privateRepoForE2E()), "linux", "x64", "ephemeral"},
	}}
	var out, errOut bytes.Buffer
	deps.Out = &out
	deps.Err = &errOut
	cmd := NewRootCommand(deps)
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "user@host", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral BYO trusted private returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	stateBytes, err := os.ReadFile(state.NewStore(stateDir).Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	for _, want := range []string{
		`"mode": "ephemeral"`,
		`"cleanup_command": "runnerkit down --repo owner/name"`,
	} {
		if !strings.Contains(string(stateBytes), want) {
			t.Fatalf("ephemeral BYO state missing %q:\n%s", want, stateBytes)
		}
	}
	if cloud.ProvisionCalls != 0 || cloud.PlanCalls != 0 || cloud.ValidateCalls != 0 {
		t.Fatalf("BYO ephemeral must not call provider: %#v", cloud)
	}
}
