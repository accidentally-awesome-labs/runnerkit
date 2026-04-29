package cli

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/salar/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type upOptions struct {
	repo                string
	yes                 bool
	nonInteractive      bool
	dryRun              bool
	allowPublicRepoRisk bool
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
	cmd.Flags().BoolVar(&opts.allowPublicRepoRisk, "allow-public-repo-risk", false, "placeholder override for future public repository persistent-runner risk gates")

	return cmd
}

func runUp(deps Dependencies, jsonOutput bool, noColor bool, opts *upOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	if opts.repo == "" {
		message := "RunnerKit can't continue because repository scope is required before auth or state actions apply."
		remediation := []string{"Pass --repo owner/name and re-run runnerkit up."}
		_ = renderer.Error("input_required", message, remediation)
		return NewExitError(ExitInputRequired, errors.New(message+" Pass --repo owner/name."))
	}
	if !repoPattern.MatchString(opts.repo) {
		message := fmt.Sprintf("RunnerKit can't continue because %q is not a GitHub repository in owner/name form.", opts.repo)
		remediation := []string{"Use --repo owner/name, for example --repo octo-org/octo-repo."}
		_ = renderer.Error("invalid_repo", message, remediation)
		return NewExitError(ExitInvalidInput, errors.New(message))
	}
	if opts.nonInteractive && !opts.dryRun && !opts.yes {
		message := "RunnerKit can't continue because non-interactive mode would require confirmation before saving foundation state."
		remediation := []string{"Pass --yes to accept the save prompt or pass --dry-run to preview without saving."}
		_ = renderer.Error("input_required", message, remediation)
		return NewExitError(ExitInputRequired, errors.New(message))
	}
	if jsonOutput {
		return renderer.JSON(upJSONPayload(opts.repo))
	}
	return renderUpHuman(renderer, opts)
}

func renderUpHuman(renderer *ui.Renderer, opts *upOptions) error {
	steps := []struct {
		title string
		lines []ui.Line
	}{
		{
			title: "Welcome",
			lines: []ui.Line{
				ui.Bullet("Prepare RunnerKit CLI, auth, safety, and state foundations for " + opts.repo + "."),
				ui.Bullet("Phase 1 does not install a runner yet."),
			},
		},
		{
			title: "Prerequisites",
			lines: []ui.Line{
				ui.Bullet("Git repository or --repo owner/name."),
				ui.Bullet("GitHub auth through gh or a future fine-grained token path."),
				ui.Bullet("Writable user-local state directory for later phases."),
			},
		},
		{
			title: "Repo/auth",
			lines: []ui.Line{
				ui.Success("Target repository: " + opts.repo),
				ui.Bullet("Repo detection and GitHub auth use placeholder adapters in Plan 01-01."),
			},
		},
		{
			title: "Safety checks",
			lines: []ui.Line{
				ui.WarningLine("Public repository and fork-risk gate is scaffolded for later persistent setup."),
				ui.Bullet("allow-public-repo-risk override placeholder: " + boolWord(opts.allowPublicRepoRisk)),
			},
		},
		{
			title: "State preview",
			lines: []ui.Line{
				ui.Bullet("Repository scope: " + opts.repo),
				ui.Bullet("Auth source reference: placeholder; no token material is saved."),
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
				ui.Next("Plan 01-02 replaces placeholder repo/auth adapters with least-privilege GitHub checks."),
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

func upJSONPayload(repo string) map[string]any {
	return map[string]any{
		"ok":               true,
		"command":          "up",
		"repo":             repo,
		"auth_source":      "placeholder",
		"state_path":       "~/.local/state/runnerkit/state.json",
		"runner_installed": false,
		"warnings":         []string{},
		"next_steps": []map[string]string{
			{
				"label":   "Continue to GitHub auth foundation",
				"command": "runnerkit up --repo " + repo,
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
