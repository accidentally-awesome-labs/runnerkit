# Phase 6: Release, Upgrade, Docs, and v1 Validation - Research

**Researched:** 2026-05-02
**Domain:** Go CLI distribution, supply-chain signing, state migration, operator documentation, live validation
**Confidence:** HIGH (Context7/web docs verified; tool versions registry-confirmed 2026-05-02)

## Summary

Phase 6 ships RunnerKit to external users via GitHub Releases + a Homebrew tap, layers cosign keyless signing on top of GoReleaser-produced checksums, adds a forward-only state-migration framework with side-by-side backups, formalizes `RKD-<COMPONENT>-NNN` error codes that link to stable troubleshooting anchors, and runs the two outstanding live smokes (Phase 1 GitHub permission + Phase 4 Hetzner billable) plus a 10-minute clean-machine stopwatch checklist before tagging v1.0.0.

All four CONTEXT.md decision blocks (D-01..D-17) are externally implementable with current tooling. The only sequencing risk is publishing the install instructions (Plan 06-01 docs) before the troubleshooting URLs (Plan 06-03) exist — both ship in the same release, so they must merge before the v1.0.0 tag, not before the first plan completes.

**Primary recommendation:** Use GoReleaser **v2.15.4** with `version: 2` schema, `homebrew_casks:` (not deprecated `brews:`), `signs:` block calling `cosign sign-blob --bundle=${signature} --yes`, in a tag-triggered GitHub Actions workflow with `id-token: write` and a separate `HOMEBREW_TAP_GITHUB_TOKEN` PAT for the cross-repo tap push. State migrations register at `internal/state/migrations.go` Migrate() and add a side-by-side `state.json.backup-v<N>-<ts>` next to atomic write. Lazy update check uses `https://api.github.com/repos/{owner}/{repo}/releases/latest` with ETag conditional GET, cached 24h at `$XDG_STATE_HOME/runnerkit/update-check.json`. Live smoke is a `make smoke-live` Makefile target gated on a Hetzner empty-project precheck and a deferred `GET /servers/{id}` 404 poll using the existing `hcloud-go v1.59.2` client.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Distribution and install (Plan 06-01)
- **D-01:** Two install channels: GitHub Releases binaries (always) + Homebrew tap (`brew install runnerkit`). No `go install`, `.deb`, or `.rpm`.
- **D-02:** Supported CLI host platforms: macOS arm64, macOS amd64, Linux amd64, Linux arm64. Runner host stays Linux/systemd.
- **D-03:** GoReleaser in GitHub Actions on `vX.Y.Z` tag push produces all platform binaries, the checksums file, the cosign signature, GitHub Release notes, and the Homebrew formula update in a single run. No local manual cuts.
- **D-04:** Each release ships per-platform binaries, `runnerkit_vX.Y.Z_checksums.txt` (SHA256), and a cosign keyless signature (GitHub Actions OIDC). No GPG, no SBOM in v1.
- **D-05:** README + troubleshooting docs show verification of both checksums and `cosign verify-blob`.

#### Upgrade detection and execution (Plan 06-02)
- **D-06:** Lazy update check on `up`/`status`/`doctor`. ~24h cache. Skip on no-net. Skip in JSON mode. Single non-blocking notice line. No always-on per-invocation network call.
- **D-07:** `runnerkit upgrade` does NOT self-replace. It detects channel (Homebrew vs Releases) and prints the right command (`brew upgrade runnerkit` or release-asset download flow).
- **D-08:** Pin GitHub Actions runner version per RunnerKit release. `runnerkit doctor` warns when the host runner is stale relative to the pin or when GitHub starts rejecting it. `runnerkit upgrade-runner` re-runs `bootstrap.Apply`/`ApplyEphemeral` against the host with the new pin.
- **D-09:** Forward-only auto migrations with a side-by-side backup of the original state file before migration. Atomic-write semantics. When state is read with a newer `schema_version` than the running CLI knows, refuse to mutate and exit non-zero with exact "upgrade RunnerKit" guidance. No interactive prompt.

#### Live v1 validation (Plan 06-04)
- **D-10:** Plan 06-04 lands the two outstanding live smokes (Phase 1 GitHub permission, Phase 4 Hetzner billable) AND a 10-minute clean-machine stopwatch checklist. All three required for v1 sign-off.
- **D-11:** Live smokes triggered manually via `make smoke-live` before tagging. NOT scheduled. NOT in CI on tag push. NOT requiring CI to hold real Hetzner/GitHub PAT secrets.
- **D-12:** Hetzner live smoke MUST gate on (1) empty-project precheck — refuse to run if any pre-existing `runnerkit-*` server/volume/SSH-key found; (2) deferred destroy verification — after `runnerkit destroy --yes`, every created resource ID must return 404 within N minutes, fail loudly otherwise.
- **D-13:** Validation results live in `06-VERIFICATION.md` (v1.0.0 baseline) AND `RELEASE-NOTES-vX.Y.Z.md` per release re-running the checklist.

#### Troubleshooting docs (Plan 06-03, DOC-04)
- **D-14:** `docs/troubleshooting/` per component: `auth.md`, `ssh.md`, `bootstrap.md`, `github.md`, `provider.md`, `cleanup.md`, plus `docs/troubleshooting/README.md` index.
- **D-15:** Stable error codes `RKD-<COMPONENT>-NNN` (e.g., `RKD-AUTH-001`). CLI prints `See: <docs URL>/troubleshooting/<component>#rkd-<component>-NNN` for every emitted code. Anchors stable across renames.
- **D-16:** v1 docs cover all four failure surfaces: setup (auth scope, public-repo block, SSH host-key, preflight); bootstrap and service (systemd unit, runner user, package install, online-verification timeout); operations (status drift, doctor findings, recover, down); cloud and cleanup (Hetzner quota/credentials, partial destroy, billable-resource verification). Nothing deferred.
- **D-17:** Each entry follows `Symptom → Diagnosis → Fix` with copyable commands. Short, structured, optimized for stuck users. No long prose, no flat FAQ.

### Claude's Discretion
- Exact file names, GoReleaser config layout, Homebrew tap repo structure (provided D-01..D-05 hold)
- Exact CLI flag/command spelling for `runnerkit upgrade` and `runnerkit upgrade-runner` (provided D-07/D-08 hold)
- Exact wording, polling cadence (within "lazy / ~24h cached"), cache file location for update notice (provided D-06 holds)
- Exact RKD numbering scheme per component (provided codes are stable, RKD- prefixed, URL-anchorable per D-15)
- Exact `make smoke-live` script implementation, env var names, host-side scratch directory (provided D-12 gates hold)
- Documentation hosting choice (GitHub repo only vs `runnerkit.dev` static site) — pick whichever is cheapest, provided D-15's stable anchors work in either world

### Deferred Ideas (OUT OF SCOPE)
- Windows CLI host support
- `runnerkit upgrade` self-replacing binary
- GPG signatures + SBOM
- `go install` channel
- Linux .deb / .rpm packages
- Scheduled / pre-release CI gate for live smokes
- Anonymous telemetry / usage analytics
- Doctor `--fix` automatic remediation
- Cost-control features (`COST-01..03`)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| **REL-05** | Developer can update the runner binary/service or follow a documented upgrade path that prevents immediate runner rot. | Plan 06-01 ships GoReleaser v2.15.4 binaries + Homebrew cask + cosign-signed checksums (D-01..D-05). Plan 06-02 adds lazy update check (gh-style 24h cache; verified pattern from `cli/cli/internal/update`), `runnerkit upgrade` channel-detect + print-only flow, `runnerkit upgrade-runner` reusing `bootstrap.Apply`/`ApplyEphemeral` against the runner-pin constant in `internal/bootstrap/script.go` (currently `2.334.0`, registry-confirmed latest 2026-04-21), and forward-only state migrations with side-by-side backup attaching to `internal/state/migrations.go::Migrate`. |
| **DOC-04** | Developer can read cleanup and troubleshooting guidance for common failure modes. | Plan 06-03 builds `docs/troubleshooting/` per-component files (`auth.md`, `ssh.md`, `bootstrap.md`, `github.md`, `provider.md`, `cleanup.md`) with `RKD-<COMPONENT>-NNN` stable anchors (Rust `E[NNNN].html` and Fly.io category-prefixed codes are the proven prior art). Plan 06-04's `make smoke-live` exercises the doc URLs as part of the 10-minute stopwatch validation. |
</phase_requirements>

## Project Constraints (from CLAUDE.md / GEMINI.md / canonical refs)

CLAUDE.md does not exist in the working directory. GEMINI.md (project-level) was checked but is not directly applicable to Phase 6 surface; the binding project-level constraints come from `.planning/PROJECT.md` and prior phase summaries:

- **Solo developer audience, CLI-only, GitHub Actions only, ~10 min setup target.** Release machinery and docs MUST stay lightweight; do not introduce a hosted control plane, dashboard, telemetry, or workflow YAML auto-edit (all explicitly Out of Scope in PROJECT.md).
- **Versioned non-secret JSON state with atomic writes; redactor flows through every output path.** Release notes, smoke output, and any logs published in `06-VERIFICATION.md` MUST flow through `internal/redact/` (already integrated into the renderer).
- **Pinned third-party versions discipline (Phase 2 runner 2.334.0; Phase 4 hcloud-go v1.59.2 against Go 1.22).** Phase 6 keeps the pinning discipline; the new pins are the GoReleaser version (recommend v2.15.4) and the cosign version in CI (recommend v3.0.6). Do NOT silently bump hcloud-go to v2 — go.mod targets Go 1.22 and Phase 4 deliberately chose v1.59.2.
- **Production CLI defaults use real services; fakes are test-only.** Smoke harness must compile against real `hcloud-go` and real `gh` auth in `make smoke-live`; the rest of Phase 6 stays on the existing fake adapters.
- **JSON-mode output suppresses interactive UX (Phase 1 contract).** D-06 lazy update notice MUST honor this — silent in JSON mode (existing pattern verified in `internal/cli/root.go`).
- **Plan-before-mutation everywhere.** `runnerkit upgrade` and `runnerkit upgrade-runner` MUST print the planned action before executing (or print only, per D-07).
- **`runnerkit down` is BYO; `runnerkit destroy` is cloud.** Live smoke must call the right command per fixture (D-10).
- **Phase 5 ephemeral lifecycle MUST NOT regress.** `upgrade-runner` re-runs `Apply`/`ApplyEphemeral`; the ephemeral one-shot systemd unit, finalizer, TTL timer, and `_diag` log preservation must keep working after a runner-pin bump (verified by re-running existing `internal/cli/up_ephemeral_test.go` in CI).

## Standard Stack

### Core (release pipeline)
| Library/Tool | Version | Purpose | Why Standard |
|---|---|---|---|
| **GoReleaser** | **v2.15.4** (verified `gh api` 2026-05-02; published 2026-04-21) | Tag-triggered cross-platform release builder | De-facto Go CLI release tool. v2 schema with `version: 2` required at top of `.goreleaser.yaml`. Built-in `signs:`, `homebrew_casks:`, `checksum:`. |
| **goreleaser/goreleaser-action** | **v7** | GitHub Actions wrapper | `version: "~> v2"` constraint pins to a v2.x line; provides standard inputs/`args`. |
| **sigstore/cosign** | **v3.0.6** (verified registry; published 2026-04-06) | Keyless signing of checksums via OIDC | Standard for Go-CLI supply chain; bundle format (`.sigstore.json`) is current recommended. v3.x is stable. |
| **actions/checkout** | **v4** | Repo checkout with `fetch-depth: 0` | GoReleaser needs full history for changelog. |
| **actions/setup-go** | **v5** | Go toolchain in CI | Standard. |
| **actions/runner** (downstream pin) | **v2.334.0** (verified registry; published 2026-04-21) | Runner version constant in `internal/bootstrap/script.go` | Already pinned in Phase 2; Phase 6 keeps the same value AND adds doctor staleness check. |

### Supporting (state migration + update check)
| Library | Version | Purpose | When to Use |
|---|---|---|---|
| **hashicorp/go-version** | **v1.9.0** (registry-confirmed) | Semver compare with prerelease handling | Use for "newer than" check in lazy update notice. Same library `gh` CLI uses. |
| **hetznercloud/hcloud-go** | **v1.59.2** (PINNED — do not bump to v2) | Hetzner client for live-smoke 404 verification | Phase 4 pinned this; Phase 6 reuses it. v2 has different module path (`github.com/hetznercloud/hcloud-go/v2/hcloud`); upgrading is out of scope. |
| **spf13/cobra** | **v1.10.1** (existing) | New `upgrade`, `upgrade-runner` subcommands | Same pattern as existing commands. |
| (no new dep) — Go stdlib `net/http` + `os` + `encoding/json` | n/a | Lazy update check HTTP + cache file | Avoids new dependency for a 30-line probe. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|---|---|---|
| GoReleaser | Manual `goreleaser`-less release script (Make + Go matrix) | Saves a tool dependency but doubles the maintenance burden and loses checksum/sign integration. Reject — D-03 already locks GoReleaser. |
| Homebrew Cask (`homebrew_casks:`) | Old `brews:` Formula | `brews:` is **deprecated in GoReleaser v2.10** (planned removal in v3) for binary-only projects; Casks are correct for pre-compiled binaries. Use `homebrew_casks:`. |
| cosign keyless | GPG | D-04 explicitly defers GPG. Cosign keyless via OIDC needs no key management. |
| `cosign verify-blob --certificate-identity` | `--certificate-identity-regexp` | Use exact identity (full URL with workflow path + ref) for users to verify. Regexp is for tooling that processes many tags. |
| Custom semver compare | hashicorp/go-version | Custom compare misses prerelease ordering. Use the library `gh` CLI uses. |
| `runnerkit upgrade` self-replaces binary | Print-only instruction | Self-replace requires re-verifying signatures and rolling back on partial failure — too much complexity for v1. D-07 locks print-only. |
| Interactive migration prompt | Auto-migrate forward + side-by-side backup | D-09 locks auto + backup. Interactive blocks JSON mode and breaks `--yes` flows. |

**Installation (CI side, in `.github/workflows/release.yml`):**

No `go get` is needed — GoReleaser and cosign are installed via their action wrappers:

```yaml
# in .github/workflows/release.yml
- uses: sigstore/cosign-installer@v3
  with:
    cosign-release: 'v3.0.6'
- uses: goreleaser/goreleaser-action@v7
  with:
    version: '~> v2'
    args: release --clean
```

For lazy update check, no new Go module dep is needed (use stdlib). For semver compare:

```bash
go get github.com/hashicorp/go-version@v1.9.0
```

**Version verification (already done by researcher 2026-05-02):**
```bash
curl -sf https://api.github.com/repos/goreleaser/goreleaser/releases/latest   # v2.15.4
curl -sf https://api.github.com/repos/sigstore/cosign/releases/latest         # v3.0.6
curl -sf https://api.github.com/repos/actions/runner/releases/latest          # v2.334.0
curl -sf https://api.github.com/repos/hashicorp/go-version/releases/latest    # v1.9.0
```

## Architecture Patterns

### Recommended Project Layout (additions only)

```
.
├── .github/workflows/release.yml             # NEW: tag-triggered GoReleaser run
├── .goreleaser.yaml                          # NEW: version: 2 config
├── Makefile                                  # NEW: smoke-live, smoke-live-byo, smoke-live-cloud, release-snapshot
├── cmd/runnerkit/main.go                     # MODIFIED: keep `var version = "dev"` for -ldflags injection
├── internal/
│   ├── bootstrap/
│   │   └── script.go                         # MODIFIED: bump runner pin constant if needed; expose pin string
│   ├── cli/
│   │   ├── upgrade.go                        # NEW: `runnerkit upgrade` (channel-detect, print-only)
│   │   ├── upgrade_runner.go                 # NEW: `runnerkit upgrade-runner` (re-Apply with new pin)
│   │   └── update_notice.go                  # NEW: lazy update check + cache; called from up/status/doctor
│   ├── update/                               # NEW package
│   │   ├── check.go                          # ETag GET + 24h cache; gh CLI pattern
│   │   ├── check_test.go                     # fake httptest.Server fixtures
│   │   └── version.go                        # hashicorp/go-version wrapper
│   ├── state/
│   │   ├── migrations.go                     # MODIFIED: forward-only chain, side-by-side backup
│   │   └── migrations_test.go                # NEW: v1→v2 round-trip + newer-schema-refuse
│   └── errcodes/                             # NEW package
│       ├── codes.go                          # RKD-<COMPONENT>-NNN registry + URL builder
│       └── codes_test.go                     # stable-anchor + duplicate-detection tests
└── docs/
    ├── troubleshooting/
    │   ├── README.md                         # NEW: index by component + global error-code table
    │   ├── auth.md                           # NEW: GH-permission, fine-grained token, 401/403 (RKD-AUTH-NNN)
    │   ├── ssh.md                            # NEW: host-key, key-not-found, port (RKD-SSH-NNN)
    │   ├── bootstrap.md                      # NEW: package-install, runner user, online-verification (RKD-BOOT-NNN)
    │   ├── github.md                         # NEW: registration token, runner offline, deregister stale (RKD-GH-NNN)
    │   ├── provider.md                       # NEW: HCLOUD_TOKEN, quota, region, partial destroy (RKD-PROV-NNN)
    │   └── cleanup.md                        # NEW: down/destroy, ephemeral logs, billable verification (RKD-CLEAN-NNN)
    ├── upgrade.md                            # NEW: channel detection, runner pin bump, state migration, rollback note
    └── release-process.md                    # NEW: maintainer-only — tag, smoke-live gates, RELEASE-NOTES file
```

### Pattern 1: GoReleaser v2 config (verified against goreleaser.com 2026-05-02)

**What:** Single `.goreleaser.yaml` produces all artifacts on `git push origin vX.Y.Z`.
**When to use:** This is the only release path (D-03).
**Example:**
```yaml
# Source: https://goreleaser.com/quick-start/ + customization/sign/ + homebrew_casks/
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
    artifacts: checksum   # only sign the checksums.txt file (D-04)

homebrew_casks:
  - name: runnerkit
    binaries: [runnerkit]
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

Key things the planner MUST get right:
- `version: 2` at top is **required** (v1 schema is unsupported).
- `homebrew_casks:` (NOT `brews:`) — `brews:` is deprecated as of GoReleaser v2.10.
- `signs[].artifacts: checksum` — sign ONLY `runnerkit_vX.Y.Z_checksums.txt` (D-04 limits scope; signing all archives is wasted CI minutes for users who only need checksum integrity).
- `--bundle=${signature}` — produces `.sigstore.json` (the **recommended** bundle format per Sigstore docs); pairs with `--yes` for non-interactive CI.
- Homebrew tap **must be a separate repo** (`salar/homebrew-runnerkit`) and the token MUST be a PAT (default `GITHUB_TOKEN` cannot push cross-repo). Convention: `HOMEBREW_TAP_GITHUB_TOKEN` repo secret.

### Pattern 2: Tag-triggered release workflow (verified)

```yaml
# Source: https://goreleaser.com/ci/actions/ — verified 2026-05-02
name: release
on:
  push:
    tags: ['v*']

permissions:
  contents: write   # publish release + assets
  id-token: write   # cosign keyless OIDC (REQUIRED)

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

Two non-obvious gotchas:
- The workflow MUST run from the upstream repo, not a fork PR — fork PRs strip OIDC `id-token: write`, breaking cosign keyless signing. Document in `docs/release-process.md`.
- `fetch-depth: 0` is required for GoReleaser changelog generation; missing it causes opaque "no commits found" errors.

### Pattern 3: Cosign keyless verify-blob command (the user-facing copy)

**What:** What users paste into a terminal to verify a downloaded checksums file.
**When to use:** Document in README + `docs/troubleshooting/README.md` install section per D-05.
**Example:**
```bash
# Source: https://docs.sigstore.dev/cosign/verifying/verify/ + sigstore-blog cosign-verify-bundles
# Verified against kwctl/policy-server real-world 2026 examples.
cosign verify-blob \
  --bundle  runnerkit_v1.0.0_checksums.txt.sigstore.json \
  --certificate-identity 'https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/v1.0.0' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_v1.0.0_checksums.txt
```

CRITICAL: `--certificate-oidc-issuer` is **`https://token.actions.githubusercontent.com`** (NOT `https://github.com/login/oauth` — that's the user-OAuth issuer for `gh auth`, not the Actions OIDC issuer). Older GoReleaser docs sometimes show the wrong one; the cosign CLI doc page is the authoritative source.

The `--certificate-identity` MUST embed the workflow path AND the exact tag ref (`@refs/tags/v1.0.0`), not the branch — release runs are tag-triggered and the tag-ref is what shows up in the OIDC claim.

### Pattern 4: State migration with side-by-side backup (D-09)

**What:** When `state.json` reads with `schema_version` < CURRENT, run forward migration; when > CURRENT, refuse.
**When to use:** Every CLI invocation that reads state — already centralized in `internal/state/store.go::Load`.
**Example:**
```go
// internal/state/migrations.go (replace current 16-line skeleton)
//
// Source: pattern adapted from terraform-plugin-sdk StateUpgrader chain
// (https://developer.hashicorp.com/terraform/plugin/sdkv2/resources/state-migration)
// Forward-only chain; each step receives previous state and emits next.

const SchemaVersion = "2"  // bump to 2 in Phase 6 to exercise the chain

type migrationFn func(map[string]any) (map[string]any, error)

var forwardMigrations = map[string]migrationFn{
    "1": migrateV1ToV2,
    // future: "2": migrateV2ToV3,
}

// Migrate runs forward-only migrations from state.SchemaVersion to SchemaVersion.
// On success, the original state file is preserved at <path>.backup-v<from>-<RFC3339>.
// On newer-schema-than-CLI-knows, returns ErrSchemaTooNew (exit non-zero).
func Migrate(raw []byte, path string) (State, error) {
    var probe struct{ SchemaVersion string `json:"schema_version"` }
    if err := json.Unmarshal(raw, &probe); err != nil {
        return State{}, fmt.Errorf("RKD-STATE-001: state file is not valid JSON: %w", err)
    }
    if probe.SchemaVersion == "" {
        probe.SchemaVersion = "1"
    }
    if cmpVersion(probe.SchemaVersion, SchemaVersion) > 0 {
        return State{}, ErrSchemaTooNew{Found: probe.SchemaVersion, Known: SchemaVersion}
    }

    // Side-by-side backup BEFORE any migration mutation. (D-09 atomic-write rule.)
    if probe.SchemaVersion != SchemaVersion {
        if err := os.WriteFile(
            path+".backup-v"+probe.SchemaVersion+"-"+time.Now().UTC().Format("20060102T150405Z"),
            raw, 0600); err != nil {
            return State{}, fmt.Errorf("RKD-STATE-002: backup write failed: %w", err)
        }
    }

    cur := decodeRaw(raw)
    for cmpVersion(curVersion(cur), SchemaVersion) < 0 {
        from := curVersion(cur)
        fn, ok := forwardMigrations[from]
        if !ok {
            return State{}, fmt.Errorf("RKD-STATE-003: no migration from %s", from)
        }
        next, err := fn(cur)
        if err != nil {
            return State{}, fmt.Errorf("RKD-STATE-004: migration %s failed: %w", from, err)
        }
        cur = next
    }
    var s State
    return s, decodeInto(cur, &s)
}
```

Critical contract:
- Backup file name is **side-by-side** (same directory, different extension), NOT in `/tmp` — solo dev must be able to recover by hand.
- Backup happens **before** migration runs, on the **raw bytes**, not on a parsed-then-re-encoded form (preserves any unknown future fields if the user manually downgrades).
- `ErrSchemaTooNew` MUST surface a specific exit code (proposal: `ExitStateSchemaTooNew = 7`) so JSON consumers can branch on it.
- All migration error messages embed an `RKD-STATE-NNN` code; `state.json.backup-v1-20260615T...Z` is the breadcrumb for `runnerkit doctor` to advertise.

### Pattern 5: Lazy update check (24h cache, ETag, JSON-silent) — D-06

**What:** Single non-blocking line printed before command exit on `up`/`status`/`doctor`.
**When to use:** Wrap the existing `Cmd.RunE` with a deferred `notice.MaybePrint(deps)` call. NEVER block on it.
**Example:**
```go
// internal/update/check.go
// Source: pattern verified against cli/cli/internal/update/update.go (gh CLI 2026-05-02)
// Same library (hashicorp/go-version), same 24h interval, same silent-on-error policy.

type CheckedRelease struct {
    Latest      string    `json:"latest"`
    URL         string    `json:"url"`
    PublishedAt time.Time `json:"published_at"`
    ETag        string    `json:"etag"`
    LastCheck   time.Time `json:"last_check"`
}

const cacheFile = "update-check.json" // under XDG_STATE_HOME/runnerkit/
const interval = 24 * time.Hour
const apiURL = "https://api.github.com/repos/salar/runnerkit/releases/latest"

// MaybePrint emits a single non-blocking line if a newer release exists.
// MUST be silent when:
//   - jsonOutput == true (Phase 1 contract)
//   - last check < 24h ago
//   - network error (drop silently — D-06)
//   - response says same tag as current
//   - no TTY on stderr (avoid polluting non-interactive callers — gh CLI rule)
func MaybePrint(jsonOutput bool, currentVersion string, stateDir string, errOut io.Writer) {
    if jsonOutput || os.Getenv("RUNNERKIT_NO_UPDATE_NOTIFIER") != "" || os.Getenv("CI") != "" {
        return
    }
    // … 24h cache check, conditional GET with If-None-Match, version compare …
}
```

Critical contract:
- **GitHub unauthenticated rate limit is 60 req/hr per IP** (verified 2026-05-02). With a 24h cache that's effectively zero pressure on any single user. But: a CI environment that happens to call `runnerkit status` in a tight loop could starve the user's local IP.
- **ETag conditional GET returns 304 and DOES NOT count against rate limit** — but only when the request "was made while correctly authorized with an `Authorization` header." For an UNauthenticated CLI, 304 still counts. Mitigation: 24h cache makes this irrelevant.
- **Skip in CI** (gh CLI does this). Detect: `$CI` env var set. `runnerkit status` running on the runner host itself MUST NOT print upgrade notices.
- **Cache file location:** match existing convention from `internal/state/store.go::DefaultBaseDir` — use `XDG_STATE_HOME/runnerkit/update-check.json` so it lives next to `state.json`, mode 0600.
- **Silent on JSON mode** is enforced by passing `jsonOutput` flag through; no env-var or auto-detect.

### Pattern 6: Channel-detection upgrade (D-07)

**What:** `runnerkit upgrade` prints the right command for how the binary was installed, never replaces itself.
**When to use:** User runs `runnerkit upgrade` after seeing the lazy notice from Pattern 5.
**Example:**
```go
// internal/cli/upgrade.go
// Source: flyctl pattern (https://fly.io/docs/flyctl/version-upgrade/ — fly version upgrade
// detects Homebrew install and runs `brew upgrade flyctl`). RunnerKit chooses to print
// instead of run, per D-07.

func detectChannel(execPath string) string {
    abs, err := filepath.EvalSymlinks(execPath)
    if err != nil {
        return "unknown"
    }
    // Homebrew installs symlink into /usr/local/bin or /opt/homebrew/bin
    // pointing into /opt/homebrew/Cellar/runnerkit/<ver>/bin/runnerkit
    if strings.Contains(abs, "/Cellar/runnerkit/") || strings.Contains(abs, "/Caskroom/runnerkit/") {
        return "homebrew"
    }
    return "binary"
}

// Output (human):
//   homebrew → "Run: brew upgrade runnerkit"
//   binary   → "Download the latest release: https://github.com/salar/runnerkit/releases/latest
//              Verify the cosign signature before installing — see docs/troubleshooting/README.md."
//   unknown  → "RunnerKit can't tell how this binary was installed. Run `which runnerkit`
//              and follow the channel-specific instructions in docs/upgrade.md."
```

The `--json` output for `runnerkit upgrade` returns `{ ok: true, channel: "homebrew"|"binary"|"unknown", commands: [...], current: "v0.5.0", latest: "v1.0.0" }` — never executes anything.

### Pattern 7: `runnerkit upgrade-runner` (D-08)

**What:** Re-run `bootstrap.Apply` (or `ApplyEphemeral`) against the saved `MachineRef` with the runner pin baked into the current RunnerKit binary.
**When to use:** When `runnerkit doctor` has flagged `RKD-BOOT-001` (runner version stale) and the user wants to roll forward.
**Example:**
```go
// internal/cli/upgrade_runner.go (sketch)
// Loads RepositoryState, builds bootstrap.Options exactly as `runnerkit up` does
// today, but with the new pin from internal/bootstrap/script.go::PinnedRunnerVersion.
// Calls bootstrap.Apply for persistent or bootstrap.ApplyEphemeral for ephemeral —
// both are already idempotent (Phase 2/5 contract). State.RunnerTemplateVersion
// updates only on success.
```

Critical contract:
- For ephemeral runners that have already terminated, `upgrade-runner` is a no-op with a clear message ("Ephemeral runner is one-shot; the next `runnerkit up --mode ephemeral` will use the new pin"). Do NOT re-create.
- For ephemeral runners currently waiting for a job, `upgrade-runner` MUST refuse without `--force` — bumping the runner mid-wait drops the queued runner registration. Document.
- Plan-before-mutation: print the current pin, the new pin, and the exact host commands before running.

### Pattern 8: Stable error code registry (RKD-<COMPONENT>-NNN) — D-15

**What:** Single source of truth maps error codes to (severity, message template, doc anchor URL).
**When to use:** Every user-facing error line and every doctor finding.
**Example:**
```go
// internal/errcodes/codes.go
// Source: prior art — Rust E[NNNN].html (doc.rust-lang.org/error_codes/) and
// Fly.io category-prefixed error codes (fly.io/docs/monitoring/error-codes/).

type Code struct {
    ID       string // RKD-AUTH-001
    Severity string // error|warning|info
    Title    string // short, human
    Anchor   string // anchor in component file: rkd-auth-001 (lowercase)
    File     string // troubleshooting/auth.md
}

// docsBase resolves the docs hosting at runtime. Default is the GitHub repo
// blob URL; can be overridden via RUNNERKIT_DOCS_BASE for `runnerkit.dev`
// hosting later without changing code (D-15 anchor stability).
func docsBase() string {
    if v := os.Getenv("RUNNERKIT_DOCS_BASE"); v != "" {
        return v
    }
    return "https://github.com/salar/runnerkit/blob/main/docs"
}

func URL(code Code) string {
    return fmt.Sprintf("%s/troubleshooting/%s#%s",
        docsBase(),
        strings.TrimSuffix(code.File, ".md"),
        code.Anchor)
}
```

Numbering convention (Claude's discretion per CONTEXT.md, but recommend):
- `RKD-AUTH-NNN` — GitHub auth, registration token, public-repo block (replaces `public_repo_risk` ad-hoc string)
- `RKD-SSH-NNN` — host-key, key path, port, dial
- `RKD-BOOT-NNN` — runner user create, package install, online verification, runner version stale
- `RKD-GH-NNN` — runner registration, deregister stale, runner offline
- `RKD-PROV-NNN` — Hetzner token, quota, region, partial destroy
- `RKD-CLEAN-NNN` — `down`/`destroy` partial, ephemeral log preserve
- `RKD-STATE-NNN` — JSON read, schema-too-new, migration, atomic write
- Reserve `RKD-CORE-NNN` for CLI shell errors (input required, version mismatch).

Numbering inside a component starts at 001 and grows monotonically. NEVER renumber an existing code; a deprecated code goes into a `## Deprecated codes` section with "see RKD-X-Y" forwarding.

Test contract: `TestCodesAreUnique`, `TestCodesHaveDocAnchor` (compile-time-style check that every code's `(File, Anchor)` resolves to an `<a name=>` or markdown `#anchor` in `docs/troubleshooting/`).

### Pattern 9: Live smoke harness (D-10..D-12)

**What:** `make smoke-live` runs three independent live checks: GitHub permission smoke (Phase 1 outstanding), Hetzner end-to-end (Phase 4 outstanding), 10-minute stopwatch checklist.
**When to use:** Maintainer-only, before tagging vX.Y.Z. Output goes to `RELEASE-NOTES-vX.Y.Z.md`.
**Example:**
```makefile
# Makefile (sketch)
.PHONY: smoke-live smoke-live-byo smoke-live-cloud smoke-stopwatch

smoke-live: smoke-live-byo smoke-live-cloud smoke-stopwatch ## Run all live smokes (maintainer-only).

smoke-live-byo:
	@test -n "$$RUNNERKIT_SMOKE_BYO_HOST" || { echo "RUNNERKIT_SMOKE_BYO_HOST=user@host required"; exit 2; }
	@test -n "$$RUNNERKIT_SMOKE_REPO"     || { echo "RUNNERKIT_SMOKE_REPO=owner/name required"; exit 2; }
	@gh auth status >/dev/null || { echo "gh auth not present"; exit 2; }
	./scripts/smoke/byo-permission.sh "$$RUNNERKIT_SMOKE_REPO" "$$RUNNERKIT_SMOKE_BYO_HOST"

smoke-live-cloud:
	@test -n "$$HCLOUD_TOKEN" || { echo "HCLOUD_TOKEN required"; exit 2; }
	@test -n "$$RUNNERKIT_SMOKE_REPO" || { echo "RUNNERKIT_SMOKE_REPO=owner/name required"; exit 2; }
	# Gate 1 from D-12: empty-project precheck
	./scripts/smoke/hetzner-empty-precheck.sh
	# Run end-to-end up + status + a real workflow + destroy
	./scripts/smoke/cloud-end-to-end.sh "$$RUNNERKIT_SMOKE_REPO"
	# Gate 2 from D-12: 404 verification with timeout
	./scripts/smoke/hetzner-destroy-verify.sh "$${RUNNERKIT_SMOKE_TIMEOUT:-300}"

smoke-stopwatch:
	@echo "Manual: open docs/release-process.md#stopwatch and record durations into RELEASE-NOTES-$(VER).md"
```

Hetzner empty-project precheck (the D-12 gate 1 contract):

```bash
# scripts/smoke/hetzner-empty-precheck.sh
# Lists all hcloud servers, ssh-keys, primary-ips, firewalls, and refuses
# if ANY name matches `runnerkit-*`. Uses hcloud-go v1.59.2 (the existing
# pinned client) via a tiny Go program in cmd/_smokebin/empty_precheck.go
# to keep behavior identical to the production destroy-verify code path.
go run ./cmd/_smokebin/empty_precheck
```

Hetzner destroy-verify (the D-12 gate 2 contract):

```go
// cmd/_smokebin/destroy_verify.go (sketch)
// Reuses internal/provider/hetzner Client.Server.GetByID / SSHKey.GetByID etc.
// Polls every 5s for up to TIMEOUT seconds. Asserts 404 (ErrorCodeNotFound)
// for every saved resource ID from state. Fails loudly on lingering resource.
//
// hcloud-go pattern (verified):
//   _, _, err := client.Server.GetByID(ctx, id)
//   if err != nil && hcloud.IsError(err, hcloud.ErrorCodeNotFound) { return nil /* gone */ }
```

Critical contract:
- Smoke binaries live in `cmd/_smokebin/` (the `_` prefix excludes them from `go build ./...` default outputs to avoid shipping them in releases).
- `make smoke-live` MUST NOT run from CI — explicitly documented and there's no GitHub Actions workflow that calls it.
- Output of every `runnerkit *` invocation flows through the existing `internal/redact/` redactor (already integrated in renderer) so `RELEASE-NOTES-vX.Y.Z.md` doesn't leak HCLOUD_TOKEN or registration tokens.

### Anti-Patterns to Avoid

- **DON'T use `brews:` for a binary-only project.** Deprecated in GoReleaser v2.10. Will be removed in v3. Use `homebrew_casks:`.
- **DON'T self-replace the binary on `runnerkit upgrade`.** D-07 forbids it; replicating cosign signature verification in-process is far too much complexity for v1, and partial-failure rollback is brittle on Linux/macOS sandboxes.
- **DON'T use `--certificate-oidc-issuer https://github.com/login/oauth`** for cosign verify — that's the user-OAuth issuer used by `gh auth login`, not the Actions OIDC issuer. Use `https://token.actions.githubusercontent.com`.
- **DON'T silently drop the state-too-new error.** D-09 requires non-zero exit and exact upgrade message. Silent fallback (e.g., "treat as v1") corrupts state.
- **DON'T poll the GitHub API on every CLI invocation.** 60 req/hr unauthenticated limit; will starve the local IP. 24h cache is the contract (D-06).
- **DON'T print the upgrade notice in JSON mode.** Phase 1 contract; JSON consumers parse stdout and stderr-printed notices break their pipelines.
- **DON'T put the cosign signature inside the checksums file.** Two separate artifacts: `runnerkit_v1.0.0_checksums.txt` AND `runnerkit_v1.0.0_checksums.txt.sigstore.json`.
- **DON'T assume `gh` CLI is installed for `runnerkit upgrade-runner`.** It's not — it's only assumed in setup paths where the user authenticated via `gh auth login`. Don't add new dependencies.
- **DON'T skip the empty-project precheck for the cloud smoke.** D-12 gate 1 is mandatory. A pre-existing `runnerkit-*` resource implies a stale runner and the smoke would either skip its destroy verification or destroy the wrong thing.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| Cross-platform Go binary builds | shell loop calling `GOOS=… GOARCH=… go build …` per target | **GoReleaser** | Handles ldflags, archive naming, checksums, and parallelization. Already locked by D-03. |
| Checksum file generation | `sha256sum *.tar.gz > checksums.txt` in CI | **GoReleaser `checksum:`** | Stable name template includes version; matches the format users will pipe into `sha256sum -c`. |
| Keyless signing | Custom OIDC token exchange | **cosign keyless via `signs:`** | Sigstore's transparency log + Fulcio cert chain is the standard; rolling your own is a footgun. |
| Homebrew formula write | A separate "after release" workflow that pushes a `.rb` file | **GoReleaser `homebrew_casks:`** | Built-in commit/PR/cross-repo support. Just use it. |
| Semver compare | Custom `strings.Split(".")` parser | **github.com/hashicorp/go-version** | Same library `gh` CLI uses; correctly orders prereleases. |
| Conditional HTTP cache | Custom file-watcher and stat-based change detection | **HTTP `If-None-Match` + ETag** with stdlib `net/http` | Standard, server-supported, doesn't count against rate limit when authorized. |
| Atomic file write | `ioutil.WriteFile` | **Existing `internal/state/store.go::writeAtomic`** | Already used for `state.json`; reuse for backup files and cache. |
| 404-poll loop | `for { try; sleep; if 404 break }` ad-hoc | **`hcloud.IsError(err, hcloud.ErrorCodeNotFound)`** | The library's standard pattern; avoids bug where 5xx is misread as 404. |
| Doc anchor URL builder | Hardcoded `https://...` in every CLI error message | **`internal/errcodes` package + `RUNNERKIT_DOCS_BASE` env override** | Lets us migrate from GitHub blob URL to `runnerkit.dev` later without touching every emit site. |
| Linux service installation in upgrade-runner | Custom systemctl logic | **Reuse `bootstrap.Apply` / `ApplyEphemeral`** | Already idempotent; running it again with a new package URL/SHA256 IS the upgrade. |

**Key insight:** Phase 6 is mostly **plumbing**, not new functionality. Every bullet above is "the standard tool already exists and Phase 1–5 either set up the integration point or wrote the underlying primitive." Resist the urge to write more code than necessary.

## Runtime State Inventory

> Phase 6 introduces a state schema bump (v1 → v2) and a release-tagging side effect. This category map applies.

| Category | Items Found | Action Required |
|---|---|---|
| **Stored data** | `state.json` `schema_version` field is currently `"1"`. Bumping to `"2"` requires forward migration AND a per-user side-by-side backup at `<state>.backup-v1-<RFC3339>`. | **data migration:** add `migrateV1ToV2` (likely no-op other than version bump if no field semantics change yet); **code edit:** bump `SchemaVersion = "2"` in `internal/state/schema.go` and replace the body of `internal/state/migrations.go::Migrate` per Pattern 4 above. |
| **Live service config** | None — RunnerKit's only "live service" is the systemd unit on the BYO/cloud host. The unit name and ExecStart already encode the runner version indirectly via the runner package URL/SHA256 baked into the install script when the unit was installed. `upgrade-runner` regenerates the install via `Apply`/`ApplyEphemeral`, which already overwrites the unit. | **code edit:** none beyond the existing `Apply`/`ApplyEphemeral` re-entry path. |
| **OS-registered state** | systemd unit `runnerkit-runner.service` (persistent) and `runnerkit-ephemeral.<runner>.service` + `.ttl.timer` (ephemeral) on the runner host. After `upgrade-runner`, these still point to the same unit names but reference a fresh runner package. | **code edit:** Phase 5's existing finalizer/TTL flow keeps working as long as `upgrade-runner` re-runs `ApplyEphemeral` instead of just bumping a constant. Verified in Phase 5 summary. |
| **Secrets/env vars** | `HOMEBREW_TAP_GITHUB_TOKEN` (new repo secret for tap PR). `HCLOUD_TOKEN` (Phase 4, smoke-only). No CLI-shipped secret. Cosign keyless needs no key — OIDC token is ephemeral and not stored. | **none** — secrets are CI-side only, not in code or state. |
| **Build artifacts / installed packages** | After release, the published artifacts in GitHub Releases AND the Homebrew tap repo. Local `dist/` directory from `goreleaser release --snapshot` (gitignored). | **action:** add `dist/` to `.gitignore` if not present. Verify `cmd/_smokebin/` is gitignored from binary outputs. |

**The canonical question:** *After every file in the repo is updated, what runtime systems still have the old string cached, stored, or registered?*
- Existing user state files at `~/.local/state/runnerkit/state.json` with `schema_version: "1"` — handled by Pattern 4 migration.
- Existing host runner installations pinned to runner v2.334.0 — handled by `runnerkit doctor` warning + `runnerkit upgrade-runner`.
- Homebrew taps already containing previous formula (if pre-1.0 prereleases were ever shipped) — `homebrew_casks:` handles this on each release.

## Common Pitfalls

### Pitfall 1: Forking a release workflow strips OIDC
**What goes wrong:** PR from a fork triggers the release workflow; cosign signing fails with "no token issuer."
**Why it happens:** GitHub strips `id-token: write` for fork PRs.
**How to avoid:** Workflow trigger is `on: push: tags: ['v*']` — tag pushes only happen from the main repo. Document in `docs/release-process.md` that all tags MUST be pushed from the upstream repo.
**Warning signs:** `cosign-installer` succeeds but `signs:` step fails with `unable to fetch certificate from sigstore`.

### Pitfall 2: Default GITHUB_TOKEN can't push to a separate tap repo
**What goes wrong:** `homebrew_casks:` step succeeds locally with a PAT but fails in CI with `403: Resource not accessible by integration`.
**Why it happens:** The job's default `GITHUB_TOKEN` is scoped to the current repo only.
**How to avoid:** Generate a fine-grained PAT scoped to the tap repo with `Contents: Read & write` permission; store as `HOMEBREW_TAP_GITHUB_TOKEN` repo secret; reference as `token: '{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}'` in `.goreleaser.yaml`.
**Warning signs:** Workflow logs show `github.com/salar/homebrew-runnerkit: 403 Forbidden` after the GoReleaser step starts the cask publish.

### Pitfall 3: Homebrew Cask quarantine (macOS Gatekeeper)
**What goes wrong:** User runs `brew install runnerkit` then `runnerkit up`, macOS pops the unsigned-binary warning.
**Why it happens:** Casks are supposed to be Apple-notarized. RunnerKit binaries are not (D-04 defers GPG/Apple notarization).
**How to avoid:** Either (a) accept the friction and document the unquarantine step in `docs/troubleshooting/README.md` (`xattr -d com.apple.quarantine /opt/homebrew/bin/runnerkit`), or (b) Apple-notarize the macOS binaries (out of scope per D-04).
**Warning signs:** macOS users report "macOS cannot verify that this app is free from malware" on first run.

### Pitfall 4: 24h cache file race between concurrent `runnerkit` invocations
**What goes wrong:** Two `runnerkit status` runs race; one writes the cache file partially while the other reads it; truncated JSON breaks both.
**Why it happens:** Naive `os.WriteFile`.
**How to avoid:** Reuse the existing atomic-write helper from `internal/state/store.go` (write to `<file>.tmp` and rename). Drop on read errors silently; the worst case is one extra HTTP request per 24h.
**Warning signs:** Sporadic `RKD-CORE-NNN` errors during high-frequency status polling.

### Pitfall 5: Runner pin staleness drift
**What goes wrong:** A user installs RunnerKit v0.5.0 (pinning runner v2.330.0), GitHub deprecates runner < v2.332, jobs fail with "runner too old."
**Why it happens:** RunnerKit's pin is baked into the binary; users must `runnerkit upgrade` THEN `runnerkit upgrade-runner` to roll forward.
**How to avoid:** `runnerkit doctor` MUST emit `RKD-BOOT-NNN` (recommend `RKD-BOOT-002`) when the installed runner version is older than the pinned-by-CLI version; the lazy update notice (Pattern 5) tells users a newer RunnerKit exists in the same session. Document in `docs/upgrade.md`.
**Warning signs:** GitHub workflows that target RunnerKit labels start failing with "the runner version is no longer supported."

### Pitfall 6: Schema-too-new corrupts when ignored
**What goes wrong:** User downgrades RunnerKit; new RunnerKit wrote `schema_version: 2` fields, old RunnerKit truncates them on its next save.
**Why it happens:** `json.Unmarshal` ignores unknown fields, then `json.Marshal` re-emits without them.
**How to avoid:** D-09 mandates refuse-to-mutate on newer schema. Implement before any code path that calls `store.Save`. Test: `TestStateRefuseDowngrade`.
**Warning signs:** User's saved Phase 5 ephemeral metadata silently disappears.

### Pitfall 7: Live smoke creates orphaned billable resources
**What goes wrong:** Maintainer hits Ctrl-C during cloud smoke; partial provisioning leaves a Hetzner server billing.
**Why it happens:** No cleanup trap.
**How to avoid:** Empty-project precheck (D-12 gate 1) catches it on the next run. Add a Bash `trap` in `scripts/smoke/cloud-end-to-end.sh` that invokes `runnerkit destroy --yes` on EXIT/INT/TERM, AND verify the destroy-verify gate (D-12 gate 2) caught it. Document in `docs/release-process.md`.
**Warning signs:** Hetzner monthly invoice has unexpected line items dated to a release-cut day.

### Pitfall 8: Cosign install drift across CI runs
**What goes wrong:** `sigstore/cosign-installer@v3` without a pinned `cosign-release` floats; a major cosign release that changes the bundle format breaks the workflow.
**Why it happens:** Floating action versions.
**How to avoid:** Pin `cosign-release: 'v3.0.6'` AND pin the installer action SHA (or `@v3` major). Same discipline RunnerKit applies to its own runner pin.
**Warning signs:** A release worked yesterday and fails today with no code change.

### Pitfall 9: Doc anchors break on heading rename
**What goes wrong:** Editor renames `## RKD-AUTH-001: Public repo blocked` to `## Public repository is blocked` — Markdown auto-anchor changes from `rkd-auth-001` to `public-repository-is-blocked`. Every CLI emit-site URL is now broken.
**Why it happens:** Markdown anchors are derived from heading text by default.
**How to avoid:** Use explicit anchors (HTML `<a name="rkd-auth-001"></a>` immediately above the heading) AND a build-time test (`internal/errcodes/codes_test.go::TestEveryCodeHasAnchor`) that greps each `docs/troubleshooting/<file>.md` for the exact anchor name. CI fails on rename.
**Warning signs:** Users report `404` from the URL printed by `runnerkit doctor`.

### Pitfall 10: GoReleaser `--snapshot` masks tag-only behavior
**What goes wrong:** PR-time CI runs `goreleaser release --snapshot --skip=publish`; PR is green; tag push fails on a real-tag-only step (e.g., `prerelease: auto` semantics, real registry token).
**Why it happens:** `--snapshot` skips validations that are tag-mode-only.
**How to avoid:** Run `goreleaser check` in PR CI (validates the config). Run `goreleaser release --snapshot --skip=publish` to validate the build matrix. Only `goreleaser release --clean` (no flags) on tag push. Plan 06-01 should include a non-tag CI workflow that runs `check` + `--snapshot` on every PR.
**Warning signs:** First-ever tag push to v0.1.0 fails on a step that was never exercised.

## Code Examples

Verified patterns from official sources:

### Cosign sign-blob (in CI, via GoReleaser `signs:`)
```yaml
# Source: https://goreleaser.com/customization/sign/ + cosign 3.x docs (verified 2026-05-02)
signs:
  - cmd: cosign
    signature: '${artifact}.sigstore.json'
    args:
      - sign-blob
      - '--bundle=${signature}'
      - '${artifact}'
      - '--yes'
    artifacts: checksum
```

### Cosign verify-blob (user-facing — paste into README + docs/troubleshooting/README.md)
```bash
# Source: https://docs.sigstore.dev/cosign/verifying/verify/ (verified 2026-05-02)
cosign verify-blob \
  --bundle  runnerkit_v1.0.0_checksums.txt.sigstore.json \
  --certificate-identity   'https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/v1.0.0' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_v1.0.0_checksums.txt
sha256sum -c runnerkit_v1.0.0_checksums.txt --ignore-missing
```

### GitHub Releases conditional GET (Go, stdlib)
```go
// Source: https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api
// Note: 304 saves rate limit ONLY when authorized. Unauthenticated CLI relies on the 24h cache instead.
req, _ := http.NewRequestWithContext(ctx, "GET",
    "https://api.github.com/repos/salar/runnerkit/releases/latest", nil)
req.Header.Set("Accept", "application/vnd.github+json")
if cached.ETag != "" {
    req.Header.Set("If-None-Match", cached.ETag)
}
resp, err := http.DefaultClient.Do(req)
// 304 → cached payload still valid, just bump LastCheck
// 200 → unmarshal, store ETag from resp.Header.Get("ETag")
// other → return silently (D-06 no-net policy)
```

### hcloud-go 404 verification (from Phase 4 verified pattern)
```go
// Source: https://github.com/hetznercloud/hcloud-go/blob/v1.59.2/hcloud/server.go
import hcloud "github.com/hetznercloud/hcloud-go/hcloud"

server, _, err := client.Server.GetByID(ctx, savedServerID)
if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
    return nil // gone — billable resource confirmed deleted
}
if err != nil {
    return fmt.Errorf("RKD-PROV-NNN: provider check failed: %w", err)
}
if server != nil {
    return fmt.Errorf("RKD-PROV-NNN: server %d still exists", savedServerID)
}
```

### `goreleaser check` and `--snapshot` for PR CI
```bash
# Source: https://goreleaser.com/customization/snapshots/ (verified 2026-05-02)
goreleaser check                                  # validate .goreleaser.yaml schema
goreleaser release --snapshot --skip=publish --clean  # build matrix dry-run
goreleaser release --clean                        # tag-only, real publish
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| GoReleaser v1 schema (no `version:` key) | `version: 2` required at top | GoReleaser v2.0 (mid-2024) | Phase 6 starts on v2; no migration needed. |
| `brews:` (Homebrew Formula) | `homebrew_casks:` (Homebrew Cask) | GoReleaser v2.10 (early 2026) | Use casks for binary-only RunnerKit. `brews:` is deprecated. |
| Cosign v1/v2 keyless with separate `.sig` and `.crt` artifacts | Bundle format `.sigstore.json` (single artifact) | Cosign v2.x → v3.0 | Use `--bundle=${signature}`. Verify with `--bundle`. |
| GitHub OIDC issuer `https://oauth2.sigstore.dev/auth` (Sigstore proxy) | `https://token.actions.githubusercontent.com` (direct) | GitHub Actions OIDC GA | Use the direct issuer for `--certificate-oidc-issuer`. |
| `goreleaser-action@v6` and earlier | `@v7` with `version: '~> v2'` | 2025 | Use v7 in workflow. |
| Custom semver compare | `hashicorp/go-version` v1.9.0 | Stable for years | Use it; same library `gh` uses. |
| GitHub unauth limit varied historically | **60 req/hr per IP** in 2026 | Stable | 24h cache is mandatory; Authorization-bearer + ETag would save quota but RunnerKit has no token to ship. |

**Deprecated/outdated (do NOT use):**
- `brews:` (deprecated, removal in GoReleaser v3)
- Cosign v1 `.sig` + `.crt` separate artifacts (use bundle)
- `--certificate-oidc-issuer https://github.com/login/oauth` for Actions verify (wrong issuer)
- `goreleaser-action@v5` and earlier (use v7)
- `hashicorp/go-version` v1.6.0 docs/examples (use latest v1.9.0; API is stable)

## Open Questions

1. **Should the lazy-update-notice URL be the `.releases/latest` API or the HTML `releases/latest` page?**
   - What we know: API gives JSON with `tag_name`, `published_at`, and ETag headers. HTML page is HTML and unstable.
   - What's unclear: nothing — API is the right answer; this is settled.
   - Recommendation: API. HIGH confidence.

2. **Does the Homebrew tap PR-based publish flow add measurable friction over direct commit?**
   - What we know: GoReleaser supports both. PR-based requires the maintainer to merge each release PR; direct commit is hands-off.
   - What's unclear: solo-developer workflow probably wants direct commit. PR-based is for orgs with multiple maintainers.
   - Recommendation: Direct commit for v1. Switch to PR-based later if community takes over the tap. (LOW — planner discretion per CONTEXT.md "Claude's Discretion".)

3. **Does `runnerkit upgrade-runner` need a `--check` flag that reports drift without re-running Apply?**
   - What we know: `runnerkit doctor` already shows runner version and could surface a finding.
   - What's unclear: whether users want a separate command or the doctor finding is enough.
   - Recommendation: Doctor finding only for v1; add `--check` if users ask. (LOW — planner discretion.)

4. **Should the v1 release include the `06-VERIFICATION.md` durations in the GitHub Release notes body?**
   - What we know: D-13 puts the durations in `RELEASE-NOTES-vX.Y.Z.md` in the repo.
   - What's unclear: whether to copy them into the release body too.
   - Recommendation: GoReleaser `release.header:` should `cat RELEASE-NOTES-vX.Y.Z.md` and inject — discoverable for users who land on the GitHub Release page. (MEDIUM — planner discretion.)

5. **Where does the docs site live? GitHub blob URL vs `runnerkit.dev` static site?**
   - What we know: GitHub blob URLs work today and survive the v1 release. A static site would let us host nicer pages.
   - What's unclear: whether `runnerkit.dev` is a registered domain and who pays/maintains it.
   - Recommendation: GitHub blob URLs for v1. The `RUNNERKIT_DOCS_BASE` env override (Pattern 8) lets us migrate later without changing every emit site. **Open for user confirmation** — flagged in CONTEXT.md as Claude's Discretion but the choice has long-term consequence.

6. **Should the cosign signature be on `checksums.txt` only (current plan), or on every archive?**
   - What we know: D-04 says minimum is checksums-only. Signing every archive doubles cosign sign calls and the signature artifact count.
   - What's unclear: whether downstream Homebrew or distro packagers want per-archive signatures.
   - Recommendation: checksums-only for v1 (matches `kwctl`/`policy-server` precedent). MEDIUM. Revisit if a packager asks.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|---|---|---|---|---|
| `go` | All Plans 06-01..06-04 | ✓ | 1.22 (per go.mod) | — |
| `make` | Plan 06-04 (`make smoke-live`) | ✓ (default macOS/Linux) | system | — |
| `git` | Plan 06-01 (tagging) | ✓ | system | — |
| `gh` CLI | Plan 06-04 (live GitHub permission smoke only) | maintainer-only | latest | n/a — smoke maintainer must install |
| `goreleaser` | Plan 06-01 PR `check` and `--snapshot`; CI install | optional locally; CI installs via action | v2.15.4 (registry-confirmed) | `goreleaser-action@v7` provides on CI |
| `cosign` | Plan 06-01 sign step in CI; user verify | optional locally; CI installs via action | v3.0.6 | `cosign-installer@v3` provides on CI |
| `hcloud-go` v1.59.2 | Plan 06-04 destroy-verify (existing pin) | ✓ in `go.mod` | v1.59.2 | — |
| `hashicorp/go-version` | Plan 06-02 lazy-update-check | needs `go get` | v1.9.0 | manual semver string compare (not recommended) |
| Hetzner project + `HCLOUD_TOKEN` | Plan 06-04 cloud smoke ONLY | maintainer-only | n/a | n/a — smoke maintainer must provide; mocked elsewhere |
| GitHub repo + tag-push permission | Plan 06-01 (release CI) | ✓ | n/a | — |
| Separate Homebrew tap repo `salar/homebrew-runnerkit` | Plan 06-01 (cask publish) | **TO CREATE** | n/a | none — pre-tag step |
| `HOMEBREW_TAP_GITHUB_TOKEN` repo secret | Plan 06-01 (cask publish) | **TO CREATE** | n/a | direct commit fails without it |
| `RUNNERKIT_SMOKE_BYO_HOST` (env) | Plan 06-04 BYO smoke | maintainer-only | n/a | n/a |
| `RUNNERKIT_SMOKE_REPO` (env) | Plan 06-04 smokes | maintainer-only | n/a | n/a |

**Missing dependencies with no fallback:**
- The Homebrew tap repository (`salar/homebrew-runnerkit`) does not exist yet. Plan 06-01 MUST include a setup-step (or `docs/release-process.md` precondition) to create it before the first release. This is a **one-time pre-tag action**, not part of CI.
- `HOMEBREW_TAP_GITHUB_TOKEN` repo secret. Plan 06-01 MUST document its creation as a maintainer prerequisite. Without it, the cask publish step will 403.

**Missing dependencies with fallback:**
- `goreleaser` and `cosign` locally — fallback to `--snapshot` validation in PR CI (`goreleaser-action@v7` + `cosign-installer@v3` install them).

## Validation Architecture

**Status:** `workflow.nyquist_validation: true` in `.planning/config.json` — section is REQUIRED.

### Test Framework
| Property | Value |
|---|---|
| Framework | Go's built-in `testing` (matches existing project pattern) |
| Config file | none (Go convention) |
| Quick run command | `go test ./internal/state/... ./internal/update/... ./internal/errcodes/... ./internal/cli/... -count=1` |
| Full suite command | `go test ./... -count=1 -race` |
| Live smoke (D-11) | `make smoke-live` (NOT in CI) |
| GoReleaser config validation | `goreleaser check` (in PR CI) |
| GoReleaser build matrix dry-run | `goreleaser release --snapshot --skip=publish --clean` (in PR CI) |
| Cosign verify round-trip (D-04, D-05) | CI step that runs `cosign verify-blob` against the artifact GoReleaser just signed (in release CI on tag) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|---|---|---|---|---|
| **REL-05 / D-01..D-04** | GoReleaser config schema valid | unit (config validation) | `goreleaser check` | ❌ Wave 0 — `.goreleaser.yaml` |
| REL-05 / D-01..D-04 | GoReleaser produces all 4 platform binaries + checksums + sigstore bundle | integration (CI snapshot) | `goreleaser release --snapshot --skip=publish --clean` then assert `dist/` contents | ❌ Wave 0 |
| REL-05 / D-04, D-05 | Cosign signature on checksums.txt is verifiable with the issuer/identity users see in README | integration (CI on tag, post-release) | `cosign verify-blob --bundle dist/runnerkit_*.txt.sigstore.json --certificate-identity 'https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/$TAG' --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' dist/runnerkit_*.txt` | ❌ Wave 0 — `.github/workflows/release.yml` |
| REL-05 / D-06 | Lazy update check is silent in JSON mode | unit | `go test ./internal/update -run TestMaybePrint_JSONMode_Silent` | ❌ Wave 0 — `internal/update/check_test.go` |
| REL-05 / D-06 | Lazy update check honors 24h cache | unit | `go test ./internal/update -run TestMaybePrint_HonorsCache` | ❌ Wave 0 |
| REL-05 / D-06 | Lazy update check skips on no-net | unit (httptest with rejecting transport) | `go test ./internal/update -run TestMaybePrint_NetworkError_Silent` | ❌ Wave 0 |
| REL-05 / D-06 | Lazy update check uses ETag conditional GET | unit (httptest fake) | `go test ./internal/update -run TestMaybePrint_ConditionalGET` | ❌ Wave 0 |
| REL-05 / D-07 | `runnerkit upgrade` detects Homebrew install via Cellar/Caskroom path | unit | `go test ./internal/cli -run TestUpgrade_DetectsHomebrew` | ❌ Wave 0 — `internal/cli/upgrade_test.go` |
| REL-05 / D-07 | `runnerkit upgrade` prints binary-channel command for non-Homebrew install | unit | `go test ./internal/cli -run TestUpgrade_DetectsBinaryChannel` | ❌ Wave 0 |
| REL-05 / D-07 | `runnerkit upgrade` JSON mode emits ok/channel/commands keys, no execution | unit | `go test ./internal/cli -run TestUpgrade_JSONContract` | ❌ Wave 0 |
| REL-05 / D-08 | `runnerkit upgrade-runner` re-runs `bootstrap.Apply` with new pin (persistent) | unit (fake remote.Executor) | `go test ./internal/cli -run TestUpgradeRunner_Persistent_ReAppliesWithNewPin` | ❌ Wave 0 — `internal/cli/upgrade_runner_test.go` |
| REL-05 / D-08 | `runnerkit upgrade-runner` skips terminated ephemeral with clear message | unit | `go test ./internal/cli -run TestUpgradeRunner_Ephemeral_TerminalNoOp` | ❌ Wave 0 |
| REL-05 / D-08 | `runnerkit upgrade-runner` refuses waiting ephemeral without `--force` | unit | `go test ./internal/cli -run TestUpgradeRunner_Ephemeral_WaitingRefusesWithoutForce` | ❌ Wave 0 |
| REL-05 / D-08 | `runnerkit doctor` warns on stale runner version | unit | `go test ./internal/ops -run TestDoctor_StaleRunnerVersion` | ❌ Wave 0 — extend `internal/ops/doctor.go` |
| REL-05 / D-09 | State migration runs forward-only v1→v2 | unit | `go test ./internal/state -run TestMigrate_V1ToV2_ForwardOnly` | ❌ Wave 0 — `internal/state/migrations_test.go` |
| REL-05 / D-09 | State migration writes side-by-side backup before mutation | unit | `go test ./internal/state -run TestMigrate_WritesBackupBeforeMutation` | ❌ Wave 0 |
| REL-05 / D-09 | State migration refuses-to-mutate on newer schema with exit code | unit | `go test ./internal/state -run TestMigrate_RefusesNewerSchema; go test ./internal/cli -run TestExitCodeStateSchemaTooNew` | ❌ Wave 0 |
| REL-05 / D-09 | State migration is atomic (no partial writes on crash) | unit (failpoint via fs interface) | `go test ./internal/state -run TestMigrate_Atomic` | ❌ Wave 0 |
| **DOC-04 / D-14, D-15** | Every CLI-emitted RKD code resolves to a real anchor in `docs/troubleshooting/` | unit (file-walking test) | `go test ./internal/errcodes -run TestEveryCodeHasDocAnchor` | ❌ Wave 0 — `internal/errcodes/codes_test.go` |
| DOC-04 / D-15 | RKD codes are unique across components | unit | `go test ./internal/errcodes -run TestCodesAreUnique` | ❌ Wave 0 |
| DOC-04 / D-15 | URL builder honors `RUNNERKIT_DOCS_BASE` override | unit | `go test ./internal/errcodes -run TestURL_RespectsEnvOverride` | ❌ Wave 0 |
| DOC-04 / D-16 | All four failure surfaces have at least one entry per component file | unit (markdown grep) | `go test ./internal/errcodes -run TestEachComponentHasMinimumOneEntry` | ❌ Wave 0 |
| DOC-04 / D-17 | Each component file has Symptom/Diagnosis/Fix structure for every code | unit (markdown structure check) | `go test ./internal/errcodes -run TestEntriesFollowSymptomDiagnosisFix` | ❌ Wave 0 |
| **D-10 / Phase 1 outstanding** | Live GH permission smoke succeeds against a real repo | live (manual) | `make smoke-live-byo` (requires `RUNNERKIT_SMOKE_BYO_HOST`, `RUNNERKIT_SMOKE_REPO`, `gh auth status`) | ❌ Wave 0 — `scripts/smoke/byo-permission.sh` |
| **D-10 / Phase 4 outstanding** | Live Hetzner end-to-end including destroy-verify succeeds | live (manual) | `make smoke-live-cloud` (requires `HCLOUD_TOKEN`, `RUNNERKIT_SMOKE_REPO`) | ❌ Wave 0 — `scripts/smoke/cloud-end-to-end.sh`, `cmd/_smokebin/empty_precheck`, `cmd/_smokebin/destroy_verify` |
| **D-12 gate 1** | Empty-project precheck refuses if any `runnerkit-*` resource exists | live (manual) AND unit (with fake hcloud client) | `make smoke-live-cloud` AND `go test ./cmd/_smokebin -run TestEmptyPrecheck_RefusesOnExisting` | ❌ Wave 0 |
| **D-12 gate 2** | Destroy-verify polls and asserts 404 within timeout | live (manual) AND unit (with fake hcloud client returning 404 on Nth poll) | `make smoke-live-cloud` AND `go test ./cmd/_smokebin -run TestDestroyVerify_Timeout` | ❌ Wave 0 |
| **D-13** | Stopwatch checklist captures BYO and Hetzner durations | manual (10-min stopwatch) | Maintainer follows checklist in `docs/release-process.md` and writes `RELEASE-NOTES-vX.Y.Z.md` | ❌ Wave 0 — `docs/release-process.md` |

### Sampling Rate
- **Per task commit:** quick run command above (`go test ./internal/state/... ./internal/update/... ./internal/errcodes/... ./internal/cli/... -count=1`)
- **Per wave merge:** full suite (`go test ./... -count=1 -race`) plus `goreleaser check` and `goreleaser release --snapshot --skip=publish --clean`
- **Phase gate (before tagging v1.0.0):** full suite green AND `make smoke-live` green AND `06-VERIFICATION.md` filled in

### Wave 0 Gaps (must exist before substantive plan tasks start)

- [ ] `.goreleaser.yaml` (skeleton, validates `goreleaser check`) — Plan 06-01
- [ ] `.github/workflows/release.yml` — Plan 06-01
- [ ] `.github/workflows/pr-checks.yml` (or extend existing PR workflow) running `goreleaser check` + `--snapshot --skip=publish` — Plan 06-01
- [ ] Separate repo `salar/homebrew-runnerkit` — Plan 06-01 (manual maintainer step before first release)
- [ ] `HOMEBREW_TAP_GITHUB_TOKEN` repo secret — Plan 06-01 (manual maintainer step)
- [ ] `internal/update/` package skeleton (check.go, version.go, check_test.go) — Plan 06-02
- [ ] `internal/errcodes/` package skeleton (codes.go, codes_test.go) — Plan 06-03
- [ ] `internal/state/migrations_test.go` (replaces 16-line stub with real chain test fixtures) — Plan 06-02
- [ ] `internal/cli/upgrade_test.go`, `internal/cli/upgrade_runner_test.go` — Plan 06-02
- [ ] `docs/troubleshooting/` directory with 6 files + README index (initially empty Symptom/Diagnosis/Fix templates per D-17) — Plan 06-03
- [ ] `Makefile` with `smoke-live`, `smoke-live-byo`, `smoke-live-cloud`, `smoke-stopwatch` targets — Plan 06-04
- [ ] `cmd/_smokebin/` Go programs for empty_precheck and destroy_verify — Plan 06-04
- [ ] `scripts/smoke/` shell wrappers — Plan 06-04
- [ ] `docs/release-process.md` (maintainer-only) and `docs/upgrade.md` (user-facing) — Plans 06-01 and 06-02

### Fake/Real Boundary

**Real in CI (release workflow on tag push):**
- GitHub Actions OIDC token (cosign keyless signs against the real Sigstore Fulcio + Rekor)
- GoReleaser builds against real Go compiler
- Real GitHub Releases API for asset upload
- Real Homebrew tap repo for cask commit

**Real in CI (PR workflow):**
- `goreleaser check` and `goreleaser release --snapshot --skip=publish --clean` run real GoReleaser
- `go test ./...` runs real Go test runner

**Faked in unit tests:**
- GitHub Releases API (`internal/update/`): `httptest.Server` returning fixture JSON + ETag
- Hetzner client (`cmd/_smokebin/` unit tests + `internal/provider/hetzner/` existing fakes)
- Filesystem (state migration tests): existing `t.TempDir()` pattern
- Time (24h cache tests): existing `Clock func() time.Time` injection in `internal/cli/Dependencies`
- `runnerkit upgrade` channel detection: file-path fixtures (no real symlinks needed if the test passes the abs path directly)

**Real only in `make smoke-live` (manual, maintainer-only, NOT in CI):**
- Real `gh auth` against a real GitHub repo (Phase 1 outstanding smoke)
- Real `hcloud-go` against a real Hetzner project (Phase 4 outstanding smoke)
- Real human stopwatch for 10-minute checklist (D-13)

### Coverage Map per Phase 6 Success Criterion

| Success Criterion (from ROADMAP.md) | Test layers covering it | Gap |
|---|---|---|
| **(1) Install official release + documented upgrade path** | unit (config check, snapshot, signs); integration (CI tag-mode verify-blob round-trip); live (BYO smoke installs from cask) | None — Plan 06-01 + Plan 06-04 BYO smoke close it. |
| **(2) State migration safe across releases or block with guidance** | unit (forward, backup, refuse-newer, atomic) | None — Plan 06-02 closes it. |
| **(3) Cleanup + troubleshooting docs** | unit (every-code-has-anchor, unique codes, structure check) | Manual review — content quality is human-checked, not automated. Plan 06-03 closes structurally; manual editorial pass on each file. |
| **(4) Fresh-user 10-min path + workflow run + clean up** | live (BYO + cloud + stopwatch checklist) | None — Plan 06-04 explicitly delivers this. |

### Gaps Closed by Plan 06-04

Two open notes from `STATE.md` Blockers/Concerns are explicitly closed by Phase 6:

1. **"Plan 01-02/01-04 validation note: a controlled live GitHub permission smoke remains recommended before public release."** → Closed by `make smoke-live-byo` script using `gh auth status` + a real `runnerkit up --repo $RUNNERKIT_SMOKE_REPO --host $RUNNERKIT_SMOKE_BYO_HOST` against a maintainer-controlled repo.
2. **"Phase 4 validation note: a controlled live Hetzner smoke remains recommended before public release."** → Closed by `make smoke-live-cloud` with the empty-project precheck (D-12 gate 1) and destroy-verify timeout (D-12 gate 2).

These are validated as part of Plan 06-04 sign-off and recorded in `06-VERIFICATION.md`.

## Per-Plan Research

### Plan 06-01: Release packaging, checksums, install instructions, supported-platform smoke tests

**Surface:** `.goreleaser.yaml`, `.github/workflows/release.yml`, `.github/workflows/pr-checks.yml`, README install matrix, `docs/troubleshooting/README.md` install verification snippet, the separate `salar/homebrew-runnerkit` repo.

**API/config snippets:** See Pattern 1 (GoReleaser config), Pattern 2 (release workflow), Pattern 3 (cosign verify-blob user copy).

**File/dir layout (additions):**
```
.goreleaser.yaml                       # version: 2 — required
.github/workflows/release.yml          # tag-triggered, id-token: write
.github/workflows/pr-checks.yml        # goreleaser check + --snapshot
docs/release-process.md                # maintainer doc (tap repo, secrets, smoke)
README.md                              # install matrix + cosign verify-blob snippet
docs/troubleshooting/README.md         # install verification + index
```

**Integration points:**
- `cmd/runnerkit/main.go::var version = "dev"` — already exists; GoReleaser ldflags `-X main.version={{.Version}}` injects on build.
- `internal/bootstrap/script.go` runner pin — Phase 6 keeps `2.334.0` (already current); document the bump procedure in `docs/release-process.md`.
- README install snippet, `docs/byo-quickstart.md`, `docs/cloud-quickstart.md` — must reference `brew install salar/runnerkit/runnerkit` and the GitHub Releases tarball flow.

**Sequencing risks:**
- **Critical:** First-ever release fails if `salar/homebrew-runnerkit` doesn't exist yet or `HOMEBREW_TAP_GITHUB_TOKEN` is missing. Plan 06-01 MUST land docs/release-process.md AND verify the tap repo + secret exist BEFORE any tag is pushed. Recommend: a `pre-release-checklist` section in `docs/release-process.md`.
- The cosign keyless signing requires the workflow to run from the upstream repo (not a fork PR). Easy to miss until first release.
- `version: 2` at top of `.goreleaser.yaml` is non-negotiable; missing it produces "unsupported config version" errors that are not obvious to a new maintainer.

**Plan 06-01 outputs (proposed task split):**
1. Add `.goreleaser.yaml` v2 schema with builds, archives, checksum, signs, homebrew_casks, release, snapshot, changelog. Verify with `goreleaser check`. Run `goreleaser release --snapshot --skip=publish --clean` locally to assert all 4 binaries build.
2. Add `.github/workflows/release.yml` (tag trigger) and `.github/workflows/pr-checks.yml` (`check` + `--snapshot`). Pin action versions.
3. Update `README.md` install matrix (Homebrew + Releases) AND copy-paste cosign verify-blob snippet (D-05).
4. Add `docs/release-process.md` (maintainer-only): tap-repo creation, `HOMEBREW_TAP_GITHUB_TOKEN` setup, tag procedure, smoke gates.
5. (Manual maintainer step) Create `salar/homebrew-runnerkit` repo + `HOMEBREW_TAP_GITHUB_TOKEN` secret; record in `docs/release-process.md` as a precondition.

### Plan 06-02: Runner/CLI upgrade workflow, state migrations, compatibility checks, rollback guidance

**Surface:** New `internal/update/` package, new `internal/cli/upgrade.go` and `internal/cli/upgrade_runner.go`, new `internal/cli/update_notice.go` (or inline in root.go), expanded `internal/state/migrations.go`, expanded `internal/ops/doctor.go` for stale-runner finding, `docs/upgrade.md`.

**API/config snippets:** See Pattern 4 (state migration), Pattern 5 (lazy update check), Pattern 6 (channel-detect upgrade), Pattern 7 (upgrade-runner).

**File/dir layout (additions):**
```
internal/update/
  check.go                            # 24h cache + ETag + version compare
  check_test.go                       # httptest fixtures
  version.go                          # hashicorp/go-version wrapper
internal/cli/
  upgrade.go                          # `runnerkit upgrade` (channel-detect, print-only)
  upgrade_test.go
  upgrade_runner.go                   # `runnerkit upgrade-runner`
  upgrade_runner_test.go
  update_notice.go                    # MaybePrint hook for up/status/doctor
internal/state/
  migrations.go                       # forward-only chain + side-by-side backup (REPLACE current 16-line stub)
  migrations_test.go                  # NEW
docs/upgrade.md                       # user-facing
```

**Integration points:**
- `cmd/runnerkit/main.go` — no changes; the `var version` is already there.
- `internal/cli/root.go` — register `newUpgradeCommand`, `newUpgradeRunnerCommand` alongside the existing `up`/`status`/`doctor`/etc.
- `internal/cli/up.go::runUp`, `internal/cli/status.go::runStatus`, `internal/cli/doctor.go::runDoctor` — append a single `defer update.MaybePrint(...)` (or `Cmd.PostRunE`) call. Must NOT block. Must respect `*jsonOutput`.
- `internal/state/store.go::Load` — already calls `Migrate(state)`; the planner replaces the body of `Migrate` per Pattern 4. The `(state State, error)` signature stays unchanged for backward compatibility with existing callers.
- `internal/state/schema.go::SchemaVersion` — bump from `"1"` to `"2"`. The first migration (`migrateV1ToV2`) can be a no-op identity if no field semantics change in v2; the migration framework existence + the side-by-side backup is what REL-05 actually needs. (Optional: combine with EphemeralMetadata canonicalization or other v2 cleanups if Phase 5 left any open items — Phase 5 summary says no.)
- `internal/bootstrap/script.go` — expose the runner pin as a `const PinnedRunnerVersion = "2.334.0"` symbol that `runnerkit doctor` reads for the staleness check. Currently only used inside the `script.go` rendering; Phase 6 promotes it to package-level usage.
- `internal/ops/doctor.go::BuildDoctorReport` — add `runner_version_stale` finding when observed runner version < `bootstrap.PinnedRunnerVersion`. Use `RKD-BOOT-002`.

**Sequencing risks:**
- The schema bump v1→v2 is hard to reverse once shipped; users running v0.x would see their state files migrated. If Phase 6 is the FIRST tagged release, no users exist yet and there's no risk. If unreleased dev builds have any fielded users, plan a deprecation note. (None expected.)
- `runnerkit upgrade-runner` against a Phase 5 ephemeral runner currently mid-job is risky (re-Apply replaces unit; running job dies). Refuse without `--force` per Pitfall above.
- Lazy update check ordering: it MUST come AFTER the actual command's output, otherwise human users will wonder why the notice prints before "Setup complete." Use `cmd.PostRunE` or `defer` after rendering, NOT before.

**Plan 06-02 outputs (proposed task split):**
1. State migration framework: bump `SchemaVersion`, replace `Migrate` body with the forward-only chain + side-by-side backup, add `ErrSchemaTooNew`, add `ExitStateSchemaTooNew`, write tests.
2. `internal/update/` package + `update_notice.go` integration into up/status/doctor; tests with `httptest.Server`.
3. `runnerkit upgrade` (channel-detect) + `runnerkit upgrade-runner` (re-Apply with new pin); tests.
4. `runnerkit doctor` `RKD-BOOT-002` stale-runner finding; tests.
5. `docs/upgrade.md` user-facing copy.

### Plan 06-03: Troubleshooting, cleanup, recovery, and common-failure documentation

**Surface:** New `internal/errcodes/` package, new `docs/troubleshooting/` directory (6 files + README index), refactored CLI error emits to call `errcodes.URL(...)` instead of hardcoded URLs.

**API/config snippets:** See Pattern 8 (RKD code registry).

**File/dir layout (additions):**
```
internal/errcodes/
  codes.go                            # registry of all RKD codes
  codes_test.go                       # unique + anchor + URL-override tests
docs/troubleshooting/
  README.md                           # index + global error-code table
  auth.md                             # RKD-AUTH-NNN
  ssh.md                              # RKD-SSH-NNN
  bootstrap.md                        # RKD-BOOT-NNN
  github.md                           # RKD-GH-NNN
  provider.md                         # RKD-PROV-NNN
  cleanup.md                          # RKD-CLEAN-NNN
docs/upgrade.md                       # cross-link from RKD-STATE-NNN entries
```

Each `docs/troubleshooting/<component>.md` follows the structure (D-17):

```markdown
<a name="rkd-auth-001"></a>
## RKD-AUTH-001: Public repository persistent runner blocked

**Severity:** error
**Component:** auth

### Symptom
`runnerkit up --repo owner/public-repo` fails with:
```
RKD-AUTH-001: persistent runner on a public repository is blocked.
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth#rkd-auth-001
```

### Diagnosis
Persistent self-hosted runners on public repositories let any PR contributor execute code on the runner host. RunnerKit blocks this by default.

### Fix
Use ephemeral cloud:
```bash
runnerkit up --repo owner/public-repo --mode ephemeral --cloud hetzner
```
Or accept the risk explicitly (NOT recommended):
```bash
runnerkit up --repo owner/public-repo --allow-public-repo-risk
```
Read [docs/safety.md](../safety.md) before allowing.
```

**Integration points:**
- Every CLI emit site that currently has a hardcoded "See: …" URL must be refactored to `errcodes.URL(errcodes.AuthPublicRepoBlocked)` (a typed constant from the new package). Touched files (from grep over `internal/cli/*` for "See: " or hardcoded `runnerkit-*` failure strings): `up.go`, `up_byo` paths, `status.go`, `doctor.go`, `recover.go`, `down.go`, `destroy.go`, `state.go`. Phase 5 already established the convention of typed copy constants (`DangerousPersistentOverrideCopy`, `PublicRepoRiskNextAction`); the RKD layer formalizes it.
- `internal/redact/` — UNAFFECTED. The redactor already runs on all renderer output; RKD codes aren't sensitive.
- README + `docs/byo-quickstart.md` + `docs/cloud-quickstart.md` + `docs/safety.md` — cross-link to the new troubleshooting files where the user is most likely to land. Don't duplicate content; link.

**Sequencing risks:**
- Refactoring hardcoded URLs in CLI emit sites is a wide-touch refactor. Recommend doing it as a single task with a comprehensive test that asserts no `See:` lines remain hardcoded outside of `internal/errcodes/`.
- Markdown auto-anchors break on heading rename (Pitfall 9). MUST use explicit `<a name="...">` anchors AND a test that walks the docs directory.
- The four failure surfaces from D-16 (setup, bootstrap+service, operations, cloud+cleanup) imply MINIMUM coverage. Don't over-engineer initial entries; ship with one entry per known production failure and grow from there. Use `internal/ops/doctor.go` finding IDs as the seed list (they enumerate the failures the CLI already detects).

**Plan 06-03 outputs (proposed task split):**
1. `internal/errcodes/` package with code registry, URL builder, env override; tests for uniqueness, anchor existence, env override.
2. Six `docs/troubleshooting/*.md` files + `docs/troubleshooting/README.md` index, each entry following Symptom/Diagnosis/Fix.
3. Refactor CLI emit sites to call `errcodes.URL`; cross-link from README/quickstarts.
4. `internal/ops/doctor.go` finding IDs aligned to RKD codes (e.g., `runner_offline` → `RKD-GH-001`).

### Plan 06-04: End-to-end v1 validation across BYO, cloud, persistent, ephemeral, status, doctor, cleanup paths

**Surface:** `Makefile`, `scripts/smoke/`, `cmd/_smokebin/`, `06-VERIFICATION.md`, first `RELEASE-NOTES-v1.0.0.md`.

**API/config snippets:** See Pattern 9 (live smoke harness).

**File/dir layout (additions):**
```
Makefile                              # smoke-live, smoke-live-byo, smoke-live-cloud, smoke-stopwatch
scripts/smoke/
  byo-permission.sh                   # Phase 1 outstanding smoke
  cloud-end-to-end.sh                 # Phase 4 outstanding smoke
  hetzner-empty-precheck.sh           # wraps cmd/_smokebin/empty_precheck
  hetzner-destroy-verify.sh           # wraps cmd/_smokebin/destroy_verify
cmd/_smokebin/
  empty_precheck/main.go              # uses hcloud-go v1.59.2
  destroy_verify/main.go              # uses hcloud-go v1.59.2
docs/release-process.md               # already from 06-01 — extend with smoke + stopwatch checklist
.planning/phases/06-release-upgrade-docs-and-v1-validation/
  06-VERIFICATION.md                  # v1.0.0 baseline durations + cost
RELEASE-NOTES-v1.0.0.md               # per-release file (D-13)
```

**Integration points:**
- `cmd/_smokebin/empty_precheck/main.go` — uses `internal/provider/hetzner.NewClient(token)` (the same client production uses). This guarantees the empty-project check sees the same resources as the production destroy path.
- `cmd/_smokebin/destroy_verify/main.go` — same client; reuses `hcloud.IsError(err, hcloud.ErrorCodeNotFound)` per Pattern 9 example. Imports the saved `state.RepositoryState` from the temp state dir the smoke creates.
- `scripts/smoke/cloud-end-to-end.sh` — sets `RUNNERKIT_STATE_DIR` to a temp dir to isolate the smoke from the maintainer's real state file. Uses `runnerkit destroy --yes` (the existing command) for cleanup, then invokes the `cmd/_smokebin/destroy_verify` to assert 404.
- The smoke result captured into `RELEASE-NOTES-v1.0.0.md` MUST flow through `internal/redact/` — easiest to do by piping `runnerkit *` output through nothing (renderer already redacts) and then `tee`-ing into the file.
- `06-VERIFICATION.md` — the artifact `/gsd:verify-work` produces. Plan 06-04 fills in the table headers; the maintainer fills in real numbers when running the smoke.

**Sequencing risks:**
- **Billable risk:** the cloud smoke creates a Hetzner server. The empty-project precheck (D-12 gate 1) blocks if a previous run left orphans. `trap` in the shell wrapper catches Ctrl-C and invokes `runnerkit destroy --yes` before exit.
- The cloud smoke depends on a real GitHub repo to register against. The repo MUST be a maintainer-controlled trusted repo (NOT a public repo with active PRs). Document in `docs/release-process.md`.
- The 10-minute stopwatch is subjective; record times in `RELEASE-NOTES-v1.0.0.md` honestly. The 10-minute claim is the load-bearing PROJECT.md promise — **measure, don't estimate**.

**Plan 06-04 outputs (proposed task split):**
1. `Makefile` + `scripts/smoke/*` shell wrappers with env-var precondition checks.
2. `cmd/_smokebin/empty_precheck` and `cmd/_smokebin/destroy_verify` Go programs + unit tests with fake `hcloud.Client`.
3. `docs/release-process.md` smoke + stopwatch sections.
4. (Maintainer step, post-merge of plans 06-01..06-03) Run `make smoke-live`, fill in `06-VERIFICATION.md` AND `RELEASE-NOTES-v1.0.0.md`, then tag `v1.0.0`.

### Cross-Plan Dependencies & Sequencing

The orchestrator can parallelize three of the four plans; one is gated:

```
Wave A (parallel):
  Plan 06-01 (release packaging)
  Plan 06-02 (upgrade workflow + state migrations)
  Plan 06-03 (troubleshooting docs + RKD codes)

Wave B (gated on Wave A — manual maintainer step):
  Plan 06-04 (live smokes + 10-min stopwatch + tag v1.0.0)
```

Why Plan 06-04 is gated:
- It validates the artifacts Plan 06-01 produces (Homebrew install path, cosign verify-blob copy in README).
- It exercises the upgrade flow Plan 06-02 implements (lazy update notice, `upgrade-runner` flow).
- It links to the troubleshooting URLs Plan 06-03 produces (`runnerkit doctor` output + `RELEASE-NOTES-v1.0.0.md` references).
- And it's the final pre-tag gate that actually creates billable resources, so it MUST come last.

Why 06-01, 06-02, 06-03 can parallelize:
- 06-01's surface is `.goreleaser.yaml`, `.github/workflows/`, README install matrix, `docs/release-process.md`. It does NOT touch `internal/`.
- 06-02's surface is `internal/update/`, `internal/cli/upgrade*.go`, `internal/state/migrations.go`, `internal/ops/doctor.go`. It does NOT touch `.github/workflows/` or `docs/troubleshooting/`.
- 06-03's surface is `internal/errcodes/`, `docs/troubleshooting/`, and a refactor of CLI emit sites. **One overlap:** the doctor finding ID alignment (06-03 task 4) touches `internal/ops/doctor.go` which 06-02 also touches for the stale-runner finding. Recommend: 06-02 lands first OR they merge in a single rebase. Conflict surface is small (different finding IDs, different functions in the same file).

The CLI emit-site refactor in Plan 06-03 task 3 is a wide-touch (every error path). Recommend that task 3 land LAST inside Plan 06-03 to avoid stomping in-progress 06-02 work.

## Sources

### Primary (HIGH confidence)
- **GoReleaser docs** — https://goreleaser.com/quick-start/, /customization/sign/, /customization/checksum/, /customization/release/, /customization/snapshots/, /customization/build/, /customization/homebrew_casks/, /ci/actions/, /deprecations/. Verified 2026-05-02. Locked: `version: 2`, `homebrew_casks:` (not `brews:`), `signs:` block syntax with `--bundle=${signature}`.
- **Sigstore cosign docs** — https://docs.sigstore.dev/cosign/signing/signing_with_blobs/, /cosign/verifying/verify/. Verified 2026-05-02. Locked: bundle (`.sigstore.json`) is recommended; `--certificate-oidc-issuer https://token.actions.githubusercontent.com`.
- **GitHub REST API docs** — https://docs.github.com/en/rest/releases/releases (GET /releases/latest response shape), https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api (ETag/If-None-Match, 304-doesn't-count-when-authorized), https://docs.github.com/en/rest/overview/rate-limits-for-the-rest-api (60 req/hr unauth). Verified 2026-05-02.
- **gh CLI source code** — https://github.com/cli/cli/blob/trunk/internal/update/update.go (24h cache, hashicorp/go-version, silent on error, skip-CI). Verified 2026-05-02.
- **GitHub Releases registry probe** (`gh api`/curl) — verified 2026-05-02 the current latest versions: GoReleaser v2.15.4, cosign v3.0.6, actions/runner v2.334.0, hashicorp/go-version v1.9.0, hetznercloud/hcloud-go v2.39.0 (RunnerKit pinned at v1.59.2 — kept).
- **hetznercloud/hcloud-go source** — https://github.com/hetznercloud/hcloud-go/blob/main/hcloud/server.go. Verified the `IsError(err, ErrorCodeNotFound)` pattern. Phase 4 already uses this.
- **RunnerKit code** — `internal/state/schema.go` (SchemaVersion="1"), `internal/state/store.go` (atomic write contract, Migrate dispatch), `internal/state/migrations.go` (16-line stub to replace), `internal/bootstrap/install.go` (Apply/ApplyEphemeral idempotence), `internal/bootstrap/script.go` (runner pin), `internal/cli/root.go` (Dependencies, Cobra command tree, Clock injection seam), `internal/cli/up.go::runUp`, `internal/cli/status.go::runStatus`, `internal/cli/doctor.go::runDoctor` (emit hooks), `internal/cli/destroy.go` + `internal/provider/hetzner/provision.go` (destroy primitives reused by smoke), `cmd/runnerkit/main.go` (`var version = "dev"` ldflags slot). Read 2026-05-02.
- **CONTEXT.md** — `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md`. Decisions D-01..D-17 are authoritative.
- **PROJECT.md, ROADMAP.md, REQUIREMENTS.md, STATE.md** — Phase 6 success criteria, REL-05/DOC-04 binding, blockers list (Phase 1 + Phase 4 outstanding live smokes).
- **Phase 5 SUMMARY files** (`05-01-SUMMARY.md`, `05-02-SUMMARY.md`) — Ephemeral lifecycle constraints `upgrade-runner` must not regress.

### Secondary (MEDIUM confidence)
- **Carlos Becker GoReleaser blog** — https://carlosbecker.com/posts/goreleaser-v2.14/, https://goreleaser.com/blog/goreleaser-v2.10/, https://goreleaser.com/blog/goreleaser-v2.14/ (2026 release notes confirming homebrew_casks rollout + brews deprecation).
- **flyctl docs** — https://fly.io/docs/flyctl/version-upgrade/ (channel detection prior art for `runnerkit upgrade`).
- **Sigstore community blog** — https://blog.sigstore.dev/cosign-verify-bundles/ (real-world verify-blob examples for kwctl/policy-server).
- **kubewarden verification docs** — https://docs.kubewarden.io/tutorials/verifying-kubewarden (concrete `cosign verify-blob --bundle … --certificate-identity … --certificate-oidc-issuer https://token.actions.githubusercontent.com` real-world pattern).
- **Rust error code index** — https://doc.rust-lang.org/error_codes/E0001.html (prior art for `RKD-<COMPONENT>-NNN.html`-style stable URL convention).
- **Fly.io error codes index** — https://fly.io/docs/monitoring/error-codes/ (prior art for category-prefixed error codes).
- **Terraform state migration** — https://developer.hashicorp.com/terraform/plugin/sdkv2/resources/state-migration (forward-only StateUpgrader chain prior art).
- **GoReleaser v2 announcement** — https://goreleaser.com/blog/goreleaser-v2/ (v2 schema rationale).

### Tertiary (LOW confidence — not relied on for any binding claim)
- General WebSearch summarizations of older blog posts about GoReleaser pre-v2 syntax and pre-v3 cosign — flagged and explicitly excluded from the recommendations above.

## Metadata

**Confidence breakdown:**
- Standard stack (GoReleaser v2.15.4, cosign v3.0.6, runner v2.334.0, hashicorp/go-version v1.9.0, hcloud-go v1.59.2): **HIGH** — verified against GitHub Releases registry on 2026-05-02 AND cross-referenced with official docs.
- Architecture patterns (8 patterns above): **HIGH** — Pattern 1, 2, 3, 4, 5 verified against current docs and existing prior art (gh CLI, terraform). Pattern 6 (channel detect) and Pattern 7 (upgrade-runner) are HIGH because they reuse existing RunnerKit primitives. Pattern 8 (RKD codes) is HIGH on the prior art (Rust E[NNNN], Fly.io categories) and MEDIUM on the exact numbering choice (Claude's discretion). Pattern 9 (smoke harness) is HIGH on the hcloud-go API contract and MEDIUM on the shell-script orchestration (multiple workable shapes; D-12 gates are the load-bearing parts).
- Pitfalls: **HIGH** — every pitfall is grounded in verified docs or RunnerKit code paths.
- Validation Architecture: **HIGH** — every test maps to existing Go test conventions in this repo; live smoke per D-10..D-12.

**Research date:** 2026-05-02
**Valid until:** 2026-06-02 (30 days for the stable parts: GoReleaser/cosign config schema, hcloud-go API). Lazy-update-check rate-limit numbers may change — re-verify if planning slips beyond 30 days. Re-verify GoReleaser version and runner pin before tagging.

## RESEARCH COMPLETE

**Phase:** 6 — Release, Upgrade, Docs, and v1 Validation
**Confidence:** HIGH

### Key Findings
- **GoReleaser v2.15.4 with `version: 2` schema, `homebrew_casks:` (NOT deprecated `brews:`), and `signs[].artifacts: checksum`** is the correct release pipeline shape for D-01..D-05. `goreleaser-action@v7` with `version: '~> v2'` is the workflow wrapper.
- **Cosign keyless OIDC issuer is `https://token.actions.githubusercontent.com`** (NOT `https://github.com/login/oauth`); the user verify-blob command needs `--bundle <file>.sigstore.json --certificate-identity 'https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/vX.Y.Z' --certificate-oidc-issuer 'https://token.actions.githubusercontent.com'`.
- **GitHub unauthenticated REST rate limit is 60 req/hr per IP**; the lazy-update-check (D-06) must use a 24h cache (matching `gh` CLI's exact pattern in `cli/cli/internal/update/update.go`), be silent in JSON mode, silent on network error, and silent in CI. ETag/If-None-Match doesn't save quota for unauthenticated calls — the cache is the only mitigation.
- **State migrations attach to `internal/state/migrations.go::Migrate`**; replace the 16-line stub with a forward-only chain that writes a side-by-side `state.json.backup-v<N>-<RFC3339>` BEFORE mutation, refuses-to-mutate on newer schema with a dedicated exit code, and bumps `SchemaVersion` from `"1"` to `"2"` (the first migration can be a no-op identity if no field semantics change).
- **`runnerkit upgrade-runner` is a thin re-entry into `bootstrap.Apply`/`ApplyEphemeral`** with the new pin from `internal/bootstrap/script.go::PinnedRunnerVersion` (currently 2.334.0, registry-confirmed latest 2026-04-21). Both paths are already idempotent (Phase 2/5 contract). Refuse without `--force` when an ephemeral runner is currently waiting for a job.
- **RKD-<COMPONENT>-NNN error codes follow Rust `E[NNNN].html` and Fly.io category-prefix prior art**; implement as a typed registry in `internal/errcodes/` with a `RUNNERKIT_DOCS_BASE` env override so the docs hosting can migrate from GitHub blob URLs to a static site later without touching every emit site. Use explicit HTML `<a name="rkd-…">` anchors in `docs/troubleshooting/<component>.md` (Markdown auto-anchors break on heading rename).
- **Live smoke (`make smoke-live`) is maintainer-only, NOT in CI**; gate cloud smoke on (1) Hetzner empty-project precheck (refuse on any pre-existing `runnerkit-*` resource) and (2) post-destroy 404 polling using the existing `hcloud-go v1.59.2` `hcloud.IsError(err, hcloud.ErrorCodeNotFound)` pattern. Keep smoke binaries in `cmd/_smokebin/` (the `_` prefix excludes from `go build ./...`).
- **Three plans (06-01, 06-02, 06-03) parallelize with one shared file conflict** in `internal/ops/doctor.go` (06-02 adds stale-runner finding; 06-03 aligns finding IDs to RKD codes). Plan 06-04 is gated on the other three completing because it exercises their artifacts.

### File Created
`/Users/salar/Projects/spool/.planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md`

### Confidence Assessment
| Area | Level | Reason |
|---|---|---|
| Standard Stack | HIGH | All 5 tool versions registry-confirmed 2026-05-02 against `api.github.com/repos/.../releases/latest`. |
| Architecture | HIGH | Patterns 1–5 verified against current official docs + gh CLI/terraform prior art. Patterns 6–9 reuse RunnerKit primitives or have D-* contracts dictating shape. |
| Pitfalls | HIGH | Every pitfall grounded in verified docs or RunnerKit code paths read 2026-05-02. |
| Validation Architecture | HIGH | Test layers map to existing Go test conventions; live smoke gates derive from D-10..D-12 directly. |

### Open Questions (for user / planner)
- Documentation hosting: GitHub blob URLs (default, zero-cost, recommended) vs `runnerkit.dev` static site (deferrable, env-overridable).
- Per-archive cosign signatures vs checksums-only (D-04 says checksums-only minimum; planner discretion to extend).
- `runnerkit upgrade-runner --check` flag (defer; doctor finding likely sufficient for v1).
- GitHub Release body content: cat `RELEASE-NOTES-vX.Y.Z.md` into `release.header:` so users see durations on the release page.

### Ready for Planning
Research complete. Planner can now create PLAN.md files for 06-01 (release), 06-02 (upgrade), 06-03 (troubleshooting), and 06-04 (validation), with parallel-Wave-A / serial-Wave-B sequencing as documented above.
