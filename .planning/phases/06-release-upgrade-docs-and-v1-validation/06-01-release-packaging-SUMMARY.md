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
  - Locked cert-identity URL pattern: https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}.
affects: [06-04-v1-validation-and-live-smoke]

tech-stack:
  added:
    - GoReleaser v2 (action v7, runtime constrained via "version: '~> v2'")
    - sigstore/cosign-installer@v3 (cosign-release: v3.0.6)
    - Homebrew tap salar/homebrew-runnerkit (external repo; one-time creation deferred to maintainer per checkpoint Task 5)
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
  - "Task 5 (human-action checkpoint) intentionally not auto-resolved. Per plan acceptance, the maintainer must either (a) create salar/homebrew-runnerkit + HOMEBREW_TAP_GITHUB_TOKEN secret and signal 'tap-ready', or (b) signal 'deferred' acknowledging the v1.0.0 tag in Plan 06-04 will fail at the homebrew_casks: step until both exist. Repository artifacts (Tasks 1-4) are correct and complete regardless of the resolution."
  - "go.mod (go 1.22) and bootstrap.RunnerVersion (2.334.0) are NOT bumped in this plan. Both pinned versions were intentionally preserved per the <interfaces> block; bumping is a separate PR per docs/release-process.md."

requirements-completed: [REL-05]

duration: 5m
completed: 2026-05-02
---

# Phase 6 Plan 01: Release Packaging Summary

**Tag-triggered GoReleaser pipeline producing 4-platform signed binaries (cosign keyless via GitHub Actions OIDC), Homebrew Cask publishing to a separate tap repo, plus install/verify documentation and a maintainer-only release process — closing REL-05 distribution requirement and Phase 6 success criterion 1.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-05-02T20:56:53Z
- **Completed:** 2026-05-02T21:01:26Z
- **Tasks:** 4 of 5 complete (Tasks 1-4 auto). Task 5 is a `checkpoint:human-action` gate awaiting maintainer creation of `salar/homebrew-runnerkit` repo + `HOMEBREW_TAP_GITHUB_TOKEN` secret.
- **Files created:** 5 (`.goreleaser.yaml`, `.github/workflows/release.yml`, `.github/workflows/pr-checks.yml`, `.gitignore`, `docs/release-process.md`)
- **Files modified:** 1 (`README.md`)

## Accomplishments

- **Tag-triggered release pipeline is wired (D-03).** `.github/workflows/release.yml` runs on `push: tags: ['v*']` with `id-token: write` + `contents: write`, pins `actions/checkout@v4` (`fetch-depth: 0`) + `actions/setup-go@v5` (1.22) + `sigstore/cosign-installer@v3` (`cosign-release: v3.0.6`) + `goreleaser/goreleaser-action@v7` (`version: '~> v2'`), and runs `release --clean` (production, never `--snapshot`).
- **PR CI validates the entire release path (D-03 Pitfall 10 mitigation).** `.github/workflows/pr-checks.yml` runs `goreleaser check` + `goreleaser release --snapshot --skip=publish --clean` + a 4-arch `dist/` content assertion (`darwin_amd64/darwin_arm64/linux_amd64/linux_arm64` archives + `checksums.txt`) + `go test ./... -count=1 -race`. Any PR that breaks the GoReleaser config or the 4-platform matrix fails red.
- **GoReleaser v2 config produces 4-platform binaries + checksums + sigstore bundle + Homebrew Cask (D-01..D-05).** `.goreleaser.yaml` declares `version: 2` (NOT v1), `goos: [darwin, linux]` × `goarch: [amd64, arm64]` (NOT windows/386/arm), `signs[].artifacts: checksum` (sign ONLY checksums.txt per D-04), `signs[].args` includes `--bundle=${signature}` and `--yes` for non-interactive keyless, `homebrew_casks:` (NOT deprecated `brews:`) targeting `salar/homebrew-runnerkit` with `HOMEBREW_TAP_GITHUB_TOKEN`, and `-X main.version={{.Version}}` injected into `cmd/runnerkit/main.go::var version = "dev"`.
- **Install + verification documented in README.md (D-05).** New `## Install` section provides Homebrew tap command (`brew install salar/runnerkit/runnerkit`), 4-platform asset matrix, GitHub Releases download flow with the canonical `runnerkit_<version>_<os>_<arch>.tar.gz` + `_checksums.txt` + `_checksums.txt.sigstore.json` filenames, literal `cosign verify-blob` snippet pinning the cert-identity URL to `.github/workflows/release.yml@refs/tags/${TAG}` and OIDC issuer `https://token.actions.githubusercontent.com`, follow-up `sha256sum -c`, extraction/install commands, and a forward link to `docs/troubleshooting/README.md` for verification failures.
- **Maintainer release procedure documented (`docs/release-process.md`).** Covers the two one-time prerequisites (creating `salar/homebrew-runnerkit` with `Casks/` + `main` branch; creating a fine-grained PAT scoped to that repo with `Contents: Read and write` and storing it as `HOMEBREW_TAP_GITHUB_TOKEN`), pre-tag checklist (forward references to `make smoke-live` from Plan 06-04 + 10-min stopwatch from D-13), tag-from-upstream-only requirement (fork PRs strip the `id-token: write` permission cosign keyless requires per Pitfall 1), post-tag verify command, Common Failures table, and the `RELEASE-NOTES-vX.Y.Z.md` convention.

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | GoReleaser v2 config + dist gitignore | `17f2244` | `.goreleaser.yaml`, `.gitignore` |
| 2 | release.yml + pr-checks.yml workflows | `7168af3` | `.github/workflows/release.yml`, `.github/workflows/pr-checks.yml` |
| 3 | README.md Install section + cosign verify-blob | `6f3b59b` | `README.md` |
| 4 | docs/release-process.md (maintainer-only) | `7123d3d` | `docs/release-process.md` |
| 5 | Maintainer creates tap repo + PAT secret | (checkpoint:human-action — no commit) | (no repo files) |

## Files Created

- `.goreleaser.yaml` — GoReleaser v2 schema; 4-platform builds; sha256 checksums; cosign keyless sign-blob with bundle (`.sigstore.json`); Homebrew Cask publish to `salar/homebrew-runnerkit`; `-X main.version` ldflags injection.
- `.github/workflows/release.yml` — Tag-triggered (`v*`) release pipeline with `id-token: write` + `contents: write` permissions, pinned cosign-installer v3.0.6 + goreleaser-action@v7 + setup-go@v5/1.22 + checkout@v4 (fetch-depth: 0). Env: `GITHUB_TOKEN` + `HOMEBREW_TAP_GITHUB_TOKEN`.
- `.github/workflows/pr-checks.yml` — PR/main-push CI: `goreleaser check`, `goreleaser release --snapshot --skip=publish --clean`, dist 4-arch assertion, plus `go test ./... -count=1 -race` job. `permissions: contents: read` (no signing on PRs).
- `.gitignore` — Single line: `dist/`.
- `docs/release-process.md` — Maintainer-only release procedure (118 lines): one-time prerequisites, pre-tag checklist, tag-push procedure, post-tag verify, common-failures table, RELEASE-NOTES convention.

## Files Modified

- `README.md` — Inserted `## Install` section between the intro paragraph and `## Safety: persistent vs ephemeral`. Adds Homebrew install command, 4-row supported-platforms table, GitHub Releases download bash block, literal `cosign verify-blob` snippet (with `--bundle`, `--certificate-identity`, `--certificate-oidc-issuer`, positional checksums file), `sha256sum -c` follow-up, extract/install steps, and forward link to `docs/troubleshooting/README.md`. Existing sections (Safety, BYO quickstart, Cloud quickstart, Troubleshooting from Plan 06-03, BYO operations) preserved untouched.

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

## Outstanding Maintainer Prerequisites (Task 5 — Checkpoint)

This plan landed all repository artifacts but Task 5 is a `checkpoint:human-action` gate that ONLY a human with admin access to `github.com/salar` can resolve. Two external one-time setup actions are required before the first `vX.Y.Z` tag push (Plan 06-04 will exercise this):

1. **Create `salar/homebrew-runnerkit` GitHub repo** with a `Casks/` directory and default branch `main`. Verify at <https://github.com/salar/homebrew-runnerkit>.
2. **Create a fine-grained PAT** scoped to `salar/homebrew-runnerkit` with `Contents: Read and write`, then store it as the `HOMEBREW_TAP_GITHUB_TOKEN` repo secret in `salar/runnerkit` settings → Secrets and variables → Actions. Verify at <https://github.com/salar/runnerkit/settings/secrets/actions>.

If either is missing when the v1.0.0 tag is pushed, the GoReleaser run fails at the `homebrew_casks:` step with `403: Resource not accessible by integration`. The release binaries + cosign signature still publish (those depend only on `GITHUB_TOKEN` + `id-token: write`); only the cask formula update fails.

**Resume signals (from plan):**

- `tap-ready` — both the repo and the secret exist; safe to push tags.
- `deferred` — explicit acknowledgement that v1.0.0 tag push will fail at `homebrew_casks:` until resolved.

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
- Task 5 is a `checkpoint:human-action` with no expected commit — verified intentional gap.
- All Task 1-4 automated grep verifications pass.
- `go build ./...` clean; `go vet ./...` clean; `go test ./... -count=1` 16/16 packages green.
- No emojis in any deliverable file.
