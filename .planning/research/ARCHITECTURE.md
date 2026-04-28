# Architecture Research

**Domain:** CLI-first self-hosted GitHub Actions runner provisioning and lifecycle management
**Researched:** 2026-04-28
**Confidence:** MEDIUM-HIGH (patterns are stable; exact GitHub API permissions and cloud defaults should be verified against official docs during implementation)

## Standard Architecture

### System Overview

RunnerKit should be a local-first orchestrator. The CLI owns user interaction, desired state, reconciliation, and cleanup. GitHub owns runner registration and runner inventory. Machine providers own host creation/destruction. Remote hosts run the GitHub Actions runner plus minimal RunnerKit-managed metadata/service wrappers. Avoid a hosted control plane in v1.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              User Workstation                               │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌───────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │ CLI        │→ │ Orchestrator  │→ │ Plan/Confirm │→ │ Status/Repair   │  │
│  └─────┬──────┘  └───────┬───────┘  └──────┬───────┘  └────────┬────────┘  │
│        │                 │                 │                   │           │
│        │                 ↓                 ↓                   ↓           │
│  ┌─────┴────────────────────────────────────────────────────────────────┐  │
│  │ Local config/state store + secrets adapter + redacted logs             │  │
│  └─────┬────────────────────────────────────────────────────────────────┘  │
├────────┼────────────────────────────────────────────────────────────────────┤
│        │                       Adapter Boundary                             │
├────────▼───────────┬───────────────────┬──────────────────┬────────────────┤
│ GitHub Adapter     │ Provider Adapters │ Remote Executor  │ Plugin Runtime │
│ - auth/session     │ - BYO SSH         │ - SSH/SCP        │ - providers    │
│ - repo runner API  │ - default cloud   │ - cloud-init     │ - validators   │
│ - labels/status    │ - future clouds   │ - remote probes  │ - manifests    │
├────────┬───────────┴────────┬──────────┴────────┬─────────┴────────────────┤
│        │                    │                   │                           │
│        ▼                    ▼                   ▼                           │
│  GitHub Actions       Cloud Provider API       BYO Host / Cloud VM           │
│  - registration token - create/destroy VM      - runnerkit host metadata     │
│  - removal token      - firewall/SSH key       - GitHub runner binary        │
│  - runner inventory   - volume/network         - systemd service             │
│  - online/busy status - cost estimate          - logs/health probe           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
| --------- | -------------- | ---------------------- |
| CLI command layer | Commands such as `up`, `status`, `doctor`, `repair`, `down`, `logs`, `ssh`; prompts and output | Thin command handlers; no cloud/GitHub logic directly in commands |
| Core orchestrator | Converts intent into idempotent operations, orders steps, handles rollback/partial failure, emits progress | Use-case services: create runner, reconcile runner, destroy runner, upgrade runner |
| Plan/dry-run engine | Shows resources, cost estimate, labels, SSH target, and destructive cleanup effects | Provider `plan()` + GitHub permission checks + local state diff |
| GitHub adapter | Auth discovery, repo resolution, registration/removal tokens, runner list/status/delete, labels | REST API wrapper; reuse `gh` auth where possible; redact tokens |
| CI provider boundary | Encapsulates GitHub Actions-specific runner operations | Small v1 interface with one implementation; do not build multi-CI yet |
| Machine provider boundary | Creates/describes/destroys machines or resolves existing hosts | `MachineProvider` contract with `byo-ssh` and one default cloud provider |
| Remote executor | Executes commands, uploads bootstrap assets, streams logs, verifies host identity | SSH/SCP for BYO/cloud; cloud-init optional for fresh cloud VMs |
| Bootstrap installer | Installs OS packages, runner user, runner binary, service units, labels, metadata | Generated version-pinned scripts with checksum verification |
| Runner lifecycle manager | Register, start, stop, restart, unregister, upgrade, re-register stale runners | Remote scripts + GitHub short-lived tokens; no durable registration token storage |
| Host-side RunnerKit shim | Minimal host metadata, health probe, logs wrapper, optional ephemeral re-registration loop | Files under `/opt/runnerkit` and `/var/lib/runnerkit`; systemd service/timer |
| Health/status reconciler | Merges local state, GitHub status, provider state, SSH reachability, and service health | Powers `status`, `doctor`, and safe `repair` actions |
| Cleanup orchestrator | Stops service, unregisters runner, deletes stale GitHub records, destroys cloud resources, erases state | Transactional best-effort cleanup with resumable checkpoints |
| State/config store | Desired state, runner inventory, provider instance IDs, labels, SSH fingerprints, last known status | Local app config/state files or SQLite; secrets are references only |
| Secrets manager | GitHub token references, SSH key references, cloud credential references | OS keychain where possible; `gh`/provider CLI/env support; redaction in logs |
| Plugin runtime | Loads provider plugins, validates capability contracts, enforces boundaries | Stable provider SDK after BYO + first cloud provider prove the contract |

## Recommended Project Structure

Assuming a TypeScript/Node CLI. If implemented in Go or Rust, keep the same package boundaries.

```
src/
├── cli/                    # Command definitions, prompts, output rendering
│   ├── commands/           # up/status/doctor/repair/down/logs/ssh
│   └── ui/                 # spinners, tables, error formatting
├── core/                   # Use cases and orchestration, independent of vendors
│   ├── workflows/          # create-runner, reconcile, cleanup, upgrade
│   ├── planning/           # dry-run plans, cost estimates, diffs
│   ├── lifecycle/          # runner lifecycle state machine
│   └── errors/             # typed user-fixable errors
├── github/                 # GitHub Actions implementation of CI provider
│   ├── auth/               # gh/device/PAT auth adapters
│   ├── api/                # REST client wrappers
│   └── runners/            # registration/status/labels/remove APIs
├── providers/              # Machine provider implementations
│   ├── interface.ts        # MachineProvider contract and capabilities
│   ├── byo-ssh/            # Existing host provider
│   └── default-cloud/      # First supported low-cost cloud provider
├── remote/                 # SSH/cloud-init execution boundary
│   ├── ssh/                # connect, upload, exec, stream logs, fingerprinting
│   ├── bootstrap/          # generated install scripts/assets
│   └── probes/             # OS, systemd, runner service, disk/network checks
├── state/                  # Local state/config persistence and migrations
│   ├── config.ts           # User/project config schema
│   ├── store.ts            # JSON/SQLite state store
│   └── secrets.ts          # keychain/env/gh/provider credential references
├── plugins/                # Plugin loading, manifest validation, sandbox policy
├── labels/                 # Runner label conventions and validation
└── test-support/           # Fake GitHub/provider/SSH adapters for integration tests
```

### Structure Rationale

- **cli/ stays thin:** interactive UX and command flags stay separate from infrastructure mutation.
- **core/ owns workflows:** every mutating action is retryable, testable, and provider-independent.
- **github/ is isolated:** GitHub Actions is v1-only, but API quirks should not leak everywhere.
- **providers/ use one contract:** BYO SSH is modeled as a provider that returns an existing `MachineRef`; cloud providers return created machines.
- **remote/ is a hard boundary:** all host changes go through audited SSH/cloud-init helpers with redacted logs.
- **state/ is explicit:** repair and cleanup require durable records of runner IDs, instance IDs, labels, and SSH fingerprints.
- **plugins/ later, interfaces now:** define contracts in v1, but ship only BYO + one excellent cloud provider initially.

## Architectural Patterns

### Pattern 1: Local Orchestrator + Idempotent Steps

**What:** Each user action becomes a sequence of idempotent, checkpointed steps: create machine, wait for SSH, request GitHub token, install runner, register, start service, verify online.

**When to use:** All provisioning, repair, upgrade, and cleanup flows.

**Trade-offs:** More bookkeeping, but fewer orphaned VMs, stale GitHub runners, and unrecoverable half-installs.

**Example:**
```typescript
type StepResult = { status: 'done' | 'skipped' | 'retryable' | 'failed'; checkpoint?: object };

interface OperationStep<C> {
  id: string;
  alreadyDone(ctx: C): Promise<boolean>;
  run(ctx: C): Promise<StepResult>;
  rollback?(ctx: C): Promise<void>;
}
```

### Pattern 2: Provider Interface Returning a MachineRef

**What:** Core workflows do not know cloud-specific resources. Providers create or resolve machines and return a normalized connection/resource reference.

**When to use:** Both cloud and BYO flows.

**Trade-offs:** Some provider-specific features are hidden; expose them as capabilities instead of leaking cloud APIs.

**Example:**
```typescript
interface MachineProvider {
  name: string;
  capabilities: ProviderCapabilities;
  validateConfig(config: ProviderConfig): Promise<ValidationResult>;
  plan(input: ProvisionInput): Promise<ProvisionPlan>;
  provision(input: ProvisionInput): Promise<MachineRef>;
  describe(machine: MachineRef): Promise<MachineStatus>;
  destroy(machine: MachineRef): Promise<void>;
}

interface MachineRef {
  provider: string;
  id: string;
  publicHost: string;
  sshUser: string;
  sshPort: number;
  sshFingerprint?: string;
  metadata: Record<string, string>;
}
```

### Pattern 3: GitHub Registration Tokens Are Ephemeral Inputs, Not State

**What:** Request registration/removal tokens immediately before use, pass them to remote commands via stdin or ephemeral env, redact them, and never persist them.

**When to use:** Registration, re-registration, and cleanup.

**Trade-offs:** Requires GitHub API access for each register/unregister operation, but avoids token leaks and stale-token bugs.

### Pattern 4: Reconciliation Instead of Blind Status

**What:** `status` compares desired/local state with GitHub, provider, and remote host facts, then explains drift and repair actions.

**When to use:** Status, doctor, repair, cleanup, and preflight before destructive commands.

**Trade-offs:** More API calls and SSH probes, but this directly supports the product promise of reducing fragility.

### Pattern 5: Persistent Runner Default, Ephemeral as Host-Side Mode

**What:** Default v1 to a persistent runner service for simplicity/cost. Model ephemeral runners as a mode on the same host lifecycle: configure `--ephemeral`, run one job, then a RunnerKit host shim re-registers or shuts down based on policy.

**When to use:** Persistent for solo developers who want easy always-available capacity; ephemeral for untrusted workloads or cleaner job isolation.

**Trade-offs:** True scale-to-zero ephemeral-per-job requires a queue listener/webhook/control plane or ARC-like architecture. A one-shot local CLI cannot autoscale after the user's machine exits.

## Data Flow

### First-Run Cloud Provisioning Flow

```
User runs `runnerkit up`
    ↓
CLI resolves repo, labels, mode, provider, size/region
    ↓
GitHub adapter checks auth and repo permissions
    ↓
Provider.plan() returns resources, cost, warnings
    ↓
Provider.provision() creates VM, SSH key/firewall rules, optional cloud-init seed
    ↓
Remote executor waits for SSH and verifies host fingerprint
    ↓
GitHub adapter creates short-lived runner registration token
    ↓
Bootstrap installer uploads pinned runner install and service scripts
    ↓
Remote host creates runner user, installs dependencies/runner, configures labels
    ↓
Runner lifecycle manager starts systemd service
    ↓
Health reconciler checks GitHub runner online and remote service healthy
    ↓
State store records runner ID/name, labels, provider instance ID, SSH fingerprint
    ↓
CLI prints labels and workflow YAML hint
```

### BYO Machine Flow

```
User runs `runnerkit up --provider byo-ssh --host user@host`
    ↓
Remote executor validates SSH, OS, systemd/sudo, disk, and network egress to GitHub
    ↓
GitHub adapter creates registration token
    ↓
Bootstrap installer configures runner service on existing host
    ↓
State store records host metadata but marks machine as user-owned
    ↓
Cleanup later unregisters/stops runner but does not destroy the host by default
```

### Status / Doctor Flow

```
Local state
    ↓
GitHub runner inventory/status ─┐
Provider instance status ───────┼→ Reconciler → Human-readable status + repair plan
Remote SSH/service probes ──────┘
```

Status should distinguish:
- GitHub sees runner `online`, `offline`, or `busy`.
- Provider instance is running/stopped/missing.
- SSH is reachable/unreachable.
- Runner service is active/inactive/failed.
- Runner registration is stale/missing.
- Local state is missing or inconsistent.

### Cleanup Flow

```
User runs `runnerkit down`
    ↓
Load local runner/machine state and confirm destructive actions
    ↓
Create GitHub removal token if remote runner config exists
    ↓
Remote executor stops service and unconfigures runner
    ↓
GitHub adapter deletes stale runner record if still present
    ↓
Provider destroys cloud VM OR skips destroy for BYO unless explicitly requested
    ↓
State store removes runner records and retained secret references
```

### Ephemeral Runner Flow (Advanced Boundary)

```
Host-side RunnerKit shim starts
    ↓
Obtain fresh registration token through a secure pre-authorized channel
    ↓
Configure runner with --ephemeral and labels
    ↓
Runner accepts one job and auto de-registers
    ↓
Shim collects logs/exit status and cleans work directory
    ↓
Policy: re-register for next job OR shut down/destroy host
```

Important implication: a pure local CLI cannot respond to GitHub job-queue demand after the user's laptop exits. Full ephemeral-per-job autoscaling needs a resident controller on the host, a webhook listener, or a Kubernetes/ARC-style system.

## State Management

### Where State Lives

| State | Location | Notes |
| ----- | -------- | ----- |
| Project defaults | Optional `.runnerkit/config.yaml` in repo | Non-secret defaults: provider, labels, runner mode, size; safe to commit if desired |
| User/global config | OS app config dir, e.g. `~/.config/runnerkit/config.toml` | Provider, region, SSH key path, output preferences |
| Runner inventory | Local state store, e.g. `~/.local/state/runnerkit/state.json` or SQLite | Runner name/ID, repo, labels, provider instance ID, SSH fingerprint, install path, mode |
| Secrets | OS keychain, `gh` auth, provider CLI credentials, environment variables | Store references only; never write GitHub registration tokens to disk |
| Remote host metadata | `/var/lib/runnerkit/metadata.json` and `/opt/runnerkit/` scripts | Supports repair/uninstall from the host side |
| Logs | Local logs plus remote service logs via journalctl | Redact tokens and credential-bearing command lines |

### State Principles

1. **Desired vs observed:** Store what RunnerKit intended to create; compute observed state fresh from GitHub/provider/remote probes.
2. **Resumable mutations:** Every mutating command writes enough checkpoint state to continue cleanup if interrupted.
3. **No durable registration tokens:** Registration and removal tokens are short-lived command inputs only.
4. **Ownership markers:** Cloud-created machines can be destroyed by RunnerKit; BYO hosts are never destroyed unless explicitly requested.
5. **Schema migrations:** Version state schema from day one to avoid breaking existing installs.

## Secrets and Token Handling

- Prefer existing `gh` authentication where available; otherwise use a documented GitHub device/PAT flow with minimal scopes.
- Request GitHub runner registration/removal tokens only when needed; pass them over SSH via stdin or temporary env; redact from logs; do not persist.
- Use separate SSH keys created or selected by the user; record public key/fingerprint, not private key material.
- For cloud providers, prefer provider CLI auth files or environment variables; RunnerKit should not become a cloud secret vault.
- Never print full GitHub tokens, cloud credentials, registration tokens, SSH private keys, or command lines containing secrets.
- Remote bootstrap scripts should avoid writing tokens into shell history, metadata files, or systemd units.

## Remote Install / Provisioning Strategy

### Recommended v1 Approach

1. **Cloud path:** provider creates a small VM with SSH access and optional cloud-init for base user/package setup; RunnerKit finishes setup over SSH so cloud and BYO share one installer.
2. **BYO path:** SSH to the existing host and run preflight checks before mutation.
3. **Install user:** create/use a non-root `runnerkit-runner` user; avoid running CI jobs as root.
4. **Install location:** use `/opt/actions-runner/<runner-name>` for runner binaries and `/var/lib/runnerkit` for metadata/state.
5. **Service manager:** systemd Linux-first for v1. macOS/Windows should be deferred unless requirements change.
6. **Runner package:** retrieve GitHub Actions runner release by OS/architecture and verify checksum before install.
7. **Labels:** standard labels: `self-hosted`, OS/arch labels, `runnerkit`, provider, pool, and a user-visible project label.
8. **Verification:** do not declare success until GitHub reports the runner online and the remote service is active.

### Preflight Checks

- SSH authentication works and host fingerprint is accepted/recorded.
- Supported OS/architecture and service manager are present.
- `sudo` permissions are sufficient, or user selected a no-root compatible path.
- Disk space, network egress to GitHub, archive tools, and time sync are healthy.
- Existing runner directory/service name does not conflict, or the command is explicitly a repair/adopt flow.

## Plugin / Provider Boundaries

### MachineProvider Contract

Providers own machine lifecycle and cost estimates, not runner registration. Core owns GitHub and runner install.

Required methods:
- `validateConfig(config)` — credentials, region, image, SSH key, quota.
- `plan(input)` — resources to create, estimated hourly/monthly cost, warnings.
- `provision(input)` — create or resolve a machine and return `MachineRef`.
- `waitUntilReady(machine)` — block until SSH reachable or return actionable error.
- `describe(machine)` — current provider-side state.
- `destroy(machine)` — remove resources RunnerKit created.

Optional capabilities:
- `supportsCloudInit`
- `supportsSnapshots`
- `supportsStopStart`
- `supportsCostEstimate`
- `supportsFirewallRules`
- `supportsEphemeralDestroyAfterJob`

### CiProvider Contract

Keep this small for v1; GitHub is the only implementation.

Required methods:
- `resolveTarget(repoOrUrl)`
- `checkPermissions(target)`
- `createRegistrationToken(target)`
- `createRemovalToken(target)`
- `listRunners(target)`
- `getRunner(target, runnerId)`
- `deleteRunner(target, runnerId)`
- `formatWorkflowHint(labels)`

### RemoteExecutor Contract

Remote execution should be independent from provider logic.

Required methods:
- `probe(machine)` — OS, arch, service manager, dependencies.
- `upload(machine, local, remote)`
- `exec(machine, command, options)` — redacted, timeout-aware, streamable logs.
- `installService(machine, serviceSpec)`
- `readLogs(machine, unit)`
- `remove(machine, installRef)`

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
| ------- | ------------------- | ----- |
| GitHub Actions REST API | CLI-side API client for auth checks, registration/removal tokens, runner inventory/status, stale runner deletion | Repo-level runner is enough for v1 |
| `gh` CLI / GitHub auth | Reuse existing auth where possible; fallback to documented token/device flow | Reduces first-run friction; missing scopes must be clear |
| Default cloud provider | Provider plugin creates VM/SSH/firewall and estimates cost | Pick one excellent low-cost provider first |
| BYO SSH host | Provider resolves user-supplied SSH target; remote executor mutates host | Cleanup must respect user ownership |
| GitHub runner releases | Remote bootstrap retrieves version/arch-specific runner package | Verify checksum and support upgrades/rollback |
| systemd/journalctl | Host service management and logs | Linux-first v1 simplifies remote management |

### Internal Boundaries

| Boundary | Communication | Notes |
| -------- | ------------- | ----- |
| CLI ↔ Core | Function calls/events | CLI handles UX; core handles workflows |
| Core ↔ GitHub adapter | `CiProvider` interface | Keeps GitHub-specific API quirks isolated |
| Core ↔ Machine provider | `MachineProvider` interface | Providers know machines, not runner registration |
| Core ↔ Remote executor | `RemoteExecutor` interface | SSH/cloud-init details isolated from workflows |
| Lifecycle ↔ State store | Repository pattern | Mutating steps checkpoint state for recovery |
| Status ↔ Adapters | Read-only probes | Reconciler should not mutate unless invoked as `repair` |
| Plugin runtime ↔ Providers | Manifest + typed interface | Avoid arbitrary plugin access to secrets/state |

## Suggested Build Order

1. **Core CLI skeleton + state/config model** — commands, schemas, redaction, logging, fake adapters.
2. **GitHub adapter for repo-level runners** — auth detection, permissions, registration/removal tokens, list/status/delete, labels.
3. **BYO SSH persistent runner path** — fastest proof of core value without cloud complexity; includes preflight, install, service, status, cleanup.
4. **Health/doctor/repair loop** — GitHub + SSH + service checks, restart/re-register/stale cleanup guidance.
5. **Default cloud provider** — provision VM and reuse the same remote installer; include cost estimate and destroy flow.
6. **Robust cleanup and resumability** — partial-failure cleanup for stale GitHub runners and cloud orphans before broad use.
7. **Persistent pool polish** — labels, multiple runners per repo, naming, upgrade path, logs.
8. **Ephemeral mode** — only after persistent lifecycle is reliable; start with ephemeral-on-host, then evaluate queue-driven autoscaling.
9. **Provider/plugin hardening** — publish provider SDK once two implementations prove the interface.

## Scaling Considerations

| Scale | Architecture Adjustments |
| ----- | ------------------------ |
| 1 solo developer / 1 repo / 1 runner | Local JSON state is fine; SSH orchestration; persistent runner default |
| Multiple repos or small team / 2-10 runners | Add runner groups/pools, SQLite state, concurrency locks, clearer naming/labels, bulk status |
| Dozens of runners / many repos | Consider a resident controller/agent, event-driven reconciliation, centralized state, cost policies |
| Ephemeral per job / autoscaling | Requires webhook/queue listener or ARC-like controller; CLI alone is insufficient |

### Scaling Priorities

1. **First bottleneck: drift and partial failures.** Fix with idempotent workflows, checkpointed state, and reconciliation before adding providers.
2. **Second bottleneck: SSH/install variability.** Fix with strict preflight checks, Linux-first support, pinned runner versions, and clear errors.
3. **Third bottleneck: status ambiguity.** Fix with multi-source status and actionable repair.
4. **Fourth bottleneck: ephemeral autoscaling expectations.** Fix by documenting mode boundaries and not promising scale-to-zero without a controller.

## Anti-Patterns

### Anti-Pattern 1: CLI as a Bag of Shell Scripts

**What people do:** Put GitHub API calls, cloud CLI commands, SSH commands, and parsing directly inside command handlers.

**Why it's wrong:** Hard to retry safely, test, repair, or add providers without breaking flows.

**Do this instead:** Thin CLI, typed orchestration workflows, provider adapters, and checkpointed state.

### Anti-Pattern 2: Persisting Registration Tokens or Printing Secret Commands

**What people do:** Store GitHub registration tokens in config/logs or ask users to copy commands containing tokens.

**Why it's wrong:** Tokens are short-lived secrets; leaks and stale-token failures are likely.

**Do this instead:** Request tokens just in time, pass over secure execution channels, redact everywhere, never persist.

### Anti-Pattern 3: Treating GitHub Online Status as the Only Health Check

**What people do:** `runner online` equals healthy.

**Why it's wrong:** The runner may have wrong labels, full disk, broken Docker, failed service restarts, or stale state.

**Do this instead:** Reconcile GitHub status with remote service, provider instance, labels, logs, disk/network preflight, and last heartbeat.

### Anti-Pattern 4: Promising Ephemeral Autoscaling Without a Controller

**What people do:** Market ephemeral-per-job runners from a one-shot local CLI.

**Why it's wrong:** Something must observe demand and create/register runners when jobs are queued. If the user's CLI is gone, nothing scales.

**Do this instead:** Default to persistent runners; offer ephemeral-on-host for isolation; add a resident controller only when necessary.

### Anti-Pattern 5: Overbuilding Provider Plugins Before One Provider Works Well

**What people do:** Create a broad plugin system and many providers before nailing one happy path.

**Why it's wrong:** The core value is fast, headache-free setup; shallow provider coverage increases support burden.

**Do this instead:** Define provider boundaries early, ship BYO + one excellent cloud provider, then generalize from real implementation pressure.

## Sources

- GitHub Docs: self-hosted runners overview, adding self-hosted runners, labels, service configuration.
- GitHub REST API docs: self-hosted runner endpoints for repositories/organizations, registration tokens, removal tokens, runner list/status/delete.
- GitHub Docs: ephemeral self-hosted runners and autoscaling guidance.
- `actions/runner` repository documentation and runner `config.sh`/service behavior.
- Actions Runner Controller architecture as a reference for future autoscaling fleets, not a v1 default.
- Cloud-init documentation for first-boot provisioning patterns.
- systemd service management and journald logging conventions for Linux hosts.

---
*Architecture research for: RunnerKit CLI-first self-hosted GitHub Actions runner management*
*Researched: 2026-04-28*
