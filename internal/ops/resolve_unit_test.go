package ops

import (
	"context"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

func TestResolveActionsRunnerSystemdUnitUsesSavedNameWhenLoaded(t *testing.T) {
	saved := "actions.runner.runnerkit-owner-repo-local.service"
	exec := &testsupport.RemoteExecutor{
		Results: map[string]remote.Result{
			resolveRunnerUnitShow: {Stdout: "LoadState=loaded\n", ExitCode: 0},
		},
	}
	got := ResolveActionsRunnerSystemdUnit(context.Background(), exec, remote.Target{}, saved)
	if got != saved {
		t.Fatalf("got %q want %q", got, saved)
	}
}

func TestResolveActionsRunnerSystemdUnitFallsBackToListUnits(t *testing.T) {
	saved := "actions.runner.runnerkit-owner-repo-local.service"
	actual := "actions.runner.owner-repo.runnerkit-owner-repo-local.service"
	exec := &testsupport.RemoteExecutor{
		Results: map[string]remote.Result{
			resolveRunnerUnitShow: {Stdout: "LoadState=not-found\n", ExitCode: 0},
			resolveRunnerUnitList: {Stdout: actual + " loaded active running\n", ExitCode: 0},
		},
	}
	got := ResolveActionsRunnerSystemdUnit(context.Background(), exec, remote.Target{}, saved)
	if got != actual {
		t.Fatalf("got %q want %q", got, actual)
	}
	ids := strings.Join(exec.CommandIDs(), ",")
	if !strings.Contains(ids, resolveRunnerUnitShow) || !strings.Contains(ids, resolveRunnerUnitList) {
		t.Fatalf("expected resolve probes in %s", ids)
	}
}
