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

func TestPublicRepoRiskNextActionRecommendsEphemeralCloud(t *testing.T) {
	want := "Use `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner` for stronger isolation, or use GitHub-hosted runners."
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

func TestEvaluateSafetyPublicWarningsIncludeEphemeralCloudCommand(t *testing.T) {
	decision := EvaluateSafety(Repo{FullName: "owner/name", Private: false}, SafetyOptions{})
	combined := strings.Join(decision.Warnings, " | ")
	if !strings.Contains(combined, "runnerkit up --repo owner/name --mode ephemeral --cloud hetzner") {
		t.Fatalf("expected ephemeral cloud command in warnings: %q", combined)
	}
}
