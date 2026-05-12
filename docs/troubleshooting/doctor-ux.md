# Doctor and CLI UX helpers

## First-run wizard

With **no saved repositories**, running `runnerkit` with no subcommand starts a short wizard (TTY required for interactive mode). If you already have saved runners, the same command shows standard help.

Use `runnerkit --json` with no subcommand for machine-readable `next_actions` when automation cannot use the wizard.

## Explain mode

Pass **`--explain`** on any subcommand (global flag) to print short **WHY / RUNS / TAKES** blocks before major steps where implemented (`init`, BYO `up` / `register` path).

## Progress checklists

During BYO **`up`** or **`register`**, RunnerKit writes resumable progress under **`sessions/`** next to `state.json` and prints a checklist after preflight.

## Doctor remediation

The `doctor` command can **persistently ignore** a finding id:

```bash
runnerkit doctor --repo owner/name --ignore runner_version_stale
```

Ignored ids are stored in **`config.json`** in the same directory as `state.json`.

For supported findings, you can apply automated fixes interactively:

```bash
runnerkit doctor --repo owner/name --fix
```

Use **`--yes`** with **`--fix`** only in trusted automation (skips confirmation). Re-run `doctor` after fixes to confirm health.

Currently **`runner_version_stale`** maps to running **`upgrade-runner`** for the same repository.
