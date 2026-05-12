# RunnerKit v1.1.0 Release Notes

Date: 2026-05-12

## Highlights (SEED-004 UX polish — tier 1)

- **First-run wizard:** With no saved repositories, `runnerkit` (no subcommand) starts a short BYO vs cloud wizard; otherwise shows help. **`runnerkit --json`** returns structured `next_actions` without a TTY.
- **Stage + JSON contract:** `runnerkit status --json` and `doctor --json` include **`schema_version`**, **`stage`**, and versioned **`next_actions`** where applicable. Human **`doctor`** output shows **`STAGE:`** (`running`, `error`, etc.).
- **Boxed commands:** `internal/ui/box.go` renders copy-paste panels (ASCII by default; **`--unicode`** for UTF-8 borders).
- **BYO progress:** Checklists during `runnerkit up` / `register` with session files under **`{state dir}/sessions/`** (stable key `byo-<repo>__<host>`).
- **Global flags:** **`--explain`** (WHY/RUNS/TAKES on `init` and BYO `up`), **`--unicode`** (box borders).
- **`doctor --fix` / `--ignore`:** Persist ignored finding IDs in **`config.json`** next to `state.json`. **`runner_version_stale`** can run **`upgrade-runner`** (confirm unless **`doctor --fix --yes`**). **`--fix`** cannot be combined with **`--json`**.
- **Smoke:** [`scripts/smoke/assert-doctor-json-contract.sh`](scripts/smoke/assert-doctor-json-contract.sh) asserts **`schema_version`** and **`stage`** on doctor JSON.

## Docs

- [`docs/troubleshooting/doctor-ux.md`](docs/troubleshooting/doctor-ux.md) — wizard, explain, checklists, doctor fix/ignore.
- Tier 2 backlog: [`.planning/phases/08-ux-polish-seed-004/08-02-TIER2-PLAN.md`](.planning/phases/08-ux-polish-seed-004/08-02-TIER2-PLAN.md).

## Stopwatch / Live Smoke (maintainer, D-13)

Recorded same day as development verification; see **`RELEASE-NOTES-v1.0.0.md`** (Maintainer smoke re-run table) and **`06-VERIFICATION.md`** for BYO **149s** / Hetzner **287s**.

## Upgrade Path

From [docs/upgrade.md](docs/upgrade.md): CLI via **`runnerkit upgrade`**; bundled runner pin via **`runnerkit upgrade-runner`**.
