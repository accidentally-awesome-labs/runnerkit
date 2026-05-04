package ops

import (
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

type RecoveryAction string

const (
	ActionRestartService   RecoveryAction = "restart_service"
	ActionReinstallService RecoveryAction = "reinstall_service"
	ActionReregisterRunner RecoveryAction = "reregister_runner"
)

type RecoveryStep struct {
	ID                   string         `json:"id"`
	Action               RecoveryAction `json:"action"`
	Description          string         `json:"description"`
	CommandID            string         `json:"command_id"`
	RequiresConfirmation bool           `json:"requires_confirmation"`
}

type RecoveryPlan struct {
	Repo        string         `json:"repo"`
	RunnerName  string         `json:"runner_name"`
	DryRun      bool           `json:"dry_run"`
	Blocked     bool           `json:"blocked"`
	BlockReason string         `json:"block_reason,omitempty"`
	Steps       []RecoveryStep `json:"steps"`
}

func BuildRecoveryPlan(repoState state.RepositoryState, observed ObservedRunner, requested []RecoveryAction, dryRun bool) RecoveryPlan {
	plan := RecoveryPlan{Repo: repoState.Repo.FullName, RunnerName: repoState.Runner.Name, DryRun: dryRun, Steps: []RecoveryStep{}}
	if observed.SSH.HostKey == "mismatch" {
		plan.Blocked = true
		plan.BlockReason = "SSH host key mismatch; verify the machine identity before recovery."
		return plan
	}
	if !observed.SSH.Reachable {
		plan.Blocked = true
		plan.BlockReason = "SSH unreachable; fix SSH access before recovery."
		return plan
	}
	actions := append([]RecoveryAction(nil), requested...)
	if len(actions) == 0 {
		action, ok := recommendedRecoveryAction(repoState, observed)
		if !ok {
			plan.Blocked = true
			plan.BlockReason = "No recovery action is recommended; run runnerkit doctor --repo " + repoState.Repo.FullName + "."
			return plan
		}
		actions = append(actions, action)
	}
	for _, action := range actions {
		plan.Steps = append(plan.Steps, recoveryStep(repoState, action))
	}
	return plan
}

func recommendedRecoveryAction(repoState state.RepositoryState, observed ObservedRunner) (RecoveryAction, bool) {
	if serviceFailed(observed.Service) || (observed.Service.ActiveState != "" && observed.Service.ActiveState != "active") {
		return ActionRestartService, true
	}
	if (observed.Service.LoadState == "not-found" || strings.Contains(strings.ToLower(observed.Service.Error), "missing")) && repoState.Machine.InstallPath != "" {
		return ActionReinstallService, true
	}
	if !observed.GitHub.Found || !observed.Labels.Match {
		if repoState.Machine.InstallPath != "" {
			return ActionReregisterRunner, true
		}
	}
	return "", false
}

func recoveryStep(repoState state.RepositoryState, action RecoveryAction) RecoveryStep {
	switch action {
	case ActionRestartService:
		return RecoveryStep{ID: "restart_service", Action: action, Description: "Restart systemd service " + repoState.Machine.ServiceName, CommandID: "recover.service.restart", RequiresConfirmation: true}
	case ActionReinstallService:
		return RecoveryStep{ID: "reinstall_service", Action: action, Description: "Reinstall and start service from " + repoState.Machine.InstallPath, CommandID: "recover.service.reinstall", RequiresConfirmation: true}
	case ActionReregisterRunner:
		return RecoveryStep{ID: "reregister_runner", Action: action, Description: "Re-register " + repoState.Runner.Name + " with saved labels and work dir " + repoState.Machine.WorkDir, CommandID: "recover.runner.configure", RequiresConfirmation: true}
	default:
		return RecoveryStep{ID: string(action), Action: action, Description: string(action), RequiresConfirmation: true}
	}
}
