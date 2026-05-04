---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 01
subsystem: release-packaging
tags: [goreleaser, cosign-keyless, sigstore-bundle, homebrew-cask, github-actions-release, oidc, install-matrix]

requires:
  - phase: 02-byo-persistent-runner-happy-path
    provides: bootstrap.RunnerVersion 2.334.0 pin (consumed by docs/release-process.md pre-tag checklist)
  - phase: 06-release-upgrade-docs-and-v1-validation/03
    provides: docs/troubleshooting/README.md (forward-linked from README.md install verification section)
provides:
  - .goreleaser.yaml v2 schema producing 4-platform binaries + checksums.txt + cosign sigstore bundle + Homebrew Cask formula update.
  - .github/workflows/release.yml — tag-triggered (v*) GoReleaser pipeline with id-token: write OIDC for cosign keyless signing.
  - .github/workflows/pr-checks.yml — every-PR validation running goreleaser check, snapshot build with 4-arch dist assertion, and full go test ./... -count=1 -race.
  - README.md ## Install section with Homebrew tap command, 4-platform asset matrix, GitHub Releases download flow, literal cosign verify-blob snippet (D-05), and forward link to docs/troubleshooting/.
  - docs/release-process.md — maintainer-only one-time prerequisites (tap repo + PAT secret), pre-tag checklist (forward refs to make smoke-live in Plan 06-04), tag procedure, post-tag verify, common-failures table.
  - dist/ gitignored.
  - Locked OIDC issuer: https://token.actions.githubusercontent.com (NOT the user-OAuth issuer https://github.com/login/oauth).
  - Locked cert-identity URL pattern: https://github.com/accidentally-awesome-labs/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}.
affects: [06-04-v1-validation-and-live-smoke]

tech-stack:
  added:
    - GoReleaser v2 (action v7, runtime constrained via "version: '~> v2'")
    - sigstore/cosign-installer@v3 (cosign-release: v3.0.6)
    - Homebrew tap accidentally-awesome-labs/homebrew-runnerkit (external repo; created during Task 5 closure with Casks/.gitkeep, default branch main, public)
  patterns:
    - "Single-tag release pipeline: pushing vX.Y.Z from upstream produces 4 binaries + checksums.txt + sigstore bundle + GitHub Release + Cask formula update in one workflow run. No local cuts; no per-platform CI matrix."
    - "Cosign keyless signing via GitHub Actions OIDC: signs[].artifacts: checksum signs ONLY checksums.txt (not each archive) per D-04; bundle output (.sigstore.json) over deprecated v1/v2 .sig+.crt format."
    - "Tag-only trigger (on: push: tags: ['v*']) with no pull_request/workflow_dispatch: sidesteps fork-PR OIDC strip (Pitfall 1). PR validation runs the snapshot path on a separate workflow with no signing."
    - "Install verification matches the workflow path: README's --certificate-identity URL string-equals the workflow file path embedded in cosign keyless cert. Any rename of release.yml requires updating README."
    - "Per-task atomic commits: Task 1 (.goreleaser.yaml + .gitignore), Task 2 (workflows), Task 3 (README install), Task 4 (release-process.md). Each commit verifiable in isolation."

key-files:
  created:
    - .goreleaser.yaml
    - .github/workflows/release.yml
    - .github/workflows/pr-checks.yml
    - .gitignore
    - docs/release-process.md
  modified:
    - README.md (## Install section inserted between intro and ## Safety section)

key-decisions:
  - "Inserted ## Install section AFTER the intro paragraph and BEFORE ## Safety, not at the bottom of README. Rationale: Install is the first thing a new user does; Safety/quickstarts come after they have the binary. Preserves Plan 06-03's ## Troubleshooting cross-link (which sits between cloud-quickstart and BYO operations sections) untouched."
  - "Removed Windows mention from the platform-exclusion list. The plan body said 'Windows, Linux 386, and 32-bit ARM are not supported.' but the plan's own automated grep enforces `! grep -q windows`. Resolved by listing only 'Linux 386 and 32-bit ARM' as not supported and relying on the explicit 4-row platform table to communicate that Windows is not a supported target. Net behavior matches D-02 (no Windows support); literal text adjusted to satisfy the grep contract."
  - "Removed 'macOS' from the not-supported list. (Same grep-contract reasoning — the explicit table lists macOS arm64/amd64 as supported, so the negative sentence does not need to repeat 'macOS'.)"
  - "Task 5 (human-action checkpoint) resolved 'tap-ready' on 2026-05-02. Maintainer created accidentally-awesome-labs/homebrew-runnerkit (public, default branch main, Casks/.gitkeep present) and provisioned a fine-grained PAT scoped to that repo (Contents: Read+Write, 1yr expiry); PAT stored as HOMEBREW_TAP_GITHUB_TOKEN repo secret in accidentally-awesome-labs/runnerkit Actions settings."
  - "Org migration: All release artifacts and source were renamed from `salar/...` to `accidentally-awesome-labs/...` during Task 5 closure to match the new GitHub org that owns the canonical repo. Migration committed atomically as `c359831 refactor(06-01): rename module to github.com/accidentally-awesome-labs/runnerkit` (go.mod + 79 .go files import paths + .goreleaser.yaml owners/homepage + README install/releases/cosign-cert-identity + docs/release-process.md tap+PAT+verify URLs + docs/upgrade.md brew install + docs/troubleshooting/*.md See: links + internal/update/check.go defaultAPIURL + internal/errcodes/url.go defaultDocsBase). Tasks 1-4 artifact contents remain semantically identical; only the owner segment changed."
  - "PAT rotation: The HOMEBREW_TAP_GITHUB_TOKEN PAT was pasted into chat during Task 5 resolution. Maintainer has been advised to rotate the token before the first public v1.0.0 release; no PAT value is recorded in any committed file or planning doc."
  - "go.mod (go 1.22) and bootstrap.RunnerVersion (2.334.0) are NOT bumped in this plan. Both pinned versions were intentionally preserved per the <interfaces> block; bumping is a separate PR per docs/release-process.md. (go.mod module PATH was renamed in c359831 — version unchanged.)"

requirements-completed: [REL-05]

duration: 5m (+ Task 5 closure)
completed: 2026-05-02
status: complete
---

# Phase 6 Plan 01: Release Packaging Summary

**Tag-triggered GoReleaser pipeline producing 4-platform signed binaries (cosign keyless via GitHub Actions OIDC), Homebrew Cask publishing to a separate tap repo, plus install/verify documentation and a maintainer-only release process — closing REL-05 distribution requirement and Phase 6 success criterion 1.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-05-02T20:56:53Z
- **Completed:** 2026-05-02T21:01:26Z
- **Tasks:** 5 of 5 complete (Tasks 1-4 auto + Task 5 human-action resolved 'tap-ready').
- **Files created:** 5 (`.goreleaser.yaml`, `.github/workflows/release.yml`, `.github/workflows/pr-checks.yml`, `.gitignore`, `docs/release-process.md`)
- **Files modified:** 1 in Tasks 1-4 (`README.md`); +12 in Task 5 closure org migration commit `c359831` (`go.mod`, `internal/update/check.go`, `internal/errcodes/url.go`, `.goreleaser.yaml`, `README.md`, `docs/release-process.md`, `docs/upgrade.md`, `docs/troubleshooting/auth.md`, `docs/troubleshooting/bootstrap.md`, `docs/troubleshooting/cleanup.md`, `docs/troubleshooting/github.md`, `docs/troubleshooting/provider.md`, `docs/troubleshooting/ssh.md`) plus 79 Go source files for import-path rename.

## Accomplishments

- **Tag-triggered release pipeline is wired (D-03).** `.github/workflows/release.yml` runs on `push: tags: ['v*']` with `id-token: write` + `contents: write`, pins `actions/checkout@v4` (`fetch-depth: 0`) + `actions/setup-go@v5` (1.22) + `sigstore/cosign-installer@v3` (`cosign-release: v3.0.6`) + `goreleaser/goreleaser-action@v7` (`version: '~> v2'`), and runs `release --clean` (production, never `--snapshot`).
- **PR CI validates the entire release path (D-03 Pitfall 10 mitigation).** `.github/workflows/pr-checks.yml` runs `goreleaser check` + `goreleaser release --snapshot --skip=publish --clean` + a 4-arch `dist/` content assertion (`darwin_amd64/darwin_arm64/linux_amd64/linux_arm64` archives + `checksums.txt`) + `go test ./... -count=1 -race`. Any PR that breaks the GoReleaser config or the 4-platform matrix fails red.
- **GoReleaser v2 config produces 4-platform binaries + checksums + sigstore bundle + Homebrew Cask (D-01..D-05).** `.goreleaser.yaml` declares `version: 2` (NOT v1), `goos: [darwin, linux]` × `goarch: [amd64, arm64]` (NOT windows/386/arm), `signs[].artifacts: checksum` (sign ONLY checksums.txt per D-04), `signs[].args` includes `--bundle=${signature}` and `--yes` for non-interactive keyless, `homebrew_casks:` (NOT deprecated `brews:`) targeting `accidentally-awesome-labs/homebrew-runnerkit` with `HOMEBREW_TAP_GITHUB_TOKEN`, and `-X main.version={{.Version}}` injected into `cmd/runnerkit/main.go::var version = "dev"`.
- **Install + verification documented in README.md (D-05).** New `## Install` section provides Homebrew tap command (`brew install accidentally-awesome-labs/runnerkit/runnerkit`), 4-platform asset matrix, GitHub Releases download flow with the canonical `runnerkit_<version>_<os>_<arch>.tar.gz` + `_checksums.txt` + `_checksums.txt.sigstore.json` filenames, literal `cosign verify-blob` snippet pinning the cert-identity URL to `.github/workflows/release.yml@refs/tags/${TAG}` and OIDC issuer `https://token.actions.githubusercontent.com`, follow-up `sha256sum -c`, extraction/install commands, and a forward link to `docs/troubleshooting/README.md` for verification failures.
- **Maintainer release procedure documented (`docs/release-process.md`).** Covers the two one-time prerequisites (creating `accidentally-awesome-labs/homebrew-runnerkit` with `Casks/` + `main` branch; creating a fine-grained PAT scoped to that repo with `Contents: Read and write` and storing it as `HOMEBREW_TAP_GITHUB_TOKEN`), pre-tag checklist (forward references to `make smoke-live` from Plan 06-04 + 10-min stopwatch from D-13), tag-from-upstream-only requirement (fork PRs strip the `id-token: write` permission cosign keyless requires per Pitfall 1), post-tag verify command, Common Failures table, and the `RELEASE-NOTES-vX.Y.Z.md` convention.
- **Org migration to `accidentally-awesome-labs` (Task 5 closure).** All `salar/...` references renamed to `accidentally-awesome-labs/...` in a single atomic commit `c359831` (Go module path + 79 source files + .goreleaser.yaml owners/homepage + README install/releases/cosign cert-identity + docs/release-process.md tap+PAT+verify URLs + docs/upgrade.md + 7 docs/troubleshooting/*.md `See:` link bases + internal/update/check.go defaultAPIURL + internal/errcodes/url.go defaultDocsBase — the latter was already in place from Plan 06-03 and confirmed unchanged in this rename). The migration is a literal owner-segment rename; no behavior changed.
- **Tap repo + PAT secret created (Task 5 closure).** Maintainer created `accidentally-awesome-labs/homebrew-runnerkit` (public, default branch `main`, `Casks/.gitkeep` present) and stored `HOMEBREW_TAP_GITHUB_TOKEN` (fine-grained PAT, scoped to homebrew-runnerkit, Contents R+W, 1yr expiration) as a repo secret in `accidentally-awesome-labs/runnerkit` Actions settings. Verifiable via `gh repo view accidentally-awesome-labs/homebrew-runnerkit` and `gh secret list -R accidentally-awesome-labs/runnerkit`.

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | GoReleaser v2 config + dist gitignore | `17f2244` | `.goreleaser.yaml`, `.gitignore` |
| 2 | release.yml + pr-checks.yml workflows | `7168af3` | `.github/workflows/release.yml`, `.github/workflows/pr-checks.yml` |
| 3 | README.md Install section + cosign verify-blob | `6f3b59b` | `README.md` |
| 4 | docs/release-process.md (maintainer-only) | `7123d3d` | `docs/release-process.md` |
| —    | Tasks 1-4 SUMMARY + STATE/ROADMAP metadata | `67f9f2d` | `.planning/...` |
| —    | Org migration: salar → accidentally-awesome-labs | `c359831` | `go.mod`, 79 .go files, `.goreleaser.yaml`, `README.md`, `docs/release-process.md`, `docs/upgrade.md`, 7 `docs/troubleshooting/*.md`, `internal/update/check.go`, `internal/errcodes/url.go` (already migrated) |
| 5 | Maintainer creates tap repo + PAT secret | resolved 'tap-ready' (external; this closure commit) | `accidentally-awesome-labs/homebrew-runnerkit` repo + `HOMEBREW_TAP_GITHUB_TOKEN` secret in `accidentally-awesome-labs/runnerkit` |

## Files Created

- `.goreleaser.yaml` — GoReleaser v2 schema; 4-platform builds; sha256 checksums; cosign keyless sign-blob with bundle (`.sigstore.json`); Homebrew Cask publish to `accidentally-awesome-labs/homebrew-runnerkit`; `-X main.version` ldflags injection.
- `.github/workflows/release.yml` — Tag-triggered (`v*`) release pipeline with `id-token: write` + `contents: write` permissions, pinned cosign-installer v3.0.6 + goreleaser-action@v7 + setup-go@v5/1.22 + checkout@v4 (fetch-depth: 0). Env: `GITHUB_TOKEN` + `HOMEBREW_TAP_GITHUB_TOKEN`.
- `.github/workflows/pr-checks.yml` — PR/main-push CI: `goreleaser check`, `goreleaser release --snapshot --skip=publish --clean`, dist 4-arch assertion, plus `go test ./... -count=1 -race` job. `permissions: contents: read` (no signing on PRs).
- `.gitignore` — Single line: `dist/`.
- `docs/release-process.md` — Maintainer-only release procedure (118 lines): one-time prerequisites, pre-tag checklist, tag-push procedure, post-tag verify, common-failures table, RELEASE-NOTES convention.

## Files Modified

Plan 06-01 Tasks 1-4 modified one file:

- `README.md` — Inserted `## Install` section between the intro paragraph and `## Safety: persistent vs ephemeral`. Adds Homebrew install command, 4-row supported-platforms table, GitHub Releases download bash block, literal `cosign verify-blob` snippet (with `--bundle`, `--certificate-identity`, `--certificate-oidc-issuer`, positional checksums file), `sha256sum -c` follow-up, extract/install steps, and forward link to `docs/troubleshooting/README.md`. Existing sections (Safety, BYO quickstart, Cloud quickstart, Troubleshooting from Plan 06-03, BYO operations) preserved untouched.

Task 5 closure org migration (commit `c359831`) modified the following additional files (literal `salar/...` → `accidentally-awesome-labs/...` rename; no behavior change):

- `go.mod` — module path renamed.
- 79 Go source files — import paths refactored.
- `.goreleaser.yaml` — `homebrew_casks[].repository.owner`, `homebrew_casks[].homepage`, `release.github.owner` renamed.
- `README.md` — `brew install ...`, releases URLs, cosign `--certificate-identity` URL renamed.
- `docs/release-process.md` — tap repo + PAT instructions + cosign cert + curl URLs renamed.
- `docs/upgrade.md` — `brew install` line renamed.
- `docs/troubleshooting/auth.md`, `bootstrap.md`, `cleanup.md`, `github.md`, `provider.md`, `ssh.md` — RKD-code `See:` link bases renamed (7 files total including README.md unchanged in those 7).
- `internal/update/check.go` — `defaultAPIURL` renamed.
- `internal/errcodes/url.go` — `defaultDocsBase` was already migrated by Plan 06-03; confirmed unchanged in `c359831`.

## Decisions Made

See `key-decisions` in the frontmatter. The notable ones:

- **Install section placement:** After intro, before `## Safety`. Install is the first user action; safety/quickstarts come after they have a binary. Plan 06-03's `## Troubleshooting` section sits later in the doc and is preserved.
- **Plan grep-contract resolved by phrasing:** The plan body's "Windows, Linux 386, and 32-bit ARM are not supported." conflicts with the plan's own automated check `! grep -q windows`. Resolved by listing only "Linux 386 and 32-bit ARM are not supported" (the platform table communicates Windows non-support implicitly).
- **Task 5 deferred unless maintainer signals:** Repository artifacts are complete and correct; the missing one-time external-repo + PAT secret are required for the v1.0.0 tag push (Plan 06-04), not for any of the file outputs of this plan.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] Plan body's "Windows, Linux 386, and 32-bit ARM are not supported." conflicted with the plan's own automated grep `! grep -q "windows"`**

- **Found during:** Task 3 verification.
- **Issue:** Writing the literal plan text would fail Task 3's automated criterion `! grep -q "windows" README.md`. The plan's `<acceptance_criteria>` says: "Does NOT contain `go install`, `windows`, `.deb`, or `.rpm`."
- **Fix:** Replaced "Windows, Linux 386, and 32-bit ARM are not supported." with "Linux 386 and 32-bit ARM are not supported." Net behavior matches D-02 (no Windows support) — the explicit 4-row platform table communicates Windows non-support directly, so the negative sentence does not need to repeat it.
- **Files modified:** `README.md`.
- **Verification:** `grep -qi "windows" README.md` returns non-zero (no match).
- **Committed in:** `6f3b59b` (Task 3).

**2. [Rule 3 — Blocking] PreToolUse hook initially blocked Write of `.github/workflows/release.yml` with a security reminder**

- **Found during:** Task 2 (Write of release.yml).
- **Issue:** The repo's `security_reminder_hook.py` PreToolUse hook flagged the workflow Write because workflows can be vulnerable to command injection from `github.event.*` user inputs. The reminder is principle-only; the actual workflows do NOT consume any `github.event.*` user input in `run:` blocks (only `secrets.*` and pinned action references), so there is no injection surface.
- **Fix:** Re-issued the Write call. Both workflow files were created on the second attempt.
- **Files modified:** none (re-attempt only).
- **Verification:** Both workflow files exist, pass all Task 2 grep checks, and contain no `github.event.*` references in any `run:` block.
- **Committed in:** `7168af3` (Task 2).

### Non-deviations (pre-existing untracked files)

- `.pi-lens/` and `.pi/` directories were already present as untracked items at plan start (Vercel plugin local cache). Left untouched per scope-boundary rule (out-of-scope; not caused by this plan's changes).

## Task 5 resolution (resolved 'tap-ready' 2026-05-02)

Task 5 was a `checkpoint:human-action` gate requiring admin access to a GitHub org. The maintainer resolved it externally with the signal `tap-ready` after performing four out-of-band steps. All four are verified via `gh` CLI / filesystem checks (no automation can perform these — interactive PAT creation + cross-repo admin access is required).

1. **Org migration to `accidentally-awesome-labs`.** Tasks 1-4 originally referenced `salar/...` per the plan-as-written. Maintainer chose to publish RunnerKit under a new GitHub org (`accidentally-awesome-labs`) before the first tag push. Migration was done atomically as commit `c359831 refactor(06-01): rename module to github.com/accidentally-awesome-labs/runnerkit`:
   - `go.mod` module path: `github.com/salar/runnerkit` → `github.com/accidentally-awesome-labs/runnerkit`
   - 79 Go source-file imports refactored
   - `.goreleaser.yaml`: `homebrew_casks[].repository.owner`, `release.github.owner`, and `homebrew_casks[].homepage` all renamed
   - `README.md`: `brew install ...`, releases URLs, cosign `--certificate-identity` URL all renamed
   - `docs/release-process.md`: tap repo + PAT instructions + cosign cert + curl URLs all renamed
   - `docs/upgrade.md`: brew install line renamed
   - `docs/troubleshooting/*.md` (7 files): RKD-code `See:` link bases all renamed
   - `internal/update/check.go::defaultAPIURL` renamed
   - `internal/errcodes/url.go::defaultDocsBase` confirmed already at `accidentally-awesome-labs` (Plan 06-03 had already migrated it; no change needed in `c359831`).

2. **Tap repo created.** `accidentally-awesome-labs/homebrew-runnerkit` is public, default branch `main`, contains `Casks/.gitkeep` and a README. Verify: `gh repo view accidentally-awesome-labs/homebrew-runnerkit` returns success; `gh api repos/accidentally-awesome-labs/homebrew-runnerkit/contents/Casks/.gitkeep` returns the file.

3. **PAT created and secret stored.** `HOMEBREW_TAP_GITHUB_TOKEN` exists in `accidentally-awesome-labs/runnerkit` repo Actions secrets (created 2026-05-02). Token is fine-grained, scoped to `homebrew-runnerkit` only, with `Contents: Read and write`, 1-year expiration. Verify: `gh secret list -R accidentally-awesome-labs/runnerkit` shows the secret.

4. **Runnerkit repo created + remote configured.** `accidentally-awesome-labs/runnerkit` is live and `git remote -v` reports `origin https://github.com/accidentally-awesome-labs/runnerkit.git`.

**Resolution status:** `tap-ready` — both the tap repo and the secret exist. The first `vX.Y.Z` tag push (Plan 06-04) will execute the full release pipeline including `homebrew_casks:` formula publish.

**PAT rotation requirement.** During Task 5 resolution the maintainer pasted the fine-grained PAT into a chat session. The PAT is NOT recorded in any committed file or planning doc, but the maintainer has been advised to rotate `HOMEBREW_TAP_GITHUB_TOKEN` (revoke the pasted PAT, generate a new one, replace the secret) before the public v1.0.0 release tag is pushed.

## Forward References Created

- **README.md → `docs/troubleshooting/README.md`** — install verification troubleshooting link (target file created by Plan 06-03).
- **`docs/release-process.md` → `make smoke-live`** — pre-tag checklist requirement (target Makefile target created by Plan 06-04).
- **`docs/release-process.md` → `RELEASE-NOTES-v1.0.0.md`** — first release notes file (created by Plan 06-04).
- **`docs/release-process.md` → `06-VERIFICATION.md`** — Phase 6 verification baseline (created by `/gsd:verify-work` after Plan 06-04).

## Cross-plan notes

- **Confirmed runner pin location.** `06-RESEARCH.md` referenced `internal/bootstrap/script.go::RunnerVersion`, but the actual constant lives in `internal/bootstrap/package.go::RunnerVersion = "2.334.0"`. The `<interfaces>` block of this plan called this out and `docs/release-process.md` references the correct file. Downstream plans (06-04 in particular) should treat `internal/bootstrap/package.go` as authoritative for the runner pin.
- **README.md will be touched again by Plan 06-04** for any v1.0.0 release-notes hyperlink. The `## Install` section added here is the canonical install copy and should be preserved.
- **`docs/release-process.md` and `.gitignore` will be touched by Plan 06-04.** Per the orchestrator notes, 06-04 appends additional sections (10-min stopwatch checklist body, smoke artifacts ignore patterns) to the structures this plan ships. The structure here is intentionally skeletal in the smoke-checklist area; Plan 06-04 fills it in.
- **No Phase 5 invariant regressions.** `go build ./...` clean; `go vet ./...` clean; `go test ./... -count=1` green across all 16 packages. The redactor flow (`internal/redact/`) is untouched; `cmd/runnerkit/main.go::var version = "dev"` remains the single ldflags injection slot.

## Validation matrix coverage (06-VALIDATION.md)

- **Line 51 — GoReleaser config schema valid:** wired (Task 1 output + Task 2 PR CI exercises `goreleaser check`); will turn green on first PR run.
- **Line 52 — All 4 platforms + checksums + sigstore bundle produced:** wired (Task 2 pr-checks snapshot job + 4-arch dist assertion); will turn green on first PR run.
- **Line 53 — Cosign signature verifies for issuer/identity in README:** wired (Tasks 1+2+3 string-equality between README cert-identity URL and workflow file path); exercised end-to-end on first real `v*` tag push (Plan 06-04).

The signing path is dry-runnable on PRs only via the snapshot config (sigstore signing is suppressed by `--snapshot`); full validation is a Plan 06-04 + tag-push concern.

## Self-Check: PASSED

- All 5 created/modified files exist on disk.
- All 4 task commits exist in git log (`17f2244`, `7168af3`, `6f3b59b`, `7123d3d`).
- Tasks 1-4 SUMMARY/STATE/ROADMAP closure committed in `67f9f2d`.
- Org migration committed in `c359831` (verified: `head -1 go.mod` reports `module github.com/accidentally-awesome-labs/runnerkit`; no `salar/` references remain in `.goreleaser.yaml`, README.md, docs/release-process.md, docs/upgrade.md, docs/troubleshooting/*.md, internal/update/check.go, internal/errcodes/url.go).
- Task 5 resolved 'tap-ready' on 2026-05-02. Verified via `gh` CLI:
  - `gh repo view accidentally-awesome-labs/homebrew-runnerkit` → public, default branch `main`, exists.
  - `gh api repos/accidentally-awesome-labs/homebrew-runnerkit/contents/Casks/.gitkeep` → present.
  - `gh secret list -R accidentally-awesome-labs/runnerkit` → `HOMEBREW_TAP_GITHUB_TOKEN` listed.
  - `git remote -v` → `origin https://github.com/accidentally-awesome-labs/runnerkit.git`.
- All Task 1-4 automated grep verifications still pass under the migrated owner segment (cert-identity URL in README.md now `https://github.com/accidentally-awesome-labs/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}`).
- `go build ./...` clean; `go vet ./...` clean; `go test ./... -count=1` 16/16 packages green (asserted post-rename in `c359831`).
- No emojis in any deliverable file.
- No PAT value written to any planning file or commit.
