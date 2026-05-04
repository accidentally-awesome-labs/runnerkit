package cli

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/preflight"
	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

type doctorOptions struct {
	repo    string
	verbose bool
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
	return cmd
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
		_ = renderer.Error("state_io_failed", "RunnerKit can't read saved runner state.", []string{"Check permissions for " + store.Path() + "."})
		return NewExitError(ExitStateIO, err)
	}
	if !ok {
		message := "No RunnerKit-managed runner found for " + repo.FullName + "."
		_ = renderer.Error("state_not_found", message, []string{"Run runnerkit up --repo " + repo.FullName + " --host user@host first."})
		return NewExitError(ExitStateIO, errors.New(message))
	}
	renderer.Redactor().Register(redact.MachineRef, repoState.Machine.HostRef)
	status := collectStatus(ctx, deps, store.Path(), repoState, true)
	checks := collectDoctorChecks(ctx, deps, repoState)
	report := ops.BuildDoctorReport(repoState, status.Observed, checks)
	if jsonOutput {
		return renderer.JSON(map[string]any{"ok": true, "command": "doctor", "repo": repo.FullName, "state_path": store.Path(), "health": report.Health, "findings": report.Findings, "next_actions": report.NextActions})
	}
	return renderDoctorHuman(renderer, report, opts.verbose)
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
	checks := ops.DeepChecks{InstallPathOK: installErr == nil && installResult.ExitCode == 0, WorkDirOK: workErr == nil && workResult.ExitCode == 0, Preflight: report}
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

func renderDoctorHuman(renderer *ui.Renderer, report ops.DoctorReport, verbose bool) error {
	lines := []ui.Line{ui.Bullet("Health: " + string(report.Health.State) + " — " + report.Health.Summary)}
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
	if len(report.NextActions) > 0 {
		lines = append(lines, ui.Next("Next: "+report.NextActions[0].Command))
	}
	return renderer.Step(1, 1, "doctor", lines...)
}
