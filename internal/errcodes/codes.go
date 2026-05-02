// Package errcodes is the single source of truth for stable RKD-<COMPONENT>-NNN
// error codes. Every user-facing failure and every `runnerkit doctor` finding
// resolves to a Code in this Registry; the Code's URL() points at a stable
// HTML anchor in docs/troubleshooting/<component>.md.
//
// Numbering rules (CONTEXT D-15):
//   - Codes are STABLE across renames. Never renumber.
//   - Numbering starts at 001 per component and grows monotonically.
//   - To deprecate a code, leave it in the registry with a "deprecated" note;
//     do NOT delete or renumber. Add a new code for the replacement and link
//     to it from the deprecated entry's troubleshooting section.
//
// Components (D-15):
//
//	AUTH  — GitHub auth, registration token, public-repo block
//	SSH   — host-key, key path, port, dial
//	BOOT  — runner user create, package install, online verification, runner version stale
//	GH    — runner registration, deregister stale, runner offline, label drift
//	PROV  — Hetzner token, quota, region, partial destroy, billable lingering
//	CLEAN — down/destroy partial, ephemeral log preserve
//	STATE — JSON read, schema-too-new, migration, atomic write
//	CORE  — CLI shell errors (input required, invalid input)
package errcodes

// Severity follows internal/ops.Severity values; lowercased here to keep the
// errcodes package zero-dep on internal/ops (avoiding import cycles).
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Code is a single stable error/finding identifier with its docs anchor.
type Code struct {
	ID       string // e.g. "RKD-AUTH-001"
	Severity Severity
	Title    string // short, human-readable
	File     string // markdown file under docs/troubleshooting/, e.g. "auth.md"
	Anchor   string // explicit anchor in that file, e.g. "rkd-auth-001"
}

// AuthPublicRepoBlocked et al — typed handles for every emit site.
// Keep ordered by component prefix, then number.
var (
	// AUTH
	AuthPublicRepoBlocked                = Code{ID: "RKD-AUTH-001", Severity: SeverityError, Title: "Persistent runner on public repository is blocked", File: "auth.md", Anchor: "rkd-auth-001"}
	AuthNetworkGitHubFailed              = Code{ID: "RKD-AUTH-002", Severity: SeverityError, Title: "Cannot reach github.com / api.github.com", File: "auth.md", Anchor: "rkd-auth-002"}
	AuthEphemeralBYOPublicForkAck        = Code{ID: "RKD-AUTH-003", Severity: SeverityError, Title: "Ephemeral BYO on public/fork repo requires acknowledgment", File: "auth.md", Anchor: "rkd-auth-003"}
	AuthRunnerManagementPermissionDenied = Code{ID: "RKD-AUTH-004", Severity: SeverityError, Title: "Token lacks runner-management permission", File: "auth.md", Anchor: "rkd-auth-004"}

	// SSH
	SSHHostKeyMismatch = Code{ID: "RKD-SSH-001", Severity: SeverityError, Title: "SSH host key fingerprint mismatch", File: "ssh.md", Anchor: "rkd-ssh-001"}
	SSHUnreachable     = Code{ID: "RKD-SSH-002", Severity: SeverityError, Title: "SSH host unreachable", File: "ssh.md", Anchor: "rkd-ssh-002"}
	SSHKeyPathNotFound = Code{ID: "RKD-SSH-003", Severity: SeverityError, Title: "SSH private key file not found", File: "ssh.md", Anchor: "rkd-ssh-003"}
	SSHPortUnreachable = Code{ID: "RKD-SSH-004", Severity: SeverityError, Title: "SSH port unreachable", File: "ssh.md", Anchor: "rkd-ssh-004"}

	// BOOT (RKD-BOOT-001 reserved for future use; numbering starts at 002).
	BootRunnerVersionStale         = Code{ID: "RKD-BOOT-002", Severity: SeverityWarning, Title: "Bundled runner pin is newer than installed runner", File: "bootstrap.md", Anchor: "rkd-boot-002"}
	BootServiceFailed              = Code{ID: "RKD-BOOT-003", Severity: SeverityError, Title: "systemd service failed", File: "bootstrap.md", Anchor: "rkd-boot-003"}
	BootServiceMissing             = Code{ID: "RKD-BOOT-004", Severity: SeverityError, Title: "systemd service missing", File: "bootstrap.md", Anchor: "rkd-boot-004"}
	BootInstallPathMissing         = Code{ID: "RKD-BOOT-005", Severity: SeverityError, Title: "Runner install directory missing on host", File: "bootstrap.md", Anchor: "rkd-boot-005"}
	BootWorkDirMissing             = Code{ID: "RKD-BOOT-006", Severity: SeverityWarning, Title: "Runner work directory missing on host", File: "bootstrap.md", Anchor: "rkd-boot-006"}
	BootDiskLow                    = Code{ID: "RKD-BOOT-007", Severity: SeverityWarning, Title: "Disk space low under /opt or /var/lib", File: "bootstrap.md", Anchor: "rkd-boot-007"}
	BootToolsMissing               = Code{ID: "RKD-BOOT-008", Severity: SeverityWarning, Title: "Required CLI tools missing on host", File: "bootstrap.md", Anchor: "rkd-boot-008"}
	BootTimeUnsynchronized         = Code{ID: "RKD-BOOT-009", Severity: SeverityWarning, Title: "Host clock not synchronized (NTP)", File: "bootstrap.md", Anchor: "rkd-boot-009"}
	BootPreflightUnsupportedDistro = Code{ID: "RKD-BOOT-010", Severity: SeverityWarning, Title: "Linux distribution not in supported matrix", File: "bootstrap.md", Anchor: "rkd-boot-010"}
	BootPreflightFailed            = Code{ID: "RKD-BOOT-011", Severity: SeverityError, Title: "Preflight check failed", File: "bootstrap.md", Anchor: "rkd-boot-011"}
	BootRunnerUserCreateFailed     = Code{ID: "RKD-BOOT-012", Severity: SeverityError, Title: "runnerkit-runner user creation failed", File: "bootstrap.md", Anchor: "rkd-boot-012"}
	BootRunnerPackageInstallFailed = Code{ID: "RKD-BOOT-013", Severity: SeverityError, Title: "Runner tarball install failed", File: "bootstrap.md", Anchor: "rkd-boot-013"}
	BootRunnerOnlineVerifyTimeout  = Code{ID: "RKD-BOOT-014", Severity: SeverityError, Title: "Runner did not report online before timeout", File: "bootstrap.md", Anchor: "rkd-boot-014"}

	// GH
	GHRunnerOffline                 = Code{ID: "RKD-GH-001", Severity: SeverityWarning, Title: "GitHub reports runner offline", File: "github.md", Anchor: "rkd-gh-001"}
	GHDuplicateCandidates           = Code{ID: "RKD-GH-002", Severity: SeverityError, Title: "Multiple RunnerKit runner candidates found in GitHub", File: "github.md", Anchor: "rkd-gh-002"}
	GHLabelDrift                    = Code{ID: "RKD-GH-003", Severity: SeverityWarning, Title: "Saved labels drift from GitHub-reported labels", File: "github.md", Anchor: "rkd-gh-003"}
	GHRegistrationTokenCreateFailed = Code{ID: "RKD-GH-004", Severity: SeverityError, Title: "Failed to create runner registration token", File: "github.md", Anchor: "rkd-gh-004"}
	GHRunnerRegisterFailed          = Code{ID: "RKD-GH-005", Severity: SeverityError, Title: "Runner registration failed", File: "github.md", Anchor: "rkd-gh-005"}
	GHDeregisterStaleFailed         = Code{ID: "RKD-GH-006", Severity: SeverityWarning, Title: "Stale GitHub runner deregistration failed", File: "github.md", Anchor: "rkd-gh-006"}
	GHRecoverReregisterFailed       = Code{ID: "RKD-GH-007", Severity: SeverityError, Title: "recover --reregister failed", File: "github.md", Anchor: "rkd-gh-007"}

	// PROV
	ProvProviderError             = Code{ID: "RKD-PROV-001", Severity: SeverityWarning, Title: "Hetzner provider returned error during status", File: "provider.md", Anchor: "rkd-prov-001"}
	ProvResourceMissing           = Code{ID: "RKD-PROV-002", Severity: SeverityWarning, Title: "Hetzner resource missing for saved IDs", File: "provider.md", Anchor: "rkd-prov-002"}
	ProvDrift                     = Code{ID: "RKD-PROV-003", Severity: SeverityWarning, Title: "Hetzner inventory drift from saved state", File: "provider.md", Anchor: "rkd-prov-003"}
	ProvHCloudTokenMissing        = Code{ID: "RKD-PROV-004", Severity: SeverityError, Title: "HCLOUD_TOKEN environment variable not set", File: "provider.md", Anchor: "rkd-prov-004"}
	ProvHCloudQuotaExceeded       = Code{ID: "RKD-PROV-005", Severity: SeverityError, Title: "Hetzner project quota exceeded", File: "provider.md", Anchor: "rkd-prov-005"}
	ProvHCloudPartialDestroy      = Code{ID: "RKD-PROV-006", Severity: SeverityWarning, Title: "Hetzner partial destroy — resources remain", File: "provider.md", Anchor: "rkd-prov-006"}
	ProvBillableResourceLingering = Code{ID: "RKD-PROV-007", Severity: SeverityError, Title: "Hetzner resource still billable after destroy", File: "provider.md", Anchor: "rkd-prov-007"}

	// CLEAN
	CleanCleanupPending             = Code{ID: "RKD-CLEAN-001", Severity: SeverityWarning, Title: "Cleanup checkpoints or notes are pending", File: "cleanup.md", Anchor: "rkd-clean-001"}
	CleanEphemeralCleanupPending    = Code{ID: "RKD-CLEAN-002", Severity: SeverityWarning, Title: "Ephemeral cleanup checkpoints are pending", File: "cleanup.md", Anchor: "rkd-clean-002"}
	CleanDownFilesRemoveFailed      = Code{ID: "RKD-CLEAN-003", Severity: SeverityError, Title: "down: file removal failed", File: "cleanup.md", Anchor: "rkd-clean-003"}
	CleanEphemeralLogPreserveFailed = Code{ID: "RKD-CLEAN-004", Severity: SeverityWarning, Title: "Ephemeral log preservation failed", File: "cleanup.md", Anchor: "rkd-clean-004"}
	CleanDestroyPartial             = Code{ID: "RKD-CLEAN-005", Severity: SeverityWarning, Title: "destroy: partial cleanup, checkpoints retained", File: "cleanup.md", Anchor: "rkd-clean-005"}

	// STATE — for STATE codes the docs entry lives in cleanup.md (state failures
	// are ultimately cleanup/recovery operations from the user's perspective).
	StateInvalidJSON       = Code{ID: "RKD-STATE-001", Severity: SeverityError, Title: "state.json is not valid JSON", File: "cleanup.md", Anchor: "rkd-state-001"}
	StateBackupWriteFailed = Code{ID: "RKD-STATE-002", Severity: SeverityError, Title: "state backup write failed", File: "cleanup.md", Anchor: "rkd-state-002"}
	StateMigrationFailed   = Code{ID: "RKD-STATE-003", Severity: SeverityError, Title: "state migration failed", File: "cleanup.md", Anchor: "rkd-state-003"}
	StateSchemaTooNew      = Code{ID: "RKD-STATE-004", Severity: SeverityError, Title: "state schema_version newer than this CLI knows", File: "cleanup.md", Anchor: "rkd-state-004"}

	// CORE — for CORE codes the docs entry lives in cleanup.md (catch-all index).
	CoreInputRequired = Code{ID: "RKD-CORE-001", Severity: SeverityError, Title: "Input required for non-interactive flow", File: "cleanup.md", Anchor: "rkd-core-001"}
	CoreInvalidInput  = Code{ID: "RKD-CORE-002", Severity: SeverityError, Title: "Invalid CLI input", File: "cleanup.md", Anchor: "rkd-core-002"}
)

// Registry is the slice of every Code defined in this package. Tests walk
// this slice to verify (a) uniqueness, (b) docs anchor presence,
// (c) component file coverage, (d) Symptom/Diagnosis/Fix structure.
var Registry = []Code{
	AuthPublicRepoBlocked, AuthNetworkGitHubFailed, AuthEphemeralBYOPublicForkAck, AuthRunnerManagementPermissionDenied,
	SSHHostKeyMismatch, SSHUnreachable, SSHKeyPathNotFound, SSHPortUnreachable,
	BootRunnerVersionStale, BootServiceFailed, BootServiceMissing, BootInstallPathMissing, BootWorkDirMissing,
	BootDiskLow, BootToolsMissing, BootTimeUnsynchronized, BootPreflightUnsupportedDistro, BootPreflightFailed,
	BootRunnerUserCreateFailed, BootRunnerPackageInstallFailed, BootRunnerOnlineVerifyTimeout,
	GHRunnerOffline, GHDuplicateCandidates, GHLabelDrift, GHRegistrationTokenCreateFailed, GHRunnerRegisterFailed,
	GHDeregisterStaleFailed, GHRecoverReregisterFailed,
	ProvProviderError, ProvResourceMissing, ProvDrift, ProvHCloudTokenMissing, ProvHCloudQuotaExceeded,
	ProvHCloudPartialDestroy, ProvBillableResourceLingering,
	CleanCleanupPending, CleanEphemeralCleanupPending, CleanDownFilesRemoveFailed, CleanEphemeralLogPreserveFailed, CleanDestroyPartial,
	StateInvalidJSON, StateBackupWriteFailed, StateMigrationFailed, StateSchemaTooNew,
	CoreInputRequired, CoreInvalidInput,
}
