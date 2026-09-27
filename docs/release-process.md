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

1. **Verify CI green on the release commit.** `pr-checks` must pass on the
   merge commit (tests, `goreleaser check`, snapshot build, `go generate`
   drift check). The release workflow repeats `go vet`, the `go generate`
   drift check and `go test -race` before GoReleaser; a failure there means
   no release. For local verification it is acceptable to run
   `goreleaser release --snapshot --skip=publish --clean --skip=sign`; tag
   releases in upstream CI must keep signing enabled.
2. **Run `make generate-check` on a clean checkout** of the release commit.
   It fails if `go generate ./...` changes `install.sh` (whose sudoers block
   is generated from `bootstrap.RenderSudoersEntry`) or leaves untracked
   files.
3. **Read the govulncheck result for the release commit.** Run the
   `pr-checks` workflow by hand (`workflow_dispatch`) on the release SHA, or
   use the run from the push to `main`, and read the `govulncheck` job
   summary (the job is report-only: it stays green and raises a warning
   annotation when there are findings). CI builds with the latest Go 1.26.x
   patch release, so the result is expected to show **no reachable
   findings**. A reachable standard-library finding means a newer Go
   patch release is needed (re-run the job so `setup-go` picks it up, or
   raise the Go version if the fix is only on a newer line, and raise the
   `toolchain` line in `go.mod` to that patch so source builds with
   `GOTOOLCHAIN=auto` get it too); a
   **reachable** finding in a module dependency (`golang.org/x/*` or any
   other non-stdlib module) means that module bump goes into this release.
   Anything knowingly shipped goes under Known issues in the CHANGELOG.
   Link the run in the CHANGELOG section.
4. **Real GitHub job on a fresh password-sudo host (BYO gate).** No release
   may claim the BYO path works unless, for that commit, a real job ran on a
   fresh password-sudo Ubuntu 24.04 x86_64 host prepared **only** by that
   commit's `install.sh`:
   1. prepare the host (a container with systemd, openssh-server and a sudo
      user **with a password** is enough) with the candidate `install.sh`;
   2. `runnerkit up --host user@<host> --repo <throwaway private repo>`;
   3. dispatch a workflow on the runner's labels that runs `gcc hello.c` and
      `docker run hello-world`.

   Pass: the runner is online, the job is green, and
   `id -nG runnerkit-runner` contains `docker`. Record the run URL in the
   CHANGELOG. The runbook is [testkit/release-gate.md](testkit/release-gate.md):
   `scripts/testkit/gate.sh` dispatches the job, checks each criterion
   (including that the host's passwordless sudo is exactly what the
   candidate `install.sh` writes, and that the runner took the job at
   RunnerKit's pinned version without updating itself) and writes the
   evidence with the CHANGELOG line. **If it does not pass** (and cannot be fixed within the time
   box), do not tag with a BYO claim: ship the fallback that makes BYO
   `up`/`register` refuse without `--accept-known-issues`, and change the
   README and CHANGELOG Known-issues text to "BYO is not supported in this
   release".

   On the same host, try the revocation steps in
   [security-posture.md](security-posture.md#if-you-already-installed-runnerkit)
   before publishing them. After step 2, confirm that the runner user can
   no longer replace `svc.sh` or `bin`:
   `sudo -u runnerkit-runner mv <install>/svc.sh <install>/svc.sh.orig` and
   the same for `bin` must fail with "Permission denied". Until that has
   been checked on a runner host, the revocation gate is not met.
   `scripts/testkit/gate.sh --after-revocation` then checks from a real job
   that Docker and the install directory are refused and the SSH user has
   no passwordless sudo ([part 3 of the runbook](testkit/release-gate.md#part-3-revocation-drill)).
5. **Verify the Homebrew tap token.** `HOMEBREW_TAP_GITHUB_TOKEN` must be a
   fine-grained PAT with `Contents: Read and write` on
   `accidentally-awesome-labs/homebrew-tap` only, not expired (check the
   expiry in GitHub settings), and SSO-authorized if required. Rotate it if
   it was ever exposed. See [One-Time Prerequisites §2](#2-create-the-homebrew_tap_github_token-repo-secret).
   If the release machinery has been idle for a long time, also expect to
   re-check cosign keyless (OIDC) signing on the first run.
6. **Manual refusal check (no Hetzner spend).** With a scratch
   `RUNNERKIT_STATE_DIR`, seeded state and a throwaway working directory,
   confirm with the built binary that: `down` on cloud state is refused;
   `up --replace` on cloud state is refused; `upgrade-runner` and
   `doctor --fix` are refused; `--mode ephemeral` with `--cloud hetzner` is refused
   even with `--experimental`; `--cloud` without `--experimental`, and
   without `--cloud-region`, is refused before any network call; nothing is
   written to the working directory. Optionally, a `--dry-run` cloud plan
   with a real **read-only** Hetzner token shows the API price and its label
   (it creates nothing).
7. **Docs and notes.** The CHANGELOG section lists every shipped change, has
   a Known-issues block that matches the README "Known issues" section, and
   says BYO works on password-sudo hosts **only** if step 4 passed. LICENSE,
   the README banner and `docs/security-posture.md` are merged.
8. **Optional live smokes (maintainer-only).** `make smoke-live` runs the
   BYO permission smoke and the Hetzner end-to-end smoke (empty-project
   precheck and destroy-verify). The cloud leg creates billable resources
   and needs `RUNNERKIT_SMOKE_CLOUD_REGION`. Both run
   `scripts/smoke/assert-doctor-json-contract.sh` (`doctor --json` has
   `schema_version`, `stage`, and `host_incident_hints`/`next_actions` as
   arrays; `doctor --deep --json` succeeds; set
   `RUNNERKIT_SMOKE_SKIP_DOCTOR_DEEP=1` to skip the deep pass) and
   `scripts/smoke/assert-list-json-contract.sh`. Requires `python3`. For the
   optional BYO multi-repo leg set `RUNNERKIT_SMOKE_MULTI_REPO=1` and
   `RUNNERKIT_SMOKE_REPO2`. These smokes do not run a workflow job, so they
   do not replace step 4.
9. **Confirm the bundled runner pin.** `internal/bootstrap/package.go`
   `RunnerVersion` (2.337.0 in v1.3.4) and its SHA-256s match the
   `actions/runner` release page. RunnerKit must never pass
   `--disableupdate`; `TestRenderedScriptsNeverDisableUpdate` enforces it.

If the tag cannot ship by its planned date, deprecate the Homebrew cask
(`deprecate!` pointing at the README "Known issues" section) instead of
leaving the broken release as the recommended install.

### Push the tag

From the upstream repo (NOT a fork — fork tag pushes do not trigger the
upstream workflow, AND fork PRs strip the OIDC `id-token: write` permission
that cosign keyless requires):

```bash
git tag -a vX.Y.Z -m "RunnerKit vX.Y.Z — short summary"
git push origin vX.Y.Z
```

The release workflow will:

1. Run `go vet`, the `go generate` drift check and `go test -race` (job
   `test`); GoReleaser runs only if they pass, and only in the upstream
   repository.
2. Build all 4 platform binaries (`darwin_arm64`, `darwin_amd64`, `linux_amd64`, `linux_arm64`).
3. Generate `runnerkit_X.Y.Z_checksums.txt`.
4. Sign the checksums file with cosign keyless (OIDC) → `runnerkit_X.Y.Z_checksums.txt.sigstore.json`.
5. Publish the GitHub Release with all assets.
6. Push the Cask formula update to `accidentally-awesome-labs/homebrew-tap`.

### Post-tag verification

After the workflow completes, verify the release end-to-end as a user would:

```bash
# From a clean directory
TAG=vX.Y.Z
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
| `test` job fails (vet, `go generate` drift, or a test) | The tagged commit is not releasable; GoReleaser does not run | Fix on `main`, delete the tag (`git push origin :refs/tags/vX.Y.Z`), and tag the fixed commit (or use the next patch version) |
| `signs:` step: `unable to fetch certificate from sigstore` | Workflow ran from a fork PR (OIDC stripped) | Push tag from upstream repo only |
| `homebrew_casks:` step: `403` / `401` | `HOMEBREW_TAP_GITHUB_TOKEN` missing, PAT not SSO-authorized, or scoped wrong | See "One-Time Prerequisites" §2 |
| `notarize.macos:` step fails (`Unauthorized`, `Invalid credentials`, timeout) | Apple notary secrets missing/invalid | Verify the 5 `MACOS_*` secrets from "One-Time Prerequisites" §3 |
| `goreleaser` `unsupported config version` | `.goreleaser.yaml` missing `version: 2` | Add `version: 2` as the first line |
| User reports "macOS cannot verify that this app is free from malware" | macOS Gatekeeper quarantine on unsigned cask binary | User runs `xattr -d com.apple.quarantine /opt/homebrew/bin/runnerkit` (documented in `docs/troubleshooting/README.md`) |

## Release notes

From v1.3.4, release notes live in [CHANGELOG.md](../CHANGELOG.md)
(Keep a Changelog). Move the `Unreleased` section under the new version
heading when you tag, and paste it into the GitHub Release body. The older
`RELEASE-NOTES-v*.md` files in the repository root are historical and are
summarized in the CHANGELOG.

## Setup timing (optional)

`make smoke-stopwatch` points here. To time a BYO and a cloud setup end to
end on a clean machine:

1. Start a fresh Ubuntu 24.04 x86_64 host (BYO) and run the current
   `install.sh` on it. Start the stopwatch.
2. Run `runnerkit up --repo owner/name --host user@host --yes` and note the
   time until the runner is reported online.
3. Trigger a workflow that uses the printed `runs-on:` labels and note the
   time until the job finishes. Stop the stopwatch.
4. For cloud, repeat from step 2 with
   `runnerkit up --repo owner/name --experimental --cloud hetzner --cloud-region <location> --yes`,
   then run `runnerkit destroy --repo owner/name` and confirm it verifies
   deletion.
5. Record the durations in your own notes. Do not publish them as a
   setup-time promise.
