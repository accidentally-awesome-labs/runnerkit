# Recommended Cloud Runner Quickstart

Use this path when you do not already have a Linux machine and want RunnerKit to provision the recommended Phase 4 cloud runner on Hetzner.

## Prerequisites

- A trusted private GitHub repository.
- GitHub authentication that can manage repository self-hosted runners.
- A Hetzner Cloud API token from the Hetzner Cloud Console.
- An SSH public key available via `--ssh-key <path>` plus `<path>.pub`, or a standard local public key such as `~/.ssh/id_ed25519.pub`.

```bash
export HCLOUD_TOKEN=...
```

`HETZNER_CLOUD_TOKEN` is also accepted as an alias. RunnerKit uses provider credentials from the environment and does not persist provider API tokens in local state, logs, diagnostics, or command output.

## Cost and billing caveat

Cost estimates are approximate and billing stops only after relevant provider resources are destroyed or verified non-billable.

RunnerKit supports one recommended cloud path in Phase 4.
The default cloud runner is persistent and intended for trusted private repositories.
Ephemeral mode is deferred to Phase 5.
RunnerKit prints labels/snippets and does not edit workflow YAML.

## Run setup

Preview the plan first if you want to review cost, labels, ownership tags, and future cleanup commands:

```bash
runnerkit up --repo owner/name --cloud hetzner
runnerkit up --repo owner/name --cloud hetzner --yes
```

Provision cloud runner

## Add the workflow labels

RunnerKit prints the exact labels to use. Add them to your workflow job yourself:

```yaml
runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]
```

RunnerKit prints labels/snippets and does not edit workflow YAML.

## Check status and logs

Use read-only operations before manually SSHing into the runner:

```bash
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --since 30m --lines 200
runnerkit doctor --repo owner/name
```

## Destroy and verify cleanup

Always review the destroy plan before applying cleanup:

```bash
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name
runnerkit destroy --repo owner/name --yes
```

RunnerKit removes local state only after GitHub runner registration and provider cleanup are verified. If cleanup is partial, rerun `runnerkit destroy --repo owner/name` after fixing the blocker; RunnerKit keeps pending checkpoints and provider resource IDs in state.

## Limitations

RunnerKit supports one recommended cloud path in Phase 4.
The default cloud runner is persistent and intended for trusted private repositories.
Ephemeral mode is deferred to Phase 5.
RunnerKit prints labels/snippets and does not edit workflow YAML.
Cost estimates are approximate and billing stops only after relevant provider resources are destroyed or verified non-billable.

## Optional live smoke test

A live smoke test requires real Hetzner credentials and creates billable resources. Run it only in a repository and Hetzner project you control, then verify cleanup with:

```bash
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name --yes
```
