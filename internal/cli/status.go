package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/labels"
	"github.com/salar/runnerkit/internal/ops"
	"github.com/salar/runnerkit/internal/provider"
	"github.com/salar/runnerkit/internal/remote"
	rkstate "github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

type statusOptions struct {
	repo string
	all  bool
}

type statusResult struct {
	Repo     rkstate.RepositoryState `json:"-"`
	Observed ops.ObservedRunner      `json:"observed"`
	Health   ops.Health              `json:"health"`
}

const statusProviderJSONPath = "sources.provider"

func newStatusCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &statusOptions{}
	cmd := &cobra.Command{Use: "status"}
	cmd.Short = "Show RunnerKit-managed runner status"
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runStatus(deps, *jsonOutput, *noColor, opts)
	}
	cmd.Flags().StringVar(&opts.repo, "repo", "", "target GitHub repository as owner/name")
	cmd.Flags().BoolVar(&opts.all, "all", false, "show all locally managed runners")
	return cmd
}

func runStatus(deps Dependencies, jsonOutput bool, noColor bool, opts *statusOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()
	store := rkstate.NewStore(deps.StateBaseDir)
	if opts.all {
		repos, err := store.ListRepositories()
		if err != nil {
			_ = renderer.Error("state_io_failed", "RunnerKit can't read saved runner state.", []string{"Check permissions for " + store.Path() + "."})
			return NewExitError(ExitStateIO, err)
		}
		results := make([]statusResult, 0, len(repos))
		for _, repoState := range repos {
			results = append(results, collectStatus(ctx, deps, store.Path(), repoState, true))
		}
		if jsonOutput {
			items := make([]any, 0, len(results))
			for _, result := range results {
				items = append(items, statusJSONPayload("all", store.Path(), result.Observed, result.Health))
			}
			return renderer.JSON(map[string]any{"ok": true, "command": "status", "scope": "all", "runners": items})
		}
		if len(results) == 0 {
			return renderer.Step(1, 1, "runner status", ui.WarningLine("No RunnerKit-managed runner found"), ui.Bullet("Run runnerkit up --repo owner/name --host user@host to create a BYO runner, or pass --all to list saved runners."), ui.WarningLine("No RunnerKit-managed cloud runner is saved for `owner/name`."), ui.Bullet("Run `runnerkit up --repo owner/name --cloud hetzner` to create one, or pass `--host user@host` to use an existing machine."))
		}
		lines := []ui.Line{}
		for _, result := range results {
			lines = append(lines, statusSummaryLine(result))
		}
		return renderer.Step(1, 1, "runner status", lines...)
	}

	repo, err := resolveReadOnlyRepo(ctx, deps, renderer, opts.repo, "Pass --repo owner/name or run runnerkit status from a GitHub repository.")
	if err != nil {
		return err
	}
	repoState, ok, err := store.GetRepository(repo.FullName)
	if err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't read saved runner state.", []string{"Check permissions for " + store.Path() + "."})
		return NewExitError(ExitStateIO, err)
	}
	if !ok {
		observed := ops.ObservedRunner{Repo: repo.FullName, StatePath: store.Path(), StatePresent: false}
		health := ops.Classify(observed)
		if jsonOutput {
			return renderer.JSON(statusJSONPayload("repo", store.Path(), observed, health))
		}
		return renderer.Step(1, 1, "runner status",
			ui.WarningLine("No RunnerKit-managed runner found"),
			ui.Bullet("Run runnerkit up --repo owner/name --host user@host to create a BYO runner, or pass --all to list saved runners."),
			ui.WarningLine("No RunnerKit-managed cloud runner is saved for `owner/name`."),
			ui.Bullet("Run `runnerkit up --repo owner/name --cloud hetzner` to create one, or pass `--host user@host` to use an existing machine."),
			ui.WarningLine("No RunnerKit-managed runner is saved for `"+repo.FullName+"`."),
			ui.Bullet("Run `runnerkit up --repo "+repo.FullName+" --mode ephemeral --cloud hetzner` for a one-job cloud runner, or use `--host user@host` for an existing machine."),
		)
	}
	result := collectStatus(ctx, deps, store.Path(), repoState, true)
	if jsonOutput {
		return renderer.JSON(statusJSONPayload("repo", store.Path(), result.Observed, result.Health))
	}
	return renderStatusHuman(renderer, store.Path(), result)
}

func resolveReadOnlyRepo(ctx context.Context, deps Dependencies, renderer *ui.Renderer, rawRepo string, remediation string) (gh.Repo, error) {
	resolution, err := gh.ResolveTarget(ctx, gh.ResolveOptions{Repo: rawRepo, CommandRunner: deps.CommandRunner})
	if err != nil {
		message := fmt.Sprintf("RunnerKit can't continue because %s.", err.Error())
		_ = renderer.Error("invalid_repo", message, []string{remediation})
		code := ExitInvalidInput
		if rawRepo == "" {
			code = ExitInputRequired
		}
		return gh.Repo{}, NewExitError(code, err)
	}
	return resolution.Repo, nil
}

func collectStatus(ctx context.Context, deps Dependencies, statePath string, repoState rkstate.RepositoryState, probeRemote bool) statusResult {
	observed := ops.ObservedRunner{Repo: repoState.Repo.FullName, StatePath: statePath, StatePresent: true, State: &repoState}
	runners, err := deps.GitHub.ListRunners(ctx, repoState.Repo)
	if err != nil {
		observed.GitHub.Error = err.Error()
	} else {
		observed.GitHub = githubFactFor(repoState, runners)
	}
	observed.Labels = ops.CompareLabels(repoState.Runner.Labels, observed.GitHub.Labels)
	observed.Provider = collectProviderFact(ctx, deps, repoState)
	if probeRemote {
		if target, err := targetFromState(repoState); err == nil {
			observed.SSH, observed.Service = ops.ProbeRemoteStatus(ctx, deps.RemoteExecutor, target, repoState.Machine.HostKeyFingerprint, repoState.Machine.ServiceName)
		} else {
			observed.SSH = ops.SSHFact{Reachable: false, HostKey: "unknown", Error: err.Error()}
			observed.Service = ops.ServiceFact{Service: repoState.Machine.ServiceName, Error: "SSH target unavailable"}
		}
	}
	if repoState.Runner.Mode == "ephemeral" {
		observed.Ephemeral = collectEphemeralFact(ctx, deps, repoState, observed.SSH.Reachable)
	}
	health := ops.Classify(observed)
	return statusResult{Repo: repoState, Observed: observed, Health: health}
}

// collectEphemeralFact populates the EphemeralFact from saved metadata
// and, when SSH is reachable, attempts to read the remote sentinel
// state.json the finalizer writes so RunnerKit can surface the
// observed finalizer_status (completed/ttl_expired/pending).
func collectEphemeralFact(ctx context.Context, deps Dependencies, repoState rkstate.RepositoryState, sshReachable bool) ops.EphemeralFact {
	fact := ops.EphemeralFact{
		Mode:            "ephemeral",
		TTL:             repoState.Ephemeral.TTL,
		LogArchivePath:  repoState.Ephemeral.LogArchivePath,
		FinalizerStatus: repoState.Ephemeral.FinalizerStatus,
		CleanupCommand:  repoState.Ephemeral.CleanupCommand,
	}
	if repoState.Ephemeral.ExpiresAt != nil {
		fact.ExpiresAt = repoState.Ephemeral.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
	}
	fact.State = strings.ToLower(repoState.Ephemeral.FinalizerStatus)
	if sshReachable && deps.RemoteExecutor != nil {
		target, err := targetFromState(repoState)
		if err == nil {
			parent := strings.TrimSuffix(repoState.Ephemeral.LogArchivePath, "/logs")
			if parent == "" || parent == repoState.Ephemeral.LogArchivePath {
				return fact
			}
			result, err := deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "status.ephemeral.state", Script: "test -f " + shellQuote(parent+"/state.json") + " && cat " + shellQuote(parent+"/state.json") + " || true", Timeout: 5 * time.Second})
			if err == nil && result.ExitCode == 0 {
				if status := parseFinalizerStatus(result.Stdout); status != "" {
					fact.FinalizerStatus = status
					fact.State = status
				}
			}
		}
	}
	return fact
}

func parseFinalizerStatus(stdout string) string {
	for _, candidate := range []string{"completed", "ttl_expired", "pending"} {
		if strings.Contains(stdout, `"finalizer_status":"`+candidate+`"`) || strings.Contains(stdout, `"finalizer_status": "`+candidate+`"`) {
			return candidate
		}
	}
	return ""
}

func collectProviderFact(ctx context.Context, deps Dependencies, repoState rkstate.RepositoryState) ops.ProviderFact {
	if !isCloudProvider(repoState.Provider) {
		return ops.ProviderFact{Kind: defaultString(repoState.Provider.Kind, "byo"), Found: true, BillableResources: []string{}, Drift: []string{}}
	}
	fact := providerFactFromState(repoState.Provider)
	cloudProvider, ok := deps.Providers.Get(provider.HetznerProvider)
	if !ok || cloudProvider == nil {
		fact.Error = "provider dependency unavailable"
		return fact
	}
	status, err := cloudProvider.Describe(ctx, repoState.Provider)
	if err != nil {
		fact.Error = err.Error()
	}
	return mergeProviderStatus(fact, status)
}

func isCloudProvider(ref rkstate.ProviderRef) bool {
	return ref.Kind == provider.HetznerProvider || ref.Name == provider.HetznerProvider
}

func providerFactFromState(ref rkstate.ProviderRef) ops.ProviderFact {
	cloud := ref.Cloud
	return ops.ProviderFact{
		Kind:              defaultString(ref.Kind, defaultString(ref.Name, provider.HetznerProvider)),
		Found:             true,
		Status:            cloud.ServerStatus,
		Region:            defaultString(ref.Region, cloud.Region),
		ServerType:        defaultString(ref.Profile, cloud.ServerType),
		Image:             cloud.Image,
		PublicHost:        defaultString(cloud.PublicIPv4, cloud.PublicIPv6),
		BillableResources: cloudProviderResourceIDList(ref.ResourceIDs),
		Drift:             []string{},
	}
}

func mergeProviderStatus(base ops.ProviderFact, status provider.ProviderStatus) ops.ProviderFact {
	if status.Kind != "" {
		base.Kind = status.Kind
	}
	base.Found = status.Found
	if status.Status != "" {
		base.Status = status.Status
	}
	if status.Region != "" {
		base.Region = status.Region
	}
	if status.ServerType != "" {
		base.ServerType = status.ServerType
	}
	if status.Image != "" {
		base.Image = status.Image
	}
	if status.PublicHost != "" {
		base.PublicHost = status.PublicHost
	}
	if status.BillableResources != nil {
		base.BillableResources = append([]string(nil), status.BillableResources...)
	}
	if status.Drift != nil {
		base.Drift = append([]string(nil), status.Drift...)
	}
	if status.Error != "" {
		base.Error = status.Error
	}
	return base
}

func githubFactFor(repoState rkstate.RepositoryState, runners []gh.Runner) ops.GitHubFact {
	var candidates []gh.Runner
	repoLabel := runnerkitRepoLabel(repoState)
	for _, runner := range runners {
		if runner.ID == repoState.Cleanup.GitHubRunnerID && repoState.Cleanup.GitHubRunnerID != 0 {
			return ops.GitHubFact{Found: true, ID: runner.ID, Name: runner.Name, Status: runner.Status, Busy: runner.Busy, Labels: runner.Labels}
		}
		if runner.Name == repoState.Runner.Name || runnerHasLabels(runner.Labels, "runnerkit", repoLabel) {
			candidates = append(candidates, runner)
		}
	}
	if len(candidates) == 1 {
		runner := candidates[0]
		return ops.GitHubFact{Found: true, ID: runner.ID, Name: runner.Name, Status: runner.Status, Busy: runner.Busy, Labels: runner.Labels}
	}
	if len(candidates) > 1 {
		return ops.GitHubFact{Found: false, DuplicateCandidates: candidates}
	}
	return ops.GitHubFact{Found: false}
}

func runnerkitRepoLabel(repoState rkstate.RepositoryState) string {
	for _, label := range repoState.Runner.Labels {
		if strings.HasPrefix(label, "runnerkit-") && label != "runnerkit" {
			return label
		}
	}
	return "runnerkit-" + strings.ReplaceAll(repoState.Repo.FullName, "/", "-")
}

func runnerHasLabels(actual []string, required ...string) bool {
	set := map[string]bool{}
	for _, label := range actual {
		set[label] = true
	}
	for _, label := range required {
		if label == "" || !set[label] {
			return false
		}
	}
	return true
}

func targetFromState(repoState rkstate.RepositoryState) (remote.Target, error) {
	if strings.TrimSpace(repoState.Machine.HostRef) == "" {
		return remote.Target{}, errors.New("saved machine target is missing")
	}
	target, err := remote.ParseTarget(repoState.Machine.HostRef, repoState.Machine.Port)
	if err != nil {
		return remote.Target{}, err
	}
	target.KeyPath = repoState.Machine.KeyPathRef
	return target, nil
}

func renderStatusHuman(renderer *ui.Renderer, statePath string, result statusResult) error {
	observed := result.Observed
	repoState := result.Repo
	lines := []ui.Line{
		healthLine(result.Health),
		ui.Bullet("Repository: " + repoState.Repo.FullName),
		ui.Bullet("Runner: " + repoState.Runner.Name),
		ui.Bullet("Machine: " + repoState.Machine.HostRef),
		ui.Bullet("State path: " + statePath),
		ui.Bullet("Sources:"),
		ui.Bullet("    State       OK       saved runner metadata found"),
		ui.Bullet("    Provider   " + providerSourceText(observed.Provider)),
		ui.Bullet("    GitHub     " + githubSourceText(observed.GitHub)),
		ui.Bullet("    SSH        " + sshSourceText(observed.SSH)),
		ui.Bullet("    Service    " + serviceSourceText(observed.Service)),
		ui.Bullet("    Labels     " + labelsSourceText(observed.Labels)),
	}
	if repoState.Runner.Mode == "ephemeral" {
		lines = append(lines, ui.Bullet("Mode: ephemeral"))
		if profile := repoState.Safety.SafetyProfile; profile != "" {
			lines = append(lines, ui.Bullet("Safety profile: "+profile))
		}
		if archive := repoState.Ephemeral.LogArchivePath; archive != "" {
			lines = append(lines, ui.Bullet("Log archive: "+archive))
		}
		if cleanup := repoState.Ephemeral.CleanupCommand; cleanup != "" {
			lines = append(lines, ui.Bullet("Cleanup after the job: "+cleanup))
		}
	}
	lines = append(lines,
		ui.Bullet(repoState.Runner.WorkflowSnippet),
		ui.WarningLine(labels.SelfHostedAloneWarning),
	)
	if len(result.Health.NextActions) > 0 && result.Health.State != ops.HealthReady {
		lines = append(lines, ui.Next("Next: "+result.Health.NextActions[0].Command))
	}
	return renderer.Step(1, 1, "runner status", lines...)
}

func healthLine(health ops.Health) ui.Line {
	text := fmt.Sprintf("Health: %s — %s", health.State, health.Summary)
	switch health.State {
	case ops.HealthReady, ops.HealthBusy:
		return ui.Success(text)
	case ops.HealthBroken:
		return ui.ErrorLine(text)
	default:
		return ui.WarningLine(text)
	}
}

func statusSummaryLine(result statusResult) ui.Line {
	return ui.Bullet(fmt.Sprintf("%s: %s — %s", result.Repo.Repo.FullName, result.Health.State, result.Health.Summary))
}

func providerSourceText(fact ops.ProviderFact) string {
	if fact.Kind == "" || fact.Kind == "byo" {
		return "OK       byo, no billable provider resources"
	}
	if fact.Error != "" {
		return "WARNING  " + fact.Error
	}
	if !fact.Found {
		return "WARNING  " + displayProviderKind(fact.Kind) + " resource missing"
	}
	if len(fact.Drift) > 0 {
		return "WARNING  " + displayProviderKind(fact.Kind) + " drift: " + strings.Join(fact.Drift, ", ")
	}
	return "OK       " + displayProviderKind(fact.Kind) + " " + fact.Region + " " + fact.ServerType + " " + fact.Image + ", server " + providerServerID(fact.BillableResources) + ", state " + defaultString(fact.Status, "unknown") + ", public host " + fact.PublicHost
}

func displayProviderKind(kind string) string {
	if kind == provider.HetznerProvider {
		return "Hetzner"
	}
	if kind == "" {
		return "Provider"
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}

func providerServerID(resources []string) string {
	for _, resource := range resources {
		if strings.HasPrefix(resource, "server:") {
			return strings.TrimPrefix(resource, "server:")
		}
	}
	return "unknown"
}

func githubSourceText(fact ops.GitHubFact) string {
	if fact.Error != "" {
		return "WARNING  " + fact.Error
	}
	if !fact.Found {
		return "WARNING  missing"
	}
	busy := "not busy"
	if fact.Busy {
		busy = "busy"
	}
	return fmt.Sprintf("OK       %s, id %d, %s", defaultString(fact.Status, "unknown"), fact.ID, busy)
}

func sshSourceText(fact ops.SSHFact) string {
	if fact.HostKey == "mismatch" {
		return "ERROR    host key mismatch"
	}
	if !fact.Reachable {
		return "WARNING  " + defaultString(fact.Error, "unreachable")
	}
	return "OK       reachable, host key " + fact.HostKey
}

func serviceSourceText(fact ops.ServiceFact) string {
	if fact.Error != "" {
		return "WARNING  " + fact.Error
	}
	if fact.ActiveState == "active" {
		return "OK       active (" + fact.Service + ")"
	}
	if fact.ActiveState == "failed" {
		return "ERROR    failed (" + fact.Service + ")"
	}
	return "WARNING  " + defaultString(fact.ActiveState, "unknown") + " (" + fact.Service + ")"
}

func labelsSourceText(fact ops.LabelFact) string {
	if fact.Match {
		return "OK       saved labels match GitHub labels"
	}
	return "WARNING  missing [" + strings.Join(fact.Missing, ", ") + "] extra [" + strings.Join(fact.Extra, ", ") + "]"
}

func providerJSONSource(fact ops.ProviderFact) map[string]any {
	return map[string]any{
		"kind":               fact.Kind,
		"found":              fact.Found,
		"status":             fact.Status,
		"region":             fact.Region,
		"server_type":        fact.ServerType,
		"image":              fact.Image,
		"public_host":        fact.PublicHost,
		"billable_resources": fact.BillableResources,
		"drift":              fact.Drift,
		"error":              fact.Error,
	}
}

func statusJSONPayload(scope string, statePath string, observed ops.ObservedRunner, health ops.Health) map[string]any {
	runner := map[string]any{}
	if observed.State != nil {
		runner = map[string]any{"name": observed.State.Runner.Name, "labels": observed.State.Runner.Labels, "workflow_snippet": observed.State.Runner.WorkflowSnippet}
	}
	sources := map[string]any{
		"state":    map[string]any{"present": observed.StatePresent},
		"provider": providerJSONSource(observed.Provider),
		"github":   map[string]any{"found": observed.GitHub.Found, "id": observed.GitHub.ID, "status": observed.GitHub.Status, "busy": observed.GitHub.Busy, "labels": observed.GitHub.Labels},
		"ssh":      map[string]any{"reachable": observed.SSH.Reachable, "host_key": observed.SSH.HostKey},
		"systemd":  map[string]any{"service": observed.Service.Service, "active_state": observed.Service.ActiveState, "sub_state": observed.Service.SubState},
		"labels":   map[string]any{"match": observed.Labels.Match, "missing": observed.Labels.Missing, "extra": observed.Labels.Extra},
	}
	if observed.State != nil && observed.State.Runner.Mode == "ephemeral" {
		sources["ephemeral"] = map[string]any{
			"mode":             observed.Ephemeral.Mode,
			"state":            observed.Ephemeral.State,
			"ttl":              observed.Ephemeral.TTL,
			"expires_at":       observed.Ephemeral.ExpiresAt,
			"log_archive_path": observed.Ephemeral.LogArchivePath,
			"finalizer_status": observed.Ephemeral.FinalizerStatus,
			"cleanup_command":  observed.Ephemeral.CleanupCommand,
		}
	}
	return map[string]any{
		"ok":         true,
		"command":    "status",
		"scope":      scope,
		"repo":       observed.Repo,
		"state_path": statePath,
		"health":     health,
		"runner":     runner,
		"sources":    sources,
	}
}
