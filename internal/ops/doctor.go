package ops

import (
	"fmt"
	"strings"

	"github.com/salar/runnerkit/internal/preflight"
	"github.com/salar/runnerkit/internal/state"
)

type Finding struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Source      string `json:"source"`
	Evidence    string `json:"evidence"`
	Remediation string `json:"remediation"`
}

type DoctorReport struct {
	Repo        string       `json:"repo"`
	StatePath   string       `json:"state_path"`
	Health      Health       `json:"health"`
	Findings    []Finding    `json:"findings"`
	NextActions []NextAction `json:"next_actions"`
}

type DeepChecks struct {
	InstallPathOK    bool
	WorkDirOK        bool
	InstallPathError string
	WorkDirError     string
	Preflight        preflight.Report
}

func BuildDoctorReport(repoState state.RepositoryState, observed ObservedRunner, checks DeepChecks) DoctorReport {
	health := Classify(observed)
	repo := repoState.Repo.FullName
	report := DoctorReport{Repo: repo, StatePath: observed.StatePath, Health: health, Findings: []Finding{}, NextActions: health.NextActions}
	add := func(id string, severity Severity, source string, evidence string, remediation string) {
		report.Findings = append(report.Findings, Finding{ID: id, Severity: string(severity), Source: source, Evidence: evidence, Remediation: remediation})
	}
	statusCmd := "runnerkit status --repo " + repo
	logsCmd := "runnerkit logs --repo " + repo + " --since 30m"
	recoverReregister := "runnerkit recover --repo " + repo + " --reregister --dry-run"
	downDryRun := "runnerkit down --repo " + repo + " --dry-run"
	add("state_present", SeverityPass, "state", "local RunnerKit state is present", statusCmd)
	if observed.GitHub.Found {
		add("github_runner_found", SeverityPass, "github", fmt.Sprintf("GitHub runner %s id %d is present", observed.GitHub.Name, observed.GitHub.ID), statusCmd)
		if strings.EqualFold(observed.GitHub.Status, "offline") {
			add("github_runner_offline", SeverityWarning, "github", "GitHub reports runner status offline", logsCmd)
		}
	} else if len(observed.GitHub.DuplicateCandidates) > 1 {
		add("github_duplicate_candidates", SeverityError, "github", "multiple RunnerKit runner candidates found", downDryRun)
	} else {
		add("github_runner_offline", SeverityWarning, "github", "GitHub runner is missing or offline", logsCmd)
	}
	if !observed.SSH.Reachable {
		if observed.SSH.HostKey == "mismatch" {
			add("ssh_host_key_mismatch", SeverityError, "ssh", "saved host key fingerprint does not match observed host", "Verify the machine identity before running runnerkit recover --repo "+repo+".")
		} else {
			add("ssh_unreachable", SeverityError, "ssh", "SSH is unreachable", "Verify SSH access to "+repoState.Machine.HostRef+", then re-run runnerkit doctor --repo "+repo+".")
		}
	}
	if observed.Service.ActiveState == "active" {
		add("service_active", SeverityPass, "systemd", "systemd reports ActiveState=active", statusCmd)
	} else if serviceFailed(observed.Service) {
		add("service_failed", SeverityError, "systemd", "systemd reports ActiveState=failed for runnerkit-runner.", logsCmd)
	} else if observed.Service.LoadState == "not-found" || strings.Contains(strings.ToLower(observed.Service.Error), "missing") {
		add("service_missing", SeverityError, "systemd", "saved systemd service is missing", "runnerkit recover --repo "+repo+" --reinstall-service --dry-run")
	}
	if !observed.Labels.Match {
		add("label_drift", SeverityWarning, "labels", labelEvidence(observed.Labels), recoverReregister)
	}
	if observed.Provider.Kind != "" && observed.Provider.Kind != "byo" {
		providerCleanup := "Run runnerkit destroy --repo " + repo + " --dry-run to review billable resources before cleanup."
		if observed.Provider.Error != "" {
			add("provider_error", SeverityWarning, "provider", observed.Provider.Error, providerCleanup)
		} else if !observed.Provider.Found {
			add("provider_resource_missing", SeverityWarning, "provider", "provider resource is missing", providerCleanup)
		} else if len(observed.Provider.Drift) > 0 {
			add("provider_drift", SeverityWarning, "provider", strings.Join(observed.Provider.Drift, ", "), providerCleanup)
		} else {
			add("provider_found", SeverityPass, "provider", observed.Provider.Kind+" resources are present", "runnerkit status --repo "+repo)
		}
	}
	if !checks.InstallPathOK {
		evidence := defaultEvidence(checks.InstallPathError, "install path check failed")
		add("install_path_missing", SeverityError, "remote", evidence, recoverReregister)
	}
	if !checks.WorkDirOK {
		evidence := defaultEvidence(checks.WorkDirError, "work dir check failed")
		add("work_dir_missing", SeverityWarning, "remote", evidence, recoverReregister)
	}
	for _, result := range checks.Preflight.Results {
		switch result.ID {
		case preflight.CheckDisk:
			if result.Severity != preflight.SeverityPass {
				add("disk_low", SeverityWarning, "preflight", result.Message, "Free disk space under /opt and /var/lib before restarting the runner.")
			}
		case preflight.CheckTools:
			if result.Severity != preflight.SeverityPass {
				add("tools_missing", SeverityWarning, "preflight", result.Message, "Install missing tools or re-run runnerkit up after fixing preflight.")
			}
		case preflight.CheckNetworkGitHub:
			if result.Severity != preflight.SeverityPass {
				add("network_github_failed", SeverityError, "preflight", result.Message, "Allow HTTPS egress to https://github.com and https://api.github.com.")
			}
		case preflight.CheckTime:
			if result.Severity != preflight.SeverityPass {
				add("time_unsynchronized", SeverityWarning, "preflight", result.Message, "Enable NTP/time sync if TLS or token expiry errors occur.")
			}
		}
	}
	if len(repoState.Cleanup.Notes) > 0 || len(repoState.Operations) > 0 {
		add("cleanup_pending", SeverityWarning, "state", "cleanup checkpoints or notes are pending", downDryRun)
	}
	// Ephemeral lifecycle findings: surface waiting/busy/completed/
	// ttl_expired/cleanup_pending so doctor reports the same vocabulary
	// as status without re-running the live remote probes.
	if repoState.Runner.Mode == "ephemeral" {
		cleanup := repoState.Ephemeral.CleanupCommand
		if cleanup == "" {
			cleanup = downDryRun
		}
		switch {
		case hasEphemeralCleanupPending(repoState.Operations, repoState.Cleanup.Notes):
			add(ReasonEphemeralCleanupPending, SeverityWarning, "ephemeral", "ephemeral cleanup checkpoints are pending", cleanup)
		case ttlExpired(&repoState) && repoState.Ephemeral.FinalizerStatus != "completed":
			add(ReasonEphemeralTTLExpired, SeverityWarning, "ephemeral", "ephemeral TTL safeguard expired before completion", cleanup)
		case !observed.GitHub.Found && repoState.Ephemeral.FinalizerStatus == "completed":
			add(ReasonEphemeralCompleted, SeverityPass, "ephemeral", "ephemeral runner finalized after one job", cleanup)
		case observed.GitHub.Busy:
			add(ReasonEphemeralBusy, SeverityPass, "ephemeral", "ephemeral runner is running its one allowed job", cleanup)
		case observed.GitHub.Found:
			add(ReasonEphemeralWaiting, SeverityPass, "ephemeral", "ephemeral runner is online and waiting for its one job", cleanup)
		}
	}
	add("logs_available", SeverityPass, "logs", "bounded systemd journal and runner diag collection is available", logsCmd)
	return report
}

func defaultEvidence(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
