---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 02
subsystem: upgrade-lifecycle
tags: [migrations, lazy-update-check, runner-pin, etag, hashicorp-go-version, doctor-findings]

requires:
  - phase: 01-cli-auth-state-and-safety-foundation
    provides: SchemaVersion field, atomic state writes, ExitError contract
  - phase: 02-byo-persistent-runner-happy-path
    provides: bootstrap.RunnerVersion pin, bootstrap.Apply re-entry path
  - phase: 04-recommended-cloud-path-and-billable-cleanup
    provides: provider-aware state, target-from-state helper
  - phase: 05-scoped-ephemeral-mode-and-safety-profiles
    provides: bootstrap.ApplyEphemeral, EphemeralMetadata.FinalizerStatus, ephemeral lifecycle states
provides:
  - Forward-only state migration framework with side-by-side backup (D-09).
  - Lazy 24h-cached update notice on user-relevant commands (D-06).
  - runnerkit upgrade — print-only channel-detect command (D-07).
  - runnerkit upgrade-runner — re-applies bootstrap with bundled pin, ephemeral-aware refuse-without-force (D-08).
  - runner_version_stale doctor finding (warning) when RunnerTemplateVersion < bootstrap.RunnerVersion (D-08).
  - docs/upgrade.md three-section user-facing guide.
  - ExitStateSchemaTooNew = 7 exit code for refuse-newer-schema path.
affects: [06-01-release-packaging, 06-03-troubleshooting-docs-and-rkd-codes, 06-04-v1-validation-and-live-smoke]

tech-stack:
  added:
    - github.com/hashicorp/go-version v1.9.0 (semver comparison; same library gh CLI uses)
  patterns:
    - "Forward-only Migrate chain: forwardMigrations map[fromVersion]migrationFn ratchets state.SchemaVersion until it matches the binary's SchemaVersion. Future v2->v3 migrations attach to this map; never delete or renumber existing entries."
    - "Side-by-side backup contract: Store.Load writes state.json.backup-v<old>-<RFC3339Z> with the ORIGINAL bytes BEFORE invoking Migrate. The backup persists even if migration logic itself fails."
    - "Refuse-to-mutate exit gate: ErrSchemaTooNew is returned (no backup, no rewrite) when on-disk schema_version is newer than the binary knows; maps to ExitStateSchemaTooNew = 7."
    - "Silent-on-failure update notifier: MaybePrint exits silently on every error path (jsonOutput, $CI, $RUNNERKIT_NO_UPDATE_NOTIFIER, network error, parse error, non-200/304). Notice prints once per 24h with conditional GET (If-None-Match)."
    - "Defer-after-render integration: runUp/runStatus/runDoctor each `defer maybeShowUpdateNotice(deps, jsonOutput)` so the notice fires AFTER renderer output and never interleaves with structured stdout."
    - "Print-only upgrade channel detection: detectChannel inspects the resolved binary path; Cellar/Caskroom -> homebrew, otherwise binary, otherwise unknown. Latest version is read from the lazy update cache (no network call)."
    - "Ephemeral lifecycle gate in upgrade-runner: completed/ttl_expired -> no-op with explanatory message; waiting/busy/empty -> refuse without --force. State.RunnerTemplateVersion bump is committed only after Apply returns nil."

key-files:
  created:
    - internal/state/migrations_test.go (4 D-09 tests)
    - internal/cli/exit_test.go (ExitStateSchemaTooNew=7 contract)
    - internal/update/check.go (MaybePrint + cache + conditional GET)
    - internal/update/check_test.go (6 D-06 tests including CI/no-update-notifier bonuses)
    - internal/update/version.go (IsNewer wrapper over hashicorp/go-version)
    - internal/cli/update_notice.go (CLI->update integration adapter)
    - internal/cli/upgrade.go (runnerkit upgrade)
    - internal/cli/upgrade_test.go (5 D-07 tests)
    - internal/cli/upgrade_runner.go (runnerkit upgrade-runner)
    - internal/cli/upgrade_runner_test.go (3 D-08 tests)
    - internal/ops/doctor_runner_pin_test.go (TestDoctor_StaleRunnerVersion)
    - docs/upgrade.md (user-facing 3-section upgrade guide)
  modified:
    - internal/state/schema.go (SchemaVersion: "1" -> "2")
    - internal/state/migrations.go (full body replaced with forward-only chain)
    - internal/state/store.go (Load writes side-by-side backup before Migrate, persists migrated state)
    - internal/state/state_test.go (TestStoreSavesVersionedSecretFreeStateAtomically asserts schema_version "2" after bump)
    - internal/cli/exit.go (ExitStateSchemaTooNew = 7)
    - internal/cli/up.go (defer maybeShowUpdateNotice)
    - internal/cli/status.go (defer maybeShowUpdateNotice)
    - internal/cli/doctor.go (defer maybeShowUpdateNotice)
    - internal/cli/root.go (registers newUpgradeCommand and newUpgradeRunnerCommand)
    - internal/ops/doctor.go (runner_version_stale finding when RunnerTemplateVersion < bootstrap.RunnerVersion)
    - go.mod / go.sum (hashicorp/go-version v1.9.0 direct require)

key-decisions:
  - "Side-by-side backup is taken in Store.Load (not Migrate) because Migrate operates on a parsed struct and the backup must persist the ORIGINAL bytes byte-for-byte, including unknown future fields. Load reads bytes -> validates -> unmarshals -> writes backup if migrating -> calls Migrate -> persists migrated state via the existing atomic-write helper."
  - "writeBackup intentionally does NOT chmod the parent directory back to 0700 during a load. The directory was created with 0700 by Store.Save; silently relaxing perms during a load would surprise users who tightened them externally and was previously breaking the atomic-write test by undoing the test's chmod 0500 setup."
  - "Snake_case finding ID `runner_version_stale` (not `RKD-BOOT-002`) is used here. Plan 06-03 owns the RKD code mapping; this plan keeps the existing finding-ID convention so the cross-file change in doctor.go is line-additive (no rename collision with 06-03)."
  - "runnerkit upgrade reads the latest release tag from the lazy update-check cache rather than fetching over the network. This makes the command instantaneous and deterministic in CI; the cache is populated by the silent lazy update check in up/status/doctor."
  - "upgrade-runner's --yes is mandatory. Plan 02/04 require plan-before-mutation but the upgrade-runner change is a re-apply of an existing system-modifying step; gating it behind explicit --yes prevents accidental re-bootstraps from a TTY-less context. Adding an interactive prompt is deferred to a future change if user feedback warrants it."
  - "ExitStateSchemaTooNew = 7 is wedged between ExitInputRequired = 6 and ExitCanceled = 130 to keep the canonical 130 reserved for SIGINT-style cancels."

patterns-established:
  - "forwardMigrations map[string]migrationFn — every future state schema bump adds one entry mapping (N -> N+1). The chain advances state.SchemaVersion by one step per iteration and refuses to loop or stall (cmpVersion check after each step)."
  - "Lazy update-check cache file at <state-dir>/update-check.json (mode 0600, atomic-ish tmp+rename writes). The cache lives next to state.json so XDG_STATE_HOME and RUNNERKIT_STATE_DIR continue to control its location."
  - "Conditional GET on the GitHub Releases API (If-None-Match + ETag from cache) — 304 keeps the cache value unchanged but bumps LastCheck; 200 overwrites the cache; any other status returns silently."
  - "Defer-after-render call site for the lazy update notice — the notice goes to deps.Err while structured output goes to deps.Out, so they don't interleave on a TTY."
  - "Print-only upgrade flow as a forcing function for documentation: every install channel must be reachable via the printed instructions, otherwise the unknown branch points users at docs/upgrade.md."

requirements-completed: [REL-05]

duration: 12m
completed: 2026-05-02
---

# Phase 6 Plan 02: Upgrade and State Migration Summary

**Lazy 24h-cached update notifier, channel-detecting `runnerkit upgrade`, idempotent `upgrade-runner` re-bootstrap, forward-only state migration framework with side-by-side backup, and a stale-runner doctor warning — wired through to docs/upgrade.md.**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-05-02T20:21:21Z
- **Completed:** 2026-05-02T20:33:05Z
- **Tasks:** 3 of 3 (all auto, all TDD)
- **Files modified:** 21 (12 created, 9 modified including state_test.go regression update)

## Accomplishments

- **State migrations are now upgrade-safe (REL-05).** Loading a v1 state.json writes a sibling `state.json.backup-v1-<RFC3339Z>` of the original bytes, runs migrateV1ToV2 in-memory, and atomically saves a v2 state.json. Loading a state with a newer schema_version refuses to mutate, returns `ErrSchemaTooNew`, and exits with code 7.
- **Users get a single non-blocking `runnerkit X.Y.Z available` notice on stderr** when running `up`/`status`/`doctor`, with a 24h cache, conditional GET via ETag, and silent paths for `--json`, `$CI`, `$RUNNERKIT_NO_UPDATE_NOTIFIER`, network failures, and same-version responses.
- **`runnerkit upgrade` prints (not executes) channel-correct upgrade commands.** Detects Homebrew Cellar/Caskroom paths and falls back to "binary" for other locations and "unknown" otherwise. JSON contract: `{ok, channel, commands, current, latest, notes?}`.
- **`runnerkit upgrade-runner` re-applies the bundled runner pin** via `bootstrap.Apply` (persistent) or `bootstrap.ApplyEphemeral` (ephemeral). Refuses without `--force` when an ephemeral runner is in `waiting`/`busy`/empty FinalizerStatus; no-ops on `completed`/`ttl_expired`. State.RunnerTemplateVersion is updated only after the bootstrap returns nil.
- **`runnerkit doctor` warns on stale runner pins** with the new `runner_version_stale` finding.

## Task Commits

Each task followed RED -> GREEN TDD:

1. **Task 1: State migration framework**
   - RED: `c417bf0` (test) — TestMigrate_V1ToV2_ForwardOnly, TestMigrate_WritesBackupBeforeMutation, TestMigrate_RefusesNewerSchema, TestMigrate_Atomic, TestExitCodeStateSchemaTooNew.
   - GREEN: `8b2d2e4` (feat) — SchemaVersion bump, forward-only Migrate chain, ErrSchemaTooNew sentinel, side-by-side backup in Load, ExitStateSchemaTooNew=7.
2. **Task 2: Lazy update check + integration**
   - RED: `a3f9cc3` (test) — 6 MaybePrint tests covering JSON/cache/no-net/conditional-GET/$CI/$RUNNERKIT_NO_UPDATE_NOTIFIER.
   - GREEN: `9a9fe12` (feat) — internal/update package, internal/cli/update_notice.go, defer integration into up/status/doctor.
3. **Task 3: Upgrade commands and doctor finding**
   - RED: `9bdafa4` (test) — 8 tests for detectChannel/JSON contract/upgrade-runner persistent+ephemeral terminal+waiting/doctor stale finding.
   - GREEN: `bc8d2f2` (feat) — runnerkit upgrade, runnerkit upgrade-runner, runner_version_stale finding, docs/upgrade.md.

## Files Created/Modified

### Created
- `internal/state/migrations_test.go` — D-09 contract tests (forward, backup, refuse-newer, atomic).
- `internal/cli/exit_test.go` — ExitStateSchemaTooNew = 7 contract.
- `internal/update/check.go` — `MaybePrint`, `CheckedRelease`, atomic-ish cache writes.
- `internal/update/version.go` — `IsNewer` wrapper.
- `internal/update/check_test.go` — 6 silent-path tests.
- `internal/cli/update_notice.go` — `maybeShowUpdateNotice` adapter.
- `internal/cli/upgrade.go` — `runnerkit upgrade` + `detectChannel`.
- `internal/cli/upgrade_test.go` — channel detection + JSON/human contract.
- `internal/cli/upgrade_runner.go` — `runnerkit upgrade-runner`.
- `internal/cli/upgrade_runner_test.go` — persistent + ephemeral terminal/waiting tests.
- `internal/ops/doctor_runner_pin_test.go` — `TestDoctor_StaleRunnerVersion`.
- `docs/upgrade.md` — three-section user-facing guide (CLI upgrade, runner pin, state migrations) with cosign verification snippet.

### Modified
- `internal/state/schema.go` — `SchemaVersion` constant `"1"` -> `"2"`.
- `internal/state/migrations.go` — full body replaced with forward-only chain + ErrSchemaTooNew + cmpVersion.
- `internal/state/store.go` — `Load` now writes side-by-side backup before Migrate, refuses newer schema, persists migrated state via existing atomic-write helper.
- `internal/state/state_test.go` — `TestStoreSavesVersionedSecretFreeStateAtomically` asserts `schema_version "2"` (regression update for SchemaVersion bump; pre-existing v1 fixture loaders still pass — they exercise the new migration path).
- `internal/cli/exit.go` — adds `ExitStateSchemaTooNew = 7`.
- `internal/cli/up.go` / `status.go` / `doctor.go` — single `defer maybeShowUpdateNotice(deps, jsonOutput)` line each.
- `internal/cli/root.go` — registers `newUpgradeCommand` and `newUpgradeRunnerCommand` after `newStateCommand`.
- `internal/ops/doctor.go` — adds `runner_version_stale` warning finding (placed after `cleanup_pending`); imports `internal/bootstrap`.
- `go.mod` / `go.sum` — `github.com/hashicorp/go-version v1.9.0` as direct require.

## Decisions Made

See `key-decisions` in the frontmatter. The notable ones:

- Backup is written in `Store.Load` (not `Migrate`) so it captures the ORIGINAL raw bytes byte-for-byte. Migrate continues to operate on a parsed struct.
- `writeBackup` deliberately skips the chmod-to-0700 step that the existing `writeAtomic` helper does — relaxing perms during a load would surprise externally-tightened state dirs (and was breaking the atomic-write test by undoing the test's chmod 0500).
- The doctor finding uses snake_case `runner_version_stale`. Plan 06-03 will map it to `RKD-BOOT-002` via the new `internal/errcodes` package without renaming the finding ID.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Pre-existing test asserted `schema_version "1"` after the SchemaVersion bump**

- **Found during:** Task 1 GREEN.
- **Issue:** `TestStoreSavesVersionedSecretFreeStateAtomically` in `internal/state/state_test.go` line 38 asserted that a freshly-saved state contained `"schema_version": "1"`. After Task 1's intentional bump to `"2"`, this test failed.
- **Fix:** Updated the assertion to `"schema_version": "2"`. The other v1 fixture-loading tests in the same file (`TestLoadBackwardsCompatibleStateWithoutHostKeyFields`, `TestSafetyMetadataPersistsSafetyProfile`, `TestEphemeralMetadataPersistsAndIsBackwardsCompatible`) intentionally still write v1 fixtures — they now exercise the new v1->v2 migration path and continue to pass, which validates that the migration is field-preserving.
- **Files modified:** `internal/state/state_test.go` (one assertion line).
- **Verification:** `go test ./internal/state -count=1` passes.
- **Committed in:** `8b2d2e4` (Task 1 GREEN, alongside the SchemaVersion bump).

**2. [Rule 1 - Bug] writeBackup helper was undoing test-induced chmod 0500 by chmod-ing to 0700**

- **Found during:** Task 1 GREEN, while diagnosing `TestMigrate_Atomic` returning nil instead of an error.
- **Issue:** My initial `writeBackup` helper mirrored the existing `writeAtomic` pattern of `os.MkdirAll(dir, 0700)` followed by `os.Chmod(dir, 0700)`. The chmod silently relaxed perms on the test's deliberately-restricted dir, allowing the backup write to succeed when the test expected it to fail.
- **Fix:** Removed the chmod-to-0700 from `writeBackup`. The directory was already created with 0700 by `Store.Save` on initial state file creation; relaxing perms during a load is a behavior surprise (a user may have tightened perms externally) and is unnecessary.
- **Files modified:** `internal/state/store.go`.
- **Verification:** `TestMigrate_Atomic` now correctly observes the ENOENT/EACCES on the read-only dir.
- **Committed in:** `8b2d2e4` (Task 1 GREEN, before the commit).

## Task Verification

- **Task 1:** `go test ./internal/state -run 'TestMigrate_V1ToV2_ForwardOnly|TestMigrate_WritesBackupBeforeMutation|TestMigrate_RefusesNewerSchema|TestMigrate_Atomic' && go test ./internal/cli -run TestExitCodeStateSchemaTooNew` — PASS.
- **Task 2:** `go test ./internal/update -run 'TestMaybePrint_*'` (6/6) + `grep -q "defer maybeShowUpdateNotice" internal/cli/{up,status,doctor}.go` (3/3) + `grep -q "github.com/hashicorp/go-version" go.sum` — PASS.
- **Task 3:** `go test ./internal/cli -run 'TestUpgrade_*|TestUpgradeRunner_*'` (8/8) + `go test ./internal/ops -run TestDoctor_StaleRunnerVersion` + grep checks for `newUpgradeCommand`/`newUpgradeRunnerCommand`/`runner_version_stale`/`docs/upgrade.md` — PASS.
- **Plan-level:** `go test ./... -count=1 -race` — all 14 packages pass; no test files in `cmd/runnerkit` and `internal/testsupport` (unchanged).

## Validation matrix coverage (06-VALIDATION.md)

- **D-06 lazy update check (lines 54-57, 4 tests):** all green via internal/update package + bonus CI/no-update-notifier tests.
- **D-07 upgrade channel detect (lines 58-60, 3 tests):** all green via internal/cli/upgrade_test.go.
- **D-08 upgrade-runner persistent + ephemeral terminal/waiting (lines 61-63, 3 tests):** all green via internal/cli/upgrade_runner_test.go.
- **D-08 doctor stale-runner finding (line 64):** green via internal/ops/doctor_runner_pin_test.go.
- **D-09 state migration: forward, backup, refuse-newer, atomic (lines 65-68, 4 tests):** all green via internal/state/migrations_test.go.

All 16 D-06..D-09 validation rows are green at end of plan.

## Cross-plan notes

- `internal/ops/doctor.go` is also touched by Plan 06-03 to wire RKD codes via `internal/errcodes`. The two changes do NOT collide: this plan adds new lines (the `runner_version_stale` add() call); Plan 06-03 will replace the literal finding-ID strings with errcode lookups but does not rename the finding ID.
- `bootstrap.RunnerVersion` is now consumed by `internal/cli/upgrade_runner.go` AND `internal/ops/doctor.go`. Future bumps to the bundled pin should land in a single PR alongside the `RunnerKitVersion` bump so doctor's stale finding is consistent with the upgrade-runner outcome.

## Self-Check: PASSED

All 12 created files exist on disk; all 6 task commits present in git log; full test suite (`go test ./... -count=1 -race`) green; `go vet ./...` clean.
