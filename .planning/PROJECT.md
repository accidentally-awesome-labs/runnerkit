# RunnerKit

## What This Is

RunnerKit is a CLI-first tool that helps solo developers quickly create and manage self-hosted GitHub Actions runners without becoming infrastructure operators. It should make the first successful runner feel nearly one-command: connect GitHub, choose a simple path, register a runner, and see jobs run on affordable self-hosted capacity.

The product should support both bring-your-own machines and a recommended low-cost cloud provisioning path, while keeping the v1 experience focused on GitHub Actions and command-line workflows.

## Core Value

A solo developer can get a reliable, cost-effective GitHub Actions self-hosted runner online and usable in a project in about 10 minutes, without manual GitHub runner setup headaches.

## Requirements

### Validated

- Phase 1 complete (2026-04-29): RunnerKit has a runnable Go/Cobra CLI foundation, guided `runnerkit up` scaffold, real GitHub auth/metadata/runner-permission checks in the production default path, versioned non-secret local state, stable labels/snippet guidance, and shared redaction.
- Phase 2 complete (2026-04-29): RunnerKit has the BYO persistent runner happy path: SSH target intake, host-key trust, Linux/systemd preflight, non-root bootstrap scripts, repository runner registration, online verification, persistent state, RunnerKit label guidance, safety warnings, fake smoke coverage, and BYO quickstart docs.
- Phase 3 complete (2026-04-29): RunnerKit has BYO operations hardening: read-only `status`, bounded redacted `logs`, read-only `doctor` findings, guided `recover`, safe `down` cleanup, stale GitHub deregistration, partial cleanup checkpoints, and updated troubleshooting/cleanup docs.
- Phase 4 complete (2026-05-01): RunnerKit has one recommended Hetzner cloud path with plan-before-mutation provisioning, env-only provider credentials, cloud inventory in state, shared BYO bootstrap/registration lifecycle, provider-aware status/logs/doctor, billable `destroy` cleanup, provider verification before state removal, and cloud quickstart docs.
- Phase 5 complete (2026-05-02): RunnerKit has explicit `--mode persistent|ephemeral` selection with `--ephemeral-ttl 24h` default, mode/profile tradeoff rendering before mutation, mode-aware safety policy that blocks public/fork persistent runs and steers untrusted workloads to ephemeral, scoped one-job ephemeral lifecycle with cleanup finalizers and TTL safeguards, `_diag` log preservation across `down`/`destroy`, ephemeral-aware `status`/`logs`/`doctor`, `docs/safety.md` self-hosted guidance with quickstart updates, and E2E coverage for trusted+untrusted persistent/ephemeral.
- Phase 7 complete (2026-05-12, **v1.0.9**): Host RAM/swap warnings in preflight and `doctor` (RKD-BOOT-016/017), bounded journal heuristics for likely OOM / linker SIGKILL when the runner is unhealthy or with `doctor --deep`, JSON field **`host_incident_hints`**, troubleshooting in **`docs/troubleshooting/host-resources.md`**, live-smoke **`assert-doctor-json-contract.sh`**, and stable JSON arrays for **`host_incident_hints`** / **`next_actions`** in **`doctor --json`**. See [.planning/phases/07-host-capacity-and-oom-diagnostics/07-01-PLAN.md](phases/07-host-capacity-and-oom-diagnostics/07-01-PLAN.md).
- Phase 08 complete (2026-05-12, **v1.1.0**): SEED-004 tier 1 UX polish — first-run wizard (no subcommand), **`--explain`** / **`--unicode`**, **`stage`** + **`schema_version`** on **`status --json`** and **`doctor --json`**, boxed commands, BYO **`up`/`register`** checklist + **`sessions/`** persistence, **`doctor --fix`** / **`--ignore`** + **`config.json`**, troubleshooting [**`docs/troubleshooting/doctor-ux.md`**](docs/troubleshooting/doctor-ux.md). Tier 2: [.planning/phases/08-ux-polish-seed-004/08-02-TIER2-PLAN.md](phases/08-ux-polish-seed-004/08-02-TIER2-PLAN.md).
- **SEED-002 multi-repo (2026-05-12, v1.2.0):** Several private repos on one BYO host — shared **`runnerkit-runner`** + versioned tarball cache under **`/opt/actions-runner/runnerkit-shared-bin/<version>/`**, **`register`** lifecycle path with **`lifecycle_foundation_missing`** when foundation is absent, **`list`** / **`list --json`** with host grouping, **`unregister`** alias of **`down`**, **`doctor`** shared-host hint, live-smoke **`assert-list-json-contract.sh`** and env-gated second-repo BYO path (**`RUNNERKIT_SMOKE_MULTI_REPO`**, **`RUNNERKIT_SMOKE_REPO2`**). See [**`docs/troubleshooting/multi-repo.md`**](docs/troubleshooting/multi-repo.md).
- Phase 6 complete (2026-05-13, **v1.3.2**): Tag-triggered GoReleaser pipeline with cosign-signed 4-platform binaries, Homebrew Cask publishing to a separate tap repo, lazy 24h update notifier, channel-detecting `runnerkit upgrade`, idempotent `upgrade-runner`, forward-only state migration, stable `RKD-<COMPONENT>-NNN` error code registry, 6-component troubleshooting docs, live smoke harness, BYO non-TTY sudo gap closures (Bugs 24-27, 31), Hetzner cloud-init v3 with scoped sudoers and `--extra-packages` baseline-package parity with GitHub-hosted runner image.

### Active

_(none — next milestone planning pending. Candidate inputs: `.planning/seeds/SEED-001`, `SEED-003`; remainder of SEED-001/004 not yet shipped.)_

### Out of Scope

- Enterprise controls such as SSO, RBAC, audit logs, compliance reporting, and fleet governance - v1 is for solo developers, not enterprise platform teams.
- Multi-CI support beyond GitHub Actions - the first version should make one CI platform excellent before broadening.
- A hosted dashboard as the primary interface - the chosen experience is CLI-only for v1.
- Automatic editing of repository workflow files - v1 should register runners and expose labels, leaving workflow changes to the developer.
- Broad cloud-provider coverage on day one - v1 should pick one headache-free, cost-effective default path and design for additional providers later.

## Current State

**Milestone v1.3.2 complete (2026-05-13).** All 6 planned phases shipped: CLI foundation, BYO happy path, BYO ops/diagnostics, Hetzner cloud, scoped ephemeral mode + safety profiles, and release/upgrade/docs/v1 validation. Inline patches shipped in `v1.0.x` (Phase 7 host RAM/swap + journal OOM hints in v1.0.9), `v1.1.0` (SEED-004 tier 1 UX polish — wizard, stage JSON, explain, doctor fix, BYO checklists), `v1.2.0` (SEED-002 multi-repo BYO with shared runner cache + `list`), and `v1.3.x` (cloud-init v3 with scoped sudoers, baseline-package + workflow auto-detect parity with GitHub-hosted Ubuntu 24.04 runner image, BYO non-TTY sudo gap closure).

RunnerKit centers **`--mode persistent|ephemeral`**, mode-aware safety, Hetzner as the default cloud path, and CLI-only operations. Maintainer releases follow **`docs/release-process.md`** (including **`make smoke-live`** before tags). Distribution: **Homebrew Cask** via `accidentally-awesome-labs/homebrew-tap`, cosign-signed binaries via GitHub Releases.

**Next:** milestone TBD. Run `/gsd:new-milestone` when ready.

## Context

The idea came from frustration with self-hosted CI runner setup being too manual, too fragile, and too expensive. Existing setup flows require developers to copy commands from GitHub, manage services, think about token/registration lifecycle, troubleshoot runners going offline, and decide how to host machines economically.

The intended first audience is solo developers working on personal repositories, side projects, and small independent projects. They want the cost and control benefits of self-hosted runners without spending time building bespoke runner infrastructure.

Important product shape decisions gathered during initialization:

- GitHub Actions is the v1 CI target.
- The interface should be CLI-only.
- The first-run experience should feel like one command plus a few necessary prompts.
- RunnerKit should support both BYO machines and cloud provisioning, with research/planning choosing the most seamless and cost-effective default provider.
- RunnerKit should support both persistent managed pools and ephemeral runners, with a sensible default determined by workload/security/cost tradeoffs.
- RunnerKit should register runners and labels, but not automatically modify GitHub Actions workflow YAML in v1.

## Constraints

- **Audience**: Optimize for solo developers first - keep setup, terminology, and operations lightweight.
- **CI platform**: GitHub Actions only in v1 - prevents diluted support across CI systems.
- **Interface**: CLI-only in v1 - avoids dashboard complexity and keeps installation simple.
- **Setup time**: Target about 10 minutes from install to first usable runner - this is the key usefulness bar.
- **Cost**: Recommended defaults should be visibly cost-effective versus simply using GitHub-hosted runners for suitable workloads.
- **Reliability**: The product must reduce fragility through status checks, recovery guidance, and cleanup flows.
- **Cloud strategy**: Choose one excellent default provisioning path first, while leaving room for provider plugins or additional providers later.

## Key Decisions

| Decision                                                           | Rationale                                                                                                                                                                          | Outcome     |
| ------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- |
| Start with GitHub Actions                                          | User selected GitHub Actions as the first platform; narrow support improves quality.                                                                                               | ✓ Good — v1.3.2 ships GitHub-only with no support pressure for other CI |
| Optimize for solo developers                                       | User selected solo developers as the first audience; this keeps v1 simple and cost-focused.                                                                                        | ✓ Good — v1.3.2 wizard + checklists keep first-run friction low |
| Make the interface CLI-only                                        | User selected CLI-only for day-to-day use; avoids dashboard scope and supports fast setup.                                                                                         | ✓ Good — no dashboard scope creep through v1.3.2 |
| Register runners only, do not edit workflows                       | User wants the tool to register runners and labels; developers update workflow files themselves. Phase 2 completion output and docs print snippets without mutating workflow YAML. | Accepted    |
| Support BYO machines and cloud provisioning                        | Phases 2-4 delivered the BYO Linux/systemd persistent lifecycle and one recommended Hetzner cloud path through setup, operations, recovery/destroy, and cleanup documentation.     | Accepted    |
| Support both ephemeral and persistent runner models with a default | Phase 2 established persistent as the trusted-private default; Phase 5 added explicit `--mode persistent\|ephemeral` with 24h ephemeral TTL, mode-aware safety policy, and tradeoff rendering before mutation. | Accepted    |
| Defer enterprise features                                          | User explicitly scoped out enterprise controls for v1.                                                                                                                             | ✓ Good — none shipped through v1.3.2; not a blocker |
| Tag releases from upstream only                                    | GoReleaser OIDC signing requires push from `accidentally-awesome-labs/runnerkit`; fork tags break signing and don't update brew/releases.                                          | Accepted — codified in `docs/release-process.md` |
| Baseline-package parity with GitHub-hosted runner image            | v1.3.x users hit "linker cc not found"/missing pkg-config on minimal Ubuntu hosts; `bootstrap.BaselinePackages` plus `--extra-packages` plus workflow auto-detection fill the gap. | Accepted — shipped 2026-05-15ish during v1.3.x |
| Cloud-init v3 with scoped sudoers + readiness gate                 | Hetzner runs needed `cloud-init status --wait` rejection of `status: error` and scoped sudoers up front to avoid `host.privilege.cloud_bootstrap` failures during bootstrap.       | Accepted — `hetzner.CloudInitUserDataVersion = runnerkit-cloud-init-v3` |
| Use real GitHub service as production default                      | Phase 1 verification found fake-permitted auth/metadata unsafe; production now defaults to `gh.NewService` with `github.OSCommandRunner{}` while tests inject fakes explicitly.    | Accepted    |
| Store explicit SSH host-key trust                                  | Phase 2 requires accepted fingerprints in state and fail-closed behavior on mismatch before remote mutation.                                                                       | Accepted    |
| Install persistent BYO service as non-root                         | Phase 2 bootstrap uses the dedicated `runnerkit-runner` service user and never installs the service as root by default.                                                            | Accepted    |
| Use `down` for BYO cleanup and reserve `destroy` for cloud         | Phase 4 implemented `runnerkit destroy` for billable cloud cleanup while `runnerkit down` remains BYO-only.                                                                        | Accepted    |
| Verify cloud resources before removing local state                 | Phase 4 requires provider `Destroy` plus `VerifyDestroyed`; partial cleanup keeps state and checkpoints until resources are absent or non-billable.                                | Accepted    |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):

1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):

1. Full review of all sections
2. Core Value check - still the right priority?
3. Audit Out of Scope - reasons still valid?
4. Update Context with current state

---

_Last updated: 2026-05-18 — milestone v1.3.2 archived; all 6 phases shipped, 33/33 requirements complete._
