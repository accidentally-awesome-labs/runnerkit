package stage

import (
	"testing"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

func TestInferFromObserved_noState(t *testing.T) {
	t.Parallel()
	o := ops.ObservedRunner{StatePresent: false}
	if g := InferFromObserved(o, ops.Health{}); g != NoLocalState {
		t.Fatalf("got %s want %s", g, NoLocalState)
	}
}

func TestInferFromObserved_running(t *testing.T) {
	t.Parallel()
	repo := rkstate.RepositoryState{Repo: gh.Repo{FullName: "o/r"}}
	st := &repo
	o := ops.ObservedRunner{
		StatePresent: true,
		State:        st,
		SSH:          ops.SSHFact{Reachable: true},
		GitHub:       ops.GitHubFact{Found: true, Status: "online"},
		Service:      ops.ServiceFact{ActiveState: "active"},
	}
	h := ops.Health{State: ops.HealthReady}
	if g := InferFromObserved(o, h); g != Running {
		t.Fatalf("got %s want %s", g, Running)
	}
}

func TestInferFromDoctor_uninstalled(t *testing.T) {
	t.Parallel()
	repo := rkstate.RepositoryState{Repo: gh.Repo{FullName: "o/r"}}
	st := &repo
	o := ops.ObservedRunner{StatePresent: true, State: st, SSH: ops.SSHFact{Reachable: true}}
	h := ops.Classify(o)
	ch := ops.DeepChecks{InstallPathOK: false, WorkDirOK: false}
	if g := InferFromDoctor(o, h, ch); g != Uninstalled {
		t.Fatalf("got %s want %s", g, Uninstalled)
	}
}

func TestActionsFromOpsNext(t *testing.T) {
	t.Parallel()
	a := ActionsFromOpsNext([]ops.NextAction{{Command: "runnerkit doctor", Why: "Check"}})
	if a[0].Title != "Check" {
		t.Fatalf("title %+v", a[0])
	}
}
