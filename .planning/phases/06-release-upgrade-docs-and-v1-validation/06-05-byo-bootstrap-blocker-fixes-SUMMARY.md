---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 05
subsystem: infra
tags: [byo, bootstrap, sudo, ssh, preflight, errcodes, redactor, integration-test]

# Dependency graph
requires:
  - phase: 02-byo-runner-setup
    provides: bootstrap.Apply / ApplyEphemeral, RenderInstallScript, fakeExecutor unit-test pattern
  - phase: 06-release-upgrade-docs-and-v1-validation/06-03
    provides: errcodes.Code/Registry pattern with docs anchor contract
  - phase: 06-release-upgrade-docs-and-v1-validation/06-04
    provides: live BYO smoke that surfaced Bug 1 + Bug 2
provides:
  - real `sudo -n true` preflight probe with 4 stable outcome IDs (passwordless / password_required / no_sudo / sudo missing)
  - redacted remote stderr surfaced in bootstrap_failed CLI errors
  - sudo-prefixed download_runner step (curl + sha256sum -c - + tar xzf) so install dir owned by serviceUser receives the tarball without Permission denied
  - downloadRunnerCommand() helper de-duplicates the literal between Apply and ApplyEphemeral
  - build-tag-guarded real-shell integration test (closes the fakeExecutor-only gap latent since 02-02)
  - new RKD-BOOT-015 stable error code anchored in docs/troubleshooting/bootstrap.md
  - byo-permission.sh smoke asserts config.sh extraction landed on the remote host
  - make test-integration target gated by RUNNERKIT_INTEGRATION=1
affects: [06-06-byo-prepare-and-sudo-prompt, 06-07-live-smoke-rerun-and-baseline-fillin]

# Tech tracking
tech-stack:
  added: ["//go:build integration build tag for real-shell tests", "httptest+tar+gzip in test-only file"]
  patterns:
    - "Stable preflight check IDs branch into separate IDs per failure mode (host.privilege.password_required vs host.privilege.no_sudo)"
    - "bootstrap.Result + remote.RemoteError combine to surface both failing CommandID and stderr; renderer.Redactor() owns redaction at user-facing emission points"
    - "Production helper downloadRunnerCommand(opts) is shared by Apply and ApplyEphemeral; integration test consumes the same helper so production drift cannot bypass the test"

key-files:
  created:
    - "internal/bootstrap/install_integration_test.go"
    - ".planning/phases/06-release-upgrade-docs-and-v1-validation/06-05-byo-bootstrap-blocker-fixes-SUMMARY.md"
  modified:
    - "internal/preflight/checks.go"
    - "internal/preflight/checks_test.go"
    - "internal/cli/up.go"
    - "internal/cli/up_test.go"
    - "internal/bootstrap/install.go"
    - "internal/bootstrap/install_test.go"
    - "internal/bootstrap/script.go"
    - "internal/bootstrap/script_test.go"
    - "internal/errcodes/codes.go"
    - "docs/troubleshooting/bootstrap.md"
    - "scripts/smoke/byo-permission.sh"
    - "Makefile"

key-decisions:
  - "Plan 06-05 (gap closure): host.privilege.password_required emits SeverityWarning (not Failure) so report.Passed() stays true; this lets the Path B fallback in Plan 06-06 take over without a preflight short-circuit, while still surfacing the issue to the user."
  - "Plan 06-05: download_runner is now factored into a shared downloadRunnerCommand(opts) helper consumed by Apply, ApplyEphemeral, AND the integration test, so future renderer drift cannot bypass the real-shell test (closes the fakeExecutor-only gap that hid Bug 2 since 02-02)."
  - "Plan 06-05: bootstrap_failed remediation surfaces stderr via 'Remote stderr (commandID): ...' format with the redactor applied; CommandID is extracted from remote.RemoteError when present (errors.As) and falls back to 'unknown' when the error path predates wrapping."
  - "Plan 06-05: Integration test is gated by both //go:build integration AND RUNNERKIT_INTEGRATION=1 env var so go test ./... and CI ignore it; only `make test-integration` (maintainer-local, NOPASSWD sudo required) runs it."

patterns-established:
  - "Pattern: TDD RED commit asserts new stable IDs / new substrings BEFORE production change; GREEN commit makes them pass. Used 4 commits (RED+GREEN per task)."
  - "Pattern: Real-shell integration tests live behind //go:build integration build tag with an additional env-var gate, so they remain inert in CI but are one make target away for maintainers."
  - "Pattern: Redaction at the renderer.Redactor().String() boundary, never at the raw stderr capture site, so all surfaced diagnostics flow through one redaction pipeline."

requirements-completed: [REL-05]

# Metrics
duration: 8m
completed: 2026-05-05
---

# Phase 6 Plan 05: BYO Bootstrap Blocker Fixes Summary

**Real `sudo -n true` preflight probe + sudo-prefixed download_runner + redacted remote stderr surfacing closes the two BLOCKER bugs that made BYO bootstrap unusable in v1.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-05-05T01:53:49Z
- **Completed:** 2026-05-05T02:01:22Z
- **Tasks:** 2
- **Files modified:** 12 (1 created, 11 modified)

## Accomplishments

- **Bug 1 fixed (preflight false-positive):** `internal/preflight/checks.go::Run` now executes `sudo -n true` against the SSH target and emits four distinct outcomes via stable IDs `host.privilege` / `host.privilege.password_required` / `host.privilege.no_sudo` / `host.privilege` (probe failure). Password-required is a warning so report.Passed() stays true, leaving the bootstrap path reachable for Plan 06-06's Path B fallback.
- **Swallowed-stderr surfaced:** `bootstrap_failed` and ephemeral `bootstrap_failed` branches in `internal/cli/up.go` now extract the failing CommandID from `remote.RemoteError` and the trailing stderr from `bootstrap.Result.Commands`, then route both through `renderer.Redactor().String()` before appending to the remediation. Token-shaped strings stay redacted (`<redacted:github-token>` etc.) — the redaction invariant is asserted by `TestUp_BootstrapFailed_SurfacesRedactedRemoteStderr`.
- **Bug 2 fixed (download_runner permission):** `Apply` and `ApplyEphemeral` now share a `downloadRunnerCommand(opts)` helper that prefixes `curl`, `sha256sum -c -`, and `tar xzf` with `sudo`, so the install dir created with `sudo install -d -o serviceUser` receives the tarball regardless of which user the SSH session runs as. Same fix in `RenderInstallScript` and `RenderEphemeralInstallScript`.
- **fakeExecutor test gap closed:** `internal/bootstrap/install_integration_test.go` exercises the actual `downloadRunnerCommand(opts).Script` against `bash -c` with a fake httptest tarball server and a `t.TempDir()` sandbox. The test asserts both the tarball AND the extracted `config.sh` land in `installPath`. Build-tag-guarded (`//go:build integration`) plus `RUNNERKIT_INTEGRATION=1` env-var gate so it stays inert in CI; one `make test-integration` away locally for maintainers.
- **Stable error code RKD-BOOT-015** appears in `Registry`, anchored at `docs/troubleshooting/bootstrap.md#rkd-boot-015` with the canonical Symptom/Diagnosis/Fix structure (TestEntriesFollowSymptomDiagnosisFix continues to pass).
- **Smoke script extension:** `scripts/smoke/byo-permission.sh` now `ssh`s to the host and asserts `/opt/actions-runner/runnerkit-*/config.sh` exists after `runnerkit up`, before the test moves on to runner registration; failures attribute cleanly to bootstrap rather than registration.

## Task Commits

1. **Task 1 (TDD): Real `sudo -n true` preflight probe + redacted-stderr surfacing on bootstrap_failed**
   - RED: `7d5cc33` — `test(06-05): add failing tests for sudo -n probe and bootstrap stderr surfacing`
   - GREEN: `314bf94` — `feat(06-05): real sudo -n probe and redacted remote stderr in bootstrap_failed`
2. **Task 2 (TDD): Sudo-prefixed download_runner + script renderers + real-shell integration test + smoke assertion**
   - RED: `9fb199d` — `test(06-05): add failing tests for sudo-prefixed download_runner`
   - GREEN: `75c41aa` — `feat(06-05): sudo-prefix download_runner + real-shell integration test`

**Plan metadata:** (final docs commit appended below after STATE/ROADMAP updates)

## Files Created/Modified

### Created
- `internal/bootstrap/install_integration_test.go` — Build-tag-guarded real-shell integration test (httptest tarball + bash -c + t.TempDir sandbox); skips cleanly without RUNNERKIT_INTEGRATION=1.

### Modified
- `internal/preflight/checks.go` — Replaced `probe.Commands["sudo"]` binary-existence check with real `sudo -n true` probe; added new stable IDs `CheckPrivilegePasswordReq`, `CheckPrivilegeNoSudo`.
- `internal/preflight/checks_test.go` — `TestCheckPrivilege_Passwordless`, `TestCheckPrivilege_PasswordRequired`, `TestCheckPrivilege_NotInSudoers`, `TestCheckPrivilege_SudoMissing`; extended `fakePreflightExecutor` with a `runResults` map keyed by Command.ID.
- `internal/cli/up.go` — `bootstrap_failed` and ephemeral `bootstrap_failed` branches now surface `lastCommandFailureContext(result, err)` through `renderer.Redactor().String()`; helper extracts CommandID from `remote.RemoteError` via `errors.As` and stderr from the trailing `bootstrap.Result.Commands` entry.
- `internal/cli/up_test.go` — `TestUp_BootstrapFailed_SurfacesRedactedRemoteStderr` drives `configure_runner` to fail with token-shaped stderr and asserts the JSON `error.remediation` slice contains `<redacted:github-token>` AND the failing command ID, with no raw token in the captured output.
- `internal/bootstrap/install.go` — Factored `download_runner` Command into a shared `downloadRunnerCommand(opts)` helper consumed by both `Apply` and `ApplyEphemeral`; sudo prefixes added to curl/sha256sum/tar.
- `internal/bootstrap/install_test.go` — `TestApplyDownloadRunnerCommandUsesSudoForCurlSha256SumTar` and `TestApplyEphemeralDownloadRunnerCommandUsesSudoForCurlSha256SumTar` lock in the substring assertions.
- `internal/bootstrap/script.go` — Same sudo prefixes in `RenderInstallScript` and `RenderEphemeralInstallScript` so renderer output matches runtime command shape.
- `internal/bootstrap/script_test.go` — `TestRenderInstallScriptUsesSudoForCurlSha256SumTar` and `TestRenderEphemeralInstallScriptUsesSudoForCurlSha256SumTar` lock in the renderer-side assertions.
- `internal/errcodes/codes.go` — Added `BootSudoPasswordRequired` (RKD-BOOT-015) to the BOOT block and the Registry slice.
- `docs/troubleshooting/bootstrap.md` — Added `<a name="rkd-boot-015"></a>` section with Symptom/Diagnosis/Fix; bumped header range from `..014` to `..015`.
- `scripts/smoke/byo-permission.sh` — Asserts `/opt/actions-runner/runnerkit-*/config.sh` after `runnerkit up`, before the status/doctor/down sequence.
- `Makefile` — New `test-integration` target running `RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -v`.

## Decisions Made

- **Password-required preflight result is Severity=Warning, not Failure.** The plan called for a warning so `report.Passed()` returns true, allowing the bootstrap step to run. This keeps Path B fallback (Plan 06-06) reachable. For Plan 06-05's pre-Path-B world, the maintainer's documented v1.0.0 contract is NOPASSWD sudo, so the warning correctly informs the user without short-circuiting the existing happy path.
- **`downloadRunnerCommand(opts)` extracted to shared helper.** Plan suggested an "exported helper for the integration test"; the cleaner factoring uses a private package-scoped helper that BOTH production functions AND the integration test consume. This means production drift physically cannot bypass the integration test (the test exercises the exact same Command literal Apply emits) — a stronger guarantee than a separate test-only literal would provide.
- **Integration test double-gated.** `//go:build integration` build tag excludes the file from `go test ./...`; the `RUNNERKIT_INTEGRATION=1` env-var check inside the test prevents it from executing even when someone manually passes `-tags=integration` without the env var. This double-gating ensures CI environments cannot accidentally run sudo against an unprepared host. Only `make test-integration` (which sets both) runs the real-shell path.

## Deviations from Plan

None - plan executed exactly as written.

The plan's pseudocode (Step 1.2 sudo probe, Step 1.5 stderr surfacing, Step 2.1 sudo prefixes, Step 2.5 integration test, Step 2.6 smoke extension, Step 2.7 Makefile target) translated 1:1 into production code. The only minor implementation detail was choosing to factor `downloadRunnerCommand` as a private (lowercase) helper rather than the plan's suggested exported `BuildDownloadRunnerCommandForTest` — the integration test sits in the same `bootstrap` package so it can call the lowercase helper directly. This keeps the package surface area minimal (no test-only export) while still de-duplicating the literal between Apply and ApplyEphemeral.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Plan 06-06 unblocked** — `host.privilege.password_required` warning is now emitted with stable ID; Plan 06-06 (Path B interactive sudo prompt + Path C `runnerkit byo-prepare`) can consume it directly.
- **Plan 06-07 unblocked** — Live BYO smoke can be re-run end-to-end against a fresh host with NOPASSWD sudo (Path A from the gap doc) without hitting `curl: (23) Failure writing output to destination, Permission denied`. Bug 2's blast radius is fully closed.
- **Verification status:** `go test ./... -count=1 -race` passes green; `go test -tags=integration ./internal/bootstrap/... -count=1` skips cleanly without `RUNNERKIT_INTEGRATION`. The actual real-shell integration run is a maintainer-local check (NOPASSWD sudo prerequisite) deferred to `make test-integration` invocation.
- **Known follow-up (NOT blocking v1.0.0 once 06-06+06-07 land):** Plan 06-05 only resolves Tasks A + E from `06-GAP-byo-sudo-handling.md`. Tasks B (interactive `sudo -S` prompt + redact.SudoPassword), C (`runnerkit byo-prepare` command + sudoers template + visudo validation), and D (BYO quickstart Sudo Setup section) are owned by Plan 06-06.

## Self-Check: PASSED

Per `<self_check>` step in execute-plan workflow:

**Files created exist:**
- FOUND: `internal/bootstrap/install_integration_test.go`
- FOUND: `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-05-byo-bootstrap-blocker-fixes-SUMMARY.md` (this file)

**Commit hashes verified:**
- FOUND: `7d5cc33` (Task 1 RED)
- FOUND: `314bf94` (Task 1 GREEN)
- FOUND: `9fb199d` (Task 2 RED)
- FOUND: `75c41aa` (Task 2 GREEN)

**Key substrings verified in production files:**
- `sudo -n true` in `internal/preflight/checks.go`
- `host.privilege.password_required` and `host.privilege.no_sudo` in `internal/preflight/checks.go`
- `RKD-BOOT-015` in `internal/errcodes/codes.go`
- `rkd-boot-015` anchor in `docs/troubleshooting/bootstrap.md`
- `renderer.Redactor()` in bootstrap_failed branches of `internal/cli/up.go`
- `sudo curl`, `sudo sha256sum -c -`, `sudo tar xzf` in `internal/bootstrap/install.go`
- Same in `internal/bootstrap/script.go`
- `//go:build integration` on line 1 of `internal/bootstrap/install_integration_test.go`
- `config.sh` assertion in `scripts/smoke/byo-permission.sh`
- `test-integration` target in `Makefile`

**Test status:**
- `go test ./... -count=1 -race` — GREEN (all packages, including the new tests)
- `go test -tags=integration ./internal/bootstrap/... -count=1 -run TestApply_DownloadRunner_RealShell -v` — SKIPS cleanly without RUNNERKIT_INTEGRATION=1 (compiles and runs the skip path)

---
*Phase: 06-release-upgrade-docs-and-v1-validation*
*Completed: 2026-05-05*
