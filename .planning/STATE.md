---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: completed
stopped_at: Phase 2 complete; ready for Phase 3 planning
last_updated: "2026-04-29T18:06:21.058Z"
last_activity: 2026-04-29 - Phase 2 BYO persistent runner happy path completed and verified.
progress:
  total_phases: 6
  completed_phases: 2
  total_plans: 8
  completed_plans: 8
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-29)

**Core value:** A solo developer can get a reliable, cost-effective GitHub Actions self-hosted runner online and usable in a project in about 10 minutes, without manual GitHub runner setup headaches.
**Current focus:** Phase 3: Operations, Diagnostics, and BYO Cleanup

## Current Position

Phase: 02 of 2 (byo persistent runner happy path)
Plan: 4 of 4
Status: Milestone complete
Last activity: 2026-04-29 - Phase 2 BYO persistent runner happy path completed and verified.

Milestone Progress: [███░░░░░░░] 33%

## Performance Metrics

**Velocity:**

- Total plans completed: 8
- Average duration: 21 min
- Total execution time: 2.8 hours

**By Phase:**

| Phase | Plans | Total  | Avg/Plan |
| ----- | ----- | ------ | -------- |
| 01    | 4/4   | 71 min | 18 min   |
| 02    | 4/4   | 95 min | 24 min   |

**Recent Trend:**

- Last 5 plans: 01-04 (45 min), 02-01 (20 min), 02-02 (25 min), 02-03 (30 min), 02-04 (20 min)
- Trend: BYO happy path is now covered by fake end-to-end tests; next work should harden real-world operations, diagnostics, recovery, and cleanup.

_Updated after each plan completion_

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Initialization: GitHub Actions only for v1; CLI-only interface; solo developers first.
- Initialization: Support both BYO machines and one recommended low-cost cloud provisioning path.
- Initialization: Persistent runners are the default for trusted private repos; ephemeral mode is explicit for stronger isolation.
- Roadmap: Build BYO persistent runner before cloud provisioning; harden diagnostics/cleanup before adding billable cloud resources.
- Phase 1 context: Prefer richer CLI wizard/TUI setup, `gh` auth first with fine-grained token fallback, git remote repo detection with confirmation, fail-closed permission handling, and optional project config plus user-local state.
- Plan 01-01: Use Cobra v1.10.1 for the RunnerKit CLI command tree.
- Plan 01-01: Centralize current human and JSON output through a renderer that adds `redactions_applied: true`.
- Plan 01-02: Use runner registration token creation as the runner-management permission check and immediately register returned tokens with the redactor.
- Plan 01-02: Block public repositories by default with `public_repo_risk`; require explicit `--allow-public-repo-risk` for future persistent setup.
- Plan 01-03: Persist Phase 1 foundation state as versioned, secret-free JSON with atomic writes and migration hooks.
- Plan 01-03: Use stable `runnerkit-owner-repo` labels plus explicit `runs-on` guidance; never recommend `self-hosted` alone.
- Plan 01-03: Require typed `replace owner/repo` or `--yes --replace` before replacing existing repo state.
- Plan 01-04: Production `runnerkit up` now defaults to `gh.NewService` plus `github.OSCommandRunner{}`; fake-permitted GitHub behavior is test-only.
- Plan 01-04: The real GitHub service caches credentials in memory only and uses registration-token creation plus repository metadata for permission/safety checks.
- Phase 2 context: Extend `runnerkit up` for BYO with wizard-first SSH detail collection and automation-friendly host flags.
- Phase 2 context: Prompt with SSH host fingerprint, record accepted fingerprint in state, and fail closed on mismatch.
- Phase 2 context: Support common systemd Linux best-effort; unknown or unverified distros warn, require explicit override, then try.
- Phase 2 context: Run full preflight before remote mutation, show fix plans before applying changes, and report remote progress with redacted actionable failure copy.
- Phase 2 context: Exact privilege flow is planner discretion, but the persistent runner service must not run as root by default.
- Plan 02-01: BYO SSH target parsing, host-key trust, and preflight checks are separate packages behind `remote.Executor`.
- Plan 02-02: Bootstrap uses pinned GitHub Actions runner 2.334.0 packages and installs the service as `runnerkit-runner`.
- Plan 02-03: Runner registration is just-in-time after preflight and duplicate checks; state is saved only after GitHub reports the runner online with RunnerKit labels.
- Plan 02-04: Persistent BYO is documented and warned as trusted-private-repository only; RunnerKit prints snippets and does not edit workflow YAML.

### Pending Todos

[From .planning/todos/pending/ - ideas captured during sessions]

None yet.

### Blockers/Concerns

[Issues that affect future work]

- Plan 01-02/01-04 validation note: automated fixtures cover GitHub success/denial/redaction/default-path behavior; a controlled live GitHub permission smoke remains recommended before public release.
- Phase 4: Default cloud provider should be validated for cost, availability, quota friction, and SSH readiness before locking the user-facing recommendation.
- Plan 01-01 validation note: `go run` wraps non-zero binary exits as process exit 1 while printing `exit status 6`; the direct built binary exits 6 for input-required paths.

## Session Continuity

Last session: 2026-04-29
Stopped at: Phase 2 complete; ready for Phase 3 planning
Resume file: .planning/phases/03-operations-diagnostics-and-byo-cleanup/03-CONTEXT.md
