# Phase 2: BYO Persistent Runner Happy Path - Context

**Gathered:** 2026-04-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 2 delivers RunnerKit's first real runner setup path for a solo developer who already has a Linux machine: connect over SSH, run remote preflight checks, bootstrap a dedicated non-root persistent GitHub Actions runner service, register it to the selected repository with RunnerKit labels, verify the service/registration enough to declare the happy path complete, and print clear next-step workflow guidance.

This phase stays inside the BYO persistent happy path. It does **not** add cloud provisioning, broad cleanup/recovery/doctor workflows, ephemeral mode, hosted dashboards, organization-level runner management, or automatic workflow YAML edits. Those belong to later roadmap phases or remain out of scope for v1.

</domain>

<decisions>
## Implementation Decisions

### SSH target and preflight strictness

- **D-01:** Extend the existing `runnerkit up` flow for BYO rather than creating a separate primary command. The happy path should support automation-friendly flags such as `--repo owner/name` and `--host user@host`, while still working as a guided interactive setup when details are missing.
- **D-02:** Make host selection wizard-first in interactive mode. The user wants RunnerKit to ask for host/user/port/key details as part of the guided setup, with flags available for direct/non-interactive use.
- **D-03:** Use explicit SSH host-key trust. For an unknown host, show the fingerprint, ask the user to accept it, record the accepted fingerprint in RunnerKit state, and fail closed if a later connection sees a mismatched fingerprint.
- **D-04:** The exact privilege flow is the agent/planner's discretion, but the runner service must never run as root by default. The expected posture is elevated privileges only for installation/service setup when needed, then a dedicated unprivileged runner user for GitHub Actions jobs.
- **D-05:** Support common systemd Linux hosts on a best-effort basis rather than limiting the product copy to only Ubuntu/Debian x64. Unknown or unverified distros should show a clear warning, require an explicit override, and then try the best-effort path.
- **D-06:** Run a full preflight checklist before mutating the remote host. At minimum, check SSH connectivity, accepted fingerprint, OS/architecture, systemd, sudo/install ability, disk capacity, required tools, time sync, and outbound HTTPS access to GitHub/runner downloads.
- **D-07:** When preflight finds a missing prerequisite RunnerKit can install or fix, show a fix plan and ask before applying it. Preserve the Phase 1 pattern of plan/checklist before mutation.
- **D-08:** Report remote progress step-by-step with actionable failure copy. Default output should identify the current remote check/install category, redact remote output, and end failures with exact next actions; verbose raw-ish SSH/install details can be optional but should not be the default.

### the agent's Discretion

- Exact flag names beyond the expected `runnerkit up` + BYO host shape.
- Exact SSH implementation choice (`golang.org/x/crypto/ssh`, system `ssh`, or hybrid), provided normal developer SSH workflows are not made unnecessarily painful.
- Exact sudo/no-sudo/root-login handling, as long as the persistent runner service is non-root by default and privilege risks are explicit.
- Exact list of “common systemd Linux” distros and architecture support after Phase 2 research validates official runner package availability and bootstrap risk.
- Bootstrap/service defaults, duplicate runner handling, GitHub runner ID persistence, final completion summary copy, and BYO quickstart structure were not separately discussed; planners should use roadmap requirements, prior Phase 1 decisions, and research docs for those choices.

</decisions>

<canonical_refs>

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements

- `.planning/ROADMAP.md` §"Phase 2: BYO Persistent Runner Happy Path" - Fixed phase goal, success criteria, and four planned work slices for SSH preflight, bootstrap, registration, safety warnings, smoke test, and BYO quickstart.
- `.planning/REQUIREMENTS.md` §"CLI Onboarding" - `CLI-03` and `CLI-04` for the 10-minute happy path and completion summary.
- `.planning/REQUIREMENTS.md` §"GitHub Integration" - `GH-02`, `GH-04`, and `GH-05` for automated repository-scoped registration, predictable labels, and copy-paste `runs-on` guidance without editing workflows.
- `.planning/REQUIREMENTS.md` §"Machine Setup" - `MACH-01` and `MACH-02` for SSH preflight and BYO Linux bootstrap with non-root runner service.
- `.planning/REQUIREMENTS.md` §"Runner Modes" - `RUN-01` and `RUN-03` for persistent default behavior and public/fork/untrusted workload warnings.
- `.planning/REQUIREMENTS.md` §"Documentation and Safety" - `DOC-01` for the concise BYO quickstart.
- `.planning/PROJECT.md` §"Current State", §"Context", §"Constraints", and §"Key Decisions" - Product promise, CLI-only v1, solo-developer focus, no workflow YAML edits, BYO-before-cloud ordering, and current Phase 1 foundation state.
- `.planning/STATE.md` §"Accumulated Context" - Current decisions/flags that Phase 2 must carry forward, including `runnerkit up`, least-privilege GitHub auth, public repo safety gate, versioned state, and stable labels.

### Prior phase decisions

- `.planning/phases/01-cli-auth-state-and-safety-foundation/01-CONTEXT.md` §"Implementation Decisions" - Locked Phase 1 decisions for guided CLI flow, repo/auth targeting, public repo risk gate, state/config split, runner identity/labels, and wizard flow content.
- `.planning/phases/01-cli-auth-state-and-safety-foundation/01-RESEARCH.md` - Phase 1 research outcomes that shaped the existing Go/Cobra, GitHub auth, labels, state, and safety foundation used by Phase 2.
- `.planning/phases/01-cli-auth-state-and-safety-foundation/01-VERIFICATION.md` - Verification results and remaining notes from the completed foundation.

### Research guidance for BYO remote setup

- `.planning/research/SUMMARY.md` §"Phase 2: BYO Linux Persistent Runner Happy Path" and §"Research Flags" - Recommended Phase 2 focus and explicit need to validate official runner package discovery, checksums, service behavior, and Linux matrix.
- `.planning/research/ARCHITECTURE.md` §"BYO Machine Flow" - End-to-end target flow for BYO SSH setup.
- `.planning/research/ARCHITECTURE.md` §"Remote Install / Provisioning Strategy" and §"Preflight Checks" - Linux/systemd-first install strategy, dedicated runner user, install paths, service manager, runner package, and minimum preflight categories.
- `.planning/research/ARCHITECTURE.md` §"Secrets and Token Handling" - Registration token handling, SSH key references, redaction, and avoiding durable secrets on the runner host.
- `.planning/research/FEATURES.md` §"Table Stakes" and §"Feature Dependencies" - BYO machine install path, service management, labels/snippet guidance, token safety, and dependency ordering.
- `.planning/research/PITFALLS.md` §"Pitfall 1", §"Pitfall 2", §"Pitfall 4", §"Pitfall 5", §"Pitfall 6", §"Pitfall 8", §"Pitfall 9", §"Pitfall 11", and §"Pitfall 14" - Persistent runner safety, token lifecycle, root-service prevention, secret leakage, split-brain/stale runners, SSH/bootstrap fragility, label safety, diagnostics quality, and partial cleanup concerns.
- `.planning/research/STACK.md` §"Core Technologies", §"If BYO machine setup", and §"What NOT to Use" - Go/Cobra, SSH/systemd, official runner package considerations, and root/generic-label/token-storage anti-patterns.

</canonical_refs>

<code_context>

## Existing Code Insights

### Reusable Assets

- `internal/cli/up.go`: Existing `runnerkit up` guided foundation flow with repo resolution, GitHub auth verification, public/fork safety enforcement, state preview/save, dry-run, `--yes`, `--non-interactive`, `--json`, and human step rendering. Phase 2 should extend or route through this flow rather than inventing a parallel BYO entry point.
- `internal/workflow/plan.go`: Existing plan/checkpoint primitives (`Plan`, `Step`, `Checkpoint`, `Apply`) can grow into BYO preflight, fix-plan, bootstrap, registration, and verification steps.
- `internal/state/schema.go`: Versioned repository state already includes `RunnerIdentity`, `MachineRef`, `ProviderRef`, `CleanupMetadata`, and safety metadata. `MachineRef` has host/user/install path/workdir/service name fields that fit BYO; it likely needs an SSH host fingerprint/identity field for D-03.
- `internal/labels/labels.go`: Stable RunnerKit labels and copy-paste `runs-on` snippet already exist (`self-hosted`, `runnerkit`, repo label, OS, arch, mode). Phase 2 must keep avoiding `runs-on: self-hosted` alone and may need architecture detection beyond the current x64 default.
- `internal/github/service.go` and `internal/github/tokens.go`: Production GitHub service and registration-token permission check already exist. Phase 2 can extend the GitHub adapter from token verification into actual runner registration/removal/listing as needed.
- `internal/ui/output.go` and `internal/ui/prompt.go`: Renderer/prompt abstractions already support redacted human/JSON output and confirmations; Phase 2 should reuse them for wizard-first SSH selection, host-key confirmation, preflight plans, and actionable failure output.

### Established Patterns

- Thin Cobra command layer with injectable dependencies and test fakes.
- CLI-only guided human flow with automation flags and JSON output available from the start.
- Fail-closed safety behavior for GitHub auth and public/fork persistent-runner risk.
- Versioned, secret-free, user-local JSON state with migration hooks and explicit replace behavior.
- Centralized redaction before output; remote command and bootstrap logs must pass through the same redaction discipline.
- Stable labels and snippets are product output, not an afterthought.

### Integration Points

- `runnerkit up` is the user-facing integration point for Phase 2 BYO setup.
- A new SSH/remote executor boundary should connect the CLI/workflow layer to remote preflight and bootstrap while preserving testability.
- The state store must be updated from Phase 1 placeholder machine metadata to real BYO host/user/install/service/fingerprint metadata.
- The GitHub adapter must provide just-in-time registration tokens and enough runner inventory/status information to register, detect duplicates/stale names, and verify the registered runner came online.
- The UI renderer must show preflight/fix/install/registration progress without leaking registration tokens, SSH secrets, or sensitive host output.

</code_context>

<specifics>
## Specific Ideas

- Keep the primary setup path as `runnerkit up`, not a new top-level BYO-only command.
- Interactive BYO setup should feel wizard-first for SSH details, even though direct flags exist for automation.
- Be inclusive of common systemd Linux hosts, but do not pretend unknown distros are fully supported: warn, require explicit override, and proceed best-effort.
- Treat host-key fingerprint acceptance as a durable trust decision captured in state.
- The user prioritized transparent preflight/fix plans over silently mutating the machine.

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within Phase 2 scope. Unselected gray areas remain planner discretion bounded by the roadmap, prior decisions, and canonical references above.

</deferred>

---

_Phase: 02-byo-persistent-runner-happy-path_
_Context gathered: 2026-04-29_
