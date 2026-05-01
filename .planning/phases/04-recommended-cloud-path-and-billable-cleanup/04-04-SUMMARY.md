---
phase: 04-recommended-cloud-path-and-billable-cleanup
plan: "04"
subsystem: cloud-cleanup-docs
tags: [cloud, hetzner, destroy, cleanup, billing, docs]
requires:
  - phase: 04-03-cloud-runner-lifecycle-and-provider-operations
    provides: Final cloud state with GitHub runner ID, provider inventory, cleanup IDs, and provider-aware operations output
provides:
  - runnerkit destroy command for cloud-managed Hetzner runners
  - Dry-run cloud destroy plan covering GitHub, remote runner, provider resources, and local state
  - Provider destroy and verify support for Hetzner server, SSH key, firewall, and primary IP resources
  - Partial cleanup checkpoints that preserve provider resource IDs until verification passes
  - README and docs/cloud-quickstart.md coverage for setup, labels, operations, destroy, cost caveats, and limitations
affects: [phase-05, phase-06, cloud, cleanup, docs, provider, billing]
tech-stack:
  added: []
  patterns:
    [
      plan-before-mutation destroy flow,
      provider verification before state removal,
      partial cleanup checkpoints,
    ]
key-files:
  created:
    - internal/cli/destroy.go
    - internal/cli/destroy_test.go
    - internal/ops/cloud_destroy.go
    - internal/provider/hetzner/destroy.go
    - internal/provider/hetzner/destroy_test.go
    - docs/cloud-quickstart.md
  modified:
    - README.md
    - internal/cli/root.go
    - internal/cli/docs_test.go
    - internal/provider/hetzner/client.go
    - internal/provider/hetzner/provision.go
    - internal/provider/hetzner/provision_test.go
key-decisions:
  - "Use runnerkit destroy only for cloud cleanup; runnerkit down remains the BYO cleanup command."
  - "Remove local cloud state only after GitHub cleanup and provider VerifyDestroyed prove resources are absent or non-billable."
  - "Partial destroy keeps local state with pending destroy checkpoints and provider resource IDs for resumable cleanup."
patterns-established:
  - "Cloud cleanup has a dry-run plan, typed confirmation or --yes, apply results, provider verification, and pending checkpoint model."
  - "Hetzner provider deletion tolerates already-absent resources but still verifies absence before success."
requirements-completed: [CLEAN-01, CLEAN-04, DOC-02, MACH-05, GH-03]
duration: 39 min
completed: 2026-05-01
---

# Phase 04 Plan 04: Cloud Destroy and Quickstart Summary

**Cloud destroy now plans, applies, verifies, and documents billable Hetzner cleanup before local state removal**

## Performance

- **Duration:** 39 min
- **Started:** 2026-05-01T00:52:00Z
- **Completed:** 2026-05-01T01:31:00Z
- **Tasks:** 3
- **Files modified:** 12

## Accomplishments

- Added `runnerkit destroy` for cloud-managed Hetzner runners, wired into the root command while leaving BYO cleanup on `runnerkit down`.
- Added `destroy --dry-run` human/JSON plan output with GitHub runner, remote runner, provider server, provider SSH key, provider firewall, provider primary IP, local state, and billing-impact warnings.
- Implemented typed interactive confirmation and no-TTY/JSON `--yes` gating before destructive cloud cleanup.
- Implemented cloud destroy apply flow that cleans remote registration when reachable, removes GitHub runner records, calls provider `Destroy`, calls provider `VerifyDestroyed`, and removes local state only after verification passes.
- Added partial cleanup behavior that keeps state and pending checkpoints when GitHub, remote, provider, or verification work remains.
- Implemented Hetzner provider deletion/verification for server, SSH key, firewall, and primary IP resources.
- Added README and `docs/cloud-quickstart.md` cloud setup, labels, status/logs/doctor, destroy, cost caveat, limitation, and smoke-test guidance.

## Task Commits

1. **Task 04-04-01: Add runnerkit destroy dry-run plan, confirmation UX, and JSON contract** - `a6c6bc9` (feat)
2. **Task 04-04-02: Apply cloud destroy, verify non-billable resources, and preserve partial cleanup state** - `a6c6bc9` (feat; same destroy-flow commit as Task 1 because dry-run and apply share one command/result model)
3. **Task 04-04-03: Document cloud quickstart, cost caveats, lifecycle reuse, labels, and destroy verification** - `a2e1595` (docs)

## Files Created/Modified

- `internal/cli/destroy.go` - Cloud destroy command, dry-run JSON/human plan, typed confirmation, apply flow, partial cleanup checkpoints, and result payloads.
- `internal/cli/destroy_test.go` - Dry-run no-mutation, BYO rejection, no-TTY confirmation, interactive confirmation, complete destroy, partial cleanup, GitHub failure, and redaction tests.
- `internal/ops/cloud_destroy.go` - Cloud destroy plan artifact model and ordering.
- `internal/provider/hetzner/destroy.go` - Hetzner delete and verify-absent/non-billable logic.
- `internal/provider/hetzner/destroy_test.go` - Provider deletion and verify sequencing, billable-resource detection, and already-absent behavior tests.
- `internal/provider/hetzner/client.go` - Added SSH key/firewall lookup methods used by verification.
- `internal/cli/root.go` - Registered `newDestroyCommand`.
- `README.md` / `docs/cloud-quickstart.md` - Recommended cloud path setup, operations, destroy, cost, and limitation docs.
- `internal/cli/docs_test.go` - Required cloud docs copy assertions while preserving BYO docs expectations.

## Decisions Made

- `destroy` rejects BYO state and points users to `runnerkit down` to keep billable cloud cleanup distinct from host cleanup.
- Provider deletion responses are not trusted alone; RunnerKit requires `VerifyDestroyed` to prove resources are absent or non-billable before local state removal.
- SSH-unreachable remote cleanup is partial but does not block GitHub/provider cleanup, so users can stop billing even when the host is inaccessible.
- Documentation explicitly says Phase 4 supports one recommended cloud path, uses persistent trusted-private runners, defers ephemeral mode, and does not edit workflow YAML.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added provider lookup methods for SSH key/firewall verification**

- **Found during:** Task 04-04-02 (provider verification)
- **Issue:** The client wrapper already supported server and primary IP lookup, but verifying SSH key and firewall absence also needs lookup methods.
- **Fix:** Added `GetSSHKey` and `GetFirewall` to the Hetzner client interface and API wrapper, plus fake-client test support.
- **Files modified:** `internal/provider/hetzner/client.go`, `internal/provider/hetzner/provision_test.go`, `internal/provider/hetzner/destroy.go`, `internal/provider/hetzner/destroy_test.go`
- **Verification:** `go test ./internal/provider/hetzner`, `go test ./...`
- **Committed in:** `a6c6bc9`

---

**Total deviations:** 1 auto-fixed (1 missing critical).
**Impact on plan:** The fix strengthens the planned verification requirement and stays within cloud destroy/provider scope.

## Issues Encountered

- The implementation combines Task 04-04-01 and Task 04-04-02 in a single command/provider commit because the dry-run plan, apply flow, partial result payloads, and provider verification share one command model.

## User Setup Required

None - no USER-SETUP.md was generated. Live cloud destroy requires the same provider credentials used for provisioning (`HCLOUD_TOKEN` or `HETZNER_CLOUD_TOKEN`) and may affect billable Hetzner resources.

## Next Phase Readiness

Phase 4 is ready for phase-level verification. Phase 5 can build ephemeral mode on top of a cloud path that now provisions, operates, and destroys persistent recommended cloud runners with documented safety/cost caveats.

## Self-Check: PASSED

- Key files exist and contain required patterns: `Use: "destroy"`, `VerifyDestroyed`, `provider_verification_pending`, `runnerkit up --repo owner/name --cloud hetzner`, and `runnerkit destroy --repo owner/name`.
- Task commits are present for `04-04`: `a6c6bc9`, `a2e1595`.
- Required verification passed: `go test ./internal/cli/... ./internal/ops/... ./internal/provider/... ./internal/state/...`, `go test ./...`, and required README/cloud docs grep checks.

---

_Phase: 04-recommended-cloud-path-and-billable-cleanup_
_Completed: 2026-05-01_
