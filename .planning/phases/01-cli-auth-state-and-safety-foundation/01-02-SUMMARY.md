---
phase: 01-cli-auth-state-and-safety-foundation
plan: "02"
subsystem: github
tags: [go, github-rest, gh-cli, auth, safety, redaction]
requires:
  - phase: 01-cli-auth-state-and-safety-foundation
    provides: CLI skeleton, renderer, exit codes, and redaction foundation from Plan 01-01
provides:
  - GitHub repository target parsing from --repo and git remote URLs
  - gh-first auth discovery and fine-grained token remediation copy
  - GitHub REST client with runner registration/removal token adapters
  - Public/fork repository safety decisions integrated into runnerkit up
affects: [phase-1, cli, github-auth, safety, state, runner-registration]
tech-stack:
  added: []
  patterns:
    - Injectable command runners and GitHub services for deterministic CLI tests
    - GitHub REST wrappers with explicit API headers
    - Permission checks through ephemeral runner token creation and immediate redaction registration
key-files:
  created:
    - internal/github/types.go
    - internal/github/remote.go
    - internal/github/auth.go
    - internal/github/client.go
    - internal/github/tokens.go
    - internal/github/safety.go
    - internal/testsupport/github.go
  modified:
    - internal/cli/root.go
    - internal/cli/up.go
    - internal/cli/up_test.go
    - internal/github/github_test.go
key-decisions:
  - "Keep default runnerkit up dry-runs deterministic with a fake-permitted GitHub service while exposing real GitHub auth/client/token adapters for injection."
  - "Use runner registration token creation as the repository runner-management permission check and register returned tokens with the redactor immediately."
  - "Block public repositories by default and require --allow-public-repo-risk plus explicit acknowledgement paths before future persistent setup can proceed."
patterns-established:
  - "Repository resolution returns typed github.Repo values and records whether confirmation is required before auth."
  - "GitHub failures map to typed CLI exit codes and JSON error codes."
  - "Safety decisions carry stable codes (ok, public_repo_risk, fork_risk) for human and JSON output."
requirements-completed: [CLI-02, GH-01, STATE-02]
duration: 5 min
completed: 2026-04-29
---

# Phase 1 Plan 02: GitHub Repo, Auth, Token, and Safety Foundation Summary

**Repository-scoped GitHub auth foundation with gh-first discovery, runner-token permission fixtures, redaction, and public/fork safety gates in `runnerkit up`**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-29T02:20:13Z
- **Completed:** 2026-04-29T02:25:09Z
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments

- Added `internal/github` contracts and parsers for explicit `owner/name` targets plus HTTPS, SSH URL, and scp-style GitHub remotes.
- Wired `runnerkit up` so auto-detected repositories are displayed as `Choose repository: owner/name` and confirmed before GitHub auth or state-adjacent preview work runs.
- Implemented gh-first auth discovery, exact fine-grained token remediation, REST API headers, runner registration/removal token adapters, and token redaction registration.
- Added public/fork safety decisions and integrated public repository blocking/override warning behavior into human and JSON `up` output.

## Task Commits

Each task was committed atomically using the TDD red/green flow:

1. **Task 1 RED: repository targeting tests** - `ccb3c18` (test)
2. **Task 1 GREEN: repository target resolution** - `57893b1` (feat)
3. **Task 2 RED: auth/client/token tests** - `2be4df1` (test)
4. **Task 2 GREEN: auth and token adapters** - `412df71` (feat)
5. **Task 3 RED: safety gate tests** - `bb38f82` (test)
6. **Task 3 GREEN: safety gate integration** - `e975df9` (feat)

## Files Created/Modified

- `internal/github/types.go` - Shared GitHub repo, auth source, and permission status contracts.
- `internal/github/remote.go` - `ParseRepo`, `ParseRemote`, and `ResolveTarget` with GitHub-only owner/name validation.
- `internal/github/auth.go` - Injectable command runner, `gh auth token` discovery, in-memory token handling, and fine-grained token remediation copy.
- `internal/github/client.go` - GitHub REST client wrapper with `Accept` and `X-GitHub-Api-Version` headers.
- `internal/github/tokens.go` - Runner registration/removal token provider interface and permission-check helper.
- `internal/github/safety.go` - `EvaluateSafety` decisions for private, public, and fork-risk repositories.
- `internal/testsupport/github.go` - Reusable fake GitHub service for deterministic tests.
- `internal/cli/root.go` - Dependency injection expanded for command runners and GitHub service fakes.
- `internal/cli/up.go` - Repo confirmation, auth verification, metadata loading, safety checks, JSON/human output integration.
- `internal/cli/up_test.go` - CLI tests for confirmation ordering, permission denial JSON, and public repo safety gate behavior.
- `internal/github/github_test.go` - Parser, auth, REST header, runner token redaction, permission denial, and safety tests.

## Decisions Made

- Used injectable command runners and GitHub services so tests and Phase 1 dry-runs never require real `gh` auth or network access by default.
- Kept durable GitHub tokens in memory only and registered them with the redactor immediately when discovered from `gh` or `RUNNERKIT_GITHUB_TOKEN`.
- Treated runner registration token creation as the permission check because it exercises the same runner-management capability later phases need.
- Chose fail-closed public repository behavior with stable `public_repo_risk` JSON/human warnings and the explicit `--allow-public-repo-risk` override path.

## Deviations from Plan

None - plan executed as written.

## Known Stubs

- `internal/cli/up.go` - `defaultGitHubService` is an intentional fake-permitted Phase 1 adapter so `go run ./cmd/runnerkit up --dry-run --repo owner/name --yes --json` succeeds deterministically without real network calls. Real GitHub auth/client/token adapters exist in `internal/github` and are injectable for later wiring.

## Issues Encountered

None.

## Verification

- `go test ./...` passed.
- `go run ./cmd/runnerkit up --dry-run --repo owner/name --yes --json` emitted JSON only with `repo: "owner/name"` and `runner_installed: false`.
- Permission-denied CLI fixture test verifies exit code `3` and JSON error code `github_permission_denied`.
- Public repository fixture test verifies exit code `4` and `WARNING: Public repository risk` in human mode.
- Runner token fixture rendering test and testdata grep verify raw fixture token text does not appear in output fixtures.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for Plan 01-03 to persist non-secret repo/auth/safety state using the repo target, auth source reference, and safety decision contracts from this plan.
- Later live GitHub execution can inject the real `internal/github` auth/client/token adapters without changing the CLI output contracts.

---
*Phase: 01-cli-auth-state-and-safety-foundation*
*Completed: 2026-04-29*

## Self-Check: PASSED

- Created files exist: `types.go`, `remote.go`, `auth.go`, `client.go`, `tokens.go`, `safety.go`, and `internal/testsupport/github.go` were verified on disk.
- Task commits exist: `ccb3c18`, `57893b1`, `2be4df1`, `412df71`, `bb38f82`, and `e975df9` were verified in git history.
