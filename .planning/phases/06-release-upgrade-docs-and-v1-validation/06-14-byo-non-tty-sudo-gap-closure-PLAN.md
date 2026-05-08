---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 14
type: execute
wave: 1
depends_on: [13]
gap_closure: true
requirements: [REL-05]
status: proposed
---

# Plan 06-14: BYO Non-TTY Sudo Gap Closure

## Context

Plan 06-07 attempt-20 (2026-05-08) is SMOKE-RED. `make smoke-live` fails during
`smoke-live-byo` with non-TTY sudo errors (`RKD-BOOT-015`), preventing cloud smoke
and stopwatch baseline capture.

Observed evidence from `smoke-output.log`:

- `RunnerKit needs a sudo password but no TTY is available for prompting`
- `Remote stderr (unknown): sudo: a terminal is required ... sudo: a password is required`

## Objective

Close the BYO non-interactive sudo blocker so `make smoke-live` can complete
end-to-end in a tee/non-PTY automation context after one-time host preparation.

## Proposed Scope

1. Reproduce the failing BYO smoke path with deterministic test coverage.
2. Identify and fix the command(s) in bootstrap/install path that still require
   sudo elevation outside the byo-prepare allowlist in non-TTY execution.
3. Add regression test(s) proving `runnerkit up --yes --non-interactive` succeeds
   on a Path-C-prepared host without prompting.
4. Re-run Plan 06-07 smoke sequence after fix.

## Success Criteria

- `make smoke-live-byo` completes without `RKD-BOOT-015` in non-TTY context.
- No `sudo: a terminal is required` in BYO smoke output.
- Plan 06-07 can proceed to cloud smoke and stopwatch fill-in.

