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
	// CommandStatusSystemdListUnits queries systemctl list-units for any
	// `actions.runner.*.service` unit. Used by Bug 19 (Plan 06-10,
	// 2026-05-06) as a fallback when the saved simplified service name
	// resolves to LoadState=not-found — GitHub's svc.sh prefixes the unit
	// with the (mangled) repo segment, so the real on-disk unit name is
	// `actions.runner.<owner-repo>.<runner-name>.service` instead of the
	// simplified `actions.runner.<runner-name>.service` RunnerKit saves
	// during bootstrap. We re-show against the discovered unit so status
	// reports the active service (not WARNING inactive) for healthy
	// runners.
	CommandStatusSystemdListUnits    = "status.systemd.list_units"
	CommandStatusSystemdShowResolved = "status.systemd.show.resolved"
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
	parseSystemdShow(&service, result.Stdout)
	// Bug 19 (Plan 06-10, 2026-05-06): when the saved ServiceName is the
	// simplified `actions.runner.<runner-name>.service` form but GitHub's
	// svc.sh installed the unit as `actions.runner.<owner-repo>.<runner-
	// name>.service`, the first show returns LoadState=not-found. Fall
	// back to `systemctl list-units 'actions.runner.*'` and match the
	// discovered unit by `<runner-name>.service` suffix, then re-show.
	if service.LoadState == "not-found" {
		runnerSuffix := extractRunnerSuffix(serviceName)
		if runnerSuffix != "" {
			listResult, listErr := executor.Run(ctx, target, remote.Command{
				ID:      CommandStatusSystemdListUnits,
				Script:  "systemctl list-units 'actions.runner.*' --type=service --all --no-pager --plain --no-legend",
				Timeout: 5 * time.Second,
			})
			if listErr == nil && listResult.ExitCode == 0 {
				if resolved := matchUnitBySuffix(listResult.Stdout, runnerSuffix); resolved != "" && resolved != serviceName {
					reshow, reshowErr := executor.Run(ctx, target, remote.Command{
						ID:      CommandStatusSystemdShowResolved,
						Script:  "systemctl show " + shellQuote(resolved) + " --property=LoadState,ActiveState,SubState,UnitFileState,ExecMainStatus --no-pager",
						Timeout: 5 * time.Second,
					})
					if reshowErr == nil && reshow.ExitCode == 0 {
						// Reset and reparse so the resolved unit's facts win.
						service = ServiceFact{Service: resolved}
						parseSystemdShow(&service, reshow.Stdout)
					}
				}
			}
		}
	}
	return ssh, service
}

// parseSystemdShow parses the `systemctl show --property=…` key=value
// output into a ServiceFact. Pulled into a helper so Bug 19's resolved
// re-show shares the same parser as the initial show.
func parseSystemdShow(service *ServiceFact, stdout string) {
	for _, line := range strings.Split(stdout, "\n") {
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
}

// extractRunnerSuffix returns the `<runner-name>.service` tail of a
// saved ServiceName. The simplified RunnerKit form is
// `actions.runner.<runner-name>.service`, so we strip the
// `actions.runner.` prefix. If the prefix is absent we return the input
// as-is (best-effort; matchUnitBySuffix will accept a full match).
func extractRunnerSuffix(serviceName string) string {
	const prefix = "actions.runner."
	if strings.HasPrefix(serviceName, prefix) {
		return strings.TrimPrefix(serviceName, prefix)
	}
	return serviceName
}

// matchUnitBySuffix scans `systemctl list-units --plain --no-legend`
// output and returns the first `actions.runner.*.service` unit whose
// name ends with the given suffix (e.g. `runnerkit-owner-repo-local.service`).
// Empty if no match.
func matchUnitBySuffix(stdout string, suffix string) string {
	if suffix == "" {
		return ""
	}
	for _, raw := range strings.Split(stdout, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		// list-units --plain output: UNIT LOAD ACTIVE SUB DESCRIPTION
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		unit := fields[0]
		if !strings.HasPrefix(unit, "actions.runner.") || !strings.HasSuffix(unit, ".service") {
			continue
		}
		if strings.HasSuffix(unit, suffix) {
			return unit
		}
	}
	return ""
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
