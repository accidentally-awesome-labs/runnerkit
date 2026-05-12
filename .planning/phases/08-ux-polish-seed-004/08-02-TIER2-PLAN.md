# Phase 08-02 — SEED-004 UX polish, tier 2 (follow-up)

**Prerequisite:** [08-01-PLAN.md](08-01-PLAN.md) (tier 1) shipped — box renderer, stage on `status`/`doctor`, BYO checklist persistence, first-run wizard, `--explain`/`--unicode` on tier-1 paths, `doctor --fix`/`--ignore` with one fixable id.

**Source:** [.planning/seeds/SEED-004-ux-polish-layer.md](../../seeds/SEED-004-ux-polish-layer.md) and the “Gaps / tier 2” notes from the UX polish plan.

## Goal

Close the remaining contract and UX gaps so **every read/write command** exposes a consistent **`(stage, next_actions)`** story in `--json`, **`--explain`** covers the full verb set where it adds value, remediation copy uses **boxed commands** where appropriate, and **`doctor --fix`** grows a **small, audited** fix registry—without duplicating probe logic between doctor and stage.

## Non-goals (still)

- Full-screen TUI frameworks.
- i18n / translation pipeline.
- Optional `RUNNERKIT_PLAIN` unless a concrete a11y requirement lands.

---

## Workstream A — Uniform JSON contract (`logs`, `down`, `destroy`, `recover`)

**Problem:** Tier 1 added `schema_version` + `stage` + versioned `next_actions` on `status` and `schema_version` + `stage` on `doctor`. Agents still see heterogeneous JSON across other read/write commands.

**Deliverables:**

1. For each command, on **`--json` success paths**, emit:
   - `schema_version` (reuse [`internal/ux/nextaction`](../../internal/ux/nextaction))
   - `stage` where inferrable (often `InferFromObserved` on whatever `ObservedRunner` the command already builds; may be `unknown` when no repo state).
   - `next_actions` as a **JSON array** (never `null`) — use `MergePayload` or `ApplySchemaAndStage` + explicit empty slice pattern consistently with doctor/smoke expectations.
2. Add or extend a **single smoke/assert script** (or Go test harness) that spot-checks these commands with `--json` for required keys, reusing the “arrays not null” rule from `assert-doctor-json-contract.sh`.

**Touchpoints (indicative):** [`internal/cli/logs.go`](../../internal/cli/logs.go), [`down.go`](../../internal/cli/down.go), [`destroy.go`](../../internal/cli/destroy.go), [`recover.go`](../../internal/cli/recover.go), shared helper in `cli` or `ux/nextaction` to avoid copy-paste.

---

## Workstream B — Shared “observe” layer (stage vs doctor drift)

**Problem:** `stage.InferFromDoctor` uses deep checks; `InferFromObserved` uses status-shaped facts. Long-term drift risk if both evolve separately.

**Deliverables:**

1. Introduce a thin **`internal/ux/observe`** (or extend `internal/ops`) package that returns a **single struct** consumed by:
   - `stage` package (canonical stage string), and/or
   - doctor report builder inputs (so `STAGE` and findings derive from the same observation snapshot where possible).
2. Document in code comments **which fields are authoritative** for stage vs findings (e.g. install path only from doctor deep path).

**Verification:** Table tests proving identical inputs → identical stage classification before/after refactor; no behavior change in doctor findings ordering.

---

## Workstream C — Box rollout + JSON `boxed_command`

**Problem:** Tier 1 added `RenderBoxed` but did not systematically replace remediation strings or emit structured JSON for agents.

**Deliverables:**

1. Replace high-traffic human strings (“run this on the host…”) with `RenderBoxed` where the seed intended (install one-liner, recover snippets, etc.).
2. On relevant **`--json`** payloads, add optional **`boxed_command`** object `{host, command, why}` (additive schema; document in `nextaction` package comment or small `docs/` note).
3. Golden tests for **narrow width** + `--unicode` + `--no-color` combinations on representative strings.

---

## Workstream D — Checklist hardening

**Problem:** Tier 1 persists BYO `up`/`register` checklist files; idempotency, TTL, concurrency, and schema versioning are only partially specified.

**Deliverables:**

1. **`schema_version`** field inside session JSON; migration path if bumped.
2. **TTL or GC** for abandoned `sessions/*.json` (e.g. older than N days, configurable env).
3. **Concurrency:** document last-writer-wins vs `flock`/rename atomicity; pick one and test two parallel `up` runs.
4. Optional **`progress` / `checklist` array** on `up --json` completion payloads (additive) if agents need parity with human checklist.

---

## Workstream E — Wizard depth (seed alignment)

**Problem:** Tier 1 wizard is BYO-first + cloud pointer; seed also mentioned SSH `BatchMode` probe, `gh repo list` when available, and driving the real checklist through install.

**Deliverables (pick scope explicitly):**

- **E1 (small):** After host/repo entry, run **non-mutating** `ssh -o BatchMode=yes` probe; on failure, print boxed `ssh-copy-id` / docs pointer and exit non-zero with JSON `next_actions`.
- **E2 (medium):** If `gh` is on `PATH`, optional repo picker; else keep paste `owner/name` + `git remote` default suggestion.
- **E3 (large, optional):** Shallow cloud path that collects env hints then prints exact `runnerkit up --cloud hetzner …` (no API calls in wizard).

**Non-interactive:** Extend tests so JSON wizard path never requires stdin (already partially true); add assertion for “no TTY + not json” error path if not already covered.

---

## Workstream F — `--explain` everywhere (tier 2)

**Deliverables:**

1. Add **WHY / RUNS / TAKES** string constants **next to** the implementation for: `logs`, `down`, `destroy`, `recover`, `state`, `upgrade`, `upgrade-runner`, `version` (only where steps exist), and **cloud `up`** branches not covered in tier 1.
2. Keep stderr vs stdout rule consistent with tier 1 (`explain` to `deps.Err` in human mode; skip or embed summary policy for `--json` — document choice).

---

## Workstream G — `doctor --fix` registry expansion

**Deliverables:**

1. Refactor tier-1 special-case into a **`map[findingID]fixFn`** registry in `internal/cli` (or `internal/ops` with CLI wiring), each entry documented with **safety notes** and **dry-run** behavior where applicable.
2. Add **one** additional fixable finding only if it is **idempotent and low blast radius** (e.g. label drift → guided `recover --reregister` dry-run first — **only** if product agrees).
3. **Post-fix summary** block (human + optional JSON `fixes_applied: []`) for logs/CI.
4. Docs: safe wording for automation (`--fix` with `--yes`) in [`docs/troubleshooting/doctor-ux.md`](../../docs/troubleshooting/doctor-ux.md) without tripping docs tests that forbid contiguous `doctor --fix` in combined README/quickstart scans.

**Explicit exclusions until designed:** SSH unreachable, host-key mismatch, anything requiring secrets or destructive GH deletes without dry-run.

---

## Workstream H — Root help + ops hygiene

**Deliverables:**

1. Root **`Long` / `--help`** text states: bare `runnerkit` = first-run wizard **only** when no saved repos; otherwise help (matches gap decision).
2. **`doctor --fix` + CI:** one paragraph in troubleshooting on never running `--fix --yes` unattended unless… (conditions).

---

## Suggested order

1. **A** (uniform JSON) — unblocks agents and smoke parity.
2. **B** (observe layer) — reduces long-term maintenance cost before adding more stage consumers.
3. **F** (`--explain` sweep) — parallelizable once A stabilizes payload helpers.
4. **C** (box + `boxed_command`) — mostly copy + golden tests.
5. **D** (checklist hardening) — before expanding wizard reliance on sessions.
6. **E** (wizard depth) — product-sized slices E1 → E2 → E3.
7. **G** + **H** — expand fix registry only after registry refactor and docs guardrails.

## Verification checklist (tier 2 exit)

- [ ] `go test ./...` green; new JSON contract tests or smoke script green.
- [ ] Manual: `runnerkit <each tier-2 command> --json` returns arrays for `next_actions` / `host_incident_hints` where applicable (match doctor rules).
- [ ] `make smoke-live` (or maintainer equivalent) after smoke script updates.
- [ ] No new forbidden substrings in docs test surfaces unless `docs_test` is intentionally updated with safe phrasing.

## Estimate

**Medium–large** split across 1–2 milestones: **A+F** is roughly one focused PR; **B+C+D+E+G** is a second wave depending on E scope.
