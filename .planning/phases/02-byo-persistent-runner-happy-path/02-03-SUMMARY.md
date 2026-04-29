---
phase: 02-byo-persistent-runner-happy-path
plan: "03"
subsystem: github-runner-registration
tags: [go, github-rest, runners, labels, state]

requires:
  - phase: 02-02
    provides: bootstrap executor and service verification path
provides:
  - Repository runner inventory API wrapper
  - Duplicate-name protection before registration
  - Just-in-time registration token to bootstrap orchestration
  - Online runner polling and completion summary
affects: [github, cli, labels, state]

tech-stack:
  added: []
  patterns:
    - Fresh registration token requested only after preflight, duplicate check, and install confirmation
    - Successful state is saved only after GitHub runner is online with all RunnerKit labels

key-files:
  created:
    - internal/github/runners.go
    - internal/github/runners_test.go
  modified:
    - internal/github/service.go
    - internal/cli/up.go
    - internal/cli/up_integration_test.go
    - internal/state/schema.go

key-decisions:
  - "Phase 2 blocks duplicate online/offline runner names with runner_name_conflict instead of auto-deleting stale runners."
  - "Completion output remains snippet-only and does not edit workflow YAML."

requirements-completed: [GH-02, GH-04, GH-05, CLI-04, RUN-01]

duration: 30 min
completed: 2026-04-29
---

# Phase 02 Plan 03: GitHub Registration and Completion Summary

**RunnerKit now registers and verifies a repository-scoped persistent runner, persists secret-free BYO state, and prints copy-paste workflow guidance.**

## Accomplishments

- Added `ListRunners`, `DeleteRunner`, `Runner`, `RunnerManager`, and `FindRunnerByName` for repository runner inventory.
- Added duplicate-name blocking before fresh registration token creation.
- Added online polling with `runnerOnlineWithLabels` using `2 * time.Second` interval and `60 * time.Second` timeout defaults.
- Saves BYO state with `kind: byo-ssh`, `mode: persistent`, `github_runner_id`, machine target, service name, install path, work dir, labels, and host fingerprint.
- Human and JSON completion output includes `BYO runner ready`, runner metadata, labels, state path, and the exact `runs-on` snippet.

## Task Commits

- **Implementation:** `be68bea` (`feat(02-01): implement BYO persistent runner happy path`; body covers `02-03`)

## Deviations from Plan

- Combined with the phase-wide code commit rather than separate per-task commits; tests cover duplicate blocking, dry-run no-apply, successful human/JSON completion, and workflow file non-mutation.

## Issues Encountered

None.

## Verification

- `go test ./...` passed.
- `go vet ./...` passed.
- Grep checks found `runner_name_conflict`, `BYO runner ready`, `runner_online_timeout`, `github_runner_id`, and `runs-on: [self-hosted, runnerkit`.

## Self-Check: PASSED

- Key files exist on disk.
- Plan commit is present for `02-03` through the implementation commit body.
- No self-check failures were found.
