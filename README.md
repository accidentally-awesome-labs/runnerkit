# RunnerKit

RunnerKit is a CLI-first tool for solo developers who want reliable GitHub Actions self-hosted runners without manually copying registration commands, wiring services, or guessing which labels to use. The v1 path starts with GitHub Actions, local state, strict redaction, and a BYO Linux host flow.

## Install

RunnerKit is distributed via two channels (D-01).

### Homebrew (macOS, Linux)

```bash
brew tap accidentally-awesome-labs/tap
brew install --cask runnerkit
```

This uses the official tap repo (`accidentally-awesome-labs/homebrew-tap`) and installs the
latest release. Fully-qualified install also works:

```bash
brew install --cask accidentally-awesome-labs/tap/runnerkit
```

Upgrade with `brew upgrade --cask runnerkit`.

### GitHub Releases (all supported platforms)

Supported CLI host platforms (D-02):

| OS    | Architecture | Asset name                                      |
|-------|--------------|-------------------------------------------------|
| macOS | arm64        | `runnerkit_<version>_darwin_arm64.tar.gz`       |
| macOS | amd64        | `runnerkit_<version>_darwin_amd64.tar.gz`       |
| Linux | amd64        | `runnerkit_<version>_linux_amd64.tar.gz`        |
| Linux | arm64        | `runnerkit_<version>_linux_arm64.tar.gz`        |

Linux 386 and 32-bit ARM are not supported.

Download from <https://github.com/accidentally-awesome-labs/runnerkit/releases/latest>:

```bash
# Replace v1.0.0 with the desired tag
TAG=v1.0.0
OS=$(uname -s | tr '[:upper:]' '[:lower:]')      # darwin or linux
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

curl -fsSL -O "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_${OS}_${ARCH}.tar.gz"
curl -fsSL -O "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt"
curl -fsSL -O "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt.sigstore.json"
```

### Verify the release (D-05)

Verify the cosign keyless signature on `checksums.txt` (proves the file was
produced by the upstream release workflow), then verify the archive against
the checksums file.

```bash
# Replace v1.0.0 with the tag you downloaded.
TAG=v1.0.0

cosign verify-blob \
  --bundle  runnerkit_${TAG#v}_checksums.txt.sigstore.json \
  --certificate-identity   "https://github.com/accidentally-awesome-labs/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_${TAG#v}_checksums.txt

sha256sum -c runnerkit_${TAG#v}_checksums.txt --ignore-missing
```

A successful run prints `Verified OK` (cosign) and one `OK` line per archive
(sha256sum). If either step fails, do NOT install the binary.

Then extract and place `runnerkit` on your `PATH`:

```bash
tar -xzf runnerkit_${TAG#v}_${OS}_${ARCH}.tar.gz
sudo install -m 0755 runnerkit /usr/local/bin/runnerkit
```

Troubleshooting install verification failures: see
[docs/troubleshooting/README.md](docs/troubleshooting/README.md).

## Safety: persistent vs ephemeral

Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows. The lower-case validation phrase used throughout the docs is: persistent self-hosted runners.

Read the [Self-hosted Runner Safety Guide](docs/safety.md) before choosing a mode. A short summary:

- Persistent mode reuses one runner across many jobs and is intended for trusted private repositories.
- Ephemeral mode gives stronger isolation by using one-job GitHub runner registration, but it is not a clean VM by itself.
- Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online.

Use ephemeral cloud runner for public, fork-based, or otherwise untrusted workloads:

```bash
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --yes
runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --ephemeral-ttl 24h
```

Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.

Workflow snippets RunnerKit prints (RunnerKit prints labels/snippets and does not edit workflow YAML):

```yaml
runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]
```

```yaml
runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, ephemeral]
```

Do not use `runs-on: self-hosted` alone for RunnerKit-managed runners.

## BYO persistent runner quickstart

Use the BYO persistent runner quickstart when you already have SSH access to a trusted Linux systemd machine:

```bash
runnerkit up --repo owner/name --host user@host
```

See [docs/byo-quickstart.md](docs/byo-quickstart.md) for prerequisites, safety notes, the workflow label snippet, and troubleshooting. For OS/arch targeting and self-hosted vs hosted differences, see [docs/runner-platforms.md](docs/runner-platforms.md).

First-time BYO setup against a sudo-with-password host? See [Sudo Setup](docs/byo-quickstart.md#sudo-setup) — `runnerkit byo-prepare --host user@host` installs a scoped sudoers entry once, then every `runnerkit up` runs passwordlessly. If repository workflows use `sudo apt-get` on the runner, add `--grant-ci-sudo` on **Linux** so the job user matches hosted-runner ergonomics ([RKD-GH-008](docs/troubleshooting/github.md#rkd-gh-008)).

## Recommended cloud runner quickstart

Use the recommended cloud runner quickstart when you do not already have a Linux machine and want RunnerKit to provision the Hetzner cloud path:

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

RunnerKit supports one recommended cloud path.
The default cloud runner is persistent and intended for trusted private repositories.
For public, fork-based, or otherwise untrusted workflows, use `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner`.
RunnerKit prints labels/snippets and does not edit workflow YAML.
Cost estimates are approximate and billing stops only after relevant provider resources are destroyed or verified non-billable.

## Troubleshooting

If a `runnerkit` command prints a `See: <URL>` line, the URL points at a stable
entry in [docs/troubleshooting/](docs/troubleshooting/README.md). Index by
component:

- [Auth and safety](docs/troubleshooting/auth.md) — `RKD-AUTH-NNN`
- [SSH](docs/troubleshooting/ssh.md) — `RKD-SSH-NNN`
- [Bootstrap and service](docs/troubleshooting/bootstrap.md) — `RKD-BOOT-NNN`
- [GitHub runner](docs/troubleshooting/github.md) — `RKD-GH-NNN`
- [Cloud provider](docs/troubleshooting/provider.md) — `RKD-PROV-NNN`
- [Cleanup, state, CLI input](docs/troubleshooting/cleanup.md) — `RKD-CLEAN-NNN`, `RKD-STATE-NNN`, `RKD-CORE-NNN`

You can override the URL prefix the CLI prints with
`RUNNERKIT_DOCS_BASE=https://your-docs-host/runnerkit`.

## BYO operations

Start with read-only operations before manual SSH troubleshooting:

```bash
runnerkit status --repo owner/name
runnerkit logs --repo owner/name --lines 50
runnerkit logs --repo owner/name --since 30m --lines 200
runnerkit doctor --repo owner/name
runnerkit recover --repo owner/name --dry-run
runnerkit recover --repo owner/name --restart-service --yes
runnerkit down --repo owner/name --dry-run
runnerkit down --repo owner/name --yes
```
