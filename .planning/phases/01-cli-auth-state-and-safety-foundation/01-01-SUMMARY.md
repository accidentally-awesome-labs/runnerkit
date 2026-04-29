---
phase: 01-cli-auth-state-and-safety-foundation
plan: "01"
subsystem: cli
tags: [go, cobra, cli, renderer, redaction]
requires: []
provides:
  - Buildable `runnerkit` Go/Cobra CLI skeleton
  - Root, version, and Phase 1 `up` commands
  - Shared terminal/JSON renderer and prompt contracts
  - Central redaction package for tokens, credentials, keys, and machine refs
affects: [phase-1, cli, github-auth, state, safety]
tech-stack:
  added: [Go module, github.com/spf13/cobra]
  patterns:
    - Thin Cobra command layer with injectable dependencies
    - Typed exit-code errors via `ExitError`
    - Static terminal renderer with JSON-only automation mode
    - Redaction-before-output for human and JSON paths
key-files:
  created:
    - go.mod
    - go.sum
    - cmd/runnerkit/main.go
    - internal/cli/root.go
    - internal/cli/exit.go
    - internal/cli/up.go
    - internal/cli/root_test.go
    - internal/cli/up_test.go
    - internal/ui/output.go
    - internal/ui/output_test.go
    - internal/ui/prompt.go
    - internal/redact/redact.go
    - internal/redact/redact_test.go
    - internal/testsupport/golden.go
  modified: []
key-decisions:
  - "Use Cobra v1.10.1 for the CLI command tree."
  - "Keep `runnerkit up` honest in Phase 1 by rendering placeholder repo/auth/safety/state adapters and `runner_installed: false`."
  - "Centralize all current human and JSON output through a renderer that adds `redactions_applied: true`."
patterns-established:
  - "Command dependencies are injected through `cli.Dependencies` for tests and future adapters."
  - "Command failures return typed `ExitError` values that map to the UI-SPEC exit codes."
  - "Terminal output uses static, log-friendly step lines with ASCII fallback outside rich TTY contexts."
requirements-completed: [CLI-01, CLI-02, STATE-02]
duration: 13 min
completed: 2026-04-29
---

# Phase 1 Plan 01: CLI Skeleton, Output, Wizard, and Redaction Summary

**Go/Cobra RunnerKit CLI with deterministic version/up output, automation flags, typed exit codes, and redacted human/JSON rendering**

## Performance

- **Duration:** 13 min
- **Started:** 2026-04-29T02:03:04Z
- **Completed:** 2026-04-29T02:16:25Z
- **Tasks:** 3
- **Files modified:** 14

## Accomplishments

- Initialized the Go module and runnable `cmd/runnerkit` binary with Cobra root, `version`, and `up` command routing.
- Added UI-SPEC exit code constants and `ExitError` mapping for invalid input, input-required, cancellation, and future auth/state/safety failures.
- Built a shared terminal/JSON renderer, prompt abstraction, golden-test helpers, and central redactor with tests for GitHub tokens, runner tokens, SSH keys, provider credentials, and machine refs.
- Implemented `runnerkit up` as a safe Phase 1 foundation wizard with six ordered steps, automation flags, JSON output, no state persistence, and copy that explicitly says no runner is installed.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: CLI skeleton tests** - `18baf39` (test)
2. **Task 1 GREEN: CLI skeleton implementation** - `7a12381` (feat)
3. **Task 2 RED: renderer/redaction tests** - `64233df` (test)
4. **Task 2 GREEN: renderer, prompt, redaction implementation** - `177c161` (feat)
5. **Task 3 RED: up wizard tests** - `e2f644b` (test)
6. **Task 3 GREEN: up wizard implementation** - `a2d0164` (feat)

_Note: Tasks used the requested TDD red/green flow, so each task produced a failing-test commit and a passing implementation commit._

## Files Created/Modified

- `go.mod` / `go.sum` - Go module and Cobra dependency metadata.
- `cmd/runnerkit/main.go` - Binary entrypoint wiring injected dependencies and process exit mapping.
- `internal/cli/root.go` - Root command, persistent `--json`/`--no-color` flags, version command, and shared renderer construction.
- `internal/cli/exit.go` - Exit code constants and typed `ExitError` mapping.
- `internal/cli/up.go` - Phase 1 `runnerkit up` foundation wizard, automation flags, JSON payload, and no-TTY input-required handling.
- `internal/cli/root_test.go` / `internal/cli/up_test.go` - CLI routing, JSON, wizard, and exit-code tests.
- `internal/ui/output.go` / `internal/ui/output_test.go` - Human/JSON renderer, glyph/ASCII fallbacks, wrapping, and redaction checks.
- `internal/ui/prompt.go` - Prompt and option interfaces for future interactive implementations.
- `internal/redact/redact.go` / `internal/redact/redact_test.go` - Sensitive value registration, pattern redaction, and JSON field sanitization.
- `internal/testsupport/golden.go` - Output comparison helpers for future golden tests.

## Decisions Made

- Used Cobra for command routing to match the research recommendation and keep the command layer thin.
- Treated `runnerkit up` repository/auth/safety/state integrations as explicit placeholders in Plan 01-01; Plans 01-02 and 01-03 replace them with real GitHub and state adapters.
- Used static step output instead of a full-screen TUI so logs, tests, no-TTY runs, and screen readers receive complete lines.
- Made the renderer always add `redactions_applied: true` to JSON object payloads and sanitize bytes before writing.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added minimal UI package during Task 1**
- **Found during:** Task 1 (CLI skeleton)
- **Issue:** The required `cli.Dependencies` contract references `ui.TerminalCapabilities` and `ui.Prompter`, but Task 2 was scheduled to create `internal/ui` later. Without a minimal package, Task 1 could not compile.
- **Fix:** Created a minimal `internal/ui/output.go` in Task 1, then expanded it into the full renderer in Task 2.
- **Files modified:** `internal/ui/output.go`
- **Verification:** `go test ./...` passed after Task 1 GREEN and after Task 2 GREEN.
- **Committed in:** `7a12381`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required only to preserve the planned interface and buildability; no scope creep.

## Known Stubs

- `internal/cli/up.go` - Placeholder repo/auth/safety/state providers and `auth_source: "placeholder"` are intentional for Plan 01-01. Plan 01-02 replaces repo/auth/safety with real GitHub adapters; Plan 01-03 replaces state preview placeholders with versioned persistence.
- `cmd/runnerkit/main.go` - Version defaults to `dev` until release packaging supplies a build-time value in a later phase.

## Issues Encountered

- `go run` on this Go toolchain wraps non-zero program exits as process exit `1` while printing `exit status 6`. The direct built binary exits `6` for `runnerkit up --non-interactive --no-color`, and command-level tests verify `ExitInputRequired` mapping. All other required `go run` checks passed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for Plan 01-02 to add GitHub repository resolution, least-privilege auth checks, and real safety gates on top of the stable CLI/output/redaction foundation.
- Ready for Plan 01-03 to replace placeholder state preview with versioned, secret-free local state persistence.

---
*Phase: 01-cli-auth-state-and-safety-foundation*
*Completed: 2026-04-29*

## Self-Check: PASSED
