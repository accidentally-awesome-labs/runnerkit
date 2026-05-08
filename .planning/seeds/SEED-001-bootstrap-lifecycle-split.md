---
id: SEED-001
status: dormant
planted: 2026-05-08
planted_during: v1.0.0 / Phase 06 (release-upgrade-docs-and-v1-validation, attempt-19 smoke-red)
trigger_when: starting v1.1 milestone OR any milestone scoped at "BYO host UX rework" / "agent automation" / "MCP/skill integration" / "remove TTY dependency from bootstrap"
scope: medium
---

# SEED-001: Bootstrap/lifecycle split — separate one-time privileged install from repeated unprivileged lifecycle ops

## Why This Matters

Plan 06-07 attempt-19 (2026-05-08) hit Bug 31: preflight `sudo -n true` probe is not in byo-prepare's scoped allowlist, so Path C (recommended) does not bypass Path B's TTY prompt. Even with Plan 06-13's cheap fix (replace probe with allowlisted command), the deeper architectural issue stays: **runnerkit conflates one-time host install (privileged, requires sudo) with repeated runner lifecycle ops (registration, status, restart). It tries to own the sudo dance from the maintainer's machine over SSH, which forces every operation through Path B's interactive TTY prompt or Path C's scoped sudoers — both fragile.**

Pivot the architecture: separate the two concerns. Phase 1 (one-time install) runs natively on the host as the user's interactive shell handles the single sudo prompt. Phase 2 (lifecycle) runs SSH-as-non-root + scoped passwordless sudo for everything runnerkit ever needs. Once Phase 1 is done, runnerkit (and any agent / MCP / skill driving it) never asks for a password again, never needs a TTY, never touches `term.ReadPassword`.

Net code delta is **negative** (~-200 LOC) because it lets us delete:
- `internal/cli/byo_prepare.go` (entire file)
- `promptSudoPasswordForPathB` + Path B branching in `internal/cli/up.go`
- `RUNNERKIT_SUDO_PASSWORD` env-var threading
- `bootstrap.RewriteSudoForPasswordPipe` + sudoers-rewriter helpers
- All Path B vs Path C documentation and remediation strings

Replaces with:
- `install.sh` (signed bash script, served from a GitHub release artifact) — 200 LOC bash
- `runnerkit init --print-install-command` (or `--print-script`) — 80 LOC Go
- `runnerkit register` subcommand split out of `up` — 250 LOC Go
- `up` becomes a thin wrapper: print-install if missing, register if installed — 100 LOC Go

## When to Surface

**Trigger:** v1.1 milestone scope OR any of:
- "rework BYO UX"
- "support agent automation"
- "MCP server" / "Claude Code plugin" / "skill"
- "remove TTY dependency"
- "non-interactive bootstrap"
- "self-service install"

This is the prerequisite for SEED-002 (multi-repo per host) and SEED-003 (Claude Code plugin / skill / MCP). Without the split, those features inherit Path B/C TTY problems N times over.

## Scope Estimate

**Medium** — a phase or two. Decomposes into roughly:

- **Phase A** — design the JSON `next_actions` contract (what does every CLI command emit so callers — human + agent — get the same `current stage / next step` data).
- **Phase B** — write `install.sh`, set up GitHub release artifact + cosign signature for the script, document the curl-pipe-sudo verification step.
- **Phase C** — split `up` into `init` (print-install) + `register` (SSH non-root, token write, service start). Migrate v0.x BYO hosts via a one-time re-run docs note.
- **Phase D** — drop deprecated Path B / Path C surface. Update `docs/byo-quickstart.md` + `docs/troubleshooting/bootstrap.md` (rkd-boot-015 anchor) to reflect the new model.

Probably 2 phases (`v1.1-01-bootstrap-lifecycle-split` + `v1.1-02-deprecate-paths-bc`). No new external dependencies. Test surface: integration tests for install.sh against fresh Docker hosts (Ubuntu 22/24, Debian 12, Fedora 40); unit tests for the new register flow.

## Breadcrumbs

Code that this seed will modify or delete (paths verified in current repo as of 2026-05-08):

- `internal/cli/byo_prepare.go` — DELETE entirely (Path C command)
- `internal/cli/up.go:2096-2148` — DELETE `promptSudoPasswordForPathB` + branching
- `internal/preflight/checks.go:148` — REMOVE `sudo -n true` probe (no longer needed in Phase 2 lifecycle path)
- `internal/bootstrap/install.go:122-126` — REMOVE `RewriteSudoForPasswordPipe` codepath
- `internal/bootstrap/sudoers.go` — KEEP but invoked by install.sh, not over SSH
- `internal/redact/redact.go` (`SudoPassword` registration) — DELETE
- `docs/byo-quickstart.md` — REWRITE around the new curl-pipe-sudo install + `runnerkit register` flow
- `docs/troubleshooting/bootstrap.md` — RKD-BOOT-015 entry retired or reframed

Related decisions (from `.planning/PROJECT.md` / phase CONTEXT.md):
- D-04 (live BYO smoke) — directly affected; smoke harness becomes trivially non-TTY
- D-13 (Stopwatch Checklist / smoke-green resume signal) — re-baselined
- Phase 2 service-must-not-run-as-root invariant — preserved (install.sh creates same `runnerkit-runner` user as today)

Related artifacts to revisit before planning:
- `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md` — the entire gap doc Bugs 1-31 catalogue motivating this rework
- `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-07-live-smoke-rerun-and-baseline-fillin-PLAN.md` — the smoke that exposed the architectural conflation

## Notes

This seed was planted on 2026-05-08 right after Plan 06-07 attempt-19 smoke-red. Plan 06-13 (cheap fix to Bug 31 — change preflight probe from `sudo -n true` to `sudo -n install --version`) is the v1.0.0 unblock; SEED-001 is the architectural follow-up that prevents the same class of bug from re-emerging in future BYO surfaces.

The cleanest expression of why this matters: the user typed sudo password ONCE, on their host, in their own shell. That's it. RunnerKit (and all agents driving it) then own everything else without ever seeing a password prompt or needing a TTY. No `expect`, no `term.ReadPassword`, no Path B/C branching, no `RUNNERKIT_SUDO_PASSWORD` env var threading.

Cross-refs:
- SEED-002 (multi-repo per host) — depends on this split
- SEED-003 (Claude Code plugin) — consumes the `next_actions` JSON contract this split establishes
