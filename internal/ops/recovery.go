package ops

import (
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

type RecoveryAction string

const (
	ActionRestartService RecoveryAction = "restart_service"
	// ActionReinstallService and ActionReregisterRunner are disabled in
	// v1.3.4 (P1-10): reinstall ran `svc.sh install` on top of an existing
	// unit, and reregister uninstalled the service and then only ran
	// `svc.sh start`. The planner never recommends them and refuses them
	// when requested; the constants stay for the flag and JSON vocabulary.
	ActionReinstallService RecoveryAction = "reinstall_service"
	ActionReregisterRunner RecoveryAction = "reregister_runner"
)

// DisabledRecoveryAction reports whether action is refused in this release.
func DisabledRecoveryAction(action RecoveryAction) bool {
	return action == ActionReinstallService || action == ActionReregisterRunner
}

// IsCloudState reports whether the saved state describes a RunnerKit-managed
// cloud server (Hetzner). Cleanup and re-registration for those go through
// `runnerkit destroy`, never `runnerkit down`, which would drop the only
// record of a server that keeps billing.
func IsCloudState(repoState state.RepositoryState) bool {
	return repoState.Provider.Kind == "hetzner" || repoState.Provider.Name == "hetzner"
}

// CleanupDryRunCommand is the cleanup preview matching the saved state:
// `destroy --dry-run` for RunnerKit-managed cloud servers, `down --dry-run`
// otherwise.
func CleanupDryRunCommand(repoState state.RepositoryState) string {
	repo := repoState.Repo.FullName
	if IsCloudState(repoState) {
		return "runnerkit destroy --repo " + repo + " --dry-run"
	}
	return "runnerkit down --repo " + repo + " --dry-run"
}

// ManualReregisterSteps is the supported way to reinstall or re-register a
// runner while `upgrade-runner`, `doctor --fix`, `recover --reinstall-service`
// and `recover --reregister` are disabled: remove the runner with the
// cleanup command, then run `runnerkit up` again, which requests a fresh
// registration token and installs the service from scratch. repo may be ""
// when it is not known yet.
func ManualReregisterSteps(repo string, cloud bool) []string {
	if strings.TrimSpace(repo) == "" {
		repo = "owner/name"
	}
	if cloud {
		return []string{
			"Re-register by hand: run runnerkit destroy --repo " + repo + " --dry-run to review, then runnerkit destroy --repo " + repo + " (removes the GitHub registration and the Hetzner server).",
			"Then run runnerkit up --repo " + repo + " --experimental --cloud hetzner --cloud-region <location> to create and register a fresh runner.",
		}
	}
	// v1.3.4 ships without a passing real-job BYO gate (A-21): say so
	// before the step that removes a working runner.
	return []string{
		"BYO setup is not supported in this release: runnerkit down removes the runner, and setting it up again needs runnerkit up --accept-known-issues. If the runner still runs jobs, you can leave it as it is.",
		"Re-register by hand: run runnerkit down --repo " + repo + " --dry-run to review, then runnerkit down --repo " + repo + " (removes the GitHub registration and the service).",
		"Then run runnerkit up --repo " + repo + " --host user@host --accept-known-issues to install and register a fresh runner.",
	}
}

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
	manual := strings.Join(ManualReregisterSteps(repoState.Repo.FullName, IsCloudState(repoState)), " ")
	for _, action := range requested {
		if DisabledRecoveryAction(action) {
			plan.Blocked = true
			plan.BlockReason = "recover " + string(action) + " is disabled in this release (known issue). " + manual
			return plan
		}
	}
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
		action, ok := recommendedRecoveryAction(observed)
		if !ok {
			plan.Blocked = true
			plan.BlockReason = "No recovery action is recommended; run runnerkit doctor --repo " + repoState.Repo.FullName + "."
			if observed.GitHub.Error != "" {
				// Without GitHub facts a missing runner or label drift
				// cannot be told apart from an API failure; never send
				// users to destroy/down on that basis.
				plan.BlockReason = "GitHub facts unavailable (" + observed.GitHub.Error + "); run runnerkit doctor --repo " + repoState.Repo.FullName + " once GitHub access works."
			} else if needsReregistration(observed) {
				plan.BlockReason = "The runner needs a service reinstall or re-registration, which recover cannot do in this release. " + manual
			}
			return plan
		}
		actions = append(actions, action)
	}
	for _, action := range actions {
		plan.Steps = append(plan.Steps, recoveryStep(repoState, action))
	}
	return plan
}

// recommendedRecoveryAction only ever recommends restart_service. Cases that
// need a reinstall or re-registration are reported by needsReregistration
// with manual steps instead (A-06b).
func recommendedRecoveryAction(observed ObservedRunner) (RecoveryAction, bool) {
	if serviceMissing(observed.Service) {
		return "", false
	}
	if serviceFailed(observed.Service) || (observed.Service.ActiveState != "" && observed.Service.ActiveState != "active") {
		return ActionRestartService, true
	}
	return "", false
}

func needsReregistration(observed ObservedRunner) bool {
	if observed.GitHub.Error != "" {
		return false
	}
	return serviceMissing(observed.Service) || !observed.GitHub.Found || !observed.Labels.Match
}

func serviceMissing(service ServiceFact) bool {
	return service.LoadState == "not-found" || strings.Contains(strings.ToLower(service.Error), "missing")
}

func recoveryStep(repoState state.RepositoryState, action RecoveryAction) RecoveryStep {
	switch action {
	case ActionRestartService:
		return RecoveryStep{ID: "restart_service", Action: action, Description: "Restart systemd service " + repoState.Machine.ServiceName, CommandID: "recover.service.restart", RequiresConfirmation: true}
	default:
		return RecoveryStep{ID: string(action), Action: action, Description: string(action), RequiresConfirmation: true}
	}
}
