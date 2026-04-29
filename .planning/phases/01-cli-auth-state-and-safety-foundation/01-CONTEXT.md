# Phase 1: CLI, Auth, State, and Safety Foundation - Context

**Gathered:** 2026-04-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 1 establishes RunnerKit's safe foundation before any real runner installation flow depends on it: an installable/runnable CLI shell, richer guided setup scaffold, GitHub authentication and repo targeting foundation, versioned non-secret local state/config, runner identity/label conventions, fake/idempotent workflow primitives, and redaction rules for state-adjacent logs/output.

This phase does **not** install a BYO runner, provision cloud infrastructure, manage real runner services, implement diagnostics/cleanup, or deliver ephemeral mode. Those capabilities belong to later roadmap phases.

</domain>

<decisions>
## Implementation Decisions

### First-run CLI surface

- **D-01:** The first-run experience should feel like a richer wizard/TUI-style guided setup while staying CLI-only for v1.
- **D-02:** The agent/planner may choose the primary setup command; `runnerkit up` is the recommended baseline because it can start as a foundation/setup wizard in Phase 1 and become the real create-runner path in later phases.
- **D-03:** Future mutating flows should show a plan/checklist and require confirmation before making changes. Phase 1 should establish this interaction pattern even if it only previews/saves foundation state.
- **D-04:** The CLI should default to interactive guidance for humans, but bake in automation-friendly behavior from day one: explicit flags, a yes/assume-confirmed path where safe, and JSON/non-interactive output conventions.

### GitHub auth and repo targeting

- **D-05:** Prefer reusing existing `gh` authentication first. If unavailable or insufficient, fallback to guided fine-grained token instructions rather than broad classic PAT assumptions.
- **D-06:** The setup flow should auto-detect the GitHub repository from the local git remote, then require the user to confirm the target repo before any auth/state action applies to it.
- **D-07:** If the available GitHub credential lacks required repository runner-management permissions, fail closed with exact fix instructions. Do not silently broaden permissions or proceed with an unsafe token path.
- **D-08:** Establish a hard safety gate for public repositories or fork-risk/untrusted workflow contexts before persistent setup is allowed later. Persistent setup should require an explicit danger override for those cases.

### Local state, config, runner identity, and labels

- **D-09:** Use optional project-level config for safe repeatable defaults plus mandatory user-local state for inventory and machine/provider/cleanup metadata.
- **D-10:** State storage format is the agent/planner's discretion, but it must be versioned from day one, migration-ready, human-debuggable enough for early v1, and able to support repo scope, runner identity, labels, machine path, provider IDs, and cleanup metadata.
- **D-11:** Exact label strategy is the agent/planner's discretion, but it must avoid encouraging generic `runs-on: self-hosted` alone and must support stable RunnerKit-specific copy-paste label guidance in later phases.
- **D-12:** Exact runner naming/collision behavior is the agent/planner's discretion, but it must support predictable repo-scoped status/cleanup and avoid duplicate/stale registration confusion.

### Wizard flow content

- **D-13:** Use this guided setup order: Welcome → prerequisites → repo/auth → safety checks → state preview → next steps.
- **D-14:** Wizard scope, prerequisite depth, and success copy are the agent/planner's discretion, but the wizard must not imply a runner exists before Phase 2. The end state should honestly communicate that foundations are ready and runner installation comes next.

### the agent's Discretion

- Primary setup command and exact command aliases, with `runnerkit up` favored unless research/planning finds a better command taxonomy.
- Exact prompt/TUI library and visual treatment for the richer CLI wizard.
- Exact automation flag names and JSON output schema, as long as the command surface supports non-interactive use from Phase 1.
- Exact state file format and locations within the optional-project-config + mandatory-user-local-state split.
- Exact runner label list and runner naming/collision scheme, provided the results are stable, explicit, cleanup-friendly, and safe.
- Exact wording for wizard prerequisite explanations and completion copy, provided it does not overpromise runner installation.

</decisions>

<canonical_refs>

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements

- `.planning/ROADMAP.md` §"Phase 1: CLI, Auth, State, and Safety Foundation" - Phase boundary, success criteria, and plan breakdown for CLI shell, GitHub auth, versioned state, and redaction.
- `.planning/REQUIREMENTS.md` §"CLI Onboarding" - `CLI-01` and `CLI-02` requirements for installable CLI and guided setup prerequisites.
- `.planning/REQUIREMENTS.md` §"GitHub Integration" - `GH-01` requirement for minimum-permission GitHub repository authentication.
- `.planning/REQUIREMENTS.md` §"Cleanup and State" - `STATE-01` and `STATE-02` requirements for versioned state/config and secret redaction.
- `.planning/PROJECT.md` §"Important product shape decisions" - GitHub Actions only, CLI-only, one-command feel, BYO/cloud later, persistent/ephemeral later, no workflow YAML edits.
- `.planning/STATE.md` §"Accumulated Context" - Current decisions and flags, especially GitHub auth least-privilege verification and persistent/default roadmap ordering.

### Research guidance

- `.planning/research/SUMMARY.md` §"Phase 1: CLI Foundation, GitHub Auth, State, and Safety Rails" - Recommended Phase 1 deliverables and research flags.
- `.planning/research/STACK.md` §"Recommended Stack" - Go/Cobra, GitHub API client, config/prompts, keyring, slog/redaction, and fake/test tooling guidance.
- `.planning/research/ARCHITECTURE.md` §"State Management" and §"Secrets and Token Handling" - Local-first state principles, secret references, no durable registration tokens, and redacted logs.
- `.planning/research/ARCHITECTURE.md` §"Plugin / Provider Boundaries" - Early provider/CI/remote interfaces that Phase 1 primitives should not preclude.
- `.planning/research/FEATURES.md` §"Table Stakes" and §"MVP Definition" - CLI setup, GitHub auth, local state, token safety, labels/snippet expectations.
- `.planning/research/PITFALLS.md` §"Pitfall 2", §"Pitfall 5", §"Pitfall 9", and §"Pitfall 13" - Registration token lifecycle, redaction, label safety, and least-privilege GitHub auth pitfalls.

</canonical_refs>

<code_context>

## Existing Code Insights

### Reusable Assets

- No application source tree exists yet. The repository currently contains planning/research docs only; Phase 1 will create the initial CLI/code structure.

### Established Patterns

- No codebase conventions exist yet. Planning docs strongly favor a local-first Go CLI with a thin command layer, adapter boundaries, versioned state, idempotent/checkpointed primitives, and centralized redaction.
- Existing docs establish these product constraints: CLI-only v1, GitHub Actions only, solo developers first, no workflow YAML edits, BYO runner path before cloud, diagnostics/cleanup before billable cloud resources.

### Integration Points

- New code should be initialized from scratch in this repo, likely around a Go module/CLI structure chosen during planning.
- Phase 1 interfaces should leave room for later GitHub runner registration, BYO SSH bootstrap, provider adapters, status/doctor reconciliation, and cleanup workflows without implementing those later capabilities yet.

</code_context>

<specifics>
## Specific Ideas

- User prefers a richer wizard/TUI-style first-run experience, not a bare prompt chain, while preserving CLI-only scope.
- The guided setup should follow: Welcome → prerequisites → repo/auth → safety checks → state preview → next steps.
- Use auto-detected git remote with explicit confirmation for repo targeting.
- The CLI should be honest when Phase 1 has prepared foundations but has not installed a runner yet.

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within the Phase 1 foundation scope.

</deferred>

---

_Phase: 01-cli-auth-state-and-safety-foundation_
_Context gathered: 2026-04-29_
