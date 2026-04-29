---
phase: 03-operations-diagnostics-and-byo-cleanup
plan: "03"
subsystem: operations
tags: [recover, recovery, systemd, reregister, github-runner]
requires:
  - phase: 03-operations-diagnostics-and-byo-cleanup
    provides: Plans 03-01 and 03-02 status observations, diagnostics, redaction, and checkpoints
provides:
  - Recovery plan builder with restart_service, reinstall_service, and reregister_runner actions
  - runnerkit recover command with --dry-run, --yes, service restart/reinstall, and re-registration
  - Fresh removal/registration token reconfiguration scripts that avoid token interpolation
  - Recovery documentation for stopped service, missing service, and re-registration cases
affects: [phase-03-04, cleanup, github-deregistration, docs]
tech-stack:
  added: []
  patterns:
    [guided mutation plan, fail-closed recovery, just-in-time runner tokens]
key-files:
  created:
    - internal/ops/recovery.go
    - internal/ops/recovery_test.go
    - internal/cli/recover.go
    - internal/cli/recover_test.go
  modified:
    - internal/cli/root.go
    - internal/bootstrap/script.go
    - internal/bootstrap/script_test.go
    - internal/cli/docs_test.go
    - README.md
    - docs/byo-quickstart.md
key-decisions:
  - "Recovery is distinct from up and starts with a dry-run plan before any mutation."
  - "Re-registration uses fresh removal and registration tokens, registers them with the redactor, and updates Cleanup.GitHubRunnerID only after online verification."
patterns-established:
  - "Mutating operations require --yes in non-interactive/JSON mode or an explicit confirmation prompt."
  - "Host-key mismatch and SSH-unreachable observations block recovery before remote mutation."
requirements-completed: [REL-04, GH-03]
duration: 44 min
completed: 2026-04-29
---

# Phase 03 Plan 03: Guided Persistent Runner Recovery Summary

**Guided recover command for service restart, service reinstall, and runner re-registration with fresh redacted GitHub tokens**

## Performance

- **Duration:** 44 min
- **Started:** 2026-04-30T00:27:00Z
- **Completed:** 2026-04-30T01:11:00Z
- **Tasks:** 4
- **Files modified:** 10

## Accomplishments

- Added deterministic recovery plan selection with unsafe-case blocking for host-key mismatch and SSH unreachable conditions.
- Added reusable bootstrap scripts for host-side runner removal and reconfiguration without embedding token values.
- Wired `runnerkit recover` with dry-run, confirmation/`--yes`, service restart/reinstall, re-registration, online verification, state update, JSON output, and docs.

## Task Commits

1. **Task 03-03-01: Recovery plan builder** - `162367c` (feat)
2. **Task 03-03-03: Reconfiguration scripts** - `36b41b9` (feat)
3. **Task 03-03-02/03: CLI recover restart/reinstall/reregister** - `878ee88` (feat)
4. **Task 03-03-04: Recovery docs** - `5dbccb8` (docs)

## Files Created/Modified

- `internal/ops/recovery.go` - Recovery plan model and action selection.
- `internal/cli/recover.go` - CLI command, recovery execution, token handling, and state checkpoint update.
- `internal/bootstrap/script.go` - Removal and reconfiguration script renderers.
- `internal/cli/recover_test.go` - Dry-run, restart, reinstall, confirmation, host-key block, and reregister state tests.
- `README.md` and `docs/byo-quickstart.md` - Recovery command guidance.

## Decisions Made

- Explicit recovery flags preserve requested action order while still blocking unsafe host-key/SSH cases.
- Re-registration updates local state only after `waitForRunnerOnline` finds the runner with expected labels.
- Recovery docs warn not to blindly rerun `runnerkit up` for repair.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Commit grouping for script dependency**

- **Found during:** Task 03-03-02/03 implementation
- **Issue:** `recover.go` depends on the new bootstrap reconfiguration helpers, so compiling intermediate commits required landing script helpers before the CLI command.
- **Fix:** Committed the script helper task before the combined CLI recovery commit.
- **Files modified:** `internal/bootstrap/script.go`, `internal/cli/recover.go`
- **Verification:** `go test ./... && go vet ./...`
- **Committed in:** `36b41b9`, `878ee88`

---

**Total deviations:** 1 auto-fixed (Rule 3).
**Impact on plan:** No product scope change; only commit ordering was adjusted to preserve buildability.

## Issues Encountered

None beyond the commit ordering dependency documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan 03-04 can reuse `RenderRemoveConfigScript`, state operation checkpoints, GitHub removal token handling, and recovery's stale re-registration mechanics to implement `runnerkit down` cleanup.

## Self-Check: PASSED

- `go test ./...` passed.
- `go vet ./...` passed.
- Recovery greps and `go test ./internal/cli -run TestRecover` passed.

---

_Phase: 03-operations-diagnostics-and-byo-cleanup_
_Completed: 2026-04-29_
