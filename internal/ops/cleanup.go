package ops

import (
	"path/filepath"
	"strings"

	"github.com/salar/runnerkit/internal/state"
)

type CleanupArtifact string

const (
	ArtifactGitHubRunner     CleanupArtifact = "github_runner"
	ArtifactHostRegistration CleanupArtifact = "host_registration"
	ArtifactSystemdService   CleanupArtifact = "systemd_service"
	ArtifactRunnerFiles      CleanupArtifact = "runner_files"
	ArtifactLocalState       CleanupArtifact = "local_state"
)

type CleanupArtifactPlan struct {
	Artifact             CleanupArtifact `json:"artifact"`
	Description          string          `json:"description"`
	Action               string          `json:"action"`
	DefaultSelected      bool            `json:"default_selected"`
	RequiresConfirmation bool            `json:"requires_confirmation"`
	Blocked              bool            `json:"blocked"`
	BlockReason          string          `json:"block_reason,omitempty"`
}

type CleanupPlan struct {
	Repo          string                `json:"repo"`
	RunnerName    string                `json:"runner_name"`
	MachineTarget string                `json:"machine_target"`
	DryRun        bool                  `json:"dry_run"`
	Artifacts     []CleanupArtifactPlan `json:"artifacts"`
	Warnings      []string              `json:"warnings"`
}

func BuildCleanupPlan(repoState state.RepositoryState, dryRun bool) CleanupPlan {
	installPath, workDir, blocked, reason := SafeRunnerPaths(repoState)
	plan := CleanupPlan{Repo: repoState.Repo.FullName, RunnerName: repoState.Runner.Name, MachineTarget: repoState.Machine.HostRef, DryRun: dryRun, Warnings: []string{"RunnerKit will not remove shared /var/lib/runnerkit or shared users."}}
	plan.Artifacts = append(plan.Artifacts,
		CleanupArtifactPlan{Artifact: ArtifactGitHubRunner, Description: "GitHub runner", Action: "delete GitHub runner id 123 (runnerkit-owner-repo-local)", DefaultSelected: true, RequiresConfirmation: true},
		CleanupArtifactPlan{Artifact: ArtifactHostRegistration, Description: "Host registration", Action: "unconfigure runner registration from " + repoState.Machine.InstallPath, DefaultSelected: true, RequiresConfirmation: true},
		CleanupArtifactPlan{Artifact: ArtifactSystemdService, Description: "systemd service", Action: "stop and uninstall service " + repoState.Machine.ServiceName, DefaultSelected: true, RequiresConfirmation: true},
		CleanupArtifactPlan{Artifact: ArtifactRunnerFiles, Description: "runner files", Action: "remove " + installPath + " and " + workDir, DefaultSelected: true, RequiresConfirmation: true, Blocked: blocked, BlockReason: reason},
		CleanupArtifactPlan{Artifact: ArtifactLocalState, Description: "local state", Action: "remove local RunnerKit state for " + repoState.Repo.FullName + " after selected cleanup succeeds", DefaultSelected: true, RequiresConfirmation: true},
	)
	if repoState.Cleanup.GitHubRunnerID != 0 || repoState.Runner.Name != "" {
		plan.Artifacts[0].Action = "delete GitHub runner id " + formatInt64(repoState.Cleanup.GitHubRunnerID) + " (" + repoState.Runner.Name + ")"
	}
	return plan
}

func SafeRunnerPaths(repoState state.RepositoryState) (string, string, bool, string) {
	runnerName := strings.TrimSpace(repoState.Runner.Name)
	installPath := strings.TrimSpace(repoState.Machine.InstallPath)
	workDir := strings.TrimSpace(repoState.Machine.WorkDir)
	if runnerName == "" || unsafePath(installPath) || unsafePath(workDir) {
		return "", "", true, "unsafe or empty runner path"
	}
	installBase := filepath.Base(installPath)
	installDir := filepath.Dir(installPath)
	if installBase != runnerName || installDir != "/opt/actions-runner" {
		return "", "", true, "install path is not the recorded RunnerKit runner directory"
	}
	if workDir != filepath.Join("/var/lib/runnerkit/work", runnerName) {
		return "", "", true, "work dir is not the recorded RunnerKit runner work directory"
	}
	return installPath, workDir, false, ""
}

func unsafePath(path string) bool {
	switch strings.TrimSpace(path) {
	case "", "/", "/opt", "/opt/actions-runner", "/var/lib", "/var/lib/runnerkit":
		return true
	default:
		return false
	}
}

func formatInt64(v int64) string {
	if v == 0 {
		return "0"
	}
	var digits []byte
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	return string(digits)
}
