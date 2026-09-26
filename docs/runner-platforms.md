# Runner platforms (GitHub Actions labels)

RunnerKit installs a **GitHub Actions self-hosted runner**. Workflows select runners with `runs-on:` — typically **labels** you assign at registration time (RunnerKit sets `runnerkit-<owner>-<repo>` plus defaults).

This page maps **where RunnerKit runs today** versus common **platform combinations** teams target with labels (`ubuntu-latest`-style capacity on your own hardware).

## What RunnerKit supports today

| Runner host (SSH target for `runnerkit up`) | Status |
| --- | --- |
| **Ubuntu x86_64** (systemd; 24.04 is the tested release) | Supported. BYO path and the experimental Hetzner cloud path. Preflight warns when **MemAvailable** is low or swap is absent on small hosts — see [Host resources](troubleshooting/host-resources.md). |
| **Linux arm64** | **Not supported.** Preflight accepts it, but the runner-image setup downloads x86_64-only packages and fails. |
| **Other Linux distributions** (Debian, Fedora, RHEL, Arch, openSUSE, …) | **Not supported.** Preflight can pass, then setup fails or is untested: setup uses Ubuntu package names and Ubuntu-specific package sources. |
| **macOS** | **Not supported.** Preflight rejects non-Linux and non-systemd hosts. |
| **Windows** | **Not supported.** |

The RunnerKit **CLI** itself runs on macOS and Linux (amd64 and arm64); that
is separate from the runner host.

RunnerKit’s remote bootstrap assumes bash over SSH and systemd on the host.

## Windows and macOS runners

GitHub supports Windows and macOS self-hosted runners, but RunnerKit does not
install them. Use GitHub's own runner install instructions or another tool.

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

For public or untrusted code, use GitHub-hosted runners. RunnerKit's
ephemeral mode is not isolation ([safety](safety.md)). The CI sudo grant
above is root-equivalent ([security posture](security-posture.md)).
