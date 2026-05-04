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
