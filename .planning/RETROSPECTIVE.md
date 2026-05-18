# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.3.2 — Self-hosted GitHub Actions runner v1

**Shipped:** 2026-05-13
**Phases:** 6 | **Plans:** 35 | **Commits:** 260 | **Timeline:** 19 days (2026-04-28 → 2026-05-17)
**LOC:** 28,387 Go (13,027 tests, 46% test ratio) | **Tags shipped:** v1.0.0 → v1.3.2 (12 releases)

### What Was Built

- **Phase 1** — Go/Cobra CLI foundation with versioned non-secret state, GitHub auth (gh-first + fine-grained token), runner-token permission probes, public/fork safety gates, and shared redaction.
- **Phase 2** — BYO Linux happy path: SSH preflight with host-key trust, non-root `runnerkit-runner` systemd bootstrap, repo-scoped runner registration, label/workflow snippet guidance.
- **Phase 3** — Operations layer: read-only `status` health classification, bounded redacted `logs`, `doctor` with stable findings + remediation, guided `recover`, safe `down` cleanup with stale-runner deregistration and partial-cleanup checkpoints.
- **Phase 4** — Hetzner cloud path: plan-before-mutation provisioning, env-only credentials, provider inventory in state, shared BYO bootstrap reuse, billable `destroy` with verify-before-state-removal.
- **Phase 5** — Scoped ephemeral mode: explicit `--mode persistent|ephemeral` with 24h TTL, mode-aware safety blocking public/fork persistent runs, one-shot systemd unit with `_diag` log preservation, `docs/safety.md` self-hosted guidance.
- **Phase 6** — Release/distribution: GoReleaser pipeline with cosign keyless signing (4 platforms), Homebrew Cask via separate tap repo, lazy 24h update notifier, channel-detecting `runnerkit upgrade`, idempotent `upgrade-runner`, forward-only state migration framework, `RKD-<COMPONENT>-NNN` error registry, 6-component troubleshooting docs.
- **In-milestone patches** (v1.0.1..v1.3.2): host RAM/swap diagnostics (v1.0.9, Phase 7), SEED-004 tier 1 UX polish (v1.1.0, Phase 08), SEED-002 multi-repo BYO with shared runner cache (v1.2.0), Hetzner cloud-init v3 with scoped sudoers (v1.3.x), `bootstrap.BaselinePackages` for runner-image parity, workflow auto-detection of `apt-get install` deps, BYO non-TTY sudo gap closure (Plans 06-13..16).

### What Worked

- **Phased plan-then-execute discipline** — 35 plans landed with no execution rework above per-plan attempts; phase boundaries matched natural integration points (auth → BYO → ops → cloud → ephemeral → release).
- **Atomic per-plan commits and SUMMARY.md files** — when smoke tests went red (e.g. Plan 06-07 attempts 17-19), the surgical bug list and resume signals in summaries let work pick back up without re-discovery.
- **Fake/real adapter split** — `fakes.NewBYOExecutor`, `fakes.NewGitHubService`, and `fakes.NewHetznerProvider` enabled fast iteration on UX/CLI surface while gating live-smoke behind `make smoke-live` and an explicit JSON contract.
- **Stable error-code surface (`RKD-<COMPONENT>-NNN`)** — the late-Phase-6 introduction of stable IDs paid off immediately when supporting BYO non-TTY users (RKD-BOOT-016/017/018, RKD-GH-008).
- **Cloud-init v3 readiness gate** — switching from "boot-finished file exists" to "`cloud-init status --wait` and rejecting `status: error`" caught real-world recoverable-error cases that v2 marked ready prematurely.

### What Was Inefficient

- **Plan 06-07 smoke loop (attempts 1-20)** — non-TTY sudo + scoped sudoers allowlist needed three discovery rounds (Bug 31 → byo-prepare probe gap → preflight + Path B/C → cloud-init v3) before BYO worked end-to-end on hosts where `sudo` prompts for a password. Earlier framing of "no TTY in CI" as a primary scenario would have shortened this.
- **Milestone label drift** — roadmap milestone labeled `v1.0.0`, but actual releases shipped through `v1.3.2` (12 tags) before the milestone was archived. `STATE.md` `milestone:` field never advanced past `v1.0.0`. Future milestones should bump the label as tags ship, or commit to milestone = first public tag only.
- **`gsd-tools milestone complete --help`** — the CLI treats `--help` as a positional `version` argument, producing `--help-ROADMAP.md` and a `--help --help` MILESTONES.md entry. Required manual cleanup. Upstream fix candidate.
- **GitHub-hosted runner-image parity arrived late** — users hit "linker `cc` not found" on minimal Ubuntu hosts well into v1.x. `bootstrap.BaselinePackages` should have been part of Phase 2 or 3, not a v1.3.x patch.

### Patterns Established

- **Sudoers scoping pattern** — `internal/bootstrap/sudoers.go` plus `visudo -c` validation on a temp file became the canonical way to add NOPASSWD privileges without lockout risk. Reused by BYO Path C and Hetzner cloud-init v3.
- **JSON contract assertions in smoke** — `scripts/smoke/assert-doctor-json-contract.sh` / `assert-list-json-contract.sh` enforce schema_version, stage, and stable array fields, preventing silent JSON regressions. Adopt for every user-facing `--json` command.
- **Forward-only state migration with side-by-side backup** — state schema bumps copy old state to `state.v<N>.json.bak` before rewriting, never destructively. Already paying off for v1.0 → v1.1 → v1.2 state additions.
- **Plan-before-mutation cloud output** — Hetzner provisioning prints cost-aware plan, asks for confirmation, then mutates. Should be the default for any future provider integration.

### Key Lessons

1. **Treat the smoke harness as a first-class deliverable, not late-phase scaffolding.** Building `make smoke-live` + JSON-contract assertions in Phase 6 made Phase 6 itself harder to verify; future projects should land a minimum smoke harness in Phase 2 or 3.
2. **"Sudo with NOPASSWD" is not the typical host shape.** Default assumptions: password-protected sudo, no TTY in automation, sudoers needs scoped allowlist. Code paths must work without `(root) NOPASSWD: ALL`.
3. **Match the runner image from day one if you bootstrap CI.** GitHub-hosted Ubuntu 24.04 has ~75 baseline apt packages users implicitly depend on; missing any of them surfaces as obscure linker/cc failures, not as "missing package X".
4. **Milestone labels in STATE.md must advance with tags, or be archived early.** A `v1.0.0` milestone label across 12 public tags creates archive confusion and stale memory records.
5. **Stable error codes pay back fast.** Three weeks after introduction, RKD codes are referenced from docs, dashboards, and user issues. Should have been introduced in Phase 3 with diagnostics.

### Cost Observations

- Model mix: not tracked. (Recommend tracking model/session counts per phase starting next milestone.)
- Sessions: not tracked.
- Notable: heavy reuse of `gsd-planner` + `gsd-executor` agents; large Plan 06-07 retry loop suggests value in `/gsd:debug` sessions for stuck plans (introduced mid-milestone).

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Plans | Key Change |
|-----------|--------|-------|------------|
| v1.3.2 | 6 | 35 | First public release; established GSD plan-execute-summarize loop and per-plan atomic commits. |

### Cumulative Quality

| Milestone | Test LOC | Test ratio | Public tags shipped |
|-----------|----------|-----------|--------------------|
| v1.3.2 | 13,027 | 46% | 12 (v1.0.0 → v1.3.2) |

### Top Lessons (Verified Across Milestones)

*(Single milestone so far — re-evaluate after v1.4 / v2.0.)*
