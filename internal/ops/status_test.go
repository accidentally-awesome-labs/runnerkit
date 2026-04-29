package ops

import (
	"testing"

	gh "github.com/salar/runnerkit/internal/github"
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

func TestCompareLabelsReportsPersistentMissingAndGPUExtra(t *testing.T) {
	fact := CompareLabels([]string{"self-hosted", "runnerkit", "persistent"}, []string{"self-hosted", "runnerkit", "gpu"})
	if fact.Match || len(fact.Missing) != 1 || fact.Missing[0] != "persistent" || len(fact.Extra) != 1 || fact.Extra[0] != "gpu" {
		t.Fatalf("unexpected label drift: %#v", fact)
	}
}
