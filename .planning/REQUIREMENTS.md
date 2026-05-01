# Requirements: RunnerKit

**Defined:** 2026-04-28
**Core Value:** A solo developer can get a reliable, cost-effective GitHub Actions self-hosted runner online and usable in a project in about 10 minutes, without manual GitHub runner setup headaches.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### CLI Onboarding

- [x] **CLI-01**: Developer can install and run a `runnerkit` CLI binary on their local machine.
- [x] **CLI-02**: Developer can start a guided setup flow that explains required choices and prerequisites before making changes.
- [x] **CLI-03**: Developer can complete a happy-path setup in about 10 minutes for a supported repository and machine path.
- [x] **CLI-04**: Developer can see a clear completion summary that includes runner name, labels, machine target, and next workflow step.

### GitHub Integration

- [x] **GH-01**: Developer can authenticate RunnerKit for a GitHub repository with the minimum permissions needed for self-hosted runner management.
- [x] **GH-02**: Developer can register a repository-scoped GitHub Actions self-hosted runner without manually copying GitHub setup commands.
- [x] **GH-03**: Developer can deregister a GitHub Actions runner when removing or recreating a RunnerKit-managed runner.
- [x] **GH-04**: Developer can use predictable RunnerKit labels to target the registered runner from GitHub Actions workflows.
- [x] **GH-05**: Developer can view copy-paste `runs-on` guidance for the registered runner labels without RunnerKit editing workflow files.

### Machine Setup

- [x] **MACH-01**: Developer can connect RunnerKit to an existing Linux machine over SSH and run preflight checks before installation.
- [x] **MACH-02**: Developer can bootstrap a BYO Linux machine with a dedicated non-root runner user, required dependencies, and a managed runner service.
- [x] **MACH-03**: Developer can provision one recommended low-cost cloud machine path when they do not already have a machine.
- [x] **MACH-04**: Developer can install the runner on the provisioned cloud machine using the same lifecycle path as BYO machines.
- [x] **MACH-05**: Developer can see enough machine/provider identity in RunnerKit state to safely manage or remove the runner later.

### Runner Modes

- [x] **RUN-01**: Developer can create a persistent runner as the default mode for trusted private solo-development repositories.
- [ ] **RUN-02**: Developer can choose an explicit ephemeral runner option/profile when they want stronger isolation.
- [x] **RUN-03**: Developer receives clear warnings before using persistent runners with risky public, fork, or otherwise untrusted workloads.
- [ ] **RUN-04**: Developer can understand the tradeoff between persistent and ephemeral modes before selecting a mode.

### Reliability and Operations

- [x] **REL-01**: Developer can run `runnerkit status` to see GitHub runner status, local service status, labels, and machine reachability for managed runners.
- [x] **REL-02**: Developer can run `runnerkit logs` to inspect relevant runner, service, bootstrap, or remote-install logs without manual SSH spelunking.
- [x] **REL-03**: Developer can run `runnerkit doctor` to diagnose common setup and runner failures with actionable remediation guidance.
- [x] **REL-04**: Developer can restart or recover a stopped/offline RunnerKit-managed persistent runner using documented or guided CLI steps.
- [ ] **REL-05**: Developer can update the runner binary/service or follow a documented upgrade path that prevents immediate runner rot.

### Cleanup and State

- [x] **CLEAN-01**: Developer can run a cleanup/destroy flow that shows a plan before removing GitHub, BYO, or cloud resources.
- [x] **CLEAN-02**: Developer can deregister stale GitHub runner records even when local or remote machine state is partially missing.
- [x] **CLEAN-03**: Developer can remove BYO runner files/services from a machine without deleting unrelated user data.
- [ ] **CLEAN-04**: Developer can destroy RunnerKit-created cloud resources and verify they are no longer billable.
- [x] **STATE-01**: RunnerKit persists versioned local state/config containing repo scope, runner name, labels, machine path, provider IDs, and cleanup metadata.
- [x] **STATE-02**: RunnerKit redacts secrets and sensitive tokens from local state, logs, diagnostics, and command output.

### Documentation and Safety

- [x] **DOC-01**: Developer can follow a concise quickstart for the BYO machine path.
- [ ] **DOC-02**: Developer can follow a concise quickstart for the recommended cloud path.
- [ ] **DOC-03**: Developer can read safety guidance explaining when persistent self-hosted runners are unsafe and when ephemeral mode is recommended.
- [ ] **DOC-04**: Developer can read cleanup and troubleshooting guidance for common failure modes.

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Cost Controls

- **COST-01**: Developer can see rough hourly/monthly cost estimates before provisioning cloud resources.
- **COST-02**: Developer can detect RunnerKit-created orphaned cloud resources that may still be billing.
- **COST-03**: Developer can configure idle shutdown or reminders for cloud runners.

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature                                                               | Reason                                                                                                     |
| --------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| Enterprise dashboard, SSO, RBAC, audit logs, and compliance reporting | v1 is for solo developers, not platform teams or enterprise governance.                                    |
| Multi-CI support                                                      | GitHub Actions needs to be excellent before adding other CI systems.                                       |
| Hosted control plane                                                  | Adds SaaS operations, trust burden, accounts, and remote orchestration scope.                              |
| Automatic workflow YAML edits or commits                              | User chose runner registration only; v1 should print snippets and leave workflow changes to the developer. |
| Broad cloud-provider matrix                                           | One excellent low-cost default path is more valuable than shallow support for many providers.              |
| Kubernetes or ARC-first architecture                                  | Too heavy for solo developers who want a quick cheap runner.                                               |
| Full autoscaling fleet manager                                        | High complexity and not required to validate solo-developer setup value.                                   |
| Organization-level runner management                                  | Defer until repository-level solo workflow is validated.                                                   |
| Import/adopt existing manually-created runners                        | Useful later, but not needed for the first coherent setup flow.                                            |
| Automatic `doctor --fix` repairs                                      | Diagnostics should ship first; automatic repair can follow once real failure patterns are known.           |
| Read-only workflow label validation                                   | Useful polish after labels and setup flow stabilize; v1 only needs copy-paste guidance.                    |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase   | Status   |
| ----------- | ------- | -------- |
| CLI-01      | Phase 1 | Complete |
| CLI-02      | Phase 1 | Complete |
| CLI-03      | Phase 2 | Complete |
| CLI-04      | Phase 2 | Complete |
| GH-01       | Phase 1 | Complete |
| GH-02       | Phase 2 | Complete |
| GH-03       | Phase 3 | Complete |
| GH-04       | Phase 2 | Complete |
| GH-05       | Phase 2 | Complete |
| MACH-01     | Phase 2 | Complete |
| MACH-02     | Phase 2 | Complete |
| MACH-03     | Phase 4 | Complete |
| MACH-04     | Phase 4 | Complete |
| MACH-05     | Phase 4 | Complete |
| RUN-01      | Phase 2 | Complete |
| RUN-02      | Phase 5 | Pending  |
| RUN-03      | Phase 2 | Complete |
| RUN-04      | Phase 5 | Pending  |
| REL-01      | Phase 3 | Complete |
| REL-02      | Phase 3 | Complete |
| REL-03      | Phase 3 | Complete |
| REL-04      | Phase 3 | Complete |
| REL-05      | Phase 6 | Pending  |
| CLEAN-01    | Phase 4 | Complete |
| CLEAN-02    | Phase 3 | Complete |
| CLEAN-03    | Phase 3 | Complete |
| CLEAN-04    | Phase 4 | Pending  |
| STATE-01    | Phase 1 | Complete |
| STATE-02    | Phase 1 | Complete |
| DOC-01      | Phase 2 | Complete |
| DOC-02      | Phase 4 | Pending  |
| DOC-03      | Phase 5 | Pending  |
| DOC-04      | Phase 6 | Pending  |

**Coverage:**

- v1 requirements: 33 total
- Mapped to phases: 33
- Unmapped: 0 ✓

---

_Requirements defined: 2026-04-28_
_Last updated: 2026-05-01 after Phase 4 Plan 04-03 completion_
