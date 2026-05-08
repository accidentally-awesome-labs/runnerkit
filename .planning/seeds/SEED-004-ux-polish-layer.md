---
id: SEED-004
status: dormant
planted: 2026-05-08
planted_during: v1.0.0 / Phase 06 (release-upgrade-docs-and-v1-validation, attempt-19 smoke-red)
trigger_when: starting v1.2 milestone OR any milestone scoped at "UX polish" / "first-run experience" / "agent-friendly output" / "doctor auto-fix" / "every command shows next action" / depends on SEED-001 having landed
scope: medium
---

# SEED-004: UX polish layer — boxed renderer, state machine, first-run wizard, --explain, doctor --fix, progress checklists

## Why This Matters

SEED-001 establishes the **data layer** (`next_actions` JSON contract, bootstrap/lifecycle split). SEED-002 establishes the **CLI verb layer** (`register / unregister / list`). SEED-003 establishes the **plugin layer** (skill + slash commands + hook).

None of those alone make the CLI feel "easy to follow at every step". The user-facing polish — boxed copy-pasteable commands, stage-aware messaging, progress checklists, `runnerkit` no-args first-run wizard, `--explain` "why am I doing this", `doctor --fix` auto-remediation — is its own deliberate UX layer. Without it the architecture is correct but the experience is bare-CLI.

This seed captures that layer as a single coherent unit so it ships with intent (not piecemeal as polish on individual commands).

The plugin layer (SEED-003) **consumes** this layer — `/runnerkit-onboard` is the chat-mode equivalent of `runnerkit` no-args wizard; `/runnerkit-doctor --fix` is the chat wrapper around `runnerkit doctor --fix`. Without SEED-004 those slash commands have nothing useful to wrap.

## When to Surface

**Trigger:** v1.2 milestone scope OR any of:
- "UX polish" / "make CLI easier to follow"
- "first-run experience" / "guided onboarding"
- "agent-friendly output" / "next-step always shown"
- "doctor auto-fix" / "self-remediation"
- "progress visualization" / "step-by-step CLI"
- "--explain flag" / "why am I being asked to do this"

**Hard prerequisite:** SEED-001 (bootstrap/lifecycle split + `next_actions` contract). The polish layer is a renderer over the contract; the contract has to exist first.

**Sequencing relative to other seeds:**
- SEED-001 → SEED-004 (this) → SEED-002 (multi-repo, benefits from polished register/list output) → SEED-003 (plugin, wraps the polished CLI for chat)
- OR: SEED-001 → SEED-002 → SEED-004 → SEED-003 (if multi-repo ships before polish; the wizard then onboards multi-repo natively)

Either order works. SEED-004 must come after SEED-001 and before SEED-003.

## Scope Estimate

**Medium** — one phase, ~1000 LOC. Six discrete deliverables that share a common foundation (the state machine):

### 1. State machine — single source of truth for "what stage is the host in?"

`internal/ux/stage/` package with:
- `Stage` enum: `UNINSTALLED | INSTALLED | REGISTERED | RUNNING | ERROR`
- `Detect(ctx, host) (Stage, []string, error)` — probes host (SSH non-root) + local state, returns current stage + the data the renderer needs (e.g., list of registered repos, service health, last job timestamp)
- `Next(stage) []NextAction` — given a stage, returns the canonical next-action list

Every command resolves stage first, branches off the result. **No more "why is this asking for sudo?" surprises**: the user always sees a `STAGE` line in output and the next-action options match the stage.

### 2. Boxed-command terminal renderer

`internal/ui/box.go`:
- `RenderBoxed(cmd string, host string, why string) string` — produces:
  ```
  Copy and paste this on salar@mckee-small-desktop:

  ┌──────────────────────────────────────────────────────────────────┐
  │ curl -fL https://runnerkit.dev/install.sh | sudo bash            │
  └──────────────────────────────────────────────────────────────────┘

  You'll be prompted for your sudo password once.
  ```
- ASCII-only by default; UTF-8 box-drawing under `--unicode` or auto-detected `LANG`
- Width-aware wrapping (terminal width via `term.GetSize`)
- `--no-color` and `--json` modes both supported

The boxed renderer is the **canonical surface for any "user runs this on a remote host" output** — every place that today says "Run runnerkit byo-prepare ..." in remediation strings becomes a boxed-command emit.

### 3. Progress checklists `[✓] [→] [ ]`

`internal/ui/checklist.go`:
- `Checklist` struct — append items as work progresses; persists to `~/.runnerkit/sessions/<id>.json` so resumption restores the exact stage
- Renderer:
  ```
  [✓] Detect host OS                               (200ms)
  [✓] Detect GitHub auth                           (300ms)
  [✓] Check host stage: UNINSTALLED
  [→] Awaiting host install — paste this command:
      <boxed command>
  [ ] Verify install (will probe after you confirm)
  [ ] Generate registration token
  [ ] Register with GitHub
  [ ] Start service
  ```
- Idempotent: re-running a command picks up the existing checklist, marks already-done steps `[✓]` without redoing them, continues from `[→]`
- `--json` emits the same data as a structured array

### 4. First-run wizard — `runnerkit` no-args entry point

New top-level command in `internal/cli/wizard.go`:
- Triggered by `runnerkit` (zero args, zero flags)
- Detects: no saved hosts, no `~/.runnerkit/config.json`
- Asks (in order, with sensible defaults):
  - "Where do you want to run jobs?" — BYO / Hetzner cloud
  - "Enter SSH target (user@host)" — validates with quick `ssh -o BatchMode=yes` probe
  - "Which repo?" — autocompletes from `gh repo list` if `gh` available
  - "Ready to set up?" → runs the rest of the flow with checklist
- After completion prints the bookmark card:
  ```
  Bookmark these commands:

    runnerkit register --repo NEW            ← add repos to this host
    runnerkit status                          ← see all runners
    runnerkit doctor                          ← if anything looks wrong
  ```

### 5. `--explain` flag

Adds an `--explain` flag to every subcommand. When present:
- Before running, prints a short "WHY: ... / RUNS: ... / TAKES: ..." block per step
- Block content lives next to each step's implementation in source (Go string constants), not in a separate doc
- Reduces cognitive load for power-users: opt-in once, never again
- Example for `runnerkit init`:
  ```
  Step 1: install runnerkit on the host
    WHY: the host needs a runnerkit-runner system user, a scoped sudoers
         fragment (so future runnerkit operations don't need your password),
         and a base install dir. This is a one-time setup per host.
    RUNS: see install.sh source at github.com/aal/runnerkit/.../install.sh
    TAKES: ~30 seconds, mostly downloading the GitHub Actions runner binary.
  ```

### 6. `doctor --fix` — auto-remediation

`internal/cli/doctor.go` extension:
- Each `doctor` finding already has a remediation string (Plan 06-03 RKD-XXX-NNN system)
- Add a `Fixable bool` and `FixCommand func(ctx) error` field per finding
- `runnerkit doctor` (default) — show findings + remediations, exit non-zero if any blocker
- `runnerkit doctor --fix` — for each fixable finding, prompt y/n, run `FixCommand`, report
- `runnerkit doctor --fix --yes` — non-interactive auto-fix (for agent automation)
- `runnerkit doctor --ignore <code>` — suppress a specific finding (persists in `~/.runnerkit/config.json`)

Example fixable findings:
- Stale GitHub runner registration (host gone, runner offline) → `runnerkit unregister --gh-only --repo X`
- Orphaned install dir on host (registered → no GH-side runner) → re-register or remove
- Outdated runnerkit-runner binary (Plan 02-02 pinned hash drift) → `runnerkit upgrade-runner`

## Breadcrumbs

Code paths that this seed touches (some new, some refactored):

New packages:
- `internal/ux/stage/` — state machine
- `internal/ux/nextaction/` — created in SEED-001; this seed extends with `Box`, `Checklist` rendering helpers
- `internal/ui/box.go` — boxed-command renderer
- `internal/ui/checklist.go` — progress checklist renderer
- `internal/cli/wizard.go` — first-run wizard

Existing files modified:
- `internal/ui/renderer.go` — extend `Renderer` to consume `next_actions` for success paths (currently focused on errors)
- `internal/cli/root.go` — wire wizard as default `runnerkit` no-args
- `internal/cli/doctor.go` — `--fix`, `--yes`, `--ignore` flags + Fixable interface for findings
- All subcommands — `--explain` flag plumbed via cobra PersistentFlags
- `internal/errcodes/codes.go` — already has RKD-XXX-NNN; extend each code with optional `Fixable bool` + reference to a `FixCommand` registry

Tests:
- Stage detection: integration tests against fresh + bootstrapped + registered Docker hosts
- Boxed renderer: golden file tests for ASCII / Unicode / `--no-color` / `--json` modes
- Wizard: scripted-input integration tests (use `expect`-style fixture or testable Reader injection)
- Checklist: persistence + resumption tests (kill mid-flight, restart, verify state)
- Doctor --fix: each fixable finding has a corresponding test that injects the broken state, runs `--fix`, verifies recovery

Related decisions:
- D-04 (live BYO smoke) — smoke harness consumes the new `next_actions` JSON via `--json` + checklist for assertion structure
- DOC-04 (cleanup/troubleshooting docs) — `doctor --fix` consumes the same RKD-XXX-NNN registry the troubleshooting docs already reference
- Phase 2 service-must-not-run-as-root — preserved (no change to runtime user model)

## Notes

The unifying mental model: **every command becomes a thin renderer over a (state, next_actions) tuple**. The CLI in source stops being "lots of imperative steps inline" and starts being "compute stage → emit next_actions → render to terminal/JSON". This is what makes everything else clickable: agents, plugins, slash commands, the human user — they all consume the same tuple, render it differently.

`--explain` is intentionally low-effort to add per-command (just a string constant per step), but high-value for the "I'm new to runnerkit and want to understand what's happening" user. It's the antidote to "wait, why does it want sudo?" friction.

`doctor --fix` is the moment runnerkit becomes proactive instead of reactive. Today the user has to run doctor, read the findings, understand each remediation, run the fix command. With `--fix` they say "make it right" once.

Cross-refs:
- SEED-001 (bootstrap/lifecycle split) — hard prerequisite (gives us the JSON contract this layer renders)
- SEED-002 (multi-repo per host) — mutual benefit (multi-repo register/list output looks much better with the polish layer)
- SEED-003 (Claude Code plugin) — direct consumer (slash commands wrap CLI surfaces this seed creates)
