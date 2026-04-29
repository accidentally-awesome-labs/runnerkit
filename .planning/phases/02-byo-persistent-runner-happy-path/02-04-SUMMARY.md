---
phase: 02-byo-persistent-runner-happy-path
plan: "04"
subsystem: safety-docs-smoke
tags: [go, docs, safety, smoke-test, redaction]

requires:
  - phase: 02-03
    provides: completed fake BYO setup orchestration and completion output
provides:
  - Stronger persistent-runner safety copy
  - End-to-end fake BYO happy path smoke test
  - BYO quickstart documentation and README link
affects: [docs, safety, cli-tests, redaction]

tech-stack:
  added: []
  patterns:
    - Public/fork safety runs before SSH, preflight, token, bootstrap, or state side effects
    - Fake smoke tests assert output/state stay free of registration-token and SSH-secret fixture values

key-files:
  created:
    - README.md
    - docs/byo-quickstart.md
    - internal/cli/docs_test.go
  modified:
    - internal/github/safety.go
    - internal/cli/up_test.go
    - internal/cli/up_integration_test.go

key-decisions:
  - "Persistent BYO is documented as suitable for trusted private repositories only."
  - "RunnerKit prints workflow snippets but does not edit or commit workflow YAML."

requirements-completed: [CLI-03, RUN-03, DOC-01, GH-05]

duration: 20 min
completed: 2026-04-29
---

# Phase 02 Plan 04: Safety, Smoke Test, and BYO Quickstart Summary

**Phase 2 now has user-facing safety boundaries, a fake end-to-end BYO smoke test, and concise quickstart documentation.**

## Accomplishments

- Updated safety copy with: `Persistent self-hosted runners are intended for trusted private repositories; public, fork-based, or otherwise untrusted workflows can execute code on your machine.`
- Added tests proving public repositories block before remote/probe/bootstrap side effects unless explicitly overridden.
- Added `TestUpBYOHappyPathSmoke` covering CLI, fake GitHub, fake remote, state save, labels, completion output, GitHub runner ID, and secret non-leakage.
- Added `README.md` and `docs/byo-quickstart.md` with prerequisites, command, safety warning, what RunnerKit changes, workflow snippet, completion summary, and troubleshooting.

## Task Commits

- **Implementation:** `be68bea` (`feat(02-01): implement BYO persistent runner happy path`; body covers `02-04`)

## Deviations from Plan

- Combined with the phase-wide code commit rather than separate per-task commits; docs and smoke tests are included in that commit.

## Issues Encountered

None.

## Verification

- `go test ./...` passed.
- `go vet ./...` passed.
- Grep checks found required safety copy, `TestUpBYOHappyPathSmoke`, docs no-YAML-edit copy, and the BYO setup command.

## Self-Check: PASSED

- Key files exist on disk.
- Plan commit is present for `02-04` through the implementation commit body.
- No self-check failures were found.
