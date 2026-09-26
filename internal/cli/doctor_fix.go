package cli

import (
	"errors"

	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/stage"
)

func filterDoctorFindings(findings []ops.Finding, ignore map[string]bool) []ops.Finding {
	if len(ignore) == 0 {
		return findings
	}
	var out []ops.Finding
	for _, f := range findings {
		if ignore[f.ID] {
			continue
		}
		out = append(out, f)
	}
	return out
}

func doctorIgnoreSet(ids []string) map[string]bool {
	m := map[string]bool{}
	for _, id := range ids {
		if id != "" {
			m[id] = true
		}
	}
	return m
}

// refuseDoctorFix rejects `doctor --fix` before any GitHub, SSH or state
// access (A-06a, P0-3). Its only remediation was re-running upgrade-runner,
// which unregistered healthy runners; both are disabled in this release.
func refuseDoctorFix(renderer *ui.Renderer, jsonOutput bool, repo string) error {
	message := "doctor --fix is disabled in this release (known issue): its only fix re-ran upgrade-runner. " + upgradeRunnerDisabledMessage
	_ = doctorJSONError(renderer, jsonOutput, stage.Unknown, "command_disabled", message, upgradeRunnerRemediation(repo))
	return NewExitError(ExitInvalidInput, errors.New("doctor --fix is disabled in this release"))
}
