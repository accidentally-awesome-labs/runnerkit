---
phase: 02-byo-persistent-runner-happy-path
plan: "02"
subsystem: byo-bootstrap
tags: [go, bootstrap, systemd, github-actions-runner]

requires:
  - phase: 02-01
    provides: remote executor seam and successful preflight report
provides:
  - Pinned GitHub Actions runner package metadata
  - Idempotent bootstrap script rendering
  - Non-root runner service install through remote.Executor
affects: [bootstrap, workflow-plan, runnerkit-up]

tech-stack:
  added: []
  patterns:
    - Registration tokens are passed through remote.Command.Env and RedactArgs, never embedded in scripts
    - Service installation always uses runnerkit-runner rather than root

key-files:
  created:
    - internal/bootstrap/package.go
    - internal/bootstrap/script.go
    - internal/bootstrap/install.go
  modified:
    - internal/workflow/plan.go
    - internal/cli/up.go

key-decisions:
  - "Runner package metadata is pinned to actions runner 2.334.0 for Linux x64 and arm64."
  - "The dedicated service user is runnerkit-runner and service install uses sudo ./svc.sh install runnerkit-runner."

requirements-completed: [MACH-02, CLI-03]

duration: 25 min
completed: 2026-04-29
---

# Phase 02 Plan 02: BYO Bootstrap Installer Summary

**RunnerKit can render and apply a deterministic BYO Linux bootstrap plan for a non-root persistent runner service.**

## Accomplishments

- Added pinned package metadata and SHA-256 checksums for `actions-runner-linux-x64-2.334.0.tar.gz` and `actions-runner-linux-arm64-2.334.0.tar.gz`.
- Added dependency-fix, install, configure, and service script rendering with checksum verification.
- Added `bootstrap.Apply` command ordering for `fix_dependencies`, `create_runner_user`, `download_runner`, `configure_runner`, `install_service`, and `verify_service`.
- Wired dry-run output to show `bootstrap-plan` and non-dry-run output to require `--yes` or interactive confirmation before remote mutation.

## Task Commits

- **Implementation:** `be68bea` (`feat(02-01): implement BYO persistent runner happy path`; body covers `02-02`)

## Deviations from Plan

- Combined with the phase-wide code commit rather than separate per-task commits; bootstrap tests preserve command-order and token-redaction assertions.

## Issues Encountered

None.

## Verification

- `go test ./...` passed.
- `go vet ./...` passed.
- Grep checks found `2.334.0`, `sudo ./svc.sh install runnerkit-runner`, and `registration-token-secret-12345` only in bootstrap tests.

## Self-Check: PASSED

- Key files exist on disk.
- Plan commit is present for `02-02` through the implementation commit body.
- No self-check failures were found.
