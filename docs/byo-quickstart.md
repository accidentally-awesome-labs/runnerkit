# BYO Persistent Runner Quickstart

This guide connects RunnerKit to an existing trusted Ubuntu x86_64 host over SSH, installs a repository-scoped persistent GitHub Actions runner, and gives you the labels for your workflow job.

**Status (v1.3.4): BYO is not supported in this release.** The v1.3.4 repair of the BYO path was validated only on a local password-sudo container against a fake GitHub API, not with a real GitHub job, so `runnerkit up` and `runnerkit register` refuse a BYO host unless you add `--accept-known-issues` to the commands below. See [Known issues](../README.md#known-issues).

## Prerequisites

- A RunnerKit binary or local checkout you can run.
- GitHub authentication through `gh auth login` or `RUNNERKIT_GITHUB_TOKEN` with repository Administration read/write and Metadata read for the target repo.
- A trusted private GitHub repository such as `owner/name`.
- SSH access to an **Ubuntu x86_64** systemd host (24.04 is the tested release) as `user@host`. arm64 and other distributions are not supported ([runner platforms](runner-platforms.md)).
- Sudo ability on that host (once, to run `install.sh`).
- Roughly 5 GB of free disk: RunnerKit installs a large runner image (see [What RunnerKit does](#what-runnerkit-does)).

## Safety warning

Persistent self-hosted runners are intended for trusted private repositories; public, fork-based, or otherwise untrusted workflows can execute code on your machine.

Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.

Do not use this persistent BYO path for public pull requests or untrusted workflow code. Use GitHub-hosted runners for those; RunnerKit's ephemeral mode is not isolation.

Anyone who can run a workflow on this runner can get root on the host: the sudoers fragment below is root-equivalent and the runner user is in the `docker` group. See [security posture](security-posture.md).

For full guidance see the [Self-hosted Runner Safety Guide](safety.md).

## Sudo setup (one-time on the host)

Bootstrap runs over SSH without a TTY, so the SSH user must already have **passwordless sudo** for RunnerKit’s commands. Run the **one-time install on the runner machine** (interactive sudo once), then use `runnerkit up` from your workstation. `runnerkit register` adds more repositories to a host that `up` has set up; on a host without the shared `runnerkit-runner` user it refuses with `lifecycle_foundation_missing` and names the `up` command to run.

1. Print the install command from your workstation:

   ```bash
   runnerkit init --print-install-command
   ```

2. SSH to the Linux host and paste the `curl … install.sh | sudo bash` line (or download `install.sh` from the matching [GitHub release](https://github.com/accidentally-awesome-labs/runnerkit/releases) and verify checksums per `README.md`).

This installs `/etc/sudoers.d/runnerkit-installer` with NOPASSWD for the bootstrap command list (`apt-get`/`dnf`/`yum`, `useradd`, `install`, `curl`, `sha256sum`, `tee`, `gpg`, `mkdir`, `unzip`, `usermod`, `dpkg`, `add-apt-repository`, `chown`, `chmod`, `cp`, `cat`, `ln`, `rm`, `su`, `tar`, `systemctl`, runner `svc.sh`). The list is generated from RunnerKit's bootstrap template. It is **root-equivalent**: several of these commands with arbitrary arguments give a root shell, so treat it as passwordless root for your SSH user. See [RKD-BOOT-015](troubleshooting/bootstrap.md#rkd-boot-015) if anything blocks.

If the host was prepared with an `install.sh` from v1.3.3 or earlier, **run the current one again**: the old one lacked 16 paths and setup fails at `setup_runner_image`. (`runnerkit byo-prepare`, which the v1.3.3 notes mention, was removed in v1.0.8.)

To revert on the host:

```bash
sudo rm -f /etc/sudoers.d/runnerkit-installer
sudo visudo -c
```

For the full list of what to revoke, see [If you already installed RunnerKit](security-posture.md#if-you-already-installed-runnerkit).

### Decision tree

| Scenario | Path |
| --- | --- |
| First-time BYO host | `runnerkit init --print-install-command` → run on host → `runnerkit up` (then `runnerkit register` for more repositories) |
| CI / automation / `--json` | Host must already have install applied; otherwise RunnerKit exits with `host_install_required` and a boxed install command |
| Manual NOPASSWD ALL for the SSH user | Works; equivalent in effect to install.sh, which is also root-equivalent |

## Run setup

```bash
runnerkit up --repo owner/name --host user@host
```

Useful automation flags:

```bash
runnerkit up --repo owner/name --host user@host --yes
runnerkit up --repo owner/name --host user@host:2222 --ssh-key ~/.ssh/id_ed25519 --yes
```

RunnerKit **auto-detects** packages your workflows need by scanning `.github/workflows/*.yml` for `apt-get install` / `apt install` commands. Detected packages are merged in automatically when you run from a repo checkout. Auto-detection always runs when you start `up` in a directory with `.github/workflows/` (there is no switch to turn it off) and can pick up wrong package names from unusual lines, which then fail the install. Check the printed list, or run `up` from another directory.

You can also specify packages explicitly with `--extra-packages` (they merge with auto-detected ones):

```bash
runnerkit up --repo owner/name --host user@host \
  --extra-packages "libsecret-1-dev,dbus-x11,gnome-keyring"
```

These are installed alongside RunnerKit's required tools during the `fix_dependencies` bootstrap step and recorded in state.

RunnerKit prompts for unknown SSH host keys. Verify the `SHA256:` fingerprint before accepting it. `--yes` accepts a new host key without asking.

## What RunnerKit does

- Resolves and checks GitHub repository permissions.
- Blocks risky public/fork repository defaults unless you explicitly override the safety gate.
- Verifies the SSH host key and records the accepted fingerprint in local state.
- Runs SSH preflight checks for Linux, architecture, systemd, sudo, disk (at least 2 GiB, which is less than the image needs), **MemAvailable / swap** (warnings for low RAM or no-swap small hosts — [RKD-BOOT-016/017](troubleshooting/bootstrap.md#rkd-boot-016)), tools, time, network, and runner conflicts.
- Installs about 70 baseline apt packages (`build-essential`, `pkg-config`, `jq`, …) with the required tools (`fix_dependencies`).
- Creates or reuses the non-root `runnerkit-runner` service user (`create_runner_user`).
- On Ubuntu, installs a GitHub-hosted-style runner image (`setup_runner_image`): Docker CE, Google Chrome, ChromeDriver, Firefox, Geckodriver, OpenJDK 17, .NET 8, Node.js 20 (end of life), Python pip/venv, Go, Rust, `gh`, CMake, Ninja and zstd, from up to 6 third-party apt sources plus go.dev and `rustup`. It adds `runnerkit-runner` to the `docker` group, which is **root-equivalent** for every job. Expect roughly 4.5–5 GB of disk and about 1.4 GB of downloads (projected). Hosts set up by an earlier version re-run this step once on their next `runnerkit up --repo owner/name --host user@host --replace` (or type `replace owner/name` when prompted; RunnerKit asks before it touches the host).
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
- **Unsupported OS or architecture**: Use Ubuntu x86_64. arm64 and other distributions are not supported and fail during setup even when preflight passes; `--allow-unknown-linux` does not change that.
- **Setup fails at a named step**: the error names the step and prints `Failed command (exit N): …` plus the last remote output lines. See [Bootstrap and service](troubleshooting/bootstrap.md).
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
```

`recover --reinstall-service` and `recover --reregister` are disabled in v1.3.4 (`command_disabled`, exit 2) because they could leave the runner without a service. When the service or registration is gone, `recover --dry-run` and `doctor` print the manual steps: `runnerkit down --repo owner/name`, then `runnerkit up --repo owner/name --host user@host`.

`runnerkit upgrade-runner` is disabled too; the runner updates itself. See [upgrade.md](upgrade.md).

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

Use destroy only for RunnerKit-created Hetzner servers; BYO cleanup uses down.

If SSH is unreachable, RunnerKit can delete the stale GitHub runner and keep local state with remote_cleanup_pending so you know what may remain on the host.
