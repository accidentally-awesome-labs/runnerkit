---
id: SEED-002
status: dormant
planted: 2026-05-08
planted_during: v1.0.0 / Phase 06 (release-upgrade-docs-and-v1-validation, attempt-19 smoke-red)
trigger_when: starting v1.2 milestone OR any milestone scoped at "multi-repo" / "shared host" / "fleet" / "cost reduction" / "register/unregister UX" / depends on SEED-001 having landed
scope: medium
---

# SEED-002: Multi-repo per host — shared bootstrap, per-repo runner registration

## Why This Matters

A solo developer typically owns 3-10 repos and one beefy desktop or one cheap cloud VM. Today runnerkit is repo-scoped: each repo means a fresh BYO bootstrap or a fresh Hetzner VM. That:
- **Wastes money on cloud:** 5 EUR/month × 5 repos = 25 EUR/month, when one cpx22 idles at <10% load most of the day and could host all 5 = **80% cost reduction**.
- **Wastes effort on BYO:** re-runs of the bootstrap dance (Path B/C — see SEED-001) per repo.
- **Wastes disk/network:** each repo's install dir downloads its own copy of the GitHub Actions runner tarball (~250 MB).

GitHub Actions self-hosted runners already support multiple processes per host — runnerkit just doesn't expose that as a first-class UX. The architecture supports it (per-repo install dirs, per-repo systemd services); the gaps are bootstrap idempotency, a `register/unregister/list` UX, a shared-binary cache, and tests covering "second repo registered on a host that already has a first repo".

## When to Surface

**Trigger:** v1.2 milestone scope OR any of:
- "multi-repo support"
- "share runner across repos"
- "reduce cloud cost"
- "fleet management"
- "register/unregister UX"
- "I want one runner serving all my repos"

**Hard prerequisite:** SEED-001 (bootstrap/lifecycle split) must have landed. Without it, multi-repo registration would require N runs of the Path B/C TTY dance — strictly worse than today.

## Scope Estimate

**Medium** — a phase or two on top of the SEED-001 split. Decomposes into:

- **Phase A — bootstrap idempotency audit.** Verify: re-running `install.sh` on a host that already has `runnerkit-runner` user / scoped sudoers / install dir leaves them untouched (no error, no data loss). Add integration tests against fresh + dirty Docker hosts.
- **Phase B — `register / unregister / list` UX.** Split `runnerkit up` cleanly into:
  - `runnerkit register --host H --repo R` — adds a repo runner; idempotent if already registered.
  - `runnerkit unregister --host H --repo R` — removes one repo's runner; leaves other repos on the host untouched.
  - `runnerkit list --host H` — enumerate all registered repos + their state.
  - `runnerkit list` (no host) — enumerate all hosts and their repos.
- **Phase C — shared binary cache.** `/opt/actions-runner/runnerkit-shared-bin/<runner-version>/` populated once per `install.sh` run; per-repo install dirs hardlink (or symlink) into it. Saves ~250 MB × N-1 repos and the network bandwidth.
- **Phase D — concurrency tests.** Two repos, two parallel jobs — verify true parallelism (each runner is its own systemd service / process). Document expected resource consumption (~150 MB idle per runner).

Probably 2 phases (`v1.2-01-multi-repo-bootstrap-idempotency` + `v1.2-02-register-unregister-list-ux`). Possibly a third for shared bin cache + integration tests.

## Breadcrumbs

Code paths that already partially support multi-repo (verified 2026-05-08):

- Install dir naming: `internal/bootstrap/install.go` uses `<owner-repo>` slug → `runnerkit-<owner-repo>-local` (already per-repo namespaced)
- Service naming: `internal/cli/up.go:runnerServiceName` uses `actions.runner.<owner-repo>.<install-dir-name>.service` (already unique per repo)
- Local foundation state: `internal/state/` is keyed by repo (the `--replace existing saved foundation state for --repo` flag exists)
- Scoped sudoers glob: Plan 06-11 Bug 27 changed the entry to `/opt/actions-runner/runnerkit-*/svc.sh` (the `*` already covers multi-repo)

Code paths that block multi-repo (gaps to close):

- `internal/cli/up.go` — pre-check `runnerNameConflict` (gap doc Bug 17) — currently refuses re-runs against our own runner; needs to be a per-repo check, not per-host
- `internal/cli/down.go` — verify scope: `down --repo X` must not affect `runnerkit-runner` user, the scoped sudoers, or repo-Y's install dir / service
- `internal/cli/status.go` — currently shows one repo at a time; needs aggregation across all registered repos on the host
- `internal/bootstrap/install.go` — runner tarball download: currently per-install-dir; refactor to host-scoped cache
- No `runnerkit list` command exists yet
- Tests: `internal/bootstrap/install_integration_test.go` covers fresh-host scenarios; needs second-repo-on-existing-host case

Related decisions:
- Phase 2 context: "service must not run as root — unaffected; multi-repo runners all share the `runnerkit-runner` user, which is exactly the existing single-repo invariant."
- Security caveat: secret crosstalk between repos (jobs from repo A could read leftover files from repo B if they leak to disk). Standard self-hosted runner limitation. Document; don't try to fix in v1.2 (per-repo-user is overkill for solo-dev).
- Token scope: prefer one PAT with `repo` scope on all targeted repos; document the alternative (per-repo PATs) for users who don't trust a single broad PAT.

## Notes

The multi-repo story is what unlocks the cost angle for cloud users and the "give me runners for my whole portfolio" story for agent automation. A Hetzner cpx22 at 5 EUR/mo serving 5 repos = **1 EUR per repo per month** — beats every managed-runner-as-a-service offering on price for solo devs by ~10x.

Org-level runners (one process serves all org repos) are a separate v1.4+ direction. Most solo devs don't have orgs; the multi-repo-per-host pattern fits the portfolio model better than org-level.

Cross-refs:
- SEED-001 (bootstrap/lifecycle split) — hard prerequisite
- SEED-003 (Claude Code plugin) — consumes the `register / unregister / list` MCP-ready surface
- Future: SEED-N (org-level runner support, ephemeral isolation, cgroups for resource caps)
