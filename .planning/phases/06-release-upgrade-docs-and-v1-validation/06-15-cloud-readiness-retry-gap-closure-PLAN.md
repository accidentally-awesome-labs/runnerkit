---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 15
type: execute
wave: 1
depends_on: [14]
gap_closure: true
requirements: [REL-05]
status: proposed
---

# Plan 06-15: Cloud Readiness Retry Gap Closure

## Context

After Plan 06-14, BYO smoke succeeds in non-TTY automation. Full `make smoke-live`
is still SMOKE-RED because cloud-up exits before the Hetzner server is ready:

- `Cloud machine is not ready for runner registration yet`
- `Hetzner server <id> is not running with a public IP yet`

## Objective

Make cloud readiness robust enough that normal Hetzner provisioning delay does
not fail `runnerkit up --cloud hetzner` during smoke.

## Proposed Scope

1. Reproduce cloud readiness failure in deterministic tests.
2. Extend/adjust readiness polling and timeout behavior for the
   "server running + public IP attached" gate.
3. Keep existing destroy safety and checkpoint semantics intact.
4. Re-run `make smoke-live` to validate BYO + cloud in one pass.

## Success Criteria

- `make smoke-live-cloud` reaches runner registration path without early
  "server not running with public IP yet" failure under normal provisioning delay.
- `make smoke-live` can proceed to stopwatch and verification baseline fill-in.

