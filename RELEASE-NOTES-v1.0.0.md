# RunnerKit v1.0.0

First public release.

## What This Is

RunnerKit is a CLI-first tool that helps solo developers create and manage
self-hosted GitHub Actions runners without becoming infrastructure operators.

## Bundled Versions

- GitHub Actions runner pin: **2.334.0**
- Built with Go 1.22
- Released by GoReleaser v2.15.4 + cosign v3.0.6 (keyless OIDC signature on
  `runnerkit_v1.0.0_checksums.txt`)

## Supported CLI Host Platforms

- macOS arm64
- macOS amd64
- Linux amd64
- Linux arm64

Linux 386 and 32-bit ARM are NOT supported.

## Install

See [README.md](README.md#install) for the full install matrix.

```bash
# Homebrew
brew tap accidentally-awesome-labs/tap
brew install --cask runnerkit

# Or download from GitHub Releases and verify with cosign:
TAG=v1.0.0
cosign verify-blob \
  --bundle  runnerkit_${TAG#v}_checksums.txt.sigstore.json \
  --certificate-identity   "https://github.com/accidentally-awesome-labs/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_${TAG#v}_checksums.txt
```

## 10-Minute Stopwatch (D-13)

Measured by the maintainer on a clean machine before tagging.

| Path                     | Wall-clock | Notes                          |
| ------------------------ | ---------- | ------------------------------ |
| BYO persistent           | `126 s`    | Runner ID: `35`                |
| Hetzner cloud persistent | `167 s`    | Hetzner cost: `0.00 EUR` (est. from plan rate; replace with dashboard exact value if needed) |

## Outstanding Live Smokes Closed

- **Phase 1: live GitHub permission smoke** — closed by `make smoke-live-byo`.
  STATE.md note (Plan 01-02/01-04 validation) resolved.
- **Phase 4: live Hetzner billable smoke** — closed by `make smoke-live-cloud`
  with D-12 gate 1 (empty-project precheck) and D-12 gate 2 (destroy-verify
  404 polling within 300s timeout). STATE.md note resolved.

## Upgrade Path

This is the first release; nothing to upgrade from. Future releases follow
[docs/upgrade.md](docs/upgrade.md):

- CLI: `runnerkit upgrade` prints the right command for your install channel
  (Homebrew or Releases binary). Does NOT self-replace.
- Bundled runner: `runnerkit upgrade-runner` re-applies bootstrap with the new
  pin. Refuses without `--force` when an ephemeral runner is currently
  waiting/busy.
- State: forward-only auto migrations with side-by-side backup
  (`state.json.backup-vN-<timestamp>`). Refuses to mutate on newer schema with
  exit code 7.

## Troubleshooting

If the CLI prints a `RKD-<COMPONENT>-NNN` code, see
[docs/troubleshooting/](docs/troubleshooting/README.md). Override the URL
prefix with `RUNNERKIT_DOCS_BASE`.

## Acknowledgements

Built end-to-end through the GSD planning + execution workflow.
