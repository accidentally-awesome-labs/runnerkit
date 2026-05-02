# Recommended Cloud Runner Quickstart

Use this path when you do not already have a Linux machine and want RunnerKit to provision the recommended Hetzner cloud runner.

## Prerequisites

- A trusted private GitHub repository, or a public/fork repository where you choose ephemeral cloud mode for stronger isolation.
- GitHub authentication that can manage repository self-hosted runners.
- A Hetzner Cloud API token from the Hetzner Cloud Console.
- An SSH public key available via `--ssh-key <path>` plus `<path>.pub`, or a standard local public key such as `~/.ssh/id_ed25519.pub`.

```bash
export HCLOUD_TOKEN=...
```

`HETZNER_CLOUD_TOKEN` is also accepted as an alias. RunnerKit uses provider credentials from the environment and does not persist provider API tokens in local state, logs, diagnostics, or command output.

## Cost and billing caveat

Cost estimates are approximate and billing stops only after relevant provider resources are destroyed or verified non-billable.

Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.

Ephemeral cloud runners still create billable Hetzner resources.

Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup.

RunnerKit supports one recommended cloud path.
The default cloud runner is persistent and intended for trusted private repositories.
For stronger isolation on public, fork-based, or otherwise untrusted workflows, use ephemeral cloud mode (described below).
RunnerKit prints labels/snippets and does not edit workflow YAML.

## Persistent vs ephemeral mode

For trusted private repositories, the default persistent cloud runner has the lowest ongoing cost:

```bash
runnerkit up --repo owner/name --cloud hetzner
runnerkit up --repo owner/name --cloud hetzner --yes
```

Provision cloud runner

For public, fork-based, or otherwise untrusted workflows, use ephemeral cloud mode for stronger isolation:

```bash
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --yes
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --ephemeral-ttl 24h
```

Ephemeral cloud runners still create billable Hetzner resources. Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup. The TTL safeguard defaults to 24h so a runner that never receives a job is finalized and cleaned up automatically.

For full guidance see the [Self-hosted Runner Safety Guide](safety.md).

## Add the workflow labels

RunnerKit prints the exact labels to use. Add them to your workflow job yourself:

```yaml
runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]
```

For ephemeral mode, the snippet uses the `ephemeral` label instead:

```yaml
runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, ephemeral]
```

RunnerKit prints labels/snippets and does not edit workflow YAML.

## Check status and logs

Use read-only operations before manually SSHing into the runner:

```bash
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --since 30m --lines 200
runnerkit doctor --repo owner/name
```

For ephemeral runners, `runnerkit logs` also surfaces the preserved finalizer log archive after the one job runs (or after the TTL safeguard fires).

## Destroy and verify cleanup

Always review the destroy plan before applying cleanup:

```bash
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name
runnerkit destroy --repo owner/name --yes
```

RunnerKit removes local state only after GitHub runner registration and provider cleanup are verified. If cleanup is partial, rerun `runnerkit destroy --repo owner/name` after fixing the blocker; RunnerKit keeps pending checkpoints and provider resource IDs in state.

## Limitations

RunnerKit supports one recommended cloud path.
The default cloud runner is persistent and intended for trusted private repositories.
For stronger isolation on public, fork-based, or otherwise untrusted workflows, use `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner`.
RunnerKit prints labels/snippets and does not edit workflow YAML.
Cost estimates are approximate and billing stops only after relevant provider resources are destroyed or verified non-billable.

## Optional live smoke test

A live smoke test requires real Hetzner credentials and creates billable resources. Run it only in a repository and Hetzner project you control, then verify cleanup with:

```bash
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name --yes
```
