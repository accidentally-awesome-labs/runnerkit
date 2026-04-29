package testsupport

import (
	"time"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/labels"
	"github.com/salar/runnerkit/internal/state"
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
