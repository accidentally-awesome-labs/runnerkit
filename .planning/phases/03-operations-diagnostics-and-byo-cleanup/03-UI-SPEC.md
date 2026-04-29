---
phase: 3
slug: operations-diagnostics-and-byo-cleanup
status: draft
shadcn_initialized: false
preset: none
created: 2026-04-29
---

# Phase 3 - UI Design Contract

> Visual and interaction contract for a CLI/TUI-adjacent operations phase. Phase 3 has no browser frontend; this contract covers terminal human output, JSON output, prompts, safety confirmations, diagnostic copy, severity/color, spacing, redaction, and progressive disclosure for `status`, `logs`, `doctor`, `recover`, and `down`.

---

## Design System

| Property          | Value |
| ----------------- | ----- |
| Tool              | Manual CLI terminal renderer (`internal/ui.Renderer`); no shadcn/browser design system |
| Preset            | not applicable |
| Component library | none; implement internal terminal primitives: health line, source matrix, finding list, log section, mutation plan, artifact prompt, partial-cleanup checkpoint, JSON response |
| Icon library      | none; use status glyphs with ASCII fallbacks |
| Font              | user terminal default monospace; RunnerKit must not force font family or size |

**shadcn gate:** not applicable. Repository scan found no `components.json`, Tailwind config, React/Vite/Next source, frontend components, or CSS. Phase 3 is CLI-only.

**Status glyph contract:** `✓` pass/ready/done, `!` warning/needs attention, `✗` error/broken/blocked, `?` prompt, `→` safest next action, `•` detail. In `TERM=dumb`, non-TTY, or ASCII mode, render `OK`, `WARNING`, `ERROR`, `PROMPT`, `NEXT`, and `-` instead.

**Health states:** use exactly these derived status states in human and JSON output: `ready`, `busy`, `needs_attention`, `broken`, `unknown`.

---

## Spacing Scale

Declared values are terminal character-cell tokens (multiples of 4) used for wrapping, indentation, and column layout.

| Token | Value    | Usage |
| ----- | -------- | ----- |
| xs    | 4 cells  | Nested evidence, remediation, log metadata, wrapped continuations |
| sm    | 8 cells  | Minimum gutter between source/status columns |
| md    | 16 cells | Prompt/value label width; source name column (`GitHub`, `Service`) |
| lg    | 24 cells | Health/action column; artifact action labels |
| xl    | 32 cells | Artifact value column for runner name, service, paths |
| 2xl   | 48 cells | Short diagnostic prose target before wrapping |
| 3xl   | 64 cells | Normal terminal prose max line target |

Exceptions: status prefixes may be 2 cells (`✓ `, `! `, `✗ `, `? `, `→ `); one blank line may separate major sections; code/log blocks may preserve source indentation but must be bounded by `--lines`/`--since`; never truncate JSON output.

---

## Typography

Terminal font size and physical line height are user-controlled. Hierarchy must be created with weight, casing, spacing, and prefixes only.

| Role    | Size                 | Weight              | Line Height |
| ------- | -------------------- | ------------------- | ----------- |
| Body    | terminal default 1em | regular (400)       | 1.5 logical rhythm via wrapping and section breaks |
| Label   | terminal default 1em | semibold/bold (600) | 1.2 logical rhythm |
| Heading | terminal default 1em | semibold/bold (600) | 1.2 logical rhythm |
| Display | terminal default 1em | semibold/bold (600) | 1.2 logical rhythm |

Use exactly two emphasis weights: regular and semibold/bold. Do not use italics, dim text, or color as the only hierarchy cue.

---

## Color

Color is progressive enhancement only. All meaning must remain visible when color is disabled.

| Role            | Value | Usage |
| --------------- | ----- | ----- |
| Dominant (60%)  | terminal default foreground/background; fallback `#E5E7EB` on `#111827` | Normal prose, facts, logs, prompts, JSON-free terminal output |
| Secondary (30%) | `#64748B` | Paths, timestamps, source names, metadata, de-emphasized collection warnings |
| Accent (10%)    | `#2563EB` | Current command heading, selected/default prompt marker, next-action arrow, copy-paste commands, saved `runs-on` snippet only |
| Destructive     | `#DC2626` | Host-key mismatch, destructive cleanup confirmations, failed artifact removal, blocked unsafe recovery |

Accent reserved for: command headings, selected/default prompt markers, `→` next-action lines, copy-paste command snippets, and the saved `runs-on` YAML snippet. Never color every interactive element accent.

TTY-only semantic colors may reinforce text: success `#16A34A`, warning `#D97706`, unknown `#64748B`. Always pair with glyphs and words (`ready`, `warning`, `error`, `unknown`).

---

## Copywriting Contract

| Element                  | Copy |
| ------------------------ | ---- |
| Primary CTA              | `Check runner status` (command: `runnerkit status`) |
| Empty state heading      | `No RunnerKit-managed runner found` |
| Empty state body         | `Run runnerkit up --repo owner/name --host user@host to create a BYO runner, or pass --all to list saved runners.` |
| Error state              | `RunnerKit can't determine runner health because {specific source failed}. {Exact fix instruction}. Re-run runnerkit status when ready.` |
| Destructive confirmation | `runnerkit down`: ask artifact-by-artifact with `{action}? [y/N]`; examples below |

Destructive confirmation copy:

| Action | Confirmation copy |
| ------ | ----------------- |
| Deregister GitHub runner | `Remove GitHub runner {runner_name} from {owner/repo}? [y/N]` |
| Remove host registration | `Unconfigure the runner registration on {machine_target}? [y/N]` |
| Stop/uninstall service | `Stop and uninstall service {service_name} on {machine_target}? [y/N]` |
| Remove runner files | `Remove RunnerKit install path {install_path} and work dir {work_dir}? [y/N]` |
| Remove local state | `Remove local RunnerKit state for {owner/repo}? [y/N]` |
| Stale GitHub-only deletion | `Delete stale GitHub runner {runner_id} for {owner/repo}? [y/N]` |

Tone rules:

- Be calm, operational, and specific; avoid hype.
- Every unhealthy state must end with exactly one safest next command.
- Use `down` for BYO cleanup; reserve `destroy` for future cloud resources.
- Status/logs/doctor copy must be read-only: never say `fixed`, `removed`, or `changed` from those commands.
- Recovery/down copy must name the exact artifact before mutation.
- Always repeat: `Do not use runs-on: self-hosted alone for RunnerKit-managed runners.` when showing a workflow snippet.

---

## Terminal Interaction Contract

### Command mutability

| Command | Default interaction | Mutation allowed? | Required UX contract |
| ------- | ------------------- | ----------------- | -------------------- |
| `runnerkit status` | Read-only, fast, repo-local by default | No | top-line health, compact source matrix, raw facts summary, saved `runs-on`, label drift warning, one safest next action |
| `runnerkit status --all` | Read-only inventory | No | one compact row/section per locally managed repo; do not prompt |
| `runnerkit logs` | Read-only bounded log view | No | default `--since 1h` and `--lines 200`; section logs by source; warn that redaction is best-effort before sharing |
| `runnerkit doctor` | Read-only deep diagnostics | No | stable finding IDs, severity, evidence, remediation commands; no `--fix` mutation |
| `runnerkit recover` | Guided mutation plan | Yes | show plan, support `--dry-run`, require confirmation or `--yes`, fail closed on SSH unreachable or host-key mismatch |
| `runnerkit down` | Guided cleanup plan | Yes | show artifact plan, ask per artifact interactively, support `--dry-run`, `--yes` safe default, record partial cleanup |

### Status output layout

Human `runnerkit status` output must use this order:

1. Heading: `Step 1 of 1: runner status`
2. Health line: `{glyph} Health: {state} — {one-sentence summary}`
3. Identity facts: repo, runner name, machine target, state path
4. Source matrix in order: `State`, `GitHub`, `SSH`, `Service`, `Labels`
5. Saved `runs-on` snippet and self-hosted-alone warning
6. One safest next action line; omit only when `ready` and not `busy`

Example:

```text
Step 1 of 1: runner status
! Health: needs_attention — GitHub reports the runner offline while SSH is reachable and the service is failed.
• Repository: owner/name
• Runner: runnerkit-owner-name-local
• Machine: user@host:22
• Sources:
    State       OK       saved runner metadata found
    GitHub     WARNING  offline, id 123, not busy
    SSH        OK       reachable, host key matched
    Service    ERROR    failed (runnerkit-runner)
    Labels     OK       saved labels match GitHub labels
• runs-on: [self-hosted, runnerkit, runnerkit-owner-name, linux, x64, persistent]
! Do not use runs-on: self-hosted alone for RunnerKit-managed runners.
→ Next: runnerkit doctor --repo owner/name
```

If terminal width is below 80 cells, render the matrix as key-value bullets instead of a dense table.

### Health state contract

| State | Human severity | Required summary behavior | Next action |
| ----- | -------------- | ------------------------- | ----------- |
| `ready` | success | GitHub online, not busy, SSH reachable, service active, labels match | no next action required; may show `Use the runs-on snippet above.` |
| `busy` | success/info | Runner is online and currently running a job | `Wait for the current GitHub Actions job, or inspect GitHub Actions if it appears stuck.` |
| `needs_attention` | warning | Recoverable drift such as offline runner, failed/inactive service, label drift, stale ID fallback | one of `runnerkit doctor`, `runnerkit logs`, or `runnerkit recover --dry-run` |
| `broken` | error/destructive | Unsafe or ambiguous condition such as host-key mismatch, duplicate candidates, service missing plus GitHub missing | safest non-mutating command, usually `runnerkit doctor`; never auto-recover |
| `unknown` | warning/secondary | Not enough evidence because state/auth/SSH/GitHub facts are missing or unavailable | fix missing input/auth/SSH or run explicit stale cleanup command |

### Logs output contract

- Default to `runnerkit logs --repo owner/name --since 1h --lines 200` behavior when flags are omitted.
- Section order: `collection summary`, `systemd journal`, `runner diag`, `collection warnings`, `next action`.
- Prefix each source with path/service metadata, then print bounded log excerpts.
- Do not print environment dumps or commands containing tokens.
- Always include: `Review logs before sharing; redaction is best-effort for workflow-produced secrets.`
- Partial failures are warnings, not fatal, when at least one log source is collected.

### Doctor output contract

Findings must use stable IDs and this shape:

```text
! service_failed (error)
    Evidence: systemd reports ActiveState=failed for runnerkit-runner.
    Remediation: runnerkit logs --repo owner/name --since 30m
```

Severity values: `pass`, `warning`, `error`. `pass` findings may be collapsed in human output unless `--verbose`; JSON must include all findings collected.

### Recovery and cleanup plans

Mutating commands must show a plan before mutation:

```text
Step 1 of 1: cleanup plan
! This will remove RunnerKit-managed runner artifacts for owner/name.
• GitHub runner: delete id 123 (runnerkit-owner-name-local)
• Host registration: unconfigure from /opt/actions-runner/runnerkit-owner-name-local
• Service: stop and uninstall runnerkit-runner
• Files: remove /opt/actions-runner/runnerkit-owner-name-local and /var/lib/runnerkit/work/runnerkit-owner-name-local
• Local state: remove owner/name after selected cleanup succeeds
→ Next: answer each prompt, pass --dry-run to preview only, or pass --yes for the safe default plan.
```

Rules:

- Interactive `down` asks each artifact prompt separately; defaults are **No**.
- `down --yes` applies only the safe default plan: GitHub runner record, recorded service, exact recorded install path, exact recorded work dir, matching local state. It must not delete shared users or shared `/var/lib/runnerkit` parents unless proven empty and exclusively RunnerKit-owned.
- `recover --yes` applies only the displayed recovery plan.
- `--dry-run` prints the plan and exits without mutation.
- If cleanup is partial, print `Cleanup incomplete` with completed/skipped/failed artifacts and keep/update local state with pending notes.

---

## Command Output and JSON Contract

### Stream rules

- Human-readable normal output: stdout.
- Warnings and errors: stderr in human mode.
- TTY-only spinners/transient progress: stderr only; never required for meaning.
- `--json`: stdout must contain JSON only; no ANSI, spinner frames, prose, or warnings outside JSON. Stderr may contain fatal process-level errors only.
- `NO_COLOR`, `CLICOLOR=0`, `TERM=dumb`, or `--no-color` disables ANSI color and animations.

### No-TTY and automation

- Read-only commands (`status`, `logs`, `doctor`) must not require prompts.
- Mutating commands in no-TTY must require enough flags plus `--yes`, or exit `6` with exact missing flags.
- Use existing exit code contract: `0` success, `1` unexpected, `2` invalid input, `3` GitHub auth/permission, `4` safety gate, `5` state I/O, `6` input required, `130` canceled.

### JSON shape: status

JSON must expose the same health model as human output plus raw source facts.

```json
{
  "ok": true,
  "command": "status",
  "scope": "repo",
  "repo": "owner/name",
  "state_path": "~/.local/state/runnerkit/state.json",
  "health": {
    "state": "needs_attention",
    "summary": "GitHub reports the runner offline while SSH is reachable and the service is failed.",
    "reasons": [
      {"id": "service_failed", "severity": "error", "source": "systemd"}
    ],
    "next_actions": [
      {"command": "runnerkit doctor --repo owner/name", "why": "Inspect service logs before recovery."}
    ]
  },
  "runner": {
    "name": "runnerkit-owner-name-local",
    "labels": ["self-hosted", "runnerkit", "runnerkit-owner-name", "linux", "x64", "persistent"],
    "workflow_snippet": "runs-on: [self-hosted, runnerkit, runnerkit-owner-name, linux, x64, persistent]"
  },
  "sources": {
    "state": {"present": true},
    "github": {"found": true, "id": 123, "status": "offline", "busy": false, "labels": ["self-hosted", "runnerkit"]},
    "ssh": {"reachable": true, "host_key": "matched"},
    "systemd": {"service": "runnerkit-runner", "active_state": "failed", "sub_state": "failed"},
    "labels": {"match": false, "missing": ["runnerkit-owner-name"], "extra": []}
  },
  "redactions_applied": true
}
```

### JSON shape: logs, doctor, recover, down

| Command | Required top-level keys |
| ------- | ----------------------- |
| `logs` | `ok`, `command`, `repo`, `state_path`, `since`, `lines`, `sections[]`, `warnings[]`, `redactions_applied` |
| `doctor` | `ok`, `command`, `repo`, `state_path`, `health`, `findings[]`, `next_actions[]`, `redactions_applied` |
| `recover` | `ok`, `command`, `repo`, `dry_run`, `plan`, `results[]`, `state_updated`, `github_runner_id`, `redactions_applied` |
| `down` | `ok`, `command`, `repo`, `dry_run`, `plan`, `results[]`, `partial_cleanup`, `pending[]`, `state_removed`, `redactions_applied` |

All keys must be stable `snake_case`. Error JSON must use existing renderer shape with `error.code`, `error.message`, and `error.remediation[]`.

---

## Safety and Redaction Contract

### Redaction display

| Sensitive value | Display as |
| --------------- | ---------- |
| GitHub auth token or PAT | `<redacted:github-token>` |
| Runner registration token | `<redacted:runner-registration-token>` |
| Runner removal token | `<redacted:runner-removal-token>` |
| SSH private key material | `<redacted:ssh-private-key>` |
| Provider credential/API token | `<redacted:provider-credential>` |
| Sensitive machine/provider identifier in diagnostics | `<redacted:machine-ref>` |

Rules:

- Route all human output, JSON, remote command output, journal snippets, runner `_diag` logs, errors, and partial-cleanup notes through the shared redactor.
- Register fresh registration/removal tokens with the redactor immediately after creation; never persist tokens.
- Do not print command lines that include tokens. Prefer stdin/protected temp files for remote token use.
- Log collection is bounded by default and must not dump environment variables.
- Because workflow-produced secrets may appear in runner logs, always warn users to review collected logs before sharing.
- Redacted output must preserve value kind and action context so troubleshooting remains possible.
- There is no unredacted debug mode in Phase 3.

### Safety warnings

Use text-first warning blocks for unsafe states:

```text
WARNING: SSH host key changed
RunnerKit stopped because the saved host key fingerprint does not match the current host.
→ Verify the machine identity before running recover or down against this host.
```

Status, logs, and doctor must remain read-only even when warnings identify a clear fix.

---

## Progressive Disclosure Contract

| Layer | User question answered | Detail level |
| ----- | ---------------------- | ------------ |
| `status` | `Is this runner ready? If not, what do I run next?` | fastest probes, compact facts, one next action |
| `logs` | `What recent evidence exists?` | bounded redacted journal and runner logs |
| `doctor` | `Why is it unhealthy and how do I fix it?` | deeper checks, stable findings, exact remediation commands |
| `recover --dry-run` | `What would RunnerKit change to repair this?` | mutation plan without changes |
| `recover --yes` | `Repair the selected safe path.` | confirmed/automated mutation with verification |
| `down --dry-run` | `What would cleanup remove?` | artifact plan without changes |
| `down` / `down --yes` | `Remove RunnerKit-managed BYO artifacts safely.` | artifact confirmations or safe default cleanup |

Never put deep disk/tool/network/log checks in default `status`. Never hide destructive scope behind a terse confirmation.

---

## Registry Safety

| Registry        | Blocks Used | Safety Gate |
| --------------- | ----------- | ----------- |
| shadcn official | none | not applicable - CLI phase with no browser registry, 2026-04-29 |
| third-party     | none | not applicable - no registry blocks declared, 2026-04-29 |

---

## Source Decisions Used

| Source | Decisions Applied |
| ------ | ----------------- |
| `.planning/phases/03-operations-diagnostics-and-byo-cleanup/03-CONTEXT.md` | status defaults to current repo; `--all` inventory; read-only status; fast probes; top-line health/source matrix/next action; derived health plus raw facts; saved `runs-on`; label drift; JSON parity; `runnerkit down`; artifact-by-artifact cleanup prompts; `down --yes` safe defaults |
| `.planning/phases/03-operations-diagnostics-and-byo-cleanup/03-RESEARCH.md` | shared operations model; exact health states; source order; logs bounded and redacted; doctor finding shape; recover/down plan-confirm behavior; partial cleanup contract; validation-oriented JSON fields |
| `.planning/REQUIREMENTS.md` | GH-03, REL-01, REL-02, REL-03, REL-04, CLEAN-02, CLEAN-03, STATE-01, STATE-02 |
| `.planning/ROADMAP.md` | Phase 3 scope is managed BYO persistent operations, diagnostics, recovery, and cleanup; cloud destroy and ephemeral mode deferred |
| `.planning/STATE.md` / `.planning/PROJECT.md` | CLI-only v1, solo-developer focus, no workflow YAML edits, persistent trusted-private default, non-root service, redaction-first output |
| Repository scan | no frontend design system; existing `internal/ui.Renderer` supports human/JSON output, glyphs, wrapping, ASCII fallback, and redaction flag; existing exit code contract reused |

---

## Checker Sign-Off

- [ ] Dimension 1 Copywriting: PASS
- [ ] Dimension 2 Visuals: PASS
- [ ] Dimension 3 Color: PASS
- [ ] Dimension 4 Typography: PASS
- [ ] Dimension 5 Spacing: PASS
- [ ] Dimension 6 Registry Safety: PASS

**Approval:** pending
