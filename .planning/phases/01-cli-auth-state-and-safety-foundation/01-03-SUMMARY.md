---
phase: 01-cli-auth-state-and-safety-foundation
plan: "03"
subsystem: state
tags: [go, local-state, labels, workflow, cobra, redaction]
requires:
  - phase: 01-cli-auth-state-and-safety-foundation
    provides: CLI skeleton/output/redaction from 01-01 and GitHub repo/auth/safety contracts from 01-02
provides:
  - Versioned, migration-ready, secret-free RunnerKit local state schema and atomic JSON store
  - Optional `.runnerkit/config.yaml` safe project config path/schema
  - Stable RunnerKit runner names, labels, workflow snippet, and self-hosted-alone warning
  - Reusable workflow plan/checklist/apply primitives with foundation step IDs
  - End-to-end `runnerkit up` state preview/save and `runnerkit state show --repo` inspection
  - Explicit `--replace` safety for non-interactive existing-state replacement
affects: [phase-1, cli, state, labels, workflow, byo-runner, cloud-provider, diagnostics]
tech-stack:
  added: []
  patterns:
    - User-local JSON state with schema_version and migration hook
    - Atomic temp-file write, fsync, rename, 0700 directories, and 0600 state files
    - Repo-scoped deterministic label/name builder
    - Plan/checkpoint/apply primitives separated from CLI rendering
key-files:
  created:
    - internal/state/schema.go
    - internal/state/store.go
    - internal/state/config.go
    - internal/state/migrations.go
    - internal/state/state_test.go
    - internal/labels/labels.go
    - internal/labels/labels_test.go
    - internal/workflow/plan.go
    - internal/workflow/plan_test.go
    - internal/cli/state.go
    - internal/cli/up_integration_test.go
  modified:
    - internal/cli/root.go
    - internal/cli/up.go
    - internal/ui/output.go
    - internal/redact/redact.go
    - internal/redact/redact_test.go
key-decisions:
  - "Persist Phase 1 foundation state as versioned, human-debuggable JSON in the user-local RunnerKit state directory, with tests injecting a temp base directory."
  - "Use `runnerkit-owner-repo` plus OS, arch, and mode labels as the stable recommended routing set and never recommend `self-hosted` alone."
  - "Require explicit `--yes --replace` or typed `replace owner/repo` before overwriting existing repository state."
  - "Expose safe provider metadata in state show while continuing to redact provider credential fields."
patterns-established:
  - "State writes go through `state.Store` and `RepositoryState` rather than ad hoc CLI file writes."
  - "Human and JSON `up` output are built from label/workflow/state primitives shared with persisted state."
  - "Future mutating flows can reuse `workflow.Plan`, `Step`, `Checkpoint`, `Checklist`, and `Apply`."
requirements-completed: [CLI-01, CLI-02, STATE-01, STATE-02]
duration: 8 min
completed: 2026-04-29
---

# Phase 1 Plan 03: State, Labels, Workflow, and Save/Show Summary

**Versioned secret-free RunnerKit foundation state with atomic saves, deterministic labels, reusable workflow plans, `runnerkit up` persistence, and redacted `state show` inspection**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-29T02:28:14Z
- **Completed:** 2026-04-29T02:35:37Z
- **Tasks:** 3
- **Files modified:** 16

## Accomplishments

- Added schema v1 state records covering repo scope, auth reference, runner identity, labels/snippet, machine/provider placeholders, cleanup metadata, safety metadata, timestamps, and RunnerKit version.
- Implemented atomic user-local JSON persistence with 0700 state directories, 0600 state files, temp-file writes, fsync/rename, migration hook, and secret-key validation.
- Added stable `runnerkit-owner-repo` label/name generation and exact `runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]` guidance.
- Added reusable workflow plan primitives and foundation step IDs: `resolve_repo`, `verify_auth`, `check_safety`, `preview_state`, and `save_state`.
- Wired `runnerkit up` to preview/save Phase 1 foundation state and added `runnerkit state show --repo` in human and JSON modes.
- Added `--replace` behavior so existing repository state cannot be overwritten non-interactively without explicit intent.

## Task Commits

Each task was committed atomically using the TDD red/green flow:

1. **Task 1 RED: state persistence tests** - `2cc5675` (test)
2. **Task 1 GREEN: versioned state store** - `e598edb` (feat)
3. **Task 2 RED: label/workflow tests** - `5959219` (test)
4. **Task 2 GREEN: labels and workflow primitives** - `891471c` (feat)
5. **Task 3 RED: CLI integration tests** - `04855b8` (test)
6. **Task 3 GREEN: up save/state show wiring** - `9018fcd` (feat)
7. **Task 3 fix: safe provider state display** - `f9c3694` (fix)

## Files Created/Modified

- `internal/state/schema.go` - State v1 schema for repo, auth, runner, machine, provider, cleanup, safety, and version metadata.
- `internal/state/store.go` - Atomic JSON state store, default/injected state paths, migration loading, repo upsert, and raw secret-key validation.
- `internal/state/config.go` - Optional safe project config path/schema for `.runnerkit/config.yaml`.
- `internal/state/migrations.go` - Migration hook for schema version 1.
- `internal/state/state_test.go` - Filesystem, permissions, migration, config path, and secret persistence tests.
- `internal/labels/labels.go` / `internal/labels/labels_test.go` - Deterministic runner name, label, snippet, and warning builder.
- `internal/workflow/plan.go` / `internal/workflow/plan_test.go` - Plan, step, checkpoint, checklist, and apply primitives.
- `internal/cli/up.go` - State preview/save, label/snippet rendering, plan checklist, save confirmation, and `--replace` enforcement.
- `internal/cli/state.go` - `runnerkit state show --repo` human/JSON command.
- `internal/cli/up_integration_test.go` - End-to-end dry-run no-write, save JSON, state show, and replacement tests.
- `internal/cli/root.go` - Registered `state` command and test-injectable state base directory.
- `internal/ui/output.go` - Preserves copy-paste `runs-on:` snippets on one line.
- `internal/redact/redact.go` / `internal/redact/redact_test.go` - Keeps safe provider metadata visible while redacting credential fields.

## Decisions Made

- Used JSON state in the user-local RunnerKit state directory with `RUNNERKIT_STATE_DIR`/injected base directory support for deterministic tests and verification.
- Kept Phase 1 machine/provider values as explicit placeholders (`phase1-placeholder`, `none`) while preserving schema space for later BYO/cloud cleanup metadata.
- Made `runnerkit up --repo owner/repo --yes --json` save state using deterministic fake default GitHub adapters and no live network/auth.
- Chose `--yes --replace` for non-interactive replacement and typed `replace owner/repo` for future interactive replacement.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Preserved exact workflow snippet output**
- **Found during:** Task 3 (CLI integration verification)
- **Issue:** The generic line wrapper split the copy-paste `runs-on:` snippet, which would fail the required exact output contract.
- **Fix:** Updated `internal/ui/output.go` to keep `runs-on:` command snippets on one line.
- **Files modified:** `internal/ui/output.go`
- **Verification:** Dry-run command output includes the exact required snippet.
- **Committed in:** `9018fcd`

**2. [Rule 1 - Bug] Avoided over-redacting safe provider state metadata**
- **Found during:** Task 3 (state show verification)
- **Issue:** Redaction treated any JSON key containing `provider` as a credential and hid safe provider metadata/placeholders from `state show`.
- **Fix:** Narrowed provider redaction to credential/secret keys while preserving safe provider state fields; added a regression test.
- **Files modified:** `internal/redact/redact.go`, `internal/redact/redact_test.go`
- **Verification:** `go test ./... && go vet ./...`; `state show --json` displays provider placeholders and no forbidden token/private-key/provider credential text.
- **Committed in:** `f9c3694`

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes strengthened the planned output/state contracts without adding scope.

## Known Stubs

- `internal/cli/up.go` - `MachineRef{Kind: "phase1-placeholder"}` and provider `none` are intentional Phase 1 placeholders. Phase 2 BYO and Phase 4 cloud plans populate real machine/provider identity.
- `internal/state/state_test.go` - Empty cleanup/provider slices exercise secret-free placeholder persistence until later setup/cleanup plans create real resources.

## Issues Encountered

None beyond the auto-fixed output/redaction issues above.

## Verification

- `go test ./... && go vet ./...` passed.
- `RUNNERKIT_STATE_DIR=$(mktemp -d) go run ./cmd/runnerkit up --repo owner/repo --dry-run --yes --no-color` included the exact `runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]` snippet and did not create a state file.
- `RUNNERKIT_STATE_DIR=$(mktemp -d) go run ./cmd/runnerkit up --repo owner/repo --yes --json` emitted JSON only with `"runner_installed":false`, `"state_path"`, and `"redactions_applied":true`.
- `RUNNERKIT_STATE_DIR=<same temp dir> go run ./cmd/runnerkit state show --repo owner/repo --json` emitted JSON with `"redactions_applied":true` and no raw `token`, `registration_token`, `remove_token`, `private_key`, or `provider_credential` text.
- Grep confirmed the recommended output warning: `Do not use runs-on: self-hosted alone for RunnerKit-managed runners.`

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 1 foundation is complete: CLI/auth/safety output now has secret-free persistent state, deterministic labels, and safe inspection commands.
- Phase 2 can consume `RepositoryState`, `RunnerIdentity`, `MachineRef`, `ProviderRef`, `CleanupMetadata`, labels, and workflow plan primitives for the BYO persistent runner happy path.
- No live GitHub/network auth is required for current fake-adapter verification; a controlled live GitHub smoke remains recommended before public release.

---
*Phase: 01-cli-auth-state-and-safety-foundation*
*Completed: 2026-04-29*

## Self-Check: PASSED
