# Release Process (Maintainer-Only)

This document is for the upstream RunnerKit maintainer. End users do not run
any of these steps — see the user-facing [README.md](../README.md#install)
instead.

## Shipping changes through user channels

Code merged to `main` is the source tree, but **installable artifacts only advance when you push a version tag.** Until then, users who install via **Homebrew** or **GitHub Releases** stay on the **previous** tagged release.

| Channel | What updates it |
| --- | --- |
| **GitHub Releases** (`releases/download/vX.Y.Z/…`) | Annotated tag `vX.Y.Z` pushed to **upstream** `runnerkit` → release workflow |
| **Homebrew cask** (`accidentally-awesome-labs/homebrew-tap`, `brew install --cask runnerkit`) | Same workflow — GoReleaser commits `Casks/runnerkit.rb` (needs `HOMEBREW_TAP_GITHUB_TOKEN`) |
| **Docs on `main`** | Live on GitHub immediately; they may describe behavior **newer** than the latest tag until you release |

**Version tags:** Use SemVer `vMAJOR.MINOR.PATCH` aligned with user-visible impact (patch for fixes and small additive CLI; minor/major when behavior or compatibility warrants it). The tag message should summarize what shipped.

**Operational flow:** merge to `main` → pre-tag checklist below → `git tag -a vX.Y.Z -m "…"` → `git push origin vX.Y.Z` → watch Actions → verify Release assets and tap bump.

**Agents / automation:** Do not assume `main` equals “released.” For a concise checklist, see [CLAUDE.md](../CLAUDE.md) in the repo root.

## One-Time Prerequisites

Before the first `vX.Y.Z` tag can be pushed, manual setup steps must be done.
Both are one-time and outside CI.

### 1. Create the Homebrew tap repository

GoReleaser publishes the Cask formula update to a separate repo on every
tag. That repo must exist before the first release.

1. Create a public GitHub repo named `accidentally-awesome-labs/homebrew-tap`.
2. Initialize with a `Casks/` directory (empty file is fine: `Casks/.gitkeep`).
3. The default branch must be `main` (matches `.goreleaser.yaml`
   `homebrew_casks[].repository.branch: main`).

### 2. Create the `HOMEBREW_TAP_GITHUB_TOKEN` repo secret

The default `GITHUB_TOKEN` issued to a workflow can only push to the workflow's
own repo. Pushing the formula update to `accidentally-awesome-labs/homebrew-tap` requires a
PAT scoped to that repo.

1. On <https://github.com/settings/tokens?type=beta> create a fine-grained personal access token with:
   - Resource owner: `accidentally-awesome-labs`
   - Repository access: only `accidentally-awesome-labs/homebrew-tap`
   - Repository permissions: `Contents: Read and write`
   - Expiration: 1 year (rotate before expiry).
   - If the org uses SAML SSO: authorize the token for `accidentally-awesome-labs`.
   - You can use any **token name / note** in the UI (for example
     `RUNNERKIT_HOMEBREW_TAP_REPO_ACCESS_TOKEN`) so you remember what it is for;
     that label is only for you and is unrelated to Actions.
2. In `accidentally-awesome-labs/runnerkit` repo settings → Secrets and variables → Actions, add:
   - Name: **`HOMEBREW_TAP_GITHUB_TOKEN`** (this name must match the workflow)
   - Value: (paste the PAT from step 1)

GoReleaser reads that value from the environment variable
`HOMEBREW_TAP_GITHUB_TOKEN` (see `.goreleaser.yaml`).

If this secret is missing or invalid, the GoReleaser run will fail at the
`homebrew_casks:` step with `403: Resource not accessible by integration` or
`401 Bad credentials`.

### 3. (Optional, recommended) Configure OSS notarization secrets

RunnerKit uses OSS GoReleaser's cross-platform `notarize.macos` flow. The
notarization block is enabled only when all secrets below are present.

1. Create Apple credentials:
   - `MACOS_SIGN_P12`: base64 of your Developer ID Application `.p12`
   - `MACOS_SIGN_PASSWORD`: password for that `.p12`
   - `MACOS_NOTARY_KEY`: base64 of your App Store Connect API `.p8`
   - `MACOS_NOTARY_KEY_ID`: App Store Connect key ID
   - `MACOS_NOTARY_ISSUER_ID`: App Store Connect issuer UUID
2. Add those 5 values as repository Actions secrets in
   `accidentally-awesome-labs/runnerkit`.
3. Keep `MACOS_SIGN_P12` and `MACOS_NOTARY_KEY` as single-line base64 values.

When omitted, releases still work, but macOS users may need the quarantine
workaround from `docs/troubleshooting/README.md`.

## Tag a Release

The release workflow is `.github/workflows/release.yml`. It triggers on a tag
matching `v*` pushed from the upstream repo.

### Pre-tag checklist

Before pushing a tag, the maintainer must:

1. **Run live smokes (D-11):** `make smoke-live` (see Plan 06-04). This
   exercises the BYO permission smoke and the Hetzner end-to-end smoke
   (including the empty-project precheck D-12 gate 1 and the destroy-verify
   D-12 gate 2).    Both paths run **`scripts/smoke/assert-doctor-json-contract.sh`**
   after the interactive `runnerkit doctor` step to assert the **`doctor --json`**
   envelope includes **`schema_version`**, **`stage`**, **`host_incident_hints`** and **`next_actions`** as JSON **arrays**
   (empty `[]` when there is nothing to report — never `null`) and that **`doctor --deep --json`**
   succeeds (Phase 7 host-capacity / journal heuristics plumbing). Requires **`python3`**.
   Set **`RUNNERKIT_SMOKE_SKIP_DOCTOR_DEEP=1`** to skip the `--deep` pass if SSH
   journal collection is temporarily unavailable. The maintainer captures durations into
   `RELEASE-NOTES-vX.Y.Z.md` and `06-VERIFICATION.md` per D-13.
2. **Run the 10-minute stopwatch (D-13):** Follow the stopwatch checklist
   added by Plan 06-04 in this same file. Record wall-clock numbers honestly.
3. **Verify CI green:** Confirm the `pr-checks` workflow passed on the merge
   commit. This proves `goreleaser check` and the snapshot build matrix work.
   For local, non-interactive verification, it is acceptable to run
   `goreleaser release --snapshot --skip=publish --clean --skip=sign` when
   keyless cosign device flow is not available. Tag releases in upstream CI
   MUST keep signing enabled (no `--skip=sign`).
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
5. Push the Cask formula update to `accidentally-awesome-labs/homebrew-tap`.

### Post-tag verification

After the workflow completes, verify the release end-to-end as a user would:

```bash
# From a clean directory
TAG=v1.0.0
curl -fsSL -O "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt"
curl -fsSL -O "https://github.com/accidentally-awesome-labs/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt.sigstore.json"

cosign verify-blob \
  --bundle  runnerkit_${TAG#v}_checksums.txt.sigstore.json \
  --certificate-identity   "https://github.com/accidentally-awesome-labs/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_${TAG#v}_checksums.txt
```

A `Verified OK` confirms the release is signed by the upstream workflow.

## Common Failures

| Failure | Likely cause | Fix |
|---|---|---|
| `signs:` step: `unable to fetch certificate from sigstore` | Workflow ran from a fork PR (OIDC stripped) | Push tag from upstream repo only |
| `homebrew_casks:` step: `403` / `401` | `HOMEBREW_TAP_GITHUB_TOKEN` missing, PAT not SSO-authorized, or scoped wrong | See "One-Time Prerequisites" §2 |
| `notarize.macos:` step fails (`Unauthorized`, `Invalid credentials`, timeout) | Apple notary secrets missing/invalid | Verify the 5 `MACOS_*` secrets from "One-Time Prerequisites" §3 |
| `goreleaser` `unsupported config version` | `.goreleaser.yaml` missing `version: 2` | Add `version: 2` as the first line |
| User reports "macOS cannot verify that this app is free from malware" | macOS Gatekeeper quarantine on unsigned cask binary | User runs `xattr -d com.apple.quarantine /opt/homebrew/bin/runnerkit` (documented in `docs/troubleshooting/README.md`) |

## Release Notes File (D-13)

Each release ships a `RELEASE-NOTES-vX.Y.Z.md` file in the repo root recording
the maintainer's stopwatch durations from the pre-tag checklist. The first
release file is created in Plan 06-04 (`RELEASE-NOTES-v1.0.0.md`) and
subsequent releases follow the same template.

The `06-VERIFICATION.md` file (created by `/gsd:verify-work` for Phase 6)
holds the v1.0.0 baseline as the reference for future releases.

## Stopwatch Checklist (D-13)

This is the 10-minute reliable-runner promise from PROJECT.md Core Value.
Run this on a CLEAN machine (fresh laptop, fresh VM, clean
`$HOME/.local/state/runnerkit/`) before tagging each release. The
maintainer's wall-clock numbers go into `RELEASE-NOTES-vX.Y.Z.md`.

### BYO path (target: ≤ 10 minutes)

| Step | Description                                                       | T0  | T_now | Δ   |
| ---- | ----------------------------------------------------------------- | --- | ----- | --- |
| 1    | `gh auth login` (if not already authed)                           |     |       |     |
| 2    | `runnerkit up --repo $REPO --host user@host --mode persistent`    |     |       |     |
| 3    | Trigger a workflow targeting the `runnerkit-...` label            |     |       |     |
| 4    | Observe job runs on the new runner                                |     |       |     |
| 5    | `runnerkit down --repo $REPO --yes`                               |     |       |     |

Total wall-clock: __ minutes __ seconds.

### Hetzner cloud path (target: ≤ 10 minutes)

| Step | Description                                                       | T0  | T_now | Δ   |
| ---- | ----------------------------------------------------------------- | --- | ----- | --- |
| 1    | `gh auth login` (if not already authed)                           |     |       |     |
| 2    | `export HCLOUD_TOKEN=...` (one-time)                              |     |       |     |
| 3    | `runnerkit up --repo $REPO --cloud hetzner --mode persistent`     |     |       |     |
| 4    | Trigger a workflow targeting the `runnerkit-...` label            |     |       |     |
| 5    | Observe job runs on the new runner                                |     |       |     |
| 6    | `runnerkit destroy --repo $REPO --yes`                            |     |       |     |
| 7    | Verify Hetzner Console shows 0 `runnerkit-*` resources            |     |       |     |

Total wall-clock: __ minutes __ seconds.
Hetzner cost (from project billing dashboard): __ EUR.

### Recording

After running both paths, copy the totals into:

1. `RELEASE-NOTES-v$VERSION.md` (per-release file, committed at tag time).
2. `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md`
   for the v1.0.0 baseline (ONE-TIME — overwritten only if the baseline
   methodology changes).

If either path exceeds 10 minutes, do NOT tag the release. Investigate the
slow step, fix it, and re-run the stopwatch.
