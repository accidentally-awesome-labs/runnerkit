package ops

import (
	"strings"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/state"
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
	ReasonCollectionError           = "collection_error"
)

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

type ObservedRunner struct {
	Repo             string                 `json:"repo"`
	StatePath        string                 `json:"state_path"`
	StatePresent     bool                   `json:"state_present"`
	State            *state.RepositoryState `json:"state,omitempty"`
	GitHub           GitHubFact             `json:"github"`
	SSH              SSHFact                `json:"ssh"`
	Service          ServiceFact            `json:"service"`
	Labels           LabelFact              `json:"labels"`
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

func serviceFailed(service ServiceFact) bool {
	return service.ActiveState == "failed" || service.SubState == "failed" || service.ExecMainStatus != "" && service.ExecMainStatus != "0"
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
