# Release Process (Maintainer-Only)

This document is for the upstream RunnerKit maintainer. End users do not run
any of these steps — see the user-facing [README.md](../README.md#install)
instead.

## One-Time Prerequisites

Before the first `vX.Y.Z` tag can be pushed, two manual steps must be done.
Both are one-time and outside CI.

### 1. Create the Homebrew tap repository

GoReleaser publishes the Cask formula update to a separate repo on every
tag. That repo must exist before the first release.

1. Create a public GitHub repo named `salar/homebrew-runnerkit`.
2. Initialize with a `Casks/` directory (empty file is fine: `Casks/.gitkeep`).
3. The default branch must be `main` (matches `.goreleaser.yaml`
   `homebrew_casks[].repository.branch: main`).

### 2. Create the HOMEBREW_TAP_GITHUB_TOKEN repo secret

The default `GITHUB_TOKEN` issued to a workflow can only push to the workflow's
own repo. Pushing the formula update to `salar/homebrew-runnerkit` requires a
PAT scoped to that repo.

1. On <https://github.com/settings/tokens?type=beta> create a fine-grained personal access token with:
   - Resource owner: `salar`
   - Repository access: only `salar/homebrew-runnerkit`
   - Repository permissions: `Contents: Read and write`
   - Expiration: 1 year (rotate before expiry).
2. In `salar/runnerkit` repo settings → Secrets and variables → Actions, add:
   - Name: `HOMEBREW_TAP_GITHUB_TOKEN`
   - Value: (paste the PAT)

If this secret is missing, the GoReleaser run will fail at the
`homebrew_casks:` step with `403: Resource not accessible by integration`.

## Tag a Release

The release workflow is `.github/workflows/release.yml`. It triggers on a tag
matching `v*` pushed from the upstream repo.

### Pre-tag checklist

Before pushing a tag, the maintainer must:

1. **Run live smokes (D-11):** `make smoke-live` (see Plan 06-04). This
   exercises the BYO permission smoke and the Hetzner end-to-end smoke
   (including the empty-project precheck D-12 gate 1 and the destroy-verify
   D-12 gate 2). The maintainer captures durations into
   `RELEASE-NOTES-vX.Y.Z.md` and `06-VERIFICATION.md` per D-13.
2. **Run the 10-minute stopwatch (D-13):** Follow the stopwatch checklist
   added by Plan 06-04 in this same file. Record wall-clock numbers honestly.
3. **Verify CI green:** Confirm the `pr-checks` workflow passed on the merge
   commit. This proves `goreleaser check` and the snapshot build matrix work.
4. **Confirm the bundled runner pin:** `internal/bootstrap/package.go`
   `RunnerVersion` is a known-good GitHub Actions runner version (currently
   `2.334.0`). Bumping is a separate PR.

### Push the tag

From the upstream repo (NOT a fork — fork tag pushes do not trigger the
upstream workflow, AND fork PRs strip the OIDC `id-token: write` permission
that cosign keyless requires):

```bash
# Example for v1.0.0
git tag -a v1.0.0 -m "RunnerKit v1.0.0"
git push origin v1.0.0
```

The release workflow will:

1. Build all 4 platform binaries (`darwin_arm64`, `darwin_amd64`, `linux_amd64`, `linux_arm64`).
2. Generate `runnerkit_v1.0.0_checksums.txt`.
3. Sign the checksums file with cosign keyless (OIDC) → `runnerkit_v1.0.0_checksums.txt.sigstore.json`.
4. Publish the GitHub Release with all assets.
5. Push the Cask formula update to `salar/homebrew-runnerkit`.

### Post-tag verification

After the workflow completes, verify the release end-to-end as a user would:

```bash
# From a clean directory
TAG=v1.0.0
curl -fsSL -O "https://github.com/salar/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt"
curl -fsSL -O "https://github.com/salar/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt.sigstore.json"

cosign verify-blob \
  --bundle  runnerkit_${TAG#v}_checksums.txt.sigstore.json \
  --certificate-identity   "https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_${TAG#v}_checksums.txt
```

A `Verified OK` confirms the release is signed by the upstream workflow.

## Common Failures

| Failure | Likely cause | Fix |
|---|---|---|
| `signs:` step: `unable to fetch certificate from sigstore` | Workflow ran from a fork PR (OIDC stripped) | Push tag from upstream repo only |
| `homebrew_casks:` step: `403: Resource not accessible by integration` | `HOMEBREW_TAP_GITHUB_TOKEN` missing or scoped wrong | See "One-Time Prerequisites" §2 |
| `goreleaser` `unsupported config version` | `.goreleaser.yaml` missing `version: 2` | Add `version: 2` as the first line |
| User reports "macOS cannot verify that this app is free from malware" | macOS Gatekeeper quarantine on unsigned cask binary | User runs `xattr -d com.apple.quarantine /opt/homebrew/bin/runnerkit` (documented in `docs/troubleshooting/README.md`) |

## Release Notes File (D-13)

Each release ships a `RELEASE-NOTES-vX.Y.Z.md` file in the repo root recording
the maintainer's stopwatch durations from the pre-tag checklist. The first
release file is created in Plan 06-04 (`RELEASE-NOTES-v1.0.0.md`) and
subsequent releases follow the same template.

The `06-VERIFICATION.md` file (created by `/gsd:verify-work` for Phase 6)
holds the v1.0.0 baseline as the reference for future releases.
