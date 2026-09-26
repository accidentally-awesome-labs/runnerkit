# Doctor and CLI UX helpers

## First-run wizard

With **no saved repositories**, running `runnerkit` with no subcommand starts a short wizard (TTY required for interactive mode). If you already have saved runners, the same command shows standard help.

Use `runnerkit --json` with no subcommand for machine-readable `next_actions` when automation cannot use the wizard.

## Explain mode

Pass **`--explain`** on any subcommand (global flag) to print short **WHY / RUNS / TAKES** blocks before major steps where implemented (`init`, BYO `up` / `register` path).

## Progress checklists

During BYO **`up`** or **`register`**, RunnerKit writes resumable progress under **`sessions/`** in the RunnerKit state directory (next to `state.json`) and prints a checklist after preflight. Versions v1.1.0 through v1.3.3 wrote `sessions/` into the current working directory instead; delete any stray `sessions/` directory there (it contains `user@host`).

## Doctor remediation

The `doctor` command can **persistently ignore** a finding id:

```bash
runnerkit doctor --repo owner/name --ignore runner_version_stale
```

Ignored ids are stored in **`config.json`** in the same directory as `state.json` (v1.3.3 and earlier wrote it into the current working directory).

`doctor --fix` is disabled in v1.3.4 (`command_disabled`, exit 2). Its only
fix ran `upgrade-runner`, which in v1.3.3 and earlier deleted the runner's
credentials and unregistered it. `doctor` findings now print the manual
steps instead; for `runner_version_stale`, usually no action is needed
because the GitHub runner updates itself. See [upgrade.md](../upgrade.md).
