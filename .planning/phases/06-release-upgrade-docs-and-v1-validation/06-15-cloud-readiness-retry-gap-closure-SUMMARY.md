---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 15
result: smoke-red
completed: 2026-05-08
resume_signal: "smoke-red cloud readiness ssh auth/convergence still failing"
---

# Plan 06-15 Summary

Plan 06-15 improved cloud readiness resilience and diagnostics, but did not
close the live cloud smoke gate. Full `make smoke-live-cloud` remains red.

## Implemented

- Increased Hetzner provider readiness budget in
  `internal/provider/hetzner/readiness.go` from ~30s to ~5m:
  - `readinessPollInterval`: 2s
  - `readinessMaxAttempts`: 150
- Added retry handling for transient SSH transport failures during cloud-init
  wait in `internal/cli/up.go` (`runCloudInitWaitWithRetry`).
- Increased default cloud-init wait budget from 5m to 10m:
  - `defaultCloudInitTimeout = 10 * time.Minute`
- Improved cloud-init readiness failure surfacing so remote stderr is included
  when available (instead of opaque `exit status 255` only).
- Added regression test:
  - `TestWaitCloudTargetReady_RetriesTransientCloudInitSSHError`
- Updated timeout budget expectations in:
  - `TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget`

## Verification

- `go test ./internal/cli/... -count=1 -race` ✅
- `go test ./... -count=1 -race` ✅
- `make smoke-live-cloud` ❌
  - still fails at cloud readiness with SSH-level `exit status 255`
  - failure now occurs after extended budget (~10m), indicating this is not
    a short retry-window issue.

## Outcome

- **BYO path remains green** after Plan 06-14.
- **Cloud path remains blocked** by persistent SSH/cloud-init readiness failure.
- Plan 06-07 baseline fill-in remains blocked pending cloud fix.

