package ops

import (
	"context"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

func TestProbeRemoteStatusMatchedHostKeyRunsOnlyStatusCommands(t *testing.T) {
	exec := &testsupport.RemoteExecutor{
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
		Results: map[string]remote.Result{
			CommandStatusSSHReachable: {ExitCode: 0},
			CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=active\nSubState=running\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0},
		},
	}
	ssh, service := ProbeRemoteStatus(context.Background(), exec, target(), "SHA256:fakehostfingerprint", testsupport.TestServiceName)
	if !ssh.Reachable || ssh.HostKey != "matched" || service.ActiveState != "active" || service.SubState != "running" {
		t.Fatalf("unexpected facts: %#v %#v", ssh, service)
	}
	ids := strings.Join(exec.CommandIDs(), ",")
	if exec.ProbeHostKeyCalls != 1 || ids != "status.ssh.reachable,status.systemd.show" {
		t.Fatalf("ProbeHostKey/order = %d/%s", exec.ProbeHostKeyCalls, ids)
	}
	if strings.Contains(ids, "host.network.github") || strings.Contains(ids, "host.disk") || strings.Contains(ids, "runner.conflict") {
		t.Fatalf("status probes ran full preflight IDs: %s", ids)
	}
}

func TestProbeRemoteStatusHostKeyMismatchSkipsSSHAndSystemd(t *testing.T) {
	exec := &testsupport.RemoteExecutor{ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:changed"}}
	ssh, service := ProbeRemoteStatus(context.Background(), exec, target(), "SHA256:fakehostfingerprint", testsupport.TestServiceName)
	if ssh.HostKey != "mismatch" || service.Error != "skipped because SSH host key mismatch" {
		t.Fatalf("unexpected mismatch facts: %#v %#v", ssh, service)
	}
	if len(exec.Commands) != 0 {
		t.Fatalf("host-key mismatch should not run commands: %#v", exec.CommandIDs())
	}
}

func target() remote.Target {
	return remote.Target{Host: "example.com", User: "alice", Port: 22, Raw: "alice@example.com:22"}
}

// Bug 19 (Plan 06-10, 2026-05-06): when the saved ServiceName is the
// simplified `actions.runner.<runner-name>.service` form but the actual
// systemd unit name produced by GitHub's svc.sh is
// `actions.runner.<owner-repo>.<runner-name>.service`, ProbeRemoteStatus
// must (a) detect the LoadState=not-found result on the saved name,
// (b) query systemctl list-units to discover the actual unit by
// `<runner-name>.service` suffix, and (c) re-show the discovered unit so
// the returned ServiceFact reports the active service (not WARNING
// inactive).
func TestProbeRemoteStatusFallsBackToListUnitsWhenSavedNameIsNotFound(t *testing.T) {
	savedName := testsupport.TestServiceName // simplified form
	actualName := "actions.runner.owner-repo." + testsupport.TestRunnerName + ".service"
	exec := &testsupport.RemoteExecutor{
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
		Results: map[string]remote.Result{
			CommandStatusSSHReachable: {ExitCode: 0},
			// First show against the simplified name returns LoadState=not-found.
			CommandStatusSystemdShow: {Stdout: "LoadState=not-found\nActiveState=inactive\nSubState=dead\nUnitFileState=\nExecMainStatus=\n", ExitCode: 0},
			// list-units returns the real GitHub-mangled unit name.
			CommandStatusSystemdListUnits: {Stdout: actualName + " loaded active running GitHub Actions Runner\n", ExitCode: 0},
			// Re-show against the resolved name returns active.
			CommandStatusSystemdShowResolved: {Stdout: "LoadState=loaded\nActiveState=active\nSubState=running\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0},
		},
	}
	_, service := ProbeRemoteStatus(context.Background(), exec, target(), "SHA256:fakehostfingerprint", savedName)
	if service.ActiveState != "active" || service.SubState != "running" {
		t.Fatalf("ProbeRemoteStatus did not resolve actual unit; service=%#v", service)
	}
	if service.Service != actualName {
		t.Fatalf("ProbeRemoteStatus must report the resolved unit name; got %q want %q", service.Service, actualName)
	}
	ids := strings.Join(exec.CommandIDs(), ",")
	if !strings.Contains(ids, CommandStatusSystemdListUnits) {
		t.Fatalf("expected list-units fallback in command order: %s", ids)
	}
}

// When the saved ServiceName matches the actual unit (no fallback
// needed), ProbeRemoteStatus must NOT issue list-units — the original
// happy path stays single-show.
func TestProbeRemoteStatusSkipsListUnitsWhenSavedNameMatches(t *testing.T) {
	exec := &testsupport.RemoteExecutor{
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
		Results: map[string]remote.Result{
			CommandStatusSSHReachable: {ExitCode: 0},
			CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=active\nSubState=running\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0},
		},
	}
	_, service := ProbeRemoteStatus(context.Background(), exec, target(), "SHA256:fakehostfingerprint", testsupport.TestServiceName)
	if service.ActiveState != "active" {
		t.Fatalf("happy-path show must remain single-show: %#v", service)
	}
	for _, id := range exec.CommandIDs() {
		if id == CommandStatusSystemdListUnits || id == CommandStatusSystemdShowResolved {
			t.Fatalf("happy-path must not run fallback IDs: %#v", exec.CommandIDs())
		}
	}
}
