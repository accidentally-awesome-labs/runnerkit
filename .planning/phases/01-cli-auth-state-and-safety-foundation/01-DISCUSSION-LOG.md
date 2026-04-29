# Phase 1: CLI, Auth, State, and Safety Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md - this log preserves the alternatives considered.

**Date:** 2026-04-29
**Phase:** 1 - CLI, Auth, State, and Safety Foundation
**Areas discussed:** First-run CLI surface, GitHub auth and repo targeting, Local state/config/identity/labels, Wizard flow content

---

## Initial Gray Area Selection

| Option                                           | Description                                                                                                         | Selected |
| ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------- | -------- |
| First-run CLI surface                            | Command names, prompt style, setup/prerequisite flow, dry-run/confirmation behavior, and automation-friendly flags. | ✓        |
| GitHub auth and repo targeting                   | How users authenticate, repo scope selection/explanation, least-privilege posture, and fallback paths.              |          |
| Local state, config, runner identity, and labels | State/config locations, project vs user scope, schema expectations, runner naming, and stable labels.               |          |
| Redaction and safety output rules                | Secret masking, debug/log output, support-bundle posture, and fail-closed behavior.                                 |          |

**User's choice:** First-run CLI surface.

---

## First-run CLI surface

### Primary guided setup command

| Option            | Description                                                                                                | Selected |
| ----------------- | ---------------------------------------------------------------------------------------------------------- | -------- |
| `runnerkit up`    | Action-oriented; can start as prerequisite/auth/state wizard and become the real create-runner path later. |          |
| `runnerkit init`  | Config-first and familiar, but may feel like it only writes files.                                         |          |
| `runnerkit setup` | Plain language and explicit, but less standard for lifecycle commands.                                     |          |
| You decide        | Let the planner pick the command shape.                                                                    | ✓        |

**User's choice:** You decide.

### Setup scaffold feel

| Option                                              | Description                                                        | Selected |
| --------------------------------------------------- | ------------------------------------------------------------------ | -------- |
| Plain interactive prompts with short explanations   | Fast, scriptable CLI feel.                                         |          |
| Richer wizard/TUI experience                        | More polished and beginner-friendly, but adds dependency/UX scope. | ✓        |
| Mostly flag-driven, prompts only for missing inputs | Best for automation, weaker guided onboarding.                     |          |
| You decide                                          | Let planner choose prompt style.                                   |          |

**User's choice:** Richer wizard/TUI experience.

### Pre-mutation interaction pattern

| Option                                                   | Description                                                         | Selected |
| -------------------------------------------------------- | ------------------------------------------------------------------- | -------- |
| Show a plan/checklist, then require confirmation         | Matches infrastructure-tool expectations and supports later safety. | ✓        |
| Explain prerequisites inline, then proceed automatically | Fastest path, weaker safety posture.                                |          |
| Always dry-run first; user re-runs/apply                 | Very safe, but more friction.                                       |          |
| You decide                                               | Let planner decide.                                                 |          |

**User's choice:** Show a plan/checklist, then require confirmation.

### Automation/non-interactive support

| Option                                              | Description                                                                   | Selected |
| --------------------------------------------------- | ----------------------------------------------------------------------------- | -------- |
| Interactive default + automation flags from day one | Guided UX plus `--yes`, `--json`, and explicit flags to avoid redesign later. | ✓        |
| Interactive-only for now                            | Fastest Phase 1 UX, but creates churn later.                                  |          |
| Automation-first, prompts as convenience layer      | Most scriptable, less guided.                                                 |          |
| You decide                                          | Let planner decide.                                                           |          |

**User's choice:** Interactive default + automation flags from day one.

**Area continuation:** User chose to move on/finish selected area.

---

## Additional Gray Area Selection #1

| Option                                           | Description                                                                                           | Selected |
| ------------------------------------------------ | ----------------------------------------------------------------------------------------------------- | -------- |
| GitHub auth and repo targeting                   | Auth path, repo selection, least-privilege messaging, and public/private risk gates.                  | ✓        |
| Local state, config, runner identity, and labels | State/config locations, project vs user scope, schema expectations, runner naming, and stable labels. |          |
| Redaction and safety output rules                | Secret masking, debug/log output, support-bundle posture, and fail-closed behavior.                   |          |
| Wizard flow content                              | What the rich guided setup should show in Phase 1.                                                    |          |

**User's choice:** GitHub auth and repo targeting.

---

## GitHub auth and repo targeting

### Preferred auth path

| Option                                                                | Description                                                 | Selected |
| --------------------------------------------------------------------- | ----------------------------------------------------------- | -------- |
| Reuse existing `gh` auth first, fallback to guided fine-grained token | Lowest friction while keeping permission guidance explicit. | ✓        |
| GitHub App/device-style flow first                                    | Cleaner UX, more implementation/research complexity.        |          |
| Fine-grained PAT flow first                                           | Transparent and repository-scoped, but manual.              |          |
| You decide                                                            | Let research/planning choose.                               |          |

**User's choice:** Reuse existing `gh` auth first, fallback to guided fine-grained token.

### Repository targeting

| Option                                          | Description                                                      | Selected |
| ----------------------------------------------- | ---------------------------------------------------------------- | -------- |
| Auto-detect git remote, then confirm            | Nearly one-command in a repo while avoiding accidental mutation. | ✓        |
| Require `--repo owner/name` or typed repo input | Explicit and safe, less magical.                                 |          |
| Show a repo picker after auth                   | Nice wizard, may require broader listing permissions.            |          |
| You decide                                      | Let planner choose.                                              |          |

**User's choice:** Auto-detect git remote, then confirm.

### Missing permissions behavior

| Option                                         | Description                                                            | Selected |
| ---------------------------------------------- | ---------------------------------------------------------------------- | -------- |
| Fail closed with exact fix instructions        | Safest posture; explain missing permissions without broadening access. | ✓        |
| Offer broader-token fallback with loud warning | Unblocks users, but unsafe classic PAT pattern risk.                   |          |
| Continue in limited/dry-run mode only          | Allows inspection, delays real setup.                                  |          |
| You decide                                     | Let research/planning determine.                                       |          |

**User's choice:** Fail closed with exact fix instructions.

### Public/fork-risk repository gate

| Option                                                     | Description                                       | Selected |
| ---------------------------------------------------------- | ------------------------------------------------- | -------- |
| Hard gate persistent setup unless explicit danger override | Matches safety promise for public/fork workflows. | ✓        |
| Warn loudly but allow normal persistent setup              | Less friction, easier unsafe choices.             |          |
| Only document the risk for now                             | Fastest, but defers core safety.                  |          |
| You decide                                                 | Let planner/research choose.                      |          |

**User's choice:** Hard gate persistent setup unless explicit danger override.

**Area continuation:** User chose to move on/finish selected area.

---

## Additional Gray Area Selection #2

| Option                                           | Description                                                                                           | Selected |
| ------------------------------------------------ | ----------------------------------------------------------------------------------------------------- | -------- |
| Local state, config, runner identity, and labels | State/config locations, project vs user scope, schema expectations, runner naming, and stable labels. | ✓        |
| Redaction and safety output rules                | Secret masking, debug/log output, support-bundle posture, and fail-closed behavior.                   |          |
| Wizard flow content                              | What the rich guided setup should show in Phase 1.                                                    |          |
| Command/output conventions                       | Verbosity, JSON/table output, exit code philosophy, and output shape.                                 |          |

**User's choice:** Local state, config, runner identity, and labels.

---

## Local state, config, runner identity, and labels

### Project config vs user-local state split

| Option                                               | Description                                                                                      | Selected |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------ | -------- |
| Optional project config + mandatory user-local state | Safe repeatable defaults in project config; inventory/secrets-adjacent machine state in OS dirs. | ✓        |
| Everything user-local only                           | Simpler/private, less repeatable.                                                                |          |
| Everything project-local where possible              | Transparent/repeatable, higher accidental commit risk.                                           |          |
| You decide                                           | Let planner decide.                                                                              |          |

**User's choice:** Optional project config + mandatory user-local state.

### Versioned state schema optimization

| Option                                                         | Description                                    | Selected |
| -------------------------------------------------------------- | ---------------------------------------------- | -------- |
| Readable JSON/TOML with explicit schema version and migrations | Inspectable and migration-ready for early v1.  |          |
| SQLite-backed inventory from day one                           | Robust for many runners/events, heavier early. |          |
| Minimal config now; expand schema later                        | Fast, but risky for cleanup/status.            |          |
| You decide                                                     | Let planner choose.                            | ✓        |

**User's choice:** You decide.

### Label strategy

| Option                                               | Description                                                           | Selected |
| ---------------------------------------------------- | --------------------------------------------------------------------- | -------- |
| Stable, explicit RunnerKit labels only               | Include RunnerKit-specific labels; avoid generic `self-hosted` alone. |          |
| Minimal labels and let users add their own           | Flexible but error-prone.                                             |          |
| Per-runner unique labels as main targeting mechanism | Precise but less stable.                                              |          |
| You decide                                           | Let planner decide.                                                   | ✓        |

**User's choice:** You decide.

### Runner identity

| Option                                                  | Description                                     | Selected |
| ------------------------------------------------------- | ----------------------------------------------- | -------- |
| Stable repo-scoped default name with collision handling | Predictable names with suffix/replace guidance. |          |
| Random unique runner names every setup                  | Avoids collisions but harder cleanup/status.    |          |
| Require user-supplied runner names                      | Explicit but adds burden.                       |          |
| You decide                                              | Let planner pick naming details.                | ✓        |

**User's choice:** You decide.

**Area continuation:** User chose to move on/finish selected area.

---

## Additional Gray Area Selection #3

| Option                            | Description                                                                         | Selected |
| --------------------------------- | ----------------------------------------------------------------------------------- | -------- |
| Redaction and safety output rules | Secret masking, debug/log output, support-bundle posture, and fail-closed behavior. |          |
| Wizard flow content               | What the rich guided setup should show in Phase 1.                                  | ✓        |
| Command/output conventions        | Verbosity, JSON/table output, exit code philosophy, and output shape.               |          |

**User's choice:** Wizard flow content.

---

## Wizard flow content

### Phase 1 guided setup purpose

| Option                                                              | Description                                                                            | Selected |
| ------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------- |
| Prepare and verify foundations, then clearly hand off to next steps | Auth, repo confirmation, safety posture, config/state preview, no fake runner install. |          |
| Feel like a full setup even if some steps are placeholders          | Exciting, but risks overpromising.                                                     |          |
| Only collect config; keep explanations minimal                      | Fast, weaker onboarding.                                                               |          |
| You decide                                                          | Let planner decide exact scope.                                                        | ✓        |

**User's choice:** You decide.

### Wizard screen/order

| Option                                                                           | Description                                                           | Selected |
| -------------------------------------------------------------------------------- | --------------------------------------------------------------------- | -------- |
| Welcome → prerequisites → repo/auth → safety checks → state preview → next steps | Explains before auth, confirms target/safety, then shows saved state. | ✓        |
| Repo/auth first → then prerequisites and safety                                  | Useful validation quickly, but asks for access first.                 |          |
| Mode/path selection first → then auth/state                                      | Helps with future mode/path choices, but later-phase scope.           |          |
| You decide                                                                       | Let planner shape the flow.                                           |          |

**User's choice:** Welcome → prerequisites → repo/auth → safety checks → state preview → next steps.

### Prerequisites depth

| Option                                                                  | Description                                                 | Selected |
| ----------------------------------------------------------------------- | ----------------------------------------------------------- | -------- |
| Explain all v1 prerequisites at high level, only validate Phase 1 items | Sets expectations without pretending to preflight machines. |          |
| Only mention Phase 1 prerequisites                                      | Focused and accurate, may surprise later.                   |          |
| Deep-check every future prerequisite now                                | Reassuring but adds later-phase scope.                      |          |
| You decide                                                              | Let planner decide.                                         | ✓        |

**User's choice:** You decide.

### Success screen emphasis

| Option                                        | Description                                           | Selected |
| --------------------------------------------- | ----------------------------------------------------- | -------- |
| Foundations are ready; runner install is next | Honest completion state with next command/phase path. |          |
| RunnerKit setup complete                      | Simpler, may overpromise.                             |          |
| Technical summary only                        | Useful but less onboarding-friendly.                  |          |
| You decide                                    | Let planner decide exact copy.                        | ✓        |

**User's choice:** You decide.

**Area continuation:** User chose to move on/finish selected area.

---

## the agent's Discretion

- Primary setup command and exact command aliases, with `runnerkit up` recommended.
- State storage format and exact state file locations within the approved config/state split.
- Stable label strategy and runner naming/collision behavior.
- Wizard scope/prerequisite depth/success copy, constrained by the selected flow order and no-overpromise requirement.

## Deferred Ideas

None.
