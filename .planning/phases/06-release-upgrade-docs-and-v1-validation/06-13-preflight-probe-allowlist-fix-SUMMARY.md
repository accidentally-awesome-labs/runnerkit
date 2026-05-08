---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 13
subsystem: preflight
tags: [byo, sudo, preflight, allowlist, bugfix]
requirements-completed: [REL-05]
completed: 2026-05-08
---

# Phase 06 Plan 13: Preflight Probe Allowlist Fix Summary

Bug 31 is closed by rebinding the preflight sudo probe to the `byo-prepare` scoped sudoers contract. The probe in `internal/preflight/checks.go` now runs `sudo -n install --version >/dev/null` (allowlisted) instead of `sudo -n true` (not allowlisted), so Path-C-prepared hosts no longer incorrectly fall into Path B's TTY sudo prompt in non-interactive contexts.

## Root Cause

- `CheckPrivilege` used `Script: "sudo -n true"` for `probe_sudo_n`.
- `true` is not in `internal/bootstrap/sudoers.go::RenderSudoersEntry` scoped allowlist.
- On already prepared hosts, probe returned password-required and misclassified privilege state.

## Fix Approach

- Applied Option A from the plan/GAP: changed probe script literal to `sudo -n install --version >/dev/null`.
- Kept `Command.ID` as `probe_sudo_n` to preserve existing executor fakes and upstream wiring.
- Preserved Bug 7 classifier behavior (stderr marker matching and warning/failure mapping unchanged).

## Files Modified

- `internal/preflight/checks.go`
- `internal/preflight/checks_test.go`
- `internal/preflight/checks_bugfix_test.go`

## Tests Added and Preserved

- Added `TestCheckPrivilege_AllowsScopedSudoers` in `internal/preflight/checks_test.go`.
  - Verifies Path-C prepared behavior classifies as `SeverityPass`.
  - Binds source to new probe literal and forbids old literal.
- Preserved/kept green:
  - Bug 7 tests in `checks_bugfix_test.go`
  - Bug 8 network probe guard test
  - Existing preflight contracts (`Passwordless`, `PasswordRequired`, `NotInSudoers`, `SudoMissing`, stable IDs, arch/unknown-linux checks)
  - Path B contract checks in `internal/cli/up_test.go`

## Commits Landed (TDD Cadence)

1. `test(06-13): bug 31 — add TestCheckPrivilege_AllowsScopedSudoers binding probe Script to byo-prepare allowlist`
2. `fix(06-13): bug 31 — preflight probe uses sudo -n install --version >/dev/null inside byo-prepare scoped allowlist`

## Invariants Preserved

- Plan 06-05 / 06-06 / 06-08..06-12 behavior contracts remain intact.
- Out-of-scope surfaces untouched:
  - `internal/bootstrap/sudoers.go` allowlist content
  - `internal/cli/up.go` Path B prompt logic
  - `internal/cli/byo_prepare.go` informational verify probe
  - `scripts/smoke/byo-permission.sh`

## Verification

- `go test ./internal/preflight/... -count=1 -race` passes.
- `go test ./internal/cli/ -run 'TestUp_SudoPasswordPrompt|TestUpRequiresHostFlag|TestUp_PreflightWiring' -count=1 -race` passes.
- `go test ./... -count=1 -race` passes.
- `go vet ./...` passes.
- `gofmt -l internal/preflight/` is empty.

## Resume Signal

Plan 06-13 is ready for Plan 06-07 attempt-20 maintainer smoke rerun (`make smoke-live` under non-TTY/`tee`) to confirm BYO no longer falls through to Path B and to capture `BYO_DURATION_SECONDS=NNN` for release gate closure.
