# RunnerKit

RunnerKit is a CLI-first tool for solo developers who want reliable GitHub Actions self-hosted runners without manually copying registration commands, wiring services, or guessing which labels to use. The v1 path starts with GitHub Actions, local state, strict redaction, and a BYO Linux host flow.

## BYO persistent runner quickstart

Use the BYO persistent runner quickstart when you already have SSH access to a trusted Linux systemd machine:

```bash
runnerkit up --repo owner/name --host user@host
```

See [docs/byo-quickstart.md](docs/byo-quickstart.md) for prerequisites, safety notes, the workflow label snippet, and troubleshooting.

## Recommended cloud runner quickstart

Use the recommended cloud runner quickstart when you do not already have a Linux machine and want RunnerKit to provision the Phase 4 Hetzner path:

```bash
export HCLOUD_TOKEN=...
runnerkit up --repo owner/name --cloud hetzner
runnerkit up --repo owner/name --cloud hetzner --yes
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --since 30m --lines 200
runnerkit doctor --repo owner/name
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name
runnerkit destroy --repo owner/name --yes
```

Provision cloud runner

```yaml
runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]
```

See [docs/cloud-quickstart.md](docs/cloud-quickstart.md) for provider authentication, cost caveats, labels, status/logs/doctor, destroy verification, and live smoke-test guidance.

RunnerKit supports one recommended cloud path in Phase 4.
The default cloud runner is persistent and intended for trusted private repositories.
Ephemeral mode is deferred to Phase 5.
RunnerKit prints labels/snippets and does not edit workflow YAML.
Cost estimates are approximate and billing stops only after relevant provider resources are destroyed or verified non-billable.

## BYO operations

Start with read-only operations before manual SSH troubleshooting:

```bash
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --lines 50
runnerkit doctor --repo owner/name
runnerkit recover --repo owner/name --dry-run
runnerkit recover --repo owner/name --restart-service --yes
runnerkit down --repo owner/name --dry-run
runnerkit down --repo owner/name --yes
```
