package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/errcodes"
	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

type downOptions struct {
	repo           string
	yes            bool
	dryRun         bool
	githubRunnerID int64
	runnerName     string
}

type cleanupResult struct {
	Artifact string `json:"artifact"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

var (
	errGitHubRunnerAmbiguous = errors.New("github_runner_ambiguous")
	errGitHubRunnerNotFound  = errors.New("github_runner_not_found")
)

func newDownCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &downOptions{}
	cmd := &cobra.Command{Use: "down"}
	cmd.Short = "Clean up a RunnerKit-managed BYO runner"
	cmd.RunE = func(_ *cobra.Command, _ []string) error { return runDown(deps, *jsonOutput, *noColor, opts) }
	cmd.Flags().StringVar(&opts.repo, "repo", "", "target GitHub repository as owner/name")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "apply the safe default cleanup plan")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "preview cleanup without mutation")
	cmd.Flags().Int64Var(&opts.githubRunnerID, "github-runner-id", 0, "delete this stale GitHub runner ID when local state is missing")
	cmd.Flags().StringVar(&opts.runnerName, "runner-name", "", "delete this stale RunnerKit runner name when local state is missing")
	return cmd
}

func runDown(deps Dependencies, jsonOutput bool, noColor bool, opts *downOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()
	repo, err := resolveReadOnlyRepo(ctx, deps, renderer, opts.repo, "Pass --repo owner/name for BYO cleanup.")
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
		return runDownWithoutState(ctx, deps, renderer, repo, opts, jsonOutput)
	}
	plan := ops.BuildCleanupPlan(repoState, opts.dryRun)
	if opts.dryRun {
		if jsonOutput {
			return renderer.JSON(cleanupPayload(repo.FullName, true, plan, nil, false, nil, false))
		}
		return renderCleanupPlanHuman(renderer, plan)
	}
	selected, err := selectCleanupArtifacts(ctx, deps, renderer, repoState, plan, opts, jsonOutput)
	if err != nil {
		return err
	}
	results, partial, pending, stateRemoved, err := applyCleanup(ctx, deps, renderer, store, repoState, selected, jsonOutput)
	if err != nil {
		return err
	}
	if jsonOutput {
		return renderer.JSON(cleanupPayload(repo.FullName, false, plan, results, partial, pending, stateRemoved))
	}
	lines := []ui.Line{}
	if partial {
		lines = append(lines, ui.WarningLine("Cleanup incomplete"))
		// Emit the canonical RKD codes that map to the most common
		// down-partial paths. The See: URL points users at the right
		// troubleshooting entry per D-15.
		for _, p := range pending {
			switch p {
			case "ephemeral_log_preservation_pending":
				lines = append(lines, ui.Bullet(errcodes.FormatLine(errcodes.CleanEphemeralLogPreserveFailed)))
			}
		}
	} else {
		lines = append(lines, ui.Success("Cleanup complete"))
	}
	for _, result := range results {
		lines = append(lines, ui.Bullet(result.Artifact+": "+result.Status+" "+result.Message))
		if result.Status == "failed" && result.Artifact == string(ops.ArtifactRunnerFiles) {
			// File removal failure surfaces as a cleanup_result; surface
			// the RKD-CLEAN-003 docs reference alongside.
			lines = append(lines, ui.Bullet(errcodes.FormatLine(errcodes.CleanDownFilesRemoveFailed)))
		}
	}
	return renderer.Step(1, 1, "cleanup complete", lines...)
}

func runDownWithoutState(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo, opts *downOptions, jsonOutput bool) error {
	if !opts.yes && !opts.dryRun {
		message := "RunnerKit can't continue because local state is missing for " + repo.FullName + "."
		_ = renderer.Error("input_required", message, []string{"Pass --github-runner-id <id> to delete a stale GitHub runner when local state is missing."})
		return NewExitError(ExitInputRequired, errors.New(message))
	}
	runners, err := deps.GitHub.ListRunners(ctx, repo)
	if err != nil {
		_ = renderer.Error("github_runner_list_failed", "RunnerKit can't list repository self-hosted runners.", []string{"Verify GitHub credentials can list repository runners for " + repo.FullName + "."})
		return NewExitError(ExitGitHubAuth, err)
	}
	id, err := selectStaleRunnerWithoutState(runners, opts)
	if err != nil {
		code := "github_runner_ambiguous"
		remediation := []string{"Pass --github-runner-id <id> to delete exactly one stale GitHub runner."}
		if errors.Is(err, errGitHubRunnerNotFound) {
			code = "github_runner_not_found"
			remediation = []string{
				"Confirm the runner still exists in GitHub, then retry with --github-runner-id <id>.",
				"Or pass --runner-name <name> when only one stale RunnerKit runner with that name exists.",
			}
		}
		_ = renderer.Error(code, err.Error(), remediation)
		return NewExitError(ExitSafetyGate, err)
	}
	plan := ops.CleanupPlan{Repo: repo.FullName, DryRun: opts.dryRun, Artifacts: []ops.CleanupArtifactPlan{{Artifact: ops.ArtifactGitHubRunner, Action: fmt.Sprintf("delete GitHub runner id %d", id), DefaultSelected: true, RequiresConfirmation: true}}}
	if opts.dryRun {
		if jsonOutput {
			return renderer.JSON(cleanupPayload(repo.FullName, true, plan, nil, false, nil, false))
		}
		return renderCleanupPlanHuman(renderer, plan)
	}
	if err := deps.GitHub.DeleteRunner(ctx, repo, id); err != nil {
		_ = renderer.Error("github_runner_delete_failed", "RunnerKit can't delete the stale GitHub runner.", []string{err.Error()})
		return NewExitError(ExitGitHubAuth, err)
	}
	results := []cleanupResult{{Artifact: string(ops.ArtifactGitHubRunner), Status: "done", Message: fmt.Sprintf("deleted runner id %d", id)}}
	if jsonOutput {
		return renderer.JSON(cleanupPayload(repo.FullName, false, plan, results, false, nil, false))
	}
	return renderer.Step(1, 1, "cleanup complete", ui.Success("Deleted stale GitHub runner id "+fmt.Sprintf("%d", id)))
}

func selectStaleRunnerWithoutState(runners []gh.Runner, opts *downOptions) (int64, error) {
	if opts.githubRunnerID != 0 {
		for _, runner := range runners {
			if runner.ID == opts.githubRunnerID {
				return runner.ID, nil
			}
		}
		return 0, fmt.Errorf("GitHub runner id %d was not found", opts.githubRunnerID)
	}
	if opts.runnerName != "" {
		var matches []gh.Runner
		for _, runner := range runners {
			if runner.Name == opts.runnerName && runnerHasLabels(runner.Labels, "runnerkit") {
				matches = append(matches, runner)
			}
		}
		if len(matches) == 1 {
			return matches[0].ID, nil
		}
		if len(matches) == 0 {
			return 0, fmt.Errorf("%w: no stale RunnerKit GitHub runner matched --runner-name %q", errGitHubRunnerNotFound, opts.runnerName)
		}
		return 0, errGitHubRunnerAmbiguous
	}
	return 0, errors.New("Pass --github-runner-id <id> to delete a stale GitHub runner when local state is missing.")
}

func selectCleanupArtifacts(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repoState rkstate.RepositoryState, plan ops.CleanupPlan, opts *downOptions, jsonOutput bool) (map[ops.CleanupArtifact]bool, error) {
	selected := map[ops.CleanupArtifact]bool{}
	if opts.yes {
		for _, artifact := range plan.Artifacts {
			selected[artifact.Artifact] = artifact.DefaultSelected && !artifact.Blocked
		}
		return selected, nil
	}
	if jsonOutput || !deps.TTY.StdinTTY || deps.Prompts == nil {
		message := "RunnerKit can't continue because BYO cleanup requires confirmation."
		_ = renderer.Error("input_required", message, []string{"Pass --yes to apply the safe default cleanup plan non-interactively, or pass --dry-run to preview only."})
		return nil, NewExitError(ExitInputRequired, errors.New(message))
	}
	prompts := cleanupPromptMessages(repoState)
	for _, artifact := range plan.Artifacts {
		if artifact.Blocked {
			continue
		}
		confirmed, err := deps.Prompts.Confirm(ctx, ui.Prompt{Message: prompts[artifact.Artifact], Default: false})
		if err != nil {
			return nil, err
		}
		selected[artifact.Artifact] = confirmed
	}
	return selected, nil
}

func cleanupPromptMessages(repoState rkstate.RepositoryState) map[ops.CleanupArtifact]string {
	return map[ops.CleanupArtifact]string{
		ops.ArtifactGitHubRunner:     "Remove GitHub runner " + repoState.Runner.Name + " from " + repoState.Repo.FullName + "? [y/N]",
		ops.ArtifactHostRegistration: "Unconfigure the runner registration on " + repoState.Machine.HostRef + "? [y/N]",
		ops.ArtifactSystemdService:   "Stop and uninstall service " + repoState.Machine.ServiceName + " on " + repoState.Machine.HostRef + "? [y/N]",
		ops.ArtifactRunnerFiles:      "Remove RunnerKit install path " + repoState.Machine.InstallPath + " and work dir " + repoState.Machine.WorkDir + "? [y/N]",
		ops.ArtifactLocalState:       "Remove local RunnerKit state for " + repoState.Repo.FullName + "? [y/N]",
	}
}

func renderCleanupPlanHuman(renderer *ui.Renderer, plan ops.CleanupPlan) error {
	lines := []ui.Line{ui.WarningLine("This will remove RunnerKit-managed runner artifacts for " + plan.Repo + ".")}
	for _, artifact := range plan.Artifacts {
		line := artifact.Description + ": " + artifact.Action
		if artifact.Blocked {
			lines = append(lines, ui.ErrorLine(line+" (blocked: "+artifact.BlockReason+")"))
		} else {
			lines = append(lines, ui.Bullet(line))
		}
	}
	for _, warning := range plan.Warnings {
		lines = append(lines, ui.WarningLine(warning))
	}
	lines = append(lines, ui.Next("Next: answer each prompt, pass --dry-run to preview only, or pass --yes for the safe default plan."))
	return renderer.Step(1, 1, "cleanup plan", lines...)
}

func applyCleanup(ctx context.Context, deps Dependencies, renderer *ui.Renderer, store rkstate.Store, repoState rkstate.RepositoryState, selected map[ops.CleanupArtifact]bool, jsonOutput bool) ([]cleanupResult, bool, []string, bool, error) {
	renderer.Redactor().Register(redact.MachineRef, repoState.Machine.HostRef)
	status := collectStatus(ctx, deps, store.Path(), repoState, true)
	sshReachable := status.Observed.SSH.Reachable
	var results []cleanupResult
	pending := []string{}
	partial := false
	stateRemoved := false
	target, targetErr := targetFromState(repoState)
	// Bug 21 (Plan 06-10, 2026-05-06): probe sudo before remote
	// cleanup. If `sudo -n install --version >/dev/null` fails on the host (password-protected
	// sudo, no NOPASSWD scope for rm/svc.sh on the requested paths),
	// prompt for the sudo password (TTY required) and thread it
	// through service-uninstall + files-remove via the same
	// printf|sudo -S -v pattern bootstrap uses (Plan 06-09 Bug 10).
	// On hosts where sudo IS passwordless (NOPASSWD ALL or Path C
	// byo-prepare), the probe exits 0 and we keep the existing
	// unwrapped happy path.
	if targetErr == nil && needsAnyRemoteSudo(selected) {
		needs, probeErr := probeSudoNeedsPassword(ctx, deps.RemoteExecutor, target)
		if probeErr == nil && needs {
			err := RenderHostInstallRequired(renderer, jsonOutput, deps.Version)
			return nil, false, nil, false, err
		}
	}
	needsRunnerStop := selected[ops.ArtifactSystemdService] || selected[ops.ArtifactHostRegistration] || selected[ops.ArtifactRunnerFiles]
	needsSvcTeardown := selected[ops.ArtifactSystemdService] || selected[ops.ArtifactHostRegistration]
	resolvedUnit := repoState.Machine.ServiceName
	if sshReachable && targetErr == nil && needsRunnerStop {
		resolvedUnit = ops.ResolveActionsRunnerSystemdUnit(ctx, deps.RemoteExecutor, target, repoState.Machine.ServiceName)
	}
	// Bug 19 (GitHub unit naming): stop the *resolved* actions.runner.* unit
	// before other runner teardown. For `config.sh remove`, the Actions
	// runner requires `./svc.sh uninstall` first — current runners emit
	// "Uninstall service first" otherwise (actions/runner#1022 documents
	// related "Unconfigure service first" stale-.service cases; upstream
	// resolution is uninstall, then remove).
	if sshReachable && targetErr == nil && needsRunnerStop {
		stopScript := renderBYORunnerStopScript(repoState.Machine.InstallPath, resolvedUnit)
		stopRes, stopErr := deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "down.service.stop", Script: stopScript, Timeout: 60 * time.Second})
		var teardownRes remote.Result
		var teardownErr error
		preremoveUninstall := selected[ops.ArtifactHostRegistration] && needsSvcTeardown
		systemdOnlyUninstall := needsSvcTeardown && !selected[ops.ArtifactHostRegistration]
		if preremoveUninstall {
			teardownScript := renderBYORunnerSvcTeardownScript(repoState.Machine.InstallPath, resolvedUnit)
			teardownRes, teardownErr = deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "down.service.uninstall", Script: teardownScript, Timeout: 60 * time.Second})
		}
		if selected[ops.ArtifactHostRegistration] {
			removal, err := deps.GitHub.CreateRemovalToken(ctx, repoState.Repo)
			if err != nil {
				return nil, false, nil, false, NewExitError(ExitGitHubAuth, err)
			}
			renderer.Redactor().Register(redact.RunnerRemovalToken, removal.Token)
			removeScript := bootstrap.RenderRemoveConfigScript(repoState.Machine.InstallPath, bootstrap.DefaultServiceUser)
			result, err := deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "down.runner.remove", Script: removeScript, Env: map[string]string{"RUNNERKIT_REMOVAL_TOKEN": removal.Token}, RedactArgs: []string{removal.Token}, Timeout: 60 * time.Second})
			if err != nil || result.ExitCode != 0 {
				detail := remoteCleanupDetail(renderer.Redactor(), result)
				if isAlreadyAbsent(result.Stdout + " " + result.Stderr) {
					results = append(results, cleanupResult{Artifact: string(ops.ArtifactHostRegistration), Status: "skipped", Message: "already absent"})
				} else {
					partial = true
					pending = append(pending, "remote_cleanup_pending")
					msg := "remote_cleanup_pending"
					if detail != "" {
						msg += " (" + detail + ")"
					}
					results = append(results, cleanupResult{Artifact: string(ops.ArtifactHostRegistration), Status: "pending", Message: msg})
				}
			} else {
				results = append(results, cleanupResult{Artifact: string(ops.ArtifactHostRegistration), Status: "done"})
			}
		}
		if systemdOnlyUninstall {
			teardownScript := renderBYORunnerSvcTeardownScript(repoState.Machine.InstallPath, resolvedUnit)
			teardownRes, teardownErr = deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "down.service.uninstall", Script: teardownScript, Timeout: 60 * time.Second})
		}
		if selected[ops.ArtifactSystemdService] {
			combined := mergeSystemdCleanupStatuses(statusFromRemoteResult(stopRes, stopErr), statusFromRemoteResult(teardownRes, teardownErr))
			if combined == "failed" {
				partial = true
				pending = appendUnique(pending, "remote_cleanup_pending")
			}
			msg := "stop: " + shortRunnerCleanupMsg(stopRes, stopErr)
			if needsSvcTeardown {
				msg += "; uninstall: " + shortRunnerCleanupMsg(teardownRes, teardownErr)
			}
			results = append(results, cleanupResult{Artifact: string(ops.ArtifactSystemdService), Status: combined, Message: strings.TrimSpace(msg)})
		}
		if selected[ops.ArtifactRunnerFiles] {
			// Preserve ephemeral _diag and journal logs to the host archive
			// directory before removing the install path. We never block
			// file removal on the preservation step; failures are recorded
			// as a pending checkpoint via Cleanup.Notes/Operations below.
			if repoState.Runner.Mode == "ephemeral" {
				preserveResult, preserveErr := deps.RemoteExecutor.Run(ctx, target, remote.Command{
					ID:      "ephemeral.logs.preserve",
					Script:  bootstrap.RenderEphemeralLogPreservationScript(repoState.Machine.InstallPath, repoState.Ephemeral.LogArchivePath, repoState.Machine.ServiceName),
					Sudo:    true,
					Timeout: 60 * time.Second,
				})
				if preserveErr != nil || preserveResult.ExitCode != 0 {
					partial = true
					pending = appendUnique(pending, "ephemeral_log_preservation_pending")
				}
			}
			installPath, workDir, blocked, reason := ops.SafeRunnerPaths(repoState)
			if blocked {
				results = append(results, cleanupResult{Artifact: string(ops.ArtifactRunnerFiles), Status: "blocked", Message: reason})
				partial = true
			} else {
				script := "sudo rm -rf -- " + shellQuote(installPath) + " " + shellQuote(workDir)
				cmd := remote.Command{ID: "down.files.remove", Script: script, Timeout: 60 * time.Second}
				result, err := deps.RemoteExecutor.Run(ctx, target, cmd)
				results = append(results, cleanupResult{Artifact: string(ops.ArtifactRunnerFiles), Status: statusFromRemoteResult(result, err), Message: idempotentMessage(result)})
			}
		}
	}
	if selected[ops.ArtifactGitHubRunner] {
		deleted, err := deleteGitHubRunnerCandidate(ctx, deps, repoState)
		if err != nil {
			partial = true
			pending = append(pending, "github_cleanup_pending")
			results = append(results, cleanupResult{Artifact: string(ops.ArtifactGitHubRunner), Status: "pending", Message: "github_cleanup_pending"})
		} else if deleted == 0 {
			results = append(results, cleanupResult{Artifact: string(ops.ArtifactGitHubRunner), Status: "skipped", Message: "GitHub runner already absent"})
		} else {
			results = append(results, cleanupResult{Artifact: string(ops.ArtifactGitHubRunner), Status: "done", Message: fmt.Sprintf("deleted runner id %d", deleted)})
		}
	}
	if !sshReachable {
		partial = true
		pending = appendUnique(pending, "remote_cleanup_pending")
		repoState.Cleanup.Notes = appendUnique(repoState.Cleanup.Notes, "remote_cleanup_pending")
		repoState.Operations = append(repoState.Operations, rkstate.OperationCheckpoint{Command: "down", Artifact: "remote", Status: "pending", Message: "SSH unreachable during cleanup", UpdatedAt: deps.Clock()})
		repoState.UpdatedAt = deps.Clock()
		_ = store.UpdateRepository(repoState)
		return results, partial, pending, false, nil
	}
	if selected[ops.ArtifactLocalState] && !partial {
		removed, err := store.RemoveRepository(repoState.Repo.FullName)
		if err != nil {
			return results, false, pending, false, NewExitError(ExitStateIO, err)
		}
		stateRemoved = removed
		results = append(results, cleanupResult{Artifact: string(ops.ArtifactLocalState), Status: "done"})
	} else if partial {
		if contains(pending, "github_cleanup_pending") {
			repoState.Cleanup.Notes = appendUnique(repoState.Cleanup.Notes, "github_cleanup_pending")
			repoState.Operations = append(repoState.Operations, rkstate.OperationCheckpoint{Command: "down", Artifact: "github_runner", Status: "pending", Message: "GitHub cleanup pending", UpdatedAt: deps.Clock()})
		}
		if len(pending) > 0 {
			repoState.UpdatedAt = deps.Clock()
			_ = store.UpdateRepository(repoState)
		}
	}
	return results, partial, pending, stateRemoved, nil
}

func deleteGitHubRunnerCandidate(ctx context.Context, deps Dependencies, repoState rkstate.RepositoryState) (int64, error) {
	runners, err := deps.GitHub.ListRunners(ctx, repoState.Repo)
	if err != nil {
		return 0, err
	}
	var candidates []gh.Runner
	for _, runner := range runners {
		if repoState.Cleanup.GitHubRunnerID != 0 && runner.ID == repoState.Cleanup.GitHubRunnerID {
			candidates = []gh.Runner{runner}
			break
		}
		if runner.Name == repoState.Runner.Name && runnerHasLabels(runner.Labels, "runnerkit", runnerkitRepoLabel(repoState)) {
			candidates = append(candidates, runner)
		}
	}
	if len(candidates) == 0 {
		return 0, nil
	}
	if len(candidates) > 1 {
		return 0, errors.New("github_runner_ambiguous")
	}
	if err := deps.GitHub.DeleteRunner(ctx, repoState.Repo, candidates[0].ID); err != nil {
		return 0, err
	}
	return candidates[0].ID, nil
}

func statusFromRemoteResult(result remote.Result, err error) string {
	if err == nil && result.ExitCode == 0 {
		return "done"
	}
	if isAlreadyAbsent(result.Stdout + " " + result.Stderr) {
		return "skipped"
	}
	return "failed"
}

func idempotentMessage(result remote.Result) string {
	text := result.Stdout + " " + result.Stderr
	if isAlreadyAbsent(text) {
		return "already absent"
	}
	return strings.TrimSpace(text)
}

func renderBYORunnerStopScript(installPath, systemdUnit string) string {
	return "cd " + shellQuote(installPath) + " && sudo ./svc.sh stop || true\nsudo systemctl stop " + shellQuote(systemdUnit) + " || true\nsleep 3\n"
}

func renderBYORunnerSvcTeardownScript(installPath, systemdUnit string) string {
	return "cd " + shellQuote(installPath) + " && sudo ./svc.sh uninstall || true\nsudo systemctl disable --now " + shellQuote(systemdUnit) + " || true\n"
}

func mergeSystemdCleanupStatuses(stopStatus, teardownStatus string) string {
	if stopStatus == "failed" || teardownStatus == "failed" {
		return "failed"
	}
	if stopStatus == "pending" || teardownStatus == "pending" {
		return "pending"
	}
	if stopStatus == "done" || teardownStatus == "done" {
		return "done"
	}
	return "skipped"
}

func remoteCleanupDetail(red *redact.Redactor, result remote.Result) string {
	s := strings.TrimSpace(result.Stdout + " " + result.Stderr)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if red != nil {
		s = red.String(s)
	}
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}

func shortRunnerCleanupMsg(res remote.Result, err error) string {
	if err != nil {
		return strings.TrimSpace(err.Error())
	}
	msg := strings.TrimSpace(res.Stdout + " " + res.Stderr)
	if isAlreadyAbsent(msg) {
		return "already absent"
	}
	if res.ExitCode != 0 && msg != "" {
		return msg
	}
	if res.ExitCode != 0 {
		return "non-zero exit"
	}
	return "ok"
}

func appendUnique(values []string, value string) []string {
	if contains(values, value) {
		return values
	}
	return append(values, value)
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

// needsAnyRemoteSudo returns true when the selected cleanup artifacts
// involve a remote sudo invocation (svc.sh stop/uninstall, systemctl
// disable, or rm -rf of the install path). Bug 21 only needs to probe
// + thread sudo when at least one of these is selected — the GitHub +
// local-state cleanup paths run locally and never invoke remote sudo.
func needsAnyRemoteSudo(selected map[ops.CleanupArtifact]bool) bool {
	return selected[ops.ArtifactSystemdService] || selected[ops.ArtifactRunnerFiles] || selected[ops.ArtifactHostRegistration]
}

// probeSudoNeedsPassword runs `sudo -n install --version >/dev/null` on
// the remote host. Exit
// code 0 means sudo is passwordless (NOPASSWD ALL, Path C byo-prepare,
// or a previously-cached cred). Non-zero exit + stderr containing
// `password is required` / `a terminal is required` / `a password is
// required` indicates password-protected sudo. Other non-zero exits
// (e.g. command-not-found, network) return needs=false and the caller
// keeps the existing happy path; the underlying cleanup will surface
// the real error if any.
//
// Bug 28 (Plan 06-12, 2026-05-06): the real SSH executor returns
// `err = *exec.ExitError` for ANY non-zero remote rc — that's the
// EXPECTED case for a password-protected sudo (rc=1 + stderr marker).
// The previous early `if err != nil { return false, nil }` guard
// misclassified that as "no password needed" and skipped the prompt.
// The fix: inspect `result.ExitCode` + `result.Stderr + result.Stdout`
// REGARDLESS of err. See internal/remote/system.go:81-89 for the
// err+ExitCode contract — exec.ExitError populates result.ExitCode;
// other err sets result.ExitCode = -1 (treated as "unknown, fall
// through" via the default branch below). Plan 06-07 attempt-17
// smoke-output.log showed `probe-direct: rc=1 err=exit status 1`
// followed by `probe: needs=false` — Bug 28 closes that cascade.
func probeSudoNeedsPassword(ctx context.Context, executor remote.Executor, target remote.Target) (bool, error) {
	if executor == nil {
		return false, nil
	}
	result, _ := executor.Run(ctx, target, remote.Command{
		ID:      "down.sudo.probe",
		Script:  "sudo -n install --version >/dev/null",
		Timeout: 5 * time.Second,
	})
	// Happy path: sudo passwordless (NOPASSWD / Path C / cached cred).
	if result.ExitCode == 0 {
		return false, nil
	}
	// Password-required path: non-zero exit code with marker substring.
	// This is the Bug 28 surface — works regardless of whether the
	// executor returned a wrapping err alongside the result.
	stderr := strings.ToLower(result.Stderr + " " + result.Stdout)
	for _, marker := range []string{"password is required", "a terminal is required", "no tty present"} {
		if strings.Contains(stderr, marker) {
			return true, nil
		}
	}
	// Unknown non-zero — assume sudo works but in an unexpected mode and
	// keep the unwrapped path. If it does need a password the cleanup
	// will report the canonical sudo failure verbatim. This branch also
	// catches executor-startup failures (dial timeout, context cancel)
	// where result.ExitCode = -1 and err is non-nil but not an
	// exit-status wrapper — preserves the existing graceful-failure
	// semantics.
	return false, nil
}

func cleanupPayload(repo string, dryRun bool, plan ops.CleanupPlan, results []cleanupResult, partial bool, pending []string, stateRemoved bool) map[string]any {
	if results == nil {
		results = []cleanupResult{}
	}
	if pending == nil {
		pending = []string{}
	}
	return map[string]any{"ok": !partial, "command": "down", "repo": repo, "dry_run": dryRun, "plan": plan, "results": results, "partial_cleanup": partial, "pending": pending, "state_removed": stateRemoved}
}
