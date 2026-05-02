---
phase: 06
slug: release-upgrade-docs-and-v1-validation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-02
---

# Phase 06 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `06-RESEARCH.md` §"Validation Architecture".

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go's built-in `testing` (matches existing project pattern) |
| **Config file** | none (Go convention) |
| **Quick run command** | `go test ./internal/state/... ./internal/update/... ./internal/errcodes/... ./internal/cli/... -count=1` |
| **Full suite command** | `go test ./... -count=1 -race` |
| **GoReleaser config validation** | `goreleaser check` (in PR CI) |
| **GoReleaser dry-run** | `goreleaser release --snapshot --skip=publish --clean` (in PR CI) |
| **Cosign verify round-trip** | CI step on tag, post-release: `cosign verify-blob` against artifact GoReleaser just signed |
| **Live smoke (D-11, manual)** | `make smoke-live` (NOT in CI) |
| **Estimated runtime — unit** | ~30 s |
| **Estimated runtime — full + race** | ~90 s |
| **Estimated runtime — snapshot** | ~120 s |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/state/... ./internal/update/... ./internal/errcodes/... ./internal/cli/... -count=1`
- **After every plan wave:** `go test ./... -count=1 -race` plus `goreleaser check` and `goreleaser release --snapshot --skip=publish --clean`
- **Before `/gsd:verify-work`:** Full suite green AND (if applicable) `make smoke-live` green
- **Phase gate (before tagging v1.0.0):** Full suite green AND `make smoke-live` green AND `06-VERIFICATION.md` filled in
- **Max feedback latency:** ~30 s (quick run) / ~90 s (wave merge)

---

## Per-Task Verification Map

Tasks below are derived from RESEARCH "Phase Requirements → Test Map". `Plan` and `Wave` columns are PROVISIONAL — the planner fills exact task IDs against the final PLAN.md task numbering. Every requirement-bearing task in the final plans MUST map back to a row here (or extend it).

| Req / Decision | Plan | Wave | Behavior | Test Type | Automated Command | File Exists | Status |
|---|---|---|---|---|---|---|---|
| REL-05 / D-01..D-04 | 06-01 | A | GoReleaser config schema valid | unit (config validation) | `goreleaser check` | ❌ W0 — `.goreleaser.yaml` | ⬜ pending |
| REL-05 / D-01..D-04 | 06-01 | A | All 4 platforms + checksums + sigstore bundle produced | integration (CI snapshot) | `goreleaser release --snapshot --skip=publish --clean` then assert `dist/` contents | ❌ W0 | ⬜ pending |
| REL-05 / D-04, D-05 | 06-01 | A | Cosign signature on checksums.txt verifies for issuer/identity in README | integration (CI on tag) | `cosign verify-blob --bundle dist/runnerkit_*.txt.sigstore.json --certificate-identity 'https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/$TAG' --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' dist/runnerkit_*.txt` | ❌ W0 — `.github/workflows/release.yml` | ⬜ pending |
| REL-05 / D-06 | 06-02 | A | Lazy update check silent in JSON mode | unit | `go test ./internal/update -run TestMaybePrint_JSONMode_Silent` | ❌ W0 — `internal/update/check_test.go` | ⬜ pending |
| REL-05 / D-06 | 06-02 | A | Lazy update check honors 24h cache | unit | `go test ./internal/update -run TestMaybePrint_HonorsCache` | ❌ W0 | ⬜ pending |
| REL-05 / D-06 | 06-02 | A | Lazy update check skips on no-net | unit (rejecting transport) | `go test ./internal/update -run TestMaybePrint_NetworkError_Silent` | ❌ W0 | ⬜ pending |
| REL-05 / D-06 | 06-02 | A | Lazy update check uses ETag conditional GET | unit (httptest fake) | `go test ./internal/update -run TestMaybePrint_ConditionalGET` | ❌ W0 | ⬜ pending |
| REL-05 / D-07 | 06-02 | A | `runnerkit upgrade` detects Homebrew via Cellar/Caskroom path | unit | `go test ./internal/cli -run TestUpgrade_DetectsHomebrew` | ❌ W0 — `internal/cli/upgrade_test.go` | ⬜ pending |
| REL-05 / D-07 | 06-02 | A | `runnerkit upgrade` prints binary-channel command for non-Homebrew | unit | `go test ./internal/cli -run TestUpgrade_DetectsBinaryChannel` | ❌ W0 | ⬜ pending |
| REL-05 / D-07 | 06-02 | A | `runnerkit upgrade` JSON contract: ok/channel/commands keys, no execution | unit | `go test ./internal/cli -run TestUpgrade_JSONContract` | ❌ W0 | ⬜ pending |
| REL-05 / D-08 | 06-02 | A | `runnerkit upgrade-runner` re-runs `bootstrap.Apply` with new pin (persistent) | unit (fake remote.Executor) | `go test ./internal/cli -run TestUpgradeRunner_Persistent_ReAppliesWithNewPin` | ❌ W0 — `internal/cli/upgrade_runner_test.go` | ⬜ pending |
| REL-05 / D-08 | 06-02 | A | `runnerkit upgrade-runner` skips terminated ephemeral cleanly | unit | `go test ./internal/cli -run TestUpgradeRunner_Ephemeral_TerminalNoOp` | ❌ W0 | ⬜ pending |
| REL-05 / D-08 | 06-02 | A | `runnerkit upgrade-runner` refuses waiting ephemeral without `--force` | unit | `go test ./internal/cli -run TestUpgradeRunner_Ephemeral_WaitingRefusesWithoutForce` | ❌ W0 | ⬜ pending |
| REL-05 / D-08 | 06-02 | A | `runnerkit doctor` warns on stale runner version | unit | `go test ./internal/ops -run TestDoctor_StaleRunnerVersion` | ❌ W0 — extend `internal/ops/doctor.go` | ⬜ pending |
| REL-05 / D-09 | 06-02 | A | State migration runs forward-only v1→v2 | unit | `go test ./internal/state -run TestMigrate_V1ToV2_ForwardOnly` | ❌ W0 — `internal/state/migrations_test.go` | ⬜ pending |
| REL-05 / D-09 | 06-02 | A | State migration writes side-by-side backup before mutation | unit | `go test ./internal/state -run TestMigrate_WritesBackupBeforeMutation` | ❌ W0 | ⬜ pending |
| REL-05 / D-09 | 06-02 | A | State migration refuses-to-mutate on newer schema with exit code | unit | `go test ./internal/state -run TestMigrate_RefusesNewerSchema; go test ./internal/cli -run TestExitCodeStateSchemaTooNew` | ❌ W0 | ⬜ pending |
| REL-05 / D-09 | 06-02 | A | State migration is atomic (no partial writes on crash) | unit (failpoint via fs interface) | `go test ./internal/state -run TestMigrate_Atomic` | ❌ W0 | ⬜ pending |
| DOC-04 / D-14, D-15 | 06-03 | A | Every CLI-emitted RKD code resolves to a real anchor in `docs/troubleshooting/` | unit (file-walking test) | `go test ./internal/errcodes -run TestEveryCodeHasDocAnchor` | ❌ W0 — `internal/errcodes/codes_test.go` | ⬜ pending |
| DOC-04 / D-15 | 06-03 | A | RKD codes are unique across components | unit | `go test ./internal/errcodes -run TestCodesAreUnique` | ❌ W0 | ⬜ pending |
| DOC-04 / D-15 | 06-03 | A | URL builder honors `RUNNERKIT_DOCS_BASE` override | unit | `go test ./internal/errcodes -run TestURL_RespectsEnvOverride` | ❌ W0 | ⬜ pending |
| DOC-04 / D-16 | 06-03 | A | All four failure surfaces have at least one entry per component file | unit (markdown grep) | `go test ./internal/errcodes -run TestEachComponentHasMinimumOneEntry` | ❌ W0 | ⬜ pending |
| DOC-04 / D-17 | 06-03 | A | Each component file has Symptom/Diagnosis/Fix structure for every code | unit (markdown structure check) | `go test ./internal/errcodes -run TestEntriesFollowSymptomDiagnosisFix` | ❌ W0 | ⬜ pending |
| D-10 / Phase 1 outstanding | 06-04 | B | Live GH permission smoke succeeds against a real repo | live (manual) | `make smoke-live-byo` (requires `RUNNERKIT_SMOKE_BYO_HOST`, `RUNNERKIT_SMOKE_REPO`, `gh auth status`) | ❌ W0 — `scripts/smoke/byo-permission.sh` | ⬜ pending |
| D-10 / Phase 4 outstanding | 06-04 | B | Live Hetzner end-to-end including destroy-verify succeeds | live (manual) | `make smoke-live-cloud` (requires `HCLOUD_TOKEN`, `RUNNERKIT_SMOKE_REPO`) | ❌ W0 — `scripts/smoke/cloud-end-to-end.sh`, `cmd/_smokebin/empty_precheck`, `cmd/_smokebin/destroy_verify` | ⬜ pending |
| D-12 gate 1 | 06-04 | B | Empty-project precheck refuses if any `runnerkit-*` resource exists | live (manual) AND unit (fake hcloud client) | `make smoke-live-cloud` AND `go test ./cmd/_smokebin -run TestEmptyPrecheck_RefusesOnExisting` | ❌ W0 | ⬜ pending |
| D-12 gate 2 | 06-04 | B | Destroy-verify polls and asserts 404 within timeout | live (manual) AND unit (fake hcloud returning 404 on Nth poll) | `make smoke-live-cloud` AND `go test ./cmd/_smokebin -run TestDestroyVerify_Timeout` | ❌ W0 | ⬜ pending |
| D-13 | 06-04 | B | Stopwatch checklist captures BYO and Hetzner durations | manual (10-min stopwatch) | Maintainer follows checklist in `docs/release-process.md` and writes `RELEASE-NOTES-vX.Y.Z.md` | ❌ W0 — `docs/release-process.md` | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Files/dirs that MUST exist before substantive plan tasks start (each owned by the listed plan):

- [ ] `.goreleaser.yaml` (skeleton, validates `goreleaser check`) — Plan 06-01
- [ ] `.github/workflows/release.yml` — Plan 06-01
- [ ] `.github/workflows/pr-checks.yml` (or extend existing PR workflow) running `goreleaser check` + `--snapshot --skip=publish` — Plan 06-01
- [ ] Separate repo `salar/homebrew-runnerkit` — Plan 06-01 (manual maintainer step before first release)
- [ ] `HOMEBREW_TAP_GITHUB_TOKEN` repo secret — Plan 06-01 (manual maintainer step)
- [ ] `internal/update/` package skeleton (`check.go`, `version.go`, `check_test.go`) — Plan 06-02
- [ ] `internal/errcodes/` package skeleton (`codes.go`, `codes_test.go`) — Plan 06-03
- [ ] `internal/state/migrations_test.go` (replaces 16-line stub with real chain test fixtures) — Plan 06-02
- [ ] `internal/cli/upgrade_test.go`, `internal/cli/upgrade_runner_test.go` — Plan 06-02
- [ ] `docs/troubleshooting/` directory with 6 component files + README index (initially empty Symptom/Diagnosis/Fix templates per D-17) — Plan 06-03
- [ ] `Makefile` with `smoke-live`, `smoke-live-byo`, `smoke-live-cloud`, `smoke-stopwatch` targets — Plan 06-04
- [ ] `cmd/_smokebin/` Go programs for `empty_precheck` and `destroy_verify` — Plan 06-04
- [ ] `scripts/smoke/` shell wrappers — Plan 06-04
- [ ] `docs/release-process.md` (maintainer-only) and `docs/upgrade.md` (user-facing) — Plans 06-01 and 06-02

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live BYO `runnerkit up` against a real GitHub repo | REL-05 / D-10 | Real `gh auth` + real GitHub Actions API; no fake exists; closes Phase 1 outstanding live-smoke note | Maintainer exports `RUNNERKIT_SMOKE_BYO_HOST` and `RUNNERKIT_SMOKE_REPO`, runs `make smoke-live-byo`, captures duration into `06-VERIFICATION.md` and `RELEASE-NOTES-vX.Y.Z.md` |
| Live Hetzner end-to-end (provision → register → run → destroy → verify-404) | REL-05 / D-10, D-12 | Billable resources; closes Phase 4 outstanding live-smoke note | Maintainer ensures Hetzner project is empty of `runnerkit-*` resources, exports `HCLOUD_TOKEN` and `RUNNERKIT_SMOKE_REPO`, runs `make smoke-live-cloud`. Empty-project precheck (D-12 gate 1) must pass; destroy-verify polling must return 404 within timeout (D-12 gate 2). Lingering resources fail the smoke loudly. |
| 10-minute stopwatch checklist on a clean machine | REL-05 / D-13, Success Criterion 4 | Measures the load-bearing "10-minute reliable runner" promise from PROJECT.md | Maintainer follows checklist in `docs/release-process.md` on a fresh laptop / clean VM, captures wall-clock durations into `RELEASE-NOTES-vX.Y.Z.md` per release |
| Editorial pass on each `docs/troubleshooting/<component>.md` | DOC-04 / D-17 | Content quality (clarity, copyable commands, accurate symptoms) is human-reviewed; only structure is automatable | Maintainer reads each component file end-to-end as if currently stuck; verifies copyable commands work; checks each Symptom/Diagnosis/Fix block has the right granularity |

---

## Fake/Real Boundary (summary)

**Real in CI (release workflow on tag push):** GitHub Actions OIDC token (cosign keyless against real Sigstore Fulcio + Rekor); GoReleaser against real Go compiler; real GitHub Releases API for asset upload; real Homebrew tap repo for cask commit.

**Real in CI (PR workflow):** `goreleaser check` and `goreleaser release --snapshot --skip=publish --clean`; `go test ./...`.

**Faked in unit tests:** GitHub Releases API (`httptest.Server` returning fixture JSON + ETag); Hetzner client (existing fakes in `internal/provider/hetzner/` + new fakes in `cmd/_smokebin/`); filesystem (`t.TempDir()`); time (`Clock func() time.Time` injection in `internal/cli/Dependencies`); upgrade channel detection (file-path fixtures).

**Real only in `make smoke-live` (manual, maintainer-only, NOT in CI):** real `gh auth` against a real GitHub repo; real `hcloud-go` against a real Hetzner project; real human stopwatch.

---

## Coverage Map per Phase 6 Success Criterion

| Success Criterion (from ROADMAP.md) | Test layers | Closing plan(s) |
|---|---|---|
| (1) Install official release + documented upgrade path | unit (config check, snapshot, signs); integration (CI tag-mode verify-blob round-trip); live (BYO smoke installs from cask) | 06-01 + 06-04 BYO smoke |
| (2) State migration safe across releases or block with guidance | unit (forward, backup, refuse-newer, atomic) | 06-02 |
| (3) Cleanup + troubleshooting docs | unit (every-code-has-anchor, unique codes, structure check) + manual editorial pass | 06-03 |
| (4) Fresh-user 10-min path + workflow run + clean up | live (BYO + cloud + stopwatch checklist) | 06-04 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30 s for quick run, < 90 s for wave merge
- [ ] `nyquist_compliant: true` set in frontmatter
- [ ] Live smoke (`make smoke-live`) green before tagging v1.0.0
- [ ] `06-VERIFICATION.md` filled with v1.0.0 baseline durations + runner IDs + Hetzner cost

**Approval:** pending
