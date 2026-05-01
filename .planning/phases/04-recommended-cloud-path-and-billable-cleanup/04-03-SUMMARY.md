---
phase: 04-recommended-cloud-path-and-billable-cleanup
plan: "03"
subsystem: cloud-operations
tags: [cloud, hetzner, bootstrap, status, logs, doctor, state]
requires:
  - phase: 04-02-hetzner-provisioning-and-readiness
    provides: Hetzner provisioned machine targets, cloud inventory, pending checkpoints, and readiness/preflight gates
provides:
  - Cloud runner installation through the same bootstrap.Apply, registration-token, service, and online-verification path as BYO
  - Final cloud runner state with GitHub runner ID, cloud-ssh target, provider inventory, cleanup IDs, managed paths, and no pending checkpoints
  - Provider facts in status human/JSON output with sources.provider parity
  - Provider drift/missing-resource findings in doctor and provider metadata in logs
  - Read-only operation tests proving status, doctor, and logs do not mutate provider or GitHub runner resources
affects: [phase-04-04, cloud, provider, state, status, logs, doctor, cleanup]
tech-stack:
  added: []
  patterns:
    [
      shared bootstrap lifecycle for cloud/BYO,
      ProviderFact operational model,
      cloud final-state replacement,
    ]
key-files:
  created: []
  modified:
    - internal/cli/up.go
    - internal/cli/up_cloud_test.go
    - internal/cli/root_test.go
    - internal/cli/status.go
    - internal/cli/status_test.go
    - internal/cli/doctor_test.go
    - internal/cli/logs.go
    - internal/cli/logs_test.go
    - internal/ops/status.go
    - internal/ops/doctor.go
    - internal/testsupport/state.go
key-decisions:
  - "Cloud setup now mints the GitHub registration token only after provider readiness, SSH/cloud-init readiness, and BYO preflight pass."
  - "Final cloud success replaces pending checkpoints with a complete cloud-ssh state and deterministic provider_resource_ids."
  - "Operational commands use provider Describe for cloud facts and never call provider mutation methods."
patterns-established:
  - "Cloud runners reuse buildBootstrapOptions, bootstrap.Apply, waitForRunnerOnline, labels.Build, and the existing service naming convention."
  - "ProviderFact is the status/doctor/logs boundary for cloud provider visibility and drift warnings."
requirements-completed: [MACH-04, MACH-05, REL-01, REL-02, REL-03]
duration: 27 min
completed: 2026-05-01
---

# Phase 04 Plan 03: Cloud Runner Lifecycle and Provider Operations Summary

**Provisioned Hetzner machines now become RunnerKit runners through the shared BYO bootstrap path, with final cloud state and provider-aware status/logs/doctor output**

## Performance

- **Duration:** 27 min
- **Started:** 2026-05-01T00:24:38Z
- **Completed:** 2026-05-01T00:51:00Z
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments

- Wired the successful cloud path through `bootstrap.Apply`, `CreateRegistrationToken`, and `waitForRunnerOnline` after provider readiness, cloud-init, SSH host-key capture, and BYO preflight checks pass.
- Saved final cloud state with `cloud-ssh` machine identity, host-key fingerprint, install/work/service paths, GitHub runner ID, Hetzner inventory, billable resource IDs, and no lingering `cloud_provision_pending` or `cloud_readiness_pending` checkpoints.
- Added provider facts to `runnerkit status` human output and `sources.provider` JSON output.
- Added provider drift/missing-resource health warnings and doctor findings with `runnerkit destroy --repo <repo> --dry-run` remediation.
- Added cloud provider metadata to logs output while keeping log collection bounded and read-only.

## Task Commits

1. **Task 04-03-01: Reuse BYO bootstrap for cloud up after readiness and preserve token timing** - `4737ebc` (feat)
2. **Task 04-03-02: Save final cloud runner state with full provider inventory and cleanup IDs** - `4737ebc` (feat; same cloud lifecycle commit as Task 1 because the shared bootstrap and final-state save are one success path)
3. **Task 04-03-03: Add provider facts to status, doctor, and logs without mutations** - `ac6f878` (feat)

## Files Created/Modified

- `internal/cli/up.go` - Cloud success now applies bootstrap, waits for online runner, saves final cloud state, and renders destroy-ready cloud completion JSON/human output.
- `internal/cli/up_cloud_test.go` - Verifies readiness before registration, bootstrap command execution, final cloud state, cleanup IDs, and completion copy.
- `internal/cli/root_test.go` - Test GitHub fake now returns runner names/labels for the requested repository.
- `internal/ops/status.go` - Adds `ProviderFact`, provider drift/missing-resource reasons, and health classification.
- `internal/cli/status.go` - Collects provider facts via `Describe`, renders Provider source line, and includes `sources.provider` JSON keys.
- `internal/ops/doctor.go` / `internal/cli/doctor_test.go` - Adds provider drift/missing-resource findings and dry-run destroy remediation.
- `internal/cli/logs.go` / `internal/cli/logs_test.go` - Adds cloud provider and billable-resource metadata to logs output.
- `internal/cli/status_test.go` - Verifies provider JSON keys and cloud empty-state copy.
- `internal/testsupport/state.go` - Adds reusable cloud repository state fixture.

## Decisions Made

- Kept one cloud success path instead of building a parallel installer: provider readiness hands a `remote.Target` to the same bootstrap and online-verification primitives used by BYO.
- Replaced pending cloud state with final cloud state only after GitHub confirms the runner is online with the expected labels.
- Treated provider drift as a warning health condition that does not override harder GitHub/SSH/systemd failures.
- Kept logs provider-aware without provider API calls; status/doctor use `Describe` only.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Test GitHub fake needed repo-aware runner labels**

- **Found during:** Task 04-03-01 (shared cloud bootstrap)
- **Issue:** Cloud success for `owner/name` waited for `runnerkit-owner-name-local`, but the test fake always returned `runnerkit-owner-repo-local`, causing online verification to time out.
- **Fix:** Updated the fake GitHub service to build the returned runner name and labels from the requested repository.
- **Files modified:** `internal/cli/root_test.go`
- **Verification:** `go test ./internal/cli/... ./internal/bootstrap/...`; `go test ./...`
- **Committed in:** `4737ebc`

---

**Total deviations:** 1 auto-fixed (1 blocking).
**Impact on plan:** The fix was test-support only and made the new cloud online-verification path accurately exercise repository-specific labels.

## Issues Encountered

- The configured child executor became unavailable due an external usage limit during 04-03, so this plan was executed inline as the runtime fallback permits.
- Task 04-03-01 and 04-03-02 landed in a shared implementation commit because cloud bootstrap and final-state replacement are coupled in the same success path.

## User Setup Required

None - no USER-SETUP.md was generated. Live cloud provisioning still requires valid Hetzner credentials and SSH key material as already enforced by `runnerkit up --cloud hetzner`.

## Next Phase Readiness

Plan 04-04 can now implement `runnerkit destroy` against complete cloud state: GitHub runner ID, managed paths, provider resource IDs, and provider inventory are available, while status/logs/doctor expose provider facts needed to review cleanup risk.

## Self-Check: PASSED

- Key files exist and contain required patterns: `buildCloudRepositoryState`, `sources.provider`, `Provider: Hetzner fsn1 cpx22 ubuntu-24.04`, `runnerkit destroy --repo owner/name`, and cloud empty-state copy.
- Task commits are present for `04-03`: `4737ebc`, `ac6f878`.
- Required verification passed: `go test ./internal/cli/... ./internal/bootstrap/...`, `go test ./internal/ops/... ./internal/cli/... ./internal/state/...`, and `go test ./...`.

---

_Phase: 04-recommended-cloud-path-and-billable-cleanup_
_Completed: 2026-05-01_
