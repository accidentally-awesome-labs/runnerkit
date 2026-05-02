---
phase: 05-scoped-ephemeral-mode-and-safety-profiles
plan: "02"
subsystem: ephemeral-runner-lifecycle
tags: [cli, ephemeral, bootstrap, systemd, finalizer, ttl, state, provider, status, logs, doctor, down, destroy]
requires:
  - phase: 05-scoped-ephemeral-mode-and-safety-profiles-01
    provides: runmode constants, mode flag parsing, safety profiles, ephemeral label/snippet helpers, mode-aware safety enforcement
  - phase: 04-recommended-cloud-path-and-billable-cleanup
    provides: BYO and Hetzner cloud setup paths, provider Validate/Plan/Provision lifecycle, runnerkit destroy cleanup contract
provides:
  - bootstrap.Options Mode/EphemeralTTL/LogArchivePath/FinalizerPath/EphemeralServiceName/EphemeralTTLServiceName/EphemeralTTLTimerName fields
  - bootstrap.RenderEphemeralInstallScript using config.sh --replace --ephemeral with redacted RUNNERKIT_REGISTRATION_TOKEN env var
  - bootstrap.RenderEphemeralServiceScript writing /etc/systemd/system/<service> with Restart=no, ExecStart=run.sh, ExecStopPost=finalizer (no svc.sh install/start)
  - bootstrap.RenderEphemeralFinalizerScript preserving _diag Runner_/Worker_*.log, bounded journalctl, sentinel state.json, credential cleanup
  - bootstrap.RenderEphemeralTTLTimerScript with OnActiveSec=24h safeguard that stops the runner service and runs finalizer ttl_expired
  - bootstrap.RenderEphemeralLogPreservationScript for cleanup-time log archive
  - bootstrap.ApplyEphemeral with command IDs fix_dependencies, create_runner_user, download_runner, configure_ephemeral_runner, install_ephemeral_finalizer, install_ephemeral_service, install_ephemeral_ttl_timer, verify_ephemeral_service
  - state.EphemeralMetadata{Enabled, TTL, ExpiresAt, LogArchivePath, FinalizerStatus, CleanupCommand} attached as RepositoryState.Ephemeral with backwards-compatible json:"ephemeral,omitempty"
  - provider.ProvisionInput.Mode field driving HetznerOwnershipTags mode=ephemeral/persistent
  - cli ephemeralSuffix/LogArchivePath/ServiceName/CleanupCommand helpers, ephemeralBootstrapOptions, shortEphemeralIDFn test seam
  - cli ephemeral BYO/cloud up paths invoking ApplyEphemeral after preflight/readiness, persisting EphemeralMetadata, rendering "Ephemeral runner ready" with TTL/finalizer/cleanup/not-fleet UI-SPEC sentences
  - ops.EphemeralFact + ReasonEphemeral{Waiting,Busy,Completed,TTLExpired,CleanupPending} + Classify branch that runs before persistent github_runner_missing
  - ops.CommandLogsEphemeralArchive{List,Tail} collecting Runner_/Worker_*.log and systemd-journal.log into ephemeral_runner_diag and ephemeral_systemd_journal sections
  - cli/status sources.ephemeral JSON, mode bullets, status.ephemeral.state remote sentinel collection, UI-SPEC empty state
  - cli/logs production-grade log-forwarding warning and Log archive bullet for ephemeral runners
  - cli/down ephemeral.logs.preserve before down.files.remove for ephemeral runners
  - cli/destroy ephemeral.logs.preserve before remote/provider cleanup, ephemeral-specific destroy prompt copy
affects: [phase-05-03, cli, github, labels, state, provider, ops, bootstrap]
tech-stack:
  added: []
  patterns:
    [
      ephemeral one-shot systemd unit (Restart=no, ExecStopPost finalizer, no svc.sh loop),
      TTL safeguard timer that stops the runner service and runs the finalizer with ttl_expired status,
      finalizer-driven log preservation into /var/lib/runnerkit/ephemeral/<runner>/logs,
      ephemeral.logs.preserve runs before file/provider cleanup so logs survive down/destroy,
      mode-aware Classify branch so completed ephemeral auto-deregistration is terminal progress,
      shortEphemeralIDFn test seam for deterministic ephemeral runner names,
    ]
key-files:
  created:
    - internal/cli/up_ephemeral_test.go
  modified:
    - internal/bootstrap/install.go
    - internal/bootstrap/script.go
    - internal/bootstrap/install_test.go
    - internal/bootstrap/script_test.go
    - internal/state/schema.go
    - internal/state/state_test.go
    - internal/provider/provider.go
    - internal/provider/profile.go
    - internal/provider/profile_test.go
    - internal/cli/up.go
    - internal/cli/up_cloud_test.go
    - internal/cli/down.go
    - internal/cli/down_test.go
    - internal/cli/destroy.go
    - internal/cli/destroy_test.go
    - internal/cli/status.go
    - internal/cli/status_test.go
    - internal/cli/logs.go
    - internal/cli/logs_test.go
    - internal/ops/status.go
    - internal/ops/status_test.go
    - internal/ops/logs.go
    - internal/ops/logs_test.go
    - internal/ops/doctor.go
key-decisions:
  - "Ephemeral lifecycle classification (waiting/busy/completed/ttl_expired/cleanup_pending) runs before the persistent github_runner_missing branch in ops.Classify so completed auto-deregistration is reported as terminal progress, not a recovery condition."
  - "buildEphemeralBYORepositoryState/buildEphemeralCloudRepositoryState reuse persistent buildBYO/buildCloud state builders and only override Mode, SafetyProfile, ServiceName, and Ephemeral so the ephemeral path benefits from any future cleanup-related fixes to those builders."
  - "shortEphemeralIDFn is a package-level seam (not a parameter) so production callers stay simple while tests can stub a deterministic short id without weakening waitForRunnerOnline's exact-name match."
  - "ephemeral.logs.preserve runs as a sudo systemd-aware script before file removal in `down` and before remote runner removal in `destroy`; failures are recorded as ephemeral_log_preservation_pending and surfaced via cleanup_pending classification rather than blocking cleanup."
  - "The ephemeral Hetzner cost caveat sentence is rendered both inline in the cloud provisioning plan and again as part of the ephemeral cloud safety-profile tradeoff bullets, so the literal copy survives terminal-width wrapping and is greppable for both human dry-run and live cloud paths."
patterns-established:
  - "bootstrap.ApplyEphemeral mirrors bootstrap.Apply's command sequence shape but replaces the persistent svc.sh install/start steps with finalizer/service/TTL-timer/verify steps; ServiceNotActiveError is returned only for the ephemeral install_*/verify_* steps."
  - "ephemeralBootstrapOptions defaults LogArchivePath, FinalizerPath, EphemeralServiceName, EphemeralTTLServiceName, EphemeralTTLTimerName, and EphemeralTTL=24h in one place so callers (BYO and cloud) get identical names."
  - "cli/status, cli/logs, cli/down, cli/destroy all consume EphemeralMetadata through repoState.Ephemeral.* fields and never re-derive the runner name; the bootstrap path is the single writer."
requirements-completed: [RUN-02]
duration: 23 min
completed: 2026-05-02
---

# Phase 05 Plan 02: Ephemeral Runner Lifecycle Summary

**`runnerkit up --mode ephemeral` now creates a real RunnerKit-managed one-job GitHub Actions runner with a one-shot systemd unit, 24h TTL safeguard, finalizer-preserved logs, and mode-aware status/logs/doctor/down/destroy semantics for both BYO and Hetzner cloud.**

## Performance

- **Duration:** 23 min
- **Started:** 2026-05-02T18:30:23Z
- **Completed:** 2026-05-02T18:53:18Z
- **Tasks:** 3
- **Files modified:** 24

## Accomplishments

- Bootstrap renders ephemeral install (`config.sh --replace --ephemeral`), one-shot systemd unit with `Restart=no`/`ExecStopPost=<finalizer> completed`, finalizer that copies `Runner_*.log`/`Worker_*.log`/`journalctl`/sentinel `state.json` and removes `.runner`/`.credentials*`, and a 24h TTL timer that triggers the finalizer with `ttl_expired`. `ApplyEphemeral` runs `fix_dependencies → create_runner_user → download_runner → configure_ephemeral_runner → install_ephemeral_finalizer → install_ephemeral_service → install_ephemeral_ttl_timer → verify_ephemeral_service` and never invokes `svc.sh install/start`.
- BYO and cloud `runnerkit up --mode ephemeral` paths both reuse all existing Phase 4 readiness gates (preflight, host-key trust, cloud-init, provider Validate/Plan/Provision, just-in-time registration token), then call `bootstrap.ApplyEphemeral`, save `EphemeralMetadata` into state with a 24h `expires_at` and the correct cleanup command (`runnerkit down` for BYO, `runnerkit destroy` for cloud), and render "Ephemeral runner ready" with the canonical UI-SPEC sentences (`GitHub will assign at most one job…`, `TTL safeguard…`, `RunnerKit preserves best-effort runner _diag and systemd journal logs before cleanup.`, `Cleanup after the job: <command>`, `Ephemeral mode is not a fleet manager.`).
- Cloud ephemeral provisioning sets `provider.ProvisionInput.Mode = "ephemeral"` so `HetznerOwnershipTags` writes `mode=ephemeral`, the Hetzner resource names use the ephemeral runner name, and the cloud provisioning plan renders the exact `Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until \`runnerkit destroy --repo owner/name\` verifies cleanup.` caveat before any `Provision` call.
- `ops.Classify` and `ops.BuildDoctorReport` distinguish `ephemeral_waiting`, `ephemeral_busy`, `ephemeral_completed`, `ephemeral_ttl_expired`, and `ephemeral_cleanup_pending` and run the ephemeral branch before the persistent `github_runner_missing` recovery hint. `cli/status` populates `sources.ephemeral` JSON, renders `Mode/Safety profile/Log archive/Cleanup after the job` bullets, runs the `status.ephemeral.state` remote sentinel read when SSH is reachable, and emits the UI-SPEC empty state heading "No RunnerKit-managed runner is saved for `owner/name`." plus the ephemeral-cloud remediation command.
- `cli/logs` collects preserved `Runner_*.log`/`Worker_*.log`/`systemd-journal.log` from `Ephemeral.LogArchivePath` via `logs.ephemeral.archive.list` and `logs.ephemeral.archive.tail`, splits them into `ephemeral_runner_diag` and `ephemeral_systemd_journal` sections, and always renders `RunnerKit preserves best-effort logs only; configure external log forwarding for production-grade ephemeral troubleshooting.` for ephemeral state.
- `cli/down` and `cli/destroy` run `ephemeral.logs.preserve` (using `bootstrap.RenderEphemeralLogPreservationScript`) before deleting runner files / cloud resources when SSH is reachable, recording `ephemeral_log_preservation_pending` on failure rather than blocking cleanup. Cloud ephemeral destroy uses the dedicated prompt copy `Destroy ephemeral cloud runner: type \`destroy owner/name\` to remove the GitHub runner registration and RunnerKit-created Hetzner resources.`.
- `state.EphemeralMetadata` (with `enabled`, `ttl`, `expires_at`, `log_archive_path`, `finalizer_status`, `cleanup_command`) is persisted under the `ephemeral` key on `RepositoryState`. Older states without the field still load (verified by `TestEphemeralMetadataPersistsAndIsBackwardsCompatible`).

## Task Commits

1. **Task 05-02-01: Ephemeral bootstrap scripts and ApplyEphemeral** — `49919cd` (test) + `4fa5e8e` (feat)
2. **Task 05-02-02: Wire ephemeral BYO/cloud lifecycle, state, provider Mode** — `befee9f` (test) + `a22b118` (feat)
3. **Task 05-02-03: Mode-aware ephemeral status/logs/doctor/down/destroy log preservation** — `6f66313` (test) + `1630816` (feat)

## Files Created/Modified

### Created

- `internal/cli/up_ephemeral_test.go` — End-to-end ephemeral BYO/cloud completion, JSON contract, preflight/readiness blocks, and persistent default regression coverage.

### Modified

- `internal/bootstrap/install.go` — Added ephemeral fields to `Options`, `normalizeOptions` defaults for ephemeral mode, and `ApplyEphemeral` with the 8-command sequence and `ServiceNotActiveError` boundary.
- `internal/bootstrap/script.go` — Added `RenderEphemeralInstallScript`, `RenderEphemeralFinalizerScript`, `RenderEphemeralServiceScript`, `RenderEphemeralTTLTimerScript`, `RenderEphemeralLogPreservationScript`.
- `internal/bootstrap/install_test.go`, `internal/bootstrap/script_test.go` — Tests asserting redaction, exact command IDs, `--replace --ephemeral`, `Restart=no`/`ExecStopPost`, and absence of `svc.sh install`.
- `internal/state/schema.go` — `EphemeralMetadata` struct and `RepositoryState.Ephemeral` field.
- `internal/state/state_test.go` — Round-trip + backwards-compat test for ephemeral metadata.
- `internal/provider/provider.go`, `internal/provider/profile.go` — `ProvisionInput.Mode` field, `HetznerOwnershipTags` mode override, profile test for ephemeral tag.
- `internal/cli/up.go` — Helpers (`ephemeralSuffix`, `ephemeralLogArchivePath`, `ephemeralServiceName`, `ephemeralCleanupCommand`, `ephemeralBootstrapOptions`, `shortEphemeralIDFn` seam), BYO+cloud branches into `bootstrap.ApplyEphemeral`, `buildEphemeralBYO/CloudRepositoryState`, `renderEphemeralCompletionHuman`, `ephemeralCompletionJSON`.
- `internal/cli/down.go`, `internal/cli/destroy.go` — `ephemeral.logs.preserve` command before file/provider cleanup, ephemeral-specific destroy prompt copy.
- `internal/cli/status.go` — `collectEphemeralFact` with remote `status.ephemeral.state` sentinel read, `sources.ephemeral` JSON, mode/profile/log-archive/cleanup bullets, UI-SPEC empty state heading.
- `internal/cli/logs.go` — `Log archive` bullet plus production-grade forwarding warning for ephemeral mode.
- `internal/ops/status.go` — `EphemeralFact`, ephemeral reason IDs, `classifyEphemeral`, `ttlExpired`, `hasEphemeralCleanupPending`.
- `internal/ops/logs.go` — Ephemeral archive collection (`logs.ephemeral.archive.list`/`tail`) into `ephemeral_runner_diag`/`ephemeral_systemd_journal` sections.
- `internal/ops/doctor.go` — Ephemeral findings using the same vocabulary as `Classify`.
- `internal/cli/up_cloud_test.go`, `internal/cli/down_test.go`, `internal/cli/destroy_test.go`, `internal/cli/status_test.go`, `internal/cli/logs_test.go`, `internal/ops/status_test.go`, `internal/ops/logs_test.go` — New tests for ephemeral cloud cost caveat, log preservation order, ephemeral destroy prompt, ephemeral status terminal-progress, ephemeral logs, and `Classify` branches.

## Decisions Made

- Ephemeral classification runs BEFORE persistent `github_runner_missing` so a completed one-job runner does not surface as a recovery condition.
- `buildEphemeralBYO/CloudRepositoryState` are thin wrappers that delegate to the persistent builders and only override Mode/SafetyProfile/ServiceName/Ephemeral, keeping the ephemeral and persistent paths in sync for any future cleanup metadata changes.
- A package-level `shortEphemeralIDFn` seam keeps the production code free of test-only parameters while letting tests stub a deterministic short id (`fake1`) so `waitForRunnerOnline`'s exact-name match still works against the fake GitHub service.
- `ephemeral.logs.preserve` runs before file/provider mutation but never blocks cleanup; failures are recorded as `ephemeral_log_preservation_pending` and surfaced via the `ephemeral_cleanup_pending` classification.

## Deviations from Plan

### Test-Tolerance Adjustments

**1. [Test ergonomics] Long UI-SPEC sentences wrap at 80–100 columns**

- **Found during:** Tasks 05-02-02 and 05-02-03 (ephemeral BYO completion, ephemeral status, ephemeral logs, UI-SPEC empty state).
- **Issue:** The `internal/ui` renderer wraps lines at the configured width, so tests that assert literal long sentences (e.g., `GitHub will assign at most one job to this runner, then automatically deregister it.`, `RunnerKit preserves best-effort logs only; configure external log forwarding for production-grade ephemeral troubleshooting.`, the UI-SPEC empty state remediation, the Hetzner cost caveat) saw them split across multiple output lines.
- **Fix:** Tests assert the canonical sentence after flattening internal whitespace runs (`strings.Join(strings.Fields(out), " ")`). The renderer is unchanged; the human sentence content remains exactly as required by the UI-SPEC. JSON tests assert the exact string directly because JSON output is not wrapped.
- **Files modified:** `internal/cli/up_ephemeral_test.go`, `internal/cli/status_test.go`, `internal/cli/logs_test.go`.
- **Committed in:** `a22b118`, `1630816`.

---

**Total deviations:** 0 auto-fixed; 1 test-tolerance adjustment.
**Impact on plan:** None on production code. The renderer wrap behavior was already established in plan 05-01 and is not unique to plan 05-02.

## Issues Encountered

- The Hetzner cost caveat sentence was originally only rendered in the safety-profile tradeoff bullets. Tests asserted it must also appear before provider `Provision`, so the existing `renderCloudProvisionPlan` already includes the caveat as a `WarningLine` when the safety profile is `ephemeral-cloud`; both the dry-run and live `runCloudUp` paths reuse that renderer, so no extra rendering was required for plan 05-02.
- The `selectCleanupArtifacts` BYO `--yes` path requires `ops.SafeRunnerPaths` to accept the install path under `/opt/actions-runner/<runnerName>`. Ephemeral down tests must therefore set `repoState.Machine.InstallPath`/`WorkDir` to match the ephemeral runner name; this is a test-fixture detail and does not affect production behavior.
- The `docs/`/`README.md` Hetzner cost caveat verification check from the plan is intentionally deferred to plan 05-03 (docs and end-to-end tests). All code-side checks (`internal/bootstrap`, `internal/cli`) succeed.

## User Setup Required

None — no `USER-SETUP.md` was generated. Live verification of an ephemeral one-job run still depends on a real GitHub repo (BYO) or a real `HCLOUD_TOKEN` (cloud), but plan 05-02 wires the live ephemeral lifecycle and is exercised end-to-end by the `internal/cli/up_ephemeral_test.go` tests with the existing fake GitHub/remote/provider services.

## Next Phase Readiness

- Plan 05-03 can build documentation, end-to-end fake tests, and the Hetzner cost caveat references in `docs/`/`README.md` on top of the implementation behaviors implemented here. The `requirements_addressed: [RUN-02]` field is now satisfied by the BYO and cloud ephemeral lifecycle.
- The `runmode` and `labels` package surfaces from plan 05-01 are unchanged, so plan 05-03 docs can quote the same constants.

## Verification

- `go test ./internal/bootstrap/...` exits 0.
- `go test ./internal/cli/... ./internal/provider/... ./internal/state/... ./internal/bootstrap/...` exits 0.
- `go test ./internal/ops/... ./internal/cli/... ./internal/state/... ./internal/bootstrap/...` exits 0.
- `go test ./...` exits 0.
- `grep -R -- "--ephemeral" internal/bootstrap internal/cli` returns matches.
- `grep -R "RenderEphemeralFinalizerScript" internal/bootstrap` returns a match.
- `grep -R "EphemeralMetadata" internal/state internal/cli` returns matches.
- `grep -R "ephemeral_completed" internal/ops internal/cli` returns matches.
- `grep -R "ephemeral.logs.preserve" internal/cli internal/bootstrap` returns matches.
- `grep -R "RunnerKit preserves best-effort logs only" internal/cli internal/ops` returns matches.
- `grep -R "No RunnerKit-managed runner is saved for" internal/cli` returns matches.
- `grep -R "Destroy ephemeral cloud runner: type" internal/cli` returns matches.
- `grep -R "Estimated cost is approximate. Hetzner pricing varies by region and time" internal/cli` returns matches. (`docs/`/`README.md` references deferred to plan 05-03.)

---

_Phase: 05-scoped-ephemeral-mode-and-safety-profiles_
_Completed: 2026-05-02_

## Self-Check: PASSED

- All key files exist on disk: `internal/cli/up_ephemeral_test.go` (created), and the 24 modified files in `internal/bootstrap`, `internal/cli`, `internal/ops`, `internal/provider`, `internal/state`.
- All task commits are present in git history: `49919cd`, `4fa5e8e`, `befee9f`, `a22b118`, `6f66313`, `1630816`.
- Required verification commands pass: `go test ./internal/bootstrap/...`, `go test ./internal/cli/... ./internal/provider/... ./internal/state/... ./internal/bootstrap/...`, `go test ./internal/ops/... ./internal/cli/... ./internal/state/... ./internal/bootstrap/...`, `go test ./...`. Required grep checks confirm `--ephemeral`, `RenderEphemeralFinalizerScript`, `EphemeralMetadata`, `ephemeral_completed`, `ephemeral.logs.preserve`, `RunnerKit preserves best-effort logs only`, `No RunnerKit-managed runner is saved for`, `Destroy ephemeral cloud runner: type`, and the exact Hetzner cost caveat all appear in the expected files.
