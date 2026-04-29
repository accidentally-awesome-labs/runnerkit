---
phase: 1
slug: cli-auth-state-and-safety-foundation
status: approved
reviewed_at: 2026-04-29
shadcn_initialized: false
preset: none
created: 2026-04-29
---

# Phase 1 - UI Design Contract

> Visual and interaction contract for a CLI/terminal phase. Phase 1 has no browser frontend; this contract covers `runnerkit` terminal prompts, guided setup flow, human and JSON output, safety warnings, redaction display, spacing, color, and no-TTY behavior.

---

## Design System

| Property          | Value                                                                                                                                      |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Tool              | Manual CLI terminal renderer; no shadcn/browser design system                                                                              |
| Preset            | not applicable                                                                                                                             |
| Component library | none; implement internal terminal primitives: heading, step list, prompt, warning, plan preview, state preview, error block, JSON response |
| Icon library      | none; use status glyphs with ASCII fallbacks                                                                                               |
| Font              | user terminal default monospace; RunnerKit must not force font family or size                                                              |

**Status glyph contract:** `✓` success, `!` warning, `✗` error, `?` prompt, `→` next action, `•` bullet. In `TERM=dumb`, non-TTY, or ASCII mode, render `OK`, `WARNING`, `ERROR`, `PROMPT`, `NEXT`, and `-` instead.

---

## Spacing Scale

Declared values are terminal character-cell tokens (multiples of 4) used for wrapping, indentation, and column layout.

| Token | Value    | Usage                                                       |
| ----- | -------- | ----------------------------------------------------------- |
| xs    | 4 cells  | Nested details under a step, wrapped continuation indent    |
| sm    | 8 cells  | Minimum gutter between table columns                        |
| md    | 16 cells | Prompt/value label width and compact metadata columns       |
| lg    | 24 cells | Standard status/action column width                         |
| xl    | 32 cells | Standard table value column width                           |
| 2xl   | 48 cells | Short prose line target before wrapping in narrow terminals |
| 3xl   | 64 cells | Wizard prose max line target on normal terminals            |

Exceptions: one blank line may separate major sections; prompt prefixes may be 2 cells (`? `); status prefixes may be 2 cells (`✓ `, `! `, `✗ `); never truncate JSON output.

---

## Typography

Terminal font size and physical line height are user-controlled. Hierarchy must be created with weight, casing, spacing, and prefixes only.

| Role    | Size                 | Weight              | Line Height                                                        |
| ------- | -------------------- | ------------------- | ------------------------------------------------------------------ |
| Body    | terminal default 1em | regular (400)       | 1.5 logical rhythm via wrapped lines and blank-line section breaks |
| Label   | terminal default 1em | semibold/bold (600) | 1.2 logical rhythm                                                 |
| Heading | terminal default 1em | semibold/bold (600) | 1.2 logical rhythm                                                 |
| Display | terminal default 1em | semibold/bold (600) | 1.2 logical rhythm                                                 |

Use exactly two emphasis weights: regular and semibold/bold. Do not use italics, dim text, or color as the only hierarchy cue.

---

## Color

Color is progressive enhancement only. All meaning must remain visible when color is disabled.

| Role            | Value                                                                   | Usage                                                                                                     |
| --------------- | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| Dominant (60%)  | terminal default foreground/background; fallback `#E5E7EB` on `#111827` | Normal prose, prompts, state values, tables                                                               |
| Secondary (30%) | `#64748B`                                                               | Muted helper text, paths, metadata, de-emphasized explanations                                            |
| Accent (10%)    | `#2563EB`                                                               | Current step heading, selected/default option marker, next-action arrow, copy-paste command snippets only |
| Destructive     | `#DC2626`                                                               | Blocked safety gates, failed checks, destructive/danger confirmations only                                |

Accent reserved for: active wizard step heading, selected/default option marker, `→` next-action line, and command snippets the user should copy. Never color every interactive element accent.

TTY-only semantic colors may be used for status reinforcement: success `#16A34A`, warning `#D97706`. They must always be paired with `✓`/`WARNING:`/`ERROR:` text.

---

## Copywriting Contract

| Element                  | Copy                                                                                                                                                   |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Primary CTA              | `Prepare foundation` (command entry point: `runnerkit up`)                                                                                             |
| Empty state heading      | `No RunnerKit state found`                                                                                                                             |
| Empty state body         | `Run runnerkit up to choose a repository, verify GitHub auth, review safety checks, and save foundation state. Phase 1 does not install a runner yet.` |
| Error state              | `RunnerKit can't continue because {specific problem}. {Exact fix instruction}. Re-run {command} when ready.`                                           |
| Destructive confirmation | `Replace saved foundation state`: `Type replace {owner/repo} to overwrite the existing RunnerKit state for this repository.`                           |

Tone rules:

- Be direct, calm, and security-conscious; avoid hype.
- Never imply a runner exists in Phase 1. Use `foundation ready`, not `runner ready`.
- Prefer action verbs: `Choose repository`, `Verify GitHub auth`, `Review safety`, `Save state`.
- Every error must include a next step; every warning must explain the risk and the safe alternative.

---

## Terminal Interaction Contract

### Guided setup order

`runnerkit up` must present the first-run wizard in this order:

1. **Welcome** - explain RunnerKit will prepare CLI/auth/state foundations and will not install a runner in Phase 1.
2. **Prerequisites** - show required local prerequisites before changes: Git repository, GitHub remote or `--repo`, `gh` auth or fine-grained token path, writable state directory.
3. **Repo/auth** - auto-detect GitHub repo from git remote, then require explicit confirmation before auth or state writes apply to it.
4. **Safety checks** - check repository visibility and obvious fork/public risk; fail closed or require explicit danger override for risky persistent setup.
5. **State preview** - show exactly what will be saved, where it will be saved, and what will not be saved.
6. **Next steps** - confirm foundation state is ready and state that runner installation comes in later phases.

### Prompt rules

- Default to interactive guidance only when stdin and stdout are TTYs.
- Before any state write, show a plan/checklist and ask `Save this foundation state? [y/N]`; default is **No** for mutations.
- Auto-detected repo confirmation may default to yes, but no auth/state action may occur before the user confirms the repo.
- Risk and destructive confirmations must never default to yes. Use typed confirmation for replacement and danger overrides.
- `Ctrl-C` must exit cleanly. If no state was written, print `Canceled; no changes made.` If state was written, print the state path and the safe rerun command.

### Human output layout

Human output should be vertically scanned, not dense. Use this sequence for each major step:

```text
Step 3 of 6: Verify GitHub auth
✓ Found gh authentication for github.com
! Token is missing repository runner-management permission
→ Fix: refresh gh auth or use a fine-grained token for owner/repo
```

A state preview must include:

- repository scope (`owner/repo`)
- auth source reference (`gh`, `fine-grained-token`, or `env`) without token material
- state path and optional project config path
- planned runner name and label convention
- safety status
- explicit `Will not install a runner in Phase 1`

### Label/snippet display

Phase 1 should establish label copy without encouraging generic routing. Recommended preview:

```yaml
runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]
```

Always print: `Do not use runs-on: self-hosted alone for RunnerKit-managed runners.`

---

## Command Output and JSON Contract

### Stream rules

- Human-readable normal output: stdout.
- Progress spinners and transient redraws: stderr, TTY only.
- Errors and warnings: stderr in human mode.
- `--json`: stdout must contain JSON only; no ANSI, spinner frames, prose, or warnings outside JSON. Stderr may contain fatal process-level errors only.

### No-TTY and automation

- If a prompt would be required and no TTY is available, exit with code `6` and explain the missing flags.
- Non-interactive persistence requires all required inputs plus `--yes`.
- `--dry-run` may run without `--yes` and must not write state.
- `NO_COLOR`, `CLICOLOR=0`, `TERM=dumb`, or `--no-color` disables ANSI color and animations.

### JSON shape

Success responses must use stable snake_case keys and never include secrets:

```json
{
  "ok": true,
  "command": "up",
  "repo": "owner/name",
  "auth_source": "gh",
  "state_path": "~/.local/state/runnerkit/state.json",
  "runner_installed": false,
  "warnings": [],
  "next_steps": [
    {
      "label": "Review saved state",
      "command": "runnerkit state show --repo owner/name"
    }
  ],
  "redactions_applied": true
}
```

Error responses:

```json
{
  "ok": false,
  "error": {
    "code": "github_permission_denied",
    "message": "RunnerKit can't create a repository runner registration token for owner/name.",
    "remediation": [
      "Use gh auth with the required repository permission or create a fine-grained token scoped to owner/name."
    ]
  },
  "redactions_applied": true
}
```

Exit codes: `0` success, `1` unexpected error, `2` invalid input/flags, `3` GitHub auth or permission failure, `4` safety gate blocked, `5` state read/write failure, `6` interactive input required but unavailable, `130` canceled by user.

---

## Safety and Redaction Contract

### Safety warning format

Use a text-first warning block with a specific risk and safe alternative:

```text
WARNING: Public repository risk
Persistent self-hosted runners can execute untrusted workflow code from forks or public contributors.
RunnerKit will not continue with persistent setup unless you choose a safer mode or pass --allow-public-repo-risk.
```

Phase 1 must establish the gate even if persistent setup is implemented later.

### Redaction display

| Sensitive value                                   | Display as                             |
| ------------------------------------------------- | -------------------------------------- |
| GitHub auth token or PAT                          | `<redacted:github-token>`              |
| Runner registration token                         | `<redacted:runner-registration-token>` |
| Runner removal token                              | `<redacted:runner-removal-token>`      |
| SSH private key material                          | `<redacted:ssh-private-key>`           |
| Provider credential/API token                     | `<redacted:provider-credential>`       |
| Sensitive host/provider identifier in diagnostics | `<redacted:machine-ref>`               |

Rules:

- Never print durable GitHub tokens, registration/removal tokens, SSH private keys, provider credentials, or command lines containing those values.
- Redaction applies to human output, JSON, logs, debug output, state previews, and errors.
- Redacted output should preserve the kind of value and the surrounding action so the user can still debug.
- There is no unredacted debug mode in Phase 1.

---

## Accessibility and Narrow-Terminal Contract

- Do not rely on color, animation, cursor position, or Unicode symbols for meaning.
- Render a complete static line for each completed step so screen readers and logs retain context.
- Wrap prose at 64 cells by default and never require wider than 80 columns. If width is below 80, switch tables to key-value lists.
- All prompts must be keyboard-only and have non-interactive flag equivalents.
- Avoid timed prompts. Do not auto-advance safety decisions.
- JSON output is the accessibility and automation fallback for parsers, CI, and no-TTY use.

---

## Registry Safety

| Registry        | Blocks Used | Safety Gate                                                     |
| --------------- | ----------- | --------------------------------------------------------------- |
| shadcn official | none        | not applicable - CLI phase with no browser registry, 2026-04-29 |
| third-party     | none        | not applicable - no registry blocks declared, 2026-04-29        |

---

## Source Decisions Used

| Source                                                                    | Decisions Applied                                                                                                                                                                                  |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `.planning/phases/01-cli-auth-state-and-safety-foundation/01-CONTEXT.md`  | richer CLI wizard, `runnerkit up`, plan/confirm before mutation, `gh` first, fine-grained token fallback, repo confirmation, fail-closed permissions, public repo safety gate, state preview order |
| `.planning/REQUIREMENTS.md`                                               | CLI-01, CLI-02, GH-01, STATE-01, STATE-02                                                                                                                                                          |
| `.planning/ROADMAP.md`                                                    | Phase 1 is foundation only; no runner install/provisioning/cleanup implementation yet                                                                                                              |
| `.planning/phases/01-cli-auth-state-and-safety-foundation/01-RESEARCH.md` | Go/Cobra CLI, redaction-first, JSON/non-interactive output, stable label conventions, idempotent plan/checkpoint primitives                                                                        |
| Repository scan                                                           | no existing app source, design system, shadcn config, Tailwind config, or frontend components found                                                                                                |

---

## Checker Sign-Off

- [ ] Dimension 1 Copywriting: PASS
- [ ] Dimension 2 Visuals: PASS
- [ ] Dimension 3 Color: PASS
- [ ] Dimension 4 Typography: PASS
- [ ] Dimension 5 Spacing: PASS
- [ ] Dimension 6 Registry Safety: PASS

**Approval:** pending
