package ops

import (
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

// A-06b (P1-10): the recovery planner never recommends reinstall_service or
// reregister_runner, and refuses them when requested explicitly.
func TestBuildRecoveryPlan_NeverRecommendsDisabledActions(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	cases := map[string]func(o *ObservedRunner){
		"github_runner_missing": func(o *ObservedRunner) { o.GitHub = GitHubFact{Found: false} },
		"label_drift":           func(o *ObservedRunner) { o.Labels = CompareLabels(repo.Runner.Labels, []string{"self-hosted"}) },
		"service_missing": func(o *ObservedRunner) {
			o.Service = ServiceFact{Service: repo.Machine.ServiceName, LoadState: "not-found", ActiveState: "inactive"}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			observed := baseRecoveryObserved(repo)
			mutate(&observed)
			plan := BuildRecoveryPlan(repo, observed, nil, true)
			for _, step := range plan.Steps {
				if DisabledRecoveryAction(step.Action) {
					t.Fatalf("planner recommended disabled action %q: %#v", step.Action, plan)
				}
			}
			if !plan.Blocked || !strings.Contains(plan.BlockReason, "runnerkit down --repo owner/repo") || !strings.Contains(plan.BlockReason, "runnerkit up --repo owner/repo") {
				t.Fatalf("expected a blocked plan with manual re-register steps: %#v", plan)
			}
		})
	}
}

func TestBuildRecoveryPlan_RefusesRequestedDisabledActions(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	for _, action := range []RecoveryAction{ActionReinstallService, ActionReregisterRunner} {
		plan := BuildRecoveryPlan(repo, baseRecoveryObserved(repo), []RecoveryAction{action}, false)
		if !plan.Blocked || len(plan.Steps) != 0 {
			t.Fatalf("%s should be refused: %#v", action, plan)
		}
		if !strings.Contains(plan.BlockReason, "disabled") || !strings.Contains(plan.BlockReason, "runnerkit up --repo owner/repo") {
			t.Fatalf("%s refusal missing manual steps: %q", action, plan.BlockReason)
		}
	}
	cloud := testsupport.CloudRepositoryState()
	plan := BuildRecoveryPlan(cloud, baseRecoveryObserved(cloud), []RecoveryAction{ActionReregisterRunner}, false)
	if !strings.Contains(plan.BlockReason, "runnerkit destroy --repo owner/repo") || strings.Contains(plan.BlockReason, "runnerkit down") {
		t.Fatalf("cloud state must be pointed at destroy, not down: %q", plan.BlockReason)
	}
}

// A-04 (doctor part): cleanup_pending on RunnerKit-managed cloud state points
// at `destroy --dry-run`; `down` would drop the record of a billing server.
func TestDoctor_CleanupPendingCloudPointsToDestroy(t *testing.T) {
	cloud := testsupport.CloudRepositoryState()
	cloud.Cleanup.Notes = []string{"github runner removal pending"}
	byo := testsupport.HealthyRepositoryState()
	byo.Cleanup.Notes = []string{"github runner removal pending"}
	for _, tc := range []struct {
		name    string
		want    string
		notWant string
	}{
		{name: "cloud", want: "runnerkit destroy --repo owner/repo --dry-run", notWant: "runnerkit down"},
		{name: "byo", want: "runnerkit down --repo owner/repo --dry-run", notWant: "runnerkit destroy"},
	} {
		repoState := byo
		if tc.name == "cloud" {
			repoState = cloud
		}
		observed := baseRecoveryObserved(repoState)
		report := BuildDoctorReport(repoState, observed, DeepChecks{InstallPathOK: true, WorkDirOK: true}, nil)
		var remediation string
		for _, f := range report.Findings {
			if f.ID == "cleanup_pending" {
				remediation = f.Remediation
			}
		}
		if !strings.Contains(remediation, tc.want) || strings.Contains(remediation, tc.notWant) {
			t.Fatalf("%s cleanup_pending remediation = %q, want %q and not %q", tc.name, remediation, tc.want, tc.notWant)
		}
	}
}
