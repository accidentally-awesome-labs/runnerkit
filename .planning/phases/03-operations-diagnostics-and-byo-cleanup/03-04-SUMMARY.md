---
phase: 03-operations-diagnostics-and-byo-cleanup
plan: "04"
subsystem: operations
tags: [down, cleanup, deregistration, stale-runner, state-checkpoints]
requires:
  - phase: 03-operations-diagnostics-and-byo-cleanup
    provides: Status facts, state helpers, removal token scripts, recovery/checkpoint patterns
provides:
  - Safe BYO cleanup artifact planner with path validation and shared-parent guardrails
  - runnerkit down command with dry-run, --yes, prompts, JSON, stale GitHub-only deletion, and partial cleanup checkpoints
  - Host-side config removal, GitHub stale delete fallback, service uninstall, exact file removal, and local state removal
  - BYO cleanup documentation with safety boundaries and stale GitHub runner guidance
affects: [phase-04, cloud-destroy, cleanup, docs]
tech-stack:
  added: []
  patterns:
    [artifact cleanup plan, partial cleanup checkpoint, stale GitHub deletion]
key-files:
  created:
    - internal/ops/cleanup.go
    - internal/ops/cleanup_test.go
    - internal/cli/down.go
    - internal/cli/down_test.go
  modified:
    - internal/cli/root.go
    - internal/cli/docs_test.go
    - README.md
    - docs/byo-quickstart.md
key-decisions:
  - "BYO cleanup uses down; destroy remains reserved for future cloud resources."
  - "down --yes only removes recorded runner-specific artifacts and never targets shared /var/lib/runnerkit parents directly."
patterns-established:
  - "Partial cleanup keeps state with remote_cleanup_pending or github_cleanup_pending checkpoints."
  - "Local-state-missing stale deletion requires explicit --github-runner-id or an unambiguous RunnerKit-labeled runner name."
requirements-completed: [GH-03, CLEAN-02, CLEAN-03, STATE-02]
duration: 52 min
completed: 2026-04-29
---

# Phase 03 Plan 04: BYO Cleanup and Stale Deregistration Summary

**Safe `runnerkit down` cleanup with artifact planning, stale GitHub deletion, exact host artifact removal, and partial-state checkpoints**

## Performance

- **Duration:** 52 min
- **Started:** 2026-04-30T01:12:00Z
- **Completed:** 2026-04-30T02:04:00Z
- **Tasks:** 5
- **Files modified:** 8

## Accomplishments

- Added cleanup planning and path validation that rejects shared parents such as `/var/lib/runnerkit`, `/opt`, and `/`.
- Added `runnerkit down` with dry-run, JSON, no-TTY `--yes` enforcement, interactive artifact prompts, GitHub deletion, host-side unregister, service uninstall, exact file removal, state removal, and partial cleanup persistence.
- Added docs for BYO cleanup boundaries, stale GitHub deletion, and `remote_cleanup_pending` state.

## Task Commits

1. **Task 03-04-01: Cleanup plan and path validation** - `6a0d56d` (feat)
2. **Tasks 03-04-02/03/04: Down CLI, execution, stale/partial handling** - `f9439a6` (feat)
3. **Task 03-04-05: Cleanup docs** - `a50a4ca` (docs)

## Files Created/Modified

- `internal/ops/cleanup.go` - Cleanup artifact plan and safe path validation.
- `internal/cli/down.go` - CLI command, confirmations, stale deletion, host cleanup, partial checkpoint handling, and JSON output.
- `internal/cli/down_test.go` - Dry-run, JSON, prompts, complete cleanup, stale GitHub-only deletion, partial cleanup, ambiguity, and pending-state tests.
- `README.md` and `docs/byo-quickstart.md` - BYO cleanup commands and safety boundaries.

## Decisions Made

- SSH-unreachable cleanup can still delete the stale GitHub runner and preserves local state with `remote_cleanup_pending`.
- GitHub delete failures preserve state with `github_cleanup_pending` rather than pretending cleanup completed.
- Stale GitHub-only deletion without local state is allowed only with explicit runner ID or an unambiguous RunnerKit-labeled runner name.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Combined down execution tasks into one buildable CLI commit**

- **Found during:** Tasks 03-04-02 through 03-04-04
- **Issue:** Dry-run, host unregister, GitHub fallback, partial checkpoints, and state removal share the same command execution path and tests.
- **Fix:** Landed them in one cohesive CLI commit while keeping tests separated by behavior.
- **Files modified:** `internal/cli/down.go`, `internal/cli/down_test.go`
- **Verification:** `go test ./... && go vet ./...`
- **Committed in:** `f9439a6`

---

**Total deviations:** 1 auto-fixed (Rule 3).
**Impact on plan:** No product scope change; implementation was grouped to keep the command path coherent and buildable.

## Issues Encountered

None beyond the grouped command-path commit documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 4 cloud cleanup can reuse the artifact-plan pattern, explicit stale GitHub deletion behavior, partial checkpoint semantics, and docs distinction between BYO `down` and future cloud `destroy`.

## Self-Check: PASSED

- `go test ./...` passed.
- `go vet ./...` passed.
- Down greps and `go test ./internal/cli -run TestDown` passed.

---

_Phase: 03-operations-diagnostics-and-byo-cleanup_
_Completed: 2026-04-29_
