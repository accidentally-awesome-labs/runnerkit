package ops

import (
	"context"
	"strings"
	"testing"

	"github.com/salar/runnerkit/internal/remote"
	"github.com/salar/runnerkit/internal/testsupport"
)

func TestCollectLogsRunsBoundedJournalAndDiagCommands(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	exec := &testsupport.RemoteExecutor{Results: map[string]remote.Result{
		CommandLogsSystemdJournal: {Stdout: "journal", ExitCode: 0},
		CommandLogsRunnerDiagList: {Stdout: "/opt/actions-runner/runnerkit-owner-repo-local/_diag/Runner_1.log\n/opt/actions-runner/runnerkit-owner-repo-local/_diag/Worker_1.log\n", ExitCode: 0},
		CommandLogsRunnerDiagTail: {Stdout: "diag", ExitCode: 0},
	}}
	bundle := CollectLogs(context.Background(), exec, target(), repo, "30m", 2000)
	if bundle.Lines != 1000 || len(bundle.Sections) != 2 {
		t.Fatalf("unexpected bundle: %#v", bundle)
	}
	ids := strings.Join(exec.CommandIDs(), ",")
	for _, want := range []string{"logs.systemd.journal", "logs.runner.diag.list", "logs.runner.diag.tail"} {
		if !strings.Contains(ids, want) {
			t.Fatalf("missing command id %q in %s", want, ids)
		}
	}
	if !strings.Contains(exec.Commands[0].Script, "journalctl -u") || !strings.Contains(exec.Commands[1].Script, "_diag") {
		t.Fatalf("unexpected scripts: %#v", exec.Commands)
	}
}
