package runmode

import (
	"reflect"
	"strings"
	"testing"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
)

func TestModeAndProfileConstants(t *testing.T) {
	if ModePersistent != "persistent" {
		t.Fatalf("ModePersistent = %q", ModePersistent)
	}
	if ModeEphemeral != "ephemeral" {
		t.Fatalf("ModeEphemeral = %q", ModeEphemeral)
	}
	if ProfilePersistentTrusted != "persistent-trusted" {
		t.Fatalf("ProfilePersistentTrusted = %q", ProfilePersistentTrusted)
	}
	if ProfilePersistentRisky != "persistent-risky" {
		t.Fatalf("ProfilePersistentRisky = %q", ProfilePersistentRisky)
	}
	if ProfileEphemeralBYO != "ephemeral-byo" {
		t.Fatalf("ProfileEphemeralBYO = %q", ProfileEphemeralBYO)
	}
	if ProfileEphemeralCloud != "ephemeral-cloud" {
		t.Fatalf("ProfileEphemeralCloud = %q", ProfileEphemeralCloud)
	}
	if DefaultEphemeralTTL != 24*time.Hour {
		t.Fatalf("DefaultEphemeralTTL = %v, want 24h", DefaultEphemeralTTL)
	}
}

func TestNormalizeAcceptsEmptyAndPersistentAndEphemeral(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", ModePersistent},
		{"persistent", ModePersistent},
		{"ephemeral", ModeEphemeral},
		{"  ephemeral  ", ModeEphemeral},
		{"PERSISTENT", ModePersistent},
	}
	for _, tc := range cases {
		got, err := Normalize(tc.input)
		if err != nil {
			t.Fatalf("Normalize(%q) error = %v", tc.input, err)
		}
		if got != tc.want {
			t.Fatalf("Normalize(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeRejectsInvalidValuesWithExpectedCopy(t *testing.T) {
	for _, raw := range []string{"static", "fleet", "ephemeralish", "EPHEM"} {
		_, err := Normalize(raw)
		if err == nil {
			t.Fatalf("Normalize(%q) expected error", raw)
		}
		if !strings.Contains(err.Error(), "Supported modes: --mode persistent or --mode ephemeral.") {
			t.Fatalf("Normalize(%q) err = %v, want supported modes copy", raw, err)
		}
	}
}

func TestEvaluatePersistentTrustedTradeoffStrings(t *testing.T) {
	repo := gh.Repo{FullName: "owner/name", Private: true}
	decision := Evaluate(repo, Options{Mode: ModePersistent, SetupPath: "byo"})
	if decision.Mode != ModePersistent {
		t.Fatalf("decision.Mode = %q", decision.Mode)
	}
	if decision.SafetyProfile != ProfilePersistentTrusted {
		t.Fatalf("SafetyProfile = %q", decision.SafetyProfile)
	}
	wantCost := "Lowest ongoing setup cost for repeated trusted private jobs; cloud resources keep billing until cleanup."
	if decision.Tradeoffs.Cost != wantCost {
		t.Fatalf("Cost = %q, want %q", decision.Tradeoffs.Cost, wantCost)
	}
	wantIsolation := "Reuses one runner across jobs, so workflow code can persist on the machine."
	if decision.Tradeoffs.Isolation != wantIsolation {
		t.Fatalf("Isolation = %q", decision.Tradeoffs.Isolation)
	}
	wantCleanup := "Requires runnerkit down for BYO or runnerkit destroy for cloud."
	if decision.Tradeoffs.Cleanup != wantCleanup {
		t.Fatalf("Cleanup = %q", decision.Tradeoffs.Cleanup)
	}
	wantOps := "Best for trusted private solo-development workflows that run repeatedly."
	if decision.Tradeoffs.Operations != wantOps {
		t.Fatalf("Operations = %q", decision.Tradeoffs.Operations)
	}
	wantLogs := "Live runner _diag and systemd logs remain on the machine until cleanup."
	if decision.Tradeoffs.Logs != wantLogs {
		t.Fatalf("Logs = %q", decision.Tradeoffs.Logs)
	}
}

func TestEvaluatePersistentRiskyForPublicRepoExposesRecommendedFlag(t *testing.T) {
	repo := gh.Repo{FullName: "owner/name", Private: false}
	decision := Evaluate(repo, Options{Mode: ModePersistent, SetupPath: "byo"})
	if decision.SafetyProfile != ProfilePersistentRisky {
		t.Fatalf("SafetyProfile = %q, want %q", decision.SafetyProfile, ProfilePersistentRisky)
	}
	combined := strings.Join(decision.NotRecommendedFor, " | ") + " | " + strings.Join(decision.Warnings, " | ")
	if !strings.Contains(combined, "public") || !strings.Contains(combined, "fork") {
		t.Fatalf("expected public/fork warnings: %q", combined)
	}
}

func TestEvaluateEphemeralBYOTradeoffStrings(t *testing.T) {
	repo := gh.Repo{FullName: "owner/name", Private: true}
	decision := Evaluate(repo, Options{Mode: ModeEphemeral, SetupPath: "byo"})
	if decision.Mode != ModeEphemeral {
		t.Fatalf("Mode = %q", decision.Mode)
	}
	if decision.SafetyProfile != ProfileEphemeralBYO {
		t.Fatalf("SafetyProfile = %q", decision.SafetyProfile)
	}
	wantCost := "Higher setup and cleanup cost per run; cloud resources keep billing until destroy verifies cleanup."
	if decision.Tradeoffs.Cost != wantCost {
		t.Fatalf("Cost = %q", decision.Tradeoffs.Cost)
	}
	wantIsolation := "GitHub assigns at most one job then deregisters the runner; BYO hosts are still reused machines."
	if decision.Tradeoffs.Isolation != wantIsolation {
		t.Fatalf("Isolation = %q", decision.Tradeoffs.Isolation)
	}
	wantCleanup := "TTL finalizer preserves logs and cleanup still uses down for BYO or destroy for cloud."
	if decision.Tradeoffs.Cleanup != wantCleanup {
		t.Fatalf("Cleanup = %q", decision.Tradeoffs.Cleanup)
	}
	wantOps := "One scoped runner only; not autoscaling or a fleet manager."
	if decision.Tradeoffs.Operations != wantOps {
		t.Fatalf("Operations = %q", decision.Tradeoffs.Operations)
	}
	wantLogs := "RunnerKit preserves best-effort _diag and systemd journal logs before cleanup."
	if decision.Tradeoffs.Logs != wantLogs {
		t.Fatalf("Logs = %q", decision.Tradeoffs.Logs)
	}
}

func TestEvaluateEphemeralCloudProfile(t *testing.T) {
	repo := gh.Repo{FullName: "owner/name", Private: false}
	decision := Evaluate(repo, Options{Mode: ModeEphemeral, SetupPath: "cloud"})
	if decision.SafetyProfile != ProfileEphemeralCloud {
		t.Fatalf("SafetyProfile = %q", decision.SafetyProfile)
	}
	if decision.Mode != ModeEphemeral {
		t.Fatalf("Mode = %q", decision.Mode)
	}
	if decision.Tradeoffs.Operations != "One scoped runner only; not autoscaling or a fleet manager." {
		t.Fatalf("ephemeral cloud Operations = %q", decision.Tradeoffs.Operations)
	}
	if decision.Tradeoffs.Logs != "RunnerKit preserves best-effort _diag and systemd journal logs before cleanup." {
		t.Fatalf("ephemeral cloud Logs = %q", decision.Tradeoffs.Logs)
	}
}

func TestDecisionRecommendedForAndNotRecommendedFor(t *testing.T) {
	persistent := Evaluate(gh.Repo{FullName: "owner/name", Private: true}, Options{Mode: ModePersistent, SetupPath: "byo"})
	if len(persistent.RecommendedFor) == 0 || len(persistent.NotRecommendedFor) == 0 {
		t.Fatalf("persistent missing recommended/not-recommended: %#v", persistent)
	}
	ephemeralCloud := Evaluate(gh.Repo{FullName: "owner/name", Private: false}, Options{Mode: ModeEphemeral, SetupPath: "cloud"})
	if len(ephemeralCloud.RecommendedFor) == 0 {
		t.Fatalf("ephemeral cloud RecommendedFor empty")
	}
	// Ensure equality checks with deep equal tolerate slice ordering by sorting joined strings.
	if reflect.DeepEqual(persistent.RecommendedFor, ephemeralCloud.RecommendedFor) {
		t.Fatalf("persistent and ephemeral cloud should differ: %v == %v", persistent.RecommendedFor, ephemeralCloud.RecommendedFor)
	}
}
