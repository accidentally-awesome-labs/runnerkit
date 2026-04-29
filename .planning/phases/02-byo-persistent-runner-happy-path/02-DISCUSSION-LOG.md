# Phase 2: BYO Persistent Runner Happy Path - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md - this log preserves the alternatives considered.

**Date:** 2026-04-29T17:17:27Z
**Phase:** 2-BYO Persistent Runner Happy Path
**Areas discussed:** SSH target and preflight strictness

---

## Initial gray area selection

| Option                                         | Description                                                                                                                                                                                                               | Selected |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| SSH target and preflight strictness            | How the user supplies/selects `user@host`, host-key trust, sudo/non-root expectations, and how strict unsupported-host failures should be. Recommended because this shapes the first real BYO UX.                         | ✓        |
| Bootstrap and service defaults                 | How opinionated RunnerKit should be about runner user, install paths, work directory, dependencies, official runner download/checks, and customization knobs. Existing state already has MachineRef fields for these.     |          |
| Registration, naming, and labels after install | How to handle duplicate runner names, stable labels, storing GitHub runner ID, and exact copy-paste `runs-on` guidance. Existing labels package and state schema provide a baseline.                                      |          |
| Safety warnings and completion experience      | How loud persistent-runner risk warnings should be, when overrides are acceptable, and what the 10-minute success summary/quickstart should emphasize. Recommended because public/fork risk was already gated in Phase 1. |          |

---

## SSH target and preflight strictness

### Question 1: BYO setup invocation

| Option                             | Description                                                                                                                                               | Selected |
| ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Keep `runnerkit up` with BYO flags | Recommended. Extend the existing guided setup: e.g. `runnerkit up --repo owner/name --host user@host`, with interactive prompts for missing host details. | ✓        |
| Add a dedicated BYO command        | Example: `runnerkit byo up --host user@host`. Clearer taxonomy, but splits the one-command feel established in Phase 1.                                   |          |
| Ask interactively only             | User runs `runnerkit up` and the wizard asks for host/user details; flags still exist later for automation.                                               |          |
| You decide                         | Planner can choose the command shape as long as BYO remains CLI-only and automation-friendly.                                                             |          |

**User's choice:** Keep `runnerkit up` with BYO flags.
**Notes:** Preserve the Phase 1 `runnerkit up` flow and extend it for BYO.

### Question 2: SSH host-key handling

| Option                                        | Description                                                                                                                                                      | Selected |
| --------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Prompt + record fingerprint; fail on mismatch | Recommended. Trust-on-first-use for unknown hosts with a clear fingerprint prompt, store the accepted fingerprint in state, and fail closed if it changes later. | ✓        |
| Require existing `known_hosts` entry          | Most secure by default but adds friction; setup fails until the developer pre-connects with SSH manually.                                                        |          |
| Simple yes/no trust prompt only               | Fastest path, but less auditable because the accepted fingerprint is not explicit in RunnerKit state.                                                            |          |
| You decide                                    | Planner can pick a safe standard approach.                                                                                                                       |          |

**User's choice:** Prompt + record fingerprint; fail on mismatch.
**Notes:** Host-key trust should become durable RunnerKit state, not just a transient prompt.

### Question 3: Privilege model

| Option                                    | Description                                                                                                                                               | Selected |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Sudo for install, non-root runner service | Recommended. SSH user may use sudo for package/user/service setup, but the GitHub runner itself runs as a dedicated unprivileged `runnerkit`/runner user. |          |
| Strict no-sudo BYO mode                   | Only works if the host already has required dependencies, directories, and user permissions. Safer on shared hosts, but not as happy-path friendly.       |          |
| Allow root SSH bootstrap                  | Permits logging in as root for bootstrap on fresh servers, but still must create and run the service as a non-root runner user.                           |          |
| You decide                                | Planner can decide the exact privilege flow as long as the runner service never runs as root by default.                                                  | ✓        |

**User's choice:** You decide.
**Notes:** Planner discretion, with the hard constraint that the persistent runner service should not run as root by default.

### Question 4: Supported Linux/systemd matrix

| Option                                      | Description                                                                                                                       | Selected |
| ------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Ubuntu/Debian x64 only, fail fast elsewhere | Most reliable 10-minute happy path and simplest docs. Existing labels default to linux/x64.                                       |          |
| Ubuntu/Debian x64 + arm64                   | Still narrow, but supports common cheap ARM VPS/homelab machines. Requires runner download/label handling for both architectures. |          |
| Any common systemd Linux, best effort       | Broader appeal, but risks fragile bootstrap and distro-specific package-manager failures in the first happy path.                 | ✓        |
| You decide                                  | Planner can choose the supported matrix based on official runner package/support research.                                        |          |

**User's choice:** Any common systemd Linux, best effort.
**Notes:** User favors broader common-systemd coverage over a narrow Ubuntu/Debian-only claim.

### Follow-up: Unknown or unverified distro behavior

| Option                                    | Description                                                                                               | Selected |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------------- | -------- |
| Warn, require explicit override, then try | Recommended for the broader support choice. Keeps the happy path open while making support risk explicit. | ✓        |
| Proceed with warning only                 | Fastest, but users may think unsupported hosts are fully supported.                                       |          |
| Fail closed unless distro is verified     | More reliable, but effectively narrows support despite the broader matrix.                                |          |
| You decide                                | Planner can choose the balance during research.                                                           |          |

**User's choice:** Warn, require explicit override, then try.
**Notes:** Best-effort support should not silently imply full support.

### Question 5: Existing SSH setup UX

| Option                                | Description                                                                                                                                | Selected |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| Reuse OpenSSH config/agent by default | Recommended. Let `user@host`, SSH config aliases, default keys, and ssh-agent work like normal `ssh`; add explicit flags only when needed. |          |
| Require explicit host/user/key flags  | More deterministic and easier to document, but more typing and less friendly to existing SSH workflows.                                    |          |
| Wizard-first SSH selection            | Ask for host, user, port, and key interactively; less copy-paste friendly for automation.                                                  | ✓        |
| You decide                            | Planner can choose the exact SSH UX.                                                                                                       |          |

**User's choice:** Wizard-first SSH selection.
**Notes:** Interactive flow should ask for SSH details, while earlier invocation choice still preserves flags for automation.

### Question 6: Pre-mutation preflight posture

| Option                                 | Description                                                                                                                                                             | Selected |
| -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Full checklist, no mutation until pass | Recommended. Verify SSH, fingerprint, OS/arch, systemd, sudo/install ability, disk, required tools, time sync, and outbound HTTPS to GitHub before installing anything. | ✓        |
| Minimal preflight, fix during install  | Faster start, but failures happen mid-bootstrap and are harder to recover from.                                                                                         |          |
| Two-stage: quick check then deep check | Quick SSH/OS check first, then a deeper pre-mutation checklist right before install.                                                                                    |          |
| You decide                             | Planner can choose which checks are mandatory.                                                                                                                          |          |

**User's choice:** Full checklist, no mutation until pass.
**Notes:** Preflight should reduce partial remote mutations and make failures understandable before bootstrap.

### Question 7: Missing prerequisite fix UX

| Option                             | Description                                                                                                                  | Selected |
| ---------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | -------- |
| Show fix plan, ask before applying | Recommended. Keep Phase 1's plan/checklist pattern: explain package/user/service changes, then ask before mutating the host. | ✓        |
| Auto-fix safe prerequisites        | Fastest 10-minute path, but less transparent for a tool modifying a user's server.                                           |          |
| Never auto-fix in preflight        | Preflight only reports; bootstrap later installs. Clear boundary, but potentially repetitive.                                |          |
| You decide                         | Planner can choose the repair/install split.                                                                                 |          |

**User's choice:** Show fix plan, ask before applying.
**Notes:** Transparency before mutation matters more than silently maximizing speed.

### Question 8: Progress and failure reporting

| Option                                             | Description                                                                                                                                         | Selected |
| -------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Step-by-step progress with actionable failure copy | Recommended. Show the current remote check/command category, redact output, and end with exact next action; keep raw SSH details out unless needed. | ✓        |
| Verbose by default                                 | Print most remote command output as it happens. Good for power users, but noisier and higher redaction risk.                                        |          |
| Quiet unless failure                               | Cleanest happy path, but long SSH/package checks can feel hung.                                                                                     |          |
| You decide                                         | Planner can decide the output detail.                                                                                                               |          |

**User's choice:** Step-by-step progress with actionable failure copy.
**Notes:** Keep happy-path output understandable and redacted; detailed remote output should be optional.

---

## the agent's Discretion

- Exact privilege flow, as long as the runner service is non-root by default.
- Exact SSH implementation and final flag names.
- Exact supported distro/architecture list after research validates runner package and bootstrap behavior.
- Unselected gray areas: bootstrap/service defaults, registration/naming/labels after install, safety warning copy, completion summary, and BYO quickstart structure.

## Deferred Ideas

None.
