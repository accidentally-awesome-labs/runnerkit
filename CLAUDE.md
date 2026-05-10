# RunnerKit — maintainer and agent notes

## Shipping changes to end users

**Merging to `main` does not update Homebrew or GitHub Releases.** Install docs point users at those channels; they advance only when a **`v*`** tag is pushed on the **upstream** repo (`accidentally-awesome-labs/runnerkit`), which triggers `.github/workflows/release.yml` (GoReleaser).

**Rough sequence**

1. Land work on `main` (PR merge or direct push).
2. Run the **pre-tag checklist** in [`docs/release-process.md`](docs/release-process.md) (CI green, smoke/stopwatch expectations as applicable).
3. Choose the next **SemVer** tag (`v1.0.x` patch for fixes/small additive CLI; bump minor/major when warranted).
4. Create an **annotated** tag and push **only the tag** (or push tag after verifying commit):

   ```bash
   git fetch origin && git checkout main && git pull origin main
   git tag -a vX.Y.Z -m "RunnerKit vX.Y.Z — short summary"
   git push origin vX.Y.Z
   ```

5. Confirm in GitHub **Actions** that the release workflow succeeded.
6. Confirm **GitHub Releases** has assets for `vX.Y.Z` and **`accidentally-awesome-labs/homebrew-tap`** received the cask bump (GoReleaser commit, e.g. `runnerkit: bump cask to vX.Y.Z`).

**Fork caveat:** Tag pushes from forks do not run upstream releases and may break OIDC signing — always release from the upstream repository.

Full prerequisites (Homebrew PAT, optional Apple notarization), failure modes, and verification commands: **`docs/release-process.md`**.
