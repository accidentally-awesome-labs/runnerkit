---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 01
type: execute
wave: A
depends_on: []
files_modified:
  - .goreleaser.yaml
  - .github/workflows/release.yml
  - .github/workflows/pr-checks.yml
  - .gitignore
  - README.md
  - docs/release-process.md
autonomous: false
requirements: [REL-05]
must_haves:
  truths:
    - "Pushing a vX.Y.Z tag from the upstream repo produces darwin_arm64, darwin_amd64, linux_amd64, linux_arm64 binaries plus a checksums.txt and a checksums.txt.sigstore.json bundle."
    - "Every PR runs `goreleaser check` and `goreleaser release --snapshot --skip=publish --clean` and fails red if either step fails."
    - "A maintainer can copy-paste the cosign verify-blob snippet from README.md and verify a published checksums.txt against the embedded cert-identity URL."
    - "Homebrew tap `salar/homebrew-runnerkit` receives a Cask formula update on every successful tag run."
    - "The release workflow refuses to run from fork PRs (cosign keyless requires id-token: write that forks strip)."
  artifacts:
    - path: ".goreleaser.yaml"
      provides: "GoReleaser v2 schema config producing 4 platform binaries, checksums.txt, sigstore bundle, Homebrew Cask"
      contains: "version: 2"
      contains_also: "homebrew_casks:"
      contains_also2: "artifacts: checksum"
    - path: ".github/workflows/release.yml"
      provides: "Tag-triggered (`on: push: tags: ['v*']`) GoReleaser release workflow with id-token: write"
      contains: "id-token: write"
      contains_also: "sigstore/cosign-installer@v3"
      contains_also2: "goreleaser/goreleaser-action@v7"
    - path: ".github/workflows/pr-checks.yml"
      provides: "PR CI gate: `goreleaser check` + `goreleaser release --snapshot --skip=publish --clean`"
      contains: "goreleaser check"
      contains_also: "release --snapshot --skip=publish --clean"
    - path: "README.md"
      provides: "Install matrix (Homebrew + Releases) + literal cosign verify-blob snippet (D-05)"
      contains: "brew install salar/runnerkit/runnerkit"
      contains_also: "cosign verify-blob"
      contains_also2: "https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/"
      contains_also3: "https://token.actions.githubusercontent.com"
    - path: "docs/release-process.md"
      provides: "Maintainer-only checklist: tap-repo creation, HOMEBREW_TAP_GITHUB_TOKEN secret, smoke gates, tag procedure"
      contains: "salar/homebrew-runnerkit"
      contains_also: "HOMEBREW_TAP_GITHUB_TOKEN"
      contains_also2: "make smoke-live"
  key_links:
    - from: ".github/workflows/release.yml"
      to: ".goreleaser.yaml"
      via: "goreleaser/goreleaser-action@v7 with version: '~> v2'"
    - from: ".goreleaser.yaml signs:"
      to: "cosign sign-blob --bundle"
      via: "GitHub Actions OIDC token (id-token: write) → Sigstore Fulcio"
      pattern: "sign-blob.*--bundle.*--yes"
    - from: ".goreleaser.yaml homebrew_casks:"
      to: "salar/homebrew-runnerkit repo"
      via: "HOMEBREW_TAP_GITHUB_TOKEN PAT in {{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    - from: "README.md cosign verify-blob snippet"
      to: ".github/workflows/release.yml@refs/tags/$TAG"
      via: "literal --certificate-identity URL"
      pattern: "github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/"
---

<objective>
Add tag-triggered GoReleaser release pipeline that produces 4 platform binaries (darwin_arm64, darwin_amd64, linux_amd64, linux_arm64), a SHA256 checksums file, a cosign keyless signature on that checksums file, a Homebrew Cask formula update on the tap repo, and a published GitHub Release — all from a single `vX.Y.Z` tag push. Land README install matrix + cosign verify-blob snippet (D-05) and a maintainer-only `docs/release-process.md` covering the one-time tap repo + secret prerequisites.

Implements **D-01..D-05** from CONTEXT.md (release packaging). Closes the REL-05 binding requirement for distribution.

Purpose: Phase 6 success criterion 1 — "Developer can install an official RunnerKit release binary/package."

Output: A repo state where `git tag v0.0.1-test && git push --tags` would (in the upstream repo, with the tap repo + secret pre-created) produce a complete signed release. No tag is pushed in this plan; that is the maintainer step in Plan 06-04.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/REQUIREMENTS.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VALIDATION.md
@cmd/runnerkit/main.go
@internal/bootstrap/package.go

<interfaces>
<!-- Existing repo state the plan must integrate with. Extracted 2026-05-02. -->

Module path (from go.mod):
```
module github.com/salar/runnerkit
go 1.22
```

Version-injection slot (cmd/runnerkit/main.go line 12):
```go
var version = "dev"
```
GoReleaser MUST inject with `-ldflags '-s -w -X main.version={{.Version}}'` on the build for `cmd/runnerkit`.

Runner pin (internal/bootstrap/package.go — not script.go; RESEARCH had a typo):
```go
const RunnerVersion = "2.334.0"
```
Phase 6 keeps this pin. README/docs reference the bundled runner version as 2.334.0.

Hard-locked stack versions (from RESEARCH §"Standard Stack" — registry-confirmed 2026-05-02):
- GoReleaser: v2.15.4 (workflow constraint: `version: '~> v2'`)
- goreleaser-action: v7
- cosign: v3.0.6 (installer: `sigstore/cosign-installer@v3` with `cosign-release: 'v3.0.6'`)
- actions/checkout: v4 (with `fetch-depth: 0` — required for changelog)
- actions/setup-go: v5 (`go-version: '1.22'`)

Hard-locked URLs:
- Cosign cert-identity: `https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/$TAG`
- Cosign OIDC issuer: `https://token.actions.githubusercontent.com` (NOT `https://github.com/login/oauth`)

GoReleaser deprecation rules (from RESEARCH §"Anti-Patterns" / Pitfall 8):
- Use `homebrew_casks:` (NOT deprecated `brews:`)
- Use bundle format (`.sigstore.json`) (NOT v1/v2 separate `.sig`+`.crt`)
- Sign ONLY checksums (`signs[].artifacts: checksum`) per D-04 (NOT all archives)

Phase 5 invariants this plan must NOT regress:
- The redactor (`internal/redact/`) flows through all CLI output — release notes / smoke output continues through it (touched in Plan 06-04, not here).
- `cmd/runnerkit/main.go::var version = "dev"` is the ONLY ldflags slot; do not introduce others.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="false">
  <name>Task 1: Create .goreleaser.yaml v2 schema producing 4 platforms + checksums + sigstore bundle + Homebrew Cask</name>
  <files>.goreleaser.yaml, .gitignore</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (read Pattern 1 verbatim — copy YAML structure, do not paraphrase)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-01..D-05)
    - cmd/runnerkit/main.go (line 12 — `var version = "dev"` is the ldflags injection slot)
    - internal/bootstrap/package.go (confirms `RunnerVersion = "2.334.0"` is unchanged for v1)
    - go.mod (confirms module path `github.com/salar/runnerkit` and `go 1.22`)
    - .gitignore (so we know what's already ignored before we touch it)
  </read_first>
  <action>
Create `.goreleaser.yaml` at repo root with the EXACT structure below (copied verbatim from RESEARCH Pattern 1; do not adapt):

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - id: runnerkit
    main: ./cmd/runnerkit
    binary: runnerkit
    env:
      - CGO_ENABLED=0
    goos: [darwin, linux]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - id: runnerkit
    formats: [tar.gz]
    name_template: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}'

checksum:
  name_template: '{{ .ProjectName }}_{{ .Version }}_checksums.txt'
  algorithm: sha256

signs:
  - cmd: cosign
    signature: '${artifact}.sigstore.json'
    args:
      - sign-blob
      - '--bundle=${signature}'
      - '${artifact}'
      - '--yes'
    artifacts: checksum

homebrew_casks:
  - name: runnerkit
    binary: runnerkit
    repository:
      owner: salar
      name: homebrew-runnerkit
      branch: main
      token: '{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}'
    directory: Casks
    homepage: https://github.com/salar/runnerkit
    description: Reliable GitHub Actions self-hosted runners for solo developers
    commit_author:
      name: runnerkit-release-bot
      email: noreply@github.com
    commit_msg_template: 'runnerkit: bump cask to {{ .Tag }}'

release:
  github:
    owner: salar
    name: runnerkit
  draft: false
  prerelease: auto
  name_template: '{{ .Tag }}'

snapshot:
  version_template: '{{ incpatch .Version }}-SNAPSHOT-{{.ShortCommit}}'

changelog:
  sort: asc
```

Hard rules locked from CONTEXT/RESEARCH (verbatim):
- `version: 2` MUST be the first line; v1 schema is unsupported.
- `goos: [darwin, linux]` × `goarch: [amd64, arm64]` produces exactly the 4 supported platforms (D-02). Do NOT add windows, 386, or arm.
- `homebrew_casks:` (NOT deprecated `brews:`).
- `signs[].artifacts: checksum` (sign ONLY checksums.txt per D-04 — do NOT sign each archive).
- `signs[].args` MUST include `--bundle=${signature}` and `--yes` (for non-interactive CI keyless).
- `homebrew_casks[].repository.owner: salar` and `name: homebrew-runnerkit` (separate repo per RESEARCH Pitfall 2).
- `homebrew_casks[].repository.token: '{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}'` (default GITHUB_TOKEN cannot push cross-repo).

After writing the file, ensure `.gitignore` contains a `dist/` line (GoReleaser output dir; gitignored to avoid committing build artifacts). If `.gitignore` does not exist or does not contain `dist/`, add `dist/` on its own line.

Verify by running:
```bash
goreleaser check
```
If `goreleaser` is not installed locally, run via Docker per goreleaser docs OR rely on the pr-checks.yml workflow added in Task 2 (in that case mark this task complete on file existence + Task 2 acceptance and let CI catch any schema error).
  </action>
  <verify>
    <automated>test -f .goreleaser.yaml && head -1 .goreleaser.yaml | grep -qx 'version: 2' && grep -q 'homebrew_casks:' .goreleaser.yaml && grep -q 'artifacts: checksum' .goreleaser.yaml && grep -q "main: ./cmd/runnerkit" .goreleaser.yaml && grep -q 'goos: \[darwin, linux\]' .goreleaser.yaml && grep -q 'goarch: \[amd64, arm64\]' .goreleaser.yaml && grep -q '\-X main.version={{.Version}}' .goreleaser.yaml && grep -q '\-\-bundle=\${signature}' .goreleaser.yaml && grep -q "owner: salar" .goreleaser.yaml && grep -q 'name: homebrew-runnerkit' .goreleaser.yaml && grep -q 'HOMEBREW_TAP_GITHUB_TOKEN' .goreleaser.yaml && grep -qx 'dist/' .gitignore</automated>
  </verify>
  <acceptance_criteria>
    - File `.goreleaser.yaml` exists at repo root.
    - First line is exactly `version: 2`.
    - Contains the literal string `homebrew_casks:` AND does NOT contain a top-level `brews:` key (run `grep -q '^brews:' .goreleaser.yaml; [ $? -ne 0 ]`).
    - Contains `artifacts: checksum` under `signs:`.
    - Contains `main: ./cmd/runnerkit` and `binary: runnerkit` under `builds:`.
    - Contains `goos: [darwin, linux]` and `goarch: [amd64, arm64]` (exactly these — no windows, no 386).
    - Contains the ldflags line `-X main.version={{.Version}}`.
    - Contains `--bundle=${signature}` in `signs[].args`.
    - Contains `--yes` in `signs[].args` (non-interactive keyless).
    - Homebrew tap repo references: `owner: salar`, `name: homebrew-runnerkit`, `token: '{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}'`.
    - `.gitignore` contains a `dist/` entry on its own line.
    - Validation matrix row from `06-VALIDATION.md` line 51 ("GoReleaser config schema valid") is satisfied via `goreleaser check` (run locally or by Task 2 PR CI).
  </acceptance_criteria>
  <done>`.goreleaser.yaml` exists with v2 schema, 4-platform build matrix, checksums, cosign signs (bundle, --yes, artifacts: checksum), Homebrew Cask via `homebrew_casks:` pointing at `salar/homebrew-runnerkit` with HOMEBREW_TAP_GITHUB_TOKEN, and `dist/` is gitignored.</done>
</task>

<task type="auto" tdd="false">
  <name>Task 2: Add release.yml (tag-triggered, OIDC) + pr-checks.yml (goreleaser check + snapshot)</name>
  <files>.github/workflows/release.yml, .github/workflows/pr-checks.yml</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (Pattern 2 — verbatim YAML; Pitfall 1 — fork OIDC strip; Pitfall 8 — pin cosign-release; Pitfall 10 — `--snapshot` masks tag-only behavior)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-03)
    - .goreleaser.yaml (the file Task 1 just created — must reference the same module/binary names)
    - go.mod (Go version: 1.22 — must match `setup-go` input)
  </read_first>
  <action>
Create directory `.github/workflows/` if missing, then create TWO files:

**File 1: `.github/workflows/release.yml`** — exact contents (from RESEARCH Pattern 2):

```yaml
name: release
on:
  push:
    tags: ['v*']

permissions:
  contents: write
  id-token: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: sigstore/cosign-installer@v3
        with:
          cosign-release: 'v3.0.6'
      - uses: goreleaser/goreleaser-action@v7
        with:
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

Hard rules from RESEARCH:
- `on: push: tags: ['v*']` — tag-triggered ONLY (per D-03; tag pushes only happen from upstream, sidesteps Pitfall 1 fork OIDC strip).
- `permissions: id-token: write` — REQUIRED for cosign keyless OIDC. Without it, the `signs:` step fails with "no token issuer."
- `permissions: contents: write` — required for GoReleaser to publish the GitHub Release + assets.
- `actions/checkout@v4` with `fetch-depth: 0` — required for GoReleaser changelog generation.
- `cosign-release: 'v3.0.6'` — pin (Pitfall 8); floating action versions cause unreproducible failures.
- `goreleaser-action@v7` with `version: '~> v2'` — pin to v2.x line.
- `args: release --clean` — production tag-mode; NEVER `--snapshot` here.
- `HOMEBREW_TAP_GITHUB_TOKEN` is referenced from repo secrets; document its creation in Task 4 (`docs/release-process.md`).

**File 2: `.github/workflows/pr-checks.yml`** — exact contents:

```yaml
name: pr-checks
on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  goreleaser-validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: sigstore/cosign-installer@v3
        with:
          cosign-release: 'v3.0.6'
      - uses: goreleaser/goreleaser-action@v7
        with:
          version: '~> v2'
          install-only: true
      - name: goreleaser check
        run: goreleaser check
      - name: goreleaser snapshot build
        run: goreleaser release --snapshot --skip=publish --clean
      - name: assert dist contents
        run: |
          set -euo pipefail
          ls dist/
          # Assert the four expected platform archives + checksums file are present
          for arch in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
            ls dist/runnerkit_*_${arch}.tar.gz >/dev/null
          done
          ls dist/runnerkit_*_checksums.txt >/dev/null

  go-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: go test
        run: go test ./... -count=1 -race
```

Hard rules:
- PR CI runs `goreleaser check` (validates schema; per Pitfall 10) AND `goreleaser release --snapshot --skip=publish --clean` (validates build matrix produces all 4 platforms).
- The `assert dist contents` step verifies the build matrix actually produced the 4 expected archives + checksums file. This is the closest we can get to validating the tag-mode behavior on PRs (Pitfall 10 mitigation).
- `permissions: contents: read` — minimal, no OIDC needed (no signing on PRs; that requires real-tag flow only).
- `goreleaser-action` uses `install-only: true` because we run `check` and `release --snapshot` as separate `run:` steps for clearer logs.
- The `go-test` job runs the standard test suite (`go test ./... -count=1 -race`) — this is the project's Sampling-Rate full-suite command per `06-VALIDATION.md`. It runs alongside goreleaser-validate so PRs fail red on either.

Both files MUST end with a trailing newline.
  </action>
  <verify>
    <automated>test -f .github/workflows/release.yml && test -f .github/workflows/pr-checks.yml && grep -q "tags: \['v\*'\]" .github/workflows/release.yml && grep -q "id-token: write" .github/workflows/release.yml && grep -q "fetch-depth: 0" .github/workflows/release.yml && grep -q "cosign-release: 'v3.0.6'" .github/workflows/release.yml && grep -q "goreleaser/goreleaser-action@v7" .github/workflows/release.yml && grep -q "version: '~> v2'" .github/workflows/release.yml && grep -q "args: release --clean" .github/workflows/release.yml && grep -q "HOMEBREW_TAP_GITHUB_TOKEN" .github/workflows/release.yml && grep -q "goreleaser check" .github/workflows/pr-checks.yml && grep -q "release --snapshot --skip=publish --clean" .github/workflows/pr-checks.yml && grep -q "go test ./... -count=1 -race" .github/workflows/pr-checks.yml</automated>
  </verify>
  <acceptance_criteria>
    - File `.github/workflows/release.yml` exists.
    - `release.yml` triggers on `push: tags: ['v*']` only (no `pull_request`, no `workflow_dispatch`).
    - `release.yml` declares `permissions: id-token: write` AND `contents: write`.
    - `release.yml` uses `actions/checkout@v4` with `fetch-depth: 0`.
    - `release.yml` uses `sigstore/cosign-installer@v3` with `cosign-release: 'v3.0.6'`.
    - `release.yml` uses `goreleaser/goreleaser-action@v7` with `version: '~> v2'` and `args: release --clean` (NOT `--snapshot`).
    - `release.yml` env block includes both `GITHUB_TOKEN` and `HOMEBREW_TAP_GITHUB_TOKEN`.
    - File `.github/workflows/pr-checks.yml` exists.
    - `pr-checks.yml` runs `goreleaser check` AND `goreleaser release --snapshot --skip=publish --clean` AND `go test ./... -count=1 -race`.
    - `pr-checks.yml` asserts `dist/` contains all 4 expected platform archives plus a checksums file (the `for arch in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64` loop is present).
    - Validation matrix rows from `06-VALIDATION.md` line 52 ("All 4 platforms + checksums + sigstore bundle produced") and line 53 ("Cosign signature on checksums.txt verifies") are wired (line 53 is exercised by the tag-mode workflow when first tag is pushed in Plan 06-04; this plan only validates dry-run via PR CI).
  </acceptance_criteria>
  <done>Tag-triggered release workflow exists with id-token: write + cosign-installer@v3 (v3.0.6) + goreleaser-action@v7 + HOMEBREW_TAP_GITHUB_TOKEN env. PR-checks workflow runs `goreleaser check` and `goreleaser release --snapshot --skip=publish --clean` and asserts all 4 dist archives + checksums.txt exist, plus runs the full Go test suite.</done>
</task>

<task type="auto" tdd="false">
  <name>Task 3: Update README.md install matrix + literal cosign verify-blob snippet</name>
  <files>README.md</files>
  <read_first>
    - README.md (current state — must preserve any existing intro/quickstart structure)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (Pattern 3 — cosign verify-blob exact command; "Code Examples" section near line 728)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-01, D-02, D-04, D-05)
    - .goreleaser.yaml (artifact name templates determine the exact filenames users will download)
  </read_first>
  <action>
Add (or replace, if a placeholder exists) an `## Install` section in `README.md` with the following EXACT content. Insert after any existing introduction / "what is this" section, before any quickstart:

```markdown
## Install

RunnerKit is distributed via two channels (D-01).

### Homebrew (macOS, Linux)

```bash
brew install salar/runnerkit/runnerkit
```

This taps the official cask repo (`salar/homebrew-runnerkit`) and installs the
latest release. Upgrade with `brew upgrade runnerkit`.

### GitHub Releases (all supported platforms)

Supported CLI host platforms (D-02):

| OS    | Architecture | Asset name                                      |
|-------|--------------|-------------------------------------------------|
| macOS | arm64        | `runnerkit_<version>_darwin_arm64.tar.gz`       |
| macOS | amd64        | `runnerkit_<version>_darwin_amd64.tar.gz`       |
| Linux | amd64        | `runnerkit_<version>_linux_amd64.tar.gz`        |
| Linux | arm64        | `runnerkit_<version>_linux_arm64.tar.gz`        |

Windows, Linux 386, and 32-bit ARM are not supported.

Download from <https://github.com/salar/runnerkit/releases/latest>:

```bash
# Replace v1.0.0 with the desired tag
TAG=v1.0.0
OS=$(uname -s | tr '[:upper:]' '[:lower:]')      # darwin or linux
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

curl -fsSL -O "https://github.com/salar/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_${OS}_${ARCH}.tar.gz"
curl -fsSL -O "https://github.com/salar/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt"
curl -fsSL -O "https://github.com/salar/runnerkit/releases/download/${TAG}/runnerkit_${TAG#v}_checksums.txt.sigstore.json"
```

### Verify the release (D-05)

Verify the cosign keyless signature on `checksums.txt` (proves the file was
produced by the upstream release workflow), then verify the archive against
the checksums file.

```bash
# Replace v1.0.0 with the tag you downloaded.
TAG=v1.0.0

cosign verify-blob \
  --bundle  runnerkit_${TAG#v}_checksums.txt.sigstore.json \
  --certificate-identity   "https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_${TAG#v}_checksums.txt

sha256sum -c runnerkit_${TAG#v}_checksums.txt --ignore-missing
```

A successful run prints `Verified OK` (cosign) and one `OK` line per archive
(sha256sum). If either step fails, do NOT install the binary.

Then extract and place `runnerkit` on your `PATH`:

```bash
tar -xzf runnerkit_${TAG#v}_${OS}_${ARCH}.tar.gz
sudo install -m 0755 runnerkit /usr/local/bin/runnerkit
```

Troubleshooting install verification failures: see
[docs/troubleshooting/README.md](docs/troubleshooting/README.md).
```

Hard rules locked from CONTEXT/RESEARCH:
- `--certificate-identity` MUST be the literal `https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}` (RESEARCH Pattern 3 — the workflow path AND the exact tag ref are required; branch refs would be wrong because release runs are tag-triggered).
- `--certificate-oidc-issuer` MUST be `https://token.actions.githubusercontent.com` (NOT `https://github.com/login/oauth` — that's the user-OAuth issuer for `gh auth`, NOT the Actions OIDC issuer; RESEARCH Pattern 3 anti-pattern).
- `--bundle` (NOT separate `--signature` and `--certificate`) — bundle format `.sigstore.json` is the recommended Sigstore format (RESEARCH §"State of the Art").
- The link to `docs/troubleshooting/README.md` is a forward reference to a file Plan 06-03 creates; the link will be live when Plan 06-03 lands. Do not block on it.

If `README.md` does not currently exist, create one with a minimal heading (`# RunnerKit\n\nReliable GitHub Actions self-hosted runners for solo developers.\n`) followed by the Install section above. If a quickstart link block exists, leave it intact and add the Install section above it (or the natural place after the introduction).

Do NOT add any reference to Windows, .deb, .rpm, or `go install`. Those are explicitly out of scope per D-01.
  </action>
  <verify>
    <automated>grep -q "## Install" README.md && grep -q "brew install salar/runnerkit/runnerkit" README.md && grep -q "darwin_arm64" README.md && grep -q "darwin_amd64" README.md && grep -q "linux_amd64" README.md && grep -q "linux_arm64" README.md && grep -q "cosign verify-blob" README.md && grep -q "https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/" README.md && grep -q "https://token.actions.githubusercontent.com" README.md && grep -q "runnerkit_\${TAG#v}_checksums.txt.sigstore.json" README.md && grep -q "sha256sum -c" README.md && ! grep -q "go install" README.md && ! grep -q "windows" README.md</automated>
  </verify>
  <acceptance_criteria>
    - `README.md` contains a `## Install` heading.
    - Contains literal `brew install salar/runnerkit/runnerkit`.
    - Lists all 4 supported platforms in a table: `darwin_arm64`, `darwin_amd64`, `linux_amd64`, `linux_arm64`.
    - Contains literal `cosign verify-blob` command with all four flags: `--bundle`, `--certificate-identity`, `--certificate-oidc-issuer`, and the positional checksums file argument.
    - Contains the literal cert-identity URL `https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/` (exact substring).
    - Contains the literal OIDC issuer URL `https://token.actions.githubusercontent.com` (exact substring).
    - Contains `sha256sum -c` step after the `cosign verify-blob` step.
    - References the bundle filename `runnerkit_${TAG#v}_checksums.txt.sigstore.json` (exact filename pattern matching .goreleaser.yaml).
    - Contains a link to `docs/troubleshooting/README.md` (forward reference; resolves when Plan 06-03 lands).
    - Does NOT contain `go install`, `windows`, `.deb`, or `.rpm` (out of scope per D-01/D-02).
    - Validation matrix row from `06-VALIDATION.md` line 53 ("Cosign signature verifies for issuer/identity in README") aligns: the cert-identity URL in README matches the workflow path in .github/workflows/release.yml.
  </acceptance_criteria>
  <done>README.md has a complete Install section: Homebrew tap command, 4-platform asset table, GitHub Releases download flow, literal cosign verify-blob command with the exact cert-identity URL pointing at `.github/workflows/release.yml@refs/tags/$TAG` and OIDC issuer `https://token.actions.githubusercontent.com`, sha256sum verification, and a forward link to troubleshooting docs.</done>
</task>

<task type="auto" tdd="false">
  <name>Task 4: Create docs/release-process.md (maintainer-only) covering tap repo, secret, and tag procedure</name>
  <files>docs/release-process.md</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (§"Environment Availability" missing dependencies; Pitfall 1 fork OIDC; Pitfall 2 default GITHUB_TOKEN cross-repo; Pitfall 7 live smoke billable orphan trap)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-03, D-11, D-12)
    - .goreleaser.yaml (Task 1 output — references homebrew-runnerkit tap and HOMEBREW_TAP_GITHUB_TOKEN)
    - .github/workflows/release.yml (Task 2 output — references the same secret)
  </read_first>
  <action>
Create `docs/` directory if it does not exist. Then create `docs/release-process.md` with the following EXACT content:

```markdown
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

1. On <https://github.com/settings/tokens?type=beta> create a fine-grained
   personal access token with:
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
```

Hard rules locked from CONTEXT/RESEARCH:
- The tap repo name MUST be `salar/homebrew-runnerkit` (RESEARCH §"Environment Availability"). The cask consumer command in README is `brew install salar/runnerkit/runnerkit` — that's `<owner>/<tap-without-prefix>/<formula>` per Homebrew convention; the tap repo is `salar/homebrew-runnerkit` because Homebrew strips the `homebrew-` prefix when resolving a tap.
- The OIDC issuer in the post-tag verify section MUST be the literal `https://token.actions.githubusercontent.com` (RESEARCH Pattern 3 / Pitfall 1).
- The cert-identity in the post-tag verify section MUST embed the workflow path AND the tag ref (`@refs/tags/${TAG}`) (RESEARCH Pattern 3).
- The pre-tag checklist references `make smoke-live` from Plan 06-04 (D-11) as a HARD requirement before tag push.

Stopwatch checklist details (the actual 10-minute checklist content) are added by Plan 06-04 — this file ships the structure now and Plan 06-04 fills in the BYO/Hetzner stopwatch table.
  </action>
  <verify>
    <automated>test -f docs/release-process.md && grep -q "Maintainer-Only" docs/release-process.md && grep -q "salar/homebrew-runnerkit" docs/release-process.md && grep -q "HOMEBREW_TAP_GITHUB_TOKEN" docs/release-process.md && grep -q "make smoke-live" docs/release-process.md && grep -q "https://token.actions.githubusercontent.com" docs/release-process.md && grep -q "id-token: write" docs/release-process.md && grep -q "RELEASE-NOTES-vX.Y.Z.md" docs/release-process.md && grep -q "git tag -a" docs/release-process.md && grep -q "fine-grained personal access token" docs/release-process.md</automated>
  </verify>
  <acceptance_criteria>
    - File `docs/release-process.md` exists.
    - Contains "Maintainer-Only" or equivalent disclaimer that this is not for end users.
    - Contains a "One-Time Prerequisites" section covering both: (a) creating `salar/homebrew-runnerkit` repo, (b) creating `HOMEBREW_TAP_GITHUB_TOKEN` PAT and storing it as a repo secret.
    - Contains "Pre-tag checklist" requiring `make smoke-live` (forward reference to Plan 06-04).
    - Contains the literal `git tag -a vX.Y.Z` example (or equivalent) for tagging from upstream.
    - Contains the post-tag cosign verify-blob example with the exact cert-identity URL embedding `.github/workflows/release.yml@refs/tags/${TAG}` and OIDC issuer `https://token.actions.githubusercontent.com`.
    - Contains a "Common Failures" table that includes at least: fork OIDC strip (Pitfall 1), missing tap secret 403 (Pitfall 2), missing `version: 2`.
    - Contains a forward reference to `RELEASE-NOTES-vX.Y.Z.md` (created in Plan 06-04 for v1.0.0).
    - Document length >= 60 lines (sanity check that it's a real document, not a stub): `wc -l docs/release-process.md` returns >= 60.
  </acceptance_criteria>
  <done>docs/release-process.md exists, documents one-time tap-repo + PAT creation, pre-tag smoke + stopwatch checklist (forward refs to Plan 06-04), tag procedure including warning about fork OIDC strip, post-tag cosign verify-blob example, common failures table, and the RELEASE-NOTES-vX.Y.Z.md convention.</done>
</task>

<task type="checkpoint:human-action" gate="blocking">
  <name>Task 5: Maintainer creates Homebrew tap repo + HOMEBREW_TAP_GITHUB_TOKEN secret (one-time)</name>
  <files>(none — this is a human-action checkpoint; no repo files change. The maintainer creates an external GitHub repository `salar/homebrew-runnerkit` and a `HOMEBREW_TAP_GITHUB_TOKEN` repo secret in `salar/runnerkit`.)</files>
  <action>See `<what-built>` and `<how-to-verify>` below for the maintainer-only steps Claude cannot perform (interactive PAT creation + cross-repo admin access). Maintainer creates `salar/homebrew-runnerkit` GitHub repo with `Casks/` directory + default branch `main`, then creates a fine-grained PAT scoped to that repo with `Contents: Read & write` permission, then stores it as the `HOMEBREW_TAP_GITHUB_TOKEN` secret in `salar/runnerkit` repo settings. Resume the plan with the resume-signal below once both exist (or "deferred" if intentionally postponing until first real tag).</action>
  <verify><automated>echo "checkpoint:human-action — verified by maintainer resume-signal, not automation"</automated></verify>
  <done>Either: (a) `salar/homebrew-runnerkit` repo exists AND `HOMEBREW_TAP_GITHUB_TOKEN` secret exists in `salar/runnerkit` repo settings AND maintainer typed "tap-ready"; OR (b) maintainer typed "deferred" acknowledging the v1.0.0 tag push will fail until these are resolved.</done>
  <what-built>Plan 06-01 has produced a release pipeline that depends on TWO external one-time setup actions that ONLY a human with admin access to `github.com/salar` can perform: (a) creating the `salar/homebrew-runnerkit` repository, (b) creating a fine-grained PAT scoped to that repo and storing it as the `HOMEBREW_TAP_GITHUB_TOKEN` secret in the `salar/runnerkit` repo. Without these, the first tag push will fail at the `homebrew_casks:` step with 403. Claude cannot do this — the GitHub UI requires interactive PAT creation and admin access.</what-built>
  <how-to-verify>
    1. Open https://github.com/salar/homebrew-runnerkit and confirm the repo exists, has a `Casks/` directory, and the default branch is `main`.
    2. Open https://github.com/salar/runnerkit/settings/secrets/actions and confirm a secret named `HOMEBREW_TAP_GITHUB_TOKEN` is listed.
    3. (Optional sanity check) Push a `v0.0.0-test` pre-release tag and confirm the release workflow runs end-to-end. If it fails at the cask publish step, the secret is missing or scoped incorrectly.
  </how-to-verify>
  <resume-signal>Type "tap-ready" when both the repo and the secret exist. If you choose to defer this until just before the first real tag, type "deferred" — this plan still completes (the artifacts are correct), but Plan 06-04's `make smoke-live` and the v1.0.0 tag push will fail until this is resolved.</resume-signal>
</task>

</tasks>

<verification>
Phase-level checks for Plan 06-01 completion:

1. `goreleaser check` passes locally (or in pr-checks.yml CI on the next PR).
2. `goreleaser release --snapshot --skip=publish --clean` produces all 4 platform archives + checksums.txt in `dist/`.
3. README.md contains a complete Install section with the literal cosign verify-blob command.
4. `.github/workflows/release.yml` triggers only on `v*` tag push and declares `id-token: write`.
5. `.github/workflows/pr-checks.yml` runs `goreleaser check` + snapshot + go test on every PR.
6. `docs/release-process.md` documents the maintainer prerequisites (tap repo + PAT secret).
7. The release artifact identity in README cert-identity URL matches the workflow path in `.github/workflows/release.yml` (string equality on `https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/`).

Validation matrix coverage (`06-VALIDATION.md` rows):
- Line 51 (GoReleaser config schema valid): satisfied by Task 1 + Task 2 PR CI.
- Line 52 (4 platforms + checksums + sigstore bundle produced): satisfied by Task 1 + Task 2 PR CI snapshot job.
- Line 53 (Cosign signature verifies): wired by Tasks 1+2+3; exercised on first tag push (Plan 06-04).
</verification>

<success_criteria>
- `.goreleaser.yaml` exists, schema-valid (`goreleaser check` green), produces 4-platform matrix on snapshot run.
- `.github/workflows/release.yml` exists, tag-triggered, with id-token: write + cosign-installer@v3 (v3.0.6) + goreleaser-action@v7.
- `.github/workflows/pr-checks.yml` exists, runs goreleaser check + snapshot build + go test on every PR.
- README.md has Install section with brew tap command, 4-platform asset table, exact cosign verify-blob snippet matching the cert-identity URL in the release workflow.
- docs/release-process.md exists with maintainer prerequisites and tag procedure.
- `dist/` is gitignored.
- Maintainer human-action checkpoint resolves to "tap-ready" or explicit "deferred" (deferral is acceptable; v1.0.0 tag push in Plan 06-04 will require it).
- All hard rules from `<phase_specific_guidance>` Hard rules 1, 2, 3, 4, 12 are satisfied.
</success_criteria>

<output>
After completion, create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-01-SUMMARY.md` summarizing:
- Files added (`.goreleaser.yaml`, `.github/workflows/release.yml`, `.github/workflows/pr-checks.yml`, `docs/release-process.md`).
- Files modified (`README.md`, `.gitignore`).
- Locked decisions implemented (D-01..D-05).
- Outstanding maintainer prerequisites (tap repo + PAT secret) and their state per the checkpoint resolve-signal.
- Forward references created (link from README → docs/troubleshooting/README.md created by Plan 06-03; link from docs/release-process.md → make smoke-live created by Plan 06-04).
- Confirmed runner pin location is `internal/bootstrap/package.go::RunnerVersion`, NOT `internal/bootstrap/script.go` as RESEARCH stated. (Note for downstream plans referencing the pin.)
</output>
