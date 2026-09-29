package ops

import (
	"strings"
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
	plan = BuildRecoveryPlan(repo, baseRecoveryObserved(repo), []RecoveryAction{ActionRestartService}, false)
	if len(plan.Steps) != 1 || plan.Steps[0].Action != ActionRestartService {
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

// A-21: the BYO steps warn that BYO setup is unsupported before the step
// that removes the runner, and the up step carries --accept-known-issues.
func TestManualReregisterSteps_BYOWarnsBeforeDown(t *testing.T) {
	steps := strings.Join(ManualReregisterSteps("owner/repo", false), "\n")
	warn := strings.Index(steps, "BYO setup is not supported in this release")
	down := strings.Index(steps, "runnerkit down --repo owner/repo")
	if warn < 0 || down < 0 || warn > down {
		t.Fatalf("warning must come before down:\n%s", steps)
	}
	if !strings.Contains(steps, "runnerkit up --repo owner/repo --host user@host --accept-known-issues") {
		t.Fatalf("up step must carry --accept-known-issues:\n%s", steps)
	}
	if cloud := strings.Join(ManualReregisterSteps("owner/repo", true), "\n"); strings.Contains(cloud, "accept-known-issues") {
		t.Fatalf("cloud steps must not change:\n%s", cloud)
	}
}
