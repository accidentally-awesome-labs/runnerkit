package ops

import "github.com/accidentally-awesome-labs/runnerkit/internal/state"

type CloudDestroyArtifact string

const (
	ArtifactCloudGitHubRunner      CloudDestroyArtifact = "github_runner"
	ArtifactCloudRemoteRunner      CloudDestroyArtifact = "remote_runner"
	ArtifactCloudProviderServer    CloudDestroyArtifact = "provider_server"
	ArtifactCloudProviderSSHKey    CloudDestroyArtifact = "provider_ssh_key"
	ArtifactCloudProviderFirewall  CloudDestroyArtifact = "provider_firewall"
	ArtifactCloudProviderPrimaryIP CloudDestroyArtifact = "provider_primary_ip"
	ArtifactCloudLocalState        CloudDestroyArtifact = "local_state"
)

type CloudDestroyArtifactPlan struct {
	Artifact             CloudDestroyArtifact `json:"artifact"`
	Description          string               `json:"description"`
	Action               string               `json:"action"`
	DefaultSelected      bool                 `json:"default_selected"`
	RequiresConfirmation bool                 `json:"requires_confirmation"`
	Blocked              bool                 `json:"blocked"`
	BlockReason          string               `json:"block_reason,omitempty"`
}

type CloudDestroyPlan struct {
	Repo          string                     `json:"repo"`
	RunnerName    string                     `json:"runner_name"`
	Provider      string                     `json:"provider"`
	MachineTarget string                     `json:"machine_target"`
	DryRun        bool                       `json:"dry_run"`
	Artifacts     []CloudDestroyArtifactPlan `json:"artifacts"`
	Warnings      []string                   `json:"warnings"`
}

const CloudDestroyBillingWarning = "This removes RunnerKit-created cloud resources that may still be billing. Local state is removed only after GitHub and provider cleanup are verified."

func BuildCloudDestroyPlan(repoState state.RepositoryState, dryRun bool) CloudDestroyPlan {
	providerName := repoState.Provider.Kind
	if providerName == "" {
		providerName = repoState.Provider.Name
	}
	plan := CloudDestroyPlan{
		Repo:          repoState.Repo.FullName,
		RunnerName:    repoState.Runner.Name,
		Provider:      providerName,
		MachineTarget: repoState.Machine.HostRef,
		DryRun:        dryRun,
		Warnings:      []string{CloudDestroyBillingWarning},
	}
	plan.Artifacts = []CloudDestroyArtifactPlan{
		{Artifact: ArtifactCloudGitHubRunner, Description: "GitHub runner", Action: "remove GitHub runner registration " + formatInt64(repoState.Cleanup.GitHubRunnerID) + " (" + repoState.Runner.Name + ")", DefaultSelected: true, RequiresConfirmation: true},
		{Artifact: ArtifactCloudRemoteRunner, Description: "Remote runner", Action: "remove runner service and files from " + repoState.Machine.HostRef, DefaultSelected: true, RequiresConfirmation: true},
		{Artifact: ArtifactCloudProviderServer, Description: "Hetzner server", Action: "delete server " + repoState.Provider.ResourceIDs["server"], DefaultSelected: true, RequiresConfirmation: true},
		{Artifact: ArtifactCloudProviderSSHKey, Description: "RunnerKit-created SSH key", Action: "delete SSH key " + repoState.Provider.ResourceIDs["ssh_key"], DefaultSelected: true, RequiresConfirmation: true},
		{Artifact: ArtifactCloudProviderFirewall, Description: "RunnerKit-created firewall", Action: "delete firewall " + repoState.Provider.ResourceIDs["firewall"], DefaultSelected: true, RequiresConfirmation: true},
		{Artifact: ArtifactCloudProviderPrimaryIP, Description: "Primary IPv4/IPv6", Action: "delete or verify non-billable primary IPs", DefaultSelected: true, RequiresConfirmation: true},
		{Artifact: ArtifactCloudLocalState, Description: "Local state", Action: "remove local RunnerKit state for " + repoState.Repo.FullName + " after GitHub and provider cleanup verify complete", DefaultSelected: true, RequiresConfirmation: true},
	}
	return plan
}
