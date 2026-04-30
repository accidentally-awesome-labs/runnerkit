# Phase 4: Recommended Cloud Path and Billable Cleanup - Context

**Gathered:** 2026-04-30
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 4 adds one recommended low-cost cloud machine path for developers who do not already have a runner host. RunnerKit should provision the cloud machine and required access resources, wait until it is ready, then install/register/manage the runner through the same GitHub, SSH, bootstrap, status, logs, doctor, and cleanup lifecycle already proven for BYO machines. The phase must also add billable cloud destruction so users can remove RunnerKit-created cloud resources with confidence.

This phase stays inside one recommended persistent cloud runner path plus safe billable cleanup. It does **not** add a broad provider matrix, hosted control plane, autoscaling fleet manager, ephemeral mode, automatic workflow YAML edits, or general cost-control/idle-shutdown policy beyond the cost visibility and destroy safeguards required for the cloud path.

</domain>

<decisions>
## Implementation Decisions

### Recommended provider and profile

- **D-01:** The exact cloud provider and default profile are the agent/planner's discretion after targeted Phase 4 research. Do not lock the implementation to Hetzner or DigitalOcean from discussion alone; research should validate current pricing, regions, quota friction, API maturity, SSH readiness, and failure modes before choosing the single recommended path.
- **D-02:** Optimize the selected default for smooth setup and reliability first, while keeping it visibly low-cost. Prefer the provider/profile/region/image combination with fewer credential, quota, networking, image, and SSH-readiness surprises over the absolute cheapest instance if the cheapest path is brittle.
- **D-03:** The provisioning plan and docs should show approximate hourly and monthly cost estimates with clear caveats that provider pricing varies by region/time and that estimates are informational.
- **D-04:** Cloud credentials should be discovered from provider CLI authentication or environment variables first. RunnerKit should not persist provider API tokens. If credentials are missing or insufficient, fail before mutation and print exact setup steps for the chosen provider.

### Cloud setup UX

- **D-05:** Keep `runnerkit up` as the primary setup command. In interactive mode, when no `--host`/BYO target is provided, the wizard should offer BYO vs the recommended cloud path rather than requiring a separate command or silently provisioning cloud resources.
- **D-06:** Non-interactive cloud provisioning must require explicit cloud intent flags plus `--yes` before creating billable resources. A missing `--host` plus `--yes` should not be enough to provision cloud infrastructure.
- **D-07:** Before creating cloud resources, show a provisioning plan that includes the resources to create, estimated cost, resource names/tags/identity, SSH key/firewall/network shape, labels and `runs-on` snippet preview, and the exact `runnerkit destroy` command that will later remove billable resources.
- **D-08:** Provider credential, quota, region, or readiness failures should stop before any provider/GitHub/SSH/bootstrap mutation. The wizard should give exact fix steps and ask the user to rerun after prerequisites are fixed, instead of branching into provider troubleshooting after partial mutation.

### Cloud quickstart expectations

- **D-09:** The cloud quickstart should optimize for fast first successful cloud runner setup, not for exhaustive reference coverage. Keep the happy path concise and link or separate deeper troubleshooting/reference material.
- **D-10:** The quickstart must include provider auth setup, cost estimate explanation, the cloud `runnerkit up` flow, exact `runs-on` label guidance, status/logs/doctor lifecycle reuse, and `runnerkit destroy` cleanup/verification.
- **D-11:** Cost language in docs should be explicit that estimates are approximate and user-responsible: pricing can change and billing stops only when the relevant provider resources are actually destroyed or non-billable.
- **D-12:** The quickstart should clearly call out Phase 4 limitations: one recommended provider path, persistent trusted/private-repository default, ephemeral mode comes later in Phase 5, and RunnerKit still does not edit workflow YAML.

### Cloud resource identity and readiness

- **D-13:** RunnerKit should save a full managed cloud resource inventory in state after provisioning: provider kind, VM/server ID and name, region, image, size/profile, SSH key ID/name, firewall/network IDs, public host/IP, tags, and cleanup resource IDs. The existing `ProviderRef`, `MachineRef`, `CleanupMetadata.ProviderResourceIDs`, and operation checkpoint fields can be extended as needed.
- **D-14:** Every RunnerKit-created cloud resource should be named/tagged predictably with ownership metadata such as `runnerkit`, repo slug, runner name or state ID, mode, created-at, and managed=true. Destroy/status should use these tags/IDs to identify ownership and avoid deleting unrelated resources.
- **D-15:** A newly provisioned VM is not ready for runner registration/bootstrap until the provider reports it running, SSH is reachable with host-key verification, boot/cloud-init readiness is satisfied when available, and the shared BYO preflight passes. Do not request/use a GitHub registration token before this readiness gate.
- **D-16:** Provider facts should appear in `status`, `doctor`, and `destroy`, not only in hidden local state. Human and JSON output should include provider instance state, region/profile, public host/IP, billable-resource summary, and drift/orphan warnings where available.

### Billable destroy semantics

- **D-17:** Use `runnerkit destroy` for cloud resource cleanup. Keep `runnerkit down` as the BYO runner cleanup command established in Phase 3.
- **D-18:** `runnerkit destroy` should show a plan before mutation, including GitHub registration, remote runner/service cleanup, provider resources, local state changes, and billing impact. Interactive mode should require explicit confirmation for billable resource removal; `destroy --yes` should apply a safe default plan.
- **D-19:** `destroy` should not declare success until GitHub registration is removed or absent and RunnerKit-created provider resources are verified deleted or demonstrably non-billable. A provider API delete response alone is not enough if billable resources may still exist.
- **D-20:** If destroy partially fails, keep local state with pending cleanup checkpoints and enough provider/GitHub/remote identity to resume. Do not remove state or hide failures while billable resources may remain.

### the agent's Discretion

- Exact provider and profile selection after Phase 4 research, bounded by the reliability-first and low-cost decisions above.
- Exact cloud flags, provider/profile names, and JSON field names, provided non-interactive cloud provisioning requires explicit cloud intent.
- Exact cost-estimation source and wording, provided hourly/monthly estimates and caveats are visible before provisioning.
- Exact provider adapter API/client implementation and tag key names, provided every managed resource is identifiable and safe to reconcile/destroy.
- Exact human output formatting for provisioning plans, provider facts, and destroy verification, provided the required information is present in both human and JSON outputs where applicable.

</decisions>

<canonical_refs>

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements

- `.planning/ROADMAP.md` §"Phase 4: Recommended Cloud Path and Billable Cleanup" - Fixed phase goal, success criteria, and four planned work slices for provider interface/profile, VM/SSH/firewall/readiness, shared cloud runner install, destroy, billing verification, and cloud quickstart.
- `.planning/REQUIREMENTS.md` §"Machine Setup" - `MACH-03`, `MACH-04`, and `MACH-05` define cloud provisioning, shared lifecycle install, and provider/machine identity requirements.
- `.planning/REQUIREMENTS.md` §"Cleanup and State" - `CLEAN-01`, `CLEAN-04`, `STATE-01`, and `STATE-02` define cleanup plan-before-destroy, cloud resource destruction/billing verification, state metadata, and redaction constraints.
- `.planning/REQUIREMENTS.md` §"Documentation and Safety" - `DOC-02` defines the concise cloud quickstart requirement.
- `.planning/REQUIREMENTS.md` §"Reliability and Operations" - `REL-01`, `REL-02`, and `REL-03` are already implemented for BYO and must be extended/reused for cloud-managed runners.
- `.planning/PROJECT.md` §"Active", §"Out of Scope", §"Constraints", and §"Key Decisions" - One recommended cloud path, CLI-only v1, solo-developer focus, no workflow YAML edits, broad provider matrix out of scope, and `down` vs `destroy` distinction.
- `.planning/STATE.md` §"Accumulated Context" and §"Blockers/Concerns" - Current completed BYO/ops decisions and the Phase 4 warning to validate default cloud provider cost, availability, quota friction, and SSH readiness.

### Prior phase decisions and completed implementation context

- `.planning/phases/01-cli-auth-state-and-safety-foundation/01-CONTEXT.md` §"Implementation Decisions" - `runnerkit up`, guided CLI + automation flags/JSON, plan/checklist before mutation, GitHub auth safety, public repo gate, state/config split, labels, and redaction.
- `.planning/phases/02-byo-persistent-runner-happy-path/02-CONTEXT.md` §"Implementation Decisions" and §"Existing Code Insights" - BYO `runnerkit up`, SSH host-key trust, preflight-before-mutation, non-root service, remote progress, labels/snippet, and state integration points that the cloud path should reuse.
- `.planning/phases/03-operations-diagnostics-and-byo-cleanup/03-CONTEXT.md` §"Implementation Decisions" and §"Existing Code Insights" - Status/logs/doctor/recover/down lifecycle, read-only status model, safe BYO cleanup plan, and `destroy` reserved for cloud billable cleanup.
- `.planning/phases/03-operations-diagnostics-and-byo-cleanup/03-VERIFICATION.md` - Verification results for the completed BYO operations/cleanup lifecycle that Phase 4 should extend without regressing.

### Research guidance for cloud provider and lifecycle

- `.planning/research/SUMMARY.md` §"Phase 4: Low-Cost Cloud Provisioning Path" and §"Research Flags" - Research-backed Phase 4 direction and explicit need to validate Hetzner/default-provider pricing, regions, quota friction, networking/firewall, SSH-key defaults, and API behavior.
- `.planning/research/ARCHITECTURE.md` §"First-Run Cloud Provisioning Flow" - Expected cloud setup data flow from repo/provider/profile selection through provider provision, SSH readiness, BYO bootstrap, GitHub registration, state, and labels.
- `.planning/research/ARCHITECTURE.md` §"MachineProvider Contract" and §"Plugin / Provider Boundaries" - Required provider methods and separation between machine lifecycle, GitHub registration, and remote bootstrap.
- `.planning/research/ARCHITECTURE.md` §"Cleanup Flow" and §"State Management" - Provider/BYO ownership markers, resumable mutations, provider IDs, and destroy vs BYO cleanup responsibilities.
- `.planning/research/ARCHITECTURE.md` §"Remote Install / Provisioning Strategy" and §"Preflight Checks" - Shared cloud/BYO SSH bootstrap, non-root runner user, systemd Linux-first install, and readiness/preflight categories.
- `.planning/research/FEATURES.md` §"Table Stakes" rows for recommended low-cost cloud provisioning, cleanup/destroy/deregister, local state/config inventory, token/secret safety, and quickstart docs.
- `.planning/research/FEATURES.md` §"Feature Dependencies" - Cloud provisioning depends on provider credentials, cost/profile selection, SSH/cloud-init bootstrap, and cleanup/destroy.
- `.planning/research/PITFALLS.md` §"Pitfall 7", §"Pitfall 8", §"Pitfall 6", and §"Pitfall 5" - Surprise cloud bills, fragile SSH/cloud-init/bootstrap flows, split-brain runner state, and secret leakage risks.
- `.planning/research/STACK.md` §"Recommended Stack", §"If default cloud provisioning", and §"Alternatives Considered" - Go/Cobra/GitHub/SSH/systemd baseline, Hetzner as a leading candidate, DigitalOcean as an alternative, and avoid Terraform/broad provider matrix for v1.

### User-facing docs to extend

- `README.md` §"BYO persistent runner quickstart" and §"BYO operations" - Current top-level docs pattern that should gain the recommended cloud path and destroy guidance.
- `docs/byo-quickstart.md` - Existing quickstart structure and lifecycle language to mirror or cross-link from the cloud quickstart.

</canonical_refs>

<code_context>

## Existing Code Insights

### Reusable Assets

- `internal/cli/root.go`: Cobra command tree and injectable `Dependencies` are the place to add provider dependencies and a new `destroy` command while preserving testability and fake adapters.
- `internal/cli/up.go`: Existing `runnerkit up` flow resolves repos, checks GitHub auth/safety, resolves BYO targets, verifies SSH host keys, runs preflight, creates just-in-time registration tokens, applies bootstrap, waits for GitHub online status, and saves state. Phase 4 should branch into provider provisioning before remote target/preflight, then reuse the shared BYO install path.
- `internal/state/schema.go`: Existing `MachineRef`, `ProviderRef`, `CleanupMetadata.ProviderResourceIDs`, and `OperationCheckpoint` fields already anticipate provider IDs, resource references, machine identity, and resumable cleanup. They will likely need extensions for full cloud inventory and provider facts.
- `internal/remote/executor.go`: Remote executor boundary can be reused once a provider returns an SSH target for the cloud VM.
- `internal/preflight/checks.go`: BYO preflight checks (SSH, host key, OS/arch, systemd, sudo, disk, tools, GitHub network, time, runner conflict) should remain the readiness gate after cloud provisioning.
- `internal/bootstrap/install.go` and related bootstrap code: Existing non-root runner installation/service setup should be reused for cloud VMs rather than creating a separate cloud-only runner installer.
- `internal/workflow/plan.go`: Existing plan/checkpoint primitives can grow cloud provisioning/destroy plans and pending cleanup checkpoints.
- `internal/cli/status.go` and `internal/ops/status.go`: Current status reconciles local state, GitHub, SSH, service, and labels. Phase 4 should add provider facts and cloud drift/orphan signals.
- `internal/cli/down.go` and `internal/ops/cleanup.go`: BYO cleanup plan, artifact selection, idempotent remote/GitHub cleanup, and partial-state behavior are the model for `runnerkit destroy`, but cloud destroy must add provider resources and non-billable verification.
- `internal/ui/output.go`: Existing human/JSON rendering and redaction should be reused for cloud plan, cost, provider facts, and destroy output.
- `README.md` and `docs/byo-quickstart.md`: Existing docs provide the pattern for a concise quickstart plus lifecycle commands.

### Established Patterns

- Thin Cobra command handlers with injectable services and tests/fakes.
- Human-first interactive wizard with explicit flags, `--yes`, `--non-interactive`, `--json`, and dry-run behavior for automation.
- Plan/checklist before mutation and fail-closed behavior for safety-critical conditions.
- Versioned, secret-free local state with centralized redaction before output.
- Just-in-time GitHub registration/removal token creation; no durable token storage.
- Shared labels and exact copy-paste `runs-on` snippets; no workflow YAML edits.
- Read-only status; deeper diagnostics in doctor/logs; mutating cleanup/recovery behind explicit commands.
- Safe cleanup keeps enough state to resume when partial failures occur.

### Integration Points

- `runnerkit up` should introduce cloud provider/profile selection and provider prerequisite validation before remote target readiness and shared bootstrap.
- A new provider adapter boundary should own cloud credential validation, plan/cost/resource display, provision, wait/describe, tag/name, and destroy operations. It should return normalized machine/provider refs for existing remote/GitHub/bootstrap workflows.
- State persistence must expand from BYO host facts to full provider resource identity, tags, and cleanup resource IDs.
- `runnerkit status`, `runnerkit doctor`, and JSON output should reconcile provider observed state alongside GitHub, SSH, service, and labels for cloud-managed runners.
- `runnerkit destroy` should orchestrate GitHub deregistration/removal, remote runner cleanup where reachable, provider resource destruction, verification, and local-state update/checkpoints.
- The cloud quickstart should sit beside or link from existing BYO docs and the README, emphasizing auth, cost, setup, labels, operations, and destroy verification.

</code_context>

<specifics>
## Specific Ideas

- The provider choice is intentionally not locked in discussion; downstream research should make the call using current evidence, with reliability/smooth setup weighted above absolute cheapest price.
- Interactive `runnerkit up` should guide a user without a host into the recommended cloud path, while non-interactive mode must make cloud intent explicit to avoid accidental bills.
- The provisioning plan should teach cleanup up front by printing the exact future `runnerkit destroy` command before any billable resources are created.
- The quickstart should get a user to a first cloud runner quickly, but it must include cost caveats and destroy verification because this phase introduces billable resources.

</specifics>

<deferred>
## Deferred Ideas

No new deferred ideas came up during discussion. Existing roadmap deferrals still apply: additional cloud providers or provider plugin hardening after the first path is reliable, ephemeral mode in Phase 5, and richer cost controls/idle shutdown/orphan detection beyond the required Phase 4 destroy verification in later work.

</deferred>

---

_Phase: 04-recommended-cloud-path-and-billable-cleanup_
_Context gathered: 2026-04-30_
