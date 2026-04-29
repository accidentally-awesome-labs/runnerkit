---
phase: 02-byo-persistent-runner-happy-path
plan: "01"
subsystem: byo-ssh-preflight
tags: [go, cobra, ssh, preflight, state, safety]

requires:
  - phase: 01
    provides: CLI, GitHub auth, state, labels, redaction, and safety foundations
provides:
  - BYO SSH target parsing and CLI flags
  - SSH host-key trust model with persisted fingerprint fields
  - Remote executor seam and Linux/systemd preflight aggregation
affects: [runnerkit-up, remote, preflight, state]

tech-stack:
  added: []
  patterns:
    - Testable remote.Executor boundary separates CLI orchestration from SSH probes and commands
    - Host key mismatch fails closed before preflight or bootstrap commands run

key-files:
  created:
    - internal/remote/target.go
    - internal/remote/hostkey.go
    - internal/remote/executor.go
    - internal/preflight/checks.go
  modified:
    - internal/cli/up.go
    - internal/cli/root.go
    - internal/state/schema.go
    - internal/state/state_test.go

key-decisions:
  - "The Phase 2 CLI requires a BYO --host target in non-interactive/JSON flows and prompts only in interactive TTY flows."
  - "Unknown host keys can be accepted explicitly; mismatches return ssh_host_key_mismatch before preflight mutation."

requirements-completed: [MACH-01, CLI-03]

duration: 20 min
completed: 2026-04-29
---

# Phase 02 Plan 01: SSH Connectivity and Preflight Foundation Summary

**RunnerKit now has a BYO SSH target contract, host-key trust fields, and a remote preflight report before runner installation.**

## Accomplishments

- Added `remote.ParseTarget` for `user@host`, `user@host:port`, and `ssh://user@host:port` with display-safe output.
- Added host-key fingerprint helpers, decision states, and state fields for accepted host identity.
- Added `remote.Executor` plus preflight checks for SSH, host key, OS, arch, systemd, privilege, disk, tools, GitHub HTTPS, time, and runner conflicts.
- Added a default system-SSH executor using `ssh-keyscan`, local `ssh`, and shell probes so production CLI runs are not fake-only.
- Wired `runnerkit up` to require/collect a BYO target and render `ssh-preflight` in dry-run output.

## Task Commits

- **Implementation:** `be68bea` (`feat(02-01): implement BYO persistent runner happy path`)
- **Production SSH default:** `d4ada6d` (`feat(02-01): add production system SSH executor`)

## Deviations from Plan

- Combined Phase 2 implementation landed in one phase-wide code commit because the CLI orchestration, bootstrap, and registration paths were tightly coupled. The commit body references 02-01 through 02-04 for traceability.

## Issues Encountered

None.

## Verification

- `go test ./...` passed.
- `go vet ./...` passed.
- `go test ./internal/preflight -run Test` passed.
- Grep checks found `ssh.host_key`, `HostKeyFingerprint`, and `Pass --host user@host for BYO setup.`.

## Self-Check: PASSED

- Key files exist on disk.
- Plan commit is present for `02-01`.
- No self-check failures were found.
