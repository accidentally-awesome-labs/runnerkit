---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 09
status: complete
completed: 2026-05-06
gap_closure: true
requirements: [REL-05]
---

# Plan 06-09 Summary: BYO Bootstrap + Online-Check Gap-Closure Fixes

## Outcome

Closed 15 BYO-bootstrap blockers (Bugs 4-18) discovered across Plan
06-07 attempts 2-15 against `salar@mckee-small-desktop` (Ubuntu 24.04
LTS, password-protected sudo, residential IP). Plan 06-07 attempt-15
completed BYO smoke end-to-end in 125 seconds:

- `runnerkit byo-prepare` lands scoped sudoers (Bugs 4, 5, 6).
- Preflight passes against password-protected sudo + rate-limited
  IP (Bugs 7, 8).
- Bootstrap completes: configure_runner → install_service → verify_service
  (Bugs 9, 10, 11, 12, 14, 15).
- Re-runs are idempotent against stale `.runner` + systemd unit
  (Bugs 13, 14).
- Online-check matches GitHub's CamelCase auto-labels (Bug 16).
- Pre-bootstrap conflict check skips self-collision (Bug 17).
- Smoke harness assertions pass without sudo (Bug 18).

## Commits

25 commits total under `(06-09)` prefix:

| Bug | RED | GREEN |
|-----|-----|-------|
| 4 (Prompts nil) | 3813a8e | 1d1888e |
| 5 (mktemp owner) | cc44067 | 62cdd2a |
| 6 (visudo mangle) | 3c9bf59 | b1ce1c1 |
| 7 + 8 (preflight) | e7e2cfb | 9a08b01 |
| 9 (configure Sudo) | f5e35ee | f195a83 |
| 10 (cred-prime) | a176a82 | 281966d |
| 11 (su cwd) | 64b826c | beef841 |
| 12 (stderr surface) | 76f0a14 | 248c68b |
| 13 (stale .runner) | 4d76d14 | ae1a702 |
| 14 (stale unit) | da3ad73 | 7bc9b25 |
| 15 (verify cd) | ec19486 (bundled) | ec19486 |
| 16 (label case) | 91c45ff (bundled) | 91c45ff |
| 17 (self-collision) | 485daa3 (bundled) | 485daa3 |
| 18 (smoke sudo) | n/a | c7ede69 |

Plus opening doc commit `3c8be4d` (Bug 4 + Task G filed in gap doc).

## Key Files Created/Modified

**Created:**
- `internal/ui/cli_prompter.go` (CLIPrompter — Confirm/Select/Password)
- `internal/ui/cli_prompter_test.go`
- `internal/bootstrap/sudo_rewrite.go` (`RewriteSudoForPasswordPipe`)
- `internal/bootstrap/sudo_rewrite_test.go`
- `internal/preflight/checks_bugfix_test.go`
- `internal/cli/runner_online_test.go`
- `cmd/runnerkit/main_test.go`

**Modified:**
- `cmd/runnerkit/main.go` — wire `Prompts: ui.NewCLIPrompter(...)`
- `go.mod` / `go.sum` — promote `golang.org/x/term v0.10.0` to direct
- `internal/bootstrap/sudoers.go` — sudo mktemp staging
- `internal/bootstrap/install.go` — Sudo: true; cred-prime; verify cd; ServiceNotActiveError fields
- `internal/bootstrap/script.go` — register_runner cd; idempotent rm; svc.sh stop+uninstall
- `internal/cli/byo_prepare.go` — use RewriteSudoForPasswordPipe helper
- `internal/cli/up.go` — service stderr surfaced (4 sites); isRunnerKitManagedRunner; case-insensitive label match
- `internal/preflight/checks.go` — stderr-based privilege classification; curl flag fix
- `scripts/smoke/byo-permission.sh` — drop sudo from world-readable test -f checks
- `.planning/phases/06-.../06-GAP-byo-sudo-handling.md` — Bugs 4-18 + Tasks G-T documented

## Cloud Validation

The following 5 bugs surfaced during cloud-phase live smoke (Plan
06-07 attempt-15) and are deferred to Plan 06-10 (split-out):

- Bug 19: `runnerkit status` looks for wrong systemd unit name
- Bug 20: `runnerkit status` / `runnerkit doctor` label-drift detector is case-sensitive (same family as Bug 16, different code path)
- Bug 21: `runnerkit down` remote cleanup uses sudo without TTY/-S/askpass
- Bug 22: cloud `runnerkit up` SSH host-key-readiness probe failed despite Hetzner server provisioned
- Bug 23: cloud `runnerkit destroy` ordering — server destroyed but firewall + primary IPs not detached → orphaned billing resources

These are post-up surface bugs (BYO + cloud); they do not block
Plan 06-09's gap-closure goal (BYO bootstrap path works end-to-end +
smoke harness assertions pass). Plan 06-10 targets them.

## Verification

- `go test ./... -count=1 -race` passes (17/17 packages, all green).
- Plan 06-07 attempt-15 BYO live smoke completed successfully:
  `BYO_DURATION_SECONDS=125`, runner id 24 online with expected labels.
- `06-VERIFICATION.md` baseline still pending Plan 06-10 closure
  (cloud smoke + status/doctor/down clean output) before maintainer
  sign-off.

## Pending Maintainer Action

Plan 06-07 attempt-15+ (after Plan 06-10 closure) will close the
maintainer human-action checkpoint with `smoke-green` once cloud
phase + status/doctor/down all pass cleanly.
