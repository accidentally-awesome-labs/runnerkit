package github

import (
	"strings"
	"testing"
)

func TestPublicRepoRiskBodyMatchesUISpec(t *testing.T) {
	want := "Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows."
	if PublicRepoRiskBody != want {
		t.Fatalf("PublicRepoRiskBody = %q, want %q", PublicRepoRiskBody, want)
	}
}

func TestPublicRepoRiskNextActionRecommendsGitHubHosted(t *testing.T) {
	want := "Use GitHub-hosted runners for public, fork-based, or untrusted code (free and unlimited for public repositories)."
	if PublicRepoRiskNextAction != want {
		t.Fatalf("PublicRepoRiskNextAction = %q, want %q", PublicRepoRiskNextAction, want)
	}
}

func TestDangerousPersistentOverrideCopyExistsAndMatchesUISpec(t *testing.T) {
	want := "Only pass `--allow-public-repo-risk` if you accept that untrusted code can execute repeatedly on your machine."
	if DangerousPersistentOverrideCopy != want {
		t.Fatalf("DangerousPersistentOverrideCopy = %q, want %q", DangerousPersistentOverrideCopy, want)
	}
}

func TestEvaluateSafetyPublicWarningsRecommendGitHubHostedNotEphemeralCloud(t *testing.T) {
	decision := EvaluateSafety(Repo{FullName: "owner/name", Private: false}, SafetyOptions{})
	combined := strings.Join(decision.Warnings, " | ")
	if !strings.Contains(combined, "GitHub-hosted runners") {
		t.Fatalf("expected GitHub-hosted recommendation in warnings: %q", combined)
	}
	if strings.Contains(combined, "--mode ephemeral --cloud") {
		t.Fatalf("warnings must not recommend ephemeral cloud (disabled in v1.3.4): %q", combined)
	}
}
