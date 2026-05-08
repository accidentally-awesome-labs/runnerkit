---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 16
type: execute
wave: 1
depends_on: [15]
gap_closure: true
requirements: [REL-05]
status: completed
---

# Plan 06-16: Cloud SSH Auth Convergence Investigation

## Context

Plan 06-15 increased readiness budgets and retries, but live cloud smoke still
fails with SSH-level `exit status 255` during `cloud.cloudinit.wait` even after
~10 minutes.

## Objective

Identify and fix the root cause of persistent SSH auth/readiness failure on
newly provisioned Hetzner machines so cloud-up reaches runner registration
reliably in live smoke.

## Proposed Scope

1. Capture concrete SSH stderr from `cloud.cloudinit.wait` in smoke output.
2. Differentiate failure classes:
   - host unreachable / route
   - key mismatch
   - auth denied (`Permission denied`)
   - user missing (`runnerkit-admin` not created yet/ever)
3. Adjust provisioning/readiness path accordingly (likely user-data /
   cloud-init contract, target user selection, or wait probe strategy).
4. Add deterministic tests for the identified failure mode and fix.
5. Re-run `make smoke-live-cloud`.

## Success Criteria

- `make smoke-live-cloud` completes without `cloud_readiness_failed`.
- `runnerkit up --cloud hetzner` reaches registration and subsequent destroy
  verification path in live smoke.

