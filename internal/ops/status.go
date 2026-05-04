package ops

import (
	"strings"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

type HealthState string

const (
	// HealthReady HealthState = "ready"
	HealthReady          HealthState = "ready"
	HealthBusy           HealthState = "busy"
	HealthNeedsAttention HealthState = "needs_attention"
	HealthBroken         HealthState = "broken"
	HealthUnknown        HealthState = "unknown"
)

type Severity string

const (
	SeverityPass    Severity = "pass"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

const (
	ReasonStateMissing              = "state_missing"
	ReasonGitHubRunnerMissing       = "github_runner_missing"
	ReasonGitHubRunnerOffline       = "github_runner_offline"
	ReasonGitHubRunnerBusy          = "github_runner_busy"
	ReasonGitHubDuplicateCandidates = "github_duplicate_candidates"
	ReasonSSHUnreachable            = "ssh_unreachable"
	ReasonSSHHostKeyMismatch        = "ssh_host_key_mismatch"
	ReasonServiceFailed             = "service_failed"
	ReasonServiceInactive           = "service_inactive"
	ReasonServiceMissing            = "service_missing"
	ReasonLabelDrift                = "label_drift"
	ReasonProviderResourceMissing   = "provider_resource_missing"
	ReasonProviderDrift             = "provider_drift"
	ReasonCollectionError           = "collection_error"

	// Ephemeral lifecycle reason IDs.
	ReasonEphemeralWaiting        = "ephemeral_waiting"
	ReasonEphemeralBusy           = "ephemeral_busy"
	ReasonEphemeralCompleted      = "ephemeral_completed"
	ReasonEphemeralTTLExpired     = "ephemeral_ttl_expired"
	ReasonEphemeralCleanupPending = "ephemeral_cleanup_pending"
)

// EphemeralFact records the ephemeral lifecycle facts surfaced via
// `status` JSON `sources.ephemeral` and human bullets. RunnerKit
// populates it from saved EphemeralMetadata plus the remote sentinel
// state.json when SSH is reachable.
type EphemeralFact struct {
	Mode            string `json:"mode"`
	State           string `json:"state"`
	TTL             string `json:"ttl,omitempty"`
	ExpiresAt       string `json:"expires_at,omitempty"`
	LogArchivePath  string `json:"log_archive_path,omitempty"`
	FinalizerStatus string `json:"finalizer_status,omitempty"`
	CleanupCommand  string `json:"cleanup_command,omitempty"`
}

type Reason struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Evidence string `json:"evidence"`
}

type NextAction struct {
	Command string `json:"command"`
	Why     string `json:"why"`
}

type Health struct {
	State       HealthState  `json:"state"`
	Summary     string       `json:"summary"`
	Reasons     []Reason     `json:"reasons"`
	NextActions []NextAction `json:"next_actions"`
}

type CollectionError struct {
	Source string `json:"source"`
	Error  string `json:"error"`
}

type GitHubFact struct {
	Found               bool        `json:"found"`
	ID                  int64       `json:"id,omitempty"`
	Name                string      `json:"name,omitempty"`
	Status              string      `json:"status,omitempty"`
	Busy                bool        `json:"busy"`
	Labels              []string    `json:"labels,omitempty"`
	DuplicateCandidates []gh.Runner `json:"duplicate_candidates,omitempty"`
	Error               string      `json:"error,omitempty"`
}

type SSHFact struct {
	Reachable           bool   `json:"reachable"`
	HostKey             string `json:"host_key"`
	ObservedFingerprint string `json:"observed_fingerprint,omitempty"`
	Error               string `json:"error,omitempty"`
}

type ServiceFact struct {
	Service        string `json:"service"`
	LoadState      string `json:"load_state,omitempty"`
	ActiveState    string `json:"active_state,omitempty"`
	SubState       string `json:"sub_state,omitempty"`
	UnitFileState  string `json:"unit_file_state,omitempty"`
	ExecMainStatus string `json:"exec_main_status,omitempty"`
	Error          string `json:"error,omitempty"`
}

type LabelFact struct {
	Match    bool     `json:"match"`
	Expected []string `json:"expected"`
	Actual   []string `json:"actual"`
	Missing  []string `json:"missing"`
	Extra    []string `json:"extra"`
}

type ProviderFact struct {
	Kind              string   `json:"kind"`
	Found             bool     `json:"found"`
	Status            string   `json:"status"`
	Region            string   `json:"region"`
	ServerType        string   `json:"server_type"`
	Image             string   `json:"image"`
	PublicHost        string   `json:"public_host"`
	BillableResources []string `json:"billable_resources"`
	Drift             []string `json:"drift"`
	Error             string   `json:"error"`
}

type ObservedRunner struct {
	Repo             string                 `json:"repo"`
	StatePath        string                 `json:"state_path"`
	StatePresent     bool                   `json:"state_present"`
	State            *state.RepositoryState `json:"state,omitempty"`
	GitHub           GitHubFact             `json:"github"`
	SSH              SSHFact                `json:"ssh"`
	Service          ServiceFact            `json:"service"`
	Labels           LabelFact              `json:"labels"`
	Provider         ProviderFact           `json:"provider"`
	Ephemeral        EphemeralFact          `json:"ephemeral"`
	CollectionErrors []CollectionError      `json:"collection_errors,omitempty"`
}

func CompareLabels(expected, actual []string) LabelFact {
	expectedSet := map[string]bool{}
	actualSet := map[string]bool{}
	for _, label := range expected {
		expectedSet[label] = true
	}
	for _, label := range actual {
		actualSet[label] = true
	}
	var missing []string
	for _, label := range expected {
		if !actualSet[label] {
			missing = append(missing, label)
		}
	}
	var extra []string
	for _, label := range actual {
		if !expectedSet[label] {
			extra = append(extra, label)
		}
	}
	return LabelFact{Match: len(missing) == 0 && len(extra) == 0, Expected: append([]string(nil), expected...), Actual: append([]string(nil), actual...), Missing: missing, Extra: extra}
}

func Classify(observed ObservedRunner) Health {
	repo := observed.Repo
	if repo == "" && observed.State != nil {
		repo = observed.State.Repo.FullName
	}
	if !observed.StatePresent || observed.State == nil {
		return health(HealthUnknown, "RunnerKit can't determine runner health because required facts are missing.", reason(ReasonStateMissing, SeverityWarning, "state", "local RunnerKit state is missing"), next("runnerkit status --repo "+repo, "Load a saved RunnerKit-managed runner state."))
	}
	if len(observed.CollectionErrors) > 0 || observed.GitHub.Error != "" {
		return health(HealthUnknown, "RunnerKit can't determine runner health because required facts are missing.", reason(ReasonCollectionError, SeverityWarning, "collection", firstCollectionEvidence(observed)), next("runnerkit doctor --repo "+repo, "Collect deeper diagnostics for missing facts."))
	}
	if len(observed.GitHub.DuplicateCandidates) > 1 {
		return health(HealthBroken, "RunnerKit can't determine runner health because duplicate RunnerKit runner candidates were found.", reason(ReasonGitHubDuplicateCandidates, SeverityError, "github", "multiple RunnerKit runners match saved identity"), next("runnerkit down --repo "+repo+" --dry-run", "Review ambiguous GitHub runner records before cleanup."))
	}
	if observed.SSH.HostKey == "mismatch" {
		return health(HealthBroken, "RunnerKit stopped because the saved host key fingerprint does not match the current host.", reason(ReasonSSHHostKeyMismatch, SeverityError, "ssh", "saved host key fingerprint differs from observed host"), next("runnerkit doctor --repo "+repo, "Verify the machine identity before recovery."))
	}
	// Ephemeral mode classification runs BEFORE the persistent
	// `!observed.GitHub.Found` branch so missing GitHub runners after a
	// completed ephemeral job are treated as terminal progress, not
	// persistent recovery conditions.
	if observed.State.Runner.Mode == "ephemeral" {
		if classified, matched := classifyEphemeral(observed, repo); matched {
			return classified
		}
	}
	if observed.Service.LoadState == "not-found" || strings.Contains(strings.ToLower(observed.Service.Error), "missing") {
		if !observed.GitHub.Found {
			return health(HealthBroken, "RunnerKit can't determine runner health because required facts are missing.", reason(ReasonServiceMissing, SeverityError, "systemd", "saved service is missing"), next("runnerkit doctor --repo "+repo, "Inspect missing service and runner records."))
		}
		return health(HealthNeedsAttention, "RunnerKit can't determine runner health because required facts are missing.", reason(ReasonServiceMissing, SeverityError, "systemd", "saved service is missing"), next("runnerkit recover --repo "+repo+" --dry-run", "Preview service reinstall recovery."))
	}
	if !observed.GitHub.Found {
		return health(HealthNeedsAttention, "RunnerKit can't determine runner health because required facts are missing.", reason(ReasonGitHubRunnerMissing, SeverityWarning, "github", "saved GitHub runner was not found"), next("runnerkit recover --repo "+repo+" --dry-run", "Preview re-registration recovery."))
	}
	if !observed.Labels.Match {
		return health(HealthNeedsAttention, "Saved labels do not match the GitHub runner labels.", reason(ReasonLabelDrift, SeverityWarning, "labels", labelEvidence(observed.Labels)), next("runnerkit recover --repo "+repo+" --dry-run", "Preview label repair by re-registration."))
	}
	if providerNeedsAttention(observed.Provider) {
		id := ReasonProviderResourceMissing
		evidence := providerEvidence(observed.Provider)
		if len(observed.Provider.Drift) > 0 {
			id = ReasonProviderDrift
		}
		return health(HealthNeedsAttention, "Cloud provider resources need attention before this runner can be considered healthy.", reason(id, SeverityWarning, "provider", evidence), next("runnerkit destroy --repo "+repo+" --dry-run", "Review billable cloud resources before cleanup."))
	}
	if strings.EqualFold(observed.GitHub.Status, "offline") && observed.SSH.Reachable && serviceFailed(observed.Service) {
		return health(HealthNeedsAttention, "GitHub reports the runner offline while SSH is reachable and the service is failed.", reason(ReasonGitHubRunnerOffline, SeverityWarning, "github", "GitHub status is offline"), next("runnerkit doctor --repo "+repo, "Inspect service logs before recovery."))
	}
	if serviceFailed(observed.Service) {
		return health(HealthNeedsAttention, "GitHub reports the runner offline while SSH is reachable and the service is failed.", reason(ReasonServiceFailed, SeverityError, "systemd", "systemd ActiveState="+observed.Service.ActiveState), next("runnerkit logs --repo "+repo+" --since 30m", "Inspect recent service logs."))
	}
	if observed.Service.ActiveState != "" && observed.Service.ActiveState != "active" {
		return health(HealthNeedsAttention, "Runner service is inactive and needs attention.", reason(ReasonServiceInactive, SeverityWarning, "systemd", "systemd ActiveState="+observed.Service.ActiveState), next("runnerkit recover --repo "+repo+" --dry-run", "Preview service recovery."))
	}
	if !observed.SSH.Reachable {
		return health(HealthUnknown, "RunnerKit can't determine runner health because required facts are missing.", reason(ReasonSSHUnreachable, SeverityWarning, "ssh", "SSH is unreachable"), next("runnerkit doctor --repo "+repo, "Check SSH and host diagnostics."))
	}
	if observed.GitHub.Busy {
		return health(HealthBusy, "Runner is online and currently running a GitHub Actions job.", reason(ReasonGitHubRunnerBusy, SeverityPass, "github", "GitHub runner busy=true"), next("Wait for the current GitHub Actions job, or inspect GitHub Actions if it appears stuck.", "Runner is busy."))
	}
	if strings.EqualFold(observed.GitHub.Status, "online") && observed.SSH.Reachable && observed.Service.ActiveState == "active" && observed.Labels.Match {
		return health(HealthReady, "Runner is online, idle, reachable over SSH, service is active, and labels match.")
	}
	return health(HealthUnknown, "RunnerKit can't determine runner health because required facts are missing.", reason(ReasonCollectionError, SeverityWarning, "status", "insufficient status facts"), next("runnerkit doctor --repo "+repo, "Collect deeper diagnostics."))
}

// classifyEphemeral applies Phase 5 ephemeral lifecycle classification
// rules that run before persistent recovery conditions. It returns
// (Health, true) when the observed state matches an ephemeral
// terminal/active state and (zero, false) when callers should fall
// through to persistent classification.
func classifyEphemeral(observed ObservedRunner, repo string) (Health, bool) {
	repoState := observed.State
	cleanup := repoState.Ephemeral.CleanupCommand
	if cleanup == "" {
		cleanup = "runnerkit down --repo " + repo
	}
	// Prefer the observed (live remote sentinel) finalizer status over
	// the saved one so a freshly-completed or TTL-expired ephemeral
	// runner is classified as terminal even if the saved state still
	// records "pending".
	finalizer := strings.ToLower(observed.Ephemeral.FinalizerStatus)
	if finalizer == "" {
		finalizer = strings.ToLower(repoState.Ephemeral.FinalizerStatus)
	}
	// Cleanup pending: ephemeral run finished (finalizer completed) but
	// down/destroy still has pending checkpoints we must surface so the
	// CLI does not advertise success.
	if hasEphemeralCleanupPending(repoState.Operations, repoState.Cleanup.Notes) && (finalizer == "completed" || !observed.GitHub.Found) {
		return health(HealthNeedsAttention, "Ephemeral cleanup is incomplete and pending checkpoints remain.", reason(ReasonEphemeralCleanupPending, SeverityWarning, "state", "pending ephemeral cleanup checkpoints"), next(cleanup, "Re-run cleanup to finish removing the ephemeral runner.")), true
	}
	// TTL expired before completion: state expiry is in the past and
	// the finalizer never reported completed.
	if ttlExpired(repoState) && finalizer != "completed" {
		return health(HealthNeedsAttention, "Ephemeral runner TTL expired before a job completed. Run cleanup now.", reason(ReasonEphemeralTTLExpired, SeverityWarning, "ephemeral", "TTL safeguard expired"), next(cleanup, "Run cleanup to finalize the expired ephemeral runner.")), true
	}
	// Completed: GitHub auto-deregistered the ephemeral runner and the
	// finalizer reported completed. This is terminal happy progress —
	// the caller should NOT surface github_runner_missing as a
	// persistent recovery hint.
	if !observed.GitHub.Found && finalizer == "completed" {
		return health(HealthNeedsAttention, "Ephemeral runner completed one job and needs cleanup.", reason(ReasonEphemeralCompleted, SeverityPass, "ephemeral", "ephemeral runner finalized after one job"), next(cleanup, "Clean up the completed ephemeral runner.")), true
	}
	// Busy: GitHub reports the runner busy.
	if observed.GitHub.Found && observed.GitHub.Busy {
		return health(HealthBusy, "Ephemeral runner is running its one allowed job.", reason(ReasonEphemeralBusy, SeverityPass, "github", "GitHub runner busy=true"), next("Wait for the GitHub Actions job to finish, then run "+cleanup, "Ephemeral runner is running its one allowed job.")), true
	}
	// Waiting: GitHub reports the runner online and not busy.
	if observed.GitHub.Found && strings.EqualFold(observed.GitHub.Status, "online") && !observed.GitHub.Busy {
		return health(HealthReady, "Ephemeral runner is online and waiting for its one job.", reason(ReasonEphemeralWaiting, SeverityPass, "github", "GitHub runner online and idle"), next("Trigger a workflow targeting the runs-on snippet; cleanup with "+cleanup, "Ephemeral runner is online and waiting for its one job.")), true
	}
	return Health{}, false
}

func ttlExpired(repoState *state.RepositoryState) bool {
	if repoState == nil || repoState.Ephemeral.ExpiresAt == nil {
		return false
	}
	return repoState.Ephemeral.ExpiresAt.Before(time.Now())
}

func hasEphemeralCleanupPending(operations []state.OperationCheckpoint, notes []string) bool {
	for _, op := range operations {
		if op.Status == "pending" && (strings.Contains(op.Message, "ephemeral_log_preservation") || strings.Contains(op.Artifact, "ephemeral")) {
			return true
		}
	}
	for _, note := range notes {
		if strings.Contains(note, "ephemeral_log_preservation_pending") || strings.Contains(note, "ephemeral_cleanup_pending") {
			return true
		}
	}
	return false
}

func serviceFailed(service ServiceFact) bool {
	return service.ActiveState == "failed" || service.SubState == "failed" || service.ExecMainStatus != "" && service.ExecMainStatus != "0"
}

func providerNeedsAttention(provider ProviderFact) bool {
	if provider.Kind == "" || provider.Kind == "byo" {
		return false
	}
	return provider.Error != "" || !provider.Found || len(provider.Drift) > 0
}

func providerEvidence(provider ProviderFact) string {
	if provider.Error != "" {
		return provider.Error
	}
	if len(provider.Drift) > 0 {
		return "drift=" + strings.Join(provider.Drift, ",")
	}
	if !provider.Found {
		return "provider resource not found"
	}
	return "provider status=" + provider.Status
}

func labelEvidence(labels LabelFact) string {
	parts := []string{}
	if len(labels.Missing) > 0 {
		parts = append(parts, "missing="+strings.Join(labels.Missing, ","))
	}
	if len(labels.Extra) > 0 {
		parts = append(parts, "extra="+strings.Join(labels.Extra, ","))
	}
	return strings.Join(parts, " ")
}

func firstCollectionEvidence(observed ObservedRunner) string {
	if observed.GitHub.Error != "" {
		return observed.GitHub.Error
	}
	if len(observed.CollectionErrors) > 0 {
		return observed.CollectionErrors[0].Source + ": " + observed.CollectionErrors[0].Error
	}
	return "collection error"
}

func health(state HealthState, summary string, items ...any) Health {
	out := Health{State: state, Summary: summary, Reasons: []Reason{}, NextActions: []NextAction{}}
	for _, item := range items {
		switch typed := item.(type) {
		case Reason:
			out.Reasons = append(out.Reasons, typed)
		case NextAction:
			out.NextActions = append(out.NextActions, typed)
		}
	}
	return out
}

func reason(id string, severity Severity, source string, evidence string) Reason {
	return Reason{ID: id, Severity: string(severity), Source: source, Evidence: evidence}
}

func next(command string, why string) NextAction { return NextAction{Command: command, Why: why} }
