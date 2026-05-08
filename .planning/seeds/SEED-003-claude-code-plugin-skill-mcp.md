---
id: SEED-003
status: dormant
planted: 2026-05-08
planted_during: v1.0.0 / Phase 06 (release-upgrade-docs-and-v1-validation, attempt-19 smoke-red)
trigger_when: starting v1.3 milestone OR any milestone scoped at "agent integration" / "Claude Code plugin" / "MCP server" / "skill" / "AI-driven onboarding" / depends on SEED-001 + SEED-002 having landed
scope: medium
---

# SEED-003: RunnerKit Claude Code plugin (skill + slash commands + hook), with optional MCP later

## Why This Matters

Once SEED-001 (bootstrap/lifecycle split) lands and CLI commands emit a structured `next_actions` JSON contract, RunnerKit becomes trivial to drive from any AI agent. The CLI is already the source of truth — the agent's job reduces to: invoke CLI with `--json`, parse `next_actions[]`, render the boxed install command for the user, wait for confirmation, call the next action.

For Claude Code specifically, **a plugin is the right wrapper** — it bundles a skill (workflow knowledge + activation triggers), slash commands (power-user entry points), and a hook (proactive discovery in repos with `.github/workflows/`). Zero install for the agent runtime; one-command install for the user (`/plugin marketplace install runnerkit`).

**An MCP server is NOT the right primary mechanism** for this CLI-first product:
- runnerkit CLI is already on the user's machine; an MCP server adds install ceremony without adding capability.
- All state lives on local disk + remote host; no server-side session needs to be held.
- Token cost: MCP tool schemas eat agent context at session start; a skill loads lazily on activation.
- Solo-dev model (one user, one machine) doesn't benefit from MCP's centralized-server strengths (multi-user, cross-platform reach, persistent streams).

MCP becomes the right answer **later** (v1.4+) if any of these arrive:
- Streaming watch operations (`runnerkit watch --tail` for live job events).
- Cross-platform reach beyond Claude (Cursor, Cline, Continue users).
- Hosted-runner SaaS pivot (multi-user auth, billing).

The plugin design is forward-compatible — the same `next_actions` contract a skill consumes is what an MCP tool would return.

## When to Surface

**Trigger:** v1.3 milestone scope OR any of:
- "Claude Code plugin"
- "agent integration"
- "MCP server" (note: read SEED-003 first to understand why MCP is deferred to v1.4+)
- "AI-driven onboarding"
- "skill" / "hook" / "slash command" for runnerkit
- "set up a runner from chat"

**Hard prerequisites:**
- SEED-001 (bootstrap/lifecycle split) — required: the plugin's whole UX assumes the curl-pipe-sudo install + zero-TTY lifecycle.
- SEED-002 (multi-repo per host) — required: `register / unregister / list` are the natural agent verbs; without them the plugin reduces to a thin wrapper around the v1.0.0 single-repo CLI.

## Scope Estimate

**Medium** — one phase, possibly two if the streaming-watch MCP companion ships at the same time. Decomposes into:

- **Phase A — `next_actions` JSON contract finalized.** Schema lives in `internal/ux/nextaction/` (created in SEED-001). Every CLI subcommand emits it under `--json`. Versioned (`schema_version: 1`), additive-only.
- **Phase B — Claude Code plugin skeleton.** Layout:
  ```
  runnerkit-plugin/
  ├── plugin.json
  ├── skills/runnerkit.md                ← workflow + triggers + glossary
  ├── commands/
  │   ├── runnerkit-onboard.md           ← /runnerkit-onboard (first-run wizard)
  │   ├── runnerkit-add.md               ← /runnerkit-add owner/repo
  │   ├── runnerkit-status.md            ← /runnerkit-status
  │   └── runnerkit-doctor.md            ← /runnerkit-doctor [--fix]
  ├── hooks/detect-workflows.json         ← UserPromptSubmit: cwd has .github/workflows/ with self-hosted ⇒ surface /runnerkit-onboard once
  └── agents/runnerkit-debugger.md        ← subagent for orchestrating doctor --fix flows
  ```
- **Phase C — distribution.** Publish to a Claude Code plugin marketplace (or git URL). Versioned, signed alongside the runnerkit binary releases.
- **Phase D — docs + validation.** Walkthrough doc: "I'm a developer chatting with Claude. I say 'I have a workflow that needs a self-hosted runner.' What happens?" End-to-end: hook fires, skill activates, `/runnerkit-onboard` runs, agent shows curl-pipe-sudo box, user confirms, agent calls `runnerkit register`, runner online. Validate manually + add a CI smoke that exercises the slash commands against a fixture repo.
- **Phase E (deferred to v1.4+) — optional MCP server.** Same `next_actions` contract, exposed as MCP tools. Bundle into the plugin so `/plugin marketplace install runnerkit` gives both. Triggered by demand: streaming-watch use case OR non-Claude IDE asks.

Probably 1 phase for the plugin itself; Phase E is a separate later milestone.

## Breadcrumbs

Existing plugin/skill/MCP machinery in this repo's local Claude Code ecosystem (referenced by user during planting):

- `~/.claude/plugins/` — plugin examples (vercel-plugin, agent-sdk-dev, hookify, etc.) — pattern to follow
- `~/.claude/skills/` — many skills exist; `runnerkit.md` would join them
- `.claude/skills/` (per-project) — also valid location for a project-bundled skill (less likely for runnerkit since the user typically runs it from any repo)

Code that becomes the JSON contract source:
- `internal/cli/*.go` — every subcommand's `--json` emit path (after SEED-001 lands)
- `internal/ux/nextaction/` — proposed new package, `NextAction` struct + serialization
- `internal/ui/renderer.go` — currently emits structured errors; extend with `next_actions` field on success paths too

Related decisions:
- D-04 (live BYO smoke) — the smoke harness becomes one of the consumers of the JSON contract; `make smoke-live` parses `next_actions` to drive its assertions.
- DOC-04 (cleanup/troubleshooting docs) — the `runnerkit doctor --fix` slash command consumes findings already structured by Plan 06-03's RKD-XXX-NNN error code system.

## Notes

The plugin model is THE delivery vehicle for "RunnerKit as a chat-driven product". Two canonical UX moments to design for:

1. **Cold-start chat:** "I have a private repo and want a self-hosted runner." Hook fires, skill activates, `/runnerkit-onboard` runs the wizard, user types `salar@my-desktop` and `me/proj-a`, gets the boxed curl-pipe-sudo command, pastes once on host, confirms in chat, runner online — all from chat.
2. **Mid-cycle add-a-repo:** "Also add me/proj-b to the same host." Skill recognizes the host already exists in `runnerkit list`, calls `runnerkit register --host salar@my-desktop --repo me/proj-b`, runner online in ~10s. Zero new TTY interaction.

The skill's job is to encode these patterns so the agent never improvises them.

For MCP deferral: the cleanest signal that MCP becomes valuable is when **someone asks for it** — Cursor user file an issue, or a streaming-watch feature gets prioritized. Don't pre-build the MCP server; ship the plugin first, watch for demand, add MCP within the plugin bundle when triggered.

Cross-refs:
- SEED-001 (bootstrap/lifecycle split) — hard prerequisite (gives us the JSON contract)
- SEED-002 (multi-repo per host) — hard prerequisite (gives us the agent verbs)
- Future: SEED-N (MCP server for streaming watch / cross-platform reach), SEED-M (hosted runner SaaS pivot)
