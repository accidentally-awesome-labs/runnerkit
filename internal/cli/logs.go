package cli

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/salar/runnerkit/internal/ops"
	"github.com/salar/runnerkit/internal/redact"
	rkstate "github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

type logsOptions struct {
	repo    string
	since   string
	lines   int
	service string
	runner  string
}

func newLogsCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &logsOptions{since: "1h", lines: 200}
	cmd := &cobra.Command{Use: "logs"}
	cmd.Short = "Collect bounded runner logs"
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runLogs(deps, *jsonOutput, *noColor, opts)
	}
	cmd.Flags().StringVar(&opts.repo, "repo", "", "target GitHub repository as owner/name")
	cmd.Flags().StringVar(&opts.since, "since", "1h", "journal time window")
	cmd.Flags().IntVar(&opts.lines, "lines", 200, "maximum lines per log source")
	cmd.Flags().StringVar(&opts.service, "service", "", "override service name")
	cmd.Flags().StringVar(&opts.runner, "runner", "", "override runner name")
	return cmd
}

func runLogs(deps Dependencies, jsonOutput bool, noColor bool, opts *logsOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()
	repo, err := resolveReadOnlyRepo(ctx, deps, renderer, opts.repo, "Pass --repo owner/name or run runnerkit logs from a GitHub repository.")
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
	if opts.service != "" {
		repoState.Machine.ServiceName = opts.service
	}
	if opts.runner != "" {
		repoState.Runner.Name = opts.runner
	}
	target, err := targetFromState(repoState)
	if err != nil {
		_ = renderer.Error("invalid_state", "RunnerKit can't collect logs because saved SSH target is invalid.", []string{err.Error()})
		return NewExitError(ExitStateIO, err)
	}
	renderer.Redactor().Register(redact.MachineRef, repoState.Machine.HostRef)
	bundle := ops.CollectLogs(ctx, deps.RemoteExecutor, target, repoState, opts.since, opts.lines)
	bundle.StatePath = store.Path()
	if jsonOutput {
		return renderer.JSON(map[string]any{"ok": true, "command": "logs", "repo": repo.FullName, "state_path": store.Path(), "since": bundle.Since, "lines": bundle.Lines, "sections": bundle.Sections, "warnings": bundle.Warnings})
	}
	return renderLogsHuman(renderer, repoState, bundle)
}

func renderLogsHuman(renderer *ui.Renderer, repoState rkstate.RepositoryState, bundle ops.LogBundle) error {
	lines := []ui.Line{
		ui.Bullet("collection summary"),
		ui.Bullet("Repository: " + repoState.Repo.FullName),
		ui.Bullet("Since: " + bundle.Since),
		ui.Bullet("Lines: " + strconv.Itoa(bundle.Lines)),
	}
	for _, section := range bundle.Sections {
		lines = append(lines, ui.Bullet(section.Title))
		if section.Metadata != "" {
			lines = append(lines, ui.Bullet("    Metadata: "+section.Metadata))
		}
		content := renderer.Redactor().String(section.Content)
		for _, logLine := range strings.Split(strings.TrimSpace(content), "\n") {
			if strings.TrimSpace(logLine) != "" {
				lines = append(lines, ui.Bullet("    "+logLine))
			}
		}
	}
	lines = append(lines, ui.WarningLine("Review logs before sharing; redaction is best-effort for workflow-produced secrets."))
	if len(bundle.Warnings) > 0 {
		lines = append(lines, ui.Bullet("collection warnings"))
		for _, warning := range bundle.Warnings {
			lines = append(lines, ui.WarningLine(warning))
		}
		lines = append(lines, ui.Next("Next: runnerkit doctor --repo "+repoState.Repo.FullName))
	} else {
		lines = append(lines, ui.Bullet("collection warnings: none"))
		lines = append(lines, ui.Next("Next: runnerkit doctor --repo "+repoState.Repo.FullName))
	}
	return renderer.Step(1, 1, "runner logs", lines...)
}
