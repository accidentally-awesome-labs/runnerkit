package ops

import (
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

func baseRecoveryObserved(repoState any) ObservedRunner {
	repo := testsupport.HealthyRepositoryState()
	return ObservedRunner{Repo: repo.Repo.FullName, StatePresent: true, State: &repo, GitHub: GitHubFact{Found: true, ID: 123, Name: repo.Runner.Name, Status: "online", Labels: repo.Runner.Labels}, SSH: SSHFact{Reachable: true, HostKey: "matched"}, Service: ServiceFact{Service: repo.Machine.ServiceName, ActiveState: "active", LoadState: "loaded"}, Labels: CompareLabels(repo.Runner.Labels, repo.Runner.Labels)}
}

func TestBuildRecoveryPlanSelectsActionsAndBlocksUnsafeCases(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	observed := baseRecoveryObserved(repo)
	observed.Service.ActiveState = "failed"
	plan := BuildRecoveryPlan(repo, observed, nil, true)
	if len(plan.Steps) != 1 || plan.Steps[0].Action != ActionRestartService || plan.Steps[0].Description != "Restart systemd service actions.runner.runnerkit-owner-repo-local.service" {
		t.Fatalf("failed service should select restart_service: %#v", plan)
	}
	observed = baseRecoveryObserved(repo)
	observed.GitHub = GitHubFact{Found: false}
	plan = BuildRecoveryPlan(repo, observed, nil, false)
	if len(plan.Steps) != 1 || plan.Steps[0].Action != ActionReregisterRunner {
		t.Fatalf("missing GitHub runner should select reregister_runner: %#v", plan)
	}
	plan = BuildRecoveryPlan(repo, baseRecoveryObserved(repo), []RecoveryAction{ActionReinstallService}, false)
	if len(plan.Steps) != 1 || plan.Steps[0].Action != ActionReinstallService {
		t.Fatalf("explicit action should be preserved: %#v", plan)
	}
	observed = baseRecoveryObserved(repo)
	observed.SSH.HostKey = "mismatch"
	plan = BuildRecoveryPlan(repo, observed, []RecoveryAction{ActionRestartService}, false)
	if !plan.Blocked || plan.BlockReason != "SSH host key mismatch; verify the machine identity before recovery." {
		t.Fatalf("host-key mismatch should block: %#v", plan)
	}
	observed = baseRecoveryObserved(repo)
	observed.SSH.Reachable = false
	plan = BuildRecoveryPlan(repo, observed, nil, false)
	if !plan.Blocked || plan.BlockReason != "SSH unreachable; fix SSH access before recovery." {
		t.Fatalf("SSH unreachable should block: %#v", plan)
	}
	plan = BuildRecoveryPlan(repo, baseRecoveryObserved(repo), nil, false)
	if !plan.Blocked || plan.BlockReason != "No recovery action is recommended; run runnerkit doctor --repo owner/repo." {
		t.Fatalf("healthy runner should block with no recommendation: %#v", plan)
	}
}
