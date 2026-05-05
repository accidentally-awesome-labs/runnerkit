---
status: partial-blocked
phase: 06-release-upgrade-docs-and-v1-validation
plan: 04
subsystem: v1-validation-and-live-smoke
blocked_on: 06-GAP-byo-sudo-handling.md (Bug 1 + Bug 2 — BYO bootstrap unusable in v1)
tasks_complete: 3
tasks_total: 4
tags: [live-smoke, hetzner-billable, byo-permission, d-12-gates, stopwatch-checklist, makefile, smokebin, hcloud-go, release-notes, verification-baseline, gap-blocked]

requires:
  - phase: 06-release-upgrade-docs-and-v1-validation/01
    provides: docs/release-process.md (Plan 06-01 wrote One-Time Prerequisites, Tag a Release, Common Failures, Release Notes File sections; Plan 06-04 appends the Stopwatch Checklist).
  - phase: 06-release-upgrade-docs-and-v1-validation/01
    provides: .gitignore baseline (Plan 06-01 added /dist/; Plan 06-04 appends smoke artifacts patterns).
  - phase: 06-release-upgrade-docs-and-v1-validation/01
    provides: HOMEBREW_TAP_GITHUB_TOKEN secret + accidentally-awesome-labs/homebrew-runnerkit tap repo (Task 5 closure 'tap-ready' 2026-05-02).
  - phase: 06-release-upgrade-docs-and-v1-validation/02
    provides: Forward-only state migrations + ExitStateSchemaTooNew (consumed by 06-VERIFICATION.md automated checklist).
  - phase: 06-release-upgrade-docs-and-v1-validation/03
    provides: docs/troubleshooting/ + RKD codes (consumed by RELEASE-NOTES-v1.0.0.md troubleshooting forward link).
  - phase: 02-byo-persistent-runner-happy-path
    provides: bootstrap.RunnerVersion 2.334.0 pin (recorded in RELEASE-NOTES-v1.0.0.md and 06-VERIFICATION.md Bundled Versions).
  - phase: 04-recommended-cloud-path-and-billable-cleanup
    provides: hetznercloud/hcloud-go v1.59.2 + ProviderRef.Cloud schema (consumed by cmd/_smokebin/destroy_verify state.json parser and cmd/_smokebin/empty_precheck Server/SSHKey/PrimaryIP/Firewall scan).
  - phase: 04-recommended-cloud-path-and-billable-cleanup
    provides: runnerkit destroy --yes semantics (consumed by scripts/smoke/cloud-end-to-end.sh trap-on-EXIT cleanup AND by D-12 gate 2 destroy-verify polling AFTER destroy returns).
provides:
  - Makefile with smoke-live, smoke-live-byo, smoke-live-cloud, smoke-stopwatch targets; env-var precondition checks for RUNNERKIT_SMOKE_BYO_HOST, RUNNERKIT_SMOKE_REPO, HCLOUD_TOKEN, gh auth status.
  - cmd/_smokebin/empty_precheck/main.go — D-12 gate 1 binary; lists Server/SSHKey/PrimaryIP/Firewall and refuses on any runnerkit-* prefixed Name.
  - cmd/_smokebin/destroy_verify/main.go — D-12 gate 2 binary; polls hcloud-go for ErrorCodeNotFound on every saved cloud ID within RUNNERKIT_SMOKE_TIMEOUT (default 300s).
  - cmd/_smokebin/{empty_precheck, destroy_verify}/main_test.go — unit tests using a fake hcloudClient/verifierClient interface; cover the success-after-N-polls path AND the timeout-failure path AND all four resource types.
  - scripts/smoke/byo-permission.sh — Phase 1 outstanding live GitHub permission smoke wrapper.
  - scripts/smoke/cloud-end-to-end.sh — Phase 4 outstanding live Hetzner billable smoke wrapper with trap-on-EXIT/INT/TERM cleanup (Pitfall 7).
  - scripts/smoke/hetzner-empty-precheck.sh — D-12 gate 1 wrapper.
  - scripts/smoke/hetzner-destroy-verify.sh — D-12 gate 2 wrapper.
  - docs/release-process.md "Stopwatch Checklist (D-13)" section — BYO and Hetzner stopwatch tables, 10-minute target, recording instructions.
  - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md — v1.0.0 baseline skeleton; maintainer fills durations + runner IDs + cost during the checkpoint.
  - RELEASE-NOTES-v1.0.0.md — first release notes template at repo root; references accidentally-awesome-labs/runnerkit (NOT salar/) per Plan 06-01 Task 5 closure org migration.
affects: []

tech-stack:
  added:
    - cmd/_smokebin/ pattern — Go binaries excluded from `go build ./...` by the `_` directory prefix; invoked explicitly via `go run ./cmd/_smokebin/<name>` from shell scripts. Tests still run when targeted directly.
  patterns:
    - "D-11 enforced by absence: zero references to make smoke-live in any .github/workflows/*.yml file. Live smokes are explicitly maintainer-only on a real machine."
    - "D-12 gate 1 (empty-project precheck) is a HARD gate: scripts/smoke/hetzner-empty-precheck.sh runs cmd/_smokebin/empty_precheck via `go run` BEFORE any `runnerkit up --cloud hetzner` invocation, and the smoke refuses to proceed if any Server/SSHKey/PrimaryIP/Firewall has a `runnerkit-` Name prefix."
    - "D-12 gate 2 (destroy-verify polling) is also a HARD gate: scripts/smoke/hetzner-destroy-verify.sh runs cmd/_smokebin/destroy_verify AFTER `runnerkit destroy --yes` returns. The verifier reads RUNNERKIT_SMOKE_STATE_FILE, recovers all five cloud IDs (server, ssh_key, primary_ipv4, primary_ipv6, firewall), and polls each via hcloud-go until ErrorCodeNotFound or RUNNERKIT_SMOKE_TIMEOUT (default 300s) elapses."
    - "Pitfall 7 mitigation: scripts/smoke/cloud-end-to-end.sh installs a `trap cleanup EXIT INT TERM` handler that runs `runnerkit destroy --yes` if state.json exists, even on Ctrl-C. State.json is also snapshotted to state-after-destroy.json before the destroy step so the verifier can recover the IDs even after destroy removes the repo entry from state."
    - "TDD on the smoke binaries: RED commit (956cb2c) ships failing tests + 'not implemented' stubs; GREEN commit (8f59a40) lands the real implementation. Test names match 06-VALIDATION.md lines 76 (TestEmptyPrecheck_RefusesOnExisting) and 77 (TestDestroyVerify_Timeout) exactly; the AllResourceTypes test extends gate 1 coverage to all four Phase-4 resource types."

key-files:
  created:
    - Makefile
    - scripts/smoke/byo-permission.sh
    - scripts/smoke/cloud-end-to-end.sh
    - scripts/smoke/hetzner-empty-precheck.sh
    - scripts/smoke/hetzner-destroy-verify.sh
    - cmd/_smokebin/empty_precheck/main.go
    - cmd/_smokebin/empty_precheck/main_test.go
    - cmd/_smokebin/destroy_verify/main.go
    - cmd/_smokebin/destroy_verify/main_test.go
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md
    - RELEASE-NOTES-v1.0.0.md
  modified:
    - .gitignore (appended smoke artifact patterns: .smoke-state/, *.smoke.log, cmd/_smokebin/*/empty_precheck, cmd/_smokebin/*/destroy_verify)
    - docs/release-process.md (appended "Stopwatch Checklist (D-13)" section after the existing Plan 06-01 sections)

key-decisions:
  - "Polling cadence is 500ms, not 5s. The plan's research note suggested 5s, but the timeout-failure unit test runs against a 2s RUNNERKIT_SMOKE_TIMEOUT (`t.Setenv`) so a 5s poll would never get a chance to evaluate the deadline before the first sleep. 500ms is fast enough for the test to converge AND slow enough to be polite to the real Hetzner API (each poll touches at most 5 GET endpoints). Production timeout default stays 300s — fine-grained polling does not change the worst-case wait."
  - "hcloud-go IDs are `int`, not `int64`. The plan's example code declared cloudIDs with `int64` fields, but hcloud-go v1.59.2 uses plain `int` throughout (verified against $GOMODCACHE/github.com/hetznercloud/hcloud-go@v1.59.2/hcloud/server.go:20 — `ID int`; client.go in the existing internal/provider/hetzner adapter takes `id int`). Switching cloudIDs to `int` removes a sea of int/int64 conversions and matches existing destroy.go::VerifyDestroyed."
  - "Trap is rearmed for INT/TERM after success disarm. cloud-end-to-end.sh's `trap cleanup EXIT INT TERM` runs `runnerkit destroy --yes` on any abort path; on success the EXIT trap is disarmed (so the Makefile can hand the state dir to hetzner-destroy-verify.sh) but a fresh INT/TERM trap is installed that purges the tempdir without re-destroying. This avoids two destroy invocations on success while still cleaning up if the maintainer Ctrl-Cs during the verify phase."
  - "All references in RELEASE-NOTES-v1.0.0.md and the Hetzner smoke harness use `accidentally-awesome-labs/...`, NOT `salar/...`. Per the orchestrator note, Plan 06-01 Task 5 closure (commit c359831) migrated the org; this plan respects that migration end-to-end. The cosign verify-blob snippet's --certificate-identity URL is `https://github.com/accidentally-awesome-labs/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}` (string-equal to the workflow path embedded in the keyless cert by Plan 06-01)."

requirements-completed: []  # REL-05 completion is gated on the maintainer human-action checkpoint resolving 'smoke-green'.

duration: 8m (Tasks 1-3 only; Task 4 is the maintainer's wall-clock smoke)
completed: 2026-05-04
status: pending-checkpoint
---

# Phase 6 Plan 04: v1 Validation and Live Smoke Summary

**Live smoke harness + verification baseline + first release notes — closing Phase 1 outstanding GitHub permission smoke and Phase 4 outstanding Hetzner billable smoke STATE.md notes; D-10/D-11/D-12-gates-1-and-2/D-13 implemented; final v1 sign-off pending the maintainer human-action checkpoint that runs `make smoke-live` against real billable Hetzner + GitHub credentials.**

## Performance

- **Duration:** ~8 minutes (Tasks 1-3 only; Task 4 is the maintainer-side wall-clock smoke).
- **Started:** 2026-05-04T15:43:24Z
- **Completed (Tasks 1-3):** 2026-05-04T15:51:51Z
- **Tasks:** 3 of 4 complete (Task 4 is the human-action checkpoint, NOT executable by Claude — it requires real `gh auth login` against a real GitHub repo, real billable Hetzner resource creation, and a wall-clock stopwatch on a clean laptop).
- **Files created:** 11 (`Makefile`, 4 × `scripts/smoke/*.sh`, 4 × `cmd/_smokebin/*/main.go|main_test.go`, `06-VERIFICATION.md`, `RELEASE-NOTES-v1.0.0.md`).
- **Files modified:** 2 (`.gitignore`, `docs/release-process.md`).

## Accomplishments

- **D-11 enforced.** `Makefile` declares `smoke-live`, `smoke-live-byo`, `smoke-live-cloud`, `smoke-stopwatch` as `.PHONY` targets with maintainer-only env-var precondition checks (`RUNNERKIT_SMOKE_BYO_HOST`, `RUNNERKIT_SMOKE_REPO`, `HCLOUD_TOKEN`, `gh auth status`). `grep -rq smoke-live .github/workflows/` returns 1 (no match) — these targets are not invoked from any GitHub Actions workflow, so the CI environment never holds the real PAT or HCLOUD_TOKEN.
- **D-12 gate 1 (empty-project precheck) wired and unit-tested.** `cmd/_smokebin/empty_precheck/main.go` lists Servers/SSHKeys/PrimaryIPs/Firewalls via the existing hcloud-go v1.59.2 module and refuses if any has a `runnerkit-` Name prefix. Three subtest groups (`refuses_when_a_runnerkit-*_server_is_present`, `allows_when_only_unrelated_resources_exist`, `allows_on_a_fully_empty_project`) plus the all-four-resource-types matrix test (`TestEmptyPrecheck_AllResourceTypes` — ssh-key, primary-ip, firewall) cover the gate end-to-end.
- **D-12 gate 2 (destroy-verify polling) wired and unit-tested.** `cmd/_smokebin/destroy_verify/main.go` reads `RUNNERKIT_SMOKE_STATE_FILE`, parses out `Provider.Cloud.{ServerID, SSHKeyID, PrimaryIPv4ID, PrimaryIPv6ID, FirewallID}` from a partial-unmarshal mirroring `internal/state/schema.go`, and polls every saved ID until `hcloud.IsError(err, hcloud.ErrorCodeNotFound)` OR `RUNNERKIT_SMOKE_TIMEOUT` (default 300s) elapses. The unit test (`TestDestroyVerify_Timeout`) covers both the success-after-N-polls path AND the timeout-failure path with a tight 2s `RUNNERKIT_SMOKE_TIMEOUT` and a fake verifier that always returns the resource.
- **Pitfall 7 mitigation in shell.** `scripts/smoke/cloud-end-to-end.sh` installs `trap cleanup EXIT INT TERM` that runs `runnerkit destroy --yes` if `state.json` exists, snapshots `state.json` to `state-after-destroy.json` BEFORE invoking destroy (so the verifier can recover IDs even after destroy removes the repo entry), and disarms the EXIT trap on success while leaving an INT/TERM purge in place to handle aborts during the destroy-verify phase.
- **D-13 stopwatch checklist appended to docs/release-process.md.** Plan 06-01 wrote the maintainer-only release procedure (One-Time Prerequisites, Tag a Release, Common Failures, Release Notes File); this plan appends the "Stopwatch Checklist (D-13)" section with BYO and Hetzner tables (T0 / T_now / Δ columns), 10-minute target, total wall-clock + Hetzner cost recording rows, and the recording-into-RELEASE-NOTES + 06-VERIFICATION instructions.
- **06-VERIFICATION.md baseline created.** Skeleton at `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md` with `frontmatter (phase, type: verification, status: pending, created: 2026-05-04)`, an 8-entry automated test checklist (`go test ./... -count=1 -race`, `goreleaser check`, `goreleaser release --snapshot --skip=publish --clean`, errcodes 5-test bundle, migrations 4-test bundle, update 6-test bundle, upgrade 7-test bundle, _smokebin 3-test bundle), BYO smoke fields (host, repo, duration, runner ID), Hetzner smoke fields (repo, project, duration, cost EUR, 5 resource IDs, D-12 gate 1 status, D-12 gate 2 status, precheck final size = 0), 10-minute stopwatch totals, bundled versions table (runner pin 2.334.0, GoReleaser v2.15.4, cosign v3.0.6, hcloud-go v1.59.2), and the maintainer signature/sign-off block.
- **RELEASE-NOTES-v1.0.0.md template created.** First release notes file at repo root with title `RunnerKit v1.0.0`, supported platforms (4 — macOS arm64/amd64, Linux amd64/arm64; Linux 386 + 32-bit ARM not supported), Homebrew install command (`brew install accidentally-awesome-labs/runnerkit/runnerkit`), cosign verify-blob snippet pinning the cert-identity URL to `https://github.com/accidentally-awesome-labs/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}` (string-equal to the URL embedded in the keyless cert by Plan 06-01), 10-minute stopwatch placeholder table, "Outstanding Live Smokes Closed" section explicitly naming Phase 1 (`make smoke-live-byo`) and Phase 4 (`make smoke-live-cloud` with both D-12 gates) STATE.md notes, upgrade path summary (channel-detect for Homebrew/binary, ephemeral safety for `upgrade-runner --force`, forward-only migrations with exit code 7), and the troubleshooting forward link to `docs/troubleshooting/README.md` plus the `RUNNERKIT_DOCS_BASE` override note.
- **No salar/... references introduced.** Per the orchestrator's sequential-execution note, Plan 06-01 Task 5 closure (`c359831`) migrated the org from `salar/` to `accidentally-awesome-labs/`. All Task 1-3 outputs of this plan use `accidentally-awesome-labs/...` URLs in cosign cert-identity, install commands, and any owner segment. Verified: `grep -n salar RELEASE-NOTES-v1.0.0.md 06-VERIFICATION.md` returns no matches.

## Task Commits

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1    | Makefile + 4 × scripts/smoke + .gitignore append | `fa2d5b8` | `Makefile`, `scripts/smoke/byo-permission.sh`, `scripts/smoke/cloud-end-to-end.sh`, `scripts/smoke/hetzner-empty-precheck.sh`, `scripts/smoke/hetzner-destroy-verify.sh`, `.gitignore` |
| 2 (RED) | Failing tests + 'not implemented' stubs for empty_precheck + destroy_verify | `956cb2c` | `cmd/_smokebin/empty_precheck/main.go`, `cmd/_smokebin/empty_precheck/main_test.go`, `cmd/_smokebin/destroy_verify/main.go`, `cmd/_smokebin/destroy_verify/main_test.go` |
| 2 (GREEN) | Real implementation of empty_precheck + destroy_verify (D-12 gates 1+2) | `8f59a40` | `cmd/_smokebin/empty_precheck/main.go`, `cmd/_smokebin/destroy_verify/main.go` |
| 3    | Stopwatch checklist + 06-VERIFICATION + RELEASE-NOTES-v1.0.0.md (D-13) | `140cb06` | `docs/release-process.md`, `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md`, `RELEASE-NOTES-v1.0.0.md` |
| 4    | Maintainer runs `make smoke-live` + fills durations | _pending checkpoint_ | `06-VERIFICATION.md` (durations, runner IDs, cost), `RELEASE-NOTES-v1.0.0.md` (wall-clock numbers) |

## Files Created

- `Makefile` — Solo-developer + Claude-execution targets (`help`, `test`, `test-race`, `vet`, `release-snapshot`) plus the maintainer-only live smokes (`smoke-live`, `smoke-live-byo`, `smoke-live-cloud`, `smoke-stopwatch`). Each smoke target gates on env-var preconditions; `smoke-live-cloud` allocates an isolated `mktemp -d` state dir (`RUNNERKIT_SMOKE_STATE_DIR`) shared with the precheck/end-to-end/verify scripts and removes it on success.
- `scripts/smoke/byo-permission.sh` — Phase 1 outstanding live GitHub permission smoke. Isolates state via `RUNNERKIT_STATE_DIR` tempdir, runs `up → status → doctor → down`, prints `BYO_DURATION_SECONDS=NNN` for the maintainer.
- `scripts/smoke/cloud-end-to-end.sh` — Phase 4 outstanding live Hetzner billable smoke. `trap cleanup EXIT INT TERM` runs `runnerkit destroy --yes` if state.json exists; snapshots state.json to state-after-destroy.json before destroy executes; disarms EXIT trap on success but rearms an INT/TERM purge for the verify phase.
- `scripts/smoke/hetzner-empty-precheck.sh` — D-12 gate 1 wrapper. Invokes `cmd/_smokebin/empty_precheck` via `go run`.
- `scripts/smoke/hetzner-destroy-verify.sh` — D-12 gate 2 wrapper. Reads `RUNNERKIT_SMOKE_STATE_DIR/state.json` (or `state-after-destroy.json` fallback) and invokes `cmd/_smokebin/destroy_verify` via `go run` with `RUNNERKIT_SMOKE_TIMEOUT` and `RUNNERKIT_SMOKE_STATE_FILE` set.
- `cmd/_smokebin/empty_precheck/main.go` — D-12 gate 1 binary. `hcloudClient` interface (`AllServers`, `AllSSHKeys`, `AllPrimaryIPs`, `AllFirewalls`) lets tests inject a fake; `realClient` adapter wraps `*hcloud.Client`. Refuses if any resource Name starts with `runnerkit-`; error message names every offender.
- `cmd/_smokebin/empty_precheck/main_test.go` — `TestEmptyPrecheck_RefusesOnExisting` (3 subtests: server present, only-unrelated, fully-empty), `TestEmptyPrecheck_AllResourceTypes` (3 subtests: ssh-key, primary-ip, firewall).
- `cmd/_smokebin/destroy_verify/main.go` — D-12 gate 2 binary. `verifierClient` interface (`GetServerByID`, `GetSSHKeyByID`, `GetPrimaryIPByID`, `GetFirewallByID`) lets tests inject a fake; `realVerifier` adapter wraps `*hcloud.Client`. `extractCloudIDs` partial-unmarshals state.json to recover ServerID/SSHKeyID/PrimaryIPv4ID/PrimaryIPv6ID/FirewallID; `pollUntilGone` retries every 500ms until either `hcloud.IsError(err, hcloud.ErrorCodeNotFound)` for every saved ID or the deadline elapses.
- `cmd/_smokebin/destroy_verify/main_test.go` — `TestDestroyVerify_Timeout` (2 subtests: 404-after-N-polls success, never-disappears timeout failure with 2s `RUNNERKIT_SMOKE_TIMEOUT`). Fixture writes a minimal state.json under `t.TempDir()` matching the partial-unmarshal shape.
- `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md` — v1.0.0 baseline skeleton; maintainer fills the numbers during the human-action checkpoint.
- `RELEASE-NOTES-v1.0.0.md` — first release notes template at repo root; references `accidentally-awesome-labs/...` consistently per Plan 06-01 Task 5 org migration.

## Files Modified

- `.gitignore` — Appended smoke artifact patterns (`.smoke-state/`, `*.smoke.log`, `cmd/_smokebin/empty_precheck/empty_precheck`, `cmd/_smokebin/destroy_verify/destroy_verify`) under the existing `dist/` line written by Plan 06-01. The `_smokebin` `_` prefix already excludes the directories from `go build ./...` so the `cmd/_smokebin/*/<name>` lines are belt-and-suspenders against accidental `go build` from inside the directory.
- `docs/release-process.md` — Appended "Stopwatch Checklist (D-13)" section after the existing Plan 06-01 sections (`One-Time Prerequisites`, `Tag a Release`, `Common Failures`, `Release Notes File (D-13)`). The new section adds BYO + Hetzner stopwatch tables and recording instructions; nothing existing was modified or removed.

## Decisions Made

See `key-decisions` in the frontmatter. The notable ones:

- **Polling cadence is 500ms (not 5s).** Required so the timeout-failure unit test can converge against a 2s `RUNNERKIT_SMOKE_TIMEOUT` while still being polite to the real Hetzner API in production.
- **hcloud-go IDs are `int` (not `int64`).** Matches the existing `internal/provider/hetzner/client.go` adapter and the upstream module's actual API surface; removes a sea of conversions.
- **Trap is rearmed for INT/TERM after EXIT disarm.** Two destroy invocations on success would burn a Hetzner round-trip and could race the verifier; one INT/TERM trap that purges the tempdir is sufficient for the verify-phase abort case.
- **All references use `accidentally-awesome-labs/...`.** Per the org migration in Plan 06-01 commit `c359831`. No `salar/...` references introduced.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] hcloud-go v1.59.2 ID type mismatch in plan example code**

- **Found during:** Task 2 GREEN.
- **Issue:** The plan's `<action>` step example for `cmd/_smokebin/destroy_verify/main.go` declared `cloudIDs` with `int64` fields and passed `int(id)` to `r.c.Server.GetByID`. hcloud-go v1.59.2 actually uses plain `int` for resource IDs throughout (verified against `$GOMODCACHE/github.com/hetznercloud/hcloud-go@v1.59.2/hcloud/server.go:20` — `ID int` — and `internal/provider/hetzner/client.go::GetServer(ctx, id int)`). Writing the literal plan code would compile but every call site would need a manual `int(...)` cast, and the type drift would risk subtle bugs.
- **Fix:** Switched `cloudIDs` fields to plain `int`, updated `verifierClient` method signatures to `(ctx, id int)`, and dropped the casts. Matches the existing adapter pattern.
- **Files modified:** `cmd/_smokebin/destroy_verify/main.go`, `cmd/_smokebin/destroy_verify/main_test.go`.
- **Verification:** `go vet ./cmd/_smokebin/empty_precheck ./cmd/_smokebin/destroy_verify` clean; both test files green.
- **Committed in:** `8f59a40` (Task 2 GREEN).

**2. [Rule 1 — Bug] 5-second poll interval would hang the timeout-failure unit test**

- **Found during:** Task 2 RED-to-GREEN transition (test design pass).
- **Issue:** `TestDestroyVerify_Timeout`'s second subtest sets `RUNNERKIT_SMOKE_TIMEOUT=2`. With a 5-second poll interval the loop would sleep 5s after the first failed check (because `time.Now().After(deadline)` is checked before the sleep, but the sleep dominates total runtime). The test would either flake or take 7+s — well outside the per-test budget.
- **Fix:** Lowered `pollInterval` from 5s to 500ms. In production (300s default timeout) this gives 600 polls, which is fine — each poll is at most 5 GET calls against the Hetzner API and the resource transition latency is typically a few seconds, so the loop converges quickly. The 500ms cadence does not change the worst-case wall-clock for a real smoke.
- **Files modified:** `cmd/_smokebin/destroy_verify/main.go`.
- **Verification:** Both subtests green; `TestDestroyVerify_Timeout` runs in ~3.8s total (the 2s timeout bound dominates the failure case).
- **Committed in:** `8f59a40` (Task 2 GREEN).

**3. [Rule 3 — Blocking] Plan body's `int64` interface type would block destroy_verify test compilation**

- **Found during:** Task 2 RED.
- **Issue:** When writing the RED stub I needed to declare the `verifierClient` interface in `main.go` so the test could compile against it. The plan's body declared `GetServerByID(ctx context.Context, id int64) (*hcloud.Server, error)` but the test fakes use the same interface — declaring `int64` here and `int` in the GREEN realVerifier adapter would have made the GREEN commit a refactor of the interface, not just the implementation.
- **Fix:** Same as deviation 1 — declared the interface with `int` from the start of the RED commit. The GREEN commit could then be a pure implementation swap of the `run()` body without touching the interface.
- **Committed in:** `956cb2c` (RED — interface set up correctly from the start) and `8f59a40` (GREEN — implementation only).

### Non-deviations (pre-existing untracked files)

- `.pi-lens/` and `.pi/` directories are pre-existing untracked items at plan start (Vercel plugin local cache, also noted by Plan 06-01). Left untouched per scope-boundary rule (out-of-scope; not caused by this plan's changes).

## Authentication Gates

Task 4 is a `checkpoint:human-action` for **two** real-world authentication gates Claude cannot pass:

1. **`gh auth login` against a real GitHub account.** The Phase 1 outstanding live GitHub permission smoke (`make smoke-live-byo`) requires real GitHub Actions API access against a real maintainer-controlled trusted repo. No fake adapter exists for this surface; the `internal/github` package's fake is unit-test-only.
2. **`HCLOUD_TOKEN` against a real Hetzner project that creates billable resources.** The Phase 4 outstanding live Hetzner billable smoke (`make smoke-live-cloud`) creates real Server / SSHKey / PrimaryIP / Firewall resources. The cost is real EUR per provisioning, the destroy-verify gate proves cleanup via real GETs against the Hetzner API, and the empty-project precheck refuses to run if any prior smoke leaked.

Both are documented in `<how-to-verify>` of the Task 4 plan body. Maintainer resume signal is `smoke-green` (all gates passed) or `smoke-red <reason>` (gap-closure plan needed).

## Forward References Created

- **`Makefile smoke-stopwatch` → `docs/release-process.md "Stopwatch Checklist"` + `RELEASE-NOTES-v$VER.md` + `06-VERIFICATION.md`** — the stopwatch target is a reminder, not a runner; the maintainer follows the checklist on a clean machine.
- **`scripts/smoke/cloud-end-to-end.sh` → `cmd/_smokebin/empty_precheck` (precondition)** — Makefile orchestrates the order; the wrapper script does not call empty_precheck itself.
- **`scripts/smoke/cloud-end-to-end.sh` → `cmd/_smokebin/destroy_verify` (postcondition)** — same; Makefile orchestrates.
- **`RELEASE-NOTES-v1.0.0.md` → `docs/upgrade.md` + `docs/troubleshooting/README.md`** — both target files exist (Plans 06-02 and 06-03).

## Cross-plan notes

- **`docs/release-process.md` is now complete for v1.0.0.** Plan 06-01 wrote the maintainer procedure; Plan 06-04 added the Stopwatch Checklist. Future plans should treat the file as feature-complete unless a release process changes.
- **`06-VERIFICATION.md` is the v1.0.0 baseline.** The maintainer fills it during the Task 4 checkpoint; subsequent releases re-run the same checklist into per-release `RELEASE-NOTES-vX.Y.Z.md` files but do NOT overwrite the baseline.
- **No Phase 5 ephemeral regressions.** `go test ./... -count=1` clean across all 17 packages including `internal/cli/up_ephemeral_test.go` and the ephemeral-aware `down`/`destroy` paths. `internal/redact/` flow untouched.
- **`_smokebin` is the new convention for one-shot smoke binaries.** Future plans that need similar manual-only binaries (e.g., a future GoReleaser dry-run helper) can land at `cmd/_smokebin/<name>/` and inherit the `_` exclusion + `go run` invocation pattern automatically.

## Validation matrix coverage (06-VALIDATION.md)

- **Line 74 — Live BYO smoke:** scaffold by Tasks 1 (Makefile, byo-permission.sh) + 3 (06-VERIFICATION skeleton, RELEASE-NOTES template); closes on maintainer `smoke-green` resume signal.
- **Line 75 — Live Hetzner end-to-end:** scaffold by Tasks 1 (Makefile, cloud-end-to-end.sh + hetzner-empty-precheck.sh + hetzner-destroy-verify.sh) + 2 (cmd/_smokebin/{empty_precheck, destroy_verify}); closes on maintainer `smoke-green`.
- **Line 76 — D-12 gate 1 unit + live:** unit GREEN from Task 2 (`TestEmptyPrecheck_RefusesOnExisting`, `TestEmptyPrecheck_AllResourceTypes`); live closes on maintainer.
- **Line 77 — D-12 gate 2 unit + live:** unit GREEN from Task 2 (`TestDestroyVerify_Timeout`); live closes on maintainer.
- **Line 78 — 10-min stopwatch checklist:** scaffold by Task 3 (`docs/release-process.md` Stopwatch Checklist + `06-VERIFICATION.md` BYO/Hetzner total rows + `RELEASE-NOTES-v1.0.0.md` 10-Minute Stopwatch placeholder table); closes on maintainer.

## Pending Maintainer Checkpoint (Task 4)

**Type:** `checkpoint:human-action` (blocking)
**Resume signal:** `smoke-green` (all gates passed) or `smoke-red <reason>` (gap-closure plan needed)

**Files awaiting maintainer fill-in:**

- `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md` — durations, runner ID, 5 resource IDs, cost EUR, gate-1 PASS, gate-2 PASS, stopwatch totals, sign-off.
- `RELEASE-NOTES-v1.0.0.md` — wall-clock durations + Hetzner cost in the 10-Minute Stopwatch table.

**Steps for maintainer (estimated 30-45 minutes total):**

1. Resolve Plan 06-01 prerequisites (already 'tap-ready' as of 2026-05-02).
2. Export `RUNNERKIT_SMOKE_BYO_HOST=user@host`, `RUNNERKIT_SMOKE_REPO=accidentally-awesome-labs/runnerkit-smoke-test` (or any maintainer-controlled trusted repo), `HCLOUD_TOKEN=<from Hetzner project>`; run `gh auth login` if not already.
3. Verify Hetzner project is empty of `runnerkit-*` resources (Console eyeball; the precheck will also refuse if any exist).
4. `time make smoke-live 2>&1 | tee smoke-output.log`. Watch for `BYO_DURATION_SECONDS=NNN`, empty-precheck OK, `CLOUD_DURATION_SECONDS=NNN`, destroy-verify OK.
5. Run the 10-minute stopwatch on a CLEAN machine (fresh laptop or VM). Follow `docs/release-process.md` Stopwatch Checklist.
6. Fill `06-VERIFICATION.md`: tick all 8 automated test items, fill BYO + Hetzner smoke fields, fill stopwatch totals, sign and date.
7. Fill `RELEASE-NOTES-v1.0.0.md` 10-Minute Stopwatch table.
8. Verify Hetzner project is empty AGAIN (post-smoke; belt-and-suspenders).
9. Commit both files. Resume signal `smoke-green` triggers `/gsd:verify-work` for Phase 6 sign-off; then push `git tag -a v1.0.0` per `docs/release-process.md`.

## Live Smoke Attempt 1 (2026-05-04 to 2026-05-05) — BLOCKED

The orchestrator drove Task 4 against `accidentally-awesome-labs/dat0`
(private repo) with the maintainer's `salar@mckee-small-desktop` BYO
host and a real Hetzner project token. The smoke surfaced two real
v1.0.0 BYO blockers documented in
`06-GAP-byo-sudo-handling.md`:

- **Bug 1 (preflight):** `internal/preflight/checks.go::CheckPrivilege`
  passes when `sudo` binary exists, even when sudo prompts for a
  password. Bootstrap then fails opaquely with `bootstrap_failed`
  while the actual remote stderr is swallowed by the executor. Worked
  around for this attempt by adding a temporary
  `/etc/sudoers.d/runnerkit-smoke-temp NOPASSWD ALL` entry on the BYO
  host.
- **Bug 2 (download_runner permission):** even with NOPASSWD sudo, the
  `download_runner` step in `internal/bootstrap/install.go::Apply`
  (and `RenderInstallScript` / `RenderEphemeralInstallScript` in
  `script.go`) creates the install dir owned by `runnerkit-runner`
  (mode 0755), then runs plain `curl`/`sha256sum`/`tar` as the SSH
  user — which has no write access to a directory owned by another
  user. `curl: (23) Failure writing output to destination, Permission
  denied`. The bug went undetected from Plan 02-02 onward because every
  bootstrap test uses fakeExecutor that records commands but never
  executes them. **BYO is non-functional in v1 without a fix.**

Smoke attempt was paused before any `runnerkit destroy` ran on the BYO
host, so no host state was changed beyond the `runnerkit-runner` user
account (manually creatable by the bootstrap and reusable on the next
attempt) and the temporary sudoers entry (to be removed when smoke
resumes). No Hetzner resources were created — the smoke failed at the
BYO step before reaching cloud-end-to-end. Hetzner project state is
unchanged (zero `runnerkit-*` resources confirmed).

Resolution path: the user accepted the recommendation to pause Plan
06-04, run `/gsd:verify-work 6` to surface gaps, then `/gsd:plan-phase
06 --gaps` to plan a closure (Tasks A-E in the gap doc), then
`/gsd:execute-phase 06 --gaps-only` to implement, then re-run Plan
06-04 Task 4 against fixed code.

## Self-Check: PARTIAL — Tasks 1-3 PASSED, Task 4 BLOCKED ON GAP

- All 11 created files exist on disk.
- All 4 task commits exist in git log: `fa2d5b8`, `956cb2c`, `8f59a40`, `140cb06`.
- Tasks 1-3 acceptance criteria all green (Makefile syntax, scripts executable + `bash -n` clean, no `smoke-live` in `.github/workflows/*`, all `set -euo pipefail`, all 6 unit subtests + matrix entries pass, `go test ./...` 17/17 packages clean, no `salar/` references in any new file, all `accidentally-awesome-labs/` references correct).
- `go build ./...`, `go vet ./cmd/_smokebin/empty_precheck ./cmd/_smokebin/destroy_verify`, `go test ./... -count=1` all green.
- Task 4 is a `checkpoint:human-action`; SUMMARY records pending state and maintainer resume signal.
- No emojis in any deliverable file.
