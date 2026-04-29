# RunnerKit

RunnerKit is a CLI-first tool for solo developers who want reliable GitHub Actions self-hosted runners without manually copying registration commands, wiring services, or guessing which labels to use. The v1 path starts with GitHub Actions, local state, strict redaction, and a BYO Linux host flow.

## BYO persistent runner quickstart

Use the BYO persistent runner quickstart when you already have SSH access to a trusted Linux systemd machine:

```bash
runnerkit up --repo owner/name --host user@host
```

See [docs/byo-quickstart.md](docs/byo-quickstart.md) for prerequisites, safety notes, the workflow label snippet, and troubleshooting.

## BYO operations

Start with read-only operations before manual SSH troubleshooting:

```bash
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --lines 50
runnerkit doctor --repo owner/name
```
