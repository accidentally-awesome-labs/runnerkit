package cli

import (
	"context"
	"fmt"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
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

func applyDoctorFixes(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo, report ops.DoctorReport, ignore map[string]bool, fixYes bool, noColor bool) error {
	var sawStale bool
	for _, f := range report.Findings {
		if ignore[f.ID] {
			continue
		}
		if f.ID == "runner_version_stale" && f.Severity != string(ops.SeverityPass) {
			sawStale = true
			break
		}
	}
	if !sawStale {
		return nil
	}
	if !fixYes {
		ok, err := deps.Prompts.Confirm(ctx, ui.Prompt{Message: "Apply fix: re-run runnerkit upgrade-runner for " + repo.FullName + "?", Default: false})
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	_ = renderer.Step(1, 1, "doctor --fix", ui.Bullet("Running upgrade-runner for "+repo.FullName))
	if err := runUpgradeRunner(deps, false, noColor, &upgradeRunnerOptions{repo: repo.FullName, yes: true, force: false}); err != nil {
		return fmt.Errorf("upgrade-runner: %w", err)
	}
	return nil
}
