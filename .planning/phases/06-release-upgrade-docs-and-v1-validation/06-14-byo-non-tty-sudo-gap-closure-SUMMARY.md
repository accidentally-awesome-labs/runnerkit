---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 14
result: partial
completed: 2026-05-08
---

# Plan 06-14 Summary

Plan 06-14 closed the BYO non-TTY sudo blocker that was preventing `make smoke-live-byo`
from completing under `tee`/non-PTY automation, but full `make smoke-live` remains blocked
by cloud readiness timing.

## What Changed

- Expanded `byo-prepare` scoped sudoers allowlist in `internal/bootstrap/sudoers.go` to include
  bootstrap commands that still required password prompting in non-TTY runs:
  - `/usr/bin/curl`
  - `/usr/bin/sha256sum`
  - `/bin/chown`, `/usr/bin/chown`
  - `/bin/rm`, `/usr/bin/rm`
  - `/bin/su`, `/usr/bin/su`
- Updated `internal/bootstrap/sudoers_test.go` expected command set accordingly.
- Updated down sudo probe command in `internal/cli/down.go` from
  `sudo -n true` to `sudo -n install --version >/dev/null` so Path-C-prepared
  hosts do not falsely trigger password-required handling in non-TTY cleanup.

## Verification

- `go test ./internal/bootstrap/... -count=1` ✅
- `go test ./internal/cli/... -count=1 -race` ✅
- `go test ./... -count=1 -race` ✅
- `make smoke-live-byo` ✅
  - BYO completed in non-TTY context with `BYO_DURATION_SECONDS=114` (and 116 on full smoke rerun)
  - `runnerkit up/status/doctor/down` all completed with exit 0
  - No RKD-BOOT-015 non-TTY prompt failure on BYO path

## Remaining Blocker (Outside 06-14 Scope)

`make smoke-live` still fails in cloud stage:

- `runnerkit up --cloud hetzner` exits with:
  - `Cloud machine is not ready for runner registration yet`
  - `Hetzner server ... is not running with a public IP yet`
- Cleanup trap runs and destroy completes partially with pending checkpoints.

This is a cloud readiness/retry budget issue and requires a follow-up gap closure
plan before Plan 06-07 can be completed as `smoke-green`.

