---
phase: 06-release-upgrade-docs-and-v1-validation
type: verification
status: gaps_found
verified: 2026-05-02T00:00:00Z
score: 6/7 plans verified (06-07 partial-blocked by Bug 3)
created: 2026-05-04
updated: 2026-05-08
re_verification: true
gaps:
  - truth: "Plan 06-07 attempt-20 smoke can run non-interactively after Plan 06-13 without TTY-dependent sudo failure."
    status: failed
    reason: "2026-05-08 attempt-20 `make smoke-live` failed in BYO step with `RKD-BOOT-015` / non-TTY sudo error. In automation (`tee`/non-PTY) context, `runnerkit up --yes` could not prompt and returned 'RunnerKit needs a sudo password but no TTY is available for prompting.' Subsequent interactive maintainer run still failed in bootstrap apply with `sudo: a terminal is required ... sudo: a password is required` after preflight passed, indicating the Path-C prepared-host expectation is still not satisfied end-to-end in this smoke path. Cloud smoke did not run because BYO failed first."
    artifacts:
      - path: "smoke-output.log"
        issue: "Attempt-20 run captured `ERROR RunnerKit needs a sudo password but no TTY is available for prompting` and exited from `smoke-live-byo` before cloud smoke."
      - path: "scripts/smoke/byo-permission.sh"
        issue: "Current smoke harness invokes `runnerkit up --yes` in a non-interactive context under tee; when password prompting is required, this path fails with RKD-BOOT-015 and cannot proceed."
    missing:
      - "File and execute Plan 06-14 to close the non-TTY BYO smoke blocker; then re-run Plan 06-07 with a green BYO pass before attempting cloud and stopwatch baseline fill-in."
  - truth: "BYO bootstrap completes end-to-end against a real host without manual sudoers preconfiguration."
    status: partial
    reason: "Two of three latent bugs in the BYO bootstrap path closed by Plans 06-05 + 06-06 (commits ee5c0a2 + 08b8708). Plan 06-07 attempt 1 re-smoke against salar@mckee-small-desktop on 2026-05-05 surfaced Bug 3 (`register_runner` runas mismatch — see Bug 3 entry below). BYO still non-functional in v1.0.0 until Task F lands. Source of truth: 06-GAP-byo-sudo-handling.md (Tasks A-E CLOSED; Task F OPEN)."
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
  - truth: "BYO `register_runner` step succeeds without `(ALL)` runas in host sudoers (i.e., scoped sudoers entry from byo-prepare alone is sufficient)."
    status: failed
    reason: "Bug 3 (discovered 2026-05-05 during Plan 06-07 re-smoke attempt 1). `internal/bootstrap/script.go:47, 83` invokes `sudo -u runnerkit-runner ./config.sh ...` to register the GitHub runner as the unprivileged service user. Linux sudoers semantics: `(root) NOPASSWD: ALL` only covers runas=root. Running as a non-root user matches a different rule and triggers password prompt. BYO host with system-wide `(root) NOPASSWD: ALL` (or with the byo-prepare scoped sudoers `ALL=(root) NOPASSWD:` template) cannot run `sudo -u runnerkit-runner` without a password. Cloud path unaffected because `internal/provider/hetzner/provision.go:241` cloud-init configures runnerkit-admin with `sudo: ALL=(ALL) NOPASSWD:ALL` — `(ALL)` runas covers runnerkit-runner. v1.0.0 cannot ship with BYO non-functional."
    artifacts:
      - path: "internal/bootstrap/script.go"
        issue: "Line 47 (RenderInstallScript) and line 83 (RenderEphemeralInstallScript) use `sudo -u runnerkit-runner ./config.sh ...` for runner registration. `sudo -u <non-root>` requires `(ALL)` runas in sudoers; both system-wide `(root) NOPASSWD: ALL` and the byo-prepare scoped template `ALL=(root) NOPASSWD:` cover only runas=root."
      - path: "internal/bootstrap/sudoers.go"
        issue: "RenderSudoersEntry (line 23) writes `<user> ALL=(root) NOPASSWD: <commands>`. Even after byo-prepare runs successfully, register_runner still hits the runas mismatch."
      - path: "internal/bootstrap/script_test.go"
        issue: "Substring assertions in script_test.go cover `sudo curl|sha256sum|tar` from Bug 2 fix but never asserted absence of `sudo -u <non-root>` patterns. Bug 3 lived undetected since Plan 02-02 because every bootstrap test uses fakeExecutor that records commands but never executes them — no integration test covers the case of an SSH user with only root NOPASSWD."
    missing:
      - "Task F: replace `sudo -u runnerkit-runner ./config.sh ...` in RenderInstallScript and RenderEphemeralInstallScript with `sudo su -s /bin/bash - runnerkit-runner -c '<config.sh ...>'`. `su` runs from root → no `(ALL)` runas needed → works on BYO host with only root NOPASSWD AND on cloud host with broader NOPASSWD."
      - "Task F: extend script_test.go with assertion that `sudo -u` is absent and `su -s /bin/bash` is present in the rendered script."
      - "Task F: extend install_integration_test.go with a sub-case that simulates an SSH user whose sudoers has only `(root) NOPASSWD: ALL` and asserts the registration command line is acceptable (root-only NOPASSWD sufficient)."
      - "Task F: re-run Plan 06-07 BYO smoke against salar@mckee-small-desktop and assert a GitHub runner ID lands before destroy."
  - truth: "10-minute stopwatch durations + Hetzner cost + Hetzner resource IDs are recorded in 06-VERIFICATION.md and RELEASE-NOTES-v1.0.0.md from a real maintainer run on a clean machine."
    status: failed
    reason: "Plan 06-07 Task 1 (live smoke re-run + maintainer stopwatch fill-in) blocked by Bug 3 above. Re-smoke attempt 1 (2026-05-05) failed at register_runner before reaching cloud-end-to-end. No Hetzner resources created. Once Task F (Plan 06-08) closes, Plan 06-07 can re-run end-to-end without host-side preconfiguration."
    artifacts:
      - path: ".planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md"
        issue: "Maintainer fill-in fields (BYO host, repo, duration, runner ID; Hetzner repo, project, duration, cost EUR, 5 resource IDs, gate 1/2 PASS, stopwatch totals, sign-off) remain blank. Skeleton baseline is in place from Plan 06-04 Task 3 commit 140cb06; numbers await closure of Bug 3."
      - path: "RELEASE-NOTES-v1.0.0.md"
        issue: "10-minute stopwatch wall-clock numbers are still placeholders. Awaits the same maintainer run."
    missing:
      - "Re-run `make smoke-live` against a fresh BYO host AND a real Hetzner project AFTER Task F (Plan 06-08) lands. Fill BYO + Hetzner duration / runner ID / 5 resource IDs / cost / gate-1 PASS / gate-2 PASS / stopwatch totals into 06-VERIFICATION.md; fill stopwatch table into RELEASE-NOTES-v1.0.0.md; sign and date the verification baseline; resume signal `smoke-green` triggers Phase 6 sign-off and v1.0.0 tag push."
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

# Verifier Verdict (gsd-verifier, 2026-05-02; updated 2026-05-05)

**Status: gaps_found** — 6 of 7 plans complete; Plan 06-07 partial-blocked on `06-GAP-byo-sudo-handling.md` Bug 3 / Task F.

## Goal Achievement

**Phase goal (from ROADMAP.md):** close v1.0.0 release readiness — REL-05 (release pipeline + upgrade lifecycle) + DOC-04 (cleanup/troubleshooting docs) + live validation harness + first release notes.

### Plan-level summary

| Plan | Subsystem | Status | Notes |
| ---- | --------- | ------ | ----- |
| 06-01 | release-packaging | VERIFIED | All must-haves present; release pipeline wired (REL-05). |
| 06-02 | upgrade-and-state-migration | VERIFIED | Forward-only migrations + ExitStateSchemaTooNew + MaybePrint + upgrade/upgrade-runner + runner_version_stale doctor finding all in place (REL-05). |
| 06-03 | troubleshooting-docs-and-rkd-codes | VERIFIED | RKD code registry + 6 troubleshooting docs + URL builder env override + 5 errcodes tests (DOC-04). |
| 06-04 | v1-validation-and-live-smoke | PARTIAL-BLOCKED | Tasks 1-3 (harness, smokebin, stopwatch checklist, RELEASE-NOTES template, 06-VERIFICATION skeleton) verified; Task 4 (live smoke + maintainer stopwatch fill-in) blocked on the BYO bugs in `06-GAP-byo-sudo-handling.md`. |
| 06-05 | byo-bootstrap-blocker-fixes | VERIFIED | Bug 1 (preflight `sudo -n true` probe + redacted remote stderr) + Bug 2 (sudo-prefixed download/verify/extract + integration test) closed. Commit ee5c0a2 2026-05-04. |
| 06-06 | byo-prepare-and-sudo-prompt | VERIFIED | Path B (interactive sudo password fallback) + Path C (`runnerkit byo-prepare` scoped sudoers + visudo gate) + doctor `byo_host_prepared` finding + DOC-04 byo-quickstart Sudo Setup section closed. Commit 08b8708 2026-05-05. |
| 06-07 | live-smoke-rerun-and-baseline-fillin | PARTIAL-BLOCKED | Re-smoke attempt 1 (2026-05-05) verified Bugs 1+2 fix landed; surfaced Bug 3 (`register_runner` runas mismatch — `sudo -u runnerkit-runner` requires `(ALL)` runas, not covered by either system-wide `(root) NOPASSWD: ALL` or byo-prepare scoped template). BYO end-to-end blocked at registration step. Cloud smoke + 10-min stopwatch SKIPPED (cannot ship v1 with BYO non-functional regardless of cloud success). |

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
| `internal/preflight/checks.go` | RESOLVED (Bug 1) | CheckPrivilege now executes real `sudo -n true` probe with passwordless / password-required / no-sudo / sudo-missing branches. Plan 06-05 commit 314bf94. |
| `internal/bootstrap/install.go` | RESOLVED (Bug 2) | `download_runner` step now uses shared `downloadRunnerCommand(opts)` helper with sudo prefixes on curl/sha256sum/tar. Plan 06-05 commit 75c41aa. |
| `internal/bootstrap/script.go` (download) | RESOLVED (Bug 2) | RenderInstallScript and RenderEphemeralInstallScript both updated with sudo-prefixed download/verify/extract. |
| `internal/bootstrap/script.go` (register) | FAILED (Bug 3) | RenderInstallScript line 47 and RenderEphemeralInstallScript line 83 invoke `sudo -u runnerkit-runner ./config.sh ...`. `sudo -u <non-root>` requires `(ALL)` runas — neither system-wide `(root) NOPASSWD: ALL` nor byo-prepare scoped template `ALL=(root) NOPASSWD:` cover it. Task F (Plan 06-08) closes by switching to `sudo su -s /bin/bash - runnerkit-runner -c '...'`. |
| `internal/bootstrap/sudoers.go` | FAILED (Bug 3) | RenderSudoersEntry uses `ALL=(root) NOPASSWD:` runas. Task F closes by keeping the template (after Bug 3 fix the `(root)` runas is sufficient). |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full Go test suite green | `go test ./... -count=1 -race` | 17/17 packages OK after 06-05 + 06-06 land; all errcodes / state / update / upgrade / smokebin / preflight / sudoers / byo-prepare / redact / install / install_integration tests pass | PASS |
| `make smoke-live` not in workflows (D-11) | `grep -rq smoke-live .github/workflows/` | no match | PASS |
| Live BYO smoke completes against real host (attempt 1, 2026-05-04) | `make smoke-live-byo` against salar@mckee-small-desktop | ERROR exit status 4 / `bootstrap_failed`; remote stderr swallowed; root cause = Bug 1 (sudo password) + Bug 2 (download_runner permission) | FAIL |
| Live BYO smoke completes against real host (attempt 2, 2026-05-05 — re-smoke after 06-05+06-06 land) | `make smoke-live-byo` against salar@mckee-small-desktop with system-wide `(root) NOPASSWD: ALL` | Bug 1+2 fixes verified (sudo probe surfaced no password prompt; `/opt/actions-runner/runnerkit-<owner>-<repo>-local/config.sh` extracted successfully). FAILED at `register_runner`: `sudo: a terminal is required to read the password` because `sudo -u runnerkit-runner ./config.sh ...` matches `(ALL:ALL) ALL` (password required) instead of `(root) NOPASSWD: ALL`. Bug 3. | FAIL |
| Live Hetzner smoke completes against real project | `make smoke-live-cloud` | NOT RUN — blocked by BYO failure earlier in `make smoke-live` chain. Cloud path would have succeeded (cloud-init configures `(ALL) NOPASSWD: ALL` on runnerkit-admin which covers `runas=runnerkit-runner`); not run because v1 cannot ship with BYO non-functional, regardless of cloud success. | SKIP |
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
| ~~`internal/preflight/checks.go`~~ | ~~127~~ | Binary-presence check masquerading as capability check | RESOLVED | Closed by Plan 06-05 commit 314bf94 (real `sudo -n true` probe with passwordless / password-required / no-sudo / sudo-missing branches). |
| ~~`internal/bootstrap/install.go`~~ | ~~74, 115~~ | `sudo install -d -o serviceUser` followed by plain `curl`/`sha256sum`/`tar` | RESOLVED | Closed by Plan 06-05 commit 75c41aa (sudo-prefixed download/verify/extract; integration test added). |
| ~~`internal/bootstrap/script.go`~~ (download_runner) | ~~lines 42-45 / 78-81~~ | Same pattern duplicated in script-renderer path | RESOLVED | Closed by Plan 06-05 commit 75c41aa (same fix applied to both renderer paths). |
| `internal/bootstrap/script.go` (register_runner) | 47, 83 | `sudo -u runnerkit-runner ./config.sh ...` — runs as non-root user; requires `(ALL)` runas in sudoers; both root NOPASSWD and byo-prepare scoped template only cover runas=root | BLOCKER | Bug 3. BYO non-functional in v1 at registration step despite Plans 06-05 + 06-06 closing Bugs 1+2. Cloud path unaffected (runnerkit-admin has broader `(ALL) NOPASSWD: ALL`). |
| `internal/bootstrap/sudoers.go` | 23 | RenderSudoersEntry uses `ALL=(root) NOPASSWD:` runas — does not cover `sudo -u runnerkit-runner` | BLOCKER | Even after byo-prepare runs, register_runner still requires password. Same root cause as the register_runner bug above. |
| Test gap | `internal/bootstrap/script_test.go` | Substring assertions cover `sudo curl|sha256sum|tar` from Bug 2 fix but never asserted absence of `sudo -u <non-root>` patterns. No fixture for SSH user with only root NOPASSWD. | WARNING | Allowed Bug 3 to escape past Plans 06-05 + 06-06 verification into live smoke. Task F closure adds presence/absence assertions + integration test fixture. |

### Human Verification Required (post-gap-closure)

| Test | Expected | Why Human |
| ---- | -------- | --------- |
| `make smoke-live` against real BYO + real Hetzner project after Task F (Plan 06-08) closes | BYO completes end-to-end without manual sudoers preconfiguration; Hetzner D-12 gates 1+2 PASS; total wall-clock ≤ 10 min on each path | Real GitHub PAT + real billable Hetzner resources; Claude cannot exercise the OIDC + billing surfaces. Maintainer fills 06-VERIFICATION + RELEASE-NOTES, types `smoke-green`. |

## Gaps Summary

The phase delivers v1.0.0 release readiness almost entirely:

- **Release pipeline (REL-05 distribution):** complete and locally verifiable. Tag push will produce signed multi-platform release.
- **Upgrade lifecycle (REL-05 lifecycle):** complete. Forward-only migrations + lazy update notice + channel-detect upgrade + upgrade-runner + stale doctor finding all wired.
- **Cleanup/troubleshooting docs (DOC-04):** complete. RKD code registry + 6 component docs + URL env override + 5 required tests green.
- **Live validation harness scaffolding:** complete. Makefile + 4 shell wrappers + 2 smoke binaries (with passing unit tests) + stopwatch checklist + RELEASE-NOTES template + 06-VERIFICATION skeleton all in place. D-11 enforced (no smoke-live in workflows). D-12 gates 1+2 wired and unit-tested.

**What's blocking v1.0.0 tag push:**

Bugs 1 + 2 are CLOSED by Plans 06-05 + 06-06 (commits ee5c0a2 + 08b8708, 2026-05-04/05). Plan 06-07 attempt-1 re-smoke against `salar@mckee-small-desktop` on 2026-05-05 verified those fixes (sudo probe surfaced no false positive; `download_runner` extracted `config.sh` to `/opt/actions-runner/runnerkit-<owner>-<repo>-local/` successfully). Then surfaced a third latent bug:

3. **Register_runner runas mismatch (Bug 3).** `internal/bootstrap/script.go:47` (RenderInstallScript) and `:83` (RenderEphemeralInstallScript) invoke `sudo -u runnerkit-runner RUNNERKIT_REGISTRATION_TOKEN=... ./config.sh --unattended --url ... --token ... --name ... --labels ... --work ... --replace` to register the GitHub runner as the unprivileged service user. `sudo -u <non-root>` matches the `(ALL)` runas spec, NOT `(root)`. The byo-prepare scoped sudoers template (`internal/bootstrap/sudoers.go:23`) and typical system-wide configurations of the form `(root) NOPASSWD: ALL` cover runas=root only. Result: `sudo: a terminal is required to read the password sudo: a password is required` from inside the install script; bootstrap fails after `download_runner` succeeds. Cloud path unaffected because `internal/provider/hetzner/provision.go:241` cloud-init configures `runnerkit-admin` with `sudo: ALL=(ALL) NOPASSWD:ALL` — the `(ALL)` runas covers `runas=runnerkit-runner`. Bug went undetected until live smoke because (a) script_test.go substring assertions never asserted absence of `sudo -u <non-root>` patterns, (b) install_integration_test.go from Plan 06-05 only fixtured `download_runner`, not `register_runner`, (c) every other bootstrap test uses fakeExecutor that records commands but never executes them.

`06-GAP-byo-sudo-handling.md` Task F (added 2026-05-05) is the actionable artifact:
- Replace `sudo -u runnerkit-runner ./config.sh ...` with `sudo su -s /bin/bash - runnerkit-runner -c '<config.sh ...>'`. `su` runs from a root sudo context → no `(ALL)` runas required → works on BYO host with only root NOPASSWD AND on cloud host with broader NOPASSWD. The byo-prepare scoped sudoers template needs no change.
- Extend `script_test.go` substring assertions: presence of `su -s /bin/bash`, absence of `sudo -u`.
- Extend `install_integration_test.go` with a fixture that asserts the new shell form is acceptable to a sudoers configuration consisting only of `(root) NOPASSWD: ALL`.
- Re-run Plan 06-07 BYO smoke; assert a GitHub runner ID lands before destroy.

Plan 06-08 is the gap-closure target. Once Task F lands, Plan 06-07 re-runs end-to-end against `salar@mckee-small-desktop`, the maintainer fills the 06-VERIFICATION baseline + RELEASE-NOTES-v1.0.0.md numbers, and Phase 6 closes with a `smoke-green` resume signal that triggers the v1.0.0 tag push per `docs/release-process.md`.

---

_Verified: 2026-05-02; updated 2026-05-05 with Bug 3 finding from Plan 06-07 attempt-1 re-smoke._
_Verifier: Claude (gsd-verifier)._
_Source of truth for the gap: 06-GAP-byo-sudo-handling.md (Tasks A-E CLOSED by 06-05+06-06; Task F OPEN, target Plan 06-08)._
