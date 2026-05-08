---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 07
result: smoke-green
completed: 2026-05-08
resume_signal: "smoke-green baseline collected (cost/sign-off pending maintainer fill)"
---

# Plan 06-07 Summary (Attempt-21)

Plan 06-07 attempt-21 completed end-to-end. BYO and cloud live smokes both
passed in one run, D-12 gate checks passed, and stopwatch baseline durations
were captured in verification artifacts.

## Outcome

- **Resume signal:** `smoke-green baseline collected (cost/sign-off pending maintainer fill)`
- **BYO smoke:** passed
- **Cloud smoke:** passed
- **10-minute stopwatch baseline:** collected (BYO + Hetzner durations)

## Evidence

From `smoke-output.log`:

- BYO up/status/doctor/down all completed.
- BYO duration: `131s`; GitHub runner ID: `33`.
- Cloud up/status/doctor/destroy completed.
- Cloud duration: `162s`; GitHub runner ID: `34`.
- D-12 gate 1: empty-project precheck passed.
- D-12 gate 2: destroy-verify 404 polling passed.
- Captured cloud resource IDs:
  `server:129984812`, `ssh_key:112006318`, `firewall:10948144`,
  `primary_ipv4:129944803`, `primary_ipv6:129944804`.

## Artifacts Updated

- Updated: `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md`
  (filled BYO + cloud baseline runtime fields and gate statuses)
- Updated: `RELEASE-NOTES-v1.0.0.md`
  (filled stopwatch table wall-clock values)

## Next Step

- Maintainer fills Hetzner project + EUR cost from dashboard.
- Maintainer signs the verification baseline.
- Proceed with release sign-off/tag path when those manual fields are complete.
