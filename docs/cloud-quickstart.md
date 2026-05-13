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

## First boot, sudo, and non-interactive bootstrap

RunnerKit injects **cloud-init user-data** when the Hetzner server is created. During first boot (before you rely on SSH for install), cloud-init stages the same **scoped** sudoers drop-in as the BYO one-liner (`/etc/sudoers.d/runnerkit-installer` — see [`install.sh`](../install.sh) and `internal/bootstrap/sudoers.go`), validates it with **`visudo`**, then installs it. The SSH user (`runnerkit-admin` by default) also keeps a quoted cloud-init **`NOPASSWD:ALL`** line for edge cases; the scoped file is what makes **`sudo apt-get`** and the rest of bootstrap reliable over a non-PTY channel.

The VM records **`runnerkit-cloud-init-v3`** in **`/var/lib/runnerkit/cloud-init.json`** and in RunnerKit state for support/debug correlation. Readiness waits for **`cloud-init status` `done`** (not `error`); a failed `runcmd` (for example **`visudo`** rejecting staged sudoers) no longer masks as success. This path applies only to **RunnerKit-provisioned** Hetzner VMs. If you instead use **`--host user@…`** against a generic cloud image you created yourself, treat it like BYO: run **`runnerkit init --print-install-command`** on the host when sudo requires a password.

## Pre-installed software (GitHub runner parity)

RunnerKit provisions cloud runners with the same software available on GitHub-hosted Ubuntu 24.04 runners:

- **~75 apt packages** installed via cloud-init at first boot: `build-essential`, `gcc`, `g++`, `make`, `curl`, `jq`, `unzip`, `wget`, `pkg-config`, `gnupg2`, `sqlite3`, `libssl-dev`, and more.
- **Language runtimes**: Node.js 20 LTS, Python 3 with pip/venv, Go (latest stable), Rust (latest stable), Java 17, .NET 8, Ruby (system).
- **Container tools**: Docker CE with buildx and compose; the runner service user is added to the `docker` group.
- **Browser testing**: Google Chrome, ChromeDriver (matching version), Firefox, Geckodriver.
- **CLI tools**: GitHub CLI (`gh`), CMake, Ninja, zstd.

Runtimes and tools are installed by the `setup_runner_image` bootstrap step after cloud-init completes. The cloud-init timeout is 15 minutes to accommodate the expanded package set on smaller VMs.

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

## Pre-installing CI dependencies

RunnerKit **auto-detects** packages your workflows need by scanning `.github/workflows/*.yml` for `apt-get install` / `apt install` commands. When you run `runnerkit up` from a repo checkout, detected packages are merged in automatically and printed to stderr:

```
Auto-detected 5 workflow package(s): libsecret-1-dev, dbus-x11, gnome-keyring, libpango1.0-dev, libssl-dev
```

You can also specify packages explicitly with `--extra-packages` (they merge with auto-detected ones):

```bash
runnerkit up --repo owner/name --cloud hetzner \
  --extra-packages "libsecret-1-dev,dbus-x11,gnome-keyring,libpango1.0-dev"
```

For cloud runners, extra packages are installed via cloud-init during first boot — before RunnerKit connects over SSH. For BYO runners, they are installed alongside missing tools during the `fix_dependencies` bootstrap step.

Extra packages are saved in RunnerKit state, so `runnerkit upgrade-runner` re-installs them automatically.

You can also set them in `.runnerkit/config.yaml` so every `runnerkit up` for this project inherits them:

```yaml
defaults:
  repo: owner/name
  extra_packages:
    - libsecret-1-dev
    - dbus-x11
    - gnome-keyring
```

Package names must be valid for the target OS package manager (apt on Ubuntu). Only alphanumerics, hyphens, dots, colons, underscores, and `+` are allowed.

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
runnerkit doctor --repo owner/name --deep
```

For ephemeral runners, `runnerkit logs` also surfaces the preserved finalizer log archive after the one job runs (or after the TTL safeguard fires).

Preflight and `doctor` surface **RAM/swap** warnings and optional **journal OOM hints** the same way as BYO; see [Host resources and OOM](troubleshooting/host-resources.md) (`docs/troubleshooting/host-resources.md`).

## Destroy and verify cleanup

Always review the destroy plan before applying cleanup:

```bash
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name
runnerkit destroy --repo owner/name --yes
```

RunnerKit removes local state only after GitHub runner registration and provider cleanup are verified. If cleanup is partial, rerun `runnerkit destroy --repo owner/name` after fixing the blocker; RunnerKit keeps pending checkpoints and provider resource IDs in state.

### If something fails

Look for a `RKD-<COMPONENT>-NNN` code in the failure output. The accompanying
`See: <URL>` link points at a Symptom / Diagnosis / Fix entry in
[docs/troubleshooting/](troubleshooting/README.md). Most cloud failures fall
in:

- [Provider](troubleshooting/provider.md) — `HCLOUD_TOKEN`, quota, partial destroy, billable lingering
- [Bootstrap and service](troubleshooting/bootstrap.md) — same as BYO (includes memory/swap preflight)
- [Host resources and OOM](troubleshooting/host-resources.md) — sizing and `doctor --deep` journal hints
- [GitHub runner](troubleshooting/github.md) — registration, online verification

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
