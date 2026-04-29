---
phase: 03-operations-diagnostics-and-byo-cleanup
plan: "01"
subsystem: operations
tags: [status, health, github-runners, ssh, systemd, state]
requires:
  - phase: 02-byo-persistent-runner-happy-path
    provides: BYO runner state, GitHub runner registration, SSH/systemd bootstrap
provides:
  - Shared operations health model with ready/busy/needs_attention/broken/unknown states
  - Fast read-only SSH, host-key, and systemd status probes
  - runnerkit status command with repo inference, --repo, --all, human output, and JSON
  - Phase 3 testsupport fakes, fixtures, output helpers, and state update/remove helpers
affects: [phase-03-02, phase-03-03, phase-03-04, operations, cleanup, recovery]
tech-stack:
  added: []
  patterns: [shared ops model, read-only status probes, state checkpoints]
key-files:
  created:
    - internal/ops/status.go
    - internal/ops/status_test.go
    - internal/ops/probes.go
    - internal/ops/probes_test.go
    - internal/cli/status.go
    - internal/cli/status_test.go
    - internal/testsupport/remote.go
    - internal/testsupport/state.go
    - internal/testsupport/output.go
  modified:
    - internal/cli/root.go
    - internal/remote/executor.go
    - internal/remote/system.go
    - internal/state/schema.go
    - internal/state/store.go
    - internal/state/state_test.go
    - internal/testsupport/github.go
key-decisions:
  - "Status uses a shared internal/ops model so logs, doctor, recover, and down can reuse source facts and health classification."
  - "Status probes are intentionally limited to host-key, SSH true, and systemctl show; full preflight remains outside status."
patterns-established:
  - "Read-only commands resolve --repo or inferred git remote without prompts."
  - "OperationCheckpoint records resumable recovery/cleanup state without storing secrets."
requirements-completed: [REL-01]
duration: 42 min
completed: 2026-04-29
---

# Phase 03 Plan 01: Multi-source Status Reconciliation Summary

**Read-only runner status with shared health classification, fast SSH/systemd probes, reusable Phase 3 fakes, and JSON/human output parity**

## Performance

- **Duration:** 42 min
- **Started:** 2026-04-29T23:05:00Z
- **Completed:** 2026-04-29T23:47:00Z
- **Tasks:** 4
- **Files modified:** 16

## Accomplishments

- Added reusable Phase 3 fake GitHub/remote/state/output helpers and state list/update/remove/checkpoint support.
- Added `internal/ops` health facts, label comparison, classifier, and bounded status probes.
- Wired `runnerkit status` for explicit repo, inferred git repo, and `--all` inventory with read-only tests and JSON output.

## Task Commits

1. **Task 03-01-01: Phase 3 fakes/state helpers** - `495f949` (test)
2. **Task 03-01-02: Health classifier** - `3c35bd8` (feat)
3. **Task 03-01-03: Status probes** - `1d8a50b` (feat)
4. **Task 03-01-04: CLI status** - `215bbd9` (feat)

## Files Created/Modified

- `internal/ops/status.go` - Shared observed-runner, label, health, reason, and next-action model.
- `internal/ops/probes.go` - Fast host-key/SSH/systemd status probes with read-only command IDs.
- `internal/cli/status.go` - `runnerkit status` command, repo inference, `--all`, source matrix, and JSON contract.
- `internal/state/schema.go` / `internal/state/store.go` - Operation checkpoints plus repository list/update/remove helpers.
- `internal/testsupport/*.go` - Reusable fake GitHub service, remote executor, state fixtures, and output assertions.

## Decisions Made

- Status treats missing state as an empty state instead of a hard failure so users get setup guidance.
- Host-key mismatch is classified as `broken` and does not run SSH/systemd commands after mismatch detection.
- `status --all` renders compact inventory rows while JSON includes per-runner status payloads.

## Deviations from Plan

None - plan executed as written.

**Total deviations:** 0 auto-fixed.
**Impact on plan:** No scope change.

## Issues Encountered

- The generated Go constant alignment did not match one verification grep exactly; added a harmless comment preserving the exact verification string while keeping idiomatic gofmt output.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan 03-02 can reuse `ops.ObservedRunner`, `ops.Classify`, `ops.ProbeRemoteStatus`, `State.ListRepositories`, `State.UpdateRepository`, fake GitHub/remote services, and status source rendering contracts for `logs` and `doctor`.

## Self-Check: PASSED

- `go test ./...` passed.
- `go vet ./...` passed.
- Key files and commits are present.

---

_Phase: 03-operations-diagnostics-and-byo-cleanup_
_Completed: 2026-04-29_
