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
| Homebrew tap (`brew install accidentally-awesome-labs/runnerkit/runnerkit`) | `brew upgrade runnerkit`                                                                                                                                                                 |
| GitHub Releases binary                                  | Download the latest release, verify the cosign signature and SHA256 checksum, then replace the binary on your `PATH`. See [README install section](../README.md) for the exact commands. |

`runnerkit upgrade` does NOT replace its own binary (per RunnerKit decision
D-07: avoiding self-replace removes a class of partial-failure bugs). It
only prints instructions; you run the printed command yourself.

You can suppress the lazy update notice by setting
`RUNNERKIT_NO_UPDATE_NOTIFIER=1` in your shell environment. The notice is
also silent when `$CI` is set or when running with `--json`.

## 2. Upgrade the bundled GitHub Actions runner pin

RunnerKit bundles a known-good GitHub Actions runner version (currently
`2.334.0`). When that version drifts behind GitHub's deprecation horizon,
`runnerkit doctor` warns:

```
- runner_version_stale (warning)
  Evidence:    installed runner version 2.330.0 is older than bundled pin 2.334.0
  Remediation: runnerkit upgrade-runner --repo owner/name
```

Roll the host runner forward:

```
runnerkit upgrade-runner --repo owner/name --yes
```

This re-applies the runner bootstrap on the saved host using the bundled
pin. It is idempotent — safe to re-run if it fails partway through.

For ephemeral runners (`--mode ephemeral`):

- If the runner is **terminated** (one-shot already completed or TTL
  expired): the upgrade is a no-op. The next
  `runnerkit up --mode ephemeral` will use the bundled pin.
- If the runner is **waiting** or **busy**: the upgrade is refused without
  `--force`. Adding `--force` will drop the registration / kill the
  running job. Use this only when you understand the consequence.

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
# Download the release binary, the checksums file, and the cosign signature.
curl -fLO https://github.com/accidentally-awesome-labs/runnerkit/releases/download/vX.Y.Z/runnerkit_vX.Y.Z_linux_amd64.tar.gz
curl -fLO https://github.com/accidentally-awesome-labs/runnerkit/releases/download/vX.Y.Z/runnerkit_vX.Y.Z_checksums.txt
curl -fLO https://github.com/accidentally-awesome-labs/runnerkit/releases/download/vX.Y.Z/runnerkit_vX.Y.Z_checksums.txt.sig
curl -fLO https://github.com/accidentally-awesome-labs/runnerkit/releases/download/vX.Y.Z/runnerkit_vX.Y.Z_checksums.txt.pem

# Verify SHA256 checksum.
sha256sum -c runnerkit_vX.Y.Z_checksums.txt --ignore-missing

# Verify cosign keyless signature (requires cosign installed).
cosign verify-blob \
  --certificate runnerkit_vX.Y.Z_checksums.txt.pem \
  --signature runnerkit_vX.Y.Z_checksums.txt.sig \
  --certificate-identity-regexp "https://github.com/accidentally-awesome-labs/runnerkit" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  runnerkit_vX.Y.Z_checksums.txt
```

If `sha256sum -c` or `cosign verify-blob` fails, do NOT install the
binary; report the discrepancy.
