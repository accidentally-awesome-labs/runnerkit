package ops

import (
	"testing"
	"time"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/testsupport"
)

func TestClassifyHealthStates(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	base := ObservedRunner{
		Repo:         repo.Repo.FullName,
		StatePresent: true,
		State:        &repo,
		GitHub:       GitHubFact{Found: true, ID: 123, Name: repo.Runner.Name, Status: "online", Labels: repo.Runner.Labels},
		SSH:          SSHFact{Reachable: true, HostKey: "matched"},
		Service:      ServiceFact{Service: repo.Machine.ServiceName, LoadState: "loaded", ActiveState: "active", SubState: "running", ExecMainStatus: "0"},
		Labels:       CompareLabels(repo.Runner.Labels, repo.Runner.Labels),
	}
	tests := []struct {
		name string
		mut  func(*ObservedRunner)
		want HealthState
	}{
		{name: "ready", want: HealthReady},
		{name: "busy", want: HealthBusy, mut: func(o *ObservedRunner) { o.GitHub.Busy = true }},
		{name: "needs_attention offline failed", want: HealthNeedsAttention, mut: func(o *ObservedRunner) {
			o.GitHub.Status = "offline"
			o.Service.ActiveState = "failed"
			o.Service.SubState = "failed"
		}},
		{name: "needs_attention label drift", want: HealthNeedsAttention, mut: func(o *ObservedRunner) {
			o.Labels = CompareLabels(repo.Runner.Labels, []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "gpu"})
		}},
		{name: "broken host-key mismatch", want: HealthBroken, mut: func(o *ObservedRunner) { o.SSH.HostKey = "mismatch" }},
		{name: "broken duplicate candidates", want: HealthBroken, mut: func(o *ObservedRunner) { o.GitHub = GitHubFact{DuplicateCandidates: []gh.Runner{{ID: 1}, {ID: 2}}} }},
		{name: "unknown SSH unreachable", want: HealthUnknown, mut: func(o *ObservedRunner) { o.SSH.Reachable = false; o.SSH.Error = "SSH unreachable" }},
		{name: "unknown missing state", want: HealthUnknown, mut: func(o *ObservedRunner) { o.StatePresent = false; o.State = nil }},
		{name: "needs_attention missing GitHub runner", want: HealthNeedsAttention, mut: func(o *ObservedRunner) { o.GitHub = GitHubFact{Found: false} }},
		{name: "needs_attention service missing", want: HealthNeedsAttention, mut: func(o *ObservedRunner) {
			o.Service = ServiceFact{Service: repo.Machine.ServiceName, LoadState: "not-found"}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			observed := base
			if tc.mut != nil {
				tc.mut(&observed)
			}
			got := Classify(observed)
			if got.State != tc.want {
				t.Fatalf("Classify() = %s, want %s (%s)", got.State, tc.want, got.Summary)
			}
		})
	}
}

func TestClassifyEphemeralStates(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	repo.Runner.Mode = "ephemeral"
	now := repo.UpdatedAt
	expires := now.Add(24 * 60 * 60 * 1e9)
	repo.Ephemeral = state.EphemeralMetadata{Enabled: true, TTL: "24h", ExpiresAt: &expires, FinalizerStatus: "pending", CleanupCommand: "runnerkit down --repo owner/repo"}

	t.Run("ephemeral_completed when github absent and finalizer completed", func(t *testing.T) {
		repoState := repo
		repoState.Ephemeral.FinalizerStatus = "completed"
		observed := ObservedRunner{Repo: repoState.Repo.FullName, StatePresent: true, State: &repoState, GitHub: GitHubFact{Found: false}, SSH: SSHFact{Reachable: true, HostKey: "matched"}, Service: ServiceFact{Service: repoState.Machine.ServiceName, ActiveState: "inactive"}, Labels: CompareLabels(repoState.Runner.Labels, nil)}
		got := Classify(observed)
		if got.Summary != "Ephemeral runner completed one job and needs cleanup." {
			t.Fatalf("unexpected summary for completed: %q", got.Summary)
		}
		matched := false
		for _, r := range got.Reasons {
			if r.ID == ReasonEphemeralCompleted {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("expected ephemeral_completed reason: %#v", got.Reasons)
		}
	})

	t.Run("ephemeral_ttl_expired when expiry passed and not completed", func(t *testing.T) {
		repoState := repo
		expired := now.Add(-time.Hour)
		repoState.Ephemeral.ExpiresAt = &expired
		observed := ObservedRunner{Repo: repoState.Repo.FullName, StatePresent: true, State: &repoState, GitHub: GitHubFact{Found: true, ID: 222, Status: "online", Labels: repoState.Runner.Labels}, SSH: SSHFact{Reachable: true, HostKey: "matched"}, Service: ServiceFact{Service: repoState.Machine.ServiceName, ActiveState: "active"}, Labels: CompareLabels(repoState.Runner.Labels, repoState.Runner.Labels), Ephemeral: EphemeralFact{Mode: "ephemeral", State: "active", FinalizerStatus: "pending", ExpiresAt: expired.Format(time.RFC3339)}}
		got := Classify(observed)
		if got.Summary != "Ephemeral runner TTL expired before a job completed. Run cleanup now." {
			t.Fatalf("unexpected summary for ttl_expired: %q", got.Summary)
		}
		matched := false
		for _, r := range got.Reasons {
			if r.ID == ReasonEphemeralTTLExpired {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("expected ephemeral_ttl_expired reason: %#v", got.Reasons)
		}
	})

	t.Run("ephemeral_busy when github reports busy", func(t *testing.T) {
		repoState := repo
		observed := ObservedRunner{Repo: repoState.Repo.FullName, StatePresent: true, State: &repoState, GitHub: GitHubFact{Found: true, Busy: true, Status: "online", Labels: repoState.Runner.Labels}, SSH: SSHFact{Reachable: true, HostKey: "matched"}, Service: ServiceFact{Service: repoState.Machine.ServiceName, ActiveState: "active"}, Labels: CompareLabels(repoState.Runner.Labels, repoState.Runner.Labels)}
		got := Classify(observed)
		if got.Summary != "Ephemeral runner is running its one allowed job." {
			t.Fatalf("unexpected summary for busy: %q", got.Summary)
		}
	})

	t.Run("ephemeral_waiting when github online and not busy", func(t *testing.T) {
		repoState := repo
		observed := ObservedRunner{Repo: repoState.Repo.FullName, StatePresent: true, State: &repoState, GitHub: GitHubFact{Found: true, Busy: false, Status: "online", Labels: repoState.Runner.Labels}, SSH: SSHFact{Reachable: true, HostKey: "matched"}, Service: ServiceFact{Service: repoState.Machine.ServiceName, ActiveState: "active"}, Labels: CompareLabels(repoState.Runner.Labels, repoState.Runner.Labels)}
		got := Classify(observed)
		if got.Summary != "Ephemeral runner is online and waiting for its one job." {
			t.Fatalf("unexpected summary for waiting: %q", got.Summary)
		}
	})

	t.Run("ephemeral_cleanup_pending when operations pending", func(t *testing.T) {
		repoState := repo
		repoState.Ephemeral.FinalizerStatus = "completed"
		repoState.Operations = []state.OperationCheckpoint{{Command: "down", Artifact: "ephemeral_log_preservation", Status: "pending", Message: "ephemeral_log_preservation_pending"}}
		observed := ObservedRunner{Repo: repoState.Repo.FullName, StatePresent: true, State: &repoState, GitHub: GitHubFact{Found: false}, SSH: SSHFact{Reachable: true, HostKey: "matched"}, Service: ServiceFact{Service: repoState.Machine.ServiceName, ActiveState: "inactive"}, Labels: CompareLabels(repoState.Runner.Labels, nil)}
		got := Classify(observed)
		if got.Summary != "Ephemeral cleanup is incomplete and pending checkpoints remain." {
			t.Fatalf("unexpected summary for cleanup_pending: %q", got.Summary)
		}
	})
}

func TestCompareLabelsReportsPersistentMissingAndGPUExtra(t *testing.T) {
	fact := CompareLabels([]string{"self-hosted", "runnerkit", "persistent"}, []string{"self-hosted", "runnerkit", "gpu"})
	if fact.Match || len(fact.Missing) != 1 || fact.Missing[0] != "persistent" || len(fact.Extra) != 1 || fact.Extra[0] != "gpu" {
		t.Fatalf("unexpected label drift: %#v", fact)
	}
}
