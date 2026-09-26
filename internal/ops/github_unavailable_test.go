package ops

import (
	"strings"
	"testing"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

// When GitHub facts could not be collected (no token, network) and SSH is
// down, doctor must not infer label drift or missing paths and must never
// recommend destroying or tearing down the runner.
func TestDoctor_GitHubUnavailableNeverRecommendsTeardown(t *testing.T) {
	for _, name := range []string{"cloud", "byo"} {
		t.Run(name, func(t *testing.T) {
			repoState := testsupport.HealthyRepositoryState()
			if name == "cloud" {
				repoState = testsupport.CloudRepositoryState()
			}
			observed := ObservedRunner{
				Repo:         repoState.Repo.FullName,
				StatePresent: true,
				State:        &repoState,
				GitHub:       GitHubFact{Error: "GitHub token is not available"},
				SSH:          SSHFact{Reachable: false, HostKey: "unknown", Error: `exec: "ssh": executable file not found in $PATH`},
				Service:      ServiceFact{Service: repoState.Machine.ServiceName, Error: "SSH target unavailable"},
				Provider:     ProviderFact{Kind: repoState.Provider.Kind, Error: "Hetzner token is not available"},
			}
			observed.Labels = CompareLabels(repoState.Runner.Labels, observed.GitHub.Labels)
			checks := DeepChecks{
				InstallPathError:    `exec: "ssh": executable file not found in $PATH`,
				WorkDirError:        `exec: "ssh": executable file not found in $PATH`,
				InstallPathProbeErr: true,
				WorkDirProbeErr:     true,
			}
			report := BuildDoctorReport(repoState, observed, checks, nil)
			ids := map[string]bool{}
			for _, f := range report.Findings {
				ids[f.ID] = true
				if strings.Contains(f.Remediation, "destroy --repo "+repoState.Repo.FullName+" (") ||
					strings.Contains(f.Remediation, "Re-register by hand") ||
					strings.Contains(f.Remediation, "down --repo") {
					t.Fatalf("finding %s recommends teardown without GitHub/SSH facts: %q", f.ID, f.Remediation)
				}
			}
			if ids["label_drift"] {
				t.Fatalf("label_drift must not be inferred from a GitHub API error: %#v", report.Findings)
			}
			if !ids["github_unavailable"] {
				t.Fatalf("expected github_unavailable finding: %#v", report.Findings)
			}
			for _, id := range []string{"install_path_missing", "work_dir_missing"} {
				for _, f := range report.Findings {
					if f.ID == id && !strings.Contains(f.Remediation, "Verify SSH access") {
						t.Fatalf("%s should point at SSH, got %q", id, f.Remediation)
					}
				}
			}
		})
	}
}

// A path probe that ran on a reachable host and exited non-zero still earns
// the manual re-register advice.
func TestDoctor_PathProbeNonZeroKeepsReregisterAdvice(t *testing.T) {
	repoState := testsupport.HealthyRepositoryState()
	observed := baseRecoveryObserved(repoState)
	report := BuildDoctorReport(repoState, observed, DeepChecks{InstallPathOK: false, WorkDirOK: true, InstallPathError: "config.sh missing"}, nil)
	for _, f := range report.Findings {
		if f.ID == "install_path_missing" {
			if !strings.Contains(f.Remediation, "Re-register by hand") {
				t.Fatalf("install_path_missing remediation = %q", f.Remediation)
			}
			return
		}
	}
	t.Fatalf("install_path_missing not reported: %#v", report.Findings)
}

func TestBuildRecoveryPlan_GitHubUnavailableBlocksWithoutTeardown(t *testing.T) {
	for _, name := range []string{"byo", "cloud"} {
		repoState := testsupport.HealthyRepositoryState()
		if name == "cloud" {
			repoState = testsupport.CloudRepositoryState()
		}
		observed := baseRecoveryObserved(repoState)
		observed.State = &repoState
		observed.GitHub = GitHubFact{Error: "GitHub token is not available"}
		observed.Labels = CompareLabels(repoState.Runner.Labels, nil)
		plan := BuildRecoveryPlan(repoState, observed, nil, true)
		if !plan.Blocked || len(plan.Steps) != 0 {
			t.Fatalf("%s: expected blocked plan, got %#v", name, plan)
		}
		if strings.Contains(plan.BlockReason, "destroy") || strings.Contains(plan.BlockReason, "down") {
			t.Fatalf("%s: BlockReason recommends teardown: %q", name, plan.BlockReason)
		}
		if !strings.Contains(plan.BlockReason, "GitHub facts unavailable") {
			t.Fatalf("%s: BlockReason = %q", name, plan.BlockReason)
		}
	}
}

// Duplicate GitHub candidates on RunnerKit-managed cloud state point at
// destroy, never down (down refuses cloud state with wrong_cleanup_command).
func TestDuplicateCandidatesCloudPointsToDestroy(t *testing.T) {
	cloud := testsupport.CloudRepositoryState()
	observed := baseRecoveryObserved(cloud)
	observed.State = &cloud
	observed.GitHub = GitHubFact{Found: false, DuplicateCandidates: []gh.Runner{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}}
	health := Classify(observed)
	if len(health.NextActions) == 0 || !strings.Contains(health.NextActions[0].Command, "destroy --repo") || strings.Contains(health.NextActions[0].Command, "down --repo") {
		t.Fatalf("Classify next action = %#v", health.NextActions)
	}
	report := BuildDoctorReport(cloud, observed, DeepChecks{InstallPathOK: true, WorkDirOK: true}, nil)
	found := false
	for _, f := range report.Findings {
		if f.ID == "github_duplicate_candidates" {
			found = true
			if !strings.Contains(f.Remediation, "destroy --repo") || strings.Contains(f.Remediation, "down --repo") {
				t.Fatalf("github_duplicate_candidates remediation = %q", f.Remediation)
			}
		}
	}
	if !found {
		t.Fatalf("missing github_duplicate_candidates: %#v", report.Findings)
	}
}

func TestClassifyEphemeralCloudDefaultCleanupIsDestroy(t *testing.T) {
	cloud := testsupport.CloudRepositoryState()
	cloud.Runner.Mode = "ephemeral"
	cloud.Ephemeral.CleanupCommand = ""
	cloud.Ephemeral.FinalizerStatus = "completed"
	observed := baseRecoveryObserved(cloud)
	observed.State = &cloud
	observed.GitHub = GitHubFact{Found: false}
	health := Classify(observed)
	if len(health.NextActions) == 0 || !strings.Contains(health.NextActions[0].Command, "runnerkit destroy --repo owner/repo") {
		t.Fatalf("ephemeral cloud cleanup next action = %#v", health.NextActions)
	}
}
