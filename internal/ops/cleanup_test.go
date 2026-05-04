package ops

import (
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

func TestBuildCleanupPlanAndSafeRunnerPaths(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	repo.Cleanup.ManagedPaths = []string{"/opt/actions-runner/runnerkit-owner-repo-local", "/var/lib/runnerkit"}
	plan := BuildCleanupPlan(repo, true)
	if len(plan.Artifacts) != 5 || !strings.Contains(plan.Warnings[0], "RunnerKit will not remove shared /var/lib/runnerkit or shared users.") {
		t.Fatalf("unexpected cleanup plan: %#v", plan)
	}
	for _, artifact := range []CleanupArtifact{ArtifactGitHubRunner, ArtifactHostRegistration, ArtifactSystemdService, ArtifactRunnerFiles, ArtifactLocalState} {
		found := false
		for _, item := range plan.Artifacts {
			if item.Artifact == artifact {
				found = true
				if strings.Contains(item.Action, "/var/lib/runnerkit and") || item.Action == "/var/lib/runnerkit" {
					t.Fatalf("cleanup action targets shared parent: %#v", item)
				}
			}
		}
		if !found {
			t.Fatalf("cleanup plan missing %s: %#v", artifact, plan)
		}
	}
	install, work, blocked, reason := SafeRunnerPaths(repo)
	if blocked || install != testsupport.TestInstallPath || work != testsupport.TestWorkDir || reason != "" {
		t.Fatalf("healthy paths should be safe: install=%q work=%q blocked=%v reason=%q", install, work, blocked, reason)
	}
}

func TestSafeRunnerPathsBlocksUnsafePaths(t *testing.T) {
	for _, unsafe := range []string{"/", "/opt", "/opt/actions-runner", "/var/lib/runnerkit"} {
		repo := testsupport.HealthyRepositoryState()
		repo.Machine.InstallPath = unsafe
		_, _, blocked, _ := SafeRunnerPaths(repo)
		if !blocked {
			t.Fatalf("install path %q should be blocked", unsafe)
		}
	}
	repo := testsupport.HealthyRepositoryState()
	repo.Machine.WorkDir = "/var/lib/runnerkit"
	_, _, blocked, _ := SafeRunnerPaths(repo)
	if !blocked {
		t.Fatalf("work dir /var/lib/runnerkit should be blocked")
	}
}
