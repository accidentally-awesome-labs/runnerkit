package testsupport

import (
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/labels"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

const (
	TestRepoFullName   = "owner/repo"
	TestRunnerName     = "runnerkit-owner-repo-local"
	TestHostRef        = "alice@example.com:22"
	TestServiceName    = "actions.runner.runnerkit-owner-repo-local.service"
	TestInstallPath    = "/opt/actions-runner/runnerkit-owner-repo-local"
	TestWorkDir        = "/var/lib/runnerkit/work/runnerkit-owner-repo-local"
	TestGitHubRunnerID = int64(123)
)

var TestLabels = []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "persistent"}

func HealthyRepositoryState() state.RepositoryState {
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	return state.RepositoryState{
		Repo:             gh.Repo{Host: "github.com", Owner: "owner", Name: "repo", FullName: TestRepoFullName, Private: true},
		Auth:             state.AuthReference{Source: "gh", Reference: "gh"},
		Runner:           state.RunnerIdentity{Name: TestRunnerName, Labels: append([]string(nil), TestLabels...), WorkflowSnippet: labels.WorkflowSnippet(TestLabels), Mode: "persistent", OS: "linux", Arch: "x64"},
		Machine:          state.MachineRef{Kind: "byo-ssh", HostRef: TestHostRef, User: "alice", Port: 22, HostKeyAlgorithm: "ssh-ed25519", HostKeyFingerprint: "SHA256:fakehostfingerprint", InstallPath: TestInstallPath, WorkDir: TestWorkDir, ServiceName: TestServiceName},
		Provider:         state.ProviderRef{Kind: "byo", IDs: map[string]string{}},
		Cleanup:          state.CleanupMetadata{GitHubRunnerID: TestGitHubRunnerID, ManagedPaths: []string{TestInstallPath, "/var/lib/runnerkit"}, ProviderResourceIDs: []string{}},
		Safety:           state.SafetyMetadata{Code: "ok", Allowed: true},
		RunnerKitVersion: "test-version",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func BusyRepositoryState() state.RepositoryState            { return HealthyRepositoryState() }
func GitHubOfflineRepositoryState() state.RepositoryState   { return HealthyRepositoryState() }
func LabelDriftRepositoryState() state.RepositoryState      { return HealthyRepositoryState() }
func SSHUnreachableRepositoryState() state.RepositoryState  { return HealthyRepositoryState() }
func HostKeyMismatchRepositoryState() state.RepositoryState { return HealthyRepositoryState() }

func MissingGitHubRunnerRepositoryState() state.RepositoryState {
	repo := HealthyRepositoryState()
	repo.Cleanup.GitHubRunnerID = 0
	return repo
}

func MissingServiceRepositoryState() state.RepositoryState {
	repo := HealthyRepositoryState()
	repo.Machine.ServiceName = ""
	return repo
}

func PartialCleanupRepositoryState() state.RepositoryState {
	repo := HealthyRepositoryState()
	repo.Cleanup.Notes = []string{"remote_cleanup_pending"}
	repo.Operations = []state.OperationCheckpoint{{Command: "down", Artifact: "remote", Status: "pending", Message: "SSH unreachable during cleanup", UpdatedAt: repo.UpdatedAt}}
	return repo
}

func StateWithRepository(repo state.RepositoryState) state.State {
	return state.State{SchemaVersion: state.SchemaVersion, Repositories: []state.RepositoryState{repo}}
}

func HealthyRunner() gh.Runner {
	return gh.Runner{ID: TestGitHubRunnerID, Name: TestRunnerName, OS: "linux", Status: "online", Busy: false, Labels: append([]string(nil), TestLabels...)}
}

// EphemeralBYORepositoryState returns a repository state fixture for an
// ephemeral BYO runner. The runner name uses the deterministic short id
// "20260501t183000" so tests can assert exact ephemeral artifact paths;
// labels include the `ephemeral` mode label; and CleanupCommand points
// at `runnerkit down --repo owner/repo` so tests can prove BYO ephemeral
// cleanup uses `down` and not `destroy`.
func EphemeralBYORepositoryState() state.RepositoryState {
	repo := HealthyRepositoryState()
	runnerName := "runnerkit-owner-repo-ephemeral-20260501t183000"
	repo.Runner.Mode = "ephemeral"
	repo.Runner.Name = runnerName
	repo.Runner.Labels = []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "ephemeral"}
	repo.Runner.WorkflowSnippet = labels.WorkflowSnippet(repo.Runner.Labels)
	repo.Machine.InstallPath = "/opt/actions-runner/" + runnerName
	repo.Machine.WorkDir = "/var/lib/runnerkit/work/" + runnerName
	repo.Machine.ServiceName = "runnerkit-ephemeral." + runnerName + ".service"
	repo.Safety.SafetyProfile = "ephemeral-byo"
	repo.Ephemeral = state.EphemeralMetadata{
		Enabled:         true,
		TTL:             "24h",
		LogArchivePath:  "/var/lib/runnerkit/ephemeral/" + runnerName + "/logs",
		FinalizerStatus: "pending",
		CleanupCommand:  "runnerkit down --repo owner/repo",
	}
	return repo
}

// EphemeralCloudRepositoryState returns a repository state fixture for
// an ephemeral cloud runner provisioned on Hetzner. The runner name uses
// the deterministic short id "20260501t183000"; labels include
// `ephemeral`; provider tags include `mode=ephemeral`; and CleanupCommand
// points at `runnerkit destroy --repo owner/repo` so cleanup tests prove
// cloud ephemeral cleanup goes through `destroy`.
func EphemeralCloudRepositoryState() state.RepositoryState {
	repo := CloudRepositoryState()
	runnerName := "runnerkit-owner-repo-ephemeral-20260501t183000"
	repo.Runner.Mode = "ephemeral"
	repo.Runner.Name = runnerName
	repo.Runner.Labels = []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "ephemeral"}
	repo.Runner.WorkflowSnippet = labels.WorkflowSnippet(repo.Runner.Labels)
	repo.Machine.InstallPath = "/opt/actions-runner/" + runnerName
	repo.Machine.WorkDir = "/var/lib/runnerkit/work/" + runnerName
	repo.Machine.ServiceName = "runnerkit-ephemeral." + runnerName + ".service"
	repo.Safety.SafetyProfile = "ephemeral-cloud"
	repo.Ephemeral = state.EphemeralMetadata{
		Enabled:         true,
		TTL:             "24h",
		LogArchivePath:  "/var/lib/runnerkit/ephemeral/" + runnerName + "/logs",
		FinalizerStatus: "pending",
		CleanupCommand:  "runnerkit destroy --repo owner/repo",
	}
	return repo
}

func CloudRepositoryState() state.RepositoryState {
	repo := HealthyRepositoryState()
	repo.Machine.Kind = "cloud-ssh"
	repo.Machine.HostRef = "runnerkit-admin@203.0.113.10:22"
	repo.Machine.User = "runnerkit-admin"
	repo.Provider = state.ProviderRef{
		Kind:        "hetzner",
		Name:        "hetzner",
		Region:      "fsn1",
		Profile:     "cpx22",
		IDs:         map[string]string{"server": "srv-123", "ssh_key": "key-123", "firewall": "fw-123", "primary_ipv4": "ip-123"},
		ResourceIDs: map[string]string{"server": "srv-123", "ssh_key": "key-123", "firewall": "fw-123", "primary_ipv4": "ip-123"},
		Cloud: state.CloudInventory{
			Provider:      "hetzner",
			ServerID:      "srv-123",
			ServerStatus:  "running",
			Region:        "fsn1",
			ServerType:    "cpx22",
			Image:         "ubuntu-24.04",
			PublicIPv4:    "203.0.113.10",
			SSHKeyID:      "key-123",
			FirewallID:    "fw-123",
			PrimaryIPv4ID: "ip-123",
		},
	}
	repo.Cleanup.ProviderResourceIDs = []string{"server:srv-123", "ssh_key:key-123", "firewall:fw-123", "primary_ipv4:ip-123"}
	return repo
}
