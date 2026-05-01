---
phase: 04-recommended-cloud-path-and-billable-cleanup
plan: "02"
subsystem: cloud-provisioning
tags: [cloud, hetzner, hcloud-go, ssh, state, readiness]
requires:
  - phase: 04-01-provider-interface-cloud-intent-and-provisioning-plan
    provides: Provider contracts, Hetzner defaults, explicit cloud intent, and plan-before-mutation output
provides:
  - Hetzner hcloud-go adapter for validating profile lookups and creating SSH key, firewall, and server resources
  - Partial-provision error handling that preserves created provider resource IDs
  - Cloud inventory and cost profile state fields for Hetzner server/access resources
  - Immediate cloud_provision_pending and cloud_readiness_pending state checkpoints for resumable destroy
  - Provider-only and CLI-level readiness gates before GitHub registration token creation
affects: [phase-04-03, phase-04-04, cloud, provider, state, readiness, cleanup]
tech-stack:
  added: [github.com/hetznercloud/hcloud-go v1.59.2]
  patterns: [provider client wrapper, partial provision checkpoints, provider-ref cloud inventory, split provider/ssh readiness]
key-files:
  created:
    - internal/provider/hetzner/client.go
    - internal/provider/hetzner/provision.go
    - internal/provider/hetzner/provision_test.go
    - internal/provider/hetzner/readiness.go
    - internal/provider/hetzner/readiness_test.go
  modified:
    - go.mod
    - go.sum
    - internal/cli/root.go
    - internal/cli/up.go
    - internal/cli/up_cloud_test.go
    - internal/provider/profile.go
    - internal/state/schema.go
    - internal/state/state_test.go
key-decisions:
  - "Use hcloud-go v1.59.2, not /v2, to stay compatible with the module's Go 1.22 target."
  - "Store cloud inventory under ProviderRef.Cloud while preserving existing ProviderRef kind/ids/region compatibility."
  - "Persist cloud_provision_pending immediately after billable resource creation and cloud_readiness_pending if provider/SSH/cloud-init/preflight readiness fails."
  - "Keep provider-only readiness in internal/provider/hetzner and SSH/cloud-init/BYO preflight readiness in the CLI layer."
patterns-established:
  - "Hetzner adapter wraps only the SDK operations RunnerKit needs, with fake-client tests and no live provider dependency."
  - "ProvisionError carries partial Machine and CreatedResourceIDs so CLI can save destroy-ready state after failures."
  - "Cloud readiness gates GitHub registration-token creation behind provider running/public IP, SSH host key, cloud-init, and BYO preflight."
requirements-completed: [MACH-03, MACH-05]
duration: 15 min
completed: 2026-05-01
---

# Phase 04 Plan 02: Hetzner Provisioning and Readiness Summary

**Hetzner VM, SSH-key, firewall, cloud inventory, pending cleanup checkpoints, and readiness gates before GitHub runner registration**

## Performance

- **Duration:** 15 min
- **Started:** 2026-05-01T00:06:05Z
- **Completed:** 2026-05-01T00:20:53Z
- **Tasks:** 3
- **Files modified:** 13

## Accomplishments

- Added a Hetzner hcloud-go adapter with lookup validation for `fsn1`, `cpx22`, and `ubuntu-24.04`, plus deterministic SSH key, firewall, and server creation inputs.
- Added fake-client coverage for resource creation order, tags, firewall CIDR, cloud-init user data, no live `HCLOUD_TOKEN` dependency, and partial `ProvisionError` resource preservation.
- Extended state with `CloudInventory` and `CostProfileRef`, including server, SSH key, firewall, primary IP, tags, cost profile, and cleanup resource IDs.
- Updated `runnerkit up --cloud hetzner` to persist `cloud_provision_pending` after billable resources exist, preserve partial state on provisioning failure, and record `cloud_readiness_pending` on readiness failure.
- Added provider-only readiness and CLI readiness gates for create action completion, server running/public IP, SSH host key, cloud-init completion, and existing BYO preflight before registration-token creation.

## Task Commits

1. **Task 04-02-01: Implement Hetzner SDK adapter create flow with deterministic resources and no live-test dependency** - `2a0b658` (feat)
2. **Task 04-02-02: Persist cloud inventory and pending checkpoints as soon as billable resources exist** - `ddb617b` (feat)
3. **Task 04-02-03: Gate cloud readiness before GitHub registration and shared bootstrap** - `faa86a0` (feat)

## Files Created/Modified

- `go.mod` / `go.sum` - Added `github.com/hetznercloud/hcloud-go v1.59.2` and transitive checksums.
- `internal/provider/hetzner/client.go` - Small hcloud client wrapper for lookup, create, wait, describe, and delete operations needed by Phase 4.
- `internal/provider/hetzner/provision.go` - Hetzner provider implementation for Validate/Plan/Provision and resource inventory normalization.
- `internal/provider/hetzner/provision_test.go` - Fake-client resource creation, tags, CIDR, cloud-init, no-live-token, and partial error coverage.
- `internal/provider/hetzner/readiness.go` - Provider-only wait for create action, running server status, and public IP assignment.
- `internal/provider/hetzner/readiness_test.go` - Provider readiness success/failure coverage.
- `internal/state/schema.go` / `internal/state/state_test.go` - Cloud inventory, cost profile, provider resource IDs, and no-secret serialization tests.
- `internal/cli/root.go` / `internal/provider/profile.go` - Default provider registry now points at the real Hetzner adapter without a parent/subpackage import cycle.
- `internal/cli/up.go` / `internal/cli/up_cloud_test.go` - Pending-state saves, readiness failure handling, cloud-init/preflight gate, public key discovery, and token-timing tests.

## Decisions Made

- Used hcloud-go v1.59.2 because it is the compatible v1 SDK for Go 1.22; `/v2` remains intentionally unused.
- Stored Hetzner inventory in `ProviderRef.Cloud` and mirrored resource IDs in both compatibility `IDs` and stable `ResourceIDs` maps.
- Split readiness responsibilities: Hetzner provider waits for provider facts only; CLI owns SSH host key, cloud-init, and BYO preflight checks.
- Cloud setup saves pending state before readiness/registration so `runnerkit destroy --repo owner/name` has resource IDs if later steps fail.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed provider package import cycle and plan-only default provider**
- **Found during:** Task 04-02-01 (Hetzner adapter implementation)
- **Issue:** The Phase 04-01 `provider.NewHetznerPlanProvider` lived in the parent provider package and imported the Hetzner credentials subpackage. Implementing `internal/provider/hetzner` against parent provider types would create an import cycle, and leaving the plan-only provider as the production default would block real provisioning.
- **Fix:** Moved production default registration to `hetzner.NewProvider(nil)`, removed the parent plan-provider implementation, and adjusted tests to inject fakes or the real adapter with fake dependencies.
- **Files modified:** `internal/provider/profile.go`, `internal/cli/root.go`, `internal/cli/up_cloud_test.go`
- **Verification:** `go test ./internal/provider/... ./internal/cli/... ./internal/state/...`, `go test ./...`
- **Committed in:** `2a0b658` (Task 04-02-01)

**2. [Rule 2 - Missing Critical] Populated cloud public SSH key input for real SSH-key creation**
- **Found during:** Task 04-02-03 final self-check
- **Issue:** The Hetzner adapter correctly requires `ProvisionInput.PublicKey` before creating an SSH key, but the CLI cloud input builder did not populate it, so live provisioning would fail before resource creation.
- **Fix:** Added public key discovery from `--ssh-key <path>` via `<path>.pub`, then common default public keys, without persisting key material in state.
- **Files modified:** `internal/cli/up.go`, `internal/cli/up_cloud_test.go`
- **Verification:** `TestBuildCloudProvisionInputReadsPublicKeyFromSSHKeyFlag`; full plan verification and `go test ./...`
- **Committed in:** `faa86a0` (Task 04-02-03)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical).
**Impact on plan:** Both fixes were necessary for a real, testable Hetzner provisioning path. Scope remained within the planned cloud provider/readiness work.

## Issues Encountered

- `go get github.com/hetznercloud/hcloud-go@v1` required adding transitive hcloud SDK dependencies via `go mod tidy`; verification confirms no `/v2` dependency.
- The cloud path currently stops after readiness and registration-token timing checks; full shared bootstrap installation and final cloud runner state are intentionally completed by Plan 04-03.

## User Setup Required

None - no USER-SETUP.md was generated. Users will still need a Hetzner token and SSH public key for live cloud provisioning, as covered by CLI remediation and upcoming cloud quickstart work.

## Next Phase Readiness

Plan 04-03 can reuse the ready `provider.Machine.Target`, pending cloud inventory, and readiness/preflight results to complete shared BYO bootstrap installation, save final GitHub runner state, and expose provider facts in status/logs/doctor.

## Self-Check: PASSED

- Key files exist: `internal/provider/hetzner/provision.go`, `internal/provider/hetzner/readiness.go`, `internal/state/schema.go`, `internal/cli/up.go`, `internal/cli/up_cloud_test.go`.
- Task commits are present for `04-02`: `2a0b658`, `ddb617b`, `faa86a0`.
- Required greps passed for hcloud-go v1, no hcloud-go/v2, `cloud_provision_pending`, `cloud-init status --wait`, future destroy guidance, and billable confirmation copy.
- Focused and full verification commands passed, including `go test ./...`.

---

_Phase: 04-recommended-cloud-path-and-billable-cleanup_
_Completed: 2026-05-01_
