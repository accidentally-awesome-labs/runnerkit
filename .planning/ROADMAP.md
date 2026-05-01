# Roadmap: RunnerKit

## Overview

RunnerKit v1 builds from a safe CLI foundation into a usable self-hosted GitHub Actions runner workflow for solo developers. The sequence proves the fastest BYO Linux persistent-runner path first, hardens diagnostics and cleanup before adding billable cloud resources, then adds scoped ephemeral operation, release packaging, upgrade support, and documentation needed for a confident public v1.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: CLI, Auth, State, and Safety Foundation** - Establish the installable CLI shell, guided setup skeleton, GitHub authentication, versioned state, and redaction rules. _(completed 2026-04-29)_
- [x] **Phase 2: BYO Persistent Runner Happy Path** - Let a solo developer connect an existing Linux machine and register a repository-scoped persistent runner with labels and next-step guidance. _(completed 2026-04-29)_
- [x] **Phase 3: Operations, Diagnostics, and BYO Cleanup** - Make managed BYO runners observable, recoverable, and safely removable when GitHub, SSH, or local state is imperfect. _(completed 2026-04-29)_
- [ ] **Phase 4: Recommended Cloud Path and Billable Cleanup** - Add one low-cost cloud provisioning path that reuses the BYO lifecycle and can be destroyed without surprise bills.
- [ ] **Phase 5: Scoped Ephemeral Mode and Safety Profiles** - Add explicit ephemeral mode for stronger isolation and make mode tradeoffs understandable before selection.
- [ ] **Phase 6: Release, Upgrade, Docs, and v1 Validation** - Package RunnerKit for real users, document operations, and validate the end-to-end v1 promise.

## Phase Details

### Phase 1: CLI, Auth, State, and Safety Foundation

**Goal**: RunnerKit has a runnable CLI foundation that can safely authenticate to GitHub, explain setup prerequisites, persist non-secret state, and redact sensitive data before any real runner install flow depends on it.
**Depends on**: Nothing (first phase)
**Requirements**: [CLI-01, CLI-02, GH-01, STATE-01, STATE-02]
**Success Criteria** (what must be TRUE):

1. Developer can install or locally run a `runnerkit` binary and start a guided setup flow that explains prerequisites before making changes.
2. Developer can authenticate RunnerKit for a target GitHub repository using only the permissions required for runner management.
3. RunnerKit persists versioned local state/config for repo scope, runner identity, labels, machine path, provider IDs, and cleanup metadata.
4. RunnerKit redacts tokens, secrets, and sensitive machine/provider values from state-adjacent logs, diagnostics, and command output.
   **Plans**: 4 plans

Plans:

- [x] 01-01-PLAN.md — Go CLI skeleton, command routing, prompts, flags, output conventions, guided setup scaffold, and redaction minimum.
- [x] 01-02-PLAN.md — GitHub repo resolution, least-privilege authentication, runner-token adapter, safety gate, and API test fixtures.
- [x] 01-03-PLAN.md — Versioned state/config schema, label conventions, fake adapters, and idempotent workflow primitives.
- [x] 01-04-PLAN.md — Production GitHub service wiring, real CLI defaults, and default-path auth/safety regression tests.

### Phase 2: BYO Persistent Runner Happy Path

**Goal**: A developer with an existing Linux machine can complete the core happy path: SSH preflight, non-root runner bootstrap, GitHub runner registration, service start, labels, and clear workflow guidance.
**Depends on**: Phase 1
**Requirements**: [CLI-03, CLI-04, GH-02, GH-04, GH-05, MACH-01, MACH-02, RUN-01, RUN-03, DOC-01]
**Success Criteria** (what must be TRUE):

1. Developer can connect RunnerKit to a supported Linux machine over SSH, pass preflight checks, and bootstrap a dedicated non-root runner service.
2. Developer can register a repository-scoped persistent GitHub Actions runner without manually copying GitHub setup commands.
3. Developer can complete the supported BYO happy path in about 10 minutes and see a completion summary with runner name, labels, machine target, and next workflow step.
4. Developer can target the runner with predictable RunnerKit labels using copy-paste `runs-on` guidance, while RunnerKit does not edit workflow YAML.
5. Developer receives clear warnings before using persistent runners on public, fork-based, or otherwise untrusted workloads, and can follow a concise BYO quickstart.
   **Plans**: 4 plans

Plans:

- [x] 02-01: SSH connectivity, host-key handling, Linux/systemd support matrix, and remote preflight checks.
- [x] 02-02: Remote bootstrap installer for dependencies, dedicated runner user, official runner download, and managed service setup.
- [x] 02-03: Repository-scoped registration, stable runner naming/labels, service start verification, and completion summary.
- [x] 02-04: Persistent default profile, public/fork risk warnings, BYO smoke test, and BYO quickstart documentation.

### Phase 3: Operations, Diagnostics, and BYO Cleanup

**Goal**: RunnerKit reduces self-hosted-runner fragility by reconciling state across GitHub, SSH, and systemd, exposing logs, diagnosing common failures, recovering persistent runners, and cleaning up BYO installs safely.
**Depends on**: Phase 2
**Requirements**: [GH-03, REL-01, REL-02, REL-03, REL-04, CLEAN-02, CLEAN-03]
**Success Criteria** (what must be TRUE):

1. Developer can run `runnerkit status` and see GitHub runner status, local service status, labels, and machine reachability for managed runners.
2. Developer can run `runnerkit logs` and `runnerkit doctor` to inspect relevant logs and receive actionable remediation guidance without manual SSH spelunking.
3. Developer can restart or recover a stopped/offline RunnerKit-managed persistent runner through documented or guided CLI steps.
4. Developer can deregister stale GitHub runner records and remove BYO runner files/services without deleting unrelated user data, even when some state is missing.
   **Plans**: 4 plans

Plans:

- [x] 03-01: Multi-source status reconciliation across local state, GitHub runner inventory, SSH reachability, and systemd.
- [x] 03-02: Remote/local log collection, diagnostic checks, redacted output, and actionable `doctor` findings.
- [x] 03-03: Guided persistent-runner restart, re-registration, and recovery workflows for common offline/stopped-service failures.
- [x] 03-04: BYO cleanup flow, stale GitHub deregistration, partial-state checkpoints, and safe file/service removal.

### Phase 4: Recommended Cloud Path and Billable Cleanup

**Goal**: Developers without a machine can provision one recommended low-cost cloud runner path, manage it through the same lifecycle as BYO, and destroy billable resources with confidence.
**Depends on**: Phase 3
**Requirements**: [MACH-03, MACH-04, MACH-05, CLEAN-01, CLEAN-04, DOC-02]
**Success Criteria** (what must be TRUE):

1. Developer can provision one recommended low-cost cloud machine path from RunnerKit when they do not already have a machine.
2. Developer can install and manage the cloud runner through the same registration, status, logs, doctor, and cleanup lifecycle as BYO machines.
3. RunnerKit state shows enough machine/provider identity to safely manage, reconcile, or remove the cloud runner later.
4. Developer can run a cleanup/destroy flow that shows a plan, removes GitHub registration and RunnerKit-created cloud resources, and verifies those resources are no longer billable.
5. Developer can follow a concise cloud quickstart for the supported provider path.
   **Plans**: 4 plans

Plans:

- [x] 04-01: Provider interface, selected low-cost default cloud profile, credential checks, and provisioning plan output.
- [x] 04-02: Cloud VM, SSH key, firewall/network, tags, and readiness workflow for the recommended provider.
- [x] 04-03: Cloud runner installation using the shared BYO bootstrap, state reconciliation, and status/logs integration.
- [x] 04-04: Cloud destroy, orphan/billing verification, provider-state cleanup, and cloud quickstart documentation.

### Phase 5: Scoped Ephemeral Mode and Safety Profiles

**Goal**: RunnerKit offers an explicit stronger-isolation ephemeral option without pretending to be a full autoscaling fleet manager, and helps developers choose the right mode for their workload.
**Depends on**: Phase 4
**Requirements**: [RUN-02, RUN-04, DOC-03]
**Success Criteria** (what must be TRUE):

1. Developer can understand the cost, isolation, cleanup, and operational tradeoffs between persistent and ephemeral runner modes before selecting a mode.
2. Developer can choose an explicit ephemeral runner option/profile when they want stronger isolation.
3. Ephemeral runs have bounded lifecycle behavior with cleanup finalizers, TTL-style safeguards, and useful log preservation for troubleshooting.
4. Developer can read safety guidance explaining when persistent self-hosted runners are unsafe and when ephemeral mode is recommended.
   **Plans**: 3 plans

Plans:

- [ ] 05-01: Mode/profile selection UX, safety policy, and persistent-vs-ephemeral tradeoff explanations.
- [ ] 05-02: Ephemeral registration/lifecycle implementation with TTLs, cleanup finalizers, and log preservation.
- [ ] 05-03: Safety guide, risky-workload validation, and end-to-end tests for trusted and untrusted workflow scenarios.

### Phase 6: Release, Upgrade, Docs, and v1 Validation

**Goal**: RunnerKit becomes shippable for early users with official distribution, an upgrade path that prevents runner rot, operational documentation, and validation that the v1 experience meets the 10-minute reliable-runner promise.
**Depends on**: Phase 5
**Requirements**: [REL-05, DOC-04]
**Success Criteria** (what must be TRUE):

1. Developer can install an official RunnerKit release binary/package and follow a documented CLI/runner upgrade path that avoids immediate runner rot.
2. RunnerKit can migrate versioned state safely across supported releases or clearly block with recovery guidance.
3. Developer can read cleanup and troubleshooting guidance for common setup, runner, GitHub, SSH, provider, and cleanup failures.
4. A fresh user can complete at least one supported setup path in about 10 minutes, run a GitHub Actions job on RunnerKit labels, and clean up confidently.
   **Plans**: 4 plans

Plans:

- [ ] 06-01: Release packaging, checksums, install instructions, and supported-platform smoke tests.
- [ ] 06-02: Runner/CLI upgrade workflow, state migrations, compatibility checks, and rollback guidance.
- [ ] 06-03: Troubleshooting, cleanup, recovery, and common-failure documentation.
- [ ] 06-04: End-to-end v1 validation across BYO, cloud, persistent, ephemeral, status, doctor, and cleanup paths.

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6

| Phase                                          | Plans Complete | Status      | Completed  |
| ---------------------------------------------- | -------------- | ----------- | ---------- |
| 1. CLI, Auth, State, and Safety Foundation     | 4/4            | Complete    | 2026-04-29 |
| 2. BYO Persistent Runner Happy Path            | 4/4            | Complete    | 2026-04-29 |
| 3. Operations, Diagnostics, and BYO Cleanup    | 4/4            | Complete    | 2026-04-29 |
| 4. Recommended Cloud Path and Billable Cleanup | 4/4            | In Progress | -          |
| 5. Scoped Ephemeral Mode and Safety Profiles   | 0/3            | Not started | -          |
| 6. Release, Upgrade, Docs, and v1 Validation   | 0/4            | Not started | -          |
