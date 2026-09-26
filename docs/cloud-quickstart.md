# Hetzner Cloud Runner Quickstart (experimental)

RunnerKit can create a Hetzner Cloud server and install a persistent GitHub
Actions runner on it. **This path is experimental and unsupported.** The
server is billed by Hetzner from the moment it is created until it is
deleted, and RunnerKit requires you to opt in explicitly:

- `--experimental` is required with `--cloud` (`experimental_required`,
  exit 2);
- `--cloud-region` is required and has no default (`cloud_region_required`,
  exit 2). Check in the Hetzner Cloud Console where your server type is
  currently offered;
- ephemeral cloud runners are disabled (`ephemeral_cloud_disabled`, exit 2):
  the VM was never destroyed after its job and kept billing.

For public, fork-based, or otherwise untrusted workflows, use
GitHub-hosted runners instead. If you already have a Linux machine, the
[BYO quickstart](byo-quickstart.md) is the supported path.

## Prerequisites

- A trusted private GitHub repository.
- GitHub authentication that can manage repository self-hosted runners.
- A Hetzner Cloud API token from the Hetzner Cloud Console (project-scoped).
- An SSH public key available via `--ssh-key <path>` plus `<path>.pub`, or a standard local public key such as `~/.ssh/id_ed25519.pub`.

```bash
export HCLOUD_TOKEN=...
```

`HETZNER_CLOUD_TOKEN` is also accepted as an alias. RunnerKit uses provider credentials from the environment and does not persist provider API tokens in local state, logs, diagnostics, or command output.

## Provision cloud runner

Preview first; the dry run creates nothing and shows the plan and price:

```bash
runnerkit up --repo owner/name --experimental --cloud hetzner --cloud-region <location> --dry-run
runnerkit up --repo owner/name --experimental --cloud hetzner --cloud-region <location>
runnerkit up --repo owner/name --experimental --cloud hetzner --cloud-region <location> --yes
```

The default server type is `cpx22` (`--cloud-profile` changes it). The
runner is persistent and intended for trusted private repositories.

If saved state for the repository still records Hetzner server, firewall or
SSH-key IDs, `up` refuses to replace it (`cloud_state_exists`), even with
`--replace`. Run `runnerkit destroy --repo owner/name` first.

## Price and billing

The plan shows the price **reported by the Hetzner API at plan time**: the
server type's price in the chosen location plus the primary IPv4 price,
hourly and monthly, net and gross. It excludes traffic overage. Example
shape of the human output:

```
Estimated cost: EUR <x>/hour gross (<y> net), EUR <x>/month gross (<y> net)
Monthly price breakdown: ...
Prices reported by the Hetzner API at <time>; excludes traffic overage
```

If Hetzner reports no price for the server type (or its primary IPv4) in
that location, RunnerKit refuses before creating anything
(`cloud_location_unpriced`, exit 2); choose another `--cloud-region`. A
failing pricing API call is reported as `cloud_plan_failed`.

With `--json`, `estimated_monthly_cost` and `estimated_hourly_cost` (in
`cloud_plan` and at the top level of the dry-run payload) are objects:

```json
{
  "amount": "<gross total>",
  "net": "<net total>",
  "gross": "<gross total>",
  "currency": "EUR",
  "period": "month",
  "source": "hetzner_api",
  "fetched_at": "<RFC3339 time>",
  "components": [
    {"resource": "server:cpx22", "net": "...", "gross": "..."},
    {"resource": "primary_ipv4", "net": "...", "gross": "..."}
  ]
}
```

Before v1.3.4 these fields were strings with a hard-coded, wrong estimate.

Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup.
Cost estimates are approximate and billing stops only after relevant provider resources are destroyed or verified non-billable.

## First boot, sudo, and non-interactive bootstrap

RunnerKit injects **cloud-init user-data** when the Hetzner server is created. During first boot, cloud-init installs the same installer sudoers drop-in as the BYO `install.sh` (`/etc/sudoers.d/runnerkit-installer`, generated from `internal/bootstrap/sudoers.go`), validated with **`visudo`**. That fragment is root-equivalent. The SSH user (`runnerkit-admin` by default) also has **`NOPASSWD:ALL`** from cloud-init. See [security-posture.md](security-posture.md).

The VM records **`runnerkit-cloud-init-v3`** in **`/var/lib/runnerkit/cloud-init.json`** and in RunnerKit state. Readiness waits for **`cloud-init status` `done`** and does not accept `error` as ready. It does not fail fast, though: any readiness failure is retried until the timeout (15 minutes by default, `RUNNERKIT_CLOUD_INIT_TIMEOUT`), while the server bills. If readiness fails, run `runnerkit destroy --repo owner/name` and start again.

The firewall allows SSH from `0.0.0.0/0` unless you pass `--ssh-allowed-cidr` (for example your own IP as `/32`).

If you instead use **`--host user@…`** against a server you created yourself, treat it like BYO: run **`runnerkit init --print-install-command`** on the host when sudo requires a password.

## Pre-installed software

Cloud runners get a large subset of the GitHub-hosted Ubuntu 24.04 image:

- **About 70 baseline apt packages** installed by cloud-init at first boot: `build-essential`, `gcc`, `g++`, `make`, `curl`, `jq`, `unzip`, `wget`, `pkg-config`, `gnupg2`, `sqlite3`, `libssl-dev`, and more.
- **Language runtimes**: Node.js 20 (end of life), Python 3 with pip/venv, Go (latest stable), Rust (latest stable, for the runner user), Java 17, .NET 8.
- **Container tools**: Docker CE with buildx and compose. The runner service user is in the `docker` group, which is **root-equivalent** for every job.
- **Browser testing**: Google Chrome, ChromeDriver (matching version), Firefox, Geckodriver.
- **CLI tools**: GitHub CLI (`gh`), CMake, Ninja, zstd.

Runtimes and tools are installed by the `setup_runner_image` bootstrap step after cloud-init completes.

## Pre-installing CI dependencies

RunnerKit **auto-detects** packages your workflows need by scanning `.github/workflows/*.yml` for `apt-get install` / `apt install` commands. When you run `runnerkit up` from a repo checkout, detected packages are merged in automatically and printed to stderr:

```
Auto-detected 5 workflow package(s): libsecret-1-dev, dbus-x11, gnome-keyring, libpango1.0-dev, libssl-dev
```

Auto-detection is on by default and can pick up wrong package names from unusual lines, which then fail the install. Check the printed list.

You can also specify packages explicitly with `--extra-packages` (they merge with auto-detected ones):

```bash
runnerkit up --repo owner/name --experimental --cloud hetzner --cloud-region <location> \
  --extra-packages "libsecret-1-dev,dbus-x11,gnome-keyring,libpango1.0-dev"
```

For cloud runners, extra packages are installed via cloud-init during first boot. Package names must be valid apt package names: only alphanumerics, hyphens, dots, colons, underscores, and `+` are allowed.

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
runnerkit doctor --repo owner/name --deep
```

Preflight and `doctor` surface **RAM/swap** warnings and optional **journal OOM hints** the same way as BYO; see [Host resources and OOM](troubleshooting/host-resources.md) (`docs/troubleshooting/host-resources.md`).

## Destroy and verify cleanup

Always review the destroy plan before applying cleanup:

```bash
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name
runnerkit destroy --repo owner/name --yes
```

RunnerKit removes local state only after GitHub runner registration and provider cleanup are verified. If cleanup is partial, rerun `runnerkit destroy --repo owner/name` after fixing the blocker; RunnerKit keeps pending checkpoints and provider resource IDs in state.

Never use `runnerkit down` for a cloud runner. It refuses cloud state
(`wrong_cleanup_command`); in v1.3.3 and earlier it deleted the local record
and left the server billing. If you lost the local state, delete servers,
firewalls, SSH keys and primary IPs labelled `runnerkit=true` in the Hetzner
Cloud Console.

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

- Experimental and unsupported; frozen except for safety fixes.
- One persistent runner per server; no ephemeral cloud runners.
- The Hetzner server bills until `runnerkit destroy` verifies cleanup.
- Only Hetzner is supported.
- RunnerKit prints labels/snippets and does not edit workflow YAML.

## Optional live smoke test

A live smoke test requires real Hetzner credentials and creates billable resources. Run it only in a repository and Hetzner project you control, then verify cleanup with:

```bash
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name --yes
```
