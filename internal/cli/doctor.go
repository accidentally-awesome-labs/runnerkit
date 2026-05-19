package cli

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/preflight"
	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/nextaction"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/stage"
	"github.com/spf13/cobra"
)

type doctorOptions struct {
	repo            string
	verbose         bool
	deep            bool
	withLogSnippets bool
	fix             bool
	fixYes          bool
	ignore          []string
}

func newDoctorCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &doctorOptions{}
	cmd := &cobra.Command{Use: "doctor"}
	cmd.Short = "Diagnose RunnerKit-managed runner health"
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runDoctor(deps, *jsonOutput, *noColor, opts)
	}
	cmd.Flags().StringVar(&opts.repo, "repo", "", "target GitHub repository as owner/name")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "show pass findings")
	cmd.Flags().BoolVar(&opts.deep, "deep", false, "collect extra host evidence (e.g. journal OOM hints) even when the runner looks healthy")
	cmd.Flags().BoolVar(&opts.withLogSnippets, "with-log-snippets", false, "with heuristics, include short matching log lines (use when sharing diagnostics)")
	cmd.Flags().BoolVar(&opts.fix, "fix", false, "attempt safe auto-remediation for supported findings")
	cmd.Flags().BoolVar(&opts.fixYes, "yes", false, "with --fix, skip confirmation prompts (use only in trusted automation)")
	cmd.Flags().StringSliceVar(&opts.ignore, "ignore", nil, "persistently ignore a doctor finding id (repeatable flag)")
	return cmd
}

// doctorJSONError emits a doctor --json error envelope that satisfies the JSON
// contract documented in CLAUDE.md (schema_version, stage, next_actions[],
// host_incident_hints[] always present as arrays, never null). Falls back to
// renderer.Error for human output.
//
// Bug 33-C (smoke-discovery 2026-05-18): `runnerkit doctor --json` with no
// state (the most common error path during agent / MCP exploratory probing)
// previously returned a stripped envelope missing all 4 contract fields,
// breaking downstream JSON consumers without the smoke harness catching it.
func doctorJSONError(renderer *ui.Renderer, jsonOutput bool, st stage.Stage, code, message string, remediation []string) error {
	if !jsonOutput {
		return renderer.Error(code, message, remediation)
	}
	base := map[string]any{
		"ok":                  false,
		"command":             "doctor",
		"error":               map[string]any{"code": code, "message": message, "remediation": remediation},
		"next_actions":        []map[string]any{},
		"host_incident_hints": []ops.HostIncidentHint{},
	}
	nextaction.ApplySchemaAndStage(base, string(st))
	return renderer.JSON(base)
}

func runDoctor(deps Dependencies, jsonOutput bool, noColor bool, opts *doctorOptions) error {
	defer maybeShowUpdateNotice(deps, jsonOutput)
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()
	repo, err := resolveReadOnlyRepo(ctx, deps, renderer, opts.repo, "Pass --repo owner/name or run runnerkit doctor from a GitHub repository.")
	if err != nil {
		return err
	}
	store := rkstate.NewStore(deps.StateBaseDir)
	repoState, ok, err := store.GetRepository(repo.FullName)
	if err != nil {
		_ = doctorJSONError(renderer, jsonOutput, stage.NoLocalState, "state_io_failed", "RunnerKit can't read saved runner state.", []string{"Check permissions for " + store.Path() + "."})
		return NewExitError(ExitStateIO, err)
	}
	if !ok {
		message := "No RunnerKit-managed runner found for " + repo.FullName + "."
		_ = doctorJSONError(renderer, jsonOutput, stage.NoLocalState, "state_not_found", message, []string{"Run runnerkit up --repo " + repo.FullName + " --host user@host first."})
		return NewExitError(ExitStateIO, errors.New(message))
	}
	renderer.Redactor().Register(redact.MachineRef, repoState.Machine.HostRef)
	status := collectStatus(ctx, deps, store.Path(), repoState, true)
	checks := collectDoctorChecks(ctx, deps, repoState)
	hints := collectDoctorHostHints(ctx, deps, repoState, status.Observed, "48h", opts.deep, opts.withLogSnippets)
	for i := range hints {
		for j := range hints[i].Snippets {
			hints[i].Snippets[j] = renderer.Redactor().String(hints[i].Snippets[j])
		}
	}
	report := ops.BuildDoctorReport(repoState, status.Observed, checks, hints)
	appendSharedHostDoctorFinding(&report, store, repoState)
	st := stage.InferFromDoctor(status.Observed, report.Health, checks)

	if opts.fix && jsonOutput {
		_ = doctorJSONError(renderer, jsonOutput, st, "doctor_fix_json", "doctor --fix cannot be combined with --json (re-run without --json to apply fixes).", nil)
		return NewExitError(ExitInvalidInput, errors.New("doctor fix with json"))
	}

	cfg, err := LoadUserConfig(deps.StateBaseDir)
	if err != nil {
		_ = doctorJSONError(renderer, jsonOutput, st, "user_config_io", "RunnerKit can't read config.json.", []string{err.Error()})
		return NewExitError(ExitStateIO, err)
	}
	if len(opts.ignore) > 0 {
		cfg.DoctorIgnoreFindingIDs = mergeUniqueStrings(cfg.DoctorIgnoreFindingIDs, opts.ignore)
		if err := SaveUserConfig(deps.StateBaseDir, cfg); err != nil {
			_ = doctorJSONError(renderer, jsonOutput, st, "user_config_io", "RunnerKit can't write config.json.", []string{err.Error()})
			return NewExitError(ExitStateIO, err)
		}
	}
	ignoreMap := doctorIgnoreSet(cfg.DoctorIgnoreFindingIDs)
	display := report
	display.Findings = filterDoctorFindings(report.Findings, ignoreMap)

	if opts.fix {
		if !opts.fixYes && deps.Prompts == nil {
			_ = doctorJSONError(renderer, jsonOutput, st, "doctor_fix_requires_prompts", "doctor --fix needs an interactive terminal or pass --yes.", nil)
			return NewExitError(ExitInputRequired, errors.New("doctor fix prompts"))
		}
		if err := applyDoctorFixes(ctx, deps, renderer, repo, report, ignoreMap, opts.fixYes, noColor); err != nil {
			return NewExitError(ExitSafetyGate, err)
		}
	}

	if jsonOutput {
		// Nil slices in map[string]any marshal as JSON null; tooling expects arrays.
		hostHints := display.HostIncidentHints
		if hostHints == nil {
			hostHints = []ops.HostIncidentHint{}
		}
		next := display.NextActions
		if next == nil {
			next = []ops.NextAction{}
		}
		base := map[string]any{"ok": true, "command": "doctor", "repo": repo.FullName, "state_path": store.Path(), "health": display.Health, "findings": display.Findings, "next_actions": next, "host_incident_hints": hostHints}
		nextaction.ApplySchemaAndStage(base, string(st))
		return renderer.JSON(base)
	}
	return renderDoctorHuman(renderer, display, opts.verbose, st)
}

func collectDoctorHostHints(ctx context.Context, deps Dependencies, repoState rkstate.RepositoryState, observed ops.ObservedRunner, since string, deep, withSnippets bool) []ops.HostIncidentHint {
	if !ops.ShouldCollectHostIncidentJournals(observed, deep) {
		return nil
	}
	target, err := targetFromState(repoState)
	if err != nil {
		return nil
	}
	runnerTxt, kernelTxt, _ := ops.CollectBoundedJournalsForHints(ctx, deps.RemoteExecutor, target, repoState, since, 400, 120)
	maxSnip := 0
	if withSnippets {
		maxSnip = 5
	}
	return ops.AnalyzeJournalForOOMHints(runnerTxt, kernelTxt, maxSnip)
}

func collectDoctorChecks(ctx context.Context, deps Dependencies, repoState rkstate.RepositoryState) ops.DeepChecks {
	target, err := targetFromState(repoState)
	if err != nil {
		return ops.DeepChecks{InstallPathError: err.Error(), WorkDirError: err.Error()}
	}
	installScript := "test -f " + shellQuote(repoState.Machine.InstallPath+"/config.sh") + " && test -f " + shellQuote(repoState.Machine.InstallPath+"/run.sh") + " && test -f " + shellQuote(repoState.Machine.InstallPath+"/.runner")
	workScript := "test -d " + shellQuote(repoState.Machine.WorkDir)
	installResult, installErr := deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "doctor.path.install", Script: installScript, Timeout: 10 * time.Second})
	workResult, workErr := deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "doctor.path.work", Script: workScript, Timeout: 10 * time.Second})
	_, _ = deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "doctor.preflight", Script: "true", Timeout: 5 * time.Second})
	report, _ := preflight.Run(ctx, deps.RemoteExecutor, target, preflight.Options{RunnerName: repoState.Runner.Name, AllowUnknownLinux: true})
	// Probe whether one-time host install applied — presence of /etc/sudoers.d/runnerkit-installer
	// is enough to emit the informational `byo_host_prepared` finding.
	byoResult, byoErr := deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "doctor.byo_host_prepared", Script: "test -f " + bootstrap.SudoersFilePath, Timeout: 5 * time.Second})
	checks := ops.DeepChecks{InstallPathOK: installErr == nil && installResult.ExitCode == 0, WorkDirOK: workErr == nil && workResult.ExitCode == 0, Preflight: report, BYOHostPrepared: byoErr == nil && byoResult.ExitCode == 0}
	if !checks.InstallPathOK {
		checks.InstallPathError = strings.TrimSpace(installResult.Stderr + " " + installResult.Stdout)
		if checks.InstallPathError == "" && installErr != nil {
			checks.InstallPathError = installErr.Error()
		}
	}
	if !checks.WorkDirOK {
		checks.WorkDirError = strings.TrimSpace(workResult.Stderr + " " + workResult.Stdout)
		if checks.WorkDirError == "" && workErr != nil {
			checks.WorkDirError = workErr.Error()
		}
	}
	return checks
}

func renderDoctorHuman(renderer *ui.Renderer, report ops.DoctorReport, verbose bool, st stage.Stage) error {
	lines := []ui.Line{ui.Bullet("STAGE: " + string(st)), ui.Bullet("Health: " + string(report.Health.State) + " — " + report.Health.Summary)}
	for _, finding := range report.Findings {
		if finding.Severity == string(ops.SeverityPass) && !verbose {
			continue
		}
		line := ui.WarningLine(finding.ID + " (" + finding.Severity + ")")
		if finding.Severity == string(ops.SeverityError) {
			line = ui.ErrorLine(finding.ID + " (" + finding.Severity + ")")
		} else if finding.Severity == string(ops.SeverityPass) {
			line = ui.Success(finding.ID + " (" + finding.Severity + ")")
		}
		lines = append(lines, line, ui.Bullet("    Evidence: "+finding.Evidence), ui.Bullet("    Remediation: "+finding.Remediation))
	}
	if len(report.HostIncidentHints) > 0 {
		lines = append(lines, ui.WarningLine("Host incident hints (heuristic; not a definitive diagnosis)"))
		for _, h := range report.HostIncidentHints {
			lines = append(lines, ui.Bullet(h.ID+": "+h.Summary))
			for _, s := range h.Snippets {
				lines = append(lines, ui.Bullet("    "+s))
			}
		}
	}
	if len(report.NextActions) > 0 {
		lines = append(lines, ui.Next("Next: "+report.NextActions[0].Command))
	}
	return renderer.Step(1, 1, "doctor", lines...)
}
