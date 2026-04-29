# Phase 3: Operations, Diagnostics, and BYO Cleanup - Context

**Gathered:** 2026-04-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 3 hardens RunnerKit's already-managed BYO persistent runner lifecycle. It delivers read-only operational status, diagnostics/log access, guided recovery for common stopped/offline runner cases, and safe BYO cleanup/deregistration when GitHub, SSH/systemd, and local state drift.

This phase stays inside managed BYO persistent runner operations. It does **not** add cloud provisioning/destruction, ephemeral runner mode, automatic workflow YAML edits, organization-level runner management, hosted dashboards, broad provider support, or automatic `doctor --fix` repair. Those belong to later roadmap phases or remain out of v1 scope.

</domain>

<decisions>
## Implementation Decisions

### Status reconciliation and output

- **D-01:** `runnerkit status` should default to the current repository when it can be inferred from the local git context, matching the existing `runnerkit up` repo-detection pattern. `runnerkit status --all` should show the inventory of all locally managed runners. `--repo owner/name` should remain available for explicit targeting.
- **D-02:** Default human status output should emphasize a top-line health state, a compact source matrix, and an exact next action. The output should answer: "Is this runner ready? If not, what do I run next?"
- **D-03:** Status should provide derived/interpreted health states while still showing raw observed facts underneath. The planner may define exact names, but expected states include concepts like Ready, Busy, Needs attention, Broken, and Unknown.
- **D-04:** `runnerkit status` must be read-only. It should not prompt to restart, deregister, remove files, or otherwise mutate local/GitHub/remote state. When drift is found, status should point to follow-on commands such as `doctor`, `logs`, recovery, or `down`.
- **D-05:** Default status probes should be fast health probes only: local RunnerKit state, GitHub runner inventory/status, SSH reachability, systemd service health, label comparison, and saved machine/path facts. Deep disk/tool/network/log checks belong to `runnerkit doctor`.
- **D-06:** Status should always show the saved recommended `runs-on` snippet for the current runner and should flag label drift when saved labels and GitHub runner labels differ. It should not scan or edit workflow YAML in this phase.
- **D-07:** When sources disagree, status should explain the likely cause and one safest next action rather than listing every possible cause. Example: GitHub offline + SSH reachable + failed service should point toward doctor/logs/restart guidance.
- **D-08:** `runnerkit status --json` should expose the same derived health model as human output plus raw source facts, reasons, next actions, labels/snippet, GitHub runner facts, SSH/service facts, and state path. This JSON contract should be useful for automation and regression tests.

### BYO cleanup and stale deregistration

- **D-09:** The primary BYO cleanup command should be `runnerkit down`, pairing with `runnerkit up`. Reserve `destroy` language for future billable cloud resources in Phase 4.
- **D-10:** In interactive mode, `runnerkit down` should present a cleanup plan and ask artifact-by-artifact before removing/deregistering each target. This should include GitHub registration, systemd service, managed runner install path, managed work path, and local state changes when applicable.
- **D-11:** Non-interactive `runnerkit down --yes` should apply the safe default cleanup plan rather than requiring per-artifact prompts or blocking automation.
- **D-12:** The safe default cleanup plan for `--yes` should only remove RunnerKit-managed runner-specific artifacts recorded in state: deregister/delete the GitHub runner record, stop/uninstall the recorded service, remove the recorded install path and runner work directory, and remove or update the matching local state record. It should avoid deleting unrelated BYO host data and should keep shared users/shared RunnerKit directories unless the planner can prove they are exclusively RunnerKit-owned and safe to remove.

### the agent's Discretion

- Exact health state names, severity levels, source matrix formatting, JSON field names, and command aliases are planner discretion as long as the decisions above are preserved.
- Exact `doctor`, `logs`, and recovery workflow details were not separately discussed. Planners should implement them within the roadmap requirements, carrying forward the status model: status is fast/read-only; doctor/logs provide deeper evidence; recovery is guided and confirmation-based.
- SSH-unreachable but GitHub-stale cleanup behavior is planner discretion. The expected direction is safe partial cleanup: if GitHub can be reconciled without SSH, deregister/delete stale GitHub records with an explicit plan, keep remote cleanup pending in state or output, and tell the user what may remain on the BYO host.
- Exact service/log commands (`systemctl`, `journalctl`, runner `_diag` access), time windows, and redaction mechanics are planner discretion, bounded by existing redaction and remote execution patterns.

</decisions>

<canonical_refs>

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements

- `.planning/ROADMAP.md` §"Phase 3: Operations, Diagnostics, and BYO Cleanup" - Fixed phase goal, success criteria, and four planned work slices for status reconciliation, logs/doctor, recovery, and BYO cleanup.
- `.planning/REQUIREMENTS.md` §"Reliability and Operations" - `REL-01`, `REL-02`, `REL-03`, and `REL-04` define status, logs, doctor, and restart/recovery requirements.
- `.planning/REQUIREMENTS.md` §"GitHub Integration" - `GH-03` defines runner deregistration when removing or recreating a managed runner.
- `.planning/REQUIREMENTS.md` §"Cleanup and State" - `CLEAN-02`, `CLEAN-03`, `STATE-01`, and `STATE-02` define stale GitHub deregistration, BYO file/service removal, state metadata, and redaction constraints.
- `.planning/PROJECT.md` §"Current State", §"Constraints", and §"Key Decisions" - Current BYO persistent baseline, CLI-only v1, solo-developer focus, no workflow YAML edits, persistent default, and non-root BYO service expectations.
- `.planning/STATE.md` §"Accumulated Context" - Completed Phase 1/2 decisions and implementation notes that Phase 3 must carry forward.

### Prior phase decisions and completed implementation context

- `.planning/phases/01-cli-auth-state-and-safety-foundation/01-CONTEXT.md` §"Implementation Decisions" - Guided CLI + automation flags/JSON, plan/checklist before mutation, GitHub auth safety, public repo gate, state/config split, labels, and redaction.
- `.planning/phases/02-byo-persistent-runner-happy-path/02-CONTEXT.md` §"Implementation Decisions" and §"Existing Code Insights" - BYO `runnerkit up`, SSH host-key trust, preflight-before-mutation, non-root service, remote progress, labels/snippet, and state integration points.
- `.planning/phases/02-byo-persistent-runner-happy-path/02-VERIFICATION.md` - Verification results for the completed BYO persistent happy path that Phase 3 should not regress.
- `docs/byo-quickstart.md` §"What RunnerKit does", §"Completion summary", and §"Troubleshooting" - Current user-facing BYO behavior and troubleshooting copy that status/logs/doctor/down should supersede or extend.

### Research guidance for operations and cleanup

- `.planning/research/SUMMARY.md` §"Phase 3: Status, Doctor, Repair, Cleanup Hardening" - Research-backed Phase 3 deliverables and anti-goals: multi-source status, doctor, repair plans, stale-delete flows, redacted support bundle, and partial cleanup checkpoints.
- `.planning/research/ARCHITECTURE.md` §"Status / Doctor Flow" - Reconciliation inputs and expected human-readable status/repair plan shape.
- `.planning/research/ARCHITECTURE.md` §"Cleanup Flow" - Cleanup sequencing across local state, GitHub removal/delete, remote service/files, provider/BYO ownership, and state removal.
- `.planning/research/ARCHITECTURE.md` §"State Management" - Desired-vs-observed state, resumable mutations, no durable registration tokens, ownership markers, and logs locations.
- `.planning/research/ARCHITECTURE.md` §"RemoteExecutor Contract" and §"CiProvider Contract" - Adapter capabilities needed for probes, logs, service operations, runner inventory/status/delete, and removal tokens.
- `.planning/research/FEATURES.md` §"Table Stakes" rows for Health/status command, Logs and troubleshooting access, Cleanup/destroy/deregister, Local state/config inventory, and Token/secret safety.
- `.planning/research/FEATURES.md` §"Feature Dependencies" - Dependencies showing status needs local state, service/log access, and GitHub runner status; doctor/repair depend on health/status and logs.
- `.planning/research/PITFALLS.md` §"Pitfall 3", §"Pitfall 5", §"Pitfall 6", §"Pitfall 9", §"Pitfall 11", and §"Pitfall 14" - Persistent contamination, diagnostic redaction, split-brain status, label safety, poor diagnostics, and partial cleanup failure modes.
- `.planning/research/STACK.md` §"Core Technologies" and §"Stack Patterns by Variant" - GitHub runner API, SSH/systemd, persistent runner mode, and operational tooling guidance.

</canonical_refs>

<code_context>

## Existing Code Insights

### Reusable Assets

- `internal/cli/root.go`: Existing Cobra root wiring and injectable `Dependencies` provide the place to add `status`, `logs`, `doctor`, recovery, and `down` commands while preserving testability.
- `internal/cli/up.go`: Existing `GitHubService` interface already includes `CreateRemovalToken`, `ListRunners`, and `DeleteRunner`. `waitForRunnerOnline`, `runnerOnlineWithLabels`, `verifyTargetHostKey`, preflight rendering, and `buildBYORepositoryState` are strong starting points for reconciliation and cleanup.
- `internal/github/runners.go`: Existing runner inventory/delete adapter exposes runner ID, name, OS, status, busy flag, and labels. This is enough for status source facts and stale GitHub deletion by ID/name/labels.
- `internal/state/schema.go`: `RepositoryState` already stores repo, runner identity/labels/snippet, BYO machine host/user/port/fingerprint/install path/work dir/service name, provider kind, GitHub runner ID, managed paths, and version metadata. These fields are the backbone for status and safe cleanup plans.
- `internal/state/store.go`: Store supports load, save, and get/upsert repository. Phase 3 likely needs update/remove/partial-cleanup helpers, but the atomic JSON persistence and secret-key validation already fit the safety model.
- `internal/remote/executor.go` and `internal/remote/system.go`: Remote `Probe` and `Run` abstractions already isolate SSH/system command execution. Phase 3 can add or layer service-status/log/cleanup commands through this boundary.
- `internal/preflight/checks.go`: Existing checks for SSH, host key, OS/arch, systemd, sudo, disk, tools, network, time, and runner conflict can be reused or selectively invoked by `doctor`; status should use a faster subset.
- `internal/bootstrap/install.go`: Existing bootstrap command IDs and service verification behavior identify managed install/service artifacts that cleanup must undo safely.
- `internal/workflow/plan.go`: Existing plan/checkpoint primitives and bootstrap plan pattern can become cleanup/recovery plans with explicit confirmation before mutation.
- `internal/ui/output.go`: Renderer already supports human/JSON output and redaction. Status/logs/doctor/down should reuse this instead of printing raw remote or GitHub output.
- `docs/byo-quickstart.md`: Current docs mention future cleanup and manual troubleshooting. Phase 3 should update this user-facing path after implementation.

### Established Patterns

- Thin Cobra command layer with injectable dependencies and test fakes.
- Human-first CLI output with JSON output available for automation and tests.
- Confirmation-before-mutation for remote host changes and state replacement.
- Fail-closed behavior for safety-critical cases such as public repository risk and SSH host-key mismatch.
- Versioned, secret-free local state with centralized redaction before output.
- Stable RunnerKit labels and copy-paste `runs-on` snippets are part of user-facing output.
- Production GitHub auth remains local to the developer machine; runner registration/removal tokens are requested just-in-time and never persisted.

### Integration Points

- `runnerkit status` should read local state, infer or accept repo scope, call GitHub runner inventory, probe SSH/service health, compare labels, and render the derived health/source matrix without mutation.
- `runnerkit down` should load local state, build an explicit cleanup plan, confirm interactively per artifact, use GitHub runner deletion/removal-token behavior where appropriate, run remote service/file cleanup only for recorded RunnerKit-managed artifacts, and update local state after partial or complete cleanup.
- `runnerkit doctor` and `runnerkit logs` should share the same state/GitHub/remote facts as status but perform deeper checks/log collection with mandatory redaction.
- Recovery commands or flows should reuse the state, GitHub, remote, and workflow plan boundaries instead of reusing `runnerkit up` as a blind reinstall path.

</code_context>

<specifics>
## Specific Ideas

- Status should be a safe command developers can run frequently. It should never surprise the user with prompts that mutate GitHub, state, or the remote host.
- The common status flow should feel repo-local: running `runnerkit status` in a project tells the developer about that project's managed runner; `--all` is the inventory view.
- The user prefers cleanup control in interactive mode: show the plan and ask per artifact before removal.
- Automation still matters: `runnerkit down --yes` should use the safe default plan without per-artifact prompts.
- Status should continue to teach the correct workflow labels by showing the exact saved `runs-on` snippet and warning about label drift.

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within Phase 3 scope. Unselected gray areas (`logs`, `doctor`, and recovery details) remain planner discretion bounded by the roadmap, requirements, prior decisions, and canonical references above.

</deferred>

---

_Phase: 03-operations-diagnostics-and-byo-cleanup_
_Context gathered: 2026-04-29_
