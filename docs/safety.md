# Self-hosted Runner Safety Guide

This guide explains when persistent self-hosted runners are safe to use, when ephemeral mode is recommended, and what RunnerKit v1 intentionally does not do. Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows. The lower-case validation phrase used throughout is: persistent self-hosted runners.

## Quick recommendation

- For trusted private repositories on a machine you already own:
  - `runnerkit up --repo owner/name --mode persistent --host user@host`
- For trusted private repositories where you want stronger isolation per job on an existing machine:
  - `runnerkit up --repo owner/name --mode ephemeral --host user@host`
- For public, fork-based, or otherwise untrusted workflows where you want stronger isolation:
  - `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner`

Use ephemeral cloud runner for risky workloads. Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.

## Persistent vs ephemeral tradeoffs

| Mode       | Cost                                                    | Isolation                                                                  | Cleanup                                                | Operations                                                            | Logs                                                              |
| ---------- | ------------------------------------------------------- | -------------------------------------------------------------------------- | ------------------------------------------------------ | --------------------------------------------------------------------- | ----------------------------------------------------------------- |
| persistent | Lowest ongoing cost; one runner reused across many jobs | Weakest; same machine handles every job, so untrusted code is unsafe       | `runnerkit down --repo owner/name`                     | Lowest friction; reuses one runner indefinitely                       | Live `_diag` and systemd journal logs while the runner is running |
| ephemeral  | Higher per-job cost (especially cloud)                  | Stronger; GitHub assigns one job and automatically deregisters the runner  | `runnerkit destroy` (cloud) or `runnerkit down` (BYO)  | One scoped runner only; not autoscaling and not a fleet manager       | Best-effort runner `_diag` and systemd journal preserved at TTL or after the job, before cleanup |

Ephemeral mode gives stronger isolation by using one-job GitHub runner registration, but it is not a clean VM by itself.

Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online.

## When persistent is appropriate

- The repository is private and trusted: workflow YAML, scripts, dependencies, and pull-request authors are all under your control.
- You want the lowest ongoing cost and one machine reused for many jobs.
- You accept that cleanup, log rotation, and host hygiene remain your responsibility.

Use this command to set up a persistent BYO runner:

```bash
runnerkit up --repo owner/name --mode persistent --host user@host
```

Do not use `runs-on: self-hosted` alone for RunnerKit-managed runners. Instead, use the printed `runs-on` snippet, for example `runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]`.

## When ephemeral is recommended

- Your repository is public, accepts pull requests from forks, or otherwise runs untrusted workflow code.
- You want each job to run against a fresh GitHub runner registration that GitHub deregisters after the one allowed job.
- You want a clean cleanup boundary even if the host is shared with other workloads.

Use these commands to set up an ephemeral runner:

```bash
runnerkit up --repo owner/name --mode ephemeral --host user@host
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --yes
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --ephemeral-ttl 24h
```

## Public and fork-based workflow risk

Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows. A persistent runner reuses the same machine for every job, so a malicious pull request can install backdoors, exfiltrate secrets, or compromise other jobs that run later on the same host.

For public, fork-based, or otherwise untrusted workflows, use:

```bash
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner
```

Or use GitHub-hosted runners.

If you fully understand the risk and still want a persistent runner for an untrusted workflow, RunnerKit blocks the setup unless you pass `--allow-public-repo-risk`. Only pass `--allow-public-repo-risk` if you accept that untrusted code can execute repeatedly on your machine.

## BYO ephemeral caveats

BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine. The host is reused, so any artifacts, packages, or secrets present on it remain after the runner deregisters.

- Do not store unrelated secrets on the host.
- Do not assume the machine is clean between ephemeral jobs.
- Use `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner` if you need stronger isolation than a one-job GitHub registration.

## Cloud ephemeral caveats

Ephemeral cloud runners still create billable Hetzner resources. Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup.

Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.

The TTL safeguard defaults to 24 hours so a runner that never receives a job is finalized and cleaned up rather than billing forever.

## Logs and troubleshooting

RunnerKit preserves best-effort runner `_diag` and systemd journal logs before cleanup.

Configure external log forwarding for production-grade ephemeral troubleshooting.

Useful read-only operations commands:

```bash
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --since 30m --lines 200
runnerkit doctor --repo owner/name
```

For ephemeral runners, RunnerKit also surfaces a preserved log archive path under `/var/lib/runnerkit/ephemeral/<runner>/logs` containing `Runner_*.log`, `Worker_*.log`, and a bounded `systemd-journal.log` excerpt.

## Cleanup commands

Use `runnerkit down --repo owner/name` for BYO cleanup; use `runnerkit destroy --repo owner/name` for cloud billable cleanup.

```bash
runnerkit down --repo owner/name --dry-run
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name --yes
```

`runnerkit destroy` verifies that GitHub runner registration is gone and that no RunnerKit-created Hetzner resources remain billable before removing local state. Cleanup keeps pending checkpoints if any step fails so you can re-run cleanup once the blocker is fixed.

## What RunnerKit does not do in v1

- No hosted control plane.
- No webhook listener or autoscaling fleet manager.
- No Actions Runner Controller, Kubernetes, runner scale sets, organization-level runner management, or JIT runner API.
- No automatic workflow YAML edits.
- No guarantee that BYO ephemeral mode is a clean VM.

RunnerKit prints labels/snippets and does not edit workflow YAML.
