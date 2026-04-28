# Feature Research

**Domain:** CLI-first self-hosted GitHub Actions runner setup and management for solo developers
**Researched:** 2026-04-28
**Confidence:** MEDIUM

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
| ------- | ------------ | ---------- | ----- |
| CLI installation and guided first-run wizard | The core promise is fast setup without reading GitHub runner docs or stitching scripts together. | MEDIUM | `runnerkit init` should explain choices, validate prerequisites, and end with a runner-ready success state. |
| GitHub repository/org auth and runner registration | A runner tool is unusable unless it can create/refresh GitHub registration tokens and register/deregister runners. | HIGH | Support repository scope first; org scope can be v1.x if needed. Prefer fine-grained token guidance or GitHub device/app flow if available. Never persist raw tokens unnecessarily. |
| BYO machine install path | Solo developers often already have a VPS, spare server, or homelab box and need a reliable bootstrap path. | MEDIUM | Linux-first is enough for v1. Include preflight checks for OS/architecture, disk, Docker, systemd, outbound network, and permissions. |
| Recommended low-cost cloud provisioning path | The project explicitly promises a headache-free path for users without a machine. | HIGH | Pick one provider for v1; provision VM, SSH access, firewall defaults, runner user, dependencies, and bootstrap script. Keep provider abstraction thin until validated. |
| Runner service management | Users expect the runner to survive reboots and be startable/stoppable without manual service commands. | MEDIUM | Systemd service install/start/stop/restart/status should be wrapped by the CLI. |
| Runner labels and workflow snippet guidance | Developers need to know exactly what `runs-on` labels to use after registration. | LOW | Register predictable labels such as `self-hosted`, `runnerkit`, provider/mode/arch labels; print copy-paste YAML but do not edit workflow files in v1. |
| Persistent vs ephemeral mode choice with safe default | Users need a cost/security tradeoff decision; persistent runners are cheap/simple, ephemeral runners reduce cross-job contamination. | HIGH | Default should likely be persistent for trusted private solo repos, with strong warnings for public/untrusted jobs and an explicit ephemeral option/profile. |
| Health/status command | Users need to know whether the runner is online, registered, idle/busy, and able to accept jobs. | MEDIUM | Combine local service state, last heartbeat/log signal, GitHub runner listing/status, machine resource checks, and label display. |
| Logs and troubleshooting access | Self-hosted runner pain is often opaque failures; a tool must surface actionable logs. | MEDIUM | `runnerkit logs`, `runnerkit doctor`, and common remediation hints for offline runners, expired registration, permission errors, Docker issues, and disk pressure. |
| Update/upgrade flow | GitHub runner binaries and the CLI need safe upgrades; stale runners become fragile. | MEDIUM | Separate `runnerkit upgrade` for CLI/provider state and runner binary/service updates; show current/available versions where possible. |
| Cleanup/destroy/deregister | Users must be able to stop paying for infrastructure and remove stale GitHub runners. | HIGH | Deregister GitHub runner, stop service, remove local files, optionally destroy cloud resources, and handle partial cleanup idempotently. |
| Local state/config inventory | Management commands need to know what was provisioned, labels used, provider IDs, runner names, and repo scope. | MEDIUM | Use a readable local state file; make destructive commands show a plan before execution. |
| Token/secret safety | CI runners handle repository code and secrets; careless token storage undermines trust. | MEDIUM | Short-lived registration tokens, minimal GitHub scopes, masked logs, warnings for public repos/forks, and no secrets in plain command output. |
| Quickstart docs and onboarding | The target user wants speed; docs must answer “what command do I run?” and “what label do I put in workflow YAML?” | LOW | Include 10-minute quickstart, BYO path, cloud path, mode selection guide, cleanup guide, and troubleshooting recipes. |

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valuable.

| Feature | Value Proposition | Complexity | Notes |
| ------- | ----------------- | ---------- | ----- |
| Near one-command happy path | Makes the product feel materially easier than GitHub's manual setup flow. | HIGH | Example: `runnerkit up --repo owner/name --cloud cheap` provisions, installs, registers, prints labels, and verifies readiness. |
| Opinionated cost-aware defaults | Aligns directly with the “too expensive” motivation. | MEDIUM | Recommend instance size/provider profile by workload; show approximate monthly/hourly cost and cleanup command. |
| “Doctor” with automatic repair actions | Converts runner fragility into guided recovery instead of docs spelunking. | HIGH | Detect and optionally fix stopped service, stale GitHub registration, bad labels, missing Docker, disk full, or broken runner directory. |
| Lightweight ephemeral mode without Kubernetes | Gives solo developers safer clean-per-job behavior without adopting actions-runner-controller/ARC. | HIGH | Could use provider APIs/webhook/polling to create a short-lived machine per job or a short-lived runner per run; defer broad autoscaling. |
| Profile-based setup | Helps non-infra users choose without understanding all runner tradeoffs. | MEDIUM | Profiles like `cheap-persistent`, `safe-ephemeral`, `docker-builds`, `arm64`, `gpu` later. |
| Workflow readiness check | Reduces first-run confusion by validating labels and showing exactly why a job would or would not match. | MEDIUM | Inspect repo runner labels and optionally a workflow file path read-only; do not auto-edit in v1. |
| Cost controls and idle guardrails | Prevents self-hosted runners from becoming surprise bills. | MEDIUM | Idle shutdown schedules, max monthly estimate warnings, destroy reminders, and orphaned resource detection. |
| Existing runner import/adopt | Lowers adoption friction for developers with already-manual runners. | MEDIUM | Discover service/config, normalize labels, write state, and enable status/log/update/cleanup management. |
| Portable project-local recipes | Makes RunnerKit repeatable across repos/machines. | MEDIUM | A `runnerkit.yml` or generated setup summary can document desired provider/mode/labels without storing secrets. |
| Security posture explanations in the CLI | Builds trust with solo developers who may not know self-hosted runner risks. | LOW | Explain when persistent runners are acceptable and when public forks/untrusted PRs require ephemeral isolation or should be avoided. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
| ------- | ------------- | --------------- | ----------- |
| Enterprise dashboard, RBAC, SSO, audit logs | Looks credible and “complete.” | Pulls the product away from solo developers and adds hosted-app/security/compliance scope. | CLI-first local management; defer enterprise controls until a different audience is validated. |
| Multi-CI support in v1 | Broadens marketable surface area. | Dilutes the GitHub Actions-specific setup, labels, registration, and troubleshooting experience. | Make GitHub Actions excellent first; design internal boundaries so other CI integrations can come later. |
| Automatic workflow YAML editing/commits | Saves one more manual step. | Risky, surprising, easy to break custom workflows, and explicitly out of v1 scope. | Print exact `runs-on` snippet and optionally perform read-only validation. |
| Broad provider matrix on day one | Users ask for their preferred cloud. | Every provider multiplies provisioning, firewall, SSH, cleanup, image, quota, and cost edge cases. | One excellent low-cost default plus BYO; add providers after demand is clear. |
| Kubernetes/ARC-first architecture | ARC is powerful and industry-recognized for autoscaling runners. | Too heavy for solo developers who want a quick cheap runner, not a cluster to operate. | Lightweight VM/BYO path first; document when ARC is a better fit. |
| Hosted control plane/account requirement | Enables easier telemetry and remote orchestration. | Undermines “self-hosted,” increases trust burden, and adds SaaS operations. | Local CLI/state by default; optional remote features only after validation. |
| Running untrusted public PRs on persistent runners by default | Users want maximum job compatibility at lowest cost. | Persistent self-hosted runners can leak state/secrets or be abused by malicious code. | Default warnings and safer ephemeral/isolation guidance for public or fork-heavy repos. |
| Full autoscaling fleet manager in v1 | Sounds like the ultimate runner product. | High operational complexity and overlaps with ARC/Cirun/RunsOn before core solo value is proven. | Start with one runner or small pool, then add bounded scaling once setup/status/cleanup are reliable. |
| Building a custom CI executor | Gives control over job execution semantics. | GitHub Actions already provides runner protocol and workflow semantics; custom execution is a huge product. | Wrap official runner setup and lifecycle instead of replacing GitHub Actions. |
| Deep secrets management | CI security is important. | Re-implementing vault/secrets workflows is non-core and dangerous. | Rely on GitHub Actions secrets and provider-native credentials; handle RunnerKit tokens minimally and safely. |

## Feature Dependencies

```text
CLI install + init wizard
    ├──requires──> local state/config inventory
    ├──requires──> docs/onboarding copy
    ├──drives────> BYO install path
    └──drives────> cloud provisioning path

GitHub auth
    ├──requires──> token/secret safety
    ├──enables───> runner registration
    ├──enables───> runner deregistration/cleanup
    └──enables───> GitHub-side status checks

BYO install path
    ├──requires──> preflight checks
    ├──requires──> SSH/local bootstrap
    └──enables───> runner service management

Cloud provisioning path
    ├──requires──> provider credentials
    ├──requires──> cost/profile selection
    ├──requires──> SSH/cloud-init bootstrap
    └──enables───> cleanup/destroy

Runner registration
    ├──requires──> GitHub auth
    ├──requires──> runner service management
    ├──enables───> labels + workflow snippet
    └──enables───> health/status

Persistent mode
    ├──requires──> service management
    ├──enhanced by──> update/upgrade
    └──conflicts with──> untrusted public PR safety as a default

Ephemeral mode
    ├──requires──> registration/deregistration automation
    ├──requires──> cleanup/destroy idempotency
    ├──requires──> job/run trigger strategy or clear manual invocation
    └──enhanced by──> cost controls and provisioning profiles

Health/status
    ├──requires──> local state/config
    ├──requires──> service/log access
    └──requires──> GitHub runner status/listing

Doctor/repair
    ├──requires──> health/status
    ├──requires──> logs/troubleshooting knowledge
    └──enhanced by──> cleanup and re-registration flows
```

### Dependency Notes

- **Runner registration requires GitHub auth:** Registration tokens are scoped and short-lived; the CLI must obtain or guide the user through obtaining them before installing the runner.
- **Status requires both local and GitHub-side checks:** A systemd service can be running while GitHub sees the runner offline, or GitHub can list a stale runner after local deletion.
- **Cloud cleanup requires provider state:** Without provider IDs and region/instance metadata, `destroy` becomes manual and fragile.
- **Ephemeral mode requires stronger lifecycle automation than persistent mode:** Each job/run needs fresh registration, execution, deregistration, and machine/working-directory cleanup.
- **Logs enable repair:** Doctor-style fixes should not guess; they should be backed by service status, runner logs, GitHub API status, and preflight results.
- **Labels depend on mode/provider choices:** Labels should communicate capabilities (`linux`, `x64`, `docker`, `gpu`, `runnerkit`, `persistent`/`ephemeral`) without becoming unstable per-run identifiers.
- **Automatic workflow editing conflicts with trust/simplicity:** Read-only validation and printed snippets satisfy onboarding without mutating repositories.

## MVP Definition

### Launch With (v1)

Minimum viable product - what's needed to validate the concept.

- [ ] CLI install and `runnerkit init/up` guided setup - proves the product can beat manual GitHub runner setup.
- [ ] GitHub repository auth/registration/deregistration - essential to create usable runners and clean stale registrations.
- [ ] BYO Linux machine bootstrap - covers users with existing cheap infrastructure.
- [ ] One recommended low-cost cloud provider path - covers users who need a machine and validates the cost-effectiveness promise.
- [ ] Persistent runner mode as the default for trusted private solo repos - lowest complexity and cost for the first happy path.
- [ ] Ephemeral mode option/profile with clear limitations - satisfies security/workload choice without making autoscaling the whole v1.
- [ ] Runner labels plus copy-paste workflow snippet - lets users actually route jobs to RunnerKit runners.
- [ ] `runnerkit status` and `runnerkit doctor` - addresses the “too fragile” complaint directly.
- [ ] `runnerkit logs` - makes failures explainable without SSH spelunking.
- [ ] `runnerkit upgrade` or documented update command - prevents immediate rot of runner installs.
- [ ] `runnerkit cleanup/destroy` - removes GitHub registrations and cloud/BYO installs safely.
- [ ] Quickstart, mode-selection, security, and troubleshooting docs - reduces support burden and improves first-run success.

### Add After Validation (v1.x)

Features to add once core is working.

- [ ] Cost estimates, idle shutdown, and orphan detection - add when users validate the cloud path and cost sensitivity.
- [ ] Automatic repair actions in `doctor --fix` - add after enough failure cases are known.
- [ ] Import/adopt existing runners - add when manual-runner users want management without reinstalling.
- [ ] Org-level runner support - add if users need multiple repositories beyond the first repo-level workflow.
- [ ] Read-only workflow label validation - add after labels are stable; still avoid auto-editing.
- [ ] Additional Linux architectures or images - add based on demand for arm64, larger CPUs, Docker-heavy builds, or GPUs.
- [ ] Provider plugin boundary and second provider - add only after the first provider path is reliable.

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] Full autoscaling pools - defer until one/few-runner lifecycle is proven.
- [ ] Web dashboard - defer because v1 value is CLI speed and low operational overhead.
- [ ] Team/enterprise governance - defer until audience expands beyond solo developers.
- [ ] Multi-CI integrations - defer until GitHub Actions setup is excellent.
- [ ] Kubernetes/ARC integration mode - useful for advanced users but too heavy for the initial solo developer path.
- [ ] Hosted control plane - defer until there is a compelling reason to trade self-contained CLI simplicity for remote orchestration.

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
| ------- | ---------- | ------------------- | -------- |
| CLI install and guided setup | HIGH | MEDIUM | P1 |
| GitHub auth and runner registration | HIGH | HIGH | P1 |
| BYO Linux bootstrap | HIGH | MEDIUM | P1 |
| One default cloud provisioning path | HIGH | HIGH | P1 |
| Runner service management | HIGH | MEDIUM | P1 |
| Labels and workflow snippet | HIGH | LOW | P1 |
| Persistent runner default | HIGH | MEDIUM | P1 |
| Ephemeral mode option | HIGH | HIGH | P1 |
| Health/status | HIGH | MEDIUM | P1 |
| Logs and troubleshooting | HIGH | MEDIUM | P1 |
| Cleanup/destroy/deregister | HIGH | HIGH | P1 |
| Update/upgrade | MEDIUM | MEDIUM | P1 |
| Quickstart and security docs | HIGH | LOW | P1 |
| Cost estimates/idle controls | HIGH | MEDIUM | P2 |
| Doctor auto-repair | HIGH | HIGH | P2 |
| Import existing runner | MEDIUM | MEDIUM | P2 |
| Org-level support | MEDIUM | MEDIUM | P2 |
| Additional providers | MEDIUM | HIGH | P2/P3 |
| Web dashboard | LOW | HIGH | P3 |
| Enterprise governance | LOW | HIGH | P3 |
| Multi-CI support | LOW | HIGH | P3 |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | GitHub native self-hosted runner flow | ARC / actions-runner-controller | Managed/BYOC runner services such as RunsOn, Cirun, Ubicloud, BuildJet | Our Approach |
| ------- | ------------------------------------ | ------------------------------- | --------------------------------------------------------------- | ------------ |
| Setup flow | Manual download/config scripts and service setup. | Kubernetes controller/scale-set setup; powerful but cluster-oriented. | Usually guided docs plus hosted account/provider integration. | CLI-first wizard and one-command happy path for solo developers. |
| GitHub registration | Official mechanism; user usually copies tokens/commands. | Automates registration through controller credentials. | Abstracted through service-specific integration. | Automate repo registration/deregistration while keeping tokens minimal and transparent. |
| BYO machine support | Possible but manual. | Requires Kubernetes or runner pods. | Varies; often cloud-account oriented. | First-class BYO Linux bootstrap with status/logs/cleanup. |
| Cloud provisioning | Not provided directly. | Uses Kubernetes cluster capacity, not simple VM provisioning. | Often provider-specific or hosted capacity. | One low-cost default provider path for v1, designed for later providers. |
| Persistent runners | Supported, but user manages lifecycle. | Possible but usually oriented toward scalable runner sets. | Often hidden/managed. | Simple trusted-private-repo default with explicit tradeoff guidance. |
| Ephemeral runners | Supported conceptually; automation is left to user/tooling. | Strong support for ephemeral/autoscaled runners. | Common managed offering. | Lightweight ephemeral option without requiring Kubernetes or a hosted control plane. |
| Labels | Supported; user must manage consistency. | Managed through runner scale set config. | Service-specific labels. | Predictable RunnerKit labels plus copy-paste workflow snippets and validation. |
| Health/status/logs | GitHub UI plus local service logs. | Kubernetes/GitHub/controller observability. | Dashboard/hosted UI. | CLI `status`, `logs`, and `doctor` focused on common solo-developer failures. |
| Cleanup | Manual deregistration and file/service cleanup. | Controller-managed if configured correctly. | Service-managed, but can be tied to account/provider setup. | Idempotent local/provider/GitHub cleanup as a core feature. |
| Target user | Anyone willing to manage runners manually. | Teams/platform engineers with Kubernetes. | Users willing to adopt a service/control plane. | Solo developers who want cheap, quick, self-hosted GitHub Actions runners. |

## Sources

- `.planning/PROJECT.md` project context and explicit v1 constraints.
- GitHub Actions self-hosted runner domain knowledge: runner registration, labels, persistent/ephemeral runner modes, service installation, and status concepts.
- Competitive landscape knowledge: GitHub native runner setup, actions-runner-controller/ARC, and managed/BYOC GitHub Actions runner services such as RunsOn, Cirun, Ubicloud, and BuildJet.
- User problem statement from initialization: current setup is too manual, too fragile, and too expensive.

---
*Feature research for: CLI-first self-hosted GitHub Actions runner setup and management for solo developers*
*Researched: 2026-04-28*
