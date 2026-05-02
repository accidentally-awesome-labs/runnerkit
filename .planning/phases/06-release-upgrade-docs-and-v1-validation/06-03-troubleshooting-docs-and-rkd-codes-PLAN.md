---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 03
type: execute
wave: B
depends_on: [02]
files_modified:
  - internal/errcodes/codes.go
  - internal/errcodes/codes_test.go
  - internal/errcodes/url.go
  - docs/troubleshooting/README.md
  - docs/troubleshooting/auth.md
  - docs/troubleshooting/ssh.md
  - docs/troubleshooting/bootstrap.md
  - docs/troubleshooting/github.md
  - docs/troubleshooting/provider.md
  - docs/troubleshooting/cleanup.md
  - internal/ops/doctor.go
  - internal/cli/up.go
  - internal/cli/down.go
  - internal/cli/destroy.go
  - internal/cli/recover.go
  - README.md
  - docs/byo-quickstart.md
  - docs/cloud-quickstart.md
  - docs/safety.md
autonomous: true
requirements: [DOC-04]
must_haves:
  truths:
    - "Every CLI-emitted user-facing failure or doctor finding can be traced to a stable RKD-<COMPONENT>-NNN code defined in `internal/errcodes/codes.go`."
    - "Every RKD code in the registry resolves to a real HTML anchor `<a name=\"rkd-component-NNN\"></a>` in the matching `docs/troubleshooting/<component>.md` file (verified by automated test)."
    - "URL builder honors `RUNNERKIT_DOCS_BASE` env override, defaulting to `https://github.com/salar/runnerkit/blob/main/docs` (verified by automated test)."
    - "All four failure surfaces from D-16 (setup, bootstrap+service, operations, cloud+cleanup) have at least one entry per component file."
    - "Every entry in the six troubleshooting component files follows the literal `### Symptom` / `### Diagnosis` / `### Fix` heading structure (D-17)."
  artifacts:
    - path: "internal/errcodes/codes.go"
      provides: "Stable RKD code registry covering all 8 component prefixes (AUTH, SSH, BOOT, GH, PROV, CLEAN, STATE, CORE)"
      contains: "type Code"
      contains_also: "RKD-AUTH-"
      contains_also2: "RKD-BOOT-"
      contains_also3: "RKD-STATE-"
    - path: "internal/errcodes/url.go"
      provides: "URL builder honoring RUNNERKIT_DOCS_BASE env override"
      contains: "RUNNERKIT_DOCS_BASE"
      contains_also: "github.com/salar/runnerkit/blob/main/docs"
    - path: "internal/errcodes/codes_test.go"
      provides: "Five required tests: TestEveryCodeHasDocAnchor, TestCodesAreUnique, TestURL_RespectsEnvOverride, TestEachComponentHasMinimumOneEntry, TestEntriesFollowSymptomDiagnosisFix"
      contains: "TestEveryCodeHasDocAnchor"
      contains_also: "TestCodesAreUnique"
      contains_also2: "TestURL_RespectsEnvOverride"
    - path: "docs/troubleshooting/README.md"
      provides: "Index of components + global error-code table + install verification snippet"
      contains: "RKD-AUTH"
      contains_also: "cosign verify-blob"
      contains_also2: "xattr -d com.apple.quarantine"
    - path: "docs/troubleshooting/auth.md"
      provides: "RKD-AUTH-NNN entries: GitHub auth scope, public-repo block, registration token"
      contains: "<a name=\"rkd-auth-001\"></a>"
      contains_also: "### Symptom"
      contains_also2: "### Diagnosis"
      contains_also3: "### Fix"
    - path: "docs/troubleshooting/bootstrap.md"
      provides: "RKD-BOOT-NNN entries including RKD-BOOT-002 (runner version stale, wired to Plan 06-02 doctor finding)"
      contains: "<a name=\"rkd-boot-002\"></a>"
      contains_also: "runnerkit upgrade-runner"
  key_links:
    - from: "internal/errcodes/codes.go::Code{ID, File, Anchor}"
      to: "docs/troubleshooting/<file>.md::<a name=\"<anchor>\">"
      via: "TestEveryCodeHasDocAnchor walks the docs dir and asserts every code's (File, Anchor) resolves"
    - from: "internal/errcodes/url.go::URL(code)"
      to: "RUNNERKIT_DOCS_BASE env or default github blob URL"
      via: "os.Getenv(\"RUNNERKIT_DOCS_BASE\")"
    - from: "internal/ops/doctor.go::BuildDoctorReport finding `runner_version_stale`"
      to: "RKD-BOOT-002 in errcodes registry"
      via: "remediation field includes errcodes.URL(BootRunnerVersionStale)"
      pattern: "errcodes\\.URL"
    - from: "internal/cli/up.go (public-repo block, persistent-on-public refusal)"
      to: "RKD-AUTH-001 / errcodes.URL"
      via: "user-facing error message includes the See: <URL> line"
---

<objective>
Land DOC-04 (cleanup and troubleshooting guidance) by: (a) creating the `internal/errcodes/` package as the single source of truth for stable `RKD-<COMPONENT>-NNN` error codes; (b) creating six `docs/troubleshooting/<component>.md` files plus a README index covering all four failure surfaces from D-16 with the Symptom/Diagnosis/Fix structure from D-17; (c) wiring CLI emit sites to call `errcodes.URL(...)` so every user-facing failure prints a stable, env-overridable doc URL; and (d) cross-linking from README and the existing quickstart/safety docs.

Implements **D-14..D-17** from CONTEXT.md. Closes the DOC-04 binding requirement.

Purpose: Phase 6 success criterion 3 — "Developer can read cleanup and troubleshooting guidance for common setup, runner, GitHub, SSH, provider, and cleanup failures."

Output: A repo where every user-facing failure has a stable RKD code, an anchored entry in `docs/troubleshooting/`, and a deterministic URL the CLI prints alongside the failure.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/REQUIREMENTS.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VALIDATION.md
@internal/ops/doctor.go
@internal/cli/up.go
@internal/cli/down.go
@internal/cli/destroy.go

<interfaces>
<!-- Existing finding IDs in internal/ops/doctor.go (extracted 2026-05-02). These are the SEED list for the RKD code registry — every existing finding gets a stable RKD code. -->

Existing doctor finding IDs (from internal/ops/doctor.go::BuildDoctorReport):
- state_present (pass) → no code needed (pass-only is informational)
- github_runner_found (pass) → no code (informational)
- github_runner_offline (warning) → RKD-GH-001
- github_duplicate_candidates (error) → RKD-GH-002
- ssh_host_key_mismatch (error) → RKD-SSH-001
- ssh_unreachable (error) → RKD-SSH-002
- service_active (pass) → no code
- service_failed (error) → RKD-BOOT-003
- service_missing (error) → RKD-BOOT-004
- label_drift (warning) → RKD-GH-003
- provider_error (warning) → RKD-PROV-001
- provider_resource_missing (warning) → RKD-PROV-002
- provider_drift (warning) → RKD-PROV-003
- provider_found (pass) → no code
- install_path_missing (error) → RKD-BOOT-005
- work_dir_missing (warning) → RKD-BOOT-006
- disk_low (warning) → RKD-BOOT-007
- tools_missing (warning) → RKD-BOOT-008
- network_github_failed (error) → RKD-AUTH-002
- time_unsynchronized (warning) → RKD-BOOT-009
- cleanup_pending (warning) → RKD-CLEAN-001
- ephemeral_cleanup_pending (warning, Phase 5) → RKD-CLEAN-002
- runner_version_stale (warning, Plan 06-02 NEW) → RKD-BOOT-002

Additional CLI emit-site failures from Phases 1–5 (canonical RKD numbering for this plan):
- public_repo_persistent_blocked (Phase 1/2 safety gate) → RKD-AUTH-001
- ephemeral_byo_public_fork_acknowledgment_required (Phase 5) → RKD-AUTH-003
- runner_management_permission_denied (Phase 1) → RKD-AUTH-004
- ssh_key_path_not_found (Phase 2) → RKD-SSH-003
- ssh_port_unreachable (Phase 2) → RKD-SSH-004
- preflight_unsupported_distro (Phase 2) → RKD-BOOT-010
- preflight_failed (Phase 2 generic) → RKD-BOOT-011
- runner_user_create_failed (Phase 2) → RKD-BOOT-012
- runner_package_install_failed (Phase 2) → RKD-BOOT-013
- runner_online_verification_timeout (Phase 2) → RKD-BOOT-014
- registration_token_create_failed (Phase 1/2) → RKD-GH-004
- runner_register_failed (Phase 2) → RKD-GH-005
- deregister_stale_failed (Phase 3) → RKD-GH-006
- recover_reregister_failed (Phase 3) → RKD-GH-007
- hcloud_token_missing (Phase 4) → RKD-PROV-004
- hcloud_quota_exceeded (Phase 4) → RKD-PROV-005
- hcloud_partial_destroy (Phase 4) → RKD-PROV-006
- destroy_billable_resource_lingering (Phase 4 / 06-04 D-12 gate 2) → RKD-PROV-007
- down_files_remove_failed (Phase 3) → RKD-CLEAN-003
- ephemeral_log_preservation_failed (Phase 5) → RKD-CLEAN-004
- destroy_partial (Phase 4) → RKD-CLEAN-005
- state_invalid_json (06-02) → RKD-STATE-001
- state_backup_write_failed (06-02) → RKD-STATE-002
- state_migration_failed (06-02) → RKD-STATE-003
- state_schema_too_new (06-02 ErrSchemaTooNew) → RKD-STATE-004
- input_required (Phase 1 ExitInputRequired) → RKD-CORE-001
- invalid_input (Phase 1 ExitInvalidInput) → RKD-CORE-002

Cobra command tree state-dir convention (internal/state/store.go::DefaultBaseDir):
- $XDG_STATE_HOME/runnerkit/ if set, else $HOME/.local/state/runnerkit/

Existing CLI failure print pattern (grep '^See: ' across internal/cli/*.go produced ZERO results — meaning RKD URL emission is a NEW convention this plan introduces; it is NOT a refactor of existing hardcoded URLs).

Coordination with Plan 06-02:
- Plan 06-02 ADDS the `runner_version_stale` finding ID to internal/ops/doctor.go. THIS plan (06-03) ADDS the call to `errcodes.URL(...)` in the remediation field of all relevant findings (including `runner_version_stale`). The two changes touch different lines of the same function and do NOT collide if Plan 06-02 lands first OR they merge in a single rebase. Recommended ordering: 06-02 lands first; this plan's Task 4 then layers the errcodes.URL calls on top.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: internal/errcodes/ package — code registry, URL builder honoring RUNNERKIT_DOCS_BASE, all 5 required tests</name>
  <files>internal/errcodes/codes.go, internal/errcodes/url.go, internal/errcodes/codes_test.go</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (Pattern 8 — RKD code registry, full Go sketch; Pitfall 9 — Markdown auto-anchors; D-15 numbering convention)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-15, D-16, D-17)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VALIDATION.md (lines 69-73: 5 required test names)
    - The `<interfaces>` block above for the canonical 32-code registry seed list
  </read_first>
  <behavior>
    - Test 1: `TestEveryCodeHasDocAnchor` — walk every Code in `Registry`, open `docs/troubleshooting/<File>` (Code.File is e.g., "auth.md"), grep for the literal `<a name="<Anchor>"></a>` (Code.Anchor is e.g., "rkd-auth-001"). Every code in the registry MUST have a matching anchor in the matching file. Any miss = test fails listing the missing codes.
    - Test 2: `TestCodesAreUnique` — assert no two Codes in `Registry` share the same `ID`. Also assert no two share the same `(File, Anchor)` pair.
    - Test 3: `TestURL_RespectsEnvOverride` — `t.Setenv("RUNNERKIT_DOCS_BASE", "https://runnerkit.dev/docs")`; assert `URL(AuthPublicRepoBlocked) == "https://runnerkit.dev/docs/troubleshooting/auth#rkd-auth-001"`. Then unset; assert default `URL(AuthPublicRepoBlocked) == "https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-001"`. NOTE the GitHub blob URL keeps the `.md` suffix BEFORE the anchor; the runnerkit.dev variant strips it. Implement the URL builder to handle both shapes (GitHub blob URLs require the `.md` extension; static-site URLs do not).
    - Test 4: `TestEachComponentHasMinimumOneEntry` — assert each of the 6 component files (`auth.md`, `ssh.md`, `bootstrap.md`, `github.md`, `provider.md`, `cleanup.md`) contains at least one Code from `Registry` whose `File` matches.
    - Test 5: `TestEntriesFollowSymptomDiagnosisFix` — for each Code in `Registry`, open `docs/troubleshooting/<File>`, locate the section starting at `<a name="<Anchor>">` and assert the section before the next `<a name=` boundary contains all three literal headings: `### Symptom`, `### Diagnosis`, `### Fix`. (Use a simple state-machine scan over the file lines.)
  </behavior>
  <action>
**Step 1: Create `internal/errcodes/codes.go`** with the registry. Use the 32-code list from `<interfaces>` above. Structure:

```go
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
//   AUTH  — GitHub auth, registration token, public-repo block
//   SSH   — host-key, key path, port, dial
//   BOOT  — runner user create, package install, online verification, runner version stale
//   GH    — runner registration, deregister stale, runner offline, label drift
//   PROV  — Hetzner token, quota, region, partial destroy, billable lingering
//   CLEAN — down/destroy partial, ephemeral log preserve
//   STATE — JSON read, schema-too-new, migration, atomic write
//   CORE  — CLI shell errors (input required, invalid input)
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
    ID       string   // e.g. "RKD-AUTH-001"
    Severity Severity
    Title    string   // short, human-readable
    File     string   // markdown file under docs/troubleshooting/, e.g. "auth.md"
    Anchor   string   // explicit anchor in that file, e.g. "rkd-auth-001"
}

// AuthPublicRepoBlocked et al — typed handles for every emit site.
// Keep ordered by component prefix, then number.
var (
    // AUTH
    AuthPublicRepoBlocked                 = Code{ID: "RKD-AUTH-001", Severity: SeverityError,   Title: "Persistent runner on public repository is blocked", File: "auth.md", Anchor: "rkd-auth-001"}
    AuthNetworkGitHubFailed               = Code{ID: "RKD-AUTH-002", Severity: SeverityError,   Title: "Cannot reach github.com / api.github.com",          File: "auth.md", Anchor: "rkd-auth-002"}
    AuthEphemeralBYOPublicForkAck         = Code{ID: "RKD-AUTH-003", Severity: SeverityError,   Title: "Ephemeral BYO on public/fork repo requires acknowledgment", File: "auth.md", Anchor: "rkd-auth-003"}
    AuthRunnerManagementPermissionDenied  = Code{ID: "RKD-AUTH-004", Severity: SeverityError,   Title: "Token lacks runner-management permission",          File: "auth.md", Anchor: "rkd-auth-004"}

    // SSH
    SSHHostKeyMismatch     = Code{ID: "RKD-SSH-001", Severity: SeverityError, Title: "SSH host key fingerprint mismatch",     File: "ssh.md", Anchor: "rkd-ssh-001"}
    SSHUnreachable         = Code{ID: "RKD-SSH-002", Severity: SeverityError, Title: "SSH host unreachable",                   File: "ssh.md", Anchor: "rkd-ssh-002"}
    SSHKeyPathNotFound     = Code{ID: "RKD-SSH-003", Severity: SeverityError, Title: "SSH private key file not found",         File: "ssh.md", Anchor: "rkd-ssh-003"}
    SSHPortUnreachable     = Code{ID: "RKD-SSH-004", Severity: SeverityError, Title: "SSH port unreachable",                   File: "ssh.md", Anchor: "rkd-ssh-004"}

    // BOOT
    BootRunnerVersionStale         = Code{ID: "RKD-BOOT-002", Severity: SeverityWarning, Title: "Bundled runner pin is newer than installed runner",   File: "bootstrap.md", Anchor: "rkd-boot-002"}
    BootServiceFailed              = Code{ID: "RKD-BOOT-003", Severity: SeverityError,   Title: "systemd service failed",                              File: "bootstrap.md", Anchor: "rkd-boot-003"}
    BootServiceMissing             = Code{ID: "RKD-BOOT-004", Severity: SeverityError,   Title: "systemd service missing",                             File: "bootstrap.md", Anchor: "rkd-boot-004"}
    BootInstallPathMissing         = Code{ID: "RKD-BOOT-005", Severity: SeverityError,   Title: "Runner install directory missing on host",            File: "bootstrap.md", Anchor: "rkd-boot-005"}
    BootWorkDirMissing             = Code{ID: "RKD-BOOT-006", Severity: SeverityWarning, Title: "Runner work directory missing on host",               File: "bootstrap.md", Anchor: "rkd-boot-006"}
    BootDiskLow                    = Code{ID: "RKD-BOOT-007", Severity: SeverityWarning, Title: "Disk space low under /opt or /var/lib",               File: "bootstrap.md", Anchor: "rkd-boot-007"}
    BootToolsMissing               = Code{ID: "RKD-BOOT-008", Severity: SeverityWarning, Title: "Required CLI tools missing on host",                  File: "bootstrap.md", Anchor: "rkd-boot-008"}
    BootTimeUnsynchronized         = Code{ID: "RKD-BOOT-009", Severity: SeverityWarning, Title: "Host clock not synchronized (NTP)",                   File: "bootstrap.md", Anchor: "rkd-boot-009"}
    BootPreflightUnsupportedDistro = Code{ID: "RKD-BOOT-010", Severity: SeverityWarning, Title: "Linux distribution not in supported matrix",          File: "bootstrap.md", Anchor: "rkd-boot-010"}
    BootPreflightFailed            = Code{ID: "RKD-BOOT-011", Severity: SeverityError,   Title: "Preflight check failed",                              File: "bootstrap.md", Anchor: "rkd-boot-011"}
    BootRunnerUserCreateFailed     = Code{ID: "RKD-BOOT-012", Severity: SeverityError,   Title: "runnerkit-runner user creation failed",               File: "bootstrap.md", Anchor: "rkd-boot-012"}
    BootRunnerPackageInstallFailed = Code{ID: "RKD-BOOT-013", Severity: SeverityError,   Title: "Runner tarball install failed",                       File: "bootstrap.md", Anchor: "rkd-boot-013"}
    BootRunnerOnlineVerifyTimeout  = Code{ID: "RKD-BOOT-014", Severity: SeverityError,   Title: "Runner did not report online before timeout",         File: "bootstrap.md", Anchor: "rkd-boot-014"}

    // GH
    GHRunnerOffline                = Code{ID: "RKD-GH-001", Severity: SeverityWarning, Title: "GitHub reports runner offline",                         File: "github.md", Anchor: "rkd-gh-001"}
    GHDuplicateCandidates          = Code{ID: "RKD-GH-002", Severity: SeverityError,   Title: "Multiple RunnerKit runner candidates found in GitHub",  File: "github.md", Anchor: "rkd-gh-002"}
    GHLabelDrift                   = Code{ID: "RKD-GH-003", Severity: SeverityWarning, Title: "Saved labels drift from GitHub-reported labels",        File: "github.md", Anchor: "rkd-gh-003"}
    GHRegistrationTokenCreateFailed = Code{ID: "RKD-GH-004", Severity: SeverityError,  Title: "Failed to create runner registration token",            File: "github.md", Anchor: "rkd-gh-004"}
    GHRunnerRegisterFailed         = Code{ID: "RKD-GH-005", Severity: SeverityError,   Title: "Runner registration failed",                            File: "github.md", Anchor: "rkd-gh-005"}
    GHDeregisterStaleFailed        = Code{ID: "RKD-GH-006", Severity: SeverityWarning, Title: "Stale GitHub runner deregistration failed",             File: "github.md", Anchor: "rkd-gh-006"}
    GHRecoverReregisterFailed      = Code{ID: "RKD-GH-007", Severity: SeverityError,   Title: "recover --reregister failed",                            File: "github.md", Anchor: "rkd-gh-007"}

    // PROV
    ProvProviderError              = Code{ID: "RKD-PROV-001", Severity: SeverityWarning, Title: "Hetzner provider returned error during status",       File: "provider.md", Anchor: "rkd-prov-001"}
    ProvResourceMissing            = Code{ID: "RKD-PROV-002", Severity: SeverityWarning, Title: "Hetzner resource missing for saved IDs",              File: "provider.md", Anchor: "rkd-prov-002"}
    ProvDrift                      = Code{ID: "RKD-PROV-003", Severity: SeverityWarning, Title: "Hetzner inventory drift from saved state",            File: "provider.md", Anchor: "rkd-prov-003"}
    ProvHCloudTokenMissing         = Code{ID: "RKD-PROV-004", Severity: SeverityError,   Title: "HCLOUD_TOKEN environment variable not set",           File: "provider.md", Anchor: "rkd-prov-004"}
    ProvHCloudQuotaExceeded        = Code{ID: "RKD-PROV-005", Severity: SeverityError,   Title: "Hetzner project quota exceeded",                      File: "provider.md", Anchor: "rkd-prov-005"}
    ProvHCloudPartialDestroy       = Code{ID: "RKD-PROV-006", Severity: SeverityWarning, Title: "Hetzner partial destroy — resources remain",          File: "provider.md", Anchor: "rkd-prov-006"}
    ProvBillableResourceLingering  = Code{ID: "RKD-PROV-007", Severity: SeverityError,   Title: "Hetzner resource still billable after destroy",       File: "provider.md", Anchor: "rkd-prov-007"}

    // CLEAN
    CleanCleanupPending            = Code{ID: "RKD-CLEAN-001", Severity: SeverityWarning, Title: "Cleanup checkpoints or notes are pending",           File: "cleanup.md", Anchor: "rkd-clean-001"}
    CleanEphemeralCleanupPending   = Code{ID: "RKD-CLEAN-002", Severity: SeverityWarning, Title: "Ephemeral cleanup checkpoints are pending",          File: "cleanup.md", Anchor: "rkd-clean-002"}
    CleanDownFilesRemoveFailed     = Code{ID: "RKD-CLEAN-003", Severity: SeverityError,   Title: "down: file removal failed",                          File: "cleanup.md", Anchor: "rkd-clean-003"}
    CleanEphemeralLogPreserveFailed = Code{ID: "RKD-CLEAN-004", Severity: SeverityWarning, Title: "Ephemeral log preservation failed",                 File: "cleanup.md", Anchor: "rkd-clean-004"}
    CleanDestroyPartial            = Code{ID: "RKD-CLEAN-005", Severity: SeverityWarning, Title: "destroy: partial cleanup, checkpoints retained",     File: "cleanup.md", Anchor: "rkd-clean-005"}

    // STATE — for STATE codes the docs entry lives in cleanup.md (state failures
    // are ultimately cleanup/recovery operations from the user's perspective).
    StateInvalidJSON         = Code{ID: "RKD-STATE-001", Severity: SeverityError, Title: "state.json is not valid JSON",                   File: "cleanup.md", Anchor: "rkd-state-001"}
    StateBackupWriteFailed   = Code{ID: "RKD-STATE-002", Severity: SeverityError, Title: "state backup write failed",                      File: "cleanup.md", Anchor: "rkd-state-002"}
    StateMigrationFailed     = Code{ID: "RKD-STATE-003", Severity: SeverityError, Title: "state migration failed",                         File: "cleanup.md", Anchor: "rkd-state-003"}
    StateSchemaTooNew        = Code{ID: "RKD-STATE-004", Severity: SeverityError, Title: "state schema_version newer than this CLI knows", File: "cleanup.md", Anchor: "rkd-state-004"}

    // CORE — for CORE codes the docs entry lives in cleanup.md (catch-all index).
    CoreInputRequired        = Code{ID: "RKD-CORE-001", Severity: SeverityError, Title: "Input required for non-interactive flow",         File: "cleanup.md", Anchor: "rkd-core-001"}
    CoreInvalidInput         = Code{ID: "RKD-CORE-002", Severity: SeverityError, Title: "Invalid CLI input",                               File: "cleanup.md", Anchor: "rkd-core-002"}
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
```

NOTE on `RKD-BOOT-001`: reserved (legacy / placeholder); skipping `001` is fine because `RKD-BOOT-002` corresponds to `runner_version_stale` and is the most-emitted runtime warning. Document this in the troubleshooting README's component-table footnote so the gap is not surprising.

NOTE on `RKD-CLEAN-NNN` vs `RKD-STATE-NNN`: CONTEXT D-14 lists 6 component files (`auth`, `ssh`, `bootstrap`, `github`, `provider`, `cleanup`); STATE and CORE codes do NOT have their own files — their entries live in `cleanup.md`. This keeps the file matrix at 6 (matching D-14) while still providing a stable anchor for STATE/CORE codes.

**Step 2: Create `internal/errcodes/url.go`** — URL builder honoring `RUNNERKIT_DOCS_BASE`:

```go
package errcodes

import (
    "os"
    "strings"
)

const defaultDocsBase = "https://github.com/salar/runnerkit/blob/main/docs"

// URL returns the canonical troubleshooting URL for a Code. Honors
// $RUNNERKIT_DOCS_BASE for static-site hosting (e.g., runnerkit.dev/docs).
//
// GitHub blob URLs need the ".md" suffix BEFORE the anchor:
//   https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-001
//
// Static site URLs typically strip the suffix:
//   https://runnerkit.dev/docs/troubleshooting/auth#rkd-auth-001
//
// Detection rule: if the docs base URL contains "/blob/", we keep the .md
// suffix; otherwise we strip it. This matches the two known hosting modes
// without needing a separate config flag.
func URL(c Code) string {
    base := strings.TrimRight(os.Getenv("RUNNERKIT_DOCS_BASE"), "/")
    if base == "" {
        base = defaultDocsBase
    }
    file := c.File
    if !strings.Contains(base, "/blob/") {
        file = strings.TrimSuffix(file, ".md")
    }
    return base + "/troubleshooting/" + file + "#" + c.Anchor
}

// FormatLine returns "RKD-XXX-NNN: <Title>\nSee: <URL>" — the canonical CLI
// emit shape per D-15. Callers should use this any time a Code is reported.
func FormatLine(c Code) string {
    return c.ID + ": " + c.Title + "\nSee: " + URL(c)
}
```

**Step 3: Create `internal/errcodes/codes_test.go`** with the 5 required tests from `<behavior>` above. All test names MUST match exactly (per `06-VALIDATION.md` lines 69-73): `TestEveryCodeHasDocAnchor`, `TestCodesAreUnique`, `TestURL_RespectsEnvOverride`, `TestEachComponentHasMinimumOneEntry`, `TestEntriesFollowSymptomDiagnosisFix`.

For tests that read `docs/troubleshooting/`, use `os.ReadFile` with relative path resolved via `filepath.Join`. The tests will run from `internal/errcodes/` cwd; the docs dir is at `../../docs/troubleshooting/`. Use `runtime.Caller` or the conventional `testdata` lookup helper to compute the absolute path robustly:

```go
func docsRoot(t *testing.T) string {
    t.Helper()
    _, thisFile, _, _ := runtime.Caller(0)
    return filepath.Join(filepath.Dir(thisFile), "..", "..", "docs", "troubleshooting")
}
```

For `TestEntriesFollowSymptomDiagnosisFix`, scan from `<a name="<anchor>">` to the next `<a name=` (or EOF) and assert all three substrings appear in the section: `### Symptom`, `### Diagnosis`, `### Fix`. Use case-sensitive matching.

These tests will FAIL until Task 2 ships the docs files. That's expected: the tests are the contract Task 2 satisfies.
  </action>
  <verify>
    <automated>go vet ./internal/errcodes/... && grep -q "type Code" internal/errcodes/codes.go && grep -q "RKD-AUTH-001" internal/errcodes/codes.go && grep -q "RKD-BOOT-002" internal/errcodes/codes.go && grep -q "RKD-STATE-004" internal/errcodes/codes.go && grep -q "RUNNERKIT_DOCS_BASE" internal/errcodes/url.go && grep -q "github.com/salar/runnerkit/blob/main/docs" internal/errcodes/url.go && grep -q "TestEveryCodeHasDocAnchor" internal/errcodes/codes_test.go && grep -q "TestCodesAreUnique" internal/errcodes/codes_test.go && grep -q "TestURL_RespectsEnvOverride" internal/errcodes/codes_test.go && grep -q "TestEachComponentHasMinimumOneEntry" internal/errcodes/codes_test.go && grep -q "TestEntriesFollowSymptomDiagnosisFix" internal/errcodes/codes_test.go && go test ./internal/errcodes -run TestURL_RespectsEnvOverride -count=1 && go test ./internal/errcodes -run TestCodesAreUnique -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/errcodes/codes.go` exports `type Code struct { ID, Title; File, Anchor string; Severity Severity }` (or equivalent fields per the snippet above).
    - `internal/errcodes/codes.go` exports `Registry []Code` containing >= 32 codes covering all 8 prefixes (AUTH, SSH, BOOT, GH, PROV, CLEAN, STATE, CORE).
    - `internal/errcodes/url.go` exports `URL(Code) string` honoring `os.Getenv("RUNNERKIT_DOCS_BASE")` with default `https://github.com/salar/runnerkit/blob/main/docs`.
    - `URL(c)` returns `<base>/troubleshooting/<file>#<anchor>` where `.md` is kept iff base contains `/blob/`.
    - `internal/errcodes/url.go` exports `FormatLine(Code) string` returning `"<ID>: <Title>\nSee: <URL>"`.
    - The 5 required tests are declared in `internal/errcodes/codes_test.go`. (Two of them — `TestURL_RespectsEnvOverride` and `TestCodesAreUnique` — pass at the end of THIS task. The other three pass after Task 2 ships docs.)
    - `go vet ./internal/errcodes/...` passes.
    - All validation matrix rows for D-15 (lines 70, 71) pass at the end of this task; rows 69, 72, 73 require Task 2 docs.
  </acceptance_criteria>
  <done>internal/errcodes package exists with full registry (>= 32 codes spanning 8 prefixes), URL builder honoring RUNNERKIT_DOCS_BASE with /blob/ vs static-site distinction, FormatLine helper, and 5 test scaffolds. TestURL_RespectsEnvOverride and TestCodesAreUnique pass; other 3 tests pass after Task 2.</done>
</task>

<task type="auto" tdd="false">
  <name>Task 2: Create six docs/troubleshooting/<component>.md files + README index, all with explicit anchors and Symptom/Diagnosis/Fix structure</name>
  <files>docs/troubleshooting/README.md, docs/troubleshooting/auth.md, docs/troubleshooting/ssh.md, docs/troubleshooting/bootstrap.md, docs/troubleshooting/github.md, docs/troubleshooting/provider.md, docs/troubleshooting/cleanup.md</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (Pattern 8 — docs structure example for `rkd-auth-001`; Pitfall 9 — explicit `<a name=>` anchors mandatory)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-14, D-16 four failure surfaces, D-17 Symptom/Diagnosis/Fix structure)
    - internal/errcodes/codes.go (Task 1 output — the canonical Registry; every code MUST have a corresponding anchor in the matching file)
    - internal/ops/doctor.go (existing finding remediation strings — copy them verbatim into the relevant Diagnosis/Fix sections)
    - docs/safety.md (Phase 5 output — referenced from auth.md for public-repo block rationale)
    - docs/byo-quickstart.md and docs/cloud-quickstart.md (existing — referenced as "rerun setup" advice in fix sections)
  </read_first>
  <action>
Create `docs/troubleshooting/` directory if missing. Create each of the 7 files below with the EXACT structure shown. Every entry MUST have an explicit `<a name="rkd-component-NNN"></a>` anchor on its own line directly above the heading (Pitfall 9: Markdown auto-anchors break on heading rename — explicit HTML anchors are immune).

**File 1: `docs/troubleshooting/README.md`** — index + global error code table:

```markdown
# Troubleshooting

Stuck? Find your `RKD-<COMPONENT>-NNN` code in the table below and follow the
component link. Every entry follows a `Symptom → Diagnosis → Fix` structure
(D-17).

If a `runnerkit` command printed a `See: <URL>` line, the URL points at the
exact entry below.

## Install verification

If `cosign verify-blob` fails or `sha256sum -c` reports a mismatch, the
downloaded archive is NOT the upstream release. Do NOT install it.

```bash
TAG=v1.0.0
cosign verify-blob \
  --bundle  runnerkit_${TAG#v}_checksums.txt.sigstore.json \
  --certificate-identity   "https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_${TAG#v}_checksums.txt
```

If you installed via Homebrew on macOS and see "macOS cannot verify that this
app is free from malware":

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/runnerkit  # Apple Silicon
xattr -d com.apple.quarantine /usr/local/bin/runnerkit     # Intel
```

(RunnerKit binaries are not Apple-notarized in v1; this is a known limitation.)

## Components

| File | Codes | Failures covered |
|---|---|---|
| [auth.md](auth.md)         | RKD-AUTH-NNN  | GitHub auth scope, public-repo block, ephemeral BYO acknowledgment, network access to github.com |
| [ssh.md](ssh.md)           | RKD-SSH-NNN   | host-key mismatch, host unreachable, key-path-not-found, port unreachable |
| [bootstrap.md](bootstrap.md) | RKD-BOOT-NNN  | systemd service, install/work paths, preflight, runner user/package install, online-verification timeout, runner version stale |
| [github.md](github.md)     | RKD-GH-NNN    | runner offline, duplicate candidates, label drift, registration/deregister/recover failures |
| [provider.md](provider.md) | RKD-PROV-NNN  | Hetzner token/quota/region, partial destroy, billable lingering |
| [cleanup.md](cleanup.md)   | RKD-CLEAN-NNN, RKD-STATE-NNN, RKD-CORE-NNN | down/destroy partial, ephemeral log preservation, state JSON read, schema-too-new, migration failure, CLI input |

> Note on numbering: codes are stable across renames; numbering grows
> monotonically per component. Some numbers may be reserved (e.g.,
> `RKD-BOOT-001` is reserved for future use). This is by design.

## Custom docs hosting

If your team hosts a fork of these docs (e.g., on a static site), set
`RUNNERKIT_DOCS_BASE` to override the URL prefix the CLI emits:

```bash
export RUNNERKIT_DOCS_BASE=https://my-docs.example.com/runnerkit
```

The CLI will then print `<my-docs>/troubleshooting/<component>#<anchor>` for
every code.
```

**File 2: `docs/troubleshooting/auth.md`** — RKD-AUTH-NNN entries. Skeleton (each entry MUST have explicit anchor + 3 sub-headings):

```markdown
# Troubleshooting: GitHub Authentication and Safety

Stable codes for this component: `RKD-AUTH-001`..`RKD-AUTH-004`. Anchors are
stable across renames (D-15).

***

<a name="rkd-auth-001"></a>
## RKD-AUTH-001: Persistent runner on public repository is blocked

**Severity:** error
**Component:** auth

### Symptom

`runnerkit up --repo owner/public-repo --mode persistent` fails with:

```
RKD-AUTH-001: Persistent runner on public repository is blocked
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-001
```

### Diagnosis

Persistent self-hosted runners on public repositories let any pull-request
contributor execute code on the runner host. RunnerKit blocks this by default
(safety policy from Phase 5; see [docs/safety.md](../safety.md)).

### Fix

Use ephemeral cloud (recommended for untrusted workloads):

```bash
runnerkit up --repo owner/public-repo --mode ephemeral --cloud hetzner
```

Or accept the risk explicitly (NOT recommended):

```bash
runnerkit up --repo owner/public-repo --allow-public-repo-risk --yes
```

Read [docs/safety.md](../safety.md) before allowing.

***

<a name="rkd-auth-002"></a>
## RKD-AUTH-002: Cannot reach github.com / api.github.com

**Severity:** error
**Component:** auth

### Symptom

`runnerkit up`, `runnerkit doctor`, or `runnerkit status` fails with:

```
RKD-AUTH-002: Cannot reach github.com / api.github.com
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-002
```

### Diagnosis

Either the host or the runner machine cannot HTTPS-connect to GitHub. RunnerKit
needs egress to `https://github.com` and `https://api.github.com` (and the
runner host needs the same).

### Fix

```bash
# From the host:
curl -fsSL https://api.github.com/zen
curl -fsSL https://github.com

# From the runner machine (replace user@host):
ssh user@host 'curl -fsSL https://api.github.com/zen'
```

If a corporate proxy is required, set `HTTPS_PROXY` and re-run. If a firewall
is blocking egress, allow HTTPS to `github.com` and `api.github.com`.

***

<a name="rkd-auth-003"></a>
## RKD-AUTH-003: Ephemeral BYO on public/fork repo requires acknowledgment

**Severity:** error
**Component:** auth

### Symptom

`runnerkit up --repo owner/public-repo --mode ephemeral` (BYO target) fails
with `RKD-AUTH-003`.

### Diagnosis

Ephemeral mode is the recommended path for public/fork workloads, but BYO
ephemeral on a shared host still carries some risk (the host's local
filesystem is touched between job runs even if the runner is one-shot).
Phase 5 requires explicit acknowledgment.

### Fix

Either typed acknowledgment in the interactive prompt, or:

```bash
runnerkit up --repo owner/public-repo --mode ephemeral --allow-ephemeral-byo-risk --yes
```

Or switch to ephemeral cloud for stronger isolation:

```bash
runnerkit up --repo owner/public-repo --mode ephemeral --cloud hetzner
```

***

<a name="rkd-auth-004"></a>
## RKD-AUTH-004: Token lacks runner-management permission

**Severity:** error
**Component:** auth

### Symptom

`runnerkit up` fails during the auth step with:

```
RKD-AUTH-004: Token lacks runner-management permission
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-004
```

### Diagnosis

The token used for auth (gh CLI cached credential or fine-grained PAT) does
not have permission to create runner registration tokens for this repository.

### Fix

If using `gh` CLI:

```bash
gh auth refresh -h github.com -s repo,workflow
```

If using a fine-grained PAT, regenerate at
<https://github.com/settings/tokens?type=beta> with:

- Repository access: `owner/repo` (the target repo)
- Repository permissions: `Administration: Read and write` (required for
  self-hosted runner management)

Then re-run `runnerkit up`.
```

**File 3: `docs/troubleshooting/ssh.md`** — RKD-SSH-001 through RKD-SSH-004. Same shape: explicit anchors, `### Symptom` / `### Diagnosis` / `### Fix` for each. Cover:

- `rkd-ssh-001` SSH host key fingerprint mismatch (Phase 2 fail-closed; remediation: `runnerkit recover --repo owner/repo --reaccept-host-key` if intentional, OR investigate compromise).
- `rkd-ssh-002` SSH host unreachable (network, firewall, sshd not running; `ssh -v user@host` to diagnose).
- `rkd-ssh-003` SSH private key file not found (path typo; `ls -l <key-path>`; convert to absolute path; chmod 600).
- `rkd-ssh-004` SSH port unreachable (`nc -zv host 22`; check sshd port if non-default; pass `--ssh-port`).

**File 4: `docs/troubleshooting/bootstrap.md`** — RKD-BOOT-002 through RKD-BOOT-014. Cover (one entry each):

- `rkd-boot-002` (KEY entry — wired from Plan 06-02 doctor finding `runner_version_stale`):
  Symptom: `runnerkit doctor` warns "installed runner version 2.330.0 is older than bundled pin 2.334.0".
  Diagnosis: GitHub may deprecate older runner versions; running stale leaves jobs failing with "the runner version is no longer supported".
  Fix: `runnerkit upgrade-runner --repo owner/name`. For ephemeral runners currently waiting/busy, see [upgrade.md](../upgrade.md#waiting).
- `rkd-boot-003` systemd service failed (`systemctl status runnerkit-runner`; `runnerkit logs`; `runnerkit recover --reinstall-service`).
- `rkd-boot-004` systemd service missing (saved unit not on host; `runnerkit recover --reinstall-service --dry-run`).
- `rkd-boot-005` install path missing (`/opt/runnerkit-runner` deleted by user; `runnerkit recover --reregister`).
- `rkd-boot-006` work dir missing (`/var/lib/runnerkit-runner` missing; `runnerkit recover --reinstall-service`).
- `rkd-boot-007` disk low (`df -h /opt /var/lib`; free space; restart service).
- `rkd-boot-008` tools missing (`apt-get install -y curl tar` on Debian/Ubuntu; rerun preflight).
- `rkd-boot-009` time unsynchronized (`timedatectl` to enable NTP; otherwise TLS fails).
- `rkd-boot-010` preflight unsupported distro (override with `--allow-unverified-distro`; report distro to maintainer).
- `rkd-boot-011` preflight failed (generic; check evidence in error message; rerun `runnerkit doctor`).
- `rkd-boot-012` runner user create failed (`useradd runnerkit-runner` permission; rerun with sudo or pre-create user).
- `rkd-boot-013` runner package install failed (network to github.com/actions/runner; verify sha256 matches; retry).
- `rkd-boot-014` runner online verification timeout (port-out, slow disk, GitHub flap; rerun `runnerkit up`; check logs).

**File 5: `docs/troubleshooting/github.md`** — RKD-GH-001 through RKD-GH-007. Cover:

- `rkd-gh-001` runner offline (host down or service stopped; `runnerkit recover --restart-service`).
- `rkd-gh-002` duplicate candidates (multiple registrations from prior failed runs; `runnerkit down --yes` then `runnerkit up`).
- `rkd-gh-003` label drift (saved labels differ from GitHub; `runnerkit recover --reregister` to align).
- `rkd-gh-004` registration token create failed (token expired; refresh `gh auth`; check RKD-AUTH-004).
- `rkd-gh-005` runner register failed (network; partial state; rerun `runnerkit up`).
- `rkd-gh-006` deregister stale failed (token permission; manual removal in `Settings → Actions → Runners`).
- `rkd-gh-007` recover --reregister failed (combine RKD-AUTH-004 + RKD-GH-005 diagnostics).

**File 6: `docs/troubleshooting/provider.md`** — RKD-PROV-001 through RKD-PROV-007. Cover:

- `rkd-prov-001` provider error during status (Hetzner API hiccup; retry; check `https://status.hetzner.com`).
- `rkd-prov-002` resource missing (saved IDs no longer in Hetzner; `runnerkit destroy --yes` to clean state, then `runnerkit up --cloud hetzner`).
- `rkd-prov-003` drift (manual changes in Hetzner Console; `runnerkit destroy --yes` and recreate).
- `rkd-prov-004` HCLOUD_TOKEN missing (`export HCLOUD_TOKEN=...`; create at <https://console.hetzner.cloud/projects/.../security/tokens>).
- `rkd-prov-005` quota exceeded (raise quota in Hetzner Console; or destroy unused servers; or pick a different region).
- `rkd-prov-006` partial destroy (some resources remain; rerun `runnerkit destroy --yes`; check Hetzner Console for orphan).
- `rkd-prov-007` billable resource lingering after destroy (D-12 gate 2 trip; CRITICAL — manually delete in Hetzner Console immediately; report bug).

**File 7: `docs/troubleshooting/cleanup.md`** — RKD-CLEAN-001 through CLEAN-005, plus all RKD-STATE-NNN and RKD-CORE-NNN. Cover:

- `rkd-clean-001` cleanup pending (checkpoints from prior partial run; `runnerkit destroy --yes` or `runnerkit down --yes`).
- `rkd-clean-002` ephemeral cleanup pending (Phase 5; finalizer didn't preserve logs; `runnerkit down --yes` to retry, or manual `tar` of `/var/lib/runnerkit/ephemeral/<runner>/logs`).
- `rkd-clean-003` down: file removal failed (permission; rerun with `--yes`; manual `sudo rm -rf <managed paths>`).
- `rkd-clean-004` ephemeral log preservation failed (`/var/lib/runnerkit/ephemeral/<runner>/logs` not writable; ensure runner user has write permission).
- `rkd-clean-005` destroy: partial cleanup (mix of provider+local state retained; checkpoints kept; rerun `runnerkit destroy --yes` after fixing root cause).
- `rkd-state-001` state.json invalid JSON (manual edit corrupted file; restore from `state.json.backup-v1-*Z` if present; otherwise `runnerkit up` to recreate).
- `rkd-state-002` state backup write failed (disk full or permission; `df -h ~/.local/state/runnerkit`; chmod 0700 on the dir).
- `rkd-state-003` state migration failed (file a bug; the original state.json is preserved at `state.json.backup-v<N>-*Z`).
- `rkd-state-004` state schema_version newer than this CLI knows (you downgraded RunnerKit; run `runnerkit upgrade` to install a CLI that understands this state, OR delete and re-run setup losing local metadata).
- `rkd-core-001` input required (you ran a non-interactive flow without `--yes` or required flags; rerun with the missing input).
- `rkd-core-002` invalid input (flag value invalid; check command help).

**Length check:** each component file should end up >= 50 lines (5+ entries × ~10 lines each plus front matter). README.md should be >= 40 lines.

**CRITICAL anchor rule:** Every entry's `<a name="rkd-component-NNN"></a>` MUST be on a line by itself directly ABOVE the `## RKD-...` heading. Pitfall 9: if you put the anchor in the heading text or rename the heading later, Markdown's auto-anchor changes and breaks every CLI emit-site URL.
  </action>
  <verify>
    <automated>test -f docs/troubleshooting/README.md && test -f docs/troubleshooting/auth.md && test -f docs/troubleshooting/ssh.md && test -f docs/troubleshooting/bootstrap.md && test -f docs/troubleshooting/github.md && test -f docs/troubleshooting/provider.md && test -f docs/troubleshooting/cleanup.md && grep -q '<a name="rkd-auth-001"></a>' docs/troubleshooting/auth.md && grep -q '<a name="rkd-boot-002"></a>' docs/troubleshooting/bootstrap.md && grep -q '<a name="rkd-state-004"></a>' docs/troubleshooting/cleanup.md && grep -q "### Symptom" docs/troubleshooting/auth.md && grep -q "### Diagnosis" docs/troubleshooting/auth.md && grep -q "### Fix" docs/troubleshooting/auth.md && grep -q "RKD-AUTH-NNN" docs/troubleshooting/README.md && grep -q "cosign verify-blob" docs/troubleshooting/README.md && grep -q "xattr -d com.apple.quarantine" docs/troubleshooting/README.md && grep -q "RUNNERKIT_DOCS_BASE" docs/troubleshooting/README.md && go test ./internal/errcodes -run 'TestEveryCodeHasDocAnchor|TestEachComponentHasMinimumOneEntry|TestEntriesFollowSymptomDiagnosisFix' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - All 7 files exist: README.md, auth.md, ssh.md, bootstrap.md, github.md, provider.md, cleanup.md.
    - README.md contains: components table linking to all 6 files; install verification cosign snippet; macOS quarantine `xattr` instructions; `RUNNERKIT_DOCS_BASE` env override docs; note about reserved code numbers.
    - auth.md contains: 4 entries with anchors `rkd-auth-001` through `rkd-auth-004`, each with `### Symptom` / `### Diagnosis` / `### Fix` headings.
    - ssh.md contains: 4 entries (`rkd-ssh-001` through `rkd-ssh-004`).
    - bootstrap.md contains: 13 entries (`rkd-boot-002` through `rkd-boot-014`; 001 is reserved). RKD-BOOT-002 specifically references `runnerkit upgrade-runner` (ties to Plan 06-02 doctor finding).
    - github.md contains: 7 entries (`rkd-gh-001` through `rkd-gh-007`).
    - provider.md contains: 7 entries (`rkd-prov-001` through `rkd-prov-007`). RKD-PROV-007 explicitly mentions D-12 gate 2 trip and "manually delete in Hetzner Console immediately".
    - cleanup.md contains: 5 RKD-CLEAN entries + 4 RKD-STATE entries + 2 RKD-CORE entries (11 entries total).
    - Every entry uses an explicit `<a name="rkd-component-NNN"></a>` anchor on its own line ABOVE the heading (Pitfall 9 mitigation).
    - All 5 errcodes tests pass: `TestEveryCodeHasDocAnchor`, `TestCodesAreUnique`, `TestURL_RespectsEnvOverride`, `TestEachComponentHasMinimumOneEntry`, `TestEntriesFollowSymptomDiagnosisFix`.
    - All 4 D-16 failure surfaces have at least one entry per component file: setup (auth + ssh), bootstrap+service (bootstrap), operations (github), cloud+cleanup (provider + cleanup).
    - All validation matrix rows for D-14, D-15, D-16, D-17 (lines 69-73) pass.
  </acceptance_criteria>
  <done>Six component troubleshooting files + README index exist; >= 46 entries total covering every code in the registry; explicit `<a name=>` anchors; `### Symptom` / `### Diagnosis` / `### Fix` structure on every entry; all 5 errcodes tests green.</done>
</task>

<task type="auto" tdd="false">
  <name>Task 3: Wire CLI emit sites to errcodes.URL — doctor remediation, public-repo block, ephemeral acknowledgment, state errors</name>
  <files>internal/ops/doctor.go, internal/cli/up.go, internal/cli/down.go, internal/cli/destroy.go, internal/cli/recover.go, internal/state/migrations.go</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-15 — every emit site prints `See: <URL>`)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (Pattern 8; "Plan 06-03 task 3 land LAST inside Plan 06-03 to avoid stomping in-progress 06-02 work")
    - internal/errcodes/codes.go (Task 1 output — typed Code constants)
    - internal/errcodes/url.go (Task 1 output — `errcodes.URL(c)` and `errcodes.FormatLine(c)`)
    - internal/ops/doctor.go (existing finding remediation strings — append the See: URL line)
    - internal/cli/up.go (find the public-repo block error path; ephemeral BYO acknowledgment path)
    - internal/state/migrations.go (Task 1 of Plan 06-02 — wraps errors in plain strings; add See: URL for ErrSchemaTooNew)
  </read_first>
  <action>
This task layers `errcodes.URL(...)` calls on top of existing emit sites. It does NOT change the failure semantics, only enriches the user-facing error message with a `See: <URL>` line. Each change is additive.

**Strategy:** Use `errcodes.FormatLine(code)` to produce `"<ID>: <Title>\nSee: <URL>"` and replace the existing free-form error message with it (or prepend if a more specific message is required). For doctor findings, append the URL to the existing Remediation string with a newline separator: `add(...remediation+"\n\nSee: "+errcodes.URL(...)...)`.

**Step 1: Doctor findings — add See: URLs.** In `internal/ops/doctor.go::BuildDoctorReport`, import `"github.com/salar/runnerkit/internal/errcodes"`, then for EACH `add(...)` call mapped to a finding ID, append the URL. Map:

```
github_runner_offline           → errcodes.GHRunnerOffline
github_duplicate_candidates     → errcodes.GHDuplicateCandidates
ssh_host_key_mismatch           → errcodes.SSHHostKeyMismatch
ssh_unreachable                 → errcodes.SSHUnreachable
service_failed                  → errcodes.BootServiceFailed
service_missing                 → errcodes.BootServiceMissing
label_drift                     → errcodes.GHLabelDrift
provider_error                  → errcodes.ProvProviderError
provider_resource_missing       → errcodes.ProvResourceMissing
provider_drift                  → errcodes.ProvDrift
install_path_missing            → errcodes.BootInstallPathMissing
work_dir_missing                → errcodes.BootWorkDirMissing
disk_low                        → errcodes.BootDiskLow
tools_missing                   → errcodes.BootToolsMissing
network_github_failed           → errcodes.AuthNetworkGitHubFailed
time_unsynchronized             → errcodes.BootTimeUnsynchronized
cleanup_pending                 → errcodes.CleanCleanupPending
ephemeral_cleanup_pending       → errcodes.CleanEphemeralCleanupPending
runner_version_stale            → errcodes.BootRunnerVersionStale  (added by Plan 06-02 — wire URL here)
```

Implementation pattern: extract a small helper at the top of `BuildDoctorReport`:

```go
addWithCode := func(id string, code errcodes.Code, source, evidence, remediation string) {
    report.Findings = append(report.Findings, Finding{
        ID:          id,
        Severity:    string(code.Severity),
        Source:      source,
        Evidence:    evidence,
        Remediation: remediation + "\n\nSee: " + errcodes.URL(code),
    })
}
```

Then replace each existing `add("github_runner_offline", SeverityWarning, "github", ...)` call with `addWithCode("github_runner_offline", errcodes.GHRunnerOffline, "github", ...)`. Pass-only findings (`state_present`, `github_runner_found`, `service_active`, `provider_found`) keep the original `add` helper — they don't get codes.

**Step 2: `internal/cli/up.go` — public-repo persistent block.** Locate the existing safety-gate error for persistent on public repo (search for "public_repo" or `DangerousPersistentOverrideCopy` or similar). Replace the user-facing error message with:

```go
return NewExitError(ExitSafetyGate, fmt.Errorf(errcodes.FormatLine(errcodes.AuthPublicRepoBlocked)+"\n\n%s", existingDetailedCopy))
```

Where `existingDetailedCopy` is whatever the current code already includes (e.g., the `DangerousPersistentOverrideCopy` constant from Phase 5). Do NOT remove existing copy — prepend the RKD line.

Same pattern for the ephemeral BYO public/fork acknowledgment refusal:

```go
return NewExitError(ExitSafetyGate, fmt.Errorf(errcodes.FormatLine(errcodes.AuthEphemeralBYOPublicForkAck)+"\n\n%s", existingCopy))
```

And the runner-management permission denied path:

```go
return NewExitError(ExitGitHubAuth, fmt.Errorf(errcodes.FormatLine(errcodes.AuthRunnerManagementPermissionDenied)+"\n\n%s", err))
```

**Step 3: `internal/state/migrations.go` — append See: URL for ErrSchemaTooNew.** Modify `ErrSchemaTooNew` (defined in Plan 06-02) to embed the URL in its message. Since errcodes is in `internal/errcodes`, and migrations is in `internal/state`, there is no import cycle:

```go
import "github.com/salar/runnerkit/internal/errcodes"

var ErrSchemaTooNew = errors.New(errcodes.FormatLine(errcodes.StateSchemaTooNew) +
    "\n\nrunnerkit state schema_version is newer than this CLI knows; upgrade RunnerKit (run `runnerkit upgrade`) to read this state")
```

NOTE: 06-02 must complete first per `depends_on: [02]`; `ErrSchemaTooNew` and the `runner_version_stale` doctor finding are guaranteed to exist when this task runs.

**Step 4: `internal/cli/down.go`, `internal/cli/destroy.go`, `internal/cli/recover.go` — wrap user-facing failure error returns with errcodes.FormatLine.** Specifically, for any `return fmt.Errorf("...")` that surfaces a known failure category, wrap it. Use grep to locate emission sites:

```bash
grep -n "fmt.Errorf\|NewExitError" internal/cli/down.go internal/cli/destroy.go internal/cli/recover.go
```

Map common errors to codes:

- `down`: file remove failure → `errcodes.CleanDownFilesRemoveFailed`
- `down`/`destroy`: ephemeral log preserve failure → `errcodes.CleanEphemeralLogPreserveFailed`
- `destroy`: partial cleanup → `errcodes.CleanDestroyPartial`
- `destroy`: HCLOUD partial → `errcodes.ProvHCloudPartialDestroy`
- `recover`: reregister failure → `errcodes.GHRecoverReregisterFailed`
- `up` cloud: HCLOUD_TOKEN missing → `errcodes.ProvHCloudTokenMissing`
- `up` cloud: quota → `errcodes.ProvHCloudQuotaExceeded`

For each: prepend `errcodes.FormatLine(code)+"\n\n"` to the existing error message string. Preserve all existing fmt.Errorf wrapping (`%w` for context).

**Step 5: Add a sentinel test** at `internal/cli/errcodes_emit_test.go` that proves the wiring: invoke `runUp` (or a known failure path) with a public-repo fixture and assert the rendered error message contains both `RKD-AUTH-001:` and `https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-001`. This single test serves as the regression gate for "every emit site references a code". (More detailed coverage is provided by the existing per-feature tests — they will all need their assertion strings updated, but those updates are part of THIS task; assertion-string updates are mechanical: append the new See: URL to the expected error substring.)

Practical guidance: run `go test ./...` after these changes; any test that asserted exact error string equality will fail. For each, update the assertion to use `strings.Contains(err.Error(), "RKD-XXX-NNN")` rather than equality. This is a deliberate testing pattern shift — error messages now have a stable code prefix and a stable URL suffix; tests should assert on those, not on the human-readable middle.
  </action>
  <verify>
    <automated>grep -q "errcodes.URL\|errcodes.FormatLine" internal/ops/doctor.go && grep -q "errcodes.AuthPublicRepoBlocked\|errcodes.FormatLine(errcodes.AuthPublicRepoBlocked)" internal/cli/up.go && grep -q "errcodes.StateSchemaTooNew\|errcodes.FormatLine(errcodes.StateSchemaTooNew)" internal/state/migrations.go && grep -q "errcodes.BootRunnerVersionStale\|BootRunnerVersionStale" internal/ops/doctor.go && go vet ./internal/ops/... ./internal/cli/... ./internal/state/... ./internal/errcodes/... && go test ./internal/ops -run TestDoctor_StaleRunnerVersion -count=1 && go test ./internal/cli -count=1 -run 'TestUp.*PublicRepo|TestErrcodesEmit'</automated>
  </verify>
  <acceptance_criteria>
    - `internal/ops/doctor.go` imports `github.com/salar/runnerkit/internal/errcodes`.
    - At least 18 doctor findings have their Remediation string augmented with `\n\nSee: <URL>` (one per finding ID in the mapping table above).
    - `internal/cli/up.go` public-repo persistent block error message contains literal `RKD-AUTH-001:` AND `errcodes.FormatLine(errcodes.AuthPublicRepoBlocked)` is the call site.
    - `internal/cli/up.go` ephemeral BYO ack refusal error message contains literal `RKD-AUTH-003:`.
    - `internal/state/migrations.go::ErrSchemaTooNew` message contains literal `RKD-STATE-004:` AND `https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/cleanup.md#rkd-state-004`.
    - `internal/cli/destroy.go` HCLOUD-partial-destroy error path contains a reference to `errcodes.ProvHCloudPartialDestroy` or `errcodes.CleanDestroyPartial`.
    - `internal/cli/down.go` ephemeral-log-preserve error path references `errcodes.CleanEphemeralLogPreserveFailed`.
    - `go vet` passes across all touched packages.
    - Existing test suites pass (after assertion-string updates that include the new RKD prefix). Any test that previously asserted a specific error message via equality has been updated to use `strings.Contains` checking for the RKD code.
    - The wiring sentinel test (`TestErrcodesEmit_PublicRepoBlocked` or similar) passes.
    - All errcodes package tests still pass: `go test ./internal/errcodes -count=1` green.
  </acceptance_criteria>
  <done>Doctor findings include See: URL in remediation; public-repo block, ephemeral acknowledgment, runner-management permission, state-schema-too-new, HCLOUD partial destroy, and ephemeral log preserve failures all emit `RKD-XXX-NNN: <Title>` followed by `See: <URL>`. Existing tests updated to assert RKD codes via `strings.Contains`. Full test suite green.</done>
</task>

<task type="auto" tdd="false">
  <name>Task 4: Cross-link from README, BYO/cloud quickstarts, and safety.md to docs/troubleshooting/</name>
  <files>README.md, docs/byo-quickstart.md, docs/cloud-quickstart.md, docs/safety.md</files>
  <read_first>
    - README.md (current state — Plan 06-01 added Install section that already links to docs/troubleshooting/README.md; verify link is live now that the file exists)
    - docs/byo-quickstart.md, docs/cloud-quickstart.md, docs/safety.md (existing — find natural call-out points where "if this fails" is implied)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (§"User-facing docs to extend")
  </read_first>
  <action>
**Step 1: README.md — verify and enrich.** Plan 06-01 added a forward link `[docs/troubleshooting/README.md](docs/troubleshooting/README.md)` in the Install section. That link now resolves (Task 2 created the file). Add a top-level "Troubleshooting" or "Help" section after the Install section:

```markdown
## Troubleshooting

If a `runnerkit` command prints a `See: <URL>` line, the URL points at a stable
entry in [docs/troubleshooting/](docs/troubleshooting/README.md). Index by
component:

- [Auth and safety](docs/troubleshooting/auth.md) — `RKD-AUTH-NNN`
- [SSH](docs/troubleshooting/ssh.md) — `RKD-SSH-NNN`
- [Bootstrap and service](docs/troubleshooting/bootstrap.md) — `RKD-BOOT-NNN`
- [GitHub runner](docs/troubleshooting/github.md) — `RKD-GH-NNN`
- [Cloud provider](docs/troubleshooting/provider.md) — `RKD-PROV-NNN`
- [Cleanup, state, CLI input](docs/troubleshooting/cleanup.md) — `RKD-CLEAN-NNN`, `RKD-STATE-NNN`, `RKD-CORE-NNN`

You can override the URL prefix the CLI prints with
`RUNNERKIT_DOCS_BASE=https://your-docs-host/runnerkit`.
```

**Step 2: docs/byo-quickstart.md — add a short call-out.** Find a natural place near the end (e.g., after the "Verify the runner is online" section). Add:

```markdown
### If something fails

Look for a `RKD-<COMPONENT>-NNN` code in the failure output. The accompanying
`See: <URL>` link points at a Symptom / Diagnosis / Fix entry in
[docs/troubleshooting/](troubleshooting/README.md). Most BYO failures fall in:

- [SSH](troubleshooting/ssh.md) — connectivity, host-key, key path
- [Bootstrap and service](troubleshooting/bootstrap.md) — preflight, runner user, systemd
- [GitHub runner](troubleshooting/github.md) — registration, online verification
```

**Step 3: docs/cloud-quickstart.md — add a similar call-out.** Near the end, after the "Verify the runner is online" section:

```markdown
### If something fails

Look for a `RKD-<COMPONENT>-NNN` code in the failure output. The accompanying
`See: <URL>` link points at a Symptom / Diagnosis / Fix entry in
[docs/troubleshooting/](troubleshooting/README.md). Most cloud failures fall
in:

- [Provider](troubleshooting/provider.md) — `HCLOUD_TOKEN`, quota, partial destroy, billable lingering
- [Bootstrap and service](troubleshooting/bootstrap.md) — same as BYO
- [GitHub runner](troubleshooting/github.md) — registration, online verification
```

**Step 4: docs/safety.md — link from public-repo / risky-workload sections.** Find the section discussing public-repo blocks (search for "public" or "DangerousPersistentOverride"). Add at the end:

```markdown
> If you see `RKD-AUTH-001` or `RKD-AUTH-003` in CLI output, the
> [auth troubleshooting page](troubleshooting/auth.md) has copy-paste fix
> commands. Read this page first before allowing.
```

NOTE: do not duplicate content. The contract is: `docs/safety.md` is the canonical safety guidance (Phase 5 D-17 — "docs/safety.md owns canonical Phase 5 safety copy"). Troubleshooting entries link BACK to it, not the other way around. Only add a single forward reference from safety.md to troubleshooting/auth.md.
  </action>
  <verify>
    <automated>grep -q "docs/troubleshooting/README.md\|docs/troubleshooting/" README.md && grep -q "RKD-AUTH-NNN\|RKD-AUTH-" README.md && grep -q "troubleshooting" docs/byo-quickstart.md && grep -q "troubleshooting" docs/cloud-quickstart.md && grep -q "troubleshooting/auth.md\|RKD-AUTH" docs/safety.md && grep -q "RUNNERKIT_DOCS_BASE" README.md</automated>
  </verify>
  <acceptance_criteria>
    - README.md has a `## Troubleshooting` section listing all 6 component files.
    - README.md mentions `RUNNERKIT_DOCS_BASE` env override.
    - `docs/byo-quickstart.md` has an "If something fails" subsection linking to ssh.md, bootstrap.md, github.md.
    - `docs/cloud-quickstart.md` has an "If something fails" subsection linking to provider.md, bootstrap.md, github.md.
    - `docs/safety.md` has a forward reference to `docs/troubleshooting/auth.md` mentioning `RKD-AUTH-001` or `RKD-AUTH-003`.
    - No duplication: troubleshooting entries reference safety.md back; safety.md only points forward, does not duplicate the troubleshooting copy.
  </acceptance_criteria>
  <done>README.md has Troubleshooting index linking all 6 files plus RUNNERKIT_DOCS_BASE; BYO and cloud quickstarts each have "If something fails" subsections; docs/safety.md has a single forward link to troubleshooting/auth.md.</done>
</task>

</tasks>

<verification>
Phase-level checks for Plan 06-03 completion:

1. `go test ./internal/errcodes -count=1` passes (5 tests: TestEveryCodeHasDocAnchor, TestCodesAreUnique, TestURL_RespectsEnvOverride, TestEachComponentHasMinimumOneEntry, TestEntriesFollowSymptomDiagnosisFix).
2. `go test ./... -count=1 -race` passes (full suite — touched test files retain or update their assertions to handle new RKD prefix).
3. Every Code in `errcodes.Registry` has a matching `<a name=>` anchor in `docs/troubleshooting/<file>.md`.
4. `runnerkit doctor` against a stale-runner fixture prints both `runner_version_stale` finding ID AND a `See: <URL>` line containing `rkd-boot-002`.
5. `runnerkit up --repo owner/public-repo --mode persistent` against a public-repo fixture prints `RKD-AUTH-001:` and a `See: <URL>` line.
6. `RUNNERKIT_DOCS_BASE=https://example.com/x runnerkit doctor` prints URLs starting with `https://example.com/x/troubleshooting/` instead of the default GitHub blob URL.

Validation matrix coverage (`06-VALIDATION.md`):
- Line 69 (Every code has doc anchor): satisfied.
- Line 70 (Codes are unique): satisfied.
- Line 71 (URL builder respects env override): satisfied.
- Line 72 (Each component has minimum one entry): satisfied.
- Line 73 (Entries follow Symptom/Diagnosis/Fix): satisfied.

All 5 D-14..D-17 validation rows are green at the end of this plan.
</verification>

<success_criteria>
- `internal/errcodes/` package exists with: 32+ Codes covering 8 component prefixes (AUTH, SSH, BOOT, GH, PROV, CLEAN, STATE, CORE), URL builder honoring `RUNNERKIT_DOCS_BASE` with `/blob/` vs static-site distinction, FormatLine helper.
- `docs/troubleshooting/` directory has README.md + 6 component files; >= 46 anchored entries with `### Symptom` / `### Diagnosis` / `### Fix` structure; RKD-BOOT-002 entry references `runnerkit upgrade-runner` (Plan 06-02 wiring); RKD-PROV-007 entry references D-12 gate 2 trip.
- README.md, BYO/cloud quickstarts, and safety.md cross-link to `docs/troubleshooting/`.
- All CLI emit sites (doctor findings, up safety gates, state migration errors, down/destroy/recover failures) emit `RKD-<COMPONENT>-NNN: <Title>` followed by `See: <URL>`.
- All 5 errcodes tests green; full test suite green.
- All hard rules from `<phase_specific_guidance>` Hard rule 10 are satisfied (RKD codes stable, URL anchor format `<docs_base>/troubleshooting/<component>#rkd-<component>-NNN`, RUNNERKIT_DOCS_BASE override).
- The `internal/ops/doctor.go` overlap with Plan 06-02 (stale-runner finding) is resolved without conflict — both plans add new lines to BuildDoctorReport but don't collide.
</success_criteria>

<output>
After completion, create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-03-SUMMARY.md` summarizing:
- Files added (`internal/errcodes/{codes.go, url.go, codes_test.go}`, `docs/troubleshooting/{README.md, auth.md, ssh.md, bootstrap.md, github.md, provider.md, cleanup.md}`, `internal/cli/errcodes_emit_test.go`).
- Files modified (`internal/ops/doctor.go` — addWithCode helper + 18 finding wirings; `internal/cli/up.go`, `down.go`, `destroy.go`, `recover.go` — RKD code emit; `internal/state/migrations.go` — ErrSchemaTooNew embeds RKD-STATE-004 URL; `README.md` — Troubleshooting section; `docs/byo-quickstart.md`, `docs/cloud-quickstart.md`, `docs/safety.md` — cross-links).
- Cross-plan resolution: doctor.go overlap with Plan 06-02 — both plans land cleanly because they add new lines (Plan 06-02 adds the `runner_version_stale` finding; this plan adds the URL wiring helper used by all findings including that one).
- Locked decisions implemented (D-14, D-15, D-16, D-17).
- Validation matrix rows closed (5 rows from `06-VALIDATION.md` lines 69-73).
- 32+ stable RKD codes registered; >= 46 anchored docs entries; all 4 D-16 failure surfaces covered.
</output>
