# Phase 3: Operations, Diagnostics, and BYO Cleanup - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md - this log preserves the alternatives considered.

**Date:** 2026-04-29T22:19:01Z
**Phase:** 3 - Operations, Diagnostics, and BYO Cleanup
**Areas discussed:** Status reconciliation and output; BYO cleanup and stale deregistration

---

## Initial Gray Area Selection

| Option                               | Description                                                                                                                                                                                                     | Selected |
| ------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Status reconciliation and output     | How `runnerkit status` should present local state, GitHub runner status, SSH reachability, systemd service health, labels, and drift. Existing code already has state, GitHub runner inventory, and SSH probes. | ✓        |
| Logs and doctor findings             | What `runnerkit logs` collects by default and how `runnerkit doctor` turns logs/preflight/service checks into actionable findings.                                                                              |          |
| Recovery workflows                   | How guided restart/re-registration should work for stopped service, offline runner, stale/missing GitHub registration, or partial install.                                                                      |          |
| BYO cleanup and stale deregistration | How safe removal should behave when local state, SSH access, systemd, runner files, or GitHub records are missing.                                                                                              |          |

**User's choice:** Status reconciliation and output

---

## Status reconciliation and output

### Question: How should `runnerkit status` choose what to show by default?

| Option                                              | Description                                                                                                                        | Selected |
| --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Current repo by default, `--all` for inventory      | Use git remote/repo context when available, matching `runnerkit up`; show one detailed runner by default. Good for solo-dev focus. | ✓        |
| All managed runners by default, `--repo` for detail | Treat status as an operations dashboard over local RunnerKit state.                                                                |          |
| Require `--repo` every time                         | Most explicit and automation-friendly, but less convenient for humans.                                                             |          |
| You decide                                          | Let the planner pick the command behavior that best fits existing Cobra/state patterns.                                            |          |

**User's choice:** Current repo by default, `--all` for inventory

### Question: What should the default human `runnerkit status` output emphasize?

| Option                                        | Description                                                                                         | Selected |
| --------------------------------------------- | --------------------------------------------------------------------------------------------------- | -------- |
| Top-line health + source matrix + next action | Ready/Needs attention/Broken, then GitHub/SSH/systemd/labels rows, then one clear next command.     | ✓        |
| Compact table only                            | Fast to scan for multiple runners, but may hide why something is unhealthy.                         |          |
| Verbose diagnostic report                     | Detailed check-by-check output by default; useful for debugging but noisier for a frequent command. |          |
| You decide                                    | Planner can tune output using existing renderer and JSON conventions.                               |          |

**User's choice:** Top-line health + source matrix + next action

### Question: How opinionated should the status health classification be?

| Option                                          | Description                                                                                              | Selected |
| ----------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------- |
| Derived health states with raw facts underneath | Use states like Ready, Busy, Needs attention, Broken, Unknown; show raw GitHub/SSH/systemd labels below. | ✓        |
| Raw source facts only                           | Show GitHub status, SSH, systemd, labels, paths, but avoid interpreting them into health.                |          |
| Strict pass/fail only                           | Simpler signal, but loses useful distinctions such as busy vs offline vs partially unreachable.          |          |
| You decide                                      | Planner can define the exact taxonomy.                                                                   |          |

**User's choice:** Derived health states with raw facts underneath

### Question: When `status` finds drift or missing evidence, should it stay read-only or start a guided fix?

| Option                                  | Description                                                                                                     | Selected |
| --------------------------------------- | --------------------------------------------------------------------------------------------------------------- | -------- |
| Read-only with exact next commands      | Status never mutates; it says `runnerkit doctor`, `runnerkit recover`, or `runnerkit down --stale-github` next. | ✓        |
| Offer interactive repair prompts inline | Convenient for humans, but makes a status command unexpectedly mutating.                                        |          |
| Fail fast on incomplete state           | Keeps implementation simple but frustrates users when drift is exactly what they need to understand.            |          |
| You decide                              | Planner can pick the safest status behavior.                                                                    |          |

**User's choice:** Read-only with exact next commands

### Continuation

**Question:** More questions about Status reconciliation and output, or move on?

| Option         | Selected |
| -------------- | -------- |
| More questions | ✓        |
| Next area      |          |

### Question: How deep should `status` probe the remote host by default?

| Option                                                | Description                                                                                                                         | Selected |
| ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Fast health probes only                               | Check SSH reachability, service status, GitHub runner status, labels, and saved paths; leave disk/tool/log deep checks to `doctor`. | ✓        |
| Deep diagnostic probes by default                     | Also check disk, tools, network, runner dirs, and recent errors every time; more complete but slower/noisier.                       |          |
| Use cached last-known remote facts unless `--refresh` | Fastest, but risks hiding real drift when the user runs status to debug.                                                            |          |
| You decide                                            | Planner decides probe depth based on implementation risk.                                                                           |          |

**User's choice:** Fast health probes only

### Question: How should `status` handle label information?

| Option                            | Description                                                                                                      | Selected |
| --------------------------------- | ---------------------------------------------------------------------------------------------------------------- | -------- |
| Show snippet and flag label drift | Always show the saved `runs-on` snippet; compare saved labels with GitHub labels and warn if missing/mismatched. | ✓        |
| Show labels only when unhealthy   | Less clutter, but users may still need the snippet even when healthy.                                            |          |
| Do not include snippet in status  | Keep status focused on health; users can use docs/state show for labels.                                         |          |
| You decide                        | Planner can choose the balance.                                                                                  |          |

**User's choice:** Show snippet and flag label drift

### Question: When sources disagree, what should `status` optimize for in its message?

| Option                                              | Description                                                                                                  | Selected |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | -------- |
| Explain the likely cause and one safest next action | Example: GitHub offline + SSH reachable + service failed → `runnerkit doctor --repo ...` / restart guidance. | ✓        |
| List every possible cause                           | Comprehensive but can feel like generic troubleshooting.                                                     |          |
| Only report the disagreement, no cause guess        | Least risky, but pushes users back into manual debugging.                                                    |          |
| You decide                                          | Planner can define drift messages.                                                                           |          |

**User's choice:** Explain the likely cause and one safest next action

### Question: What should the JSON `runnerkit status --json` contract include?

| Option                                  | Description                                                                                                    | Selected |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------- | -------- |
| Same health model plus raw source facts | Include derived health, reasons, next_actions, GitHub runner facts, SSH/service facts, labels, and state path. | ✓        |
| Raw facts only                          | Scripts can decide health themselves, but duplicate CLI interpretation logic.                                  |          |
| Minimal stable fields now               | Lowest commitment, but less useful for automation and regression tests.                                        |          |
| You decide                              | Planner can set the JSON shape.                                                                                |          |

**User's choice:** Same health model plus raw source facts

### Continuation

**Question:** More questions about Status reconciliation and output, or move on?

| Option         | Selected |
| -------------- | -------- |
| More questions |          |
| Next area      | ✓        |

---

## Explore More Gray Areas

| Option                  | Description                                 | Selected |
| ----------------------- | ------------------------------------------- | -------- |
| Explore more gray areas | Continue with additional Phase 3 decisions. | ✓        |
| I'm ready for context   | Proceed to write context.                   |          |

**User's choice:** Explore more gray areas

## Additional Gray Area Selection

| Option                                      | Description                                                                                                                                            | Selected |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| Logs and doctor findings                    | Define what `runnerkit logs` and `runnerkit doctor` collect/check, how much is shown by default, and how findings map to status next actions.          |          |
| Recovery workflows                          | Decide how the CLI guides restart, re-registration, and common offline/stopped-service recovery.                                                       |          |
| BYO cleanup and stale deregistration        | Decide safe removal behavior for GitHub records, systemd service, runner files, managed paths, and local state when SSH/GitHub/local state is partial. | ✓        |
| Operation command names and action taxonomy | Lock names/status next-action shape for follow-on commands such as `doctor`, `logs`, `recover`/`repair`, `down`/`cleanup`, and `forget`.               |          |

**User's choice:** BYO cleanup and stale deregistration

---

## BYO cleanup and stale deregistration

### Question: What should the primary BYO cleanup command be called?

| Option              | Description                                                                                                             | Selected |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------- | -------- |
| `runnerkit down`    | Pairs with `runnerkit up`; can mean take this managed runner down safely. Reserve `destroy` for future cloud resources. | ✓        |
| `runnerkit cleanup` | Plainly descriptive and less ambiguous, but less paired with setup.                                                     |          |
| `runnerkit remove`  | Specific to runner removal, but can sound like uninstalling the CLI itself.                                             |          |
| You decide          | Planner can choose command naming while preserving status next-action intent.                                           |          |

**User's choice:** `runnerkit down`

### Question: What should `runnerkit down` remove by default on a reachable BYO host?

| Option                                                     | Description                                                                                                                                                                                                  | Selected |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| Only RunnerKit-managed runner-specific artifacts           | Deregister GitHub runner, stop/uninstall the service, remove the install path and work dir recorded in state, then remove/update local state. Keep shared user/shared RunnerKit dirs unless explicitly safe. |          |
| Remove everything RunnerKit created including service user | Cleaner host, but riskier if multiple runners or user-owned files share the account/paths.                                                                                                                   |          |
| Ask interactively for each artifact                        | Maximum control, but slow and noisy for the common path.                                                                                                                                                     | ✓        |
| You decide                                                 | Planner can define exact artifact list using cleanup metadata.                                                                                                                                               |          |

**User's choice:** Ask interactively for each artifact
**Notes:** This led to a follow-up question about non-interactive cleanup.

### Question: Since interactive cleanup asks artifact-by-artifact, how should non-interactive `runnerkit down --yes` behave?

| Option                                   | Description                                                                                         | Selected |
| ---------------------------------------- | --------------------------------------------------------------------------------------------------- | -------- |
| Apply the safe default plan              | Show/emit the plan, then remove only RunnerKit-managed runner-specific artifacts recorded in state. | ✓        |
| Require explicit flags for each artifact | Most conservative, but makes automation cumbersome.                                                 |          |
| Disallow non-interactive cleanup         | Safest for BYO hosts, but conflicts with the CLI's automation-friendly pattern.                     |          |
| You decide                               | Planner can design the non-interactive cleanup contract.                                            |          |

**User's choice:** Apply the safe default plan

### Question: If SSH is unreachable but GitHub still has the RunnerKit runner record, what should `runnerkit down` do?

| Option                                            | Description                                                                                                                                            | Selected |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| Deregister GitHub and keep remote cleanup pending | Use GitHub API deletion/removal where possible, update local state with a pending remote cleanup note, and tell the user what remains on the BYO host. |          |
| Block until SSH is reachable                      | Avoids leaving host artifacts, but fails the exact stale/offline cleanup case.                                                                         |          |
| Only print manual instructions                    | Very safe, but leaves stale GitHub records and queued-job confusion.                                                                                   |          |
| You decide                                        | Planner can define partial cleanup behavior.                                                                                                           | ✓        |

**User's choice:** You decide
**Notes:** Captured as the agent's discretion, bounded by safe partial cleanup and explicit user-facing plan/output.

### Continuation

**Question:** More questions about BYO cleanup and stale deregistration, or move on?

| Option         | Selected |
| -------------- | -------- |
| More questions |          |
| Next area      | ✓        |

---

## Final Readiness

| Option                  | Selected |
| ----------------------- | -------- |
| Explore more gray areas |          |
| I'm ready for context   | ✓        |

**User's choice:** I'm ready for context

---

## the agent's Discretion

- Exact health state names, source matrix formatting, and JSON field names.
- Exact logs/doctor/recovery details, since those gray areas were not selected for detailed discussion.
- SSH-unreachable but GitHub-stale cleanup behavior, with expected safe partial cleanup direction.

## Deferred Ideas

None.
