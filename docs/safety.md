# Self-hosted Runner Safety Guide

This guide explains when a RunnerKit runner is appropriate, what the modes do
and do not protect against, and what RunnerKit intentionally does not do.
Persistent self-hosted runners are unsafe for public, fork-based, or
otherwise untrusted workflows. The lower-case validation phrase used
throughout is: persistent self-hosted runners.

**Correction (v1.3.4).** Earlier versions of this guide said ephemeral mode
gives "stronger isolation per job" and sent public or untrusted workflows to
ephemeral Hetzner cloud runners. Both were wrong: BYO ephemeral mode reuses
the same host, and ephemeral cloud VMs were never destroyed after their job
and kept billing. Ephemeral cloud is disabled in v1.3.4.

## Quick recommendation

- **Public, fork-based, or otherwise untrusted workflows:** use
  GitHub-hosted runners. They are free and unlimited for public
  repositories. Do not use RunnerKit for them.
- **Trusted private repository on a machine you already own:**
  - `runnerkit up --repo owner/name --mode persistent --host user@host`

Everything RunnerKit installs gives workflow code a path to root on the host
(root-equivalent sudoers, `docker` group membership). See
[security-posture.md](security-posture.md).

## Persistent vs ephemeral tradeoffs

| Mode       | Status in v1.3.4 | Isolation | Cleanup | Operations | Logs |
| ---------- | ---------------- | --------- | ------- | ---------- | ---- |
| persistent | Supported (BYO); experimental on cloud | None between jobs: the same machine and user run every job | `runnerkit down --repo owner/name` (BYO), `runnerkit destroy --repo owner/name` (cloud) | One runner reused indefinitely | Live `_diag` and systemd journal logs while the runner is running |
| ephemeral  | BYO only, requires `--experimental`; known defects (finalizer runs unprivileged, TTL ignored), untested; disabled on cloud | None between jobs: GitHub sends one job per registration, but the host, user and files are reused | `runnerkit down --repo owner/name` | One scoped runner only; not autoscaling and not a fleet manager | Best-effort runner `_diag` and systemd journal preserved after the job or at the TTL, before cleanup |

Ephemeral mode is a one-job GitHub runner registration. It is not isolation
and not a clean VM.

Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online.

## When persistent is appropriate

- The repository is private and trusted: workflow YAML, scripts,
  dependencies, and pull-request authors are all under your control.
- You would give those contributors root on the host.
- You accept that cleanup, log rotation, and host hygiene remain your
  responsibility.

```bash
runnerkit up --repo owner/name --mode persistent --host user@host
```

Do not use `runs-on: self-hosted` alone for RunnerKit-managed runners. Instead, use the printed `runs-on` snippet, for example `runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]`.

## Public and fork-based workflow risk

Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows. A persistent runner reuses the same machine for every job, so a malicious pull request can install backdoors, exfiltrate secrets, or compromise other jobs that run later on the same host.

For public, fork-based, or otherwise untrusted workflows, use GitHub-hosted
runners. GitHub advises that self-hosted runners should almost never be used
for public repositories.

RunnerKit blocks a persistent runner for a public repository unless you pass
`--allow-public-repo-risk`. Only pass it if you accept that untrusted code can
execute repeatedly on your machine, with a path to root.

> If you see `RKD-AUTH-001` or `RKD-AUTH-003` in CLI output, the
> [auth troubleshooting page](troubleshooting/auth.md) explains the gate.

## BYO ephemeral caveats

BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine. The host is reused, so any artifacts, packages, processes or secrets present on it remain after the runner deregisters.

- It requires `--experimental`:
  `runnerkit up --repo owner/name --mode ephemeral --experimental --host user@host`.
- It has known defects (the finalizer runs unprivileged and the TTL is
  ignored) and is untested in this release.
- Do not store unrelated secrets on the host.
- Do not assume the machine is clean between ephemeral jobs.
- If you need isolation, use GitHub-hosted runners.

## Cloud ephemeral caveats

Ephemeral cloud runners are disabled in v1.3.4
(`ephemeral_cloud_disabled`, exit 2, even with `--experimental`). The VM was
never destroyed after its job: the TTL safeguard only stopped the runner
service, and the server kept billing until `runnerkit destroy`.

If you created ephemeral cloud runners with an older version, run
`runnerkit destroy --repo owner/name` for each one. Billing stops only after
`runnerkit destroy --repo owner/name` verifies cleanup.

## Logs and troubleshooting

RunnerKit preserves best-effort runner `_diag` and systemd journal logs before cleanup.

Configure external log forwarding if you need complete job logs.

Useful read-only operations commands:

```bash
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --since 30m --lines 200
runnerkit doctor --repo owner/name
runnerkit doctor --repo owner/name --deep
```

For ephemeral runners, RunnerKit also surfaces a preserved log archive path under `/var/lib/runnerkit/ephemeral/<runner>/logs` containing `Runner_*.log`, `Worker_*.log`, and a bounded `systemd-journal.log` excerpt.

Heavy workflows can **OOM** small VMs; preflight warns on low **MemAvailable** / missing swap, and `runnerkit doctor --deep` can flag likely kernel or linker kills from bounded journals (**RKD-BOOT-016..018**). See [Host resources and OOM](troubleshooting/host-resources.md).

## Cleanup commands

Use `runnerkit down --repo owner/name` for BYO cleanup; use `runnerkit destroy --repo owner/name` for cloud billable cleanup. `down` refuses RunnerKit-managed cloud state (`wrong_cleanup_command`).

```bash
runnerkit down --repo owner/name --dry-run
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name --yes
```

`runnerkit destroy` verifies that GitHub runner registration is gone and that no RunnerKit-created Hetzner resources remain billable before removing local state. Cleanup keeps pending checkpoints if any step fails so you can re-run cleanup once the blocker is fixed.

To also remove the privileges RunnerKit granted on a host, follow
[If you already installed RunnerKit](security-posture.md#if-you-already-installed-runnerkit).

## What RunnerKit does not do

- No hosted control plane.
- No webhook listener or autoscaling fleet manager.
- No Actions Runner Controller, Kubernetes, runner scale sets, organization-level runner management, or JIT runner API.
- No automatic workflow YAML edits.
- No isolation between jobs, in any mode.

RunnerKit prints labels/snippets and does not edit workflow YAML.
