package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/errcodes"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

type recoverOptions struct {
	repo             string
	yes              bool
	dryRun           bool
	restartService   bool
	reinstallService bool
	reregister       bool
}

type recoveryResult struct {
	Step    string `json:"step"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func newRecoverCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &recoverOptions{}
	cmd := &cobra.Command{Use: "recover"}
	cmd.Short = "Recover a RunnerKit-managed persistent runner"
	cmd.RunE = func(_ *cobra.Command, _ []string) error { return runRecover(deps, *jsonOutput, *noColor, opts) }
	cmd.Flags().StringVar(&opts.repo, "repo", "", "target GitHub repository as owner/name")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "apply the displayed recovery plan")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "preview recovery without mutation")
	cmd.Flags().BoolVar(&opts.restartService, "restart-service", false, "restart the recorded systemd service")
	cmd.Flags().BoolVar(&opts.reinstallService, "reinstall-service", false, "reinstall and start the recorded service")
	cmd.Flags().BoolVar(&opts.reregister, "reregister", false, "re-register the runner with fresh GitHub tokens")
	return cmd
}

func runRecover(deps Dependencies, jsonOutput bool, noColor bool, opts *recoverOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()
	repo, err := resolveReadOnlyRepo(ctx, deps, renderer, opts.repo, "Pass --repo owner/name or run runnerkit recover from a GitHub repository.")
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
		_ = renderer.Error("state_not_found", message, []string{"Run runnerkit up --repo " + repo.FullName + " --host user@host first, or use runnerkit down with explicit stale GitHub runner targeting if cleanup is the goal."})
		return NewExitError(ExitStateIO, errors.New(message))
	}
	status := collectStatus(ctx, deps, store.Path(), repoState, true)
	requested := requestedRecoveryActions(opts)
	plan := ops.BuildRecoveryPlan(repoState, status.Observed, requested, opts.dryRun)
	if opts.dryRun {
		if jsonOutput {
			return renderer.JSON(recoveryPayload(repo.FullName, true, plan, nil, false, repoState.Cleanup.GitHubRunnerID))
		}
		return renderRecoveryPlanHuman(renderer, plan)
	}
	if plan.Blocked {
		if jsonOutput {
			_ = renderer.JSON(recoveryPayload(repo.FullName, false, plan, nil, false, repoState.Cleanup.GitHubRunnerID))
		} else {
			_ = renderRecoveryPlanHuman(renderer, plan)
		}
		return NewExitError(ExitSafetyGate, errors.New(plan.BlockReason))
	}
	if !opts.yes {
		if jsonOutput || !deps.TTY.StdinTTY || deps.Prompts == nil {
			message := "RunnerKit can't continue because applying recovery requires confirmation."
			_ = renderer.Error("input_required", message, []string{"Pass --yes to apply the displayed recovery plan non-interactively, or pass --dry-run to preview only."})
			return NewExitError(ExitInputRequired, errors.New(message))
		}
		confirmed, err := deps.Prompts.Confirm(ctx, ui.Prompt{Message: "Apply recovery plan for " + repo.FullName + "?", Default: false})
		if err != nil {
			return err
		}
		if !confirmed {
			message := "Canceled; no changes made."
			_ = renderer.Error("canceled", message, nil)
			return NewExitError(ExitCanceled, errors.New(message))
		}
	}
	results, updatedState, runnerID, err := applyRecoveryPlan(ctx, deps, renderer, store, repoState, plan)
	if err != nil {
		return err
	}
	if jsonOutput {
		return renderer.JSON(recoveryPayload(repo.FullName, false, plan, results, updatedState, runnerID))
	}
	lines := []ui.Line{ui.Success("Recovery plan applied for " + repo.FullName)}
	for _, result := range results {
		lines = append(lines, ui.Bullet(result.Step+": "+result.Status+" "+result.Message))
	}
	return renderer.Step(1, 1, "recovery complete", lines...)
}

func requestedRecoveryActions(opts *recoverOptions) []ops.RecoveryAction {
	var actions []ops.RecoveryAction
	if opts.restartService {
		actions = append(actions, ops.ActionRestartService)
	}
	if opts.reinstallService {
		actions = append(actions, ops.ActionReinstallService)
	}
	if opts.reregister {
		actions = append(actions, ops.ActionReregisterRunner)
	}
	return actions
}

func renderRecoveryPlanHuman(renderer *ui.Renderer, plan ops.RecoveryPlan) error {
	lines := []ui.Line{}
	if plan.Blocked {
		lines = append(lines, ui.ErrorLine(plan.BlockReason))
	} else {
		for _, step := range plan.Steps {
			lines = append(lines, ui.Bullet(step.Description))
		}
		lines = append(lines, ui.Next("Next: pass --yes to apply this recovery plan or keep --dry-run to preview only."))
	}
	return renderer.Step(1, 1, "recovery plan", lines...)
}

func applyRecoveryPlan(ctx context.Context, deps Dependencies, renderer *ui.Renderer, store rkstate.Store, repoState rkstate.RepositoryState, plan ops.RecoveryPlan) ([]recoveryResult, bool, int64, error) {
	target, err := targetFromState(repoState)
	if err != nil {
		_ = renderer.Error("invalid_state", "RunnerKit can't recover because saved SSH target is invalid.", []string{err.Error()})
		return nil, false, repoState.Cleanup.GitHubRunnerID, NewExitError(ExitStateIO, err)
	}
	resolvedUnit := ops.ResolveActionsRunnerSystemdUnit(ctx, deps.RemoteExecutor, target, repoState.Machine.ServiceName)
	var results []recoveryResult
	stateUpdated := false
	runnerID := repoState.Cleanup.GitHubRunnerID
	for _, step := range plan.Steps {
		switch step.Action {
		case ops.ActionRestartService:
			if err := runRecoveryCommand(ctx, deps.RemoteExecutor, target, remote.Command{ID: "recover.service.restart", Script: "sudo systemctl restart " + shellQuote(resolvedUnit), Timeout: 30 * time.Second}); err != nil {
				return nil, false, runnerID, recoveryCommandError(renderer, err)
			}
			if err := runRecoveryCommand(ctx, deps.RemoteExecutor, target, remote.Command{ID: "recover.service.verify", Script: "systemctl is-active " + shellQuote(resolvedUnit), Timeout: 15 * time.Second}); err != nil {
				return nil, false, runnerID, recoveryCommandError(renderer, err)
			}
			if runner, ok, err := waitForRunnerOnline(ctx, deps, repoState.Repo, repoState.Runner.Name, repoState.Runner.Labels); err != nil {
				return nil, false, runnerID, err
			} else if !ok {
				_ = renderer.Error("runner_online_timeout", "RunnerKit could not verify the GitHub runner came online with the expected labels.", []string{"Run runnerkit logs --repo " + repoState.Repo.FullName + " --since 30m."})
				return nil, false, runnerID, NewExitError(ExitSafetyGate, errors.New("runner_online_timeout"))
			} else {
				runnerID = runner.ID
			}
			results = append(results, recoveryResult{Step: string(step.Action), Status: "done"})
		case ops.ActionReinstallService:
			script := "cd " + shellQuote(repoState.Machine.InstallPath) + " && sudo ./svc.sh install runnerkit-runner && sudo ./svc.sh start && sudo ./svc.sh status"
			if err := runRecoveryCommand(ctx, deps.RemoteExecutor, target, remote.Command{ID: "recover.service.reinstall", Script: script, Timeout: 60 * time.Second}); err != nil {
				return nil, false, runnerID, recoveryCommandError(renderer, err)
			}
			if err := runRecoveryCommand(ctx, deps.RemoteExecutor, target, remote.Command{ID: "recover.service.verify", Script: "systemctl is-active " + shellQuote(resolvedUnit), Timeout: 15 * time.Second}); err != nil {
				return nil, false, runnerID, recoveryCommandError(renderer, err)
			}
			if runner, ok, err := waitForRunnerOnline(ctx, deps, repoState.Repo, repoState.Runner.Name, repoState.Runner.Labels); err != nil {
				return nil, false, runnerID, err
			} else if !ok {
				return nil, false, runnerID, NewExitError(ExitSafetyGate, errors.New("runner_online_timeout"))
			} else {
				runnerID = runner.ID
			}
			results = append(results, recoveryResult{Step: string(step.Action), Status: "done"})
		case ops.ActionReregisterRunner:
			updated, id, reregisterResults, err := applyReregister(ctx, deps, renderer, store, target, repoState)
			results = append(results, reregisterResults...)
			if err != nil {
				return nil, false, runnerID, err
			}
			stateUpdated = stateUpdated || updated
			runnerID = id
		}
	}
	return results, stateUpdated, runnerID, nil
}

func applyReregister(ctx context.Context, deps Dependencies, renderer *ui.Renderer, store rkstate.Store, target remote.Target, repoState rkstate.RepositoryState) (bool, int64, []recoveryResult, error) {
	var results []recoveryResult
	resolvedUnit := ops.ResolveActionsRunnerSystemdUnit(ctx, deps.RemoteExecutor, target, repoState.Machine.ServiceName)
	_ = runRecoveryCommand(ctx, deps.RemoteExecutor, target, remote.Command{ID: "recover.service.stop", Script: "sudo systemctl stop " + shellQuote(resolvedUnit) + " || true", Timeout: 30 * time.Second})
	results = append(results, recoveryResult{Step: "recover.service.stop", Status: "done"})
	teardownScript := renderBYORunnerSvcTeardownScript(repoState.Machine.InstallPath, resolvedUnit)
	_ = runRecoveryCommand(ctx, deps.RemoteExecutor, target, remote.Command{ID: "recover.service.uninstall", Script: teardownScript, Timeout: 60 * time.Second})
	results = append(results, recoveryResult{Step: "recover.service.uninstall", Status: "done"})
	removal, err := deps.GitHub.CreateRemovalToken(ctx, repoState.Repo)
	if err != nil {
		_ = renderer.Error("github_permission_denied", "RunnerKit can't create a fresh runner removal token.", []string{"Verify GitHub credentials can manage repository runners for " + repoState.Repo.FullName + "."})
		return false, repoState.Cleanup.GitHubRunnerID, results, NewExitError(ExitGitHubAuth, err)
	}
	renderer.Redactor().Register(redact.RunnerRemovalToken, removal.Token)
	removeResult, removeErr := deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "recover.runner.remove", Script: bootstrap.RenderRemoveConfigScript(repoState.Machine.InstallPath, bootstrap.DefaultServiceUser), Env: map[string]string{"RUNNERKIT_REMOVAL_TOKEN": removal.Token}, RedactArgs: []string{removal.Token}, Timeout: 60 * time.Second})
	if removeErr != nil || removeResult.ExitCode != 0 {
		text := removeResult.Stdout + " " + removeResult.Stderr
		if !isAlreadyAbsent(text) {
			return false, repoState.Cleanup.GitHubRunnerID, results, recoveryCommandError(renderer, errors.New("recover.runner.remove failed"))
		}
		results = append(results, recoveryResult{Step: "recover.runner.remove", Status: "skipped", Message: "already absent"})
	} else {
		results = append(results, recoveryResult{Step: "recover.runner.remove", Status: "done"})
	}
	registration, err := deps.GitHub.CreateRegistrationToken(ctx, repoState.Repo)
	if err != nil {
		_ = renderer.Error("github_permission_denied", "RunnerKit can't create a fresh runner registration token.", []string{"Verify GitHub credentials can manage repository runners for " + repoState.Repo.FullName + "."})
		return false, repoState.Cleanup.GitHubRunnerID, results, NewExitError(ExitGitHubAuth, err)
	}
	renderer.Redactor().Register(redact.RunnerRegistrationToken, registration.Token)
	configureScript := bootstrap.RenderReconfigureScript(bootstrap.Options{RunnerName: repoState.Runner.Name, RepoURL: "https://github.com/" + repoState.Repo.FullName, Labels: repoState.Runner.Labels, InstallPath: repoState.Machine.InstallPath, WorkDir: repoState.Machine.WorkDir, ServiceUser: bootstrap.DefaultServiceUser})
	if err := runRecoveryCommand(ctx, deps.RemoteExecutor, target, remote.Command{ID: "recover.runner.configure", Script: configureScript, Env: map[string]string{"RUNNERKIT_REGISTRATION_TOKEN": registration.Token}, RedactArgs: []string{registration.Token}, Timeout: 60 * time.Second}); err != nil {
		return false, repoState.Cleanup.GitHubRunnerID, results, recoveryCommandError(renderer, err)
	}
	results = append(results, recoveryResult{Step: "recover.runner.configure", Status: "done"})
	if err := runRecoveryCommand(ctx, deps.RemoteExecutor, target, remote.Command{ID: "recover.runner.start", Script: "cd " + shellQuote(repoState.Machine.InstallPath) + " && sudo ./svc.sh start && sudo ./svc.sh status", Timeout: 60 * time.Second}); err != nil {
		return false, repoState.Cleanup.GitHubRunnerID, results, recoveryCommandError(renderer, err)
	}
	results = append(results, recoveryResult{Step: "recover.runner.start", Status: "done"})
	runner, ok, err := waitForRunnerOnline(ctx, deps, repoState.Repo, repoState.Runner.Name, repoState.Runner.Labels)
	if err != nil {
		return false, repoState.Cleanup.GitHubRunnerID, results, err
	}
	if !ok {
		return false, repoState.Cleanup.GitHubRunnerID, results, NewExitError(ExitSafetyGate, errors.New("runner_online_timeout"))
	}
	repoState.Cleanup.GitHubRunnerID = runner.ID
	repoState.Operations = append(repoState.Operations, rkstate.OperationCheckpoint{Command: "recover", Artifact: "github_runner_id", Status: "updated", Message: "re-registered runner", UpdatedAt: deps.Clock()})
	repoState.UpdatedAt = deps.Clock()
	if err := store.UpdateRepository(repoState); err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't save recovered runner state.", []string{"Check permissions for " + store.Path() + "."})
		return false, repoState.Cleanup.GitHubRunnerID, results, NewExitError(ExitStateIO, err)
	}
	results = append(results, recoveryResult{Step: "state", Status: "updated", Message: fmt.Sprintf("github_runner_id=%d", runner.ID)})
	return true, runner.ID, results, nil
}

func runRecoveryCommand(ctx context.Context, executor remote.Executor, target remote.Target, command remote.Command) error {
	result, err := executor.Run(ctx, target, command)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("%s exited %d", command.ID, result.ExitCode)
	}
	return nil
}

func recoveryCommandError(renderer *ui.Renderer, err error) error {
	_ = renderer.Error("recovery_command_failed", "RunnerKit could not apply the recovery command.", []string{
		err.Error(),
		"Run runnerkit logs --repo owner/repo --since 30m for recent service output.",
		errcodes.FormatLine(errcodes.GHRecoverReregisterFailed),
	})
	return NewExitError(ExitSafetyGate, err)
}

func isAlreadyAbsent(text string) bool {
	lower := strings.ToLower(text)
	for _, marker := range []string{
		"not configured",
		"does not exist",
		"already removed",
		"not registered",
		"could not find a runner",
		"runner is not registered",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func recoveryPayload(repo string, dryRun bool, plan ops.RecoveryPlan, results []recoveryResult, stateUpdated bool, runnerID int64) map[string]any {
	if results == nil {
		results = []recoveryResult{}
	}
	return map[string]any{"ok": !plan.Blocked, "command": "recover", "repo": repo, "dry_run": dryRun, "plan": plan, "results": results, "state_updated": stateUpdated, "github_runner_id": runnerID}
}
