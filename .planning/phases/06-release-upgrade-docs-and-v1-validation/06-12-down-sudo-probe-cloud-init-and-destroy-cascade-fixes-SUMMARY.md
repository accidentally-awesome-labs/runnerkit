---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 12
subsystem: infra
tags: [hetzner, cloud-init, sudo, ssh, runnerkit, destroy, primary-ip, auto-delete]

# Dependency graph
requires:
  - phase: 06-release-upgrade-docs-and-v1-validation
    provides: "Plan 06-11 host-key determinism (Bug 24), down sudo probe gate (Bug 25), auto_delete cascade in destroy (Bug 26), scoped sudoers svc.sh glob (Bug 27)"
provides:
  - "Bug 28 closure: probeSudoNeedsPassword inspects ExitCode regardless of *exec.ExitError wrapper from real SSH executor"
  - "Bug 29 closure: cloud.cloudinit.wait command carries explicit Timeout (default 5m, RUNNERKIT_CLOUD_INIT_TIMEOUT override)"
  - "Bug 30 closure: destroy skips DeletePrimaryIP on AutoDelete=true and retries 409 must_be_unassigned for legacy state"
  - "state.CloudInventory.PrimaryIPv4AutoDelete + PrimaryIPv6AutoDelete fields (additive, omitempty, no schema bump)"
  - "Provider.Sleep injection point + WithSleep Option for unit tests"
affects: ["06-07-live-smoke-rerun", "v1.0.0 tag push", "06-VERIFICATION baseline"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Cascade-aware delete: skip when state flag indicates the cloud will handle teardown, retry on cascade-in-flight signal for legacy state"
    - "Env-var-driven Duration via os.Getenv + time.ParseDuration with non-positive fallback to defaults"
    - "Additive optional state fields with json omitempty preserve byte-stable serialization for pre-existing fixtures"

key-files:
  created: []
  modified:
    - "internal/cli/down.go"
    - "internal/cli/down_test.go"
    - "internal/cli/up.go"
    - "internal/cli/up_cloud_test.go"
    - "internal/provider/hetzner/destroy.go"
    - "internal/provider/hetzner/destroy_test.go"
    - "internal/provider/hetzner/provision.go"
    - "internal/state/schema.go"

key-decisions:
  - "Bug 28: dropped the early `if err != nil { return false, nil }` guard in probeSudoNeedsPassword and routed all decision paths through result.ExitCode + stderr/stdout markers; non-exit-status errors fall through the same default-fallback branch (preserves graceful semantics for dial timeout / context cancel)."
  - "Bug 29: used os.Getenv (matching Plan 06-02 + update package convention) rather than threading a deps.Env map through Dependencies; keeps the surface change small and aligns with the existing RUNNERKIT_NO_UPDATE_NOTIFIER pattern."
  - "Bug 30 dual-path: AutoDelete=true (post-Plan-06-12 default) skips DeletePrimaryIP entirely; legacy AutoDelete=false retries on 409 must_be_unassigned via new isCascadeInFlightError predicate until 404 (cascade complete via isAlreadyAbsentError) or RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT expires (default 30s)."
  - "Bug 30 schema evolution: AutoDelete fields added with `json omitempty` so SchemaVersion stays at \"2\" (pre-Plan-06-12 state files load with both flags=false → legacy retry path handles cleanup correctly without migration)."
  - "Bug 30 provision write: PrimaryIPvNAutoDelete is set to (resourceIDs[primary_ipvN] != \"\") so the flag tracks whether Hetzner actually allocated the IP for us — matches the EnableIPv4/EnableIPv6 PublicNet contract Plan 06-11 Bug 26 locked in."

patterns-established:
  - "Pattern: real SSH executor returns *exec.ExitError for any non-zero remote rc; tests must populate testsupport.RemoteExecutor.Errors[id] alongside Results[id] to mirror that contract — guarding against future regressions where pure Results-only fakes mask real-world err handling"
  - "Pattern: env-driven Duration helpers fall back to a sensible default on empty/unparseable/non-positive — never accept zero (which silently disables the deadline)"
  - "Pattern: cascade-aware delete shapes — primary path (state flag → skip), legacy fallback (bounded retry on cascade-in-flight predicate)"

requirements-completed: [REL-05]

# Metrics
duration: 26min
completed: 2026-05-07
---

# Phase 06 Plan 12: Down Sudo Probe + Cloud-Init Timeout + Destroy Cascade Reporting Fixes Summary

**Three regression-blocker bug fixes (Bugs 28-30) from Plan 06-07 attempt-17 SMOKE-RED: probe inspects ExitCode regardless of *exec.ExitError wrapper, cloud-init wait carries an explicit RUNNERKIT_CLOUD_INIT_TIMEOUT-backed deadline (default 5m), and destroy is cascade-aware (AutoDelete=true skips, legacy state retries on 409 must_be_unassigned).**

## Performance

- **Duration:** ~26 min
- **Started:** 2026-05-07T18:47:00Z
- **Completed:** 2026-05-07T19:13:14Z
- **Tasks:** 3 (each TDD: RED test commit + GREEN fix commit)
- **Files modified:** 8 production+test files

## Accomplishments

- **Bug 28 closed:** `runnerkit down --yes` against a BYO host with password-protected sudo now correctly prompts for the password and threads it through `runner_files` cleanup. The previous early `if err != nil { return false, nil }` guard at down.go:440-443 swallowed the *exec.ExitError wrapper the real SSH executor returns for any non-zero remote rc — that wrapper is the EXPECTED case for password-protected sudo (rc=1 + stderr marker). New flow: ExitCode==0 → passwordless; ExitCode!=0 with marker substring → needs password (regardless of err); fall through otherwise. Plan 06-07 attempt-17 trace `probe-direct: rc=1 err=exit status 1` no longer produces `probe: needs=false`.
- **Bug 29 closed:** `runnerkit up --cloud hetzner` cloud-init wait step now sets an explicit Timeout (5m default, aligned with `hetzner.HostKeyProbeOptions` 60×5s), with `RUNNERKIT_CLOUD_INIT_TIMEOUT` override for slower regions/images. The previous no-Timeout command aborted at 42s on Hetzner cpx22 + ubuntu-24.04 (typical cloud-init 60-120s), surfacing as `cloud_readiness_failed`.
- **Bug 30 closed:** `runnerkit destroy --yes --cloud hetzner` is now cascade-aware. AutoDelete=true state (post-Plan-06-12 default for IPs auto-allocated via Plan 06-11 Bug 26's `EnableIPv4: true, EnableIPv6: true` PublicNet block) SKIPS the explicit DeletePrimaryIP call entirely (Status=skipped, Message="auto_delete cascade"). Legacy state (AutoDelete=false) wraps DeletePrimaryIP in a bounded retry loop that treats 409 `must_be_unassigned` as a transient cascade-in-flight signal until 404 (cascade complete via `isAlreadyAbsentError`) or `RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT` expires (default 30s). New `isCascadeInFlightError` predicate covers 409+`must_be_unassigned` (StatusCode-aware) plus a substring fallback for test fakes.
- **Plan 06-11 contracts preserved:** `TestDestroy_AutoDeleteCascadeNoUnassign` (Bug 26: no `unassign:*` calls anywhere), `TestDestroyTreatsAlreadyAbsentDetachAsSuccess`, `TestDown_SudoProbeRunsEvenWhenSSHReachableFalse` (Bug 25), `TestDownThreadsSudoPasswordWhenSudoRequiresPasswordClosesBug21` (Bug 21), and `TestDownDoesNotPromptWhenSudoIsPasswordless` all still pass.

## Task Commits

Each task was committed atomically (TDD: RED then GREEN):

1. **Task 1 RED: Bug 28 failing test** - `fe71fe1` (test)
2. **Task 1 GREEN: Bug 28 production fix** - `8230a38` (fix)
3. **Task 2 RED: Bug 29 failing test** - `c4fbba8` (test)
4. **Task 2 GREEN: Bug 29 production fix** - `0e5d42c` (fix)
5. **Task 3 RED: Bug 30 failing tests (5)** - `ce016bf` (test)
6. **Task 3 GREEN: Bug 30 production fix** - `4993214` (fix)

**Plan metadata commit:** Pending — bundled with STATE.md/ROADMAP.md updates after this SUMMARY.

_Note: Each Task is TDD'd (RED → GREEN); Tasks 1 & 2 follow plan TDD shape exactly; Task 3 has 5 tests in one RED commit because they share the same destroy.go fix surface._

## Files Created/Modified

- `internal/cli/down.go` — `probeSudoNeedsPassword` rewritten to inspect `result.ExitCode` + `result.Stderr/Stdout` regardless of err (Bug 28).
- `internal/cli/down_test.go` — `TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper` locks in the real-SSH-executor contract (Errors[id] populated alongside Results[id]).
- `internal/cli/up.go` — `cloudInitTimeoutFromEnv()` helper + `defaultCloudInitTimeout = 5*time.Minute`; `waitCloudTargetReady`'s `cloud.cloudinit.wait` Run call sets `Timeout: cloudInitTimeoutFromEnv()` (Bug 29).
- `internal/cli/up_cloud_test.go` — `TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget` table-tests 5 sub-cases: default, override, invalid duration, empty string, zero duration (last three fall back to default).
- `internal/provider/hetzner/destroy.go` — new `isCascadeInFlightError` predicate, new `defaultDestroyPrimaryIPTimeout = 30s` + `destroyPrimaryIPTimeoutFromEnv()` helper, new `makePrimaryIPDeleter` closure replacing the two raw `apply()` calls for primary IPs (cascade-skip + legacy retry).
- `internal/provider/hetzner/destroy_test.go` — 5 new tests: `TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade`, `TestDestroy_RetriesPrimaryIPDeleteOn409MustBeUnassigned`, `TestDestroy_RetryExhaustsBudgetThenSurfacesAsPartial`, `TestIsCascadeInFlightError`, `TestDestroy_LegacyAutoDeleteFalseHits404FromCascade`. New `destroyFakeRetryClient` (extends `destroyFakeOrderedClient` with per-call `deleteIPErrs` slice + `defaultErr` fallback) and new `hcloudStubError` test fake implementing `StatusCode() int`.
- `internal/provider/hetzner/provision.go` — added `Sleep func(time.Duration)` field on `Provider` + `WithSleep(sleep)` `Option`; `machineFromServer` records `PrimaryIPv4AutoDelete`/`PrimaryIPv6AutoDelete = (resourceIDs[primary_ipvN] != "")` so post-Plan-06-12 state captures the Hetzner-side AutoDelete=true default.
- `internal/state/schema.go` — `CloudInventory.PrimaryIPv4AutoDelete` + `PrimaryIPv6AutoDelete` bool fields with `json:"primary_ipv*_auto_delete,omitempty"`. Additive optional fields with omitempty preserve byte-stable serialization (existing `TestCloudInventorySerializesProviderIdentityAndNoSecrets` JSON-shape assertions continue to match without modification — false zero-value is omitted).

## Decisions Made

See key-decisions in frontmatter. Highlights:

1. **Bug 28 fix shape:** the simplest correct fix — drop the early err-guard and route all decision paths through `result.ExitCode + result.Stderr/Stdout`. The plan suggested a more elaborate "only treat err as fatal when err is NOT an exit-status wrapper" check; the actual implementation is even simpler because the existing default-fallback branch (`return false, nil`) at the end already handles the executor-startup-failure case (ExitCode = -1, no marker substring → fall through). No new error-type assertion needed.
2. **Bug 29 env access pattern:** `os.Getenv` directly (matching `internal/update/check.go` `RUNNERKIT_NO_UPDATE_NOTIFIER` precedent) rather than threading a `deps.Env map[string]string` through `Dependencies`. The plan body suggested `deps.Env` but `Dependencies` only exposes `GitHubEnv`; adding a generic `Env` field would be a larger surface change for a localized fix. Tests use `t.Setenv` to control the env var (also matching `internal/update/check_test.go`).
3. **Bug 30 sleep/timeout interplay:** the budget-exhaustion test injects a real 2ms sleep via `WithSleep` so the 1ms `RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT` reliably crosses the deadline after a couple iterations. A pure no-op sleep with a 1ms deadline can in theory race the clock check; the 2ms real sleep is small enough that the test runs well under 100ms and removes the flakiness window.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `deps.Env` field does not exist; used `os.Getenv` to match codebase precedent**
- **Found during:** Task 2 (Bug 29 cloud-init timeout)
- **Issue:** Plan referenced `deps.Env["RUNNERKIT_CLOUD_INIT_TIMEOUT"]` but `cli.Dependencies` does not have a generic `Env` map — it has `GitHubEnv` only. Adding a new field would touch every call site that constructs Dependencies.
- **Fix:** Used `os.Getenv("RUNNERKIT_CLOUD_INIT_TIMEOUT")` directly inside `cloudInitTimeoutFromEnv()`. Tests use `t.Setenv` for control. This matches the existing `internal/update/check.go` `RUNNERKIT_NO_UPDATE_NOTIFIER` precedent and keeps the surface change small.
- **Files modified:** internal/cli/up.go, internal/cli/up_cloud_test.go (no Dependencies surface change)
- **Verification:** `TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget` 5 sub-cases all pass; full cli package green.
- **Committed in:** `0e5d42c` (Task 2 GREEN)

**2. [Rule 3 - Blocking] Test stub had finite error sequence; switched to defaultErr fallback for budget-exhaustion test**
- **Found during:** Task 3 (Bug 30 destroy retry budget)
- **Issue:** Initial `destroyFakeRetryClient` used a fixed `[]error{stub, stub, ...}` slice; running off the end returned nil → `delete()` returned no error → IP marked as Status=done before the 1ms timeout could fire. Test failed (Partial=false instead of Partial=true).
- **Fix:** Added `defaultErr error` field on `destroyFakeRetryClient`. When `deleteIPCallCount >= len(deleteIPErrs)`, returns `defaultErr` (nil if unset). Budget-exhaustion test uses `client := &destroyFakeRetryClient{defaultErr: stub}` for permanent 409 + a real 2ms sleep injection so the deadline reliably crosses on iteration ~2.
- **Files modified:** internal/provider/hetzner/destroy_test.go (test infrastructure only)
- **Verification:** `TestDestroy_RetryExhaustsBudgetThenSurfacesAsPartial` passes (Partial=true, provider_primary_ip_pending recorded); test runs in <100ms.
- **Committed in:** `4993214` (Task 3 GREEN — fix + test infrastructure both ready together).

**3. [Rule 3 - Blocking] gofmt reformatted `CloudInventory` struct alignment after AutoDelete additions**
- **Found during:** Task 3 verification (gofmt -l after AutoDelete fields landed)
- **Issue:** Adding two bool fields between `PrimaryIPv6ID` and `SSHKeyID` changed the column alignment of subsequent string fields; gofmt detected the reformat needed.
- **Fix:** Ran `gofmt -w` on the four affected files. The reordering keeps related fields contiguous (PrimaryIPv4ID/PrimaryIPv6ID/PrimaryIPv4AutoDelete/PrimaryIPv6AutoDelete then SSHKeyID/SSHKeyName/...). JSON tags + omitempty preserved; existing serialization tests still pass byte-for-byte.
- **Files modified:** internal/provider/hetzner/destroy_test.go, internal/provider/hetzner/provision.go, internal/state/schema.go (formatting only)
- **Verification:** `gofmt -l <files>` returns empty; full `go test ./... -count=1 -race` green.
- **Committed in:** `4993214` (Task 3 GREEN — bundled with the production change).

---

**Total deviations:** 3 auto-fixed (3× Rule 3 blocking issues; all small surface alignments to actual codebase shape vs plan's described shape).
**Impact on plan:** All deviations are subordinate to the plan's stated goal — Bugs 28/29/30 are closed at the contract level the plan specified. No scope creep, no architectural change.

## Issues Encountered

- **Embedded-method shadowing trap (Task 3):** `destroyFakeRetryClient` embeds `destroyFakeOrderedClient` and overrides `DeletePrimaryIP`. The override works correctly (verified via standalone test) but the FIRST budget-exhaustion test draft over-relied on the finite `deleteIPErrs` slice and accidentally returned nil after 10 calls. Resolved via the `defaultErr` field (see Deviation 2). Lesson: when a fake is meant to model "permanent failure", make the failure path the default and the success path explicit.
- **Sleep-vs-deadline race (Task 3):** With `WithSleep(func(time.Duration) {})` (no-op) and `RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT=10ms`, the retry loop iterates faster than the deadline check; on a fast machine the loop can run thousands of iterations before `time.Now().After(deadline)` fires. Resolved by injecting a 2ms real sleep (still well under 100ms total test wall-clock). The production code calls `time.Sleep(1*time.Second)` between attempts which gives plenty of headroom against any reasonable deadline.

## Pre-Smoke Maintainer Action Checklist

Before re-running `make smoke-live` for Plan 06-07 attempt-18:

1. **Recover the BYO host install dir.** Plan 06-07 attempt-17 left a stale install dir at `/opt/actions-runner/runnerkit-*` on `salar@mckee-small-desktop` because Bug 28 blocked cleanup. Once this plan lands (it has — `8230a38` is on main), run:

   ```bash
   go run ./cmd/runnerkit down --repo $RUNNERKIT_SMOKE_REPO --yes
   ```

   Type the sudo password when prompted (the prompter now fires correctly). Verify the install dir is gone:

   ```bash
   ssh $RUNNERKIT_SMOKE_BYO_HOST 'ls /opt/actions-runner/ 2>/dev/null | head'
   ```

   If `runnerkit-*` directories still appear, manually rm them as a one-time recovery (`sudo rm -rf /opt/actions-runner/runnerkit-*`) — Plan 06-12 only ensures FUTURE runs cleanup correctly; it cannot retroactively clean an already-orphaned install.

2. **Re-verify Hetzner project empty** (D-12 gate 1 precondition):

   ```bash
   hcloud server list ; hcloud firewall list ; hcloud primary-ip list ; hcloud ssh-key list
   ```

3. **Re-run** `make smoke-live` per Plan 06-07 sequence. Expected outcomes:
    - BYO smoke: `runnerkit down` cleans all artifacts, exit 0, no `runner_files: failed sudo: a terminal is required`.
    - Cloud smoke: `runnerkit up --cloud hetzner` succeeds within typical 60-120s (300s budget), runner registers, `runnerkit destroy` exits 0 with `provider_primary_ip: status=skipped Message="auto_delete cascade"` (NEW — Bug 30 default), `cmd/_smokebin/destroy_verify` polls saved IDs to 404.

## User Setup Required

None — no external service configuration required. RUNNERKIT_CLOUD_INIT_TIMEOUT and RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT env vars are optional overrides; their defaults (5m and 30s respectively) match the production-realistic Hetzner behavior windows.

## Next Phase Readiness

- **Plan 06-07 attempt-18 unblocked.** All three regression-blocker bugs are closed at code AND test level. Plan 06-11 contracts (Bugs 24-27) remain intact and verified live.
- **`go test ./... -count=1 -race` is green.**
- **`go vet ./...` is clean.**
- **gofmt is clean.**
- **Self-Check below** confirms each modified file exists and each commit is on main.
- After the maintainer pre-smoke checklist completes, Plan 06-07 attempt-18 produces the SMOKE-GREEN signal that fills 06-VERIFICATION.md baseline (real durations + 5 cloud resource IDs + EUR cost + signature) and unblocks the v1.0.0 tag push per D-13.
- **Pointer to closure:** Plan 06-07 SUMMARY (created after attempt-18 smoke-green) holds the final closure for the Phase 06 milestone signal.

## Self-Check: PASSED

Verified after writing this SUMMARY:

- `internal/cli/down.go` — exists; contains `probeSudoNeedsPassword` and `Bug 28` markers.
- `internal/cli/down_test.go` — exists; contains `TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper`.
- `internal/cli/up.go` — exists; contains `cloudInitTimeoutFromEnv` and `RUNNERKIT_CLOUD_INIT_TIMEOUT`.
- `internal/cli/up_cloud_test.go` — exists; contains `TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget`.
- `internal/provider/hetzner/destroy.go` — exists; contains `isCascadeInFlightError` and `RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT`.
- `internal/provider/hetzner/destroy_test.go` — exists; contains `TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade`, `TestDestroy_LegacyAutoDeleteFalseHits404FromCascade`.
- `internal/provider/hetzner/provision.go` — exists; contains `PrimaryIPv4AutoDelete`, `WithSleep`.
- `internal/state/schema.go` — exists; contains `PrimaryIPv4AutoDelete` + `PrimaryIPv6AutoDelete`.
- All 6 task commits exist on main (`fe71fe1`, `8230a38`, `c4fbba8`, `0e5d42c`, `ce016bf`, `4993214`).

---
*Phase: 06-release-upgrade-docs-and-v1-validation*
*Completed: 2026-05-07*
