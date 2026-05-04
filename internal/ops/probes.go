package ops

import (
	"context"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

const (
	CommandStatusSSHReachable = "status.ssh.reachable"
	CommandStatusSystemdShow  = "status.systemd.show"
)

func ProbeRemoteStatus(ctx context.Context, executor remote.Executor, target remote.Target, savedFingerprint string, serviceName string) (SSHFact, ServiceFact) {
	if executor == nil {
		executor = remote.UnavailableExecutor{}
	}
	hostKeyState := "not_checked"
	observed := ""
	if strings.TrimSpace(savedFingerprint) != "" {
		prober, ok := executor.(remote.HostKeyProber)
		if !ok {
			return SSHFact{Reachable: false, HostKey: "unknown", Error: "SSH host key verification unavailable"}, ServiceFact{Service: serviceName, Error: "skipped because SSH host key verification unavailable"}
		}
		hostKey, err := prober.ProbeHostKey(ctx, target)
		observed = remote.NormalizeHostKey(hostKey).Fingerprint
		if err != nil || observed == "" {
			return SSHFact{Reachable: false, HostKey: "unknown", ObservedFingerprint: observed, Error: "SSH host key verification unavailable"}, ServiceFact{Service: serviceName, Error: "skipped because SSH host key verification unavailable"}
		}
		if observed != savedFingerprint {
			return SSHFact{Reachable: false, HostKey: "mismatch", ObservedFingerprint: observed, Error: "SSH host key mismatch"}, ServiceFact{Service: serviceName, Error: "skipped because SSH host key mismatch"}
		}
		hostKeyState = "matched"
	}

	sshResult, err := executor.Run(ctx, target, remote.Command{ID: CommandStatusSSHReachable, Script: "true", Timeout: 5 * time.Second})
	if err != nil || sshResult.ExitCode != 0 {
		return SSHFact{Reachable: false, HostKey: hostKeyState, ObservedFingerprint: observed, Error: "SSH unreachable"}, ServiceFact{Service: serviceName, Error: "SSH unreachable"}
	}
	ssh := SSHFact{Reachable: true, HostKey: hostKeyState, ObservedFingerprint: observed}
	service := ServiceFact{Service: serviceName}
	if strings.TrimSpace(serviceName) == "" {
		service.Error = "service name missing"
		return ssh, service
	}
	result, err := executor.Run(ctx, target, remote.Command{ID: CommandStatusSystemdShow, Script: "systemctl show " + shellQuote(serviceName) + " --property=LoadState,ActiveState,SubState,UnitFileState,ExecMainStatus --no-pager", Timeout: 5 * time.Second})
	if err != nil || result.ExitCode != 0 {
		service.Error = "systemd status unavailable"
		return ssh, service
	}
	for _, line := range strings.Split(result.Stdout, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "LoadState":
			service.LoadState = value
		case "ActiveState":
			service.ActiveState = value
		case "SubState":
			service.SubState = value
		case "UnitFileState":
			service.UnitFileState = value
		case "ExecMainStatus":
			service.ExecMainStatus = value
		}
	}
	return ssh, service
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
