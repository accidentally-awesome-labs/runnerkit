package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/labels"
	rkstate "github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/ui"
	"github.com/salar/runnerkit/internal/workflow"
	"github.com/spf13/cobra"
)

type upOptions struct {
	repo                string
	yes                 bool
	nonInteractive      bool
	dryRun              bool
	allowPublicRepoRisk bool
	replace             bool
}

type GitHubService interface {
	Repository(ctx context.Context, repo gh.Repo) (gh.Repo, error)
	VerifyAuth(ctx context.Context, repo gh.Repo) (gh.PermissionStatus, error)
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
	cmd.Flags().BoolVar(&opts.replace, "replace", false, "replace existing saved foundation state for --repo when used with --yes")

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
	repo, err := deps.GitHub.Repository(ctx, resolution.Repo)
	if err != nil {
		message := fmt.Sprintf("RunnerKit can't read repository metadata for %s.", resolution.Repo.FullName)
		_ = renderer.Error("github_permission_denied", message, []string{"Verify GitHub credentials can read repository metadata for " + resolution.Repo.FullName + "."})
		return NewExitError(ExitGitHubAuth, err)
	}
	decision := gh.EvaluateSafety(repo, gh.SafetyOptions{AllowPublicRepoRisk: opts.allowPublicRepoRisk})
	if err := enforceSafetyDecision(ctx, deps, renderer, repo, decision, opts, jsonOutput); err != nil {
		return err
	}

	labelSet := labels.Build(repo, labels.Options{})
	foundationPlan := workflow.FoundationUpPlan()
	store := rkstate.NewStore(deps.StateBaseDir)

	if opts.dryRun {
		if jsonOutput {
			return renderer.JSON(upJSONPayload(repo.FullName, status.Source, decision.Warnings, store.Path(), labelSet, foundationPlan, false))
		}
		return renderUpHuman(renderer, opts, repo, status.Source, decision.Warnings, store.Path(), labelSet, foundationPlan, false)
	}

	if err := confirmStateSave(ctx, deps, renderer, opts, jsonOutput); err != nil {
		return err
	}
	replace := opts.replace
	if _, exists, err := store.GetRepository(repo.FullName); err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't read saved foundation state.", []string{"Check permissions for " + store.Path() + " and re-run runnerkit up."})
		return NewExitError(ExitStateIO, err)
	} else if exists && !replace {
		confirmedReplace, err := confirmStateReplace(ctx, deps, renderer, opts, repo.FullName, jsonOutput)
		if err != nil {
			return err
		}
		replace = confirmedReplace
	}
	repoState := buildFoundationRepositoryState(deps, repo, status.Source, decision, labelSet)
	if err := store.SaveRepository(repoState, replace); err != nil {
		if errors.Is(err, rkstate.ErrRepositoryExists) {
			return replacementRequired(renderer, repo.FullName)
		}
		_ = renderer.Error("state_io_failed", "RunnerKit can't save foundation state.", []string{"Check permissions for " + store.Path() + " and re-run runnerkit up."})
		return NewExitError(ExitStateIO, err)
	}

	if jsonOutput {
		return renderer.JSON(upJSONPayload(repo.FullName, status.Source, decision.Warnings, store.Path(), labelSet, foundationPlan, true))
	}
	return renderUpHuman(renderer, opts, repo, status.Source, decision.Warnings, store.Path(), labelSet, foundationPlan, true)
}

func enforceSafetyDecision(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo, decision gh.SafetyDecision, opts *upOptions, jsonOutput bool) error {
	if decision.Allowed {
		if decision.Code == gh.SafetyCodePublicRisk && opts.allowPublicRepoRisk && deps.TTY.StdinTTY && !opts.yes {
			inputPrompter, ok := deps.Prompts.(interface {
				Input(context.Context, ui.Prompt) (string, error)
			})
			if !ok {
				message := "RunnerKit can't continue because public repository risk acknowledgement requires typed confirmation."
				_ = renderer.Error("input_required", message, []string{"Type allow public repo risk for " + repo.FullName + " in an interactive terminal or pass --yes only after reviewing the risk."})
				return NewExitError(ExitInputRequired, errors.New(message))
			}
			want := "allow public repo risk for " + repo.FullName
			got, err := inputPrompter.Input(ctx, ui.Prompt{Message: want})
			if err != nil {
				return err
			}
			if got != want {
				message := "Canceled; no changes made."
				_ = renderer.Error("canceled", message, nil)
				return NewExitError(ExitCanceled, errors.New(message))
			}
		}
		return nil
	}
	if jsonOutput {
		_ = renderer.Error(decision.Code, gh.PublicRepoRiskTitle, append(decision.Warnings, gh.PublicRepoRiskNextAction))
	} else if decision.Code == gh.SafetyCodePublicRisk {
		// WARNING: Public repository risk
		_ = renderer.Warning(gh.PublicRepoRiskTitle, []string{gh.PublicRepoRiskBody}, gh.PublicRepoRiskNextAction)
	} else {
		_ = renderer.Warning("WARNING: Fork repository risk", decision.Warnings, "Use a trusted private repository before persistent setup.")
	}
	return NewExitError(ExitSafetyGate, errors.New(decision.Code))
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

func confirmStateSave(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, jsonOutput bool) error {
	if opts.yes {
		return nil
	}
	if jsonOutput || opts.nonInteractive || !deps.TTY.StdinTTY {
		message := "RunnerKit can't continue because saving foundation state requires confirmation."
		_ = renderer.Error("input_required", message, []string{"Pass --yes to save state non-interactively, or use --dry-run to preview without writing."})
		return NewExitError(ExitInputRequired, errors.New(message))
	}
	if deps.Prompts == nil {
		message := "RunnerKit can't continue because saving foundation state requires an interactive prompt."
		_ = renderer.Error("input_required", message, []string{"Pass --yes to save state non-interactively, or use --dry-run to preview without writing."})
		return NewExitError(ExitInputRequired, errors.New(message))
	}
	confirmed, err := deps.Prompts.Confirm(ctx, ui.Prompt{Message: "Save this foundation state?", Default: false})
	if err != nil {
		return err
	}
	if !confirmed {
		message := "Canceled; no changes made."
		_ = renderer.Error("canceled", message, nil)
		return NewExitError(ExitCanceled, errors.New(message))
	}
	return nil
}

func confirmStateReplace(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, fullName string, jsonOutput bool) (bool, error) {
	if jsonOutput || opts.yes || opts.nonInteractive || !deps.TTY.StdinTTY {
		return false, replacementRequired(renderer, fullName)
	}
	inputPrompter, ok := deps.Prompts.(interface {
		Input(context.Context, ui.Prompt) (string, error)
	})
	if !ok {
		return false, replacementRequired(renderer, fullName)
	}
	want := "replace " + fullName
	got, err := inputPrompter.Input(ctx, ui.Prompt{Message: "Type " + want + " to overwrite the existing RunnerKit state for this repository."})
	if err != nil {
		return false, err
	}
	if got != want {
		message := "Canceled; no changes made."
		_ = renderer.Error("canceled", message, nil)
		return false, NewExitError(ExitCanceled, errors.New(message))
	}
	return true, nil
}

func replacementRequired(renderer *ui.Renderer, fullName string) error {
	message := "RunnerKit can't continue because saved foundation state already exists for " + fullName + "."
	_ = renderer.Error("input_required", message, []string{"Type replace " + fullName + " in interactive mode, or re-run with --yes --replace after reviewing the existing state."})
	return NewExitError(ExitInputRequired, errors.New(message))
}

func renderUpHuman(renderer *ui.Renderer, opts *upOptions, repo gh.Repo, source gh.AuthSource, warnings []string, statePath string, labelSet labels.LabelSet, plan workflow.Plan, saved bool) error {
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
			lines: statePreviewLines(repo, source, statePath, labelSet, plan, saved),
		},
		{
			title: "Next steps",
			lines: nextStepLines(repo.FullName, statePath, saved),
		},
	}
	for i, step := range steps {
		if err := renderer.Step(i+1, len(steps), step.title, step.lines...); err != nil {
			return err
		}
	}
	return nil
}

func statePreviewLines(repo gh.Repo, source gh.AuthSource, statePath string, labelSet labels.LabelSet, plan workflow.Plan, saved bool) []ui.Line {
	lines := []ui.Line{
		ui.Bullet("Repository scope: " + repo.FullName),
		ui.Bullet("Auth source reference: " + defaultString(source.Kind, "gh") + "; no token material is saved."),
		ui.Bullet("State path: " + statePath),
		ui.Bullet("Project config path: " + rkstate.ProjectConfigRelativePath),
		ui.Bullet("Planned runner name: " + labelSet.RunnerName),
		ui.Bullet("Planned labels: [" + strings.Join(labelSet.Labels, ", ") + "]"),
		ui.Bullet(labelSet.RunsOnYAML),
		ui.WarningLine(labelSet.Warning),
		ui.Bullet("Safety status: " + defaultString(repoSafetyStatus(repo), "ok")),
		ui.Bullet("Will not install a runner in Phase 1"),
	}
	for _, item := range plan.Checklist() {
		lines = append(lines, ui.Bullet("Plan: "+item))
	}
	if saved {
		lines = append(lines, ui.Success("Foundation state saved."))
	} else {
		lines = append(lines, ui.Bullet("Dry run: no state file was written."))
	}
	return lines
}

func nextStepLines(fullName string, statePath string, saved bool) []ui.Line {
	if saved {
		return []ui.Line{
			ui.Next("Review saved state: runnerkit state show --repo " + fullName),
			ui.Bullet("State path: " + statePath),
			ui.Bullet("Will not install a runner in Phase 1"),
		}
	}
	return []ui.Line{
		ui.Next("Re-run without --dry-run and with --yes to save foundation state."),
		ui.Bullet("Will not install a runner in Phase 1"),
	}
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

func upJSONPayload(repo string, source gh.AuthSource, warnings []string, statePath string, labelSet labels.LabelSet, plan workflow.Plan, saved bool) map[string]any {
	if source.Kind == "" {
		source.Kind = "gh"
	}
	if warnings == nil {
		warnings = []string{}
	}
	return map[string]any{
		"ok":                  true,
		"command":             "up",
		"repo":                repo,
		"auth_source":         source.Kind,
		"state_path":          statePath,
		"project_config_path": rkstate.ProjectConfigRelativePath,
		"runner_installed":    false,
		"state_saved":         saved,
		"runner_name":         labelSet.RunnerName,
		"labels":              labelSet.Labels,
		"workflow_snippet":    labelSet.RunsOnYAML,
		"warnings":            warnings,
		"plan":                plan,
		"next_steps": []map[string]string{
			{
				"label":   "Review saved state",
				"command": "runnerkit state show --repo " + repo,
			},
		},
	}
}

func buildFoundationRepositoryState(deps Dependencies, repo gh.Repo, source gh.AuthSource, decision gh.SafetyDecision, labelSet labels.LabelSet) rkstate.RepositoryState {
	now := deps.Clock()
	if now.IsZero() {
		now = time.Now()
	}
	safety := rkstate.SafetyMetadata{Code: decision.Code, Allowed: decision.Allowed, Warnings: decision.Warnings}
	if decision.Code == gh.SafetyCodePublicRisk && decision.Allowed && len(decision.Warnings) > 0 {
		safety.AcceptedOverride = gh.AllowPublicRepoRiskFlag
		safety.AcceptedAt = &now
	}
	return rkstate.RepositoryState{
		Repo: repo,
		Auth: rkstate.AuthReference{Source: defaultString(source.Kind, "gh"), Reference: defaultString(source.Reference, source.Kind)},
		Runner: rkstate.RunnerIdentity{
			Name:            labelSet.RunnerName,
			Labels:          labelSet.Labels,
			WorkflowSnippet: labelSet.RunsOnYAML,
			Mode:            labels.DefaultMode,
			OS:              labels.DefaultOS,
			Arch:            labels.DefaultArch,
		},
		Machine:          rkstate.MachineRef{Kind: "phase1-placeholder"},
		Provider:         rkstate.ProviderRef{Kind: "none", IDs: map[string]string{}},
		Cleanup:          rkstate.CleanupMetadata{ManagedPaths: []string{}, ProviderResourceIDs: []string{}},
		Safety:           safety,
		RunnerKitVersion: deps.Version,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func repoSafetyStatus(repo gh.Repo) string {
	if !repo.Private {
		return gh.SafetyCodePublicRisk
	}
	if repo.Fork {
		return gh.SafetyCodeForkRisk
	}
	return gh.SafetyCodeOK
}

func boolWord(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
