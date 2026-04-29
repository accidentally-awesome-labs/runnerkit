---
phase: 03-operations-diagnostics-and-byo-cleanup
plan: "02"
subsystem: operations
tags: [logs, doctor, diagnostics, redaction, troubleshooting]
requires:
  - phase: 03-operations-diagnostics-and-byo-cleanup
    provides: Plan 03-01 status model, source facts, probes, and state helpers
provides:
  - runnerkit logs with bounded systemd journal and runner _diag collection
  - runnerkit doctor with stable finding IDs, evidence, remediation, and JSON output
  - Diagnostic redaction regression coverage for logs, machine refs, tokens, private keys, and provider credentials
  - BYO troubleshooting documentation that starts with status/logs/doctor
affects: [phase-03-03, phase-03-04, recovery, cleanup, docs]
tech-stack:
  added: []
  patterns:
    [bounded log collection, stable doctor findings, machine-ref redaction]
key-files:
  created:
    - internal/cli/logs.go
    - internal/cli/logs_test.go
    - internal/cli/doctor.go
    - internal/cli/doctor_test.go
    - internal/cli/quote.go
    - internal/ops/logs.go
    - internal/ops/logs_test.go
    - internal/ops/doctor.go
    - internal/ops/doctor_test.go
  modified:
    - internal/cli/root.go
    - internal/cli/docs_test.go
    - internal/redact/redact_test.go
    - README.md
    - docs/byo-quickstart.md
key-decisions:
  - "Logs and doctor are read-only and route all collected remote evidence through the shared redacting renderer."
  - "Doctor uses stable finding IDs and exact remediation commands instead of adding an automatic --fix path."
patterns-established:
  - "Diagnostic commands register sensitive machine refs before rendering support-like output."
  - "Human diagnostics collapse pass findings by default, while JSON keeps all findings."
requirements-completed: [REL-02, REL-03, STATE-02]
duration: 38 min
completed: 2026-04-29
---

# Phase 03 Plan 02: Logs and Doctor Diagnostics Summary

**Bounded redacted log collection plus read-only doctor findings with exact remediation commands and troubleshooting docs**

## Performance

- **Duration:** 38 min
- **Started:** 2026-04-29T23:48:00Z
- **Completed:** 2026-04-30T00:26:00Z
- **Tasks:** 4
- **Files modified:** 14

## Accomplishments

- Added `runnerkit logs` with `--since`, `--lines`, systemd journal collection, runner `_diag` collection, human output, and JSON output.
- Added `runnerkit doctor` with stable findings, severity/evidence/remediation rendering, verbose pass findings, and JSON parity.
- Extended redaction regression coverage and docs so users run status/logs/doctor before manual SSH troubleshooting.

## Task Commits

1. **Task 03-02-01: Logs collection** - `d7ae4d4` (feat)
2. **Task 03-02-02: Doctor findings** - `9a126a1` (feat)
3. **Task 03-02-03: Diagnostic redaction coverage** - `68c3c46` (test)
4. **Task 03-02-04: Troubleshooting docs** - `3936a6e` (docs)

## Files Created/Modified

- `internal/ops/logs.go` - Remote journal and runner diag collection scripts and bundle model.
- `internal/cli/logs.go` - CLI command, defaults, redacted human output, and JSON payload.
- `internal/ops/doctor.go` - Doctor findings, deep check input model, and remediation mapping.
- `internal/cli/doctor.go` - CLI command, deep checks, pass-finding collapse, and JSON payload.
- `internal/redact/redact_test.go` - Regression coverage for Phase 3 diagnostic secret patterns.
- `README.md` and `docs/byo-quickstart.md` - Read-only operations troubleshooting path.

## Decisions Made

- `runnerkit logs` defaults to `--since 1h --lines 200` and clamps very large line counts in the ops layer.
- `runnerkit doctor` intentionally does not expose `--fix`; remediation points to logs, recover dry-runs, and down dry-runs.
- Logs/doctor register `MachineRef` for diagnostic output so support-like evidence can hide sensitive host references.

## Deviations from Plan

None - plan executed as written.

**Total deviations:** 0 auto-fixed.
**Impact on plan:** No scope change.

## Issues Encountered

- Multi-line private key redaction required redacting the full log section before splitting into terminal lines; fixed in the logs renderer and covered by tests.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan 03-03 can use doctor remediation IDs, logs commands, state operation checkpoints, redaction coverage, and the shared status observation path to implement guided recovery.

## Self-Check: PASSED

- `go test ./...` passed.
- `go vet ./...` passed.
- Docs grep for `runnerkit doctor --repo owner/name` passed.

---

_Phase: 03-operations-diagnostics-and-byo-cleanup_
_Completed: 2026-04-29_
