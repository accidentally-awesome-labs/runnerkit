# RunnerKit

[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

RunnerKit is a command-line tool that installs a GitHub Actions self-hosted
runner on a Linux machine you already have, over SSH, and then helps you
check and clean it up (`status`, `logs`, `doctor`, `down`). It registers one
repository-scoped runner per repository, runs it as a systemd service, and
prints the `runs-on` labels to put in your workflow. It does not edit your
workflows.

**Status:** Experimental. Maintained on a capped-hours basis until a published go/kill decision on 2026-12-21.

## Known issues

This section describes the code on `main`, which will be released as
**v1.3.4**. The latest published release (what Homebrew and the Releases
page install today) is **v1.3.3**; see
[If you are on v1.3.3 or older](#if-you-are-on-v133-or-older) below.

- **BYO setup (the main path).** Repaired in v1.3.4 for fresh Ubuntu 24.04
  x86_64 hosts prepared by the v1.3.4 `install.sh`. So far this was
  validated only on a local password-sudo Ubuntu 24.04 container against a
  fake GitHub API; a run of a real GitHub job is required before v1.3.4 is
  tagged. If that run fails, v1.3.4 will ship with BYO marked as not
  supported. Hosts prepared by an older `install.sh` must re-run the new
  one. The sudoers fragment `install.sh` installs is **root-equivalent**;
  see [docs/security-posture.md](docs/security-posture.md).
- **Docker.** The runner user (`runnerkit-runner`) is in the `docker` group,
  which is **root-equivalent** for every job on the host. Existing hosts get
  the group on their next `runnerkit up --repo owner/name --host user@host
  --replace` (or type `replace owner/name` when prompted).
- **Job sudo.** Workflow steps that run `sudo apt-get` need
  `RUNNERKIT_GRANT_CI_SUDO=1` when you run `install.sh`. That grant is also
  root-equivalent.
- **Cloud and ephemeral.** The Hetzner cloud path is unsupported and requires
  `--experimental` and an explicit `--cloud-region`; the plan shows the price
  the Hetzner API reports. Ephemeral cloud runners are disabled. BYO
  ephemeral mode requires `--experimental`, is not isolation, and has known
  defects (the finalizer runs unprivileged and the TTL is ignored); it is
  untested in this release.
- **Cloud defects** (not fixed while the cloud path is frozen):
  - Readiness does not fail fast: if cloud-init ends in an error, `up`
    retries silently for 15 minutes while the server bills, then fails with
    `cloud_readiness_failed` without cloud-init's details. Run
    `runnerkit destroy --repo owner/name`.
  - `up` always uploads your SSH public key as a new Hetzner key, so it
    fails with `uniqueness_error` (before creating anything) when the key
    is already in the Hetzner project, for example from another
    repository's cloud runner.
  - Server type stock in `--cloud-region` is not checked. When Hetzner has
    none, `up` fails after creating an SSH key and a firewall; run
    `runnerkit destroy` before trying another location.
  - On cloud runners, workflow steps that use `sudo` fail (the job sudo
    grant is BYO-only), there is no swap, and the default `cpx22` is small
    enough to trigger RunnerKit's own low-memory warning. Pass a larger
    Hetzner type with `--cloud-profile` for heavy builds.
  - Once the server is gone or unreachable over SSH, `destroy` deletes the
    Hetzner resources and the GitHub runner but keeps the local record: it
    reports "Cleanup incomplete" and exits 0, and `up` then refuses that
    repository (`cloud_state_exists`) until you remove its entry from
    RunnerKit's state file (check the Hetzner Console first).
  - SSH is open to every IPv4 address unless you pass `--ssh-allowed-cidr`,
    and your SSH key can log in as `root` and as `runnerkit-admin`, which
    has passwordless sudo (SEC-11).
- **Disabled commands.** `upgrade-runner`, `doctor --fix`,
  `recover --reinstall-service` and `recover --reregister` refuse to run
  (exit 2) and print manual steps. The GitHub runner updates itself;
  to re-register, run `runnerkit down` and then `runnerkit up`.
- **Platforms.** Only **Ubuntu x86_64** runner hosts are supported. On arm64
  hosts preflight warns and setup then fails; other Linux distributions pass
  preflight and then fail during setup. macOS and Windows runner hosts are not supported.
- **Other defects:** workflow package auto-detection is on by default and
  can pick up wrong package names; `logs` and the `doctor` OOM hints can
  query the wrong systemd unit; `status` and `doctor` exit 0 even when they
  report errors.

Full list: [CHANGELOG.md](CHANGELOG.md) and
[docs/security-posture.md](docs/security-posture.md).

### If you are on v1.3.3 or older

- Do **not** run `runnerkit upgrade-runner` or `runnerkit doctor --fix`. They
  delete the runner's credentials and re-run `config.sh` with an empty token,
  which unregisters a working runner.
- Do **not** run `runnerkit recover --reinstall-service` or
  `recover --reregister`; they can leave the runner without a service.
- Do **not** run `runnerkit down` or `unregister` on a Hetzner cloud runner:
  it deletes the local record and the server keeps billing. Use
  `runnerkit destroy --repo owner/name`.
- Ephemeral cloud VMs are never destroyed after their job. Run
  `runnerkit destroy --repo owner/name` for each one.
- The "approx €4.90/month" cloud estimate is invented; check the Hetzner
  price list.
- `--ssh-allowed-cidr` with a bare address (no `/32`) opened SSH to every
  IPv4 address instead. Check the firewall of cloud runners created that
  way in the Hetzner Console.
- `runnerkit byo-prepare` does not exist (it was removed in v1.0.8, although
  the v1.3.3 release notes tell you to run it). BYO setup on password-sudo
  hosts fails at `setup_runner_image`.
- Unknown commands, bad flags and `--version` print nothing. Typed
  confirmations fail in a terminal (`--yes` works around it, but also
  accepts a new SSH host key without asking). `doctor --ignore` and
  `up --host` write `config.json` or a `sessions/` directory (containing
  `user@host`) into the current directory; delete them.

## When NOT to use RunnerKit

- **Public repositories.** GitHub-hosted runners are free and unlimited for
  public repositories, and GitHub advises that self-hosted runners should
  almost never be used for public repositories.
- **Untrusted code** (forks, outside contributors). Use GitHub-hosted runners.
  RunnerKit's ephemeral mode is not isolation.
- **To save money at low volume.** Most individual accounts fit in GitHub's
  included free minutes.
- **Ephemeral or autoscaling runner pools.** Use Actions Runner Controller
  (ARC), `actions/scaleset` or a similar tool.
- **macOS or Windows runners.**
- **Teams that want managed runners** with support and an SLA.

RunnerKit fits a single developer or small team with a private repository
and a spare Ubuntu x86_64 machine (homelab box, VPS) that they fully trust
the repository's contributors with.

## What BYO setup installs on the host

`runnerkit up --host` changes the host substantially. On Ubuntu it:

- installs `/etc/sudoers.d/runnerkit-installer` (via `install.sh`, once),
  which is root-equivalent for your SSH user;
- creates the `runnerkit-runner` system user and adds it to the `docker`
  group;
- installs about 70 baseline apt packages (compilers, `pkg-config`, `jq`, …)
  plus a GitHub-hosted-style runner image: Docker CE, Google Chrome,
  ChromeDriver, Firefox, Geckodriver, OpenJDK 17, .NET 8, Node.js 20
  (end of life), Python pip/venv, Go, Rust, `gh`, CMake and Ninja;
- adds up to 6 third-party apt sources (NodeSource, Docker, Google,
  the Mozilla PPA, GitHub CLI and, as a fallback for .NET, Microsoft) and
  downloads Go from go.dev and Rust through `rustup`;
- installs the runner under `/opt/actions-runner/`, work directories under
  `/var/lib/runnerkit/`, and an `actions.runner.*` systemd unit.

Expect roughly 4.5–5 GB of disk use and about 1.4 GB of downloads
(projected, not measured end to end). The preflight disk check only requires
2 GiB free, so check disk space yourself.

## Install the CLI

The CLI runs on macOS 12 Monterey or later, or Linux (amd64 or arm64). The
runner host must be Ubuntu x86_64. Since v1.3.4 the macOS binaries need
macOS 12 (they are built with Go 1.26); on macOS 11 or older, stay on v1.3.3.

### Homebrew (macOS, Linux)

```bash
brew tap accidentally-awesome-labs/tap
brew install --cask runnerkit
```

Or in one step: `brew install --cask accidentally-awesome-labs/tap/runnerkit`.
Upgrade with `brew upgrade --cask runnerkit`.

### GitHub Releases

| OS    | Architecture | Asset name                                |
|-------|--------------|-------------------------------------------|
| macOS | arm64        | `runnerkit_<version>_darwin_arm64.tar.gz` |
| macOS | amd64        | `runnerkit_<version>_darwin_amd64.tar.gz` |
| Linux | amd64        | `runnerkit_<version>_linux_amd64.tar.gz`  |
| Linux | arm64        | `runnerkit_<version>_linux_arm64.tar.gz`  |

Linux 386 and 32-bit ARM are not supported. Pick a tag from
<https://github.com/accidentally-awesome-labs/runnerkit/releases>:

```bash
TAG=vX.Y.Z   # replace with the release tag you chose
OS=$(uname -s | tr '[:upper:]' '[:lower:]')      # darwin or linux
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

curl -fsSL -O "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_${OS}_${ARCH}.tar.gz"
curl -fsSL -O "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt"
curl -fsSL -O "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt.sigstore.json"
```

Verify the cosign keyless signature on the checksums file (it proves the file
came from the upstream release workflow), then the archive:

```bash
cosign verify-blob \
  --bundle  runnerkit_${TAG#v}_checksums.txt.sigstore.json \
  --certificate-identity   "https://github.com/accidentally-awesome-labs/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_${TAG#v}_checksums.txt

sha256sum -c runnerkit_${TAG#v}_checksums.txt --ignore-missing
```

You should see `Verified OK` and one `OK` line for the archive. If either
check fails, do not install the binary. Then:

```bash
tar -xzf runnerkit_${TAG#v}_${OS}_${ARCH}.tar.gz
sudo install -m 0755 runnerkit /usr/local/bin/runnerkit
runnerkit --version
```

Verification problems: [docs/troubleshooting/README.md](docs/troubleshooting/README.md).

## BYO persistent runner quickstart

For a trusted private repository and an Ubuntu x86_64 host you can SSH into:

1. Once, prepare the host. Print the command on your workstation and run it
   on the host (it asks for your sudo password there):

   ```bash
   runnerkit init --print-install-command
   ```

2. Install and register the runner:

   ```bash
   runnerkit up --repo owner/name --host user@host
   ```

3. Put the printed labels in your workflow job:

   ```yaml
   runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]
   ```

   Do not use `runs-on: self-hosted` alone for RunnerKit-managed runners.

See [docs/byo-quickstart.md](docs/byo-quickstart.md) for prerequisites,
tokens, sudo setup and troubleshooting. For workflow `sudo apt-get` as the
runner user, re-run `install.sh` with `RUNNERKIT_GRANT_CI_SUDO=1`
([RKD-GH-008](docs/troubleshooting/github.md#rkd-gh-008)); this is
root-equivalent.

## Safety

Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.
Every job runs on the same machine, so a malicious job can leave a backdoor
for later jobs or read their secrets.
Use persistent self-hosted runners only for trusted private repositories.
RunnerKit refuses to set up a persistent runner for a public repository
unless you pass `--allow-public-repo-risk`. With RunnerKit's privileges
(root-equivalent sudoers, `docker` group), anyone who can run a workflow on
the runner can get root on the host.

For public or untrusted code, use GitHub-hosted runners. Read the
[Self-hosted Runner Safety Guide](docs/safety.md) and
[docs/security-posture.md](docs/security-posture.md) before you install.

## Hetzner cloud runner (experimental)

RunnerKit can also create a Hetzner Cloud server and install the runner on
it. This path is **experimental and unsupported**, and the server is billed
by Hetzner until it is deleted:

```bash
export HCLOUD_TOKEN=...
runnerkit up --repo owner/name --experimental --cloud hetzner --cloud-region <location> --dry-run
runnerkit up --repo owner/name --experimental --cloud hetzner --cloud-region <location>
runnerkit destroy --repo owner/name --dry-run
runnerkit destroy --repo owner/name
```

The plan shows the price reported by the Hetzner API at plan time (server
type in that location plus the primary IPv4, excluding traffic overage).
Ephemeral cloud runners are disabled. Never use `runnerkit down` for a cloud
runner.
Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup.
Details: [docs/cloud-quickstart.md](docs/cloud-quickstart.md).

## Day-to-day commands

Start with the read-only commands before changing anything by hand:

```bash
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --since 30m --lines 200
runnerkit doctor --repo owner/name
runnerkit doctor --repo owner/name --deep
runnerkit recover --repo owner/name --dry-run
runnerkit recover --repo owner/name --restart-service --yes
runnerkit down --repo owner/name --dry-run
runnerkit down --repo owner/name --yes
```

`runnerkit list` shows every saved runner grouped by host. State lives in
`~/.local/state/runnerkit/` (override with `RUNNERKIT_STATE_DIR`).

## Troubleshooting

When a command prints `See: <URL>`, the link points to an entry in
[docs/troubleshooting/](docs/troubleshooting/README.md):

- [Auth and safety](docs/troubleshooting/auth.md) — `RKD-AUTH-NNN`
- [SSH](docs/troubleshooting/ssh.md) — `RKD-SSH-NNN`
- [Bootstrap and service](docs/troubleshooting/bootstrap.md) — `RKD-BOOT-NNN`
- [Host resources and OOM](docs/troubleshooting/host-resources.md)
- [GitHub runner](docs/troubleshooting/github.md) — `RKD-GH-NNN`
- [Cloud provider](docs/troubleshooting/provider.md) — `RKD-PROV-NNN`
- [Cleanup, state, CLI input](docs/troubleshooting/cleanup.md) — `RKD-CLEAN-NNN`, `RKD-STATE-NNN`, `RKD-CORE-NNN`

Set `RUNNERKIT_DOCS_BASE=https://your-docs-host/runnerkit` to change the
printed URL prefix.

Set `RUNNERKIT_LOG` (`info`, `warn`, `error` or `debug`) to write JSON logs
of GitHub API calls, SSH steps and Hetzner calls; `RUNNERKIT_LOG_DEST` picks
the sink (`stderr` by default, `stdout`, or `file:/path.jsonl`). Tokens are
redacted on a best-effort basis; file logs are created world-readable.

## Contributing, security and license

- [CONTRIBUTING.md](CONTRIBUTING.md): DCO sign-off, the scope rule, and how
  to run the tests.
- [SECURITY.md](SECURITY.md): how to report a vulnerability privately.
- [CHANGELOG.md](CHANGELOG.md): release history.
- Maintainer release notes: [docs/maintainers.md](docs/maintainers.md).
- License: [Apache-2.0](LICENSE).
