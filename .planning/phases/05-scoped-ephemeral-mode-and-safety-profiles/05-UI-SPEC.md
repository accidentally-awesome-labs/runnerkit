---
phase: 5
slug: scoped-ephemeral-mode-and-safety-profiles
status: approved
reviewed_at: 2026-05-01
shadcn_initialized: false
preset: none
created: 2026-05-01
---

# Phase 5 - UI Design Contract

> Visual and interaction contract for the RunnerKit CLI ephemeral mode and safety profile experience. Generated for GSD UI phase, verified against the six UI quality dimensions.

---

## Scope

Phase 5 is a CLI-first interaction and documentation phase, not a web frontend phase. This contract locks the terminal output, prompts, safety copy, JSON parity, documentation snippets, and semantic hierarchy for:

- `runnerkit up` mode/profile selection before choosing persistent or ephemeral runners.
- Persistent-vs-ephemeral tradeoff explanations for cost, isolation, cleanup, operations, and log preservation.
- Public/fork/untrusted workload safety warnings and recommendations.
- Ephemeral setup output for BYO and Hetzner cloud paths.
- Ephemeral TTL, finalizer, log preservation, and cleanup status surfaces.
- `runnerkit status`, `runnerkit logs`, `runnerkit doctor`, `runnerkit down`, and `runnerkit destroy` copy for ephemeral states.
- README, cloud/BYO docs updates, and `docs/safety.md` safety guidance.

All implementation must preserve existing RunnerKit UI principles:

- CLI-only for v1.
- Human output through `internal/ui.Renderer` primitives.
- JSON output with stable fields and `redactions_applied: true`.
- No workflow YAML editing.
- No decorative TUI, animation, web UI, shadcn, or third-party terminal framework.
- No secret, registration-token, removal-token, provider-token, or machine-sensitive value leaks.

---

## Design System

| Property          | Value                                                                                 |
| ----------------- | ------------------------------------------------------------------------------------- |
| Tool              | Manual CLI design system using Cobra commands and `internal/ui.Renderer`              |
| Preset            | not applicable                                                                        |
| Component library | none; use `ui.Renderer`, `ui.Line`, Cobra flags, markdown docs, and JSON payloads     |
| Icon library      | none; use existing renderer glyphs `✓`, `!`, `✗`, `?`, `→`, `•` with ASCII fallbacks  |
| Font              | Terminal default monospace for CLI; documentation examples use fenced markdown blocks |

### Source-of-truth patterns

- Human output must use existing renderer primitives: `Step`, `Warning`, `Error`, `Success`, `WarningLine`, `ErrorLine`, `PromptLine`, `Next`, and `Bullet`.
- JSON output must expose structured equivalents for every human-visible choice, warning, cleanup command, TTL, log path, safety profile, and runner mode.
- Do not introduce a TUI, React, shadcn, third-party terminal UI toolkit, or decorative animation in Phase 5.
- Prefer short, concrete bullets over paragraphs in command output.
- Long risk explanations belong in docs and may be summarized in terminal output with a next command or doc path.

---

## Visual Hierarchy and Interaction States

| Surface                        | Focal point                                                                    | Visual hierarchy                                                                       | Required states                                                               |
| ------------------------------ | ------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| Mode/profile selection         | The runner mode chosen for `owner/name`                                        | Step title → plain-language tradeoff bullets → options → next action                   | persistent default, ephemeral BYO, ephemeral cloud, canceled                  |
| Persistent safety warning      | Why persistent runners are risky for public/fork workflows                     | WARNING heading → exact risk → safer command → dangerous override                      | blocked, override accepted, JSON error                                        |
| Ephemeral setup plan           | One-job behavior plus cleanup/log responsibilities                             | Step title → one-job guarantee → TTL/log preservation → labels → cleanup command       | dry-run, confirmation required, setup in progress, complete                   |
| Ephemeral cloud plan           | Stronger isolation with billable cleanup                                       | Cost warning first → one-job copy → TTL/log bullets → labels snippet → destroy command | dry-run, confirmed, readiness failed, cleanup pending                         |
| Ephemeral completion           | What user can run next                                                         | Success line → runner name/labels → one-job/TTL → logs path → cleanup command          | waiting for job, complete, TTL active                                         |
| Status/doctor ephemeral states | Whether one-job runner is waiting, busy, complete, expired, or cleanup-pending | Health summary → source facts → mode-specific next action                              | waiting, busy, completed_needs_cleanup, ttl_expired, cleanup_pending, unknown |
| Logs                           | Preserved logs before/after runner cleanup                                     | Source title → preserved archive path → runner/journal sections → warnings             | live logs, preserved logs, missing archive, SSH unreachable                   |
| Safety guide docs              | Which mode to choose                                                           | Recommendation table → commands → caveats → cleanup/log guidance                       | trusted private, public/fork, BYO, cloud, untrusted                           |

Accessibility contract:

- Icon/glyph output must always include text labels such as `OK`, `WARNING`, `ERROR`, or descriptive prose; glyphs are never the only signal.
- All prompts must be answerable from text alone in non-color terminals.
- ANSI color must be optional. Meaning must come from words, structure, and JSON fields.
- JSON output must include `mode`, `safety_profile`, `ephemeral`, `ttl`, `log_archive`, `cleanup_command`, `tradeoffs`, and `warnings` wherever the human output shows those facts.

---

## Spacing Scale

Declared values (must be multiples of 4):

| Token | Value | Usage                                                                      |
| ----- | ----- | -------------------------------------------------------------------------- |
| xs    | 4px   | Inline symbol/text gap; one terminal prefix gap after glyph or ASCII label |
| sm    | 8px   | Compact bullet grouping; adjacent command examples in docs                 |
| md    | 16px  | Default separation between step title, bullets, and next action            |
| lg    | 24px  | Section padding before warnings, tradeoff groups, and docs tables          |
| xl    | 32px  | Separation between setup, operation, and cleanup command groups            |
| 2xl   | 48px  | Major docs section breaks, especially safety guide recommendation blocks   |
| 3xl   | 64px  | Page-level spacing only in long-form docs, not normal CLI output           |

Exceptions: terminal tables and markdown pipes use character alignment rather than pixel spacing; this is allowed only for tradeoff tables, safety-profile tables, requirement traceability tables, and docs tables where column alignment is the visual goal.

---

## Typography

| Role    | Size | Weight | Line Height |
| ------- | ---- | ------ | ----------- |
| Body    | 16px | 400    | 1.5         |
| Label   | 14px | 400    | 1.4         |
| Heading | 20px | 600    | 1.25        |
| Display | 28px | 600    | 1.2         |

Typography rules:

- Use exactly these four semantic roles for docs and any future rendered UI; terminal output maps them to plain text headings, labels, and bullets.
- Use only two weights: regular `400` and semibold `600`.
- CLI command names, flags, env vars, profile names, runner modes, provider IDs, labels, resource IDs, TTL values, and paths must be wrapped in backticks in docs and appear as plain unstyled literals in terminal output.
- Avoid all-caps except stable status words already used by the renderer (`OK`, `WARNING`, `ERROR`, `NEXT`) and environment variable names such as `HCLOUD_TOKEN`.

---

## Color

| Role            | Value   | Usage                                                                                                                      |
| --------------- | ------- | -------------------------------------------------------------------------------------------------------------------------- |
| Dominant (60%)  | #111827 | Primary text and high-contrast terminal/docs foreground intent                                                             |
| Secondary (30%) | #F3F4F6 | Markdown code blocks, tables, muted docs backgrounds, and secondary surfaces                                               |
| Accent (10%)    | #2563EB | Primary recommended action text: `Use ephemeral cloud runner`, exact `runs-on` snippet callout, and future focus ring only |
| Destructive     | #DC2626 | Persistent-risk warnings, public/fork risk blocks, destructive cleanup confirmations, and TTL-expired cleanup warnings     |

Accent reserved for: `Use ephemeral cloud runner`, active setup-path selection, exact `runs-on` snippet callout, and focus ring in any future richer prompt UI. Accent must not be used for all interactive elements.

Semantic terminal fallback:

- Success: existing `✓`/`OK` plus prose.
- Warning: existing `!`/`WARNING` plus prose.
- Error/destructive: existing `✗`/`ERROR` plus remediation.
- Do not depend on ANSI color; color is optional enhancement only.

---

## Copywriting Contract

| Element                  | Copy                                                                                                                                                                                                                                                                |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Primary CTA              | Use ephemeral cloud runner                                                                                                                                                                                                                                          |
| Empty state heading      | No RunnerKit-managed runner is saved for `owner/name`.                                                                                                                                                                                                              |
| Empty state body         | Run `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner` for a one-job cloud runner, or use `--host user@host` for an existing machine.                                                                                                                |
| Error state              | Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows. Use `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner`, GitHub-hosted runners, or pass `--allow-public-repo-risk` only after reviewing the risk. |
| Destructive confirmation | Destroy ephemeral cloud runner: type `destroy owner/name` to remove the GitHub runner registration and RunnerKit-created Hetzner resources.                                                                                                                         |

### Required prompt and output copy

| Surface                            | Required copy                                                                                                                                                             |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Mode prompt                        | Choose runner mode for `owner/name`:                                                                                                                                      |
| Persistent option                  | Persistent trusted runner                                                                                                                                                 |
| Persistent option description      | Reuses one runner across trusted private jobs. Lowest ongoing friction, but unsafe for public, fork, or untrusted workflows.                                              |
| Ephemeral BYO option               | Ephemeral one-job runner on existing machine                                                                                                                              |
| Ephemeral BYO option description   | GitHub assigns one job then deregisters the runner. The machine is reused, so this is not a clean VM.                                                                     |
| Ephemeral cloud option             | Ephemeral one-job cloud runner (Hetzner)                                                                                                                                  |
| Ephemeral cloud option description | Stronger isolation for risky workloads. Creates billable resources until `runnerkit destroy` verifies cleanup.                                                            |
| Persistent default note            | Default mode: persistent for trusted private repositories.                                                                                                                |
| Ephemeral mode note                | Ephemeral mode: one GitHub job, automatic deregistration, TTL cleanup, and preserved troubleshooting logs.                                                                |
| Not fleet manager warning          | Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online.                             |
| Public/fork persistent block       | Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.                                                                       |
| Public/fork next action            | Use `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner` for stronger isolation, or use GitHub-hosted runners.                                               |
| Dangerous override copy            | Only pass `--allow-public-repo-risk` if you accept that untrusted code can execute repeatedly on your machine.                                                            |
| Ephemeral cloud cost copy          | Ephemeral cloud runners still create billable Hetzner resources. Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup.                         |
| Log preservation copy              | RunnerKit preserves best-effort runner `_diag` and systemd journal logs before cleanup. Configure external log forwarding for production-grade ephemeral troubleshooting. |
| TTL copy                           | TTL safeguard: RunnerKit finalizes the ephemeral runner after 24h if no job completes.                                                                                    |
| BYO caveat                         | BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine. Do not store unrelated secrets on the host.                                             |
| Completion heading                 | Ephemeral runner ready                                                                                                                                                    |
| Completion one-job line            | GitHub will assign at most one job to this runner, then automatically deregister it.                                                                                      |
| Completion cleanup cloud           | Cleanup after the job: `runnerkit destroy --repo owner/name`                                                                                                              |
| Completion cleanup BYO             | Cleanup after the job: `runnerkit down --repo owner/name`                                                                                                                 |
| Status completed                   | Ephemeral runner completed one job and needs cleanup.                                                                                                                     |
| Status TTL expired                 | Ephemeral runner TTL expired before a job completed. Run cleanup now.                                                                                                     |
| Safety guide heading               | Self-hosted Runner Safety Guide                                                                                                                                           |

Copywriting rules:

- CTA labels must be verb + noun, never `Submit`, `OK`, `Save`, or `Continue` alone.
- Every error must include a concrete next command, flag, env var, provider action, or docs path.
- Destructive copy must name both scopes when cloud resources exist: GitHub runner registration and RunnerKit-created Hetzner resources.
- Cost copy must always include the approximate/pricing-varies/user-responsible caveat before cloud provisioning and in docs.
- Ephemeral copy must say stronger isolation, not absolute safety.
- BYO ephemeral copy must explicitly say it is not a clean VM.
- Use `destroy` only for cloud billable resources; use `down` only for BYO cleanup.
- Do not imply RunnerKit edits workflow YAML or provides autoscaling/fleet management in Phase 5.

---

## CLI Output Contract

### Mode selection fields

Human and JSON output must include these facts before mode-dependent mutation:

- `repo`: `owner/name`
- `mode`: `persistent` or `ephemeral`
- `safety_profile`: one of `persistent-trusted`, `persistent-risky`, `ephemeral-byo`, `ephemeral-cloud`
- `tradeoffs.cost`
- `tradeoffs.isolation`
- `tradeoffs.cleanup`
- `tradeoffs.operations`
- `tradeoffs.logs`
- `recommended_for`
- `not_recommended_for`
- `warnings`
- `redactions_applied:true` in JSON output

### Required `runnerkit up` command examples

Docs and relevant command remediation must use these exact command examples:

```bash
runnerkit up --repo owner/name --mode persistent --host user@host
runnerkit up --repo owner/name --mode ephemeral --host user@host
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --yes
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --ephemeral-ttl 24h
```

### Required workflow snippets

Persistent snippet:

```yaml
runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]
```

Ephemeral snippet:

```yaml
runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, ephemeral]
```

Do not recommend `runs-on: self-hosted` alone.

### Ephemeral completion fields

Human and JSON completion output for ephemeral setup must include:

- runner name with an ephemeral suffix such as `runnerkit-owner-repo-ephemeral-<short-id>`
- `mode: ephemeral`
- `safety_profile`
- labels including `ephemeral`
- exact workflow snippet
- one-job copy: `GitHub will assign at most one job to this runner, then automatically deregister it.`
- TTL value, default `24h`
- log archive path such as `/var/lib/runnerkit/ephemeral/<runner>/logs`
- cleanup command: `runnerkit destroy --repo owner/name` for cloud or `runnerkit down --repo owner/name` for BYO
- warning that ephemeral mode is not a fleet manager
- `redactions_applied:true` in JSON output

### Ephemeral status states

Human and JSON `status` / `doctor` output must distinguish these mode-specific states:

| State                       | Human summary                                                         | Required next action                                                                               |
| --------------------------- | --------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `ephemeral_waiting`         | Ephemeral runner is online and waiting for its one job.               | Trigger a workflow using the printed `runs-on` labels or clean up if no longer needed.             |
| `ephemeral_busy`            | Ephemeral runner is running its one allowed job.                      | Wait for the job to finish, then inspect logs and clean up.                                        |
| `ephemeral_completed`       | Ephemeral runner completed one job and needs cleanup.                 | Run `runnerkit destroy --repo owner/name` for cloud or `runnerkit down --repo owner/name` for BYO. |
| `ephemeral_ttl_expired`     | Ephemeral runner TTL expired before a job completed. Run cleanup now. | Run cleanup and inspect preserved logs.                                                            |
| `ephemeral_cleanup_pending` | Ephemeral cleanup is incomplete and pending checkpoints remain.       | Re-run the cleanup command after fixing the blocker.                                               |

### Logs output

Human and JSON `logs` output for ephemeral runners must include:

- live `_diag` logs when the machine is reachable and runner files exist
- preserved log archive path when finalizer logs exist
- `Runner_*.log` and `Worker_*.log` sections when present
- bounded systemd journal section when present
- warning copy: `RunnerKit preserves best-effort logs only; configure external log forwarding for production-grade ephemeral troubleshooting.`
- no raw token/provider credential/machine secret leaks

### Cleanup output

Cloud ephemeral cleanup uses `runnerkit destroy`; BYO ephemeral cleanup uses `runnerkit down`.

Cloud human and JSON cleanup output must keep the Phase 4 billing contract:

- show a plan before mutation
- include GitHub runner, remote runner, provider server, provider SSH key, provider firewall, provider primary IP, and local state artifacts
- preserve/fetch logs before deleting cloud resources when SSH is reachable
- remove local state only after GitHub and provider verification pass
- keep pending checkpoints if cleanup is partial

BYO human and JSON cleanup output must:

- show a plan before mutation
- remove only RunnerKit-managed runner-specific artifacts
- preserve/fetch logs before deleting runner files when SSH is reachable
- keep pending checkpoints if cleanup is partial

---

## Documentation Contract

Create or update docs so users can understand mode tradeoffs before selection.

### Required docs files

- `README.md` must link to `docs/safety.md` and include the ephemeral cloud command.
- `docs/safety.md` must exist with heading `# Self-hosted Runner Safety Guide`.
- `docs/cloud-quickstart.md` must update Phase 4 wording so ephemeral mode is no longer described as deferred.
- `docs/byo-quickstart.md` must update the persistent safety warning to recommend ephemeral cloud for risky workloads.

### Required safety guide sections

`docs/safety.md` must include these exact section headings:

```markdown
# Self-hosted Runner Safety Guide

## Quick recommendation

## Persistent vs ephemeral tradeoffs

## When persistent is appropriate

## When ephemeral is recommended

## Public and fork-based workflow risk

## BYO ephemeral caveats

## Cloud ephemeral caveats

## Logs and troubleshooting

## Cleanup commands

## What RunnerKit does not do in v1
```

### Required docs command examples

Docs must include these exact command examples:

```bash
runnerkit up --repo owner/name --mode persistent --host user@host
runnerkit up --repo owner/name --mode ephemeral --host user@host
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --yes
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --ephemeral-ttl 24h
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --since 30m --lines 200
runnerkit doctor --repo owner/name
runnerkit down --repo owner/name --dry-run
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name --yes
```

### Required docs sentences

Docs must include these exact sentences where relevant:

- Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.
- Ephemeral mode gives stronger isolation by using one-job GitHub runner registration, but it is not a clean VM by itself.
- Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online.
- BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine.
- Ephemeral cloud runners still create billable Hetzner resources.
- Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup.
- RunnerKit preserves best-effort runner `_diag` and systemd journal logs before cleanup.
- Configure external log forwarding for production-grade ephemeral troubleshooting.
- RunnerKit prints labels/snippets and does not edit workflow YAML.
- Do not use `runs-on: self-hosted` alone for RunnerKit-managed runners.

---

## Registry Safety

| Registry | Blocks Used | Safety Gate                                                                               |
| -------- | ----------- | ----------------------------------------------------------------------------------------- |
| none     | none        | not applicable - manual Go CLI design system, no shadcn or third-party UI registry blocks |

No third-party component registry, shadcn block, npm UI package, terminal animation package, or browser-based component source is approved for Phase 5.

---

## Checker Sign-Off

- [x] Dimension 1 Copywriting: PASS
- [x] Dimension 2 Visuals: PASS
- [x] Dimension 3 Color: PASS
- [x] Dimension 4 Typography: PASS
- [x] Dimension 5 Spacing: PASS
- [x] Dimension 6 Registry Safety: PASS

**Approval:** approved 2026-05-01
