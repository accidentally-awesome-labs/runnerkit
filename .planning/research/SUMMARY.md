# Project Research Summary

**Project:** RunnerKit
**Domain:** CLI-first self-hosted GitHub Actions runner provisioning and lifecycle management
**Researched:** 2026-04-28
**Confidence:** MEDIUM-HIGH

## Executive Summary

RunnerKit is best treated as a local-first infrastructure orchestrator for solo developers, not as a hosted runner platform or enterprise fleet manager. The researched path is a single-binary CLI that wraps GitHub Actions runner registration, remote Linux bootstrap, status/repair, and cleanup behind an opinionated workflow that starts with BYO machines and one low-cost cloud provider.

The strongest v1 approach is GitHub Actions only, Linux/systemd first, repository-scoped runners first, persistent runners as the cheap/trusted default, and ephemeral mode as a carefully bounded safety option rather than full autoscaling. The CLI should print exact labels/workflow snippets and avoid mutating workflow YAML in v1.

The biggest risks are security and lifecycle drift: persistent runners must not silently run untrusted public/fork workflows, registration tokens must never become durable state, and cleanup/status must reconcile local state, GitHub runner records, remote services, and cloud resources. Roadmap phases should build idempotent state, redaction, labels, and cleanup early rather than treating them as polish.

## Key Findings

### Recommended Stack

Build RunnerKit as a Go CLI with a thin command layer and strong internal adapters. Go fits the core need for fast startup, static distribution, SSH/process orchestration, and package-manager-friendly releases without a Node/Python runtime. Use Cobra for command structure, `google/go-github` plus explicit GitHub REST API versioning for runner APIs, Go SSH/system `ssh` fallback for remote execution, and systemd as the v1 service target.

Use Hetzner Cloud as the leading low-cost default cloud candidate, while preserving a provider interface so the cloud choice can change or expand after the first path is reliable. Keep supporting libraries boring: config loading, secure credential references/keychain integration, redacted structured logging, GoReleaser, ShellCheck for generated scripts, API fixtures, and vulnerability/release tooling.

**Core technologies:**
- Go 1.25.x: single-binary CLI and orchestration runtime - best match for fast, cross-platform developer tooling.
- Cobra v1.10.2: CLI command framework - mature fit for `up`, `init`, `status`, `doctor`, `logs`, `destroy`, and provider subcommands.
- `google/go-github` v76.0.0 + GitHub REST API: GitHub runner registration/status/delete APIs - avoids bespoke API plumbing while allowing direct REST calls where needed.
- SSH + systemd: remote bootstrap and service management - shared foundation for BYO and cloud Linux hosts.
- Hetzner Cloud: default low-cost provisioning candidate - strong solo-developer cost story with simple VM semantics.

### Expected Features

**Must have (table stakes):**
- CLI install and guided `runnerkit init/up` flow - users expect setup to be faster than GitHub's manual runner instructions.
- GitHub repository authentication, registration, deregistration, labels, and workflow snippet output - without these the runner is not usable.
- BYO Linux machine bootstrap - covers VPS, homelab, and existing servers with minimal cost.
- One recommended low-cost cloud provisioning path - fulfills the headache-free/cost-effective promise for users without a machine.
- Persistent runner default for trusted private repos plus explicit ephemeral option/profile - gives a cost/security choice without overbuilding autoscaling.
- Runner service management, status, logs, and doctor/recovery - directly addresses the “too fragile” motivation.
- Cleanup/destroy/deregister and local state inventory - prevents stale GitHub records and surprise cloud bills.
- Token/secret safety and quickstart/security/troubleshooting docs - required for trust in a tool that touches CI infrastructure.

**Should have (competitive):**
- Near one-command happy path such as `runnerkit up --repo owner/name --cloud cheap`.
- Opinionated cost-aware profiles and estimates.
- `doctor --fix` automatic repairs after enough failure cases are known.
- Lightweight non-Kubernetes ephemeral lifecycle.
- Read-only workflow readiness/label validation.
- Existing runner import/adopt and additional providers after v1 validation.

**Defer (v2+):**
- Enterprise dashboard, RBAC, SSO, audit logs, and fleet governance.
- Multi-CI support.
- Broad provider matrix.
- Automatic workflow YAML edits/commits.
- Full autoscaling fleets, Kubernetes/ARC-first operation, and hosted control plane.
- Deep secrets management beyond safe handling of RunnerKit/GitHub/provider credentials.

### Architecture Approach

Use a local orchestrator with idempotent, checkpointed workflows. CLI commands should call core workflows; core workflows should use isolated adapters for GitHub, machine providers, remote execution, state, and secrets. BYO SSH and cloud provisioning should converge on the same Linux bootstrap/service path. `status` and `doctor` should reconcile local state, GitHub runner inventory, provider state, SSH reachability, service health, logs, labels, and cost/orphan signals.

**Major components:**
1. CLI command layer - prompts, flags, output, progress, and command routing only.
2. Core orchestrator - idempotent create/reconcile/cleanup/upgrade workflows with rollback/checkpoints.
3. GitHub adapter - auth, repo resolution, short-lived registration/removal tokens, runner inventory/status/delete, workflow hint formatting.
4. Machine provider adapters - BYO SSH and one default cloud provider behind `plan/provision/describe/destroy` semantics.
5. Remote executor/bootstrap installer - SSH/cloud-init readiness, file upload, OS preflight, runner install, systemd service, remote logs.
6. State/config/secrets layer - versioned desired state, runner inventory, provider IDs, SSH fingerprints, secret references, redaction.
7. Health/status reconciler - compares desired and observed state and powers `status`, `doctor`, repair, and cleanup.

### Critical Pitfalls

1. **Running public or untrusted workflows on persistent runners** - default persistent mode must be explicitly private/trusted; block or require loud override for risky public/fork repos.
2. **Mishandling GitHub registration/removal tokens** - request tokens just-in-time, pass them ephemerally, redact them, and re-request on retry; never persist raw tokens.
3. **Secret leakage through logs/state/processes/support bundles** - centralize redaction before debug, remote bootstrap, and diagnostic features ship.
4. **Split-brain runner state and stale registrations** - reconcile local state, GitHub, remote systemd, labels, and cloud provider facts before claiming health or cleanup success.
5. **Cloud resources that keep billing after failures** - tag every cloud resource, show cost before/after provisioning, and ship idempotent destroy/orphan detection with the cloud path.
6. **Naive ephemeral mode** - do not market full autoscaling from a one-shot CLI; scoped ephemeral mode needs one-job lifecycle, log preservation, TTLs, and cleanup finalizers.
7. **Fragile SSH/cloud-init bootstrap** - support a narrow Linux matrix, use preflight checks, retries, timeouts, resumable scripts, and actionable logs.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: CLI Foundation, GitHub Auth, State, and Safety Rails
**Rationale:** Every later flow depends on safe auth, labels, state schema, redaction, command structure, and idempotent workflow primitives.
**Delivers:** Go module/CLI skeleton, config/state schema, redacted logging, GitHub repo auth/permissions, runner registration/removal token abstraction, label conventions, workflow snippet formatting, dry-run/plan skeleton, fake adapters for tests.
**Addresses:** CLI setup, GitHub auth/registration foundation, token/secret safety, labels.
**Avoids:** Broad PAT assumptions, durable registration tokens, generic `self-hosted` routing, unredacted diagnostics.

### Phase 2: BYO Linux Persistent Runner Happy Path
**Rationale:** BYO is the fastest proof of core value and exercises the same remote installer that cloud provisioning will reuse.
**Delivers:** SSH preflight, Linux/systemd bootstrap, dedicated non-root runner user, official runner download/checksum, repo registration, service start/stop/status, logs, exact `runs-on` output, BYO cleanup/unregister.
**Uses:** Go SSH/system SSH fallback, systemd, GitHub runner APIs, versioned local state.
**Implements:** Remote executor, bootstrap installer, runner lifecycle manager.
**Avoids:** Root runner services, unsupported OS ambiguity, manual GitHub setup, duplicate/stale registrations for the same repo/name.

### Phase 3: Status, Doctor, Repair, Cleanup Hardening
**Rationale:** Fragility is one of the core user complaints; diagnostics and cleanup must be product value, not post-MVP polish.
**Delivers:** Multi-source `status`, `doctor`, actionable repair plans, safe restart/re-register/stale-delete flows, redacted support bundle, partial cleanup checkpoints, upgrade metadata fields, disk/Docker/workspace checks.
**Addresses:** Logs/troubleshooting, health/status, recovery from stopped services/offline runners/stale registrations, cleanup reliability.
**Avoids:** Split-brain status, SSH spelunking, persistent contamination blind spots, cleanup that fails when one side is already gone.

### Phase 4: Low-Cost Cloud Provisioning Path
**Rationale:** Cloud provisioning fulfills the cost-effective no-machine path, but should reuse proven BYO bootstrap and cleanup/reconciliation foundations.
**Delivers:** One default provider adapter, likely Hetzner pending implementation validation; plan/cost display; VM/SSH/firewall creation; tagging; SSH/cloud-init readiness; cloud runner install; cloud destroy; orphan detection.
**Uses:** Machine provider interface, remote executor, state/provider IDs, GitHub adapter, cost/profile selection.
**Implements:** Default cloud provider boundary and shared installer reuse.
**Avoids:** Broad provider matrix, untagged billable resources, cloud-only bootstrap divergence, surprise bills.

### Phase 5: Scoped Ephemeral Mode and Safety Profiles
**Rationale:** Ephemeral runners matter for security, but safe one-job lifecycle depends on robust registration, logs, status, and cleanup from earlier phases.
**Delivers:** Explicit mode/profile selection (`cheap-persistent` vs safer ephemeral), public/fork risk gates, ephemeral registration lifecycle, max-runtime/no-job TTLs, log preservation, cleanup finalizers, clear limitations that this is not full autoscaling.
**Addresses:** Persistent vs ephemeral choice, public/untrusted workload safety, contamination mitigation.
**Avoids:** Marketing naive `--ephemeral` as autoscaling, losing logs on destroy, leaking cloud instances after cancellation.

### Phase 6: Release, Upgrade, Docs, and Validation Polish
**Rationale:** The public launch needs installable binaries, reliable upgrades, and docs that preserve the “10-minute first success” promise.
**Delivers:** GoReleaser releases, Homebrew/package artifacts, `runnerkit upgrade` or documented update path, state migrations, quickstart, security guide, troubleshooting recipes, cleanup docs, optional read-only workflow label validation.
**Addresses:** Onboarding, runner/CLI upgrade drift, first-run confidence, support burden.
**Avoids:** Stale runner binaries, state migration breakage, users guessing labels or cleanup commands.

### Phase Ordering Rationale

- GitHub auth, token discipline, labels, state schema, and redaction are dependencies for every mutating flow.
- BYO persistent runner comes before cloud because it proves the runner lifecycle without provider/resource billing complexity.
- Status/doctor/cleanup hardening comes before cloud because cloud failures create billable orphans and harder recovery paths.
- Cloud comes before serious ephemeral work because ephemeral mode needs reliable provisioning, TTLs, logs, and destroy finalizers.
- Provider/plugin generalization should follow real pressure from BYO + one cloud provider, not precede the first reliable path.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1:** GitHub auth permissions, fine-grained PAT/GitHub App/device-flow options, and exact runner REST API version behavior should be verified against official docs.
- **Phase 2:** Official runner package discovery, checksum verification, systemd service behavior, and supported Linux distro/architecture matrix need implementation-time validation.
- **Phase 4:** Hetzner region/instance/networking/firewall/SSH-key defaults, pricing, quotas, and API behavior should be verified before locking the default provider.
- **Phase 5:** Ephemeral runner semantics, job cancellation behavior, log preservation, and the boundary between one-shot ephemeral and autoscaling need targeted validation.
- **Phase 6:** GoReleaser/package manager distribution and runner binary update policy need release-specific checks.

Phases with standard patterns (lower research need during planning):
- **Phase 1 CLI skeleton/state scaffolding:** Common Go/Cobra/config/testing patterns.
- **Phase 3 status/doctor UX structure:** The high-level reconciliation pattern is clear; details come from implemented adapters.
- **Phase 6 documentation structure:** Quickstart/security/troubleshooting docs are straightforward once commands stabilize.

## Confidence Assessment

| Area         | Confidence  | Notes |
| ------------ | ----------- | ----- |
| Stack        | MEDIUM-HIGH | Go/Cobra/GitHub REST/SSH/systemd is a coherent fit; exact versions and API headers still need verification during implementation. |
| Features     | MEDIUM      | Table stakes are clear from domain and project goals; user demand for org support, provider choice, and ephemeral depth needs validation. |
| Architecture | MEDIUM-HIGH | Local orchestrator/adapters/reconciliation are strong patterns; architecture research included a TypeScript-flavored folder example inconsistent with the Go stack, so use it as logical boundaries, not implementation language. |
| Pitfalls     | MEDIUM      | Security/lifecycle risks are well-established for self-hosted runners; exact mitigations depend on chosen auth/provider/ephemeral implementation. |

**Overall confidence:** MEDIUM-HIGH

### Gaps to Address

- **GitHub auth model:** Decide least-privilege v1 auth path and document exact scopes/permissions before exposing setup publicly.
- **Default cloud provider:** Validate Hetzner pricing, regions, quota friction, images, SSH readiness, and target-user availability before making it the default.
- **Ephemeral scope:** Define whether v1 ephemeral means on-host one-job mode, destroy-after-job cloud instances, or a future controller-backed mode; avoid implying full autoscaling without a resident controller.
- **State storage:** Start with versioned JSON only if concurrency and multi-runner needs are small; be ready to move to SQLite for pools/events.
- **Security posture:** Add tests for redaction, root-service prevention, broad PAT warnings, public repo risk gates, and no durable token persistence.
- **Cost story:** Confirm target workloads where a small self-hosted VM is actually cheaper than GitHub-hosted runners after idle time and maintenance.

## Sources

### Primary (HIGH confidence)
- `.planning/PROJECT.md` - project goals, v1 boundaries, audience, and constraints.
- GitHub official docs referenced by research - self-hosted runner registration/removal tokens, labels, runner status, services, security hardening, and ephemeral/autoscaling behavior.
- Official release/docs references in stack research - Cobra, `google/go-github`, GitHub REST API versioning, Bubble Tea, Hetzner ecosystem, GoReleaser.

### Secondary (MEDIUM confidence)
- `.planning/research/STACK.md` - recommended stack, versions, alternatives, and not-to-use guidance.
- `.planning/research/FEATURES.md` - table stakes, differentiators, anti-features, dependencies, and MVP definition.
- `.planning/research/ARCHITECTURE.md` - local orchestrator architecture, data flows, components, state/secrets boundaries, and build order.
- `.planning/research/PITFALLS.md` - critical pitfalls, warning signs, prevention strategies, and phase mapping.
- Competitive landscape references in feature research - GitHub native runner flow, ARC/actions-runner-controller, RunsOn, Cirun, Ubicloud, BuildJet.

### Tertiary (LOW confidence)
- Inferred market demand for exact v1 provider, org-level support, and ephemeral depth - validate with early users and implementation spikes.

---
*Research completed: 2026-04-28*
*Ready for roadmap: yes*
