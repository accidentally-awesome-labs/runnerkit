// Package stage maps observed runner health to a coarse lifecycle stage for UX and JSON.
package stage

import (
	"fmt"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/nextaction"
)

// Stage is a coarse lifecycle label for the runner + host (agent + human UX).
type Stage string

const (
	NoLocalState Stage = "no_local_state"
	Unknown      Stage = "unknown"
	Error        Stage = "error"
	Uninstalled  Stage = "uninstalled"
	Installed    Stage = "installed"
	Registered   Stage = "registered"
	Running      Stage = "running"
)

// InferFromObserved derives a stage from status/doctor observations without path probes.
func InferFromObserved(observed ops.ObservedRunner, health ops.Health) Stage {
	if !observed.StatePresent || observed.State == nil {
		return NoLocalState
	}
	if !observed.SSH.Reachable || observed.SSH.HostKey == "mismatch" {
		return Error
	}
	switch health.State {
	case ops.HealthReady, ops.HealthBusy:
		return Running
	case ops.HealthNeedsAttention:
		return Registered
	case ops.HealthBroken:
		return Error
	default:
		return Unknown
	}
}

// InferFromDoctor refines the stage using deep install/workdir checks from doctor.
func InferFromDoctor(observed ops.ObservedRunner, health ops.Health, checks ops.DeepChecks) Stage {
	if !observed.StatePresent || observed.State == nil {
		return NoLocalState
	}
	if !checks.InstallPathOK {
		return Uninstalled
	}
	if !checks.WorkDirOK {
		return Installed
	}
	return InferFromObserved(observed, health)
}

// ActionsFromOpsNext converts health/doctor next steps into the versioned nextaction list.
func ActionsFromOpsNext(n []ops.NextAction) []nextaction.Action {
	out := make([]nextaction.Action, 0, len(n))
	for i, a := range n {
		title := strings.TrimSpace(a.Why)
		if title == "" {
			title = "Next step"
		}
		out = append(out, nextaction.Action{
			ID:       fmt.Sprintf("next_%d", i),
			Severity: nextaction.SeverityInfo,
			Title:    title,
			Command:  strings.TrimSpace(a.Command),
			Kind:     "run_local",
		})
	}
	return out
}
