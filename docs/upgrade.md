# Upgrading RunnerKit

This guide covers three independent upgrade flows.

## 1. Upgrade the RunnerKit CLI

When `runnerkit up`, `runnerkit status`, or `runnerkit doctor` prints
`runnerkit X.Y.Z available`, run:

```
runnerkit upgrade
```

This prints the right command for your install channel.

| Install method                                          | Upgrade command                                                                                                                                                                          |
| ------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Homebrew tap (`brew tap accidentally-awesome-labs/tap && brew install --cask runnerkit`) | `brew upgrade --cask runnerkit`                                                                                                                                                          |
| GitHub Releases binary                                  | Download the latest release, verify the cosign signature and SHA256 checksum, then replace the binary on your `PATH`. See [README install section](../README.md) for the exact commands. |

`runnerkit upgrade` does NOT replace its own binary (per RunnerKit decision
D-07: avoiding self-replace removes a class of partial-failure bugs). It
only prints instructions; you run the printed command yourself.

You can suppress the lazy update notice by setting
`RUNNERKIT_NO_UPDATE_NOTIFIER=1` in your shell environment. The notice is
also silent when `$CI` is set or when running with `--json`.

## 2. Keep the GitHub Actions runner up to date

RunnerKit installs a pinned GitHub Actions runner version (currently
`2.337.0`, `bootstrap.RunnerVersion`) when it registers a runner. After that
the runner **updates itself**: RunnerKit never passes `--disableupdate` to
`config.sh`, so GitHub's runner downloads and installs newer versions when
GitHub requires them. You do not need a RunnerKit command for this.

`runnerkit doctor` may report `runner_version_stale` when the version saved
in RunnerKit state is older than the bundled pin. That finding compares
local state, not the host, so it usually needs no action.

**`runnerkit upgrade-runner` is disabled in v1.3.4** (`command_disabled`,
exit 2), and so is `runnerkit doctor --fix`, which called it. In v1.3.3 and
earlier, `upgrade-runner` deleted the runner's `.runner` and `.credentials`
files and ran `config.sh` with an empty registration token, which unregisters
a healthy runner, and it never replaced the runner binaries. It was not safe
to re-run. Do not use it with an older RunnerKit binary either.

To reinstall a runner with the bundled pin anyway, re-register it by hand:

```
runnerkit down --repo owner/name --dry-run
runnerkit down --repo owner/name
runnerkit up --repo owner/name --host user@host
```

For a RunnerKit-created Hetzner server, use `runnerkit destroy --repo
owner/name` instead of `down`, then `runnerkit up` again with your original
flags. Saved extra packages are not carried over automatically; pass
`--extra-packages` again if you need them.

## 3. State migrations

State migrations are forward-only and automatic. When you upgrade RunnerKit
to a release that bumps `schema_version` (e.g., from `"1"` to `"2"`), the
next CLI invocation that reads state will:

1. Write a side-by-side backup at
   `~/.local/state/runnerkit/state.json.backup-v<old>-<RFC3339>` (e.g.,
   `state.json.backup-v1-20260615T143000Z`). The backup contains your
   original state file byte-for-byte.
2. Migrate the in-memory state forward.
3. Save the migrated state via the same atomic-write mechanism used for
   all state mutations.

If you DOWNGRADE RunnerKit and the older binary encounters a `state.json`
with a `schema_version` newer than it knows, the older binary refuses to
mutate and exits with code `7` (`ExitStateSchemaTooNew`). The error message
tells you to run `runnerkit upgrade`. Your state file is untouched.

If something goes wrong during a migration, the side-by-side backup file
contains your original state byte-for-byte; you can restore it with:

```
cp ~/.local/state/runnerkit/state.json.backup-v1-<timestamp> ~/.local/state/runnerkit/state.json
```

Note: this will re-trigger the migration on the next CLI invocation. If
you need to stay on the older format, downgrade RunnerKit too.

## Verifying release artifacts

Before installing a downloaded release binary, verify integrity:

```
# Replace vX.Y.Z with the release tag you chose from the Releases page.
TAG=vX.Y.Z
OS=linux
ARCH=amd64

# Download the release archive, checksums, and sigstore bundle.
curl -fLO "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_${OS}_${ARCH}.tar.gz"
curl -fLO "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt"
curl -fLO "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt.sigstore.json"

# Verify SHA256 checksum.
sha256sum -c "runnerkit_${TAG#v}_checksums.txt" --ignore-missing

# Verify cosign keyless signature (requires cosign installed).
cosign verify-blob \
  --bundle "runnerkit_${TAG#v}_checksums.txt.sigstore.json" \
  --certificate-identity "https://github.com/accidentally-awesome-labs/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  "runnerkit_${TAG#v}_checksums.txt"
```

If `sha256sum -c` or `cosign verify-blob` fails, do NOT install the
binary; report the discrepancy.
