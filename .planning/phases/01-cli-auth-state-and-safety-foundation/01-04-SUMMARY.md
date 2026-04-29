---
phase: 01-cli-auth-state-and-safety-foundation
plan: "04"
subsystem: github-auth
tags: [go, cobra, github-rest, auth, safety, redaction]

requires:
  - phase: 01-02
    provides: GitHub auth discovery, REST client, runner token checks, and safety gates
  - phase: 01-03
    provides: state persistence, labels, workflow primitives, and CLI integration tests
provides:
  - Real production GitHub service for auth discovery, repository metadata, and runner-management permission checks
  - Production CLI defaults wired to github.OSCommandRunner and gh.NewService
  - Regression tests for fail-closed missing credentials and public repository metadata safety blocking
affects:
  [
    phase-1-verification,
    github-auth,
    cli-defaults,
    safety-gate,
    state-write-safety,
  ]

tech-stack:
  added: []
  patterns:
    - Real adapters are production defaults; deterministic fakes are explicit test dependencies only
    - GitHub credentials are cached in memory on the service and never persisted
    - Registration-token creation remains the runner-management permission probe

key-files:
  created:
    - internal/github/service.go
    - internal/github/service_test.go
  modified:
    - cmd/runnerkit/main.go
    - internal/cli/root.go
    - internal/cli/up.go
    - internal/cli/root_test.go
    - internal/cli/up_test.go
    - internal/cli/up_integration_test.go

key-decisions:
  - "Production runnerkit defaults now use gh.NewService with github.OSCommandRunner instead of a fake-permitted GitHub service."
  - "CLI tests that need private-repo success now inject fakePermittedGitHubService explicitly."
  - "The real GitHub service caches discovered credentials in memory only and constructs REST clients per operation with shared redaction."

patterns-established:
  - "Production dependency normalization: nil CommandRunner becomes github.OSCommandRunner; nil GitHub becomes gh.NewService."
  - "Default-path regression tests use test-only GitHubEnv, GitHubBaseURL, and GitHubHTTPClient overrides instead of replacing the GitHub service."

requirements-completed: [CLI-01, CLI-02, GH-01, STATE-01, STATE-02]

duration: 45 min
completed: 2026-04-29
---

# Phase 01 Plan 04: Production GitHub Auth Wiring Summary

**Production `runnerkit up` now performs real GitHub credential discovery, registration-token permission checks, repository metadata reads, and public-repo safety blocking by default.**

## Performance

- **Duration:** 45 min
- **Started:** 2026-04-29T16:15:00Z
- **Completed:** 2026-04-29T16:59:43Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments

- Added `internal/github.Service`, a real CLI-facing GitHub adapter that composes `DiscoverAuth`, `Client.Repository`, and `CheckRunnerManagementPermission` without persisting token material.
- Replaced the production fake-permitted CLI default with `gh.NewService` and ensured the real binary injects `github.OSCommandRunner{}` for `gh` and `git remote` subprocesses.
- Added regression coverage proving missing credentials fail closed without state writes and real public metadata blocks `runnerkit up` through the safety gate.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add a real GitHub service for the production default** - `80c026f` (test RED), `0d72ce3` (feat GREEN)
2. **Task 2: Wire production CLI defaults to OSCommandRunner and the real GitHub service** - `d5a3747` (test RED), `307ca51` (feat GREEN)
3. **Task 3: Add production-path regression tests for fail-closed auth and real metadata safety** - `009fe4b` (test)

**Plan metadata:** pending (docs: complete plan)

_Note: TDD tasks may have multiple commits (test → feat → refactor)_

## Files Created/Modified

- `internal/github/service.go` - Real GitHub service with in-memory credential cache, repository metadata reads, and runner-management permission checks.
- `internal/github/service_test.go` - `httptest` coverage for auth headers, metadata, registration-token redaction, and missing-auth remediation.
- `cmd/runnerkit/main.go` - Production binary now passes `github.OSCommandRunner{}` into CLI dependencies.
- `internal/cli/root.go` - Dependency normalization now defaults nil command runners to `gh.OSCommandRunner{}` and nil GitHub services to `gh.NewService`.
- `internal/cli/up.go` - Removed fake production `defaultGitHubService` and fake-private metadata behavior.
- `internal/cli/root_test.go` - Added explicit test fake and default-dependency assertions.
- `internal/cli/up_test.go` - Added default-path auth failure and real metadata public-repo safety regression tests.
- `internal/cli/up_integration_test.go` - Injects fake GitHub service only for deterministic save-state integration tests.

## Decisions Made

- Production defaults should fail closed even for dry-runs that need auth/metadata, because the phase goal requires auth and safety to be real before later runner flows depend on them.
- Test-only success paths should inject `fakePermittedGitHubService`; fakes should not live in production `up.go`.
- Service-level GitHub credentials may be cached in memory for repeated auth/metadata calls in a command, but token material must never be written to state or rendered raw.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Verification

- `go test ./... && go vet ./...` passed.
- `grep -R "defaultGitHubService" internal/cli/up.go` returned no matches.
- `grep -R "repo.Private = true" internal/cli/up.go` returned no matches.
- `grep -R "gh.NewService" internal/cli/root.go` returned a match.
- `grep -R "github.OSCommandRunner{}" cmd/runnerkit/main.go` returned a match.
- `go test ./internal/cli -run 'TestDefaultGitHubService(MissingCredentialsFailsClosed|UsesRealMetadataAndBlocksPublicRepo)'` passed.
- `go test ./internal/github -run TestService` passed.

## Next Phase Readiness

Phase 1 verification can be re-run. The production CLI default now uses real GitHub auth, metadata, runner-token permission checks, and OS subprocess execution, closing the previously reported GH-01 and safety-wiring gaps.

## Self-Check: PASSED

- Key files created exist on disk: `internal/github/service.go`, `internal/github/service_test.go`.
- Plan commits are present for `01-04`.
- No self-check failures were found.

---

_Phase: 01-cli-auth-state-and-safety-foundation_
_Completed: 2026-04-29_
