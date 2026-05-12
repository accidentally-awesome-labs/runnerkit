# Runner platforms (GitHub Actions labels)

RunnerKit installs a **GitHub Actions self-hosted runner**. Workflows select runners with `runs-on:` — typically **labels** you assign at registration time (RunnerKit sets `runnerkit-<owner>-<repo>` plus defaults).

This page maps **where RunnerKit runs today** versus common **platform combinations** teams target with labels (`ubuntu-latest`-style capacity on your own hardware).

## What RunnerKit supports today

| Host OS (SSH target for `runnerkit up`) | Architectures | Notes |
| --- | --- | --- |
| **Linux** (systemd) | **x86_64**, **arm64** | Primary BYO + [Hetzner](troubleshooting/provider.md) cloud path. Bootstrap uses bash over SSH. Preflight warns when **MemAvailable** is low or swap is absent on small hosts — see [Host resources](troubleshooting/host-resources.md). |
| **macOS** | **Apple Silicon (arm64)**, **Intel (amd64)** | Supported where SSH + tools align with preflight; treat as advanced BYO. |

RunnerKit’s remote bootstrap assumes a **Unix** shell environment over SSH.

## Windows runners

GitHub supports Windows self-hosted runners, but RunnerKit’s **current** install/bootstrap path is built around **Linux/macOS** SSH + systemd-style lifecycle. Running Actions on **Windows x64 / Windows ARM64** with RunnerKit is **not** a supported golden path in this release — use an official Windows runner install from GitHub or another provisioning tool, or contribute a Windows bootstrap backend.

## Choosing labels for multi-platform CI

Standard practice:

1. Register one runner per machine (or pool).
2. Attach labels that describe the **OS + arch** your workflows need, for example:
   - `self-hosted`, `linux`, `x64`
   - `self-hosted`, `linux`, `arm64`
   - `self-hosted`, `macOS`, `arm64`

RunnerKit’s repo-scoped label remains available for targeting **this** runner installation.

3. In workflows, pin jobs explicitly:

```yaml
jobs:
  build-linux-amd64:
    runs-on: [self-hosted, linux, x64]
  build-linux-arm64:
    runs-on: [self-hosted, linux, arm64]
```

Use **matrix** only across runners that actually exist in your org/repo settings — GitHub will queue jobs until a matching online runner appears.

## Hosted runners vs self-hosted

| | GitHub-hosted (`ubuntu-latest`, etc.) | RunnerKit self-hosted |
| --- | --- | --- |
| OS images | Curated, uniform | Your machine / VM |
| `sudo` in workflows | Passwordless for job user | Often **not** — on **Linux**, run [`install.sh` with `RUNNERKIT_GRANT_CI_SUDO=1`](byo-quickstart.md#sudo-setup-one-time-on-the-host) so workflow `sudo apt-get` works without a TTY ([RKD-GH-008](troubleshooting/github.md#rkd-gh-008)) |

For isolation-sensitive workloads, prefer **ephemeral** runners ([safety](safety.md)) or hosted runners — don’t mix untrusted PR code with persistent machines.
