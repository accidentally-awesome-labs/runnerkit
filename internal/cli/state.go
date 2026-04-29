package cli

import (
	"errors"
	"fmt"

	gh "github.com/salar/runnerkit/internal/github"
	rkstate "github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

type stateShowOptions struct {
	repo string
}

func newStateCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	cmd := &cobra.Command{Use: "state"}
	cmd.Short = "Inspect RunnerKit state"
	cmd.AddCommand(newStateShowCommand(deps, jsonOutput, noColor))
	return cmd
}

func newStateShowCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &stateShowOptions{}
	cmd := &cobra.Command{Use: "show"}
	cmd.Short = "Show saved foundation state"
	cmd.Long = "state show displays saved, redacted RunnerKit foundation state for a repository."
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runStateShow(deps, *jsonOutput, *noColor, opts)
	}
	cmd.Flags().StringVar(&opts.repo, "repo", "", "target GitHub repository as owner/name")
	return cmd
}

func runStateShow(deps Dependencies, jsonOutput bool, noColor bool, opts *stateShowOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	if opts.repo == "" {
		message := "RunnerKit can't continue because repository scope is required to show state."
		_ = renderer.Error("invalid_repo", message, []string{"Pass --repo owner/name."})
		return NewExitError(ExitInvalidInput, errors.New(message))
	}
	repo, err := gh.ParseRepo(opts.repo)
	if err != nil {
		message := fmt.Sprintf("RunnerKit can't continue because %s.", err.Error())
		_ = renderer.Error("invalid_repo", message, gh.TargetRemediation(err))
		return NewExitError(ExitInvalidInput, err)
	}
	store := rkstate.NewStore(deps.StateBaseDir)
	repoState, ok, err := store.GetRepository(repo.FullName)
	if err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't read saved foundation state.", []string{"Check permissions for " + store.Path() + "."})
		return NewExitError(ExitStateIO, err)
	}
	if !ok {
		message := "No RunnerKit state found for " + repo.FullName + "."
		_ = renderer.Error("state_not_found", message, []string{"Run runnerkit up --repo " + repo.FullName + " --yes to save foundation state."})
		return NewExitError(ExitStateIO, errors.New(message))
	}
	if jsonOutput {
		return renderer.JSON(map[string]any{
			"ok":         true,
			"command":    "state show",
			"repo":       repo.FullName,
			"state_path": store.Path(),
			"state":      repoState,
		})
	}
	return renderStateShowHuman(renderer, repoState, store.Path())
}

func renderStateShowHuman(renderer *ui.Renderer, repoState rkstate.RepositoryState, statePath string) error {
	return renderer.Step(1, 1, "state show",
		ui.Success("Repository: "+repoState.Repo.FullName),
		ui.Bullet("State path: "+statePath),
		ui.Bullet("Auth source: "+repoState.Auth.Source),
		ui.Bullet("Runner name: "+repoState.Runner.Name),
		ui.Bullet("Labels: "+joinLabels(repoState.Runner.Labels)),
		ui.Bullet(repoState.Runner.WorkflowSnippet),
		ui.WarningLine("Do not use runs-on: self-hosted alone for RunnerKit-managed runners."),
		ui.Bullet("Safety status: "+repoState.Safety.Code),
		ui.Bullet("Will not install a runner in Phase 1"),
	)
}

func joinLabels(labels []string) string {
	if len(labels) == 0 {
		return "[]"
	}
	out := "[" + labels[0]
	for _, label := range labels[1:] {
		out += ", " + label
	}
	return out + "]"
}
