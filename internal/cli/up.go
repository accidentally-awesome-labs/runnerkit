package cli

import (
	"context"
	"errors"
	"fmt"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

type upOptions struct {
	repo                string
	yes                 bool
	nonInteractive      bool
	dryRun              bool
	allowPublicRepoRisk bool
}

type GitHubService interface {
	Repository(ctx context.Context, repo gh.Repo) (gh.Repo, error)
	VerifyAuth(ctx context.Context, repo gh.Repo) (gh.PermissionStatus, error)
}

type defaultGitHubService struct{}

func (defaultGitHubService) Repository(_ context.Context, repo gh.Repo) (gh.Repo, error) {
	if repo.Host == "" {
		repo.Host = "github.com"
	}
	return repo, nil
}

func (defaultGitHubService) VerifyAuth(_ context.Context, _ gh.Repo) (gh.PermissionStatus, error) {
	return gh.PermissionStatus{OK: true, Source: gh.AuthSource{Kind: "gh", Reference: "gh"}}, nil
}

func newUpCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &upOptions{}
	cmd := &cobra.Command{Use: "up"}
	cmd.Short = "Prepare foundation"
	cmd.Long = "Prepare RunnerKit CLI, auth, safety, and state foundations. Phase 1 does not install a runner yet."
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runUp(deps, *jsonOutput, *noColor, opts)
	}

	cmd.Flags().StringVar(&opts.repo, "repo", "", "target GitHub repository as owner/name")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "accept safe defaults without interactive confirmation")
	cmd.Flags().BoolVar(&opts.nonInteractive, "non-interactive", false, "fail instead of prompting for missing input")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "preview the foundation plan without saving state")
	cmd.Flags().BoolVar(&opts.allowPublicRepoRisk, "allow-public-repo-risk", false, "explicitly acknowledge public repository persistent-runner risk")

	return cmd
}

func runUp(deps Dependencies, jsonOutput bool, noColor bool, opts *upOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()
	resolution, err := resolveUpRepo(ctx, deps, renderer, opts)
	if err != nil {
		return err
	}

	status, err := deps.GitHub.VerifyAuth(ctx, resolution.Repo)
	if err != nil || !status.OK {
		message := fmt.Sprintf("RunnerKit can't create a repository runner registration token for %s.", resolution.Repo.FullName)
		remediation := status.Remediation
		if len(remediation) == 0 {
			remediation = []string{"Create a fine-grained token scoped only to " + resolution.Repo.FullName + " with repository Administration read/write and Metadata read, then pass it with RUNNERKIT_GITHUB_TOKEN for this command."}
		}
		_ = renderer.Error("github_permission_denied", message, remediation)
		if err == nil {
			err = errors.New(message)
		}
		return NewExitError(ExitGitHubAuth, err)
	}
	if _, err := deps.GitHub.Repository(ctx, resolution.Repo); err != nil {
		message := fmt.Sprintf("RunnerKit can't read repository metadata for %s.", resolution.Repo.FullName)
		_ = renderer.Error("github_permission_denied", message, []string{"Verify GitHub credentials can read repository metadata for " + resolution.Repo.FullName + "."})
		return NewExitError(ExitGitHubAuth, err)
	}
	if jsonOutput {
		return renderer.JSON(upJSONPayload(resolution.Repo.FullName, status.Source, nil))
	}
	return renderUpHuman(renderer, opts, resolution.Repo, status.Source, nil)
}

func resolveUpRepo(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions) (gh.Resolution, error) {
	if opts.repo == "" && (!deps.TTY.StdinTTY || opts.nonInteractive) {
		message := "RunnerKit can't continue because repository scope is required before auth or state actions apply."
		remediation := []string{gh.TargetRemediation(nil)[0]}
		_ = renderer.Error("input_required", message, remediation)
		return gh.Resolution{}, NewExitError(ExitInputRequired, errors.New(message+" Pass --repo owner/name."))
	}
	resolution, err := gh.ResolveTarget(ctx, gh.ResolveOptions{Repo: opts.repo, CommandRunner: deps.CommandRunner})
	if err != nil {
		message := fmt.Sprintf("RunnerKit can't continue because %s.", err.Error())
		remediation := gh.TargetRemediation(err)
		code := ExitInvalidInput
		if opts.repo == "" {
			code = ExitInputRequired
		}
		_ = renderer.Error("invalid_repo", message, remediation)
		return gh.Resolution{}, NewExitError(code, err)
	}
	if resolution.NeedsConfirmation {
		if err := renderer.Step(3, 6, "Choose repository", ui.PromptLine("Choose repository: "+resolution.Repo.FullName)); err != nil {
			return gh.Resolution{}, err
		}
		if deps.Prompts == nil {
			message := "RunnerKit can't continue because repository confirmation requires an interactive prompt."
			_ = renderer.Error("input_required", message, []string{"Pass --repo " + resolution.Repo.FullName + " --yes to confirm the target repository."})
			return gh.Resolution{}, NewExitError(ExitInputRequired, errors.New(message))
		}
		confirmed, err := deps.Prompts.Confirm(ctx, ui.Prompt{Message: "Choose repository: " + resolution.Repo.FullName, Default: true})
		if err != nil {
			return gh.Resolution{}, err
		}
		if !confirmed {
			message := "Canceled; no changes made."
			_ = renderer.Error("canceled", message, nil)
			return gh.Resolution{}, NewExitError(ExitCanceled, errors.New(message))
		}
	}
	return resolution, nil
}

func renderUpHuman(renderer *ui.Renderer, opts *upOptions, repo gh.Repo, source gh.AuthSource, warnings []string) error {
	if source.Kind == "" {
		source.Kind = "gh"
	}
	steps := []struct {
		title string
		lines []ui.Line
	}{
		{
			title: "Welcome",
			lines: []ui.Line{
				ui.Bullet("Prepare RunnerKit CLI, auth, safety, and state foundations for " + repo.FullName + "."),
				ui.Bullet("Phase 1 does not install a runner yet."),
			},
		},
		{
			title: "Prerequisites",
			lines: []ui.Line{
				ui.Bullet("Git repository or --repo owner/name."),
				ui.Bullet("GitHub auth through gh or a fine-grained token path."),
				ui.Bullet("Writable user-local state directory for later phases."),
			},
		},
		{
			title: "Repo/auth - Verify GitHub auth",
			lines: []ui.Line{
				ui.Success("Choose repository: " + repo.FullName),
				ui.Success("Verify GitHub auth: " + source.Kind),
			},
		},
		{
			title: "Safety checks",
			lines: safetyLines(opts, warnings),
		},
		{
			title: "State preview",
			lines: []ui.Line{
				ui.Bullet("Repository scope: " + repo.FullName),
				ui.Bullet("Auth source reference: " + source.Kind + "; no token material is saved."),
				ui.Bullet("State path: ~/.local/state/runnerkit/state.json"),
				ui.Bullet("Planned labels: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]"),
				ui.WarningLine("Do not use runs-on: self-hosted alone for RunnerKit-managed runners."),
				ui.Bullet("Will not install a runner in Phase 1"),
				ui.PromptLine("Save this foundation state? [y/N]"),
			},
		},
		{
			title: "Next steps",
			lines: []ui.Line{
				ui.Next("Plan 01-03 adds versioned state persistence."),
				ui.Bullet("Will not install a runner in Phase 1"),
			},
		},
	}
	for i, step := range steps {
		if err := renderer.Step(i+1, len(steps), step.title, step.lines...); err != nil {
			return err
		}
	}
	return nil
}

func safetyLines(opts *upOptions, warnings []string) []ui.Line {
	if len(warnings) == 0 {
		return []ui.Line{ui.Success("Private/trusted repository safety gate passed."), ui.Bullet("allow-public-repo-risk override: " + boolWord(opts.allowPublicRepoRisk))}
	}
	lines := make([]ui.Line, 0, len(warnings)+1)
	for _, warning := range warnings {
		lines = append(lines, ui.WarningLine(warning))
	}
	lines = append(lines, ui.Bullet("allow-public-repo-risk override: "+boolWord(opts.allowPublicRepoRisk)))
	return lines
}

func upJSONPayload(repo string, source gh.AuthSource, warnings []string) map[string]any {
	if source.Kind == "" {
		source.Kind = "gh"
	}
	if warnings == nil {
		warnings = []string{}
	}
	return map[string]any{
		"ok":               true,
		"command":          "up",
		"repo":             repo,
		"auth_source":      source.Kind,
		"state_path":       "~/.local/state/runnerkit/state.json",
		"runner_installed": false,
		"warnings":         warnings,
		"next_steps": []map[string]string{
			{
				"label":   "Review saved state",
				"command": "runnerkit state show --repo " + repo,
			},
		},
	}
}

func boolWord(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}
