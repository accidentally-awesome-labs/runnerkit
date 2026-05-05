---
phase: 06-release-upgrade-docs-and-v1-validation
type: verification
status: gaps_found
verified: 2026-05-02T00:00:00Z
score: 3/4 plans verified (06-04 partial-blocked)
created: 2026-05-04
re_verification: false
gaps:
  - truth: "BYO bootstrap completes end-to-end against a real host without manual sudoers preconfiguration."
    status: failed
    reason: "Two latent bugs in the BYO bootstrap path block live smoke and make BYO unshippable in v1.0.0. Source of truth: 06-GAP-byo-sudo-handling.md (Tasks A-E)."
    artifacts:
      - path: "internal/preflight/checks.go"
        issue: "CheckPrivilege only tests `probe.Commands[\"sudo\"]` (binary present); never probes `sudo -n true`. Falsely passes when sudo requires a password, so bootstrap then fails opaquely with `bootstrap_failed` while remote stderr is swallowed."
      - path: "internal/bootstrap/install.go"
        issue: "Step `download_runner` (line 74 in Apply, line 115 in ApplyEphemeral) creates the install dir owned by `runnerkit-runner` (mode 0755) via `sudo install -d -o`, then runs plain `curl`/`sha256sum -c -`/`tar xzf` WITHOUT sudo as the SSH user. SSH user has no write access; `curl: (23) Failure writing output to destination, Permission denied`."
      - path: "internal/bootstrap/script.go"
        issue: "RenderInstallScript and RenderEphemeralInstallScript reproduce the same `sudo install -d -o serviceUser` → plain `curl`/`sha256sum`/`tar` pattern. Same permission contradiction."
      - path: "internal/cli/up.go"
        issue: "`bootstrap_failed` CLI message (line 224) swallows underlying remote stderr. Even when the bootstrap path is fixed, surfacing redacted stderr is required for self-diagnosis (Task A — surface remote stderr alongside the preflight fix)."
    missing:
      - "Task A: replace binary-existence check in CheckPrivilege with a real `sudo -n true` probe (passwordless / password-required / not-in-sudoers / sudo-missing branches); surface redacted remote stderr on bootstrap_failed."
      - "Task B: Path B — interactive sudo password fallback when preflight reports `host.privilege.password_required` (TTY prompt, `sudo -S` per command, redact.SudoPassword registration; `--non-interactive` fails with remediation pointing at byo-prepare)."
      - "Task C: Path C — new `runnerkit byo-prepare --host user@host` command (idempotent scoped sudoers entry under /etc/sudoers.d/runnerkit-installer, `visudo -c` validation, --remove inverse, doctor integration finding `byo_host_prepared`)."
      - "Task D: docs/byo-quickstart.md Sudo Setup section (Path C → Path B decision tree); docs/troubleshooting/bootstrap.md new RKD-BOOT-NNN entry for sudo password required; README.md one-liner under BYO install."
      - "Task E: prefix `curl`, `sha256sum -c`, `tar xzf` in download_runner with sudo (Option 1 — minimal diff), apply same fix in RenderInstallScript and RenderEphemeralInstallScript; add integration test exercising real shell with tmpfs sandbox to close the fakeExecutor-only test gap that hid this bug since Plan 02-02; extend scripts/smoke/byo-permission.sh to assert install dir contains config.sh after bootstrap apply."
  - truth: "10-minute stopwatch durations + Hetzner cost + Hetzner resource IDs are recorded in 06-VERIFICATION.md and RELEASE-NOTES-v1.0.0.md from a real maintainer run on a clean machine."
    status: failed
    reason: "Plan 06-04 Task 4 (live smoke + maintainer stopwatch fill-in) is blocked by the BYO bugs above. Live smoke attempt 1 (2026-05-04 to 2026-05-05) failed at the BYO step before reaching cloud-end-to-end. No Hetzner resources were created. Once 06-GAP-byo-sudo-handling.md closes, Task 4 can re-run end-to-end without host-side preconfiguration."
    artifacts:
      - path: ".planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md"
        issue: "Maintainer fill-in fields (BYO host, repo, duration, runner ID; Hetzner repo, project, duration, cost EUR, 5 resource IDs, gate 1/2 PASS, stopwatch totals, sign-off) remain blank. Skeleton baseline is in place from Task 3 commit 140cb06; numbers await closure of the BYO blockers."
      - path: "RELEASE-NOTES-v1.0.0.md"
        issue: "10-minute stopwatch wall-clock numbers are still placeholders. Awaits the same maintainer run."
    missing:
      - "Re-run `make smoke-live` against a fresh BYO host AND a real Hetzner project AFTER Tasks A-E land. Fill BYO + Hetzner duration / runner ID / 5 resource IDs / cost / gate-1 PASS / gate-2 PASS / stopwatch totals into 06-VERIFICATION.md; fill stopwatch table into RELEASE-NOTES-v1.0.0.md; sign and date the verification baseline; resume signal `smoke-green` triggers Phase 6 sign-off and v1.0.0 tag push."
gap_source: 06-GAP-byo-sudo-handling.md
---

# Phase 06 — v1.0.0 Verification Baseline

> Source-of-truth for the v1.0.0 ten-minute reliable-runner promise. Filled in
> by the maintainer running `make smoke-live` before tagging v1.0.0.
> See `docs/release-process.md` Stopwatch Checklist (D-13).

## Test Suite (automated)

- [ ] `go test ./... -count=1 -race` green
- [ ] `goreleaser check` green
- [ ] `goreleaser release --snapshot --skip=publish --clean` green; `dist/` contains 4 platform tarballs + checksums.txt
- [ ] All 5 errcodes tests green (TestEveryCodeHasDocAnchor, TestCodesAreUnique, TestURL_RespectsEnvOverride, TestEachComponentHasMinimumOneEntry, TestEntriesFollowSymptomDiagnosisFix)
- [ ] All 4 state migration tests green (TestMigrate_V1ToV2_ForwardOnly, TestMigrate_WritesBackupBeforeMutation, TestMigrate_RefusesNewerSchema, TestMigrate_Atomic)
- [ ] All 6 update-check tests green (TestMaybePrint_*)
- [ ] All 7 upgrade/upgrade-runner/doctor-stale tests green
- [ ] All 3 _smokebin unit tests green (TestEmptyPrecheck_RefusesOnExisting, TestEmptyPrecheck_AllResourceTypes, TestDestroyVerify_Timeout)

## Live Smoke (manual, maintainer-only)

### BYO smoke (closes Phase 1 outstanding)

- [ ] `make smoke-live-byo` succeeds end-to-end
- BYO host: `user@________`
- Repo (maintainer-controlled, trusted): `________/________`
- Wall-clock duration: `____ seconds`
- Runner ID assigned by GitHub: `____`

### Hetzner smoke (closes Phase 4 outstanding)

- [ ] `make smoke-live-cloud` succeeds end-to-end
- Repo (maintainer-controlled, trusted, NOT public): `________/________`
- Hetzner project: `________`
- Wall-clock duration (up → workflow → destroy): `____ seconds`
- Hetzner cost (from project dashboard, EUR): `__.__`
- Resource IDs created (server / ssh-key / primary-ip(v4) / primary-ip(v6) / firewall): `____ / ____ / ____ / ____ / ____`
- D-12 gate 1 (empty-project precheck) status: `____` (PASS / FAIL)
- D-12 gate 2 (destroy-verify 404 within timeout) status: `____` (PASS / FAIL)
- Empty-project precheck final ID list size: `0` (must be exactly zero on a successful smoke)

### 10-minute stopwatch (D-13)

- [ ] BYO total: `____ minutes ____ seconds` (target ≤ 10 minutes)
- [ ] Hetzner total: `____ minutes ____ seconds` (target ≤ 10 minutes)

## Bundled Versions

- runner pin (`internal/bootstrap/package.go::RunnerVersion`): `2.334.0`
- GoReleaser CI version: `v2.15.4`
- cosign CI version: `v3.0.6`
- hcloud-go (Phase 4 pinned): `v1.59.2`

## Sign-Off

- [ ] Maintainer signature: `____________ (date)`
- [ ] All gates green; ready to push `git tag -a v1.0.0`

---

# Verifier Verdict (gsd-verifier, 2026-05-02)

**Status: gaps_found** — 3 of 4 plans complete; Plan 06-04 partial-blocked on `06-GAP-byo-sudo-handling.md`.

## Goal Achievement

**Phase goal (from ROADMAP.md):** close v1.0.0 release readiness — REL-05 (release pipeline + upgrade lifecycle) + DOC-04 (cleanup/troubleshooting docs) + live validation harness + first release notes.

### Plan-level summary

| Plan | Subsystem | Status | Notes |
| ---- | --------- | ------ | ----- |
| 06-01 | release-packaging | VERIFIED | All must-haves present; release pipeline wired (REL-05). |
| 06-02 | upgrade-and-state-migration | VERIFIED | Forward-only migrations + ExitStateSchemaTooNew + MaybePrint + upgrade/upgrade-runner + runner_version_stale doctor finding all in place (REL-05). |
| 06-03 | troubleshooting-docs-and-rkd-codes | VERIFIED | RKD code registry + 6 troubleshooting docs + URL builder env override + 5 errcodes tests (DOC-04). |
| 06-04 | v1-validation-and-live-smoke | PARTIAL-BLOCKED | Tasks 1-3 (harness, smokebin, stopwatch checklist, RELEASE-NOTES template, 06-VERIFICATION skeleton) verified; Task 4 (live smoke + maintainer stopwatch fill-in) blocked on the two BYO bugs in `06-GAP-byo-sudo-handling.md`. |

### Observable Truths (Plan 06-01 — release-packaging, REL-05)

| # | Truth | Status | Evidence |
| - | ----- | ------ | -------- |
| 1 | Pushing a vX.Y.Z tag produces 4 platform binaries + checksums.txt + sigstore bundle | VERIFIED | `.goreleaser.yaml` `version: 2`, 4-platform matrix, `homebrew_casks:`, `signs.artifacts: checksum`, `--bundle=${signature}`, `--yes`. `.github/workflows/release.yml` tag-triggered with `id-token: write`, `cosign-installer@v3` (v3.0.6), `goreleaser-action@v7` (`~> v2`), `args: release --clean`, `HOMEBREW_TAP_GITHUB_TOKEN` env. |
| 2 | PR CI runs `goreleaser check` and `goreleaser release --snapshot --skip=publish --clean` | VERIFIED | `.github/workflows/pr-checks.yml` includes both steps + dist-archive assertions for all 4 platforms + `go test ./... -count=1 -race`. |
| 3 | Maintainer can verify checksums.txt cosign signature from README cert-identity URL | VERIFIED | README.md install section contains literal `cosign verify-blob` snippet with `--bundle`, `--certificate-identity` URL `github.com/.../release.yml@refs/tags/`, `--certificate-oidc-issuer https://token.actions.githubusercontent.com`. |
| 4 | Homebrew tap receives Cask formula update on every successful tag run | VERIFIED | `.goreleaser.yaml` `homebrew_casks:` with `repository.token: '{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}'`. Tap repo `accidentally-awesome-labs/homebrew-runnerkit` confirmed `tap-ready` 2026-05-02 (commit c359831 org migration). |
| 5 | Release workflow refuses fork PRs (cosign keyless OIDC) | VERIFIED | `release.yml` triggers ONLY on `push: tags: ['v*']` — fork tag pushes never trigger upstream workflow; `id-token: write` is upstream-only. |

### Observable Truths (Plan 06-02 — upgrade-and-state-migration, REL-05)

| # | Truth | Status | Evidence |
| - | ----- | ------ | -------- |
| 1 | `up`/`status`/`doctor` print non-blocking lazy update notice; silent in JSON/CI/no-net; 24h cache | VERIFIED | `internal/update/check.go::MaybePrint` honors RUNNERKIT_NO_UPDATE_NOTIFIER + jsonOutput + ETag conditional GET + 24h cache; deferred call wired in up/status/doctor runners. |
| 2 | `runnerkit upgrade` channel-detect (homebrew/binary/unknown), print-only | VERIFIED | `internal/cli/upgrade.go::detectChannel` matches `/Cellar/runnerkit/` and `/Caskroom/runnerkit/` paths; never replaces own binary. |
| 3 | `runnerkit upgrade-runner` re-applies bootstrap with bundled pin; refuses ephemeral-waiting without --force | VERIFIED | `internal/cli/upgrade_runner.go` calls `bootstrap.Apply` / `bootstrap.ApplyEphemeral`. (Behavior of refuse-without-force confirmed by upgrade_runner_test.go in green test suite.) |
| 4 | `runnerkit doctor` emits stale-runner-version warning | VERIFIED | `internal/ops/doctor.go` line 137 emits `runner_version_stale` finding via `errcodes.BootRunnerVersionStale`. |
| 5 | Loading state.json with v1 schema migrates forward with backup; newer schema refuses with ExitStateSchemaTooNew | VERIFIED | `internal/state/migrations.go` ErrSchemaTooNew sentinel; `internal/state/schema.go` SchemaVersion = "2"; `internal/cli/exit.go` ExitStateSchemaTooNew = 7. All 4 state migration tests green. |

### Observable Truths (Plan 06-03 — troubleshooting-docs-and-rkd-codes, DOC-04)

| # | Truth | Status | Evidence |
| - | ----- | ------ | -------- |
| 1 | Every CLI failure / doctor finding traces to a stable RKD code | VERIFIED | `internal/errcodes/codes.go` registry covers AUTH/SSH/BOOT/GH/PROV/CLEAN/STATE/CORE prefixes. |
| 2 | Every RKD code resolves to a real HTML anchor in matching docs/troubleshooting/<component>.md | VERIFIED | TestEveryCodeHasDocAnchor green (errcodes package tests pass). 7 troubleshooting files present: README, auth, bootstrap, cleanup, github, provider, ssh. |
| 3 | URL builder honors RUNNERKIT_DOCS_BASE env override | VERIFIED | `internal/errcodes/url.go` references env override; TestURL_RespectsEnvOverride green. |
| 4 | All 4 D-16 failure surfaces (setup/bootstrap+service/operations/cloud+cleanup) covered | VERIFIED | TestEachComponentHasMinimumOneEntry green. |
| 5 | Symptom/Diagnosis/Fix structure across all entries | VERIFIED | TestEntriesFollowSymptomDiagnosisFix green. |

### Observable Truths (Plan 06-04 — v1-validation-and-live-smoke, REL-05 + DOC-04)

| # | Truth | Status | Evidence |
| - | ----- | ------ | -------- |
| 1 | `make smoke-live-byo` runs against real GitHub repo + BYO host with env-var preconditions | PARTIAL | Makefile target + scripts/smoke/byo-permission.sh + RUNNERKIT_SMOKE_BYO_HOST/RUNNERKIT_SMOKE_REPO checks all in place. Live execution against `salar@mckee-small-desktop` FAILED at `bootstrap_failed` due to BYO bugs (see `06-GAP-byo-sudo-handling.md`). |
| 2 | `make smoke-live-cloud` empty-precheck D-12 gate 1 refuses on existing `runnerkit-*` resources | VERIFIED | `cmd/_smokebin/empty_precheck/main.go` namePrefix = "runnerkit-"; refuses with named offenders. TestEmptyPrecheck_RefusesOnExisting + TestEmptyPrecheck_AllResourceTypes green. Live execution did not reach this step (BYO blocked first). |
| 3 | Destroy-verify D-12 gate 2 polls every saved resource ID until 404 within RUNNERKIT_SMOKE_TIMEOUT | VERIFIED | `cmd/_smokebin/destroy_verify/main.go` polls hcloud.IsError(err, hcloud.ErrorCodeNotFound); RUNNERKIT_SMOKE_TIMEOUT default 300s; TestDestroyVerify_Timeout green (success-after-N-polls + timeout-failure subtests). |
| 4 | 10-minute stopwatch records BYO + Hetzner durations into RELEASE-NOTES + 06-VERIFICATION | FAILED | Stopwatch checklist scaffolded in `docs/release-process.md` Stopwatch Checklist (D-13) section + 06-VERIFICATION skeleton + RELEASE-NOTES-v1.0.0.md table. Maintainer fill-in BLOCKED — live smoke attempt 1 failed before reaching the cloud path; no wall-clock numbers captured. |
| 5 | `make smoke-live` is NOT triggered by any GitHub Actions workflow (D-11) | VERIFIED | `grep -rq smoke-live .github/workflows/` returns no matches. |
| 6 | `cmd/_smokebin/` binaries excluded from `go build ./...` by `_` prefix | VERIFIED | `_smokebin` directory present; `go test ./... -count=1` green across all 17 packages including the smoke binaries. |

### Required Artifacts (spot-check)

| Artifact | Status | Notes |
| -------- | ------ | ----- |
| `.goreleaser.yaml` | VERIFIED | version: 2, homebrew_casks:, artifacts: checksum, --bundle=${signature}, --yes — all present. |
| `.github/workflows/release.yml` | VERIFIED | id-token: write, cosign-installer@v3, goreleaser-action@v7 — all present. |
| `.github/workflows/pr-checks.yml` | VERIFIED | goreleaser check + snapshot build + go test — all present. |
| `internal/state/migrations.go` | VERIFIED | ErrSchemaTooNew sentinel + forwardMigrations + backup-v* — present. |
| `internal/state/schema.go` | VERIFIED | SchemaVersion = "2". |
| `internal/cli/exit.go` | VERIFIED | ExitStateSchemaTooNew = 7. |
| `internal/update/check.go` | VERIFIED | MaybePrint + RUNNERKIT_NO_UPDATE_NOTIFIER + If-None-Match — present. |
| `internal/cli/upgrade.go` | VERIFIED | detectChannel + /Cellar/runnerkit/ + /Caskroom/runnerkit/ — present. |
| `internal/cli/upgrade_runner.go` | VERIFIED | bootstrap.Apply + bootstrap.ApplyEphemeral — present. |
| `internal/ops/doctor.go` | VERIFIED | runner_version_stale finding wired to errcodes.BootRunnerVersionStale. |
| `internal/errcodes/codes.go` | VERIFIED | AUTH/BOOT/STATE codes present; full registry covers 8 prefixes. |
| `internal/errcodes/url.go` | VERIFIED | RUNNERKIT_DOCS_BASE env override + default github URL. |
| `docs/troubleshooting/*` | VERIFIED | 7 files present (README + 6 component files). |
| `Makefile` | VERIFIED | smoke-live + smoke-live-byo + smoke-live-cloud + smoke-stopwatch targets with env-var preconditions. |
| `cmd/_smokebin/empty_precheck/main.go` | VERIFIED | runnerkit- prefix gate + hcloud.NewClient + interface for testing. |
| `cmd/_smokebin/destroy_verify/main.go` | VERIFIED | ErrorCodeNotFound + RUNNERKIT_SMOKE_TIMEOUT + partial-unmarshal of state.json. |
| `scripts/smoke/cloud-end-to-end.sh` | VERIFIED | trap EXIT INT TERM + runnerkit destroy --yes guard (Pitfall 7). |
| `docs/release-process.md` | VERIFIED | Stopwatch Checklist (D-13) section + BYO + Hetzner tables. |
| `RELEASE-NOTES-v1.0.0.md` | VERIFIED (template) | Title + 2.334.0 runner pin + cosign verify-blob snippet present. Wall-clock numbers placeholder pending Task 4 maintainer fill-in. |
| `internal/preflight/checks.go` | FAILED (Bug 1) | CheckPrivilege at line 127 only checks binary presence (`probe.Commands["sudo"]`); never probes `sudo -n true`. False-positive on password-required hosts. |
| `internal/bootstrap/install.go` | FAILED (Bug 2) | `download_runner` step (line 74 in Apply, line 115 in ApplyEphemeral): `sudo install -d -o serviceUser` then plain `curl`/`sha256sum -c`/`tar xzf` without sudo — ownership/permission contradiction. |
| `internal/bootstrap/script.go` | FAILED (Bug 2) | RenderInstallScript and RenderEphemeralInstallScript reproduce the same buggy pattern. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full Go test suite green | `go test ./... -count=1` | 17/17 packages OK; all errcodes / state / update / upgrade / smokebin tests pass | PASS |
| `make smoke-live` not in workflows (D-11) | `grep -rq smoke-live .github/workflows/` | no match | PASS |
| Live BYO smoke completes against real host | `make smoke-live-byo` against salar@mckee-small-desktop | ERROR exit status 4 / `bootstrap_failed`; remote stderr swallowed; root cause = Bug 1 (sudo password) + Bug 2 (download_runner permission) | FAIL |
| Live Hetzner smoke completes against real project | `make smoke-live-cloud` | NOT RUN — blocked by BYO failure earlier in `make smoke-live` chain | SKIP |
| Maintainer 10-minute stopwatch numbers captured into 06-VERIFICATION + RELEASE-NOTES | manual fill-in after live smoke | Skeleton + tables present, numbers blank | FAIL |

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
| ----------- | ------------ | ----------- | ------ | -------- |
| REL-05 | 06-01, 06-02, 06-04 | Developer can update the runner binary/service or follow a documented upgrade path that prevents immediate runner rot. | PARTIAL | Release pipeline (06-01) + upgrade lifecycle (06-02) + harness (06-04 Tasks 1-3) all in place. Live smoke + stopwatch sign-off (06-04 Task 4) blocked on BYO bugs. REQUIREMENTS.md still flags REL-05 as Complete; this verification downgrades that to PARTIAL pending gap-closure. |
| DOC-04 | 06-03, 06-04 | Developer can read cleanup and troubleshooting guidance for common failure modes. | SATISFIED | RKD code registry + 6 troubleshooting docs (auth/ssh/bootstrap/github/provider/cleanup) + README index + URL env override all in place. 5 required errcodes tests green. DOC-04 has no cross-cutting blocker. |

No orphaned requirement IDs detected — every ID claimed by Phase 6 plans (REL-05, DOC-04) maps to artifacts in REQUIREMENTS.md.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| `internal/preflight/checks.go` | 127 | Binary-presence check masquerading as capability check (`probe.Commands["sudo"]`) | BLOCKER | Preflight false-positive on password-prompt sudo hosts; surfaces as opaque `bootstrap_failed`. |
| `internal/bootstrap/install.go` | 74, 115 | `sudo install -d -o serviceUser` followed by plain `curl`/`sha256sum`/`tar` — ownership/permission contradiction | BLOCKER | BYO non-functional in v1 against any non-`runnerkit-runner` SSH user. |
| `internal/bootstrap/script.go` | RenderInstallScript / RenderEphemeralInstallScript | Same pattern duplicated in script-renderer path | BLOCKER | Same impact as above; both code paths share the bug. |
| Test gap | `internal/bootstrap/*` | Every bootstrap test uses `fakeExecutor` that records commands but never executes them — no real-shell integration test | WARNING | Latent since Plan 02-02; allowed Bug 2 to escape to live smoke. Closure plan must add real-shell or filesystem-permission-aware test (Task E). |

### Human Verification Required (post-gap-closure)

| Test | Expected | Why Human |
| ---- | -------- | --------- |
| `make smoke-live` against real BYO + real Hetzner project after Tasks A-E close | BYO completes end-to-end without manual sudoers preconfiguration; Hetzner D-12 gates 1+2 PASS; total wall-clock ≤ 10 min on each path | Real GitHub PAT + real billable Hetzner resources; Claude cannot exercise the OIDC + billing surfaces. Maintainer fills 06-VERIFICATION + RELEASE-NOTES, types `smoke-green`. |

## Gaps Summary

The phase delivers v1.0.0 release readiness almost entirely:

- **Release pipeline (REL-05 distribution):** complete and locally verifiable. Tag push will produce signed multi-platform release.
- **Upgrade lifecycle (REL-05 lifecycle):** complete. Forward-only migrations + lazy update notice + channel-detect upgrade + upgrade-runner + stale doctor finding all wired.
- **Cleanup/troubleshooting docs (DOC-04):** complete. RKD code registry + 6 component docs + URL env override + 5 required tests green.
- **Live validation harness scaffolding:** complete. Makefile + 4 shell wrappers + 2 smoke binaries (with passing unit tests) + stopwatch checklist + RELEASE-NOTES template + 06-VERIFICATION skeleton all in place. D-11 enforced (no smoke-live in workflows). D-12 gates 1+2 wired and unit-tested.

**What's blocking v1.0.0 tag push:**

The live BYO smoke surfaced two real, latent bugs that make BYO non-functional in v1 against any non-NOPASSWD-sudo SSH user:

1. **Preflight false-positive (Bug 1).** `internal/preflight/checks.go::CheckPrivilege` only tests whether the `sudo` binary is installed; it never tests whether the SSH user can run sudo non-interactively. Hosts with sudo-with-password configurations pass preflight, then bootstrap fails opaquely with `bootstrap_failed` while the actual remote stderr is swallowed by the executor.
2. **Download_runner permission failure (Bug 2).** Even with NOPASSWD sudo configured, the `download_runner` step in `internal/bootstrap/install.go::Apply` (and the same pattern in `ApplyEphemeral` and in `script.go::RenderInstallScript` / `RenderEphemeralInstallScript`) creates the install directory owned by `runnerkit-runner` (mode 0755) via `sudo install -d -o`, then runs plain `curl` / `sha256sum -c` / `tar xzf` *without sudo* as the SSH user. The SSH user cannot write to the directory; the curl fails with `Permission denied`. This bug went undetected from Plan 02-02 forward because every bootstrap unit test uses `fakeExecutor` that records commands but never executes them — no real-shell integration test exists.

`06-GAP-byo-sudo-handling.md` enumerates Tasks A-E that close both bugs comprehensively (preflight `sudo -n true` probe + redacted-stderr surfacing for Bug 1; sudo-prefixed download/verify/extract + real-shell integration test for Bug 2; plus interactive sudo password fallback (Path B) and `runnerkit byo-prepare` command (Path C) per the user's 2026-05-04 decision; plus docs for byo-quickstart and troubleshooting/bootstrap.md). It is the actionable artifact for `/gsd:plan-phase 06 --gaps`.

Once Tasks A-E land, Plan 06-04 Task 4 (live smoke + maintainer stopwatch fill-in) re-runs end-to-end against a fresh BYO host without any sudoers preconfiguration, the maintainer fills the 06-VERIFICATION baseline + RELEASE-NOTES-v1.0.0.md numbers, and Phase 6 closes with a `smoke-green` resume signal that triggers the v1.0.0 tag push per `docs/release-process.md`.

---

_Verified: 2026-05-02_
_Verifier: Claude (gsd-verifier)_
_Source of truth for the gap: 06-GAP-byo-sudo-handling.md (Tasks A-E)._
