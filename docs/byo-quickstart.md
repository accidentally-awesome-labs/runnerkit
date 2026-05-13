# BYO Persistent Runner Quickstart

This guide covers the Phase 2 happy path: connect RunnerKit to an existing trusted Linux systemd host over SSH, install a repository-scoped persistent GitHub Actions runner, and copy the labels into your workflow job.

## Prerequisites

- A RunnerKit binary or local checkout you can run.
- GitHub authentication through `gh auth login` or `RUNNERKIT_GITHUB_TOKEN` with repository Administration read/write and Metadata read for the target repo.
- A trusted private GitHub repository such as `owner/name`.
- SSH access to a Linux systemd host as `user@host`.
- Sudo ability on that host for package install, creating the runner user, and installing the systemd service.

## Safety warning

Persistent self-hosted runners are intended for trusted private repositories; public, fork-based, or otherwise untrusted workflows can execute code on your machine.

Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.

Do not use this persistent BYO path for public pull requests or untrusted workflow code. Use runnerkit up --repo owner/name --mode ephemeral --cloud hetzner for stronger isolation, or use GitHub-hosted runners.

For full guidance see the [Self-hosted Runner Safety Guide](safety.md).

## Sudo setup (one-time on the host)

Bootstrap runs over SSH without a TTY, so the SSH user must already have **scoped passwordless sudo** for RunnerKit’s commands. Run the **one-time install on the runner machine** (interactive sudo once), then use `runnerkit register` or `runnerkit up` from your workstation forever after.

1. Print the install command from your workstation:

   ```bash
   runnerkit init --print-install-command
   ```

2. SSH to the Linux host and paste the `curl … install.sh | sudo bash` line (or download `install.sh` from the matching [GitHub release](https://github.com/accidentally-awesome-labs/runnerkit/releases) and verify checksums per `README.md`).

This installs `/etc/sudoers.d/runnerkit-installer` with NOPASSWD only for the bootstrap command set (`apt-get`/`dnf`/`yum`, `useradd`, `install`, `curl`, `sha256sum`, `chown`, `rm`, `su`, `tar`, `systemctl`, runner `svc.sh`). See [RKD-BOOT-015](troubleshooting/bootstrap.md#rkd-boot-015) if anything blocks.

To revert on the host:

```bash
sudo rm -f /etc/sudoers.d/runnerkit-installer
```

### Decision tree

| Scenario | Path |
| --- | --- |
| First-time BYO host | `runnerkit init --print-install-command` → run on host → `runnerkit up` or `runnerkit register` |
| CI / automation / `--json` | Host must already have install applied; otherwise RunnerKit exits with `host_install_required` and a boxed install command |
| Manual NOPASSWD ALL for the SSH user | Works but prefer scoped install.sh |

## Run setup

```bash
runnerkit up --repo owner/name --host user@host
```

Useful automation flags:

```bash
runnerkit up --repo owner/name --host user@host --yes
runnerkit up --repo owner/name --host user@host:2222 --ssh-key ~/.ssh/id_ed25519 --yes
```

If your CI workflows need additional OS packages (native libraries, GUI test dependencies, etc.), add them with `--extra-packages`:

```bash
runnerkit up --repo owner/name --host user@host \
  --extra-packages "libsecret-1-dev,dbus-x11,gnome-keyring"
```

These are installed alongside RunnerKit's required tools during the `fix_dependencies` bootstrap step and saved in state so `runnerkit upgrade-runner` re-installs them.

RunnerKit prompts for unknown SSH host keys. Verify the `SHA256:` fingerprint before accepting it.

## What RunnerKit does

- Resolves and checks GitHub repository permissions.
- Blocks risky public/fork repository defaults unless you explicitly override the safety gate.
- Verifies the SSH host key and records the accepted fingerprint in local state.
- Runs SSH preflight checks for Linux, architecture, systemd, sudo, disk, **MemAvailable / swap** (warnings for low RAM or no-swap small hosts — [RKD-BOOT-016/017](troubleshooting/bootstrap.md#rkd-boot-016)), tools, time, network, and runner conflicts.
- Creates or reuses the non-root `runnerkit-runner` service user.
- Downloads the official GitHub Actions runner package, verifies its SHA-256 checksum, and configures it with a short-lived registration token.
- Installs and starts the runner service through systemd.
- Verifies the GitHub runner is online with RunnerKit labels before saving successful state.

RunnerKit does not edit or commit workflow YAML for you.

## Add the workflow labels

After setup, add the completion snippet to the job you want to run on the BYO runner:

```yaml
runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]
```

Do not use `runs-on: self-hosted` alone for RunnerKit-managed runners.

## Completion summary

A successful setup prints and records:

- Runner name.
- Labels.
- Machine target.
- Service name.
- GitHub runner ID.
- State path.
- The copy-paste `runs-on` snippet.

## Troubleshooting

Start with RunnerKit's read-only operations commands before manual SSH troubleshooting.

```bash
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --since 30m --lines 200
runnerkit doctor --repo owner/name
runnerkit doctor --repo owner/name --deep
runnerkit doctor --repo owner/name --deep --with-log-snippets
```

Review logs before sharing; redaction is best-effort for workflow-produced secrets.

- **Runner died after a heavy job (OOM / linker killed):** See [Host resources and OOM](troubleshooting/host-resources.md). Preflight may warn early; `doctor --deep` collects bounded journal hints (**RKD-BOOT-018**). Optional `RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES` raises or lowers the MemAvailable warning threshold (bytes).

- **SSH connection fails**: Confirm `ssh user@host` works from the same machine and that the host/port are correct.
- **Host key changed**: Stop and verify the machine identity. RunnerKit fails closed when the stored fingerprint differs from the observed fingerprint.
- **Unsupported OS or architecture**: Use Linux `x64` or `arm64`; pass `--allow-unknown-linux` only when you understand the best-effort risk.
- **sudo or systemd missing**: Use a systemd Linux host where your SSH user can run required sudo setup commands.
- **Runner service is not active**: Run runnerkit status --repo owner/name, then runnerkit logs --repo owner/name --since 30m and runnerkit doctor --repo owner/name before restarting anything manually.
- **GitHub runner stays offline**: Check outbound HTTPS to GitHub, the runner service logs, and the repository Actions runner settings.

### If something fails

Look for a `RKD-<COMPONENT>-NNN` code in the failure output. The accompanying
`See: <URL>` link points at a Symptom / Diagnosis / Fix entry in
[docs/troubleshooting/](troubleshooting/README.md). Most BYO failures fall in:

- [SSH](troubleshooting/ssh.md) — connectivity, host-key, key path
- [Bootstrap and service](troubleshooting/bootstrap.md) — preflight (disk, memory, swap, tools, …), runner user, systemd
- [Host resources and OOM](troubleshooting/host-resources.md) — RAM/swap, parallel native CI, journal hints
- [GitHub runner](troubleshooting/github.md) — registration, online verification

## Recovery

Preview recovery before changing the host:

```bash
runnerkit recover --repo owner/name --dry-run
runnerkit recover --repo owner/name --restart-service --yes
runnerkit recover --repo owner/name --reinstall-service --yes
runnerkit recover --repo owner/name --reregister --yes
```

Do not blindly rerun runnerkit up for recovery; start with status, logs, doctor, and recover --dry-run.

RunnerKit fails closed on SSH host-key mismatch and will not recover until you verify the machine identity.

## Cleanup

```bash
runnerkit down --repo owner/name --dry-run
runnerkit down --repo owner/name
runnerkit down --repo owner/name --yes
runnerkit down --repo owner/name --github-runner-id 123 --yes
```

RunnerKit down removes only RunnerKit-managed runner-specific BYO artifacts recorded in state.

RunnerKit down does not delete the BYO machine, shared users, shared /var/lib/runnerkit parents, or unrelated user data.

Use destroy only for future cloud resources; BYO cleanup uses down.

If SSH is unreachable, RunnerKit can delete the stale GitHub runner and keep local state with remote_cleanup_pending so you know what may remain on the host.
