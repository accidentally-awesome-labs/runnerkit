# Troubleshooting

Stuck? Find your `RKD-<COMPONENT>-NNN` code in the table below and follow the
component link. Every entry follows a `Symptom → Diagnosis → Fix` structure
(D-17).

If a `runnerkit` command printed a `See: <URL>` line, the URL points at the
exact entry below.

## Install verification

If `cosign verify-blob` fails or `sha256sum -c` reports a mismatch, the
downloaded archive is NOT the upstream release. Do NOT install it.

```bash
TAG=v1.0.0
cosign verify-blob \
  --bundle  runnerkit_${TAG#v}_checksums.txt.sigstore.json \
  --certificate-identity   "https://github.com/accidentally-awesome-labs/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_${TAG#v}_checksums.txt
```

If you installed via Homebrew on macOS and see "macOS cannot verify that this
app is free from malware":

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/runnerkit  # Apple Silicon
xattr -d com.apple.quarantine /usr/local/bin/runnerkit     # Intel
```

(RunnerKit binaries are not Apple-notarized in v1; this is a known limitation.)

## Components

| File                         | Codes                                       | Failures covered                                                                                                          |
| ---------------------------- | ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| [auth.md](auth.md)           | RKD-AUTH-NNN                                | GitHub auth scope, public-repo block, ephemeral BYO acknowledgment, network access to github.com                          |
| [ssh.md](ssh.md)             | RKD-SSH-NNN                                 | host-key mismatch, host unreachable, key-path-not-found, port unreachable                                                 |
| [bootstrap.md](bootstrap.md) | RKD-BOOT-NNN                                | systemd service, install/work paths, preflight (disk, **RAM/swap**, tools, time, network), runner user/package install, online-verification timeout, runner version stale, journal OOM hints (018) |
| [github.md](github.md)       | RKD-GH-NNN                                  | runner offline, duplicate candidates, label drift, registration/deregister/recover failures, self-hosted CI sudo vs hosted |
| [provider.md](provider.md)   | RKD-PROV-NNN                                | Hetzner token/quota/region, partial destroy, billable lingering                                                           |
| [cleanup.md](cleanup.md)     | RKD-CLEAN-NNN, RKD-STATE-NNN, RKD-CORE-NNN  | down/destroy partial, ephemeral log preservation, state JSON read, schema-too-new, migration failure, CLI input          |
| [host-resources.md](host-resources.md) | (narrative; codes RKD-BOOT-016..018 live in [bootstrap.md](bootstrap.md)) | RAM/swap preflight, OOM-heavy CI, journal hints, sizing and parallelism |
| [doctor-ux.md](doctor-ux.md) | (CLI UX) | First-run wizard, `--explain`, BYO progress checklists, doctor ignore config and fix mode |

> Note on numbering: codes are stable across renames; numbering grows
> monotonically per component. Some numbers may be reserved (e.g.,
> `RKD-BOOT-001` is reserved for future use). This is by design.

## Custom docs hosting

If your team hosts a fork of these docs (e.g., on a static site), set
`RUNNERKIT_DOCS_BASE` to override the URL prefix the CLI emits:

```bash
export RUNNERKIT_DOCS_BASE=https://my-docs.example.com/runnerkit
```

The CLI will then print `<my-docs>/troubleshooting/<component>#<anchor>` for
every code.
