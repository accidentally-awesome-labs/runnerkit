---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 02
type: execute
wave: A
depends_on: []
files_modified:
  - internal/state/schema.go
  - internal/state/migrations.go
  - internal/state/migrations_test.go
  - internal/cli/exit.go
  - internal/update/check.go
  - internal/update/check_test.go
  - internal/update/version.go
  - internal/cli/update_notice.go
  - internal/cli/upgrade.go
  - internal/cli/upgrade_test.go
  - internal/cli/upgrade_runner.go
  - internal/cli/upgrade_runner_test.go
  - internal/cli/up.go
  - internal/cli/status.go
  - internal/cli/doctor.go
  - internal/cli/root.go
  - internal/ops/doctor.go
  - internal/bootstrap/package.go
  - go.mod
  - go.sum
  - docs/upgrade.md
autonomous: true
requirements: [REL-05]
must_haves:
  truths:
    - "`runnerkit up`, `runnerkit status`, `runnerkit doctor` print a single non-blocking `runnerkit X.Y.Z available` notice line on stderr when a newer release exists, AND are silent in JSON mode, AND are silent on network error, AND honor a 24h cache, AND skip when $CI is set."
    - "`runnerkit upgrade` detects whether the binary was installed via Homebrew (Cellar/Caskroom path) or as a downloaded release binary, and prints the channel-correct upgrade command. It NEVER replaces its own binary."
    - "`runnerkit upgrade-runner` re-applies bootstrap.Apply (persistent) or bootstrap.ApplyEphemeral (ephemeral) against the saved MachineRef with the bundled RunnerVersion pin, refusing without --force when an ephemeral runner is currently waiting for a job."
    - "`runnerkit doctor` emits a stale-runner-version warning finding when an installed runner version is older than `bootstrap.RunnerVersion`."
    - "Loading a state.json with `schema_version: \"1\"` runs migrateV1ToV2, writes a side-by-side `state.json.backup-v1-<RFC3339>` BEFORE mutation, and saves with `schema_version: \"2\"`. Loading a state.json with a newer schema_version than the CLI knows refuses to mutate and exits with `ExitStateSchemaTooNew`."
  artifacts:
    - path: "internal/state/migrations.go"
      provides: "Forward-only Migrate chain with side-by-side backup; ErrSchemaTooNew sentinel; migrateV1ToV2 identity migration"
      contains: "ErrSchemaTooNew"
      contains_also: "backup-v"
      contains_also2: "forwardMigrations"
    - path: "internal/state/schema.go"
      provides: "SchemaVersion constant bumped from \"1\" to \"2\""
      contains: "SchemaVersion = \"2\""
    - path: "internal/cli/exit.go"
      provides: "ExitStateSchemaTooNew = 7 exit code constant"
      contains: "ExitStateSchemaTooNew"
    - path: "internal/update/check.go"
      provides: "MaybePrint(jsonOutput, currentVersion, stateDir, errOut) — 24h cache, ETag conditional GET, silent in JSON/CI/no-net"
      contains: "MaybePrint"
      contains_also: "If-None-Match"
      contains_also2: "RUNNERKIT_NO_UPDATE_NOTIFIER"
    - path: "internal/cli/upgrade.go"
      provides: "`runnerkit upgrade` Cobra command — channel detect (homebrew/binary/unknown), print-only"
      contains: "detectChannel"
      contains_also: "/Cellar/runnerkit/"
      contains_also2: "/Caskroom/runnerkit/"
    - path: "internal/cli/upgrade_runner.go"
      provides: "`runnerkit upgrade-runner` Cobra command — re-apply bootstrap with bundled pin"
      contains: "bootstrap.Apply"
      contains_also: "bootstrap.ApplyEphemeral"
    - path: "internal/ops/doctor.go"
      provides: "stale runner version finding — id `runner_version_stale`"
      contains: "runner_version_stale"
    - path: "docs/upgrade.md"
      provides: "User-facing upgrade guide: lazy notice, channel detect, runner pin bump, state migration, schema-too-new recovery"
      contains: "runnerkit upgrade"
      contains_also: "runnerkit upgrade-runner"
      contains_also2: "schema_version"
  key_links:
    - from: "internal/cli/up.go::runUp"
      to: "internal/update.MaybePrint"
      via: "deferred call after renderer output, gated on jsonOutput flag"
      pattern: "defer .*update\\.MaybePrint|update\\.MaybePrint"
    - from: "internal/cli/status.go::runStatus"
      to: "internal/update.MaybePrint"
      via: "deferred call after renderer output, gated on jsonOutput flag"
    - from: "internal/cli/doctor.go::runDoctor"
      to: "internal/update.MaybePrint"
      via: "deferred call after renderer output, gated on jsonOutput flag"
    - from: "internal/state/store.go::Load"
      to: "internal/state/migrations.go::Migrate"
      via: "existing Load()→Migrate() dispatch, body of Migrate replaced"
    - from: "internal/cli/upgrade_runner.go"
      to: "internal/bootstrap.Apply / ApplyEphemeral"
      via: "re-entry with current bootstrap.RunnerVersion pin"
    - from: "internal/ops/doctor.go::BuildDoctorReport"
      to: "internal/bootstrap.RunnerVersion"
      via: "compare observed runner version against pin → emit `runner_version_stale` finding"
---

<objective>
Add the upgrade lifecycle that prevents runner rot per REL-05: a lazy 24h-cached update-check notice that fires on `up`/`status`/`doctor`; a `runnerkit upgrade` print-only channel-detect command; a `runnerkit upgrade-runner` command that re-applies the bootstrap with the bundled runner pin; a forward-only state migration framework with side-by-side backup and refuse-newer-schema; and a stale-runner-version doctor finding. Implements **D-06..D-09** from CONTEXT.md.

Purpose: Phase 6 success criteria 1 (documented upgrade path) and 2 (state migration safe across releases or block with guidance).

Output: A CLI where users can run `runnerkit upgrade` to learn how to update their CLI, run `runnerkit upgrade-runner` to roll the bundled runner forward without re-running setup, get warned by `runnerkit doctor` when their runner is stale, and have their state files migrated forward automatically with a backup safety net.
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
@.planning/phases/05-scoped-ephemeral-mode-and-safety-profiles/05-02-SUMMARY.md
@internal/state/schema.go
@internal/state/migrations.go
@internal/cli/exit.go
@internal/cli/root.go
@internal/cli/up.go
@internal/cli/status.go
@internal/cli/doctor.go
@internal/ops/doctor.go
@internal/bootstrap/package.go
@cmd/runnerkit/main.go

<interfaces>
<!-- Existing contracts the plan must integrate with. Extracted from codebase 2026-05-02. -->

State schema (internal/state/schema.go):
```go
const SchemaVersion = "1"      // BUMPED to "2" by this plan

type State struct {
    SchemaVersion string            `json:"schema_version"`
    Repositories  []RepositoryState `json:"repositories"`
}

type RepositoryState struct {
    // ... 17 fields including Ephemeral EphemeralMetadata `json:"ephemeral,omitempty"`
}
```

Existing migration stub (internal/state/migrations.go) — REPLACED by this plan:
```go
func Migrate(state State) (State, error) {
    if state.SchemaVersion == "" { state.SchemaVersion = SchemaVersion }
    if state.SchemaVersion != SchemaVersion { return State{}, fmt.Errorf("unsupported runnerkit state schema_version %q", state.SchemaVersion) }
    if state.Repositories == nil { state.Repositories = []RepositoryState{} }
    return state, nil
}
```
The signature `func Migrate(state State) (State, error)` MUST be preserved — internal/state/store.go::Load calls it.

Existing exit codes (internal/cli/exit.go):
```go
const (
    ExitSuccess       = 0
    ExitUnexpected    = 1
    ExitInvalidInput  = 2
    ExitGitHubAuth    = 3
    ExitSafetyGate    = 4
    ExitStateIO       = 5
    ExitInputRequired = 6
    ExitCanceled      = 130
)
```
This plan ADDS `ExitStateSchemaTooNew = 7`.

Runner pin (internal/bootstrap/package.go):
```go
const RunnerVersion = "2.334.0"
```
This is the source of truth (NOT script.go — RESEARCH had a typo). `runnerkit upgrade-runner` and `runnerkit doctor`'s stale-runner finding read this constant.

Doctor finding model (internal/ops/doctor.go):
```go
type Finding struct {
    ID          string `json:"id"`
    Severity    string `json:"severity"`
    Source      string `json:"source"`
    Evidence    string `json:"evidence"`
    Remediation string `json:"remediation"`
}

const (
    SeverityPass    Severity = "pass"
    SeverityWarning Severity = "warning"
    SeverityError   Severity = "error"
)

// Existing finding IDs include: state_present, github_runner_found, github_runner_offline,
// service_active, service_failed, service_missing, label_drift, install_path_missing,
// work_dir_missing, disk_low, tools_missing, network_github_failed, time_unsynchronized,
// cleanup_pending. This plan ADDS `runner_version_stale`.
```

Bootstrap re-entry contract (internal/bootstrap/install.go):
```go
// Apply(ctx, opts) — persistent BYO + cloud bootstrap; idempotent (Phase 2 + Phase 4 contract).
// ApplyEphemeral(ctx, opts) — ephemeral BYO + cloud bootstrap; idempotent (Phase 5 contract).
// Both already overwrite the systemd unit + runner package on re-run; that IS the upgrade.
```

Phase 5 ephemeral lifecycle states (per Phase 5 SUMMARY — `upgrade-runner` MUST honor):
- waiting   — ephemeral runner registered, waiting for first job (re-Apply WOULD drop the registration; refuse without --force)
- busy      — ephemeral runner currently running a job (re-Apply WOULD kill the job; refuse without --force)
- completed — ephemeral runner one-shot terminated, auto-deregistered (no-op with clear message; next `runnerkit up --mode ephemeral` will use new pin)
- ttl_expired — same as completed (terminal)
- cleanup_pending — finalizer didn't preserve logs cleanly; doctor surfaces this

Cobra command tree (internal/cli/root.go ~line 116-124):
```go
root.AddCommand(newVersionCommand(deps, &jsonOutput, &noColor))
root.AddCommand(newUpCommand(deps, &jsonOutput, &noColor))
root.AddCommand(newStatusCommand(deps, &jsonOutput, &noColor))
root.AddCommand(newLogsCommand(deps, &jsonOutput, &noColor))
root.AddCommand(newDoctorCommand(deps, &jsonOutput, &noColor))
root.AddCommand(newRecoverCommand(deps, &jsonOutput, &noColor))
root.AddCommand(newDownCommand(deps, &jsonOutput, &noColor))
root.AddCommand(newDestroyCommand(deps, &jsonOutput, &noColor))
root.AddCommand(newStateCommand(deps, &jsonOutput, &noColor))
// This plan ADDS: newUpgradeCommand, newUpgradeRunnerCommand
```

State directory convention (internal/state/store.go::DefaultBaseDir):
- `$XDG_STATE_HOME/runnerkit/` if set
- `$HOME/.local/state/runnerkit/` otherwise
- The lazy update cache file `update-check.json` MUST live in the SAME directory as `state.json` (mode 0600).

Module path: `github.com/salar/runnerkit` (Go 1.22)
</interfaces>

<deep_work_rules>
This is a multi-file plan touching 17+ files. Critical patterns to honor:

1. **State migration backup MUST happen on raw bytes BEFORE Migrate parses them.** This means the backup happens in `internal/state/store.go::Load`, not inside `Migrate`. Per RESEARCH Pattern 4 the cleanest split is: `Load` reads bytes → if `schema_version < SchemaVersion`, `Load` writes the backup file → then `Load` calls `Migrate(state)` which performs the field-level migrations on the parsed struct. (RESEARCH Pattern 4 example shows backup inside `Migrate(raw, path)`; we adapt to the existing `Migrate(state State)` signature by moving the backup write into `Load`.) Tasks below specify the exact split.

2. **MaybePrint is silent on EVERY failure path.** Network error, JSON parse error, write error, missing dir — return without printing. The user must NEVER see a stack trace from the update check.

3. **upgrade-runner against waiting ephemeral MUST refuse without --force** (Phase 5 invariant; documented in RESEARCH Pattern 7).

4. **The `update_notice.go` integration uses `defer` AFTER the existing `runUp`/`runStatus`/`runDoctor` returns, NOT before.** Otherwise the notice prints before "Setup complete." breaking UX (RESEARCH §"Sequencing risks").
</deep_work_rules>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: State migration framework — bump SchemaVersion to "2", forward-only chain, side-by-side backup, refuse-newer-schema, ExitStateSchemaTooNew exit code</name>
  <files>internal/state/schema.go, internal/state/migrations.go, internal/state/migrations_test.go, internal/state/store.go, internal/cli/exit.go</files>
  <read_first>
    - internal/state/schema.go (line 10: `SchemaVersion = "1"` — to be bumped)
    - internal/state/migrations.go (the 16-line stub to be replaced)
    - internal/state/store.go (read entirely — backup must happen in Load, BEFORE Migrate; the existing atomic-write helper at ~line 146 is reused for the backup write)
    - internal/cli/exit.go (existing constants: ExitSuccess=0..ExitInputRequired=6, ExitCanceled=130; this task ADDS ExitStateSchemaTooNew=7)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (Pattern 4 for the chain shape; Pitfall 6 for refuse-newer-schema; Pitfall 4 for atomic backup; Pattern 4 explicitly handles RKD-STATE-NNN error codes — Plan 06-03 will replace the literal `RKD-STATE-NNN` strings via the errcodes package, but for this task we hardcode them. Plan 06-03 has the cross-file overlap noted in CONTEXT and will refactor.)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-09)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VALIDATION.md (lines 65-68 for the 4 required test names)
  </read_first>
  <behavior>
    - Test 1: `TestMigrate_V1ToV2_ForwardOnly` — input is a valid v1 state.json with `schema_version: "1"` and at least one populated repository; after Migrate (or Load+Migrate), the returned State has `SchemaVersion == "2"` and ALL fields are preserved exactly (use reflect.DeepEqual on the Repositories slice).
    - Test 2: `TestMigrate_WritesBackupBeforeMutation` — write a v1 state.json to `t.TempDir()`, call store.Load, assert that a sibling file matching the glob `state.json.backup-v1-*Z` exists, that its bytes equal the ORIGINAL v1 bytes byte-for-byte, AND that the new state.json contains `schema_version: "2"`. The backup file MUST be present whether or not the migration succeeds (write before mutate).
    - Test 3a: `TestMigrate_RefusesNewerSchema` — write a state.json with `schema_version: "99"` to t.TempDir(), call store.Load, assert error is non-nil and `errors.Is(err, state.ErrSchemaTooNew)` is true. Assert the original file bytes are UNCHANGED on disk after the failed Load (refuse-to-mutate). Assert no backup file is written (we don't back up just to refuse; backup is only on actual mutation).
    - Test 3b: `TestExitCodeStateSchemaTooNew` — in `internal/cli/exit_test.go` (or a new test file alongside exit.go), assert that `cli.ExitCode(&cli.ExitError{Code: cli.ExitStateSchemaTooNew, Err: state.ErrSchemaTooNew}) == 7`.
    - Test 4: `TestMigrate_Atomic` — inject a write-failing temp dir (e.g., remove write permission via `os.Chmod(dir, 0500)` after writing the original) and assert that when the migration write fails, the backup file is preserved AND the original state.json is preserved AND the returned error is non-nil. (The full atomic-write contract is already enforced by `internal/state/store.go::writeAtomic`; this test verifies the backup is preserved on failure.)
  </behavior>
  <action>
**Step 1: Bump `SchemaVersion`** in `internal/state/schema.go` line 10:

```go
const SchemaVersion = "2"
```

**Step 2: Add `ExitStateSchemaTooNew = 7`** in `internal/cli/exit.go` line 17 (after `ExitInputRequired = 6`):

```go
const (
    ExitSuccess           = 0
    ExitUnexpected        = 1
    ExitInvalidInput      = 2
    ExitGitHubAuth        = 3
    ExitSafetyGate        = 4
    ExitStateIO           = 5
    ExitInputRequired     = 6
    ExitStateSchemaTooNew = 7
    ExitCanceled          = 130
)
```

**Step 3: Replace `internal/state/migrations.go` body** with a forward-only chain. Keep the signature `func Migrate(state State) (State, error)` UNCHANGED so existing callers in `store.go::Load` continue to compile. New file:

```go
package state

import (
    "errors"
    "fmt"
)

// ErrSchemaTooNew is returned when state.json was written by a newer
// RunnerKit than the current binary knows about. Refuse-to-mutate per D-09.
// Maps to ExitStateSchemaTooNew (=7) in cli/exit.go.
var ErrSchemaTooNew = errors.New("runnerkit state schema_version is newer than this CLI knows; upgrade RunnerKit (run `runnerkit upgrade`) to read this state")

type migrationFn func(State) (State, error)

// forwardMigrations maps fromVersion → migration that produces (fromVersion+1).
// Add entries here ONLY in forward order. Never delete; never renumber.
var forwardMigrations = map[string]migrationFn{
    "1": migrateV1ToV2,
}

// Migrate runs forward-only migrations from state.SchemaVersion to SchemaVersion.
// Returns ErrSchemaTooNew (refuse-to-mutate) when state was written by a newer CLI.
// CALLERS' contract: store.Load is responsible for writing the side-by-side backup
// of the ORIGINAL raw bytes BEFORE invoking Migrate (so the backup persists even
// if migration logic itself fails). See store.Load for the backup write.
func Migrate(state State) (State, error) {
    if state.SchemaVersion == "" {
        state.SchemaVersion = SchemaVersion
    }
    if cmpVersion(state.SchemaVersion, SchemaVersion) > 0 {
        return State{}, ErrSchemaTooNew
    }
    if state.Repositories == nil {
        state.Repositories = []RepositoryState{}
    }
    for cmpVersion(state.SchemaVersion, SchemaVersion) < 0 {
        from := state.SchemaVersion
        fn, ok := forwardMigrations[from]
        if !ok {
            return State{}, fmt.Errorf("no migration from schema_version %q", from)
        }
        next, err := fn(state)
        if err != nil {
            return State{}, fmt.Errorf("migration from schema_version %q failed: %w", from, err)
        }
        if cmpVersion(next.SchemaVersion, from) <= 0 {
            return State{}, fmt.Errorf("migration from schema_version %q did not advance version (got %q)", from, next.SchemaVersion)
        }
        state = next
    }
    return state, nil
}

// migrateV1ToV2 is an identity migration: no field semantics changed in v2,
// but the framework + side-by-side backup is what REL-05 requires. Future v2→v3
// migrations attach to forwardMigrations.
func migrateV1ToV2(s State) (State, error) {
    s.SchemaVersion = "2"
    return s, nil
}

// cmpVersion compares two SchemaVersion strings ("1", "2", ...) numerically.
// SchemaVersion is intentionally a small monotonic integer string; we don't
// want full semver here (state schema is not a public API surface).
// Returns -1 if a<b, 0 if equal, +1 if a>b. Empty string sorts as 0.
func cmpVersion(a, b string) int {
    ai := parseSchema(a)
    bi := parseSchema(b)
    switch {
    case ai < bi:
        return -1
    case ai > bi:
        return +1
    default:
        return 0
    }
}

func parseSchema(v string) int {
    if v == "" {
        return 0
    }
    var n int
    if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
        return -1
    }
    return n
}
```

**Step 4: Update `internal/state/store.go::Load`** to write the side-by-side backup of the raw bytes BEFORE calling `Migrate`. Locate the existing `Load` function (it currently reads bytes, json.Unmarshal, then calls `Migrate(state)`). Modify so that:

```go
// In store.go::Load (sketch — adapt to existing Load shape):
raw, err := os.ReadFile(s.path)
if err != nil { /* existing error handling */ }
var state State
if err := json.Unmarshal(raw, &state); err != nil { /* existing */ }

// NEW: probe the on-disk schema_version BEFORE migration; if it's older than
// SchemaVersion, write a side-by-side backup of the raw bytes BEFORE migration.
// If it's newer, skip the backup (refuse-to-mutate path doesn't need a backup).
if state.SchemaVersion != "" && cmpVersion(state.SchemaVersion, SchemaVersion) < 0 {
    backupPath := s.path + ".backup-v" + state.SchemaVersion + "-" + time.Now().UTC().Format("20060102T150405Z")
    if err := os.WriteFile(backupPath, raw, 0600); err != nil {
        return State{}, fmt.Errorf("write state backup: %w", err)
    }
}

migrated, err := Migrate(state)
if err != nil { return State{}, err }

// If schema was upgraded, persist the migrated state to disk via the existing
// atomic-write helper. (Caller already does this for general mutations; we
// must do it here so the backup file → migrated file transition is committed.)
if cmpVersion(migrated.SchemaVersion, state.SchemaVersion) > 0 {
    if err := s.Save(migrated); err != nil {
        return State{}, fmt.Errorf("write migrated state: %w", err)
    }
}
return migrated, nil
```

If `Load` already does some of this (e.g., writes back), preserve the existing semantics; only ADD the backup write step if missing. The key invariant is: **the backup file exists on disk before the new state.json is written**, AND **the backup file equals the original raw bytes byte-for-byte**.

**Step 5: Write `internal/state/migrations_test.go`** implementing the 4 tests in `<behavior>` above. Use Go testing conventions consistent with existing `internal/state/*_test.go` files (run `ls internal/state/*_test.go` to find them; reuse helpers).

The 4 test names MUST be exactly: `TestMigrate_V1ToV2_ForwardOnly`, `TestMigrate_WritesBackupBeforeMutation`, `TestMigrate_RefusesNewerSchema`, `TestMigrate_Atomic` (per `06-VALIDATION.md` lines 65-68).

Add a 5th test in `internal/cli/exit_test.go` (or `internal/cli/state_exit_test.go`): `TestExitCodeStateSchemaTooNew` per `06-VALIDATION.md` line 67's compound command.
  </action>
  <verify>
    <automated>go test ./internal/state -run 'TestMigrate_V1ToV2_ForwardOnly|TestMigrate_WritesBackupBeforeMutation|TestMigrate_RefusesNewerSchema|TestMigrate_Atomic' -count=1 && go test ./internal/cli -run TestExitCodeStateSchemaTooNew -count=1 && go vet ./internal/state/... ./internal/cli/...</automated>
  </verify>
  <acceptance_criteria>
    - `internal/state/schema.go` line containing `SchemaVersion =` reads exactly `const SchemaVersion = "2"`.
    - `internal/cli/exit.go` declares `ExitStateSchemaTooNew = 7` between `ExitInputRequired` and `ExitCanceled`.
    - `internal/state/migrations.go` exports `ErrSchemaTooNew error` (sentinel) and a `forwardMigrations map[string]migrationFn` containing key `"1"` mapped to `migrateV1ToV2`.
    - `Migrate` signature is unchanged: `func Migrate(state State) (State, error)`.
    - Loading a v1 state.json writes a sibling file matching `state.json.backup-v1-*Z` (RFC3339-derived timestamp suffix; the `Z` is from `Format("20060102T150405Z")`) AND the backup contains the ORIGINAL bytes (byte-equal).
    - Loading a v99 state.json returns `ErrSchemaTooNew`, leaves the file unchanged on disk, and writes no backup file.
    - `cli.ExitCode(cli.NewExitError(cli.ExitStateSchemaTooNew, state.ErrSchemaTooNew)) == 7`.
    - All 5 tests in `<behavior>` pass: `TestMigrate_V1ToV2_ForwardOnly`, `TestMigrate_WritesBackupBeforeMutation`, `TestMigrate_RefusesNewerSchema`, `TestMigrate_Atomic`, `TestExitCodeStateSchemaTooNew`.
    - `go vet ./internal/state/... ./internal/cli/...` passes.
    - All validation matrix rows for D-09 (lines 65-68 of `06-VALIDATION.md`) are green.
  </acceptance_criteria>
  <done>SchemaVersion bumped to "2"; forward-only Migrate chain + ErrSchemaTooNew + side-by-side backup in Load; ExitStateSchemaTooNew=7; all 5 tests green; pre-existing tests still pass.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: internal/update package (24h cache + ETag + JSON-silent + CI-skip + no-net-silent) and integration into up/status/doctor</name>
  <files>internal/update/check.go, internal/update/check_test.go, internal/update/version.go, internal/cli/update_notice.go, internal/cli/up.go, internal/cli/status.go, internal/cli/doctor.go, go.mod, go.sum</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (Pattern 5 — full code sketch; Pitfall 4 — atomic cache writes; "Code Examples" §"GitHub Releases conditional GET")
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-06)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VALIDATION.md (lines 54-57: 4 required test names)
    - internal/cli/up.go (find the renderer flush point — MaybePrint MUST be deferred AFTER the existing renderer output)
    - internal/cli/status.go (same)
    - internal/cli/doctor.go (same)
    - internal/cli/root.go (Dependencies struct: Out, Err, Clock; the Clock is used as the time source — pass through to MaybePrint)
    - internal/state/store.go::DefaultBaseDir (cache file lives in the same dir as state.json)
    - go.mod (current Go version 1.22; will add `github.com/hashicorp/go-version v1.9.0`)
  </read_first>
  <behavior>
    - Test 1: `TestMaybePrint_JSONMode_Silent` — call `MaybePrint(jsonOutput=true, ...)` against an `httptest.Server` that returns 200 + a newer version. Assert errOut buffer is empty AND the HTTP server was NOT called (zero requests recorded).
    - Test 2: `TestMaybePrint_HonorsCache` — pre-write a `update-check.json` cache file with `LastCheck: now-1h` and `Latest: "v9.9.9"`. Call `MaybePrint(jsonOutput=false, currentVersion="v0.1.0", ...)`. Assert the function did NOT make an HTTP request (httptest server gets zero hits) AND DID print the cached newer-than-current notice line on errOut. Then update cache to `LastCheck: now-25h` and assert the next call DOES hit the HTTP server.
    - Test 3: `TestMaybePrint_NetworkError_Silent` — call `MaybePrint` with an HTTP client whose Transport returns `errors.New("net unreachable")`. Assert errOut is empty AND no error returns (the function MUST swallow and return cleanly).
    - Test 4: `TestMaybePrint_ConditionalGET` — first call: server returns 200 + JSON + `ETag: "abc123"`. Assert cache file's `etag` field == "abc123". Second call after cache TTL: assert request was made with header `If-None-Match: "abc123"`. Server returns 304. Assert cache `LastCheck` was bumped, `Latest` unchanged, NO notice was printed (304 means same version).
    - (Bonus, defensive) `TestMaybePrint_CISkip` — set `t.Setenv("CI", "1")`, call `MaybePrint(jsonOutput=false, ...)`. Assert errOut is empty and HTTP server got 0 hits (RESEARCH §"gh CLI rule").
    - (Bonus) `TestMaybePrint_NoUpdateNotifier` — set `t.Setenv("RUNNERKIT_NO_UPDATE_NOTIFIER", "1")`. Assert silent.
  </behavior>
  <action>
**Step 1: Add `hashicorp/go-version` to go.mod:**

```bash
go get github.com/hashicorp/go-version@v1.9.0
```

This is the same library `gh` CLI uses (RESEARCH §"Standard Stack").

**Step 2: Create `internal/update/version.go`** — thin wrapper over hashicorp/go-version:

```go
package update

import gv "github.com/hashicorp/go-version"

// IsNewer returns true if `latest` is strictly greater than `current`.
// Both arguments accept "v"-prefixed or unprefixed semver strings.
// Pre-releases follow hashicorp/go-version's standard precedence rules.
// On parse failure, returns false (silent).
func IsNewer(current, latest string) bool {
    cur, err := gv.NewVersion(current)
    if err != nil {
        return false
    }
    lat, err := gv.NewVersion(latest)
    if err != nil {
        return false
    }
    return lat.GreaterThan(cur)
}
```

**Step 3: Create `internal/update/check.go`** — implements `MaybePrint` per RESEARCH Pattern 5. Full implementation (annotate copy from RESEARCH; do not paraphrase the contract):

```go
package update

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "time"
)

// CheckedRelease is the cached payload from /releases/latest.
type CheckedRelease struct {
    Latest      string    `json:"latest"`
    URL         string    `json:"url"`
    PublishedAt time.Time `json:"published_at"`
    ETag        string    `json:"etag"`
    LastCheck   time.Time `json:"last_check"`
}

const (
    cacheFileName = "update-check.json"
    cacheTTL      = 24 * time.Hour
    apiURL        = "https://api.github.com/repos/salar/runnerkit/releases/latest"
    httpTimeout   = 5 * time.Second
)

// Deps lets tests inject HTTP, time, and cache dir.
type Deps struct {
    HTTPClient *http.Client
    Now        func() time.Time
    StateDir   string
    APIURL     string // defaults to apiURL when empty (overridable for tests)
}

// MaybePrint emits a single non-blocking notice line to errOut if a newer
// release exists. Silent on any error path. Honors:
//   - jsonOutput == true → silent (Phase 1 contract)
//   - $CI set → silent (gh CLI convention)
//   - $RUNNERKIT_NO_UPDATE_NOTIFIER set → silent (per-user opt-out)
//   - last check < 24h ago → use cached value, no HTTP
//   - network error → silent
//   - response is same tag as current → silent
func MaybePrint(jsonOutput bool, currentVersion string, deps Deps, errOut io.Writer) {
    if jsonOutput {
        return
    }
    if os.Getenv("CI") != "" {
        return
    }
    if os.Getenv("RUNNERKIT_NO_UPDATE_NOTIFIER") != "" {
        return
    }
    if deps.HTTPClient == nil {
        deps.HTTPClient = &http.Client{Timeout: httpTimeout}
    }
    if deps.Now == nil {
        deps.Now = time.Now
    }
    apiTarget := deps.APIURL
    if apiTarget == "" {
        apiTarget = apiURL
    }

    cachePath := ""
    if deps.StateDir != "" {
        cachePath = filepath.Join(deps.StateDir, cacheFileName)
    }

    cached := loadCache(cachePath)
    now := deps.Now()
    var latest CheckedRelease

    // Use cache when fresh.
    if !cached.LastCheck.IsZero() && now.Sub(cached.LastCheck) < cacheTTL {
        latest = cached
    } else {
        // Fetch with conditional GET.
        ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
        defer cancel()
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiTarget, nil)
        if err != nil {
            return
        }
        req.Header.Set("Accept", "application/vnd.github+json")
        if cached.ETag != "" {
            req.Header.Set("If-None-Match", cached.ETag)
        }
        resp, err := deps.HTTPClient.Do(req)
        if err != nil {
            return // silent on no-net per D-06
        }
        defer resp.Body.Close()
        switch resp.StatusCode {
        case http.StatusNotModified:
            // 304 — payload unchanged; refresh LastCheck.
            cached.LastCheck = now
            saveCache(cachePath, cached)
            latest = cached
        case http.StatusOK:
            var payload struct {
                TagName     string    `json:"tag_name"`
                HTMLURL     string    `json:"html_url"`
                PublishedAt time.Time `json:"published_at"`
            }
            if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
                return
            }
            latest = CheckedRelease{
                Latest:      payload.TagName,
                URL:         payload.HTMLURL,
                PublishedAt: payload.PublishedAt,
                ETag:        resp.Header.Get("ETag"),
                LastCheck:   now,
            }
            saveCache(cachePath, latest)
        default:
            return // silent on non-200/304
        }
    }

    if latest.Latest == "" {
        return
    }
    if !IsNewer(currentVersion, latest.Latest) {
        return
    }
    fmt.Fprintf(errOut, "runnerkit %s available (you have %s). Run `runnerkit upgrade` for instructions.\n",
        latest.Latest, currentVersion)
}

func loadCache(path string) CheckedRelease {
    if path == "" {
        return CheckedRelease{}
    }
    raw, err := os.ReadFile(path)
    if err != nil {
        return CheckedRelease{}
    }
    var c CheckedRelease
    if err := json.Unmarshal(raw, &c); err != nil {
        return CheckedRelease{}
    }
    return c
}

func saveCache(path string, c CheckedRelease) {
    if path == "" {
        return
    }
    if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
        return
    }
    raw, err := json.Marshal(c)
    if err != nil {
        return
    }
    // Atomic-ish write: tmp + rename to avoid Pitfall 4 (cache file race).
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, raw, 0600); err != nil {
        return
    }
    _ = os.Rename(tmp, path)
}
```

**Step 4: Create `internal/update/check_test.go`** with tests from `<behavior>` above. Use `httptest.NewServer` to fake the GitHub API. Use `t.TempDir()` for `StateDir`. Use a synthetic `Now` function for cache TTL tests.

The 4 required test names per `06-VALIDATION.md`: `TestMaybePrint_JSONMode_Silent`, `TestMaybePrint_HonorsCache`, `TestMaybePrint_NetworkError_Silent`, `TestMaybePrint_ConditionalGET`. Add the two bonus tests (`TestMaybePrint_CISkip`, `TestMaybePrint_NoUpdateNotifier`) for completeness.

**Step 5: Create `internal/cli/update_notice.go`** — small adapter that resolves the state dir and version, then delegates to `update.MaybePrint`:

```go
package cli

import (
    "github.com/salar/runnerkit/internal/state"
    "github.com/salar/runnerkit/internal/update"
)

// maybeShowUpdateNotice is called via deferred invocation from up/status/doctor.
// It is the integration seam for D-06's lazy update check.
// MUST be silent on every failure path; never block; never error.
func maybeShowUpdateNotice(deps Dependencies, jsonOutput bool) {
    if jsonOutput {
        return
    }
    update.MaybePrint(jsonOutput, deps.Version, update.Deps{
        StateDir: state.DefaultBaseDir(),
        Now:      deps.Clock,
    }, deps.Err)
}
```

**Step 6: Wire the deferred call** at the END of `runUp`, `runStatus`, `runDoctor`. The integration MUST come AFTER the existing renderer output (per RESEARCH §"Sequencing risks" — otherwise notice prints before "Setup complete.").

In `internal/cli/up.go::runUp`:
```go
func runUp(deps Dependencies, jsonOutput bool, noColor bool, opts *upOptions) (returnedErr error) {
    defer maybeShowUpdateNotice(deps, jsonOutput)
    // ... existing function body unchanged
}
```

In `internal/cli/status.go::runStatus` and `internal/cli/doctor.go::runDoctor`: same single `defer maybeShowUpdateNotice(deps, jsonOutput)` line at the start.

The deferred call uses `defer` (not `cmd.PostRunE`) because `defer` fires AFTER the function's renderer flushes its final output to deps.Out, but the notice goes to deps.Err — they don't interleave on a TTY in practice for human users.
  </action>
  <verify>
    <automated>go test ./internal/update -run 'TestMaybePrint_JSONMode_Silent|TestMaybePrint_HonorsCache|TestMaybePrint_NetworkError_Silent|TestMaybePrint_ConditionalGET|TestMaybePrint_CISkip|TestMaybePrint_NoUpdateNotifier' -count=1 && go vet ./internal/update/... ./internal/cli/... && grep -q "defer maybeShowUpdateNotice" internal/cli/up.go && grep -q "defer maybeShowUpdateNotice" internal/cli/status.go && grep -q "defer maybeShowUpdateNotice" internal/cli/doctor.go && grep -q "github.com/hashicorp/go-version" go.sum</automated>
  </verify>
  <acceptance_criteria>
    - `internal/update/check.go` exports `MaybePrint(jsonOutput bool, currentVersion string, deps Deps, errOut io.Writer)`.
    - `internal/update/check.go` reads cache from `<StateDir>/update-check.json` with mode 0600, atomic write via tmp+rename.
    - `internal/update/check.go` honors all 6 silent-paths: `jsonOutput`, `$CI`, `$RUNNERKIT_NO_UPDATE_NOTIFIER`, network error, 304 response (no-print, just bump LastCheck), same-version response.
    - `internal/update/version.go` exports `IsNewer(current, latest string) bool` using `github.com/hashicorp/go-version`.
    - All 4 required tests pass: `TestMaybePrint_JSONMode_Silent`, `TestMaybePrint_HonorsCache`, `TestMaybePrint_NetworkError_Silent`, `TestMaybePrint_ConditionalGET`.
    - The two bonus tests pass: `TestMaybePrint_CISkip`, `TestMaybePrint_NoUpdateNotifier`.
    - `internal/cli/update_notice.go::maybeShowUpdateNotice` exists.
    - `runUp`, `runStatus`, `runDoctor` each contain a `defer maybeShowUpdateNotice(deps, jsonOutput)` line. (grep confirms.)
    - `go.sum` contains `github.com/hashicorp/go-version v1.9.0`.
    - `go vet ./internal/update/... ./internal/cli/...` passes.
    - All validation matrix rows for D-06 (lines 54-57) are green.
  </acceptance_criteria>
  <done>internal/update package implements lazy 24h-cached ETag-based update check with all silent-paths honored; integrated via defer-MaybeShowUpdateNotice into up/status/doctor; 6 tests green; hashicorp/go-version v1.9.0 dependency added.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: `runnerkit upgrade` (channel-detect, print-only) + `runnerkit upgrade-runner` (re-Apply with bundled pin) + doctor stale-runner finding</name>
  <files>internal/cli/upgrade.go, internal/cli/upgrade_test.go, internal/cli/upgrade_runner.go, internal/cli/upgrade_runner_test.go, internal/cli/root.go, internal/ops/doctor.go, internal/ops/doctor_test.go, internal/bootstrap/package.go</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (Pattern 6 — channel detect; Pattern 7 — upgrade-runner; Pitfall 5 — runner pin staleness drift; D-08)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-07, D-08)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VALIDATION.md (lines 58-64: 7 required test names)
    - .planning/phases/05-scoped-ephemeral-mode-and-safety-profiles/05-02-SUMMARY.md (ephemeral lifecycle states; FinalizerStatus values; refuse-without-force on waiting/busy)
    - internal/cli/root.go (line 116-124: command registration pattern; ALSO note: 06-03 plan ALSO touches doctor.go for RKD code wiring — coordinate via finding ID name only, not URL emission)
    - internal/cli/up.go (newUpCommand pattern as template for newUpgradeCommand and newUpgradeRunnerCommand)
    - internal/bootstrap/install.go (Apply and ApplyEphemeral signatures)
    - internal/bootstrap/package.go (RunnerVersion constant — source of truth for the pin)
    - internal/ops/doctor.go (Finding struct, severity constants, existing finding IDs; the `add` function pattern at line ~39)
    - internal/state/schema.go (RunnerTemplateVersion field — what the doctor compares against)
  </read_first>
  <behavior>
    - Test 1 (upgrade.go): `TestUpgrade_DetectsHomebrew` — call `detectChannel("/opt/homebrew/Cellar/runnerkit/1.0.0/bin/runnerkit")` → returns `"homebrew"`. Also `/usr/local/Cellar/runnerkit/1.0.0/bin/runnerkit` → `"homebrew"`. Also a Caskroom path containing `/Caskroom/runnerkit/` → `"homebrew"`.
    - Test 2 (upgrade.go): `TestUpgrade_DetectsBinaryChannel` — call `detectChannel("/usr/local/bin/runnerkit")` (with no Cellar/Caskroom in resolved path) → returns `"binary"`. Call `detectChannel("/home/user/.local/bin/runnerkit")` → `"binary"`.
    - Test 3 (upgrade.go): `TestUpgrade_JSONContract` — invoke the cobra command with `--json --current v0.5.0 --latest v1.0.0 --simulated-channel homebrew` (use a test seam to avoid filesystem-dependent channel detection); assert stdout JSON has keys `ok` (true), `channel` ("homebrew"), `commands` (array containing `"brew upgrade runnerkit"`), `current` ("v0.5.0"), `latest` ("v1.0.0"). Assert NO subprocess was spawned (the test doubles for exec.Command record zero invocations).
    - Test 4 (upgrade_runner.go): `TestUpgradeRunner_Persistent_ReAppliesWithNewPin` — fixture: persistent BYO RepositoryState with `RunnerTemplateVersion: "2.330.0"` (stale); inject a fake `bootstrap.Apply` adapter that records its call with the bundled pin. Assert `bootstrap.RunnerVersion` ("2.334.0") is the pin passed to Apply, AND state.RunnerTemplateVersion is updated to that pin only after Apply returns nil.
    - Test 5 (upgrade_runner.go): `TestUpgradeRunner_Ephemeral_TerminalNoOp` — fixture: ephemeral RepositoryState with `Ephemeral.FinalizerStatus: "completed"` (terminal). Run upgrade-runner. Assert NO call to ApplyEphemeral was made AND a clear message was printed (e.g., "Ephemeral runner is one-shot; the next `runnerkit up --mode ephemeral` will use the new pin"). Exit code 0.
    - Test 6 (upgrade_runner.go): `TestUpgradeRunner_Ephemeral_WaitingRefusesWithoutForce` — fixture: ephemeral with `Ephemeral.FinalizerStatus: "waiting"` (or empty, meaning registered & waiting). Run upgrade-runner WITHOUT `--force`. Assert error returned, exit code = ExitInvalidInput (or a specific UpgradeRefused exit code if introduced — see implementation), and NO call to ApplyEphemeral. Then run with `--force`; assert ApplyEphemeral IS called.
    - Test 7 (doctor.go): `TestDoctor_StaleRunnerVersion` — fixture: persistent RepositoryState with `RunnerTemplateVersion: "2.330.0"` while `bootstrap.RunnerVersion = "2.334.0"`. Run BuildDoctorReport. Assert exactly one finding has `ID: "runner_version_stale"`, `Severity: "warning"`, evidence containing both "2.330.0" and "2.334.0", and remediation referencing `runnerkit upgrade-runner`. Then test the same scenario with `RunnerTemplateVersion: bootstrap.RunnerVersion` and assert NO `runner_version_stale` finding is emitted.
  </behavior>
  <action>
**Step 1: Confirm `internal/bootstrap/package.go::RunnerVersion`** exists (it does — `const RunnerVersion = "2.334.0"`). No change needed except: ensure the constant is exported (capital R — already is). Add a comment block above it stating it is the public pin consumed by `internal/cli/upgrade_runner.go` and `internal/ops/doctor.go::BuildDoctorReport`. Do NOT change the value (Phase 6 keeps 2.334.0 per CONTEXT D-08).

**Step 2: Create `internal/cli/upgrade.go`** — channel-detect, print-only:

```go
package cli

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/salar/runnerkit/internal/state"
    "github.com/salar/runnerkit/internal/update"
    "github.com/spf13/cobra"
)

func newUpgradeCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
    cmd := &cobra.Command{Use: "upgrade", Short: "Print upgrade instructions for this RunnerKit install (does not self-replace the binary)."}
    cmd.RunE = func(_ *cobra.Command, _ []string) error {
        return runUpgrade(deps, *jsonOutput, *noColor)
    }
    return cmd
}

type upgradeReport struct {
    OK       bool     `json:"ok"`
    Channel  string   `json:"channel"`
    Commands []string `json:"commands"`
    Current  string   `json:"current"`
    Latest   string   `json:"latest"`
    Notes    string   `json:"notes,omitempty"`
}

func runUpgrade(deps Dependencies, jsonOutput bool, noColor bool) error {
    execPath, err := os.Executable()
    if err != nil {
        return NewExitError(ExitUnexpected, fmt.Errorf("resolve binary path: %w", err))
    }
    channel := detectChannel(execPath)

    // Best-effort latest lookup; silent on failure (matches D-06 ethos).
    latest := lookupLatestSilent(deps)

    report := upgradeReport{
        OK:      true,
        Channel: channel,
        Current: deps.Version,
        Latest:  latest,
    }
    switch channel {
    case "homebrew":
        report.Commands = []string{"brew upgrade runnerkit"}
    case "binary":
        report.Commands = []string{
            "Download the latest release: https://github.com/salar/runnerkit/releases/latest",
            "Then verify the cosign signature before installing — see docs/troubleshooting/README.md.",
        }
    default:
        report.Channel = "unknown"
        report.Commands = []string{
            "RunnerKit cannot tell how this binary was installed. Run `which runnerkit` and follow the channel-specific instructions in docs/upgrade.md.",
        }
        report.Notes = "Set RUNNERKIT_DOCS_BASE if your docs are hosted somewhere other than github.com/salar/runnerkit."
    }

    if jsonOutput {
        return json.NewEncoder(deps.Out).Encode(report)
    }
    fmt.Fprintf(deps.Out, "RunnerKit %s detected install channel: %s\n", deps.Version, channel)
    if latest != "" {
        fmt.Fprintf(deps.Out, "Latest released version: %s\n", latest)
    }
    fmt.Fprintln(deps.Out, "Upgrade instructions:")
    for _, c := range report.Commands {
        fmt.Fprintln(deps.Out, "  "+c)
    }
    return nil
}

// detectChannel inspects a binary path and returns "homebrew", "binary", or "unknown".
// Homebrew install paths on macOS contain `/Cellar/runnerkit/<version>/bin/runnerkit`
// or `/Caskroom/runnerkit/<version>/...` (RESEARCH Pattern 6).
func detectChannel(execPath string) string {
    abs := execPath
    if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
        abs = resolved
    }
    if strings.Contains(abs, "/Cellar/runnerkit/") || strings.Contains(abs, "/Caskroom/runnerkit/") {
        return "homebrew"
    }
    if strings.Contains(abs, "/runnerkit") {
        return "binary"
    }
    return "unknown"
}

// lookupLatestSilent reads the cached update-check.json (populated by the
// lazy update notice in update_notice.go) and returns the latest tag if known,
// otherwise empty string. Never fetches over the network — we want
// `runnerkit upgrade` to be instantaneous and deterministic.
func lookupLatestSilent(deps Dependencies) string {
    path := filepath.Join(state.DefaultBaseDir(), "update-check.json")
    raw, err := os.ReadFile(path)
    if err != nil {
        return ""
    }
    var c update.CheckedRelease
    if err := json.Unmarshal(raw, &c); err != nil {
        return ""
    }
    if c.Latest == "" {
        return ""
    }
    if !update.IsNewer(deps.Version, c.Latest) {
        // Latest is not newer than current — no upgrade prompt needed.
        // Still report it so the user sees the version line.
    }
    return c.Latest
}

// for tests — make detectChannel pluggable when needed
var _ = errors.New // keep imports stable across edits
```

**Step 3: Create `internal/cli/upgrade_test.go`** with `TestUpgrade_DetectsHomebrew`, `TestUpgrade_DetectsBinaryChannel`, `TestUpgrade_JSONContract`. The JSON contract test invokes the command with the `--json` flag and a fixture binary path; use `cobra.Command.SetArgs` and capture `deps.Out` to a `bytes.Buffer`.

**Step 4: Create `internal/cli/upgrade_runner.go`** — re-apply bootstrap with bundled pin:

```go
package cli

import (
    "context"
    "fmt"

    "github.com/salar/runnerkit/internal/bootstrap"
    "github.com/salar/runnerkit/internal/state"
    "github.com/spf13/cobra"
)

type upgradeRunnerOptions struct {
    repo  string
    force bool
    yes   bool
}

func newUpgradeRunnerCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
    opts := &upgradeRunnerOptions{}
    cmd := &cobra.Command{Use: "upgrade-runner", Short: "Re-apply runner bootstrap with the bundled runner pin (rolls forward without re-running setup)."}
    cmd.Flags().StringVar(&opts.repo, "repo", "", "owner/name (defaults to current dir's git remote)")
    cmd.Flags().BoolVar(&opts.force, "force", false, "force upgrade even when an ephemeral runner is currently waiting/busy (DROPS the queued registration)")
    cmd.Flags().BoolVar(&opts.yes, "yes", false, "skip confirmation prompt")
    cmd.RunE = func(_ *cobra.Command, _ []string) error {
        return runUpgradeRunner(deps, *jsonOutput, *noColor, opts)
    }
    return cmd
}

func runUpgradeRunner(deps Dependencies, jsonOutput bool, noColor bool, opts *upgradeRunnerOptions) error {
    ctx := context.Background()
    renderer := newRenderer(deps, jsonOutput, noColor)
    _ = renderer

    // Resolve repo, load state — reuse the same helpers other CLI commands use.
    // (Adapt to the existing helper names in internal/cli; e.g., resolveRepoState.)
    repoState, store, err := resolveRepoStateForCommand(deps, opts.repo) // implement using existing pattern; see internal/cli/status.go
    if err != nil {
        return err
    }

    currentPin := repoState.RunnerTemplateVersion
    bundled := bootstrap.RunnerVersion

    // Plan-before-mutation (Phase 2/4 contract).
    fmt.Fprintf(deps.Out, "Upgrade runner pin: %s → %s\n", currentPin, bundled)
    fmt.Fprintf(deps.Out, "Target host: %s\n", repoState.Machine.HostRef)

    // Ephemeral lifecycle gate (D-08, RESEARCH Pattern 7, Phase 5 invariant):
    if repoState.Ephemeral.Enabled {
        switch repoState.Ephemeral.FinalizerStatus {
        case "completed", "ttl_expired":
            fmt.Fprintln(deps.Out, "Ephemeral runner is one-shot and already terminated. The next `runnerkit up --mode ephemeral` will use the bundled pin "+bundled+".")
            return nil
        case "waiting", "busy", "":
            if !opts.force {
                return NewExitError(ExitInvalidInput, fmt.Errorf("ephemeral runner is currently %q; refuse to upgrade-runner without --force (would drop the queued runner registration). Re-run with --force to proceed.", repoState.Ephemeral.FinalizerStatus))
            }
            // Fall through to ApplyEphemeral below.
        }
    }

    if !opts.yes {
        // Use the existing confirmation prompt helper from internal/cli.
        if err := confirmYes(deps, "Proceed with upgrade-runner?"); err != nil {
            return err
        }
    }

    // Re-entry into Apply / ApplyEphemeral — both already idempotent.
    bopts := buildBootstrapOptionsForRepo(repoState, bundled) // adapt to existing builder
    if repoState.Ephemeral.Enabled {
        if err := bootstrap.ApplyEphemeral(ctx, bopts); err != nil {
            return NewExitError(ExitUnexpected, fmt.Errorf("apply-ephemeral: %w", err))
        }
    } else {
        if err := bootstrap.Apply(ctx, bopts); err != nil {
            return NewExitError(ExitUnexpected, fmt.Errorf("apply: %w", err))
        }
    }

    // Persist the new pin to state — only on success.
    repoState.RunnerTemplateVersion = bundled
    if err := store.UpdateRepository(repoState); err != nil {
        return NewExitError(ExitStateIO, fmt.Errorf("update state: %w", err))
    }
    fmt.Fprintf(deps.Out, "Runner pin updated to %s.\n", bundled)
    return nil
}
```

NOTE: `resolveRepoStateForCommand`, `buildBootstrapOptionsForRepo`, and `confirmYes` are placeholders for whichever existing helpers the codebase uses (e.g., grep `internal/cli/up.go` for the "load state for repo" pattern). Adapt names to existing helpers so you don't introduce duplicates. The KEY contract is: `bootstrap.Apply(ctx, bopts)` or `bootstrap.ApplyEphemeral(ctx, bopts)` is called, where `bopts` reflects the saved state's MachineRef, Runner, and the bundled pin.

**Step 5: Create `internal/cli/upgrade_runner_test.go`** with `TestUpgradeRunner_Persistent_ReAppliesWithNewPin`, `TestUpgradeRunner_Ephemeral_TerminalNoOp`, `TestUpgradeRunner_Ephemeral_WaitingRefusesWithoutForce`. Use a fake `bootstrap.Apply` injected via test seam (the existing pattern in `internal/cli/up_test.go` uses a similar seam — copy that pattern). For `TestUpgradeRunner_Ephemeral_WaitingRefusesWithoutForce`, also assert the `--force` path DOES call ApplyEphemeral.

**Step 6: Register both new commands** in `internal/cli/root.go` after line 124 (the `newStateCommand` registration):

```go
root.AddCommand(newUpgradeCommand(deps, &jsonOutput, &noColor))
root.AddCommand(newUpgradeRunnerCommand(deps, &jsonOutput, &noColor))
```

**Step 7: Add `runner_version_stale` finding in `internal/ops/doctor.go::BuildDoctorReport`.** Locate the `add` helper near line 39. Add this branch (place it logically near the `service_active`/`service_failed` block, or as a new section after the existing systemd checks):

```go
import "github.com/salar/runnerkit/internal/bootstrap"

// In BuildDoctorReport, after the existing systemd findings and BEFORE the
// cleanup_pending check:
if observedPin := repoState.RunnerTemplateVersion; observedPin != "" && observedPin != bootstrap.RunnerVersion {
    add("runner_version_stale", SeverityWarning, "bootstrap",
        fmt.Sprintf("installed runner version %s is older than bundled pin %s", observedPin, bootstrap.RunnerVersion),
        "runnerkit upgrade-runner --repo "+repo)
}
```

CONFLICT NOTE: Plan 06-03 also touches `internal/ops/doctor.go` to align finding IDs with RKD codes. The two plans conflict ONLY on the `add(...)` block above. Use the literal finding ID `runner_version_stale` (snake_case to match the existing convention). Plan 06-03 will MAP this finding ID to `RKD-BOOT-002` via `internal/errcodes/`; Plan 06-03 does NOT rename the finding ID. This means there's NO line-level conflict between the two plans — both add NEW lines in the same function but reference different concerns.

**Step 8: Add test `TestDoctor_StaleRunnerVersion`** in `internal/ops/doctor_test.go` (or `internal/ops/doctor_runner_pin_test.go` if doctor_test.go is large). Cover both scenarios from `<behavior>` Test 7.

**Step 9: Create `docs/upgrade.md`** — user-facing upgrade guide:

```markdown
# Upgrading RunnerKit

This guide covers three independent upgrade flows.

## 1. Upgrade the RunnerKit CLI

When `runnerkit up`, `runnerkit status`, or `runnerkit doctor` prints
`runnerkit X.Y.Z available`, run:

```
runnerkit upgrade
```

This prints the right command for your install channel:

| Install method | Upgrade command |
|---|---|
| Homebrew tap (`brew install salar/runnerkit/runnerkit`) | `brew upgrade runnerkit` |
| GitHub Releases binary | Download the latest release; verify cosign signature; replace the binary on PATH (see [README install section](../README.md#install)). |

`runnerkit upgrade` does NOT replace its own binary (per RunnerKit decision
D-07: avoiding self-replace removes a class of partial-failure bugs).

You can suppress the lazy update notice by setting
`RUNNERKIT_NO_UPDATE_NOTIFIER=1` in your shell environment.

## 2. Upgrade the bundled GitHub Actions runner pin

RunnerKit bundles a known-good GitHub Actions runner version (currently
`2.334.0`). When that version drifts behind GitHub's deprecation horizon,
`runnerkit doctor` warns:

```
- runner_version_stale (warning)
  Evidence:    installed runner version 2.330.0 is older than bundled pin 2.334.0
  Remediation: runnerkit upgrade-runner --repo owner/name
```

Roll the host runner forward:

```
runnerkit upgrade-runner --repo owner/name
```

For ephemeral runners:
- If the runner is **terminated** (one-shot already completed): no-op. The
  next `runnerkit up --mode ephemeral` uses the bundled pin.
- If the runner is **waiting** or **busy**: refused without `--force`.
  Adding `--force` will drop the registration / kill the running job.

## 3. State migrations

State migrations are forward-only and automatic. When you upgrade RunnerKit
to a release that bumps `schema_version` (e.g., from `"1"` to `"2"`),
the next CLI invocation that reads state will:

1. Write a side-by-side backup at
   `~/.local/state/runnerkit/state.json.backup-v<old>-<RFC3339>` (e.g.,
   `state.json.backup-v1-20260615T143000Z`).
2. Migrate the in-memory state forward.
3. Save the migrated state via the same atomic-write mechanism used for all
   state mutations.

If you DOWNGRADE RunnerKit and the older binary encounters a state file with
a newer `schema_version` than it knows, the older binary refuses to mutate
and exits with code `7` (`ExitStateSchemaTooNew`). The error message tells
you to run `runnerkit upgrade`. Your state file is untouched.

If something goes wrong during a migration, the side-by-side backup file
contains your original state byte-for-byte; you can restore it with:

```
cp ~/.local/state/runnerkit/state.json.backup-v1-<timestamp> ~/.local/state/runnerkit/state.json
```

(Note: this WILL re-trigger the migration on the next CLI invocation. If you
need to stay on the older format, downgrade RunnerKit too.)
```
  </action>
  <verify>
    <automated>go test ./internal/cli -run 'TestUpgrade_DetectsHomebrew|TestUpgrade_DetectsBinaryChannel|TestUpgrade_JSONContract|TestUpgradeRunner_Persistent_ReAppliesWithNewPin|TestUpgradeRunner_Ephemeral_TerminalNoOp|TestUpgradeRunner_Ephemeral_WaitingRefusesWithoutForce' -count=1 && go test ./internal/ops -run TestDoctor_StaleRunnerVersion -count=1 && go vet ./internal/cli/... ./internal/ops/... ./internal/bootstrap/... && grep -q "newUpgradeCommand" internal/cli/root.go && grep -q "newUpgradeRunnerCommand" internal/cli/root.go && grep -q "runner_version_stale" internal/ops/doctor.go && test -f docs/upgrade.md && grep -q "runnerkit upgrade-runner" docs/upgrade.md && grep -q "schema_version" docs/upgrade.md</automated>
  </verify>
  <acceptance_criteria>
    - `internal/cli/upgrade.go` exports `newUpgradeCommand(...)` and a `detectChannel(execPath string) string` function returning `"homebrew"` for paths containing `/Cellar/runnerkit/` or `/Caskroom/runnerkit/`, `"binary"` otherwise containing the binary name, `"unknown"` as fallback.
    - `internal/cli/upgrade.go` JSON output schema: `{ok bool, channel string, commands []string, current string, latest string, notes string?}`.
    - `internal/cli/upgrade.go` does NOT spawn any subprocess to actually upgrade (search for `exec.Command` in upgrade.go: must NOT be present).
    - `internal/cli/upgrade_runner.go` calls `bootstrap.Apply` for persistent and `bootstrap.ApplyEphemeral` for ephemeral RepositoryState; uses `bootstrap.RunnerVersion` as the pin; updates `repoState.RunnerTemplateVersion` ONLY after the apply returns nil.
    - `internal/cli/upgrade_runner.go` refuses without `--force` when `Ephemeral.Enabled && FinalizerStatus in {"waiting", "busy", ""}`.
    - `internal/cli/upgrade_runner.go` no-ops when `Ephemeral.Enabled && FinalizerStatus in {"completed", "ttl_expired"}` and prints the literal sentence "Ephemeral runner is one-shot and already terminated. The next `runnerkit up --mode ephemeral` will use the bundled pin".
    - `internal/cli/root.go` registers both new commands AFTER `newStateCommand`.
    - `internal/ops/doctor.go::BuildDoctorReport` emits a finding with `ID == "runner_version_stale"` AND `Severity == SeverityWarning` AND remediation containing the literal `runnerkit upgrade-runner` whenever `RunnerTemplateVersion != "" && RunnerTemplateVersion != bootstrap.RunnerVersion`.
    - All 7 tests in `<behavior>` pass.
    - `go vet ./internal/cli/... ./internal/ops/... ./internal/bootstrap/...` passes.
    - `docs/upgrade.md` exists with the three-section structure (CLI upgrade, runner pin upgrade, state migrations) and includes the literal commands `runnerkit upgrade`, `runnerkit upgrade-runner`, and `state.json.backup-v`.
    - All validation matrix rows for D-07/D-08 (lines 58-64) are green.
  </acceptance_criteria>
  <done>`runnerkit upgrade` (channel-detect, print-only, JSON contract) and `runnerkit upgrade-runner` (re-Apply with bundled pin, ephemeral-aware refuse-without-force) commands registered; doctor `runner_version_stale` warning finding emitted; docs/upgrade.md user-facing guide; 7 tests green.</done>
</task>

</tasks>

<verification>
Phase-level checks for Plan 06-02 completion:

1. `go test ./internal/state/... ./internal/update/... ./internal/cli/... ./internal/ops/... -count=1` passes.
2. `go test ./... -count=1 -race` passes (full suite per `06-VALIDATION.md` Sampling Rate).
3. Loading a v1 state.json through the CLI (e.g., `runnerkit status` against fixtures) produces a `state.json.backup-v1-*Z` sibling AND saves a v2 state.json.
4. `runnerkit upgrade --json` returns valid JSON with `channel`, `commands`, `current`, `latest` keys.
5. `runnerkit doctor` against a fixture with stale `RunnerTemplateVersion` includes a `runner_version_stale` finding.
6. `runnerkit up --json` does NOT print the `runnerkit X.Y.Z available` notice on stderr (silent in JSON mode).

Validation matrix coverage (`06-VALIDATION.md`):
- Lines 54-57 (D-06 lazy update check, 4 tests): satisfied by Task 2.
- Lines 58-60 (D-07 upgrade channel detect, 3 tests): satisfied by Task 3.
- Lines 61-63 (D-08 upgrade-runner persistent + ephemeral terminal/waiting, 3 tests): satisfied by Task 3.
- Line 64 (D-08 doctor stale-runner finding): satisfied by Task 3.
- Lines 65-68 (D-09 state migration: forward, backup, refuse-newer, atomic, 4 tests): satisfied by Task 1.

All 16 D-06..D-09 validation rows are green at the end of this plan.
</verification>

<success_criteria>
- State migration framework: SchemaVersion="2", forward-only Migrate chain, side-by-side backup BEFORE mutation, ErrSchemaTooNew sentinel, ExitStateSchemaTooNew=7 exit code, all 5 state tests green.
- Lazy update check: `internal/update/MaybePrint` + `internal/cli/maybeShowUpdateNotice` integrated via defer into runUp/runStatus/runDoctor; 6 tests green covering all silent-paths (JSON, CI, env, no-net, cache, ETag/304); cache file at `<state-dir>/update-check.json` mode 0600 atomic-written.
- `runnerkit upgrade`: channel-detect (homebrew via Cellar/Caskroom; binary; unknown), print-only, JSON contract `{ok, channel, commands, current, latest}`; 3 tests green.
- `runnerkit upgrade-runner`: re-applies bootstrap.Apply or ApplyEphemeral with `bootstrap.RunnerVersion` pin; persists RunnerTemplateVersion only on success; refuses without --force on waiting/busy ephemeral; no-op on completed/ttl_expired ephemeral; 3 tests green.
- `runnerkit doctor`: emits `runner_version_stale` warning finding when observed RunnerTemplateVersion < bundled pin; 1 test green.
- `docs/upgrade.md`: user-facing 3-section guide.
- `go test ./... -count=1 -race` passes.
- All hard rules from `<phase_specific_guidance>` Hard rules 5, 6, 7 are satisfied.
- Phase 5 ephemeral lifecycle invariants (waiting/busy/completed/ttl_expired) are honored — the existing `up_ephemeral_test.go` and other Phase 5 tests still pass.
</success_criteria>

<output>
After completion, create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-02-SUMMARY.md` summarizing:
- Files added (`internal/update/check.go`, `version.go`, `check_test.go`, `internal/cli/update_notice.go`, `upgrade.go`, `upgrade_test.go`, `upgrade_runner.go`, `upgrade_runner_test.go`, `internal/state/migrations_test.go`, `docs/upgrade.md`).
- Files modified (`internal/state/schema.go` — SchemaVersion bump, `internal/state/migrations.go` — full body replacement, `internal/state/store.go` — backup-before-Migrate, `internal/cli/exit.go` — ExitStateSchemaTooNew, `internal/cli/up.go`/`status.go`/`doctor.go` — defer maybeShowUpdateNotice, `internal/cli/root.go` — register new commands, `internal/ops/doctor.go` — runner_version_stale finding, `internal/bootstrap/package.go` — comment block, `go.mod`/`go.sum` — hashicorp/go-version v1.9.0).
- Cross-plan note: `internal/ops/doctor.go` is also touched by Plan 06-03 for RKD-code wiring; the two changes do not collide (different lines).
- Locked decisions implemented (D-06, D-07, D-08, D-09).
- Validation matrix rows closed (16 rows from 06-VALIDATION.md lines 54-68).
- Phase 5 invariants preserved (waiting/busy/completed/ttl_expired ephemeral lifecycle gating in upgrade-runner).
</output>
