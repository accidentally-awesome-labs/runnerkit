package ops

import (
	"fmt"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/errcodes"
	"github.com/accidentally-awesome-labs/runnerkit/internal/preflight"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
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
	// BYOHostPrepared is true when /etc/sudoers.d/runnerkit-installer
	// was observed on the remote host (Plan 06-06 Path C applied).
	// Surfaces as the informational `byo_host_prepared` finding.
	BYOHostPrepared bool
}

func BuildDoctorReport(repoState state.RepositoryState, observed ObservedRunner, checks DeepChecks) DoctorReport {
	health := Classify(observed)
	repo := repoState.Repo.FullName
	report := DoctorReport{Repo: repo, StatePath: observed.StatePath, Health: health, Findings: []Finding{}, NextActions: health.NextActions}
	add := func(id string, severity Severity, source string, evidence string, remediation string) {
		report.Findings = append(report.Findings, Finding{ID: id, Severity: string(severity), Source: source, Evidence: evidence, Remediation: remediation})
	}
	// addWithCode is a thin wrapper that appends the canonical
	// `See: <URL>` line for a registered errcodes.Code to the
	// remediation string (D-15). Pass-only findings keep using `add`.
	addWithCode := func(id string, code errcodes.Code, severity Severity, source string, evidence string, remediation string) {
		report.Findings = append(report.Findings, Finding{
			ID:          id,
			Severity:    string(severity),
			Source:      source,
			Evidence:    evidence,
			Remediation: remediation + "\n\nSee: " + errcodes.URL(code),
		})
	}
	statusCmd := "runnerkit status --repo " + repo
	logsCmd := "runnerkit logs --repo " + repo + " --since 30m"
	recoverReregister := "runnerkit recover --repo " + repo + " --reregister --dry-run"
	downDryRun := "runnerkit down --repo " + repo + " --dry-run"
	add("state_present", SeverityPass, "state", "local RunnerKit state is present", statusCmd)
	if observed.GitHub.Found {
		add("github_runner_found", SeverityPass, "github", fmt.Sprintf("GitHub runner %s id %d is present", observed.GitHub.Name, observed.GitHub.ID), statusCmd)
		if strings.EqualFold(observed.GitHub.Status, "offline") {
			addWithCode("github_runner_offline", errcodes.GHRunnerOffline, SeverityWarning, "github", "GitHub reports runner status offline", logsCmd)
		}
	} else if len(observed.GitHub.DuplicateCandidates) > 1 {
		addWithCode("github_duplicate_candidates", errcodes.GHDuplicateCandidates, SeverityError, "github", "multiple RunnerKit runner candidates found", downDryRun)
	} else {
		addWithCode("github_runner_offline", errcodes.GHRunnerOffline, SeverityWarning, "github", "GitHub runner is missing or offline", logsCmd)
	}
	if !observed.SSH.Reachable {
		if observed.SSH.HostKey == "mismatch" {
			addWithCode("ssh_host_key_mismatch", errcodes.SSHHostKeyMismatch, SeverityError, "ssh", "saved host key fingerprint does not match observed host", "Verify the machine identity before running runnerkit recover --repo "+repo+".")
		} else {
			addWithCode("ssh_unreachable", errcodes.SSHUnreachable, SeverityError, "ssh", "SSH is unreachable", "Verify SSH access to "+repoState.Machine.HostRef+", then re-run runnerkit doctor --repo "+repo+".")
		}
	}
	if observed.Service.ActiveState == "active" {
		add("service_active", SeverityPass, "systemd", "systemd reports ActiveState=active", statusCmd)
	} else if serviceFailed(observed.Service) {
		addWithCode("service_failed", errcodes.BootServiceFailed, SeverityError, "systemd", "systemd reports ActiveState=failed for runnerkit-runner.", logsCmd)
	} else if observed.Service.LoadState == "not-found" || strings.Contains(strings.ToLower(observed.Service.Error), "missing") {
		addWithCode("service_missing", errcodes.BootServiceMissing, SeverityError, "systemd", "saved systemd service is missing", "runnerkit recover --repo "+repo+" --reinstall-service --dry-run")
	}
	if !observed.Labels.Match {
		addWithCode("label_drift", errcodes.GHLabelDrift, SeverityWarning, "labels", labelEvidence(observed.Labels), recoverReregister)
	}
	if observed.Provider.Kind != "" && observed.Provider.Kind != "byo" {
		providerCleanup := "Run runnerkit destroy --repo " + repo + " --dry-run to review billable resources before cleanup."
		if observed.Provider.Error != "" {
			addWithCode("provider_error", errcodes.ProvProviderError, SeverityWarning, "provider", observed.Provider.Error, providerCleanup)
		} else if !observed.Provider.Found {
			addWithCode("provider_resource_missing", errcodes.ProvResourceMissing, SeverityWarning, "provider", "provider resource is missing", providerCleanup)
		} else if len(observed.Provider.Drift) > 0 {
			addWithCode("provider_drift", errcodes.ProvDrift, SeverityWarning, "provider", strings.Join(observed.Provider.Drift, ", "), providerCleanup)
		} else {
			add("provider_found", SeverityPass, "provider", observed.Provider.Kind+" resources are present", "runnerkit status --repo "+repo)
		}
	}
	if !checks.InstallPathOK {
		evidence := defaultEvidence(checks.InstallPathError, "install path check failed")
		addWithCode("install_path_missing", errcodes.BootInstallPathMissing, SeverityError, "remote", evidence, recoverReregister)
	}
	if !checks.WorkDirOK {
		evidence := defaultEvidence(checks.WorkDirError, "work dir check failed")
		addWithCode("work_dir_missing", errcodes.BootWorkDirMissing, SeverityWarning, "remote", evidence, recoverReregister)
	}
	for _, result := range checks.Preflight.Results {
		switch result.ID {
		case preflight.CheckDisk:
			if result.Severity != preflight.SeverityPass {
				addWithCode("disk_low", errcodes.BootDiskLow, SeverityWarning, "preflight", result.Message, "Free disk space under /opt and /var/lib before restarting the runner.")
			}
		case preflight.CheckTools:
			if result.Severity != preflight.SeverityPass {
				addWithCode("tools_missing", errcodes.BootToolsMissing, SeverityWarning, "preflight", result.Message, "Install missing tools or re-run runnerkit up after fixing preflight.")
			}
		case preflight.CheckNetworkGitHub:
			if result.Severity != preflight.SeverityPass {
				addWithCode("network_github_failed", errcodes.AuthNetworkGitHubFailed, SeverityError, "preflight", result.Message, "Allow HTTPS egress to https://github.com and https://api.github.com.")
			}
		case preflight.CheckTime:
			if result.Severity != preflight.SeverityPass {
				addWithCode("time_unsynchronized", errcodes.BootTimeUnsynchronized, SeverityWarning, "preflight", result.Message, "Enable NTP/time sync if TLS or token expiry errors occur.")
			}
		}
	}
	if len(repoState.Cleanup.Notes) > 0 || len(repoState.Operations) > 0 {
		addWithCode("cleanup_pending", errcodes.CleanCleanupPending, SeverityWarning, "state", "cleanup checkpoints or notes are pending", downDryRun)
	}
	// Stale runner version: when the saved RunnerTemplateVersion is older
	// than the bundled `bootstrap.RunnerVersion`, surface a warning that
	// points at `runnerkit upgrade-runner` (D-08). Plan 06-03 will map this
	// finding ID to RKD-BOOT-002 via the errcodes package; here we keep the
	// snake_case finding ID consistent with the existing convention.
	if observedPin := repoState.RunnerTemplateVersion; observedPin != "" && observedPin != bootstrap.RunnerVersion {
		addWithCode("runner_version_stale", errcodes.BootRunnerVersionStale, SeverityWarning, "bootstrap",
			fmt.Sprintf("installed runner version %s is older than bundled pin %s", observedPin, bootstrap.RunnerVersion),
			"runnerkit upgrade-runner --repo "+repo)
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
			addWithCode(ReasonEphemeralCleanupPending, errcodes.CleanEphemeralCleanupPending, SeverityWarning, "ephemeral", "ephemeral cleanup checkpoints are pending", cleanup)
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
	// Plan 06-06: surface whether the host has been prepared via
	// one-time host install. Severity is pass/info — the
	// absence is not an error since Path B (interactive prompt) is a
	// valid alternative.
	if checks.BYOHostPrepared {
		add("byo_host_prepared", SeverityPass, "remote", bootstrap.SudoersFilePath+" exists on remote host (runnerkit host install applied)", "sudo rm -f "+bootstrap.SudoersFilePath+" # on host to revert")
	}
	return report
}

func defaultEvidence(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
