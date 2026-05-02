---
phase: 05-scoped-ephemeral-mode-and-safety-profiles
plan: "03"
subsystem: safety-docs-and-end-to-end-validation
tags: [docs, safety, ephemeral, persistent, e2e, regression, status, logs, doctor, down, destroy]
requires:
  - phase: 05-scoped-ephemeral-mode-and-safety-profiles-01
    provides: runmode constants, mode-aware safety enforcement, ephemeral cloud cost caveat copy, ephemeral runner naming, persistent vs ephemeral tradeoff bullets
  - phase: 05-scoped-ephemeral-mode-and-safety-profiles-02
    provides: ephemeral bootstrap lifecycle, EphemeralMetadata state, status.ephemeral.state remote sentinel, logs.ephemeral.archive.* commands, ephemeral.logs.preserve cleanup ordering, ephemeral destroy prompt copy
provides:
  - docs/safety.md Self-hosted Runner Safety Guide with the UI-SPEC headings and exact non-goal bullets
  - README persistent/ephemeral runs-on snippets, ephemeral cloud commands, exact Hetzner cost caveat, and link to docs/safety.md
  - docs/byo-quickstart.md persistent-self-hosted-unsafe sentence and ephemeral cloud recommendation replacing the deferred-ephemeral wording
  - docs/cloud-quickstart.md ephemeral cloud setup commands, billable-Hetzner-resources sentence, and removal of the 'Ephemeral mode is deferred to Phase 5.' wording
  - TestSafetyGuideDocsContainRequiredCopy and TestSafetyDocsGrepContract docs assertions in internal/cli/docs_test.go
  - internal/cli/up_ephemeral_e2e_test.go with five fake end-to-end Cobra paths covering trusted persistent, public-blocked persistent, public ephemeral cloud, public ephemeral BYO acknowledgement, and trusted private ephemeral BYO cleanup commands
  - testsupport.EphemeralBYORepositoryState/EphemeralCloudRepositoryState fixtures with the deterministic ephemeral runner name `runnerkit-owner-repo-ephemeral-20260501t183000`
  - TestStatusEphemeralCompletedNeedsCleanup, TestMissingStateRendersRunnerKitEmptyState, and TestStatusEphemeralTTLExpired status regressions
  - TestLogsEphemeralPreservedArchive logs regression covering Runner_*.log/Worker_*.log/systemd-journal.log and the production-grade forwarding caveat
  - TestDoctorEphemeralCompletedRecommendsCleanup doctor regression for cloud and BYO ephemeral cleanup remediation
  - TestDownEphemeralPreservesLogsBeforeRemovingFiles and TestDestroyEphemeralCloudPreservesLogsBeforeProviderDelete cleanup-ordering regressions
  - mergeWarnings de-duplication so safety + mode warnings flow into ephemeral BYO/cloud completion output exactly once
  - classifyEphemeral preference for the live remote sentinel finalizer status so completed/ttl_expired classifications fire even when saved state still records 'pending'
affects: [phase-06, docs, cli, ops, testsupport]
tech-stack:
  added: []
  patterns:
    - documentation regression contract via docs grep tests so safety copy cannot silently regress
    - end-to-end Cobra fake tests using testsupport.GitHubService + testsupport.RemoteExecutor + provider.FakeProvider + a fixed clock + a deterministic ephemeral short id seam
    - shared EphemeralBYO/EphemeralCloud RepositoryState fixtures so status, logs, doctor, down, and destroy regressions exercise identical ephemeral state shapes
    - mode-decision warnings flow into completion output through mergeWarnings de-duplication
    - classifyEphemeral preferring observed (remote sentinel) finalizer status over saved state for live terminal classification
key-files:
  created:
    - docs/safety.md
    - internal/cli/up_ephemeral_e2e_test.go
    - .planning/phases/05-scoped-ephemeral-mode-and-safety-profiles/05-03-SUMMARY.md
  modified:
    - README.md
    - docs/byo-quickstart.md
    - docs/cloud-quickstart.md
    - internal/cli/docs_test.go
    - internal/cli/status_test.go
    - internal/cli/logs_test.go
    - internal/cli/doctor_test.go
    - internal/cli/down_test.go
    - internal/cli/destroy_test.go
    - internal/cli/up.go
    - internal/ops/status.go
    - internal/testsupport/state.go
key-decisions:
  - "docs/safety.md owns the canonical Phase 5 safety copy; README, BYO quickstart, and cloud quickstart link to it and only repeat the sentences required by the docs grep contract so future copy edits stay in one source of truth."
  - "Test fixtures use the deterministic ephemeral runner name runnerkit-owner-repo-ephemeral-20260501t183000 and clock 2026-05-01T18:30:00Z so status, logs, doctor, down, and destroy regressions assert exact ephemeral artifact paths."
  - "End-to-end up tests wrap testsupport.GitHubService with a stagedListRunnersGitHub helper so the first ListRunners call returns nil (duplicate-runner check passes) and the second returns the deterministic ephemeral runner (waitForRunnerOnline succeeds), without forking the underlying testsupport contract."
  - "modeDecision.Warnings (notably the public/fork ephemeral cloud 'Use ephemeral cloud runner: ...' recommendation) now merge into ephemeral BYO and cloud completion human/JSON output via mergeWarnings, with de-duplication against canonical per-profile copy so no sentence appears twice."
  - "classifyEphemeral prefers the observed remote sentinel finalizer status over the saved RepositoryState so a freshly-completed or TTL-expired ephemeral runner is classified as terminal even when state on disk still records 'pending'."
patterns-established:
  - "Documentation regressions via Go test docs grep contract (TestSafetyGuideDocsContainRequiredCopy + TestSafetyDocsGrepContract) protect every required heading, command, sentence, and v1 non-goal bullet."
  - "End-to-end fake CLI tests prove safety decisions without minting a real registration token, executing a real remote command, or calling a live provider; side-effect counters assert no leakage on safety-gate paths."
  - "EphemeralBYORepositoryState/EphemeralCloudRepositoryState fixtures live in testsupport and are reused by status/logs/doctor/down/destroy tests so changes to ephemeral state shape touch one file."
requirements-completed: [DOC-03, RUN-04, RUN-02]
duration: 16 min
completed: 2026-05-02
---

# Phase 05 Plan 03: Safety Docs and Risky-Workload E2E Validation Summary

**`docs/safety.md`, README, BYO quickstart, and cloud quickstart now explain exactly when persistent runners are unsafe and when ephemeral mode is recommended; fake end-to-end CLI tests plus lifecycle regression tests prove RunnerKit's UX and one-job semantics for trusted persistent, public-blocked persistent, public ephemeral cloud, and BYO ephemeral scenarios without live GitHub or Hetzner calls.**

## Performance

- **Duration:** 16 min
- **Started:** 2026-05-02T15:45:54Z
- **Completed:** 2026-05-02T16:01:15Z
- **Tasks:** 3
- **Files modified:** 13 (3 created + 10 modified)

## Accomplishments

- Added `docs/safety.md` with the full UI-SPEC heading set: `# Self-hosted Runner Safety Guide`, `## Quick recommendation`, `## Persistent vs ephemeral tradeoffs`, `## When persistent is appropriate`, `## When ephemeral is recommended`, `## Public and fork-based workflow risk`, `## BYO ephemeral caveats`, `## Cloud ephemeral caveats`, `## Logs and troubleshooting`, `## Cleanup commands`, and `## What RunnerKit does not do in v1`. The guide includes the persistent vs ephemeral tradeoffs table (Mode/Cost/Isolation/Cleanup/Operations/Logs columns and `persistent`/`ephemeral` rows), the exact Hetzner cost caveat, the BYO clean-VM caveat, the not-fleet-manager warning, the production-grade external log forwarding caveat, and the v1 non-goal bullets (no hosted control plane, no webhook listener or autoscaling fleet manager, no ARC/Kubernetes/scale sets/JIT runner API, no automatic workflow YAML edits, no clean-VM guarantee for BYO ephemeral).
- Updated README with a `Safety: persistent vs ephemeral` section that links to `docs/safety.md`, calls out the `persistent self-hosted runners` lower-case validation phrase, lists the ephemeral cloud command examples, prints both the persistent and ephemeral `runs-on` snippets, and surfaces the exact `Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until \`runnerkit destroy --repo owner/name\` verifies cleanup.` caveat.
- Updated `docs/byo-quickstart.md` to add `Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.` and replace the `wait for RunnerKit's future ephemeral mode` wording with `Use runnerkit up --repo owner/name --mode ephemeral --cloud hetzner for stronger isolation, or use GitHub-hosted runners.`.
- Updated `docs/cloud-quickstart.md` to remove `Ephemeral mode is deferred to Phase 5.`, add the ephemeral cloud setup commands (`--mode ephemeral --cloud hetzner`, `--yes`, `--ephemeral-ttl 24h`), and surface `Ephemeral cloud runners still create billable Hetzner resources.`, `Billing stops only after \`runnerkit destroy --repo owner/name\` verifies cleanup.`, and the exact Hetzner cost caveat.
- Added `internal/cli/up_ephemeral_e2e_test.go` with five fake end-to-end Cobra paths that prove: trusted private persistent dry-run still uses `Default mode: persistent for trusted private repositories.` and the persistent runs-on snippet without provider calls; public persistent setup blocks with `ExitSafetyGate` before any GitHub auth, runner-management read, registration token, remote command, or provider call and surfaces the ephemeral cloud command; public ephemeral cloud completion includes `Use ephemeral cloud runner`, the exact Hetzner cost caveat, `Ephemeral runner ready`, the one-job/automatic-deregister sentence, the not-fleet-manager warning, and `Cleanup after the job: runnerkit destroy --repo owner/name`, with `safety_profile: ephemeral-cloud` and `provider.ProvisionInput.Mode == "ephemeral"`; public ephemeral BYO without `--allow-ephemeral-byo-risk` blocks before remote probe and registration token while surfacing the BYO clean-VM caveat; trusted private ephemeral BYO writes `cleanup_command: runnerkit down --repo owner/name` with no provider calls.
- Added `internal/testsupport/state.go::EphemeralBYORepositoryState` and `EphemeralCloudRepositoryState` fixtures with the deterministic ephemeral runner name `runnerkit-owner-repo-ephemeral-20260501t183000`, ephemeral labels, the `/var/lib/runnerkit/ephemeral/<runner>/logs` archive path, and the per-mode cleanup commands (`runnerkit down --repo owner/repo` for BYO, `runnerkit destroy --repo owner/repo` for cloud).
- Added Phase 5 lifecycle regression tests in `internal/cli/status_test.go`, `logs_test.go`, `doctor_test.go`, `down_test.go`, and `destroy_test.go`: `TestStatusEphemeralCompletedNeedsCleanup` proves the cloud ephemeral status surfaces `Ephemeral runner completed one job and needs cleanup.`, `Next: runnerkit destroy --repo owner/repo`, the saved log archive, and never surfaces the persistent `runnerkit recover` hint; `TestMissingStateRendersRunnerKitEmptyState` proves the UI-SPEC `No RunnerKit-managed runner is saved for \`owner/name\`.` empty-state copy with the ephemeral cloud remediation; `TestStatusEphemeralTTLExpired` proves `Ephemeral runner TTL expired before a job completed. Run cleanup now.` plus the `ephemeral_ttl_expired` JSON reason; `TestLogsEphemeralPreservedArchive` proves `Runner_*.log`, `Worker_*.log`, `systemd-journal.log`, and the external-log-forwarding caveat; `TestDoctorEphemeralCompletedRecommendsCleanup` proves the `ephemeral_completed` finding plus the per-mode cleanup remediation in `--verbose` doctor output; `TestDownEphemeralPreservesLogsBeforeRemovingFiles` proves `ephemeral.logs.preserve` runs before `down.files.remove`; `TestDestroyEphemeralCloudPreservesLogsBeforeProviderDelete` proves `ephemeral.logs.preserve` runs once before exactly one `Provider.Destroy`, that the JSON envelope includes `provider_verification`, and that the typed-confirmation prompt copy uses the dedicated ephemeral cloud destroy template.
- Added `TestSafetyDocsGrepContract` to `internal/cli/docs_test.go` mirroring the validation strategy commands so README/docs cannot silently regress on `persistent self-hosted runners`, `Ephemeral mode is not a fleet manager`, the exact Hetzner cost caveat, the production-grade external log forwarding caveat, or accidentally introduce a promotional `autoscaling fleet manager` mention outside the v1 non-goal sentence.

## Task Commits

1. **Task 05-03-01: Create safety guide and update README/BYO/cloud docs with exact commands and caveats** — `cf6203e` (test) + `9438f00` (feat)
2. **Task 05-03-02: Add fake end-to-end tests for trusted persistent and risky ephemeral workflow scenarios** — `fc67c4d` (feat)
3. **Task 05-03-03: Add lifecycle regression tests for TTL, preserved logs, cleanup, docs grep, and full suite** — `98ec323` (feat)

## Files Created/Modified

### Created

- `docs/safety.md` — Self-hosted Runner Safety Guide with all UI-SPEC headings, exact commands, persistent vs ephemeral tradeoffs table, BYO clean-VM caveat, ephemeral cloud cost caveat, log preservation copy, cleanup commands, and v1 non-goals.
- `internal/cli/up_ephemeral_e2e_test.go` — Five fake end-to-end Cobra paths plus the `ephemeralE2EDeps` and `stagedListRunnersGitHub` helpers wiring fake GitHub, remote, provider, state, and a fixed clock without live credentials.
- `.planning/phases/05-scoped-ephemeral-mode-and-safety-profiles/05-03-SUMMARY.md` — This summary.

### Modified

- `README.md` — Adds the Safety section linking `docs/safety.md`, ephemeral cloud commands, exact Hetzner cost caveat, both persistent and ephemeral runs-on snippets, and `Use ephemeral cloud runner` callout.
- `docs/byo-quickstart.md` — Adds the persistent-self-hosted-unsafe sentence, replaces the deferred-ephemeral wording with the ephemeral cloud recommendation, and links the safety guide.
- `docs/cloud-quickstart.md` — Removes the `Ephemeral mode is deferred to Phase 5.` wording, adds ephemeral cloud setup commands, billable-Hetzner-resources sentence, and the exact Hetzner cost caveat.
- `internal/cli/docs_test.go` — Adds `TestSafetyGuideDocsContainRequiredCopy` and `TestSafetyDocsGrepContract`.
- `internal/cli/status_test.go` — Adds `TestStatusEphemeralCompletedNeedsCleanup`, `TestMissingStateRendersRunnerKitEmptyState`, and `TestStatusEphemeralTTLExpired`.
- `internal/cli/logs_test.go` — Adds `TestLogsEphemeralPreservedArchive`.
- `internal/cli/doctor_test.go` — Adds `TestDoctorEphemeralCompletedRecommendsCleanup`.
- `internal/cli/down_test.go` — Adds `TestDownEphemeralPreservesLogsBeforeRemovingFiles`.
- `internal/cli/destroy_test.go` — Adds `TestDestroyEphemeralCloudPreservesLogsBeforeProviderDelete` (JSON cleanup ordering + interactive prompt copy).
- `internal/cli/up.go` — Pipes `modeDecision.Warnings` into ephemeral BYO and cloud completion human/JSON output via `mergeWarnings`; adds de-duplication so the same warning sentence does not appear twice.
- `internal/ops/status.go` — `classifyEphemeral` prefers the observed remote sentinel finalizer status over saved state so completed/ttl_expired classifications fire when the saved state still records 'pending'.
- `internal/testsupport/state.go` — Adds `EphemeralBYORepositoryState` and `EphemeralCloudRepositoryState` fixtures.

## Decisions Made

- `docs/safety.md` owns the canonical safety copy. README, BYO quickstart, and cloud quickstart link to the safety guide and only repeat the sentences required by the docs grep contract.
- The deterministic ephemeral runner name `runnerkit-owner-repo-ephemeral-20260501t183000` and clock `2026-05-01T18:30:00Z` are shared across the EphemeralBYO and EphemeralCloud fixtures so status/logs/doctor/down/destroy tests assert identical ephemeral artifact paths.
- The `stagedListRunnersGitHub` test helper wraps `*testsupport.GitHubService` so the first `ListRunners` returns nil (duplicate-runner check passes) and the second returns the deterministic ephemeral runner (`waitForRunnerOnline` succeeds), without forking the underlying testsupport contract.
- Doctor regression tests run with `--verbose` because non-verbose doctor hides `pass` findings (including `ephemeral_completed`); the test reflects the user-visible behavior in both modes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] Mode-decision warnings did not reach ephemeral BYO/cloud completion output**

- **Found during:** Task 05-03-02 (`TestEphemeralCloudRecommendedForPublicRepo`).
- **Issue:** `enforceModeSafetyDecision` appends safer-recommendation warnings (notably `Use ephemeral cloud runner: runnerkit up --repo owner/name --mode ephemeral --cloud hetzner` for public/fork ephemeral cloud) to `*runmode.Decision.Warnings`. The ephemeral BYO and cloud completion paths only rendered `gh.SafetyDecision.Warnings`, so the `Use ephemeral cloud runner` callout never reached the user-visible output even though the docs grep contract and UI-SPEC require it.
- **Fix:** `runBYOUp` and `runCloudUp` now call `mergeWarnings(decision.Warnings, modeDecision.Warnings)` when invoking `renderEphemeralCompletionHuman` and `ephemeralCompletionJSON`. `mergeWarnings` itself is now de-duplicating so the canonical per-profile copy already rendered upstream is not repeated.
- **Files modified:** `internal/cli/up.go`.
- **Verification:** `go test ./internal/cli/...`.
- **Committed in:** `fc67c4d`.

**2. [Rule 1 - Bug] classifyEphemeral did not honor the live remote sentinel finalizer status**

- **Found during:** Task 05-03-03 (`TestStatusEphemeralCompletedNeedsCleanup`, `TestStatusEphemeralTTLExpired`).
- **Issue:** `internal/ops/status.classifyEphemeral` read `repoState.Ephemeral.FinalizerStatus` (saved state) and ignored `observed.Ephemeral.FinalizerStatus` (the value `collectEphemeralFact` populates from the `status.ephemeral.state` remote sentinel). A freshly-completed or TTL-expired ephemeral runner with `FinalizerStatus="pending"` on disk could never be classified as `ephemeral_completed`/`ephemeral_ttl_expired`, so status surfaced the persistent `service_inactive`/`runnerkit recover` hint instead.
- **Fix:** `classifyEphemeral` now prefers the observed finalizer status when set and falls back to the saved status only when the observed value is empty. Doctor still reads `repoState.Ephemeral.FinalizerStatus` directly because doctor is intentionally summary-only and does not run the live sentinel probe.
- **Files modified:** `internal/ops/status.go`.
- **Verification:** `go test ./internal/cli/... ./internal/ops/...`.
- **Committed in:** `98ec323`.

### Test-Tolerance Adjustments

**3. [Test ergonomics] Long UI-SPEC sentences wrap at 80–100 columns**

- **Found during:** Tasks 05-03-02 and 05-03-03.
- **Issue:** Tests assert literal long sentences that the `internal/ui` renderer wraps across multiple lines (e.g., the exact Hetzner cost caveat, the not-fleet-manager warning, the ephemeral preserved logs forwarding caveat, the empty-state remediation).
- **Fix:** Tests flatten internal whitespace runs (`strings.Join(strings.Fields(out), " ")`) before asserting the canonical sentences. JSON output is asserted directly because JSON is not wrapped.
- **Files modified:** `internal/cli/up_ephemeral_e2e_test.go`, `internal/cli/status_test.go`, `internal/cli/logs_test.go`, `internal/cli/doctor_test.go`.
- **Committed in:** `fc67c4d`, `98ec323`.

---

**Total deviations:** 1 missing-functionality fix (Rule 2), 1 bug fix (Rule 1), 1 test-tolerance adjustment.
**Impact on plan:** Strengthens plan-required behavior (Rules 1-2 fixes); test-tolerance keeps the literal UI-SPEC content unchanged.

## Issues Encountered

- The `--json` mode of `runnerkit destroy` requires `--yes` (it does not run typed-confirmation interactively). The destroy regression test therefore runs once with `--yes --json` to assert log preservation ordering and `provider_verification`, then runs a second time interactively against a fresh state directory to assert the prompt copy.
- The `internal/cli` test infrastructure does not let `executeForTest` inject a fake provider, so the new E2E tests construct `Dependencies` directly; this matches existing 05-01/05-02 patterns.

## User Setup Required

None — no `USER-SETUP.md` was generated. A controlled live cloud-ephemeral one-job smoke remains recommended before public release because it requires real GitHub + `HCLOUD_TOKEN` credentials and creates billable Hetzner resources.

## Next Phase Readiness

- Phase 6 release validation can rely on `go test ./...`, `grep -R "persistent self-hosted runners" README.md docs`, `grep -R "Ephemeral mode is not a fleet manager" README.md docs`, and `grep -R "Estimated cost is approximate. Hetzner pricing varies by region and time" README.md docs` as a regression contract.
- The fake E2E coverage in `internal/cli/up_ephemeral_e2e_test.go` and the Phase 5 lifecycle regressions in status/logs/doctor/down/destroy give Phase 6 a known-good baseline before swapping in any live smoke test fixtures.

## Verification

- `go test ./internal/cli/... && grep -R "runnerkit up --repo owner/name --mode ephemeral --cloud hetzner" README.md docs && grep -R "Self-hosted Runner Safety Guide" README.md docs/safety.md` exits 0.
- `go test ./internal/cli/... ./internal/bootstrap/... ./internal/provider/... ./internal/state/...` exits 0.
- `go test ./... && grep -R "persistent self-hosted runners" README.md docs && grep -R "Ephemeral mode is not a fleet manager" README.md docs && grep -R "Estimated cost is approximate. Hetzner pricing varies by region and time" README.md docs` exits 0.
- `grep -R "# Self-hosted Runner Safety Guide" docs/safety.md` returns a match.
- `grep -R "BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine." README.md docs` returns matches.
- `grep -R "Billing stops only after \`runnerkit destroy --repo owner/name\` verifies cleanup." README.md docs` returns matches.
- `grep -R "Estimated cost is approximate. Hetzner pricing varies by region and time" README.md docs` returns matches.
- `grep -R "No RunnerKit-managed runner is saved for" internal/cli/*_test.go` returns matches.
- `grep -R "Destroy ephemeral cloud runner: type" internal/cli/*_test.go` returns matches.
- `grep -R "TestEphemeralCloudRecommendedForPublicRepo" internal/cli/up_ephemeral_e2e_test.go` returns a match.
- `grep -R "ephemeral.logs.preserve" internal/cli/*_test.go` returns matches.
- `go test ./...` exits 0.

---

_Phase: 05-scoped-ephemeral-mode-and-safety-profiles_
_Completed: 2026-05-02_

## Self-Check: PASSED

- All key files exist on disk: created `docs/safety.md`, `internal/cli/up_ephemeral_e2e_test.go`, `.planning/phases/05-scoped-ephemeral-mode-and-safety-profiles/05-03-SUMMARY.md`; modified `README.md`, `docs/byo-quickstart.md`, `docs/cloud-quickstart.md`, `internal/cli/docs_test.go`, `internal/cli/status_test.go`, `internal/cli/logs_test.go`, `internal/cli/doctor_test.go`, `internal/cli/down_test.go`, `internal/cli/destroy_test.go`, `internal/cli/up.go`, `internal/ops/status.go`, `internal/testsupport/state.go`.
- All task commits are present in git history: `cf6203e`, `9438f00`, `fc67c4d`, `98ec323`.
- Required verification commands pass: `go test ./internal/cli/...`, `go test ./internal/cli/... ./internal/bootstrap/... ./internal/provider/... ./internal/state/...`, `go test ./...`. Required grep checks confirm `persistent self-hosted runners`, `Ephemeral mode is not a fleet manager`, the exact Hetzner cost caveat, the safety guide heading, BYO clean-VM caveat, billing-stops-after-destroy sentence, the empty-state heading, the destroy ephemeral cloud prompt copy, `TestEphemeralCloudRecommendedForPublicRepo`, and `ephemeral.logs.preserve` all appear in the expected files.
