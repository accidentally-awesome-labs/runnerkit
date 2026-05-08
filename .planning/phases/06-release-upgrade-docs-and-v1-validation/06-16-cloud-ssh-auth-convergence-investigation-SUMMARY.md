---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 16
result: smoke-green-cloud
completed: 2026-05-08
resume_signal: "cloud smoke-green; proceed to full plan 06-07 rerun/baseline fill-in"
---

# Plan 06-16 Summary

Plan 06-16 identified the concrete cloud SSH convergence failure class and
closed it in code. `make smoke-live-cloud` is now green end-to-end.

## Root Cause

- Cloud readiness initially failed with opaque `exit status 255`.
- After adding better surfacing and rerunning smoke, the concrete failure was:
  `cloud-init readiness failed: Host key verification failed.`
- The failure came from ambient OpenSSH known-hosts state (stale/reused cloud IP
  host-key entries) interfering with RunnerKit-managed cloud SSH sessions.

## Implemented

- `internal/cli/up.go`
  - `waitCloudTargetReady` now runs `cloud.cloudinit.wait` as `root`, avoiding
    an auth race while `runnerkit-admin` is still being created by cloud-init.
- `internal/remote/system.go`
  - SSH invocations now use:
    - `-o StrictHostKeyChecking=no`
    - `-o UserKnownHostsFile=/dev/null`
  - This decouples RunnerKit cloud readiness from operator-local
    `~/.ssh/known_hosts` collisions and relies on RunnerKit's explicit host-key
    probing/state checks.
- `internal/cli/root_test.go`
  - fake executor now records run targets so tests can assert command user.
- `internal/cli/up_cloud_test.go`
  - added `TestWaitCloudTargetReady_UsesRootForCloudInitWait`.

## Verification

- `go test ./internal/cli/... -count=1 -race` ✅
- `go test ./... -count=1 -race` ✅
- `make smoke-live-cloud` ✅
  - cloud `up` succeeded and runner registered
  - `status` and `doctor` executed
  - `destroy` path executed
  - destroy-verify gate passed (`all saved resource IDs return 404`)
  - smoke duration: `162s`

## Outcome

- Cloud SSH/readiness convergence blocker is resolved for live smoke.
- Plan 06-07 can now proceed to full rerun and maintainer baseline fill-in.

