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

### Active

- [ ] Developer can use a recommended cost-effective cloud provisioning path when they do not already have a machine.
- [ ] Developer can choose, or be guided toward, ephemeral-per-job runners when stronger isolation is needed.

### Out of Scope

- Enterprise controls such as SSO, RBAC, audit logs, compliance reporting, and fleet governance - v1 is for solo developers, not enterprise platform teams.
- Multi-CI support beyond GitHub Actions - the first version should make one CI platform excellent before broadening.
- A hosted dashboard as the primary interface - the chosen experience is CLI-only for v1.
- Automatic editing of repository workflow files - v1 should register runners and expose labels, leaving workflow changes to the developer.
- Broad cloud-provider coverage on day one - v1 should pick one headache-free, cost-effective default path and design for additional providers later.

## Current State

Phase 3 is complete and verified. RunnerKit can run the BYO persistent happy path and now has operational hardening around it: `runnerkit status` reconciles local state, GitHub runner inventory, SSH, systemd, and labels; `runnerkit logs` and `runnerkit doctor` provide redacted diagnostics; `runnerkit recover` guides restart/reinstall/re-registration; and `runnerkit down` safely removes RunnerKit-managed BYO artifacts or deletes stale GitHub runner records while preserving partial cleanup state.

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
| Start with GitHub Actions                                          | User selected GitHub Actions as the first platform; narrow support improves quality.                                                                                               | - Pending   |
| Optimize for solo developers                                       | User selected solo developers as the first audience; this keeps v1 simple and cost-focused.                                                                                        | - Pending   |
| Make the interface CLI-only                                        | User selected CLI-only for day-to-day use; avoids dashboard scope and supports fast setup.                                                                                         | - Pending   |
| Register runners only, do not edit workflows                       | User wants the tool to register runners and labels; developers update workflow files themselves. Phase 2 completion output and docs print snippets without mutating workflow YAML. | Accepted    |
| Support BYO machines and cloud provisioning                        | Phases 2-3 delivered the BYO Linux/systemd persistent lifecycle through setup, operations, recovery, and cleanup; cloud provisioning remains planned for Phase 4.                  | In Progress |
| Support both ephemeral and persistent runner models with a default | Phase 2 established persistent as the trusted-private default; explicit ephemeral mode remains planned for Phase 5.                                                                | In Progress |
| Defer enterprise features                                          | User explicitly scoped out enterprise controls for v1.                                                                                                                             | - Pending   |
| Use real GitHub service as production default                      | Phase 1 verification found fake-permitted auth/metadata unsafe; production now defaults to `gh.NewService` with `github.OSCommandRunner{}` while tests inject fakes explicitly.    | Accepted    |
| Store explicit SSH host-key trust                                  | Phase 2 requires accepted fingerprints in state and fail-closed behavior on mismatch before remote mutation.                                                                       | Accepted    |
| Install persistent BYO service as non-root                         | Phase 2 bootstrap uses the dedicated `runnerkit-runner` service user and never installs the service as root by default.                                                            | Accepted    |
| Use `down` for BYO cleanup and reserve `destroy` for cloud         | Phase 3 established safe BYO cleanup through `runnerkit down`; future billable cloud resource removal should use a separate destroy flow.                                          | Accepted    |

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

_Last updated: 2026-04-29 after Phase 3 completion_
