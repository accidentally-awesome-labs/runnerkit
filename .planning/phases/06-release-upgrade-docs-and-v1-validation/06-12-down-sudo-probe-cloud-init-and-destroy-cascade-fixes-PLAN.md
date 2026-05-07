---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 12
type: execute
wave: 1
depends_on: [11]
files_modified:
  - internal/cli/down.go
  - internal/cli/down_test.go
  - internal/cli/up.go
  - internal/cli/up_test.go
  - internal/provider/hetzner/destroy.go
  - internal/provider/hetzner/destroy_test.go
autonomous: true
gap_closure: true
requirements: [REL-05, DOC-04]
must_haves:
  truths:
    - "Bug 28: `runnerkit down --repo X --yes` against a BYO host with password-protected sudo correctly prompts for the password (TTY) and threads it through `runner_files` cleanup. The sudo probe inspects `result.ExitCode` + `result.Stderr` to detect password-required, and does NOT short-circuit on a generic `err = exit status N` returned by the real SSH executor for any non-zero remote rc. Plan 06-07 attempt-17 trace (smoke-output.log lines 36-58) shows `[BUG25-TRACE] probe-direct: rc=1 err=exit status 1` followed by `probe: needs=false probeErr=<nil>` because the early `if err != nil { return false, nil }` guard at down.go:440-443 swallows the err before the marker check runs."
    - "Bug 29: `runnerkit up --repo X --cloud hetzner` waits long enough for Hetzner cloud-init on `cpx22`/`ubuntu-24.04` to finish before declaring `cloud_readiness_failed`. The cloud-init `--wait` step at up.go:908 currently uses the default executor timeout (no explicit Timeout field on the remote.Command), and the live attempt-17 smoke aborted at 42s with `Cloud machine is not ready for runner registration yet`. The fix gives this step an explicit deadline aligned with `hetzner.HostKeyProbeOptions` (default 60×5s = 300s) and exposes a `RUNNERKIT_CLOUD_INIT_TIMEOUT` override so slower images / regions can extend the budget without code changes."
    - "Bug 30: `runnerkit destroy --repo X --yes --cloud hetzner` exits 0 with `provider_primary_ip: done` (NOT `pending Primary IP must be unassigned`) when the auto_delete cascade is in flight. Hetzner returns HTTP 409 `must_be_unassigned` (NOT 404) on `DeletePrimaryIP` while the server-delete cascade is still removing the IP; `isAlreadyAbsentError` only matches 404 and surfaces 409 as a hard failure with `RKD-PROV-006`. The fix skips the explicit `DeletePrimaryIP` calls entirely when state shows the IPs were auto-allocated by Hetzner (the new default per Plan 06-11 Bug 26 — `EnableIPv4: true, EnableIPv6: true` carries `AutoDelete: true`), and treats 409 `must_be_unassigned` as a transient cascade-in-flight signal that retries via the existing 404-tolerant path until the cascade completes (bounded by `RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT`, default 30s) for legacy state where AutoDelete might not be set."
    - "Plan 06-07 attempt-18+ live smoke against fresh BYO + real Hetzner project completes both BYO and cloud paths end-to-end with: `runnerkit down` cleaning all artifacts (no `runner_files: failed sudo: a terminal is required`), `runnerkit up --cloud hetzner` succeeding within 120s typical / 300s budget for cloud-init, `runnerkit destroy --yes` exiting 0 with `provider_primary_ip: status=done`, and `cmd/_smokebin/destroy_verify` polling each saved cloud ID to 404 within `RUNNERKIT_SMOKE_TIMEOUT`. The Hetzner project ends empty; `06-VERIFICATION.md` is fillable + signable; the v1.0.0 tag is pushable per D-13."
    - "Plan 06-11 fixes (Bugs 24, 26-cascade, 27 svc.sh glob) remain VERIFIED LIVE and are NOT regressed. Bug 24 (host-key match) was confirmed live: smoke-output.log line 26 shows `SSH OK reachable, host key matched`. Bug 26 cascade behavior was confirmed live: smoke-output.log lines 71-73 show `Hetzner project ended empty (verified post-test) — auto_delete cascade DID work to clean up primary IPs, despite destroy reporting pending. Plan 06-11 Bug 26's cascade approach works in practice` — the cascade is correct; only the synchronous reporting in `destroy.go:95-96` needs the 409 / AutoDelete-skip fix. Bug 27 (svc.sh glob) was implicitly confirmed: smoke-output.log line 28 shows `Service OK active` after a Path C-prepared run (Plan 06-11 SUMMARY pre-smoke action ran byo-prepare against the new glob)."
  artifacts:
    - path: "internal/cli/down.go"
      provides: "probeSudoNeedsPassword no longer early-returns when the executor returns `err = exit status N` for non-zero remote rc. The function inspects `result.ExitCode` + `result.Stderr + result.Stdout` regardless of err, and only treats err as fatal when err is NOT a remote-exit-status wrapper (e.g., context cancellation, dial failure, executor unable to start the command). Behavior on `result.ExitCode == 0`: `(false, nil)` — sudo passwordless. Behavior on non-zero `result.ExitCode` with marker substring in stderr: `(true, nil)` — prompt. Behavior on non-zero `result.ExitCode` without marker: `(false, nil)` — keep existing happy path. Behavior on err that is NOT an exit-status wrapper: `(false, nil)` — fall through (preserve existing graceful-failure semantics)."
      contains: "probeSudoNeedsPassword"
    - path: "internal/cli/down_test.go"
      provides: "TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper exercises the realistic real-SSH-executor case where `Run` returns `(remote.Result{ExitCode: 1, Stderr: \"sudo: a password is required\"}, errors.New(\"exit status 1\"))`. The test fails on the pre-fix code (probe early-returns false, prompt does not fire, runner_files cleanup fails). The test passes on the post-fix code (probe inspects stderr, returns true, prompt fires once, RUNNERKIT_SUDO_PASSWORD threads through commands)."
      contains: "TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper"
    - path: "internal/cli/up.go"
      provides: "waitCloudTargetReady's cloud-init readiness step (`cloud.cloudinit.wait`) sets an explicit Timeout aligned with the Hetzner host-key probe budget (default 300s, overridable via RUNNERKIT_CLOUD_INIT_TIMEOUT env var). The smoke harness can set a smaller value if needed; production cloud-up gets a 60-120s typical case with 300s headroom. Failure mode unchanged: if cloud-init genuinely never finishes, the step returns the same `cloud-init readiness failed` RemoteError as before — only the wall-clock budget extends."
      contains: "RUNNERKIT_CLOUD_INIT_TIMEOUT"
    - path: "internal/cli/up_test.go"
      provides: "TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget verifies the cloud.cloudinit.wait command carries a non-zero Timeout >= 120s by default, and that RUNNERKIT_CLOUD_INIT_TIMEOUT (e.g. \"45s\") overrides it. Also asserts cloud-init readiness completes successfully when the fake executor takes 90s (fast-forwarded via injected clock) — the gate must not abort under normal Hetzner cloud-init wall-clock."
      contains: "TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget"
    - path: "internal/provider/hetzner/destroy.go"
      provides: "Destroy skips explicit DeletePrimaryIP calls for IPs whose `state.ProviderRef.Cloud.PrimaryIPAutoDelete` flag is true (the new default per Plan 06-11 Bug 26). For legacy state (or any IP where AutoDelete is unset/false), DeletePrimaryIP is wrapped in a bounded retry loop that treats both 404 (already absent — existing behavior) AND 409 `must_be_unassigned` (cascade in flight — new) as transient. The retry loop polls every 1s up to RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT (default 30s) until the IP is gone (404) or the timeout expires. New isCascadeInFlightError predicate matches 409 + `must_be_unassigned` substring; isAlreadyAbsentError still matches only 404."
      contains: "isCascadeInFlightError"
    - path: "internal/provider/hetzner/destroy_test.go"
      provides: "TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade asserts the destroy call sequence omits any `delete:primary_ipv4` / `delete:primary_ipv6` calls when AutoDelete=true is recorded in state. TestDestroy_RetriesPrimaryIPDeleteOn409MustBeUnassigned asserts the legacy fallback path: simulated 409 must_be_unassigned on first call, 404 on the second; destroy completes without partial. TestIsCascadeInFlightError covers the 409 + substring match contract."
      contains: "TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade"
  key_links:
    - from: "Plan 06-07 attempt-18+ smoke against fresh BYO + Hetzner project"
      to: "smoke-green resume signal -> v1.0.0 tag push per D-13"
      via: "Bug 28 closes the BYO down cleanup surface (probe inspects ExitCode regardless of err); Bug 29 closes the cloud-up SSH-readiness surface (cloud-init budget aligned with host-key probe); Bug 30 closes the cloud destroy reporting surface (auto_delete skip + 409 retry); destroy_verify polls saved IDs to 404 within RUNNERKIT_SMOKE_TIMEOUT; 06-VERIFICATION.md becomes fillable + signable"
      pattern: "smoke-green"

tasks:
  - id: bug-28-down-probe-exit-status
    name: "down: probeSudoNeedsPassword inspects ExitCode regardless of err=exit-status-N wrapper (Bug 28)"
    autonomous: true
  - id: bug-29-cloud-init-timeout-budget
    name: "up: cloud-init readiness step uses an explicit budget aligned with host-key probe (Bug 29)"
    autonomous: true
  - id: bug-30-destroy-cascade-reporting
    name: "destroy: skip DeletePrimaryIP when AutoDelete=true; retry 409 must_be_unassigned for legacy state (Bug 30)"
    autonomous: true
---

# Plan 06-12: Down Sudo Probe + Cloud-Init Timeout + Destroy Cascade Reporting Fixes

## Context

Plan 06-07 attempt-17 (2026-05-06 21:40-21:51 UTC) re-ran `make smoke-live`
against `salar@mckee-small-desktop` and a real Hetzner `dat0` project after
Plan 06-11 (Bugs 24-27) landed. Plan 06-11's fixes were VERIFIED LIVE in
this same attempt (preserved as positive findings — see truth #5):

- Bug 24 host-key fingerprint determinism: `SSH OK reachable, host key matched`
  (smoke-output.log line 26).
- Bug 26 auto_delete cascade actually deleted IPs post-test (Hetzner project
  empty afterward; smoke-output.log lines 71-73).
- Bug 27 svc.sh glob: Path C scoped sudoers entry covered the runtime
  `/opt/actions-runner/runnerkit-*/svc.sh` path; `Service OK active` after
  bootstrap (smoke-output.log line 28).

Three NEW bugs blocked the v1.0.0 `smoke-green` resume signal:

1. **Bug 28** — `runnerkit down --yes` failed `runner_files` cleanup with
   `sudo: a terminal is required to read the password` despite Plan 06-11's
   sshReachable-independent gate. Trace via temporary `fmt.Fprintf` in
   `down.go` (since reverted) showed:
       `[BUG25-TRACE] gate: targetErr=<nil> needsAnyRemoteSudo=true sshReachable=true`
       `[BUG25-TRACE] probe-direct: rc=1 err=exit status 1`
       `[BUG25-TRACE] probe: needs=false probeErr=<nil>`
   The real SSH executor returns `err = exit status 1` for any non-zero
   remote rc (Go's `exec.ExitError` wrapping). `probeSudoNeedsPassword`
   at `internal/cli/down.go:440-443` swallows that err and exits "no
   password needed" before ever inspecting the marker substrings in
   `result.Stderr`. Plan 06-11 Bug 25's
   `TestDown_SudoProbeRunsEvenWhenSSHReachableFalse` used a fake executor
   that returns `(result, nil)` for non-zero rc, so the regression slipped
   past CI.

2. **Bug 29** — `runnerkit up --cloud hetzner` aborted at 42s wall-clock
   with `ERROR Cloud machine is not ready for runner registration yet`.
   Hetzner cloud-init on `cpx22`/`ubuntu-24.04` typically needs 60-120s.
   Plan 06-10 Bug 22 set `hetzner.ProbeHostKeyWithRetry` budget to 60×5s =
   300s for the HOST KEY install gate, but the cloud-init `--wait` step at
   `internal/cli/up.go:908` does not set an explicit `Timeout` on the
   `remote.Command`, so it inherits whatever default deadline the remote
   executor applies. The 42s observed wall-clock indicates that default is
   well below cloud-init's typical needs.

3. **Bug 30** — `runnerkit destroy --yes` (via the smoke trap) reported
   `provider_primary_ip: pending Primary IP must be unassigned
   (must_be_unassigned, ...)` for both IPv4 + IPv6 IDs and exited non-zero
   with `RKD-PROV-006`. The Hetzner project DID end up empty post-test
   (the cascade worked), but the synchronous report was wrong because:
       - `internal/provider/hetzner/destroy.go:95-96` still calls
         `DeletePrimaryIP` explicitly for both saved IDs.
       - The auto_delete cascade is triggered by `server.Delete` but is
         async on Hetzner's side. While in flight, `PrimaryIP.Delete`
         returns HTTP 409 `must_be_unassigned`, NOT 404.
       - `isAlreadyAbsentError` (destroy.go:198) only matches 404 + the
         text "not found" / "404", so 409 surfaces as a hard failure.
   Plan 06-11 Bug 26's commit comment claims "the explicit DeletePrimaryIP
   calls now race against the cascade (Hetzner returns 404 once the
   cascade completes; isAlreadyAbsentError already treats 404 as a no-op)"
   — that's correct ONLY after the cascade finishes; while it's in flight
   the API returns 409, which is what the live test hit.

## Bug Summary

| Bug | Description | Surface | Detected | Severity |
|-----|-------------|---------|----------|----------|
| 28 | `probeSudoNeedsPassword` early-returns on `err = exit status N` from real SSH executor | `runnerkit down` BYO cleanup | Plan 06-07 attempt-17 BYO smoke 2026-05-06 | BLOCKER |
| 29 | `cloud.cloudinit.wait` has no explicit Timeout; aborts at 42s before Hetzner cloud-init typical 60-120s | `runnerkit up --cloud hetzner` | Plan 06-07 attempt-17 cloud smoke 2026-05-06 | BLOCKER |
| 30 | `DeletePrimaryIP` race vs auto_delete cascade returns 409 `must_be_unassigned` (not 404); `isAlreadyAbsentError` only matches 404 | `runnerkit destroy --cloud hetzner` | Plan 06-07 attempt-17 cloud smoke 2026-05-06 | BLOCKER |

## Approach

- **Bug 28:** rewrite `probeSudoNeedsPassword` (down.go:431-458) to inspect
  `result.ExitCode` + `result.Stderr + result.Stdout` REGARDLESS of err. A
  non-zero exit code with err=exit-status-N is the EXPECTED case for a
  password-protected sudo (rc=1 + stderr containing "password is required"
  or "a terminal is required"). The early `if err != nil { return false, nil }`
  guard misclassifies that as "no password needed" and skips the prompt.
  Only treat err as fatal when err is NOT an exit-status wrapper (e.g.,
  context.Canceled, dial failure, executor unable to start the command).
  Implementation: keep the function signature; reorder so the ExitCode
  check happens first, and only fall through on err if `result.ExitCode == 0
  && err != nil` (which means the executor returned a non-exit error).
  Add a regression test that uses a fake executor returning `(remote.Result{ExitCode: 1, Stderr: "sudo: a password is required\n"}, errors.New("exit status 1"))`
  to lock in the real-SSH-executor contract.

- **Bug 29:** add explicit `Timeout` to the `cloud.cloudinit.wait` remote
  command at up.go:908. Default 300s aligns with the existing
  `hetzner.ProbeHostKeyWithRetry` budget (60×5s) so cloud-init has a
  single coherent deadline. Expose `RUNNERKIT_CLOUD_INIT_TIMEOUT` env var
  as an override (parse via `time.ParseDuration`; default to 300s when
  empty/unparseable). Log a renderer.Step `cloud.cloudinit.wait` line so
  smoke output shows the budget. Add a regression test
  `TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget` that asserts:
  (a) the cloud.cloudinit.wait command carries Timeout >= 120s by default,
  (b) RUNNERKIT_CLOUD_INIT_TIMEOUT="45s" overrides it to 45s, and
  (c) cloud-init readiness completes successfully when the fake executor
  reports rc=0 after a simulated 90s wall-clock (verified via the existing
  testsupport.RemoteExecutor pattern).

- **Bug 30:** dual-path fix in `internal/provider/hetzner/destroy.go`:
    - **Primary path (auto_delete cascade — the new Plan 06-11 default):**
      check `state.ProviderRef.Cloud.PrimaryIPv4AutoDelete` /
      `PrimaryIPv6AutoDelete` (or equivalent inferred-from-provision flag);
      when true, SKIP the explicit `DeletePrimaryIP` call entirely. The
      cascade handles deletion; `cmd/_smokebin/destroy_verify` polls until
      404 within `RUNNERKIT_SMOKE_TIMEOUT`. If the AutoDelete flag is not
      stored in state today, this plan adds it: provision.go records
      `AutoDelete: true` for both IPv4 + IPv6 at create time (matches the
      Hetzner default behavior post-Plan-06-11 Bug 26), and destroy.go
      reads it from `mergedProviderIDs` or a new `mergedAutoDeleteFlags`
      helper.
    - **Legacy fallback (AutoDelete unset/false):** wrap the existing
      `DeletePrimaryIP` call in a bounded retry loop. Treat 409
      `must_be_unassigned` as a transient cascade-in-flight signal via a
      new `isCascadeInFlightError(err)` predicate (`StatusCode() == 409`
      AND error message lower-cases to contain "must_be_unassigned"); on
      that condition, sleep 1s and retry up to
      `RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT` (default 30s). On 404,
      proceed (already-absent — existing behavior). On any other error,
      surface as a real failure. Sleep is injectable for tests.

  Add three regression tests in destroy_test.go:
    - `TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade`: ProviderRef
      has Cloud.PrimaryIPv4AutoDelete=true + PrimaryIPv6AutoDelete=true;
      assert call sequence omits both `delete:primary_ipv4` and
      `delete:primary_ipv6`.
    - `TestDestroy_RetriesPrimaryIPDeleteOn409MustBeUnassigned`: legacy
      ProviderRef (AutoDelete unset); fake client returns a 409
      `must_be_unassigned` error on the first DeletePrimaryIP call and a
      404 on the second; assert destroy completes without Partial=true.
    - `TestIsCascadeInFlightError`: unit test for the new predicate
      covering the 409 + substring match contract.

## Out of Scope

- Plan 06-11 work (Bugs 24-27) — verified live in attempt-17, NOT regressed.
- The maintainer human-action checkpoint (Plan 06-07) — closes after Plan
  06-12 lands and full smoke passes attempt-18.
- New error codes for Bugs 28/29/30 — existing RKD-CLEAN-003 (down cleanup),
  RKD-PROV-006 (cloud destroy), and the cloud-readiness-failed surface
  cover these; no codes.go additions.
- Re-architecting `probeSudoNeedsPassword` to return more granular signal
  (e.g., `(needs bool, isPasswordless bool, err error)`) — the current
  bool return is sufficient if the early-err-guard is fixed.
- Restoring `Plan 06-10 Bug 23` manual unassign-before-delete ordering —
  Plan 06-11 Bug 26 explicitly rejected that path because of the
  `Server must be offline` failure mode. The 409 retry path here is the
  smaller surface change.

## Out-of-Plan Maintainer Action (post-landing)

After Plan 06-12 lands, before re-running `make smoke-live`:

1. **Recover the BYO host install dir.** Plan 06-07 attempt-17 left a stale
   install dir at `/opt/actions-runner/runnerkit-*` on
   `salar@mckee-small-desktop` because Bug 28 blocked cleanup. Once the
   probe fix lands, run:

   ```bash
   go run ./cmd/runnerkit down --repo $RUNNERKIT_SMOKE_REPO --yes
   ```

   Type the sudo password when prompted (the prompter now fires
   correctly). Verify the install dir is gone:

   ```bash
   ssh $RUNNERKIT_SMOKE_BYO_HOST 'ls /opt/actions-runner/ 2>/dev/null | head'
   ```

   If `runnerkit-*` directories still appear, manually rm them as a
   one-time recovery (`sudo rm -rf /opt/actions-runner/runnerkit-*`) — the
   Plan 06-12 fix only ensures FUTURE runs cleanup correctly; it cannot
   retroactively clean an already-orphaned install.

2. **Re-verify Hetzner project empty** (D-12 gate 1 precondition):

   ```bash
   hcloud server list ; hcloud firewall list ; hcloud primary-ip list ; hcloud ssh-key list
   ```

3. **Re-run** `make smoke-live` per Plan 06-07 sequence. Expected outcomes:
    - BYO smoke: `runnerkit down` cleans all artifacts, exit 0, no
      `runner_files: failed sudo: a terminal is required`.
    - Cloud smoke: `runnerkit up --cloud hetzner` succeeds within
      typical 60-120s (300s budget), runner registers, `runnerkit destroy`
      exits 0 with `provider_primary_ip: status=done`, destroy_verify
      polls saved IDs to 404.

## Verification

- Full repo `go test ./... -count=1 -race` passes.
- New unit tests:
    - `internal/cli/down_test.go::TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper`
    - `internal/cli/up_test.go::TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget`
    - `internal/provider/hetzner/destroy_test.go::TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade`
    - `internal/provider/hetzner/destroy_test.go::TestDestroy_RetriesPrimaryIPDeleteOn409MustBeUnassigned`
    - `internal/provider/hetzner/destroy_test.go::TestIsCascadeInFlightError`
- Existing tests unchanged: `TestDown_SudoProbeRunsEvenWhenSSHReachableFalse`
  (Plan 06-11 Bug 25), `TestDownThreadsSudoPasswordWhenSudoRequiresPasswordClosesBug21`
  (Plan 06-10 Bug 21), `TestDestroy_AutoDeleteCascadeNoUnassign`
  (Plan 06-11 Bug 26), `TestDestroyTreatsAlreadyAbsentDetachAsSuccess`
  (Plan 06-11 Bug 26).
- Plan 06-07 attempt-18+ BYO+cloud smoke completes end-to-end; Hetzner
  project empty afterward; 06-VERIFICATION.md fillable + signable.

<objective>
Close three bugs (Bugs 28, 29, 30) surfaced by Plan 06-07 attempt-17 live
smoke (2026-05-06) so the v1.0.0 maintainer smoke can complete end-to-end
on attempt-18+:

- Bug 28: `runnerkit down` BYO cleanup correctly prompts for sudo password
  when probe returns `err = exit status N` for non-zero remote rc.
- Bug 29: `runnerkit up --cloud hetzner` waits long enough for Hetzner
  cloud-init (typical 60-120s; 300s budget aligned with Plan 06-10 Bug 22
  host-key probe).
- Bug 30: `runnerkit destroy --yes` exits 0 when AutoDelete cascade
  removes primary IPs (skip explicit DeletePrimaryIP on AutoDelete=true;
  retry on 409 must_be_unassigned for legacy state).

Purpose: Plan 06-07 attempt-18 produces SMOKE-GREEN; 06-VERIFICATION.md
gets filled with real durations + 5 cloud resource IDs + EUR cost +
maintainer signature; v1.0.0 tag is pushable per D-13. Closes the
re-verification cycle Plan 06-11 -> attempt-17 SMOKE-RED -> Plan 06-12.

Output: 6 production+test file modifications committed; full test suite
green; ready for `/gsd:execute-phase 06 --gaps-only` re-run of Plan 06-07.
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
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-11-status-down-sudoers-cloud-destroy-fixes-PLAN.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-11-status-down-sudoers-cloud-destroy-fixes-SUMMARY.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-07-live-smoke-rerun-and-baseline-fillin-PLAN.md
@smoke-output.log
@internal/cli/down.go
@internal/cli/up.go
@internal/provider/hetzner/destroy.go
@internal/provider/hetzner/client.go
@internal/provider/hetzner/provision.go

<interfaces>
<!-- Existing contracts the executor uses directly. -->

internal/cli/down.go (Bug 28 target):
- `probeSudoNeedsPassword(ctx, executor remote.Executor, target remote.Target) (bool, error)`
  at line 431. Currently early-returns `(false, nil)` on any non-nil err
  (line 440-443). Real SSH executor returns `err = exit status N` for any
  non-zero remote rc — see `internal/remote/system.go:81-89`:
    err := cmd.Run()
    result := Result{Stdout: ..., Stderr: ..., ExitCode: 0}
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            result.ExitCode = exitErr.ExitCode()
        } else {
            result.ExitCode = -1
        }
    }
    return result, err
  So `result.ExitCode` is populated correctly even when err != nil; the
  fix is to inspect ExitCode + Stderr regardless.

internal/cli/up.go (Bug 29 target):
- `waitCloudTargetReady(ctx, deps, machine) (preflight.Report, remote.HostKey, provider.Machine, error)`
  at line 891. The cloud-init wait is at line 908:
    result, err := deps.RemoteExecutor.Run(ctx, target, remote.Command{
        ID: "cloud.cloudinit.wait",
        Script: "cloud-init status --wait || test -f /var/lib/cloud/instance/boot-finished",
    })
  No explicit Timeout. The fix adds:
    Timeout: cloudInitTimeoutFromEnv(deps.Env, defaultCloudInitTimeout),
  where defaultCloudInitTimeout = 5 * time.Minute (300s, aligned with
  hetzner.HostKeyProbeOptions.Attempts × Interval = 60 × 5s = 300s).
- Existing pattern for env-var-driven Duration parsing: see
  `internal/update/check.go` for HTTP timeout patterns; this plan adds a
  small helper `cloudInitTimeoutFromEnv(env map[string]string, fallback time.Duration) time.Duration`.

internal/provider/hetzner/destroy.go (Bug 30 target):
- `Destroy(ctx, ref state.ProviderRef) (provider.DestroyResult, error)` at
  line 19. Lines 81-97 handle the post-cascade explicit deletes. The
  current `apply` closure at lines 26-45 calls `delete(ctx, parsed)` once
  and uses `isAlreadyAbsentError` to suppress 404. Bug 30 fix replaces
  the primary-IP `apply` calls with a new `applyPrimaryIPDeleteWithRetry`
  closure that:
    1. Checks state for AutoDelete=true and skips entirely if set.
    2. Otherwise wraps the delete in a bounded retry loop, treating 409
       must_be_unassigned as transient via `isCascadeInFlightError(err)`.
- New predicate `isCascadeInFlightError(err error) bool` lives next to
  `isAlreadyAbsentError(err error) bool` at line 198. Pattern matches:
    var statusErr interface{ StatusCode() int }
    if errors.As(err, &statusErr) && statusErr.StatusCode() == 409 {
        return strings.Contains(strings.ToLower(err.Error()), "must_be_unassigned")
    }
    return strings.Contains(strings.ToLower(err.Error()), "must_be_unassigned")
  (The substring fallback covers test fakes that return
  `errors.New("must_be_unassigned ...")` without a StatusCode method.)

internal/state package (Bug 30 state shape):
- The relevant state shape is `state.ProviderRef.Cloud` (an existing
  struct that holds ServerID, SSHKeyID, FirewallID, PrimaryIPv4ID,
  PrimaryIPv6ID — see `internal/provider/hetzner/destroy.go:189-194`).
  Plan 06-12 adds two boolean fields:
    Cloud.PrimaryIPv4AutoDelete bool
    Cloud.PrimaryIPv6AutoDelete bool
  `internal/provider/hetzner/provision.go` sets these to true at create
  time (matches Hetzner's default behavior for auto-allocated primary IPs
  via `EnableIPv4: true, EnableIPv6: true`). State migration is NOT
  required: a missing field defaults to false, in which case Bug 30's
  legacy fallback (409 retry) handles the cleanup correctly. New state
  written by post-Plan-06-12 binaries records true.

internal/testsupport/remote.go (Bug 28 + Bug 29 test fakes):
- `RemoteExecutor.Errors map[string]error` is the existing hook for
  associating an error with a command ID. The Bug 28 test sets:
    Errors: map[string]error{"down.sudo.probe": errors.New("exit status 1")}
    Results: map[string]remote.Result{
        "down.sudo.probe": {ExitCode: 1, Stderr: "sudo: a password is required\n"},
    }
  to simulate the real-SSH-executor case. Existing
  `TestDown_SudoProbeRunsEvenWhenSSHReachableFalse` only sets Results
  (Errors map empty), which is why the regression slipped past CI.

internal/provider/hetzner/destroy_test.go (Bug 30 test fakes):
- `destroyFakeOrderedClient` already records call sequence via
  `client.calls`. Bug 30 tests extend it with `deletePrimaryIPCallCount
  int` + `deletePrimaryIPErrSequence []error` so the retry-loop test can
  return 409 on call N and 404 on call N+1.

Plan 06-11 cross-context (preserve, do not regress):
- Plan 06-11 SUMMARY confirms cascade behavior is correct in practice
  (project ends empty). The cascade-removed primary IPs are NOT a
  RunnerKit bug; the bug is the synchronous `DeletePrimaryIP` call
  reporting 409 as a failure.
- `TestDestroy_AutoDeleteCascadeNoUnassign` (06-11 destroy_test.go:183)
  asserts `delete:primary_ipv4` + `delete:primary_ipv6` ARE in the call
  sequence. After Plan 06-12 (when AutoDelete=true), those calls must
  NOT be present — the test will need a small update OR Plan 06-12 keeps
  the existing test by gating its destroyRefWithBothPrimaryIPs() on the
  legacy (AutoDelete unset) state shape, and the new
  TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade uses a separate
  ref helper that sets AutoDelete=true. The legacy test then continues
  to exercise the 404-tolerant path; the new test exercises the
  AutoDelete-skip path.

Plan 06-10 cross-context (preserve, do not regress):
- `hetzner.HostKeyProbeOptions{Attempts: 60, Interval: 5s}` is the
  source-of-truth for the cloud-init / SSH-readiness wall-clock budget
  (60 × 5s = 300s). Bug 29's RUNNERKIT_CLOUD_INIT_TIMEOUT default mirrors
  this so cloud-init has a single coherent deadline.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Bug 28 — probeSudoNeedsPassword inspects ExitCode regardless of err=exit-status-N</name>
  <files>internal/cli/down.go, internal/cli/down_test.go</files>
  <read_first>
    - internal/cli/down.go (target — lines 431-458 probeSudoNeedsPassword)
    - internal/cli/down_test.go (existing tests TestDown_SudoProbeRunsEvenWhenSSHReachableFalse + TestDownThreadsSudoPasswordWhenSudoRequiresPasswordClosesBug21 to mirror their fake-executor wiring)
    - internal/remote/system.go lines 70-90 (real SSH executor's err = *exec.ExitError contract; result.ExitCode populated alongside non-nil err)
    - internal/testsupport/remote.go (RemoteExecutor.Errors map — the new test populates Errors[id]; existing tests only set Results)
    - smoke-output.log lines 30-58 (live trace showing probe-direct: rc=1 err=exit status 1)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-11-status-down-sudoers-cloud-destroy-fixes-PLAN.md (Bug 25 context — gate fix is correct logic; Bug 28 is the deeper guard fix beneath it)
  </read_first>
  <behavior>
    Test cases for `probeSudoNeedsPassword`:
    - Test 1 (NEW — TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper):
      executor returns `(remote.Result{ExitCode: 1, Stderr: "sudo: a password is required\n"}, errors.New("exit status 1"))`
      → probe returns `(true, nil)`; password prompter is invoked once; `down.service.uninstall` + `down.files.remove` Scripts contain `printf '%s\n' "$RUNNERKIT_SUDO_PASSWORD" | sudo -S -v`. This is the EXACT shape the real SSH executor produces; the pre-fix code returns `(false, nil)` here.
    - Test 2 (EXISTING — TestDownThreadsSudoPasswordWhenSudoRequiresPasswordClosesBug21):
      executor returns `(remote.Result{ExitCode: 1, Stderr: "sudo: a password is required\n"}, nil)`
      → probe returns `(true, nil)`; existing assertions hold. (Pre-Plan-06-12 fake-executor-with-no-error case.)
    - Test 3 (EXISTING — TestDownDoesNotPromptWhenSudoIsPasswordless):
      executor returns `(remote.Result{ExitCode: 0}, nil)` → probe returns `(false, nil)`; no prompt; no script wrapping.
    - Test 4 (EXISTING — TestDown_SudoProbeRunsEvenWhenSSHReachableFalse):
      probe runs even when collectStatus reports sshReachable=false; executor returns ExitCode=1 + stderr marker → prompt fires once.
  </behavior>
  <action>
    Rewrite `probeSudoNeedsPassword` in `internal/cli/down.go` (lines 431-458) to inspect `result.ExitCode` and `result.Stderr + result.Stdout` regardless of whether err is non-nil. The new logic:

    ```go
    func probeSudoNeedsPassword(ctx context.Context, executor remote.Executor, target remote.Target) (bool, error) {
        if executor == nil {
            return false, nil
        }
        result, err := executor.Run(ctx, target, remote.Command{
            ID:      "down.sudo.probe",
            Script:  "sudo -n true",
            Timeout: 5 * time.Second,
        })
        // Bug 28 (Plan 06-12, 2026-05-06): the real SSH executor returns
        // err = *exec.ExitError for any non-zero remote rc — that's the
        // EXPECTED case for a password-protected sudo (rc=1 + stderr
        // marker). Inspect result.ExitCode + result.Stderr REGARDLESS of
        // err. Only treat err as fatal when the executor failed to run
        // the command at all (result.ExitCode == 0 with non-nil err
        // means dial failure / context cancellation / executor unable
        // to start the command — fall through to the unwrapped path).
        // See internal/remote/system.go for the err+ExitCode contract:
        // exec.ExitError -> result.ExitCode populated; other err ->
        // result.ExitCode = -1 (treated as "unknown, fall through").
        if result.ExitCode == 0 {
            return false, nil
        }
        stderr := strings.ToLower(result.Stderr + " " + result.Stdout)
        for _, marker := range []string{"password is required", "a terminal is required", "no tty present"} {
            if strings.Contains(stderr, marker) {
                return true, nil
            }
        }
        // Non-zero exit without a marker: keep the unwrapped happy path.
        // If sudo is genuinely broken the cleanup surface will surface
        // the canonical sudo failure verbatim. err (if present) is the
        // exec.ExitError wrapper around the same exit code; not fatal.
        _ = err
        return false, nil
    }
    ```

    Key invariants (lock these in via test):
    - `result.ExitCode == 0` short-circuits to passwordless before consulting err. This preserves the happy path when sudo is configured NOPASSWD.
    - `result.ExitCode != 0` + marker substring -> `(true, nil)` regardless of err. This is the password-protected-sudo case and the Bug 28 fix.
    - `result.ExitCode != 0` + no marker -> `(false, nil)` regardless of err. Defensive default that preserves Plan 06-10's documented behavior.
    - `result.ExitCode == 0` + err != nil never happens with the real SSH executor (see system.go:82-89), but defensively returns `(false, nil)` because `ExitCode==0` means sudo passwordless from the remote side.
    - Note: Plan 06-12 intentionally does NOT add a `(needs bool, err error)` distinction for executor-startup-failure (e.g., dial timeout). That is out of scope; the existing graceful-failure semantics ("fall through to unwrapped path; cleanup will surface the real error") are preserved.

    Add `TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper` in `internal/cli/down_test.go` modeled on `TestDownThreadsSudoPasswordWhenSudoRequiresPasswordClosesBug21` (line 128) but populating `exec.Errors["down.sudo.probe"] = errors.New("exit status 1")` in addition to `exec.Results["down.sudo.probe"] = remote.Result{ExitCode: 1, Stderr: "sudo: a password is required\n"}`. Assertions:
    - `prompts.calls == 1` (password prompter fired)
    - `down.service.uninstall` + `down.files.remove` Scripts contain `printf '%s\\n' "$RUNNERKIT_SUDO_PASSWORD" | sudo -S -v`
    - `RUNNERKIT_SUDO_PASSWORD=hunter2` Env on those commands
    - No "hunter2" leak in stdout/stderr

    Test name: `TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper`.

    Reference: Bug 28 closes the cascade where Plan 06-11 Bug 25 fix
    (gate drops sshReachable) was correct but ineffective because the
    deeper probe guard swallowed the err.
  </action>
  <verify>
    <automated>cd /Users/salar/Projects/spool && go test ./internal/cli/ -run "TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper|TestDownThreadsSudoPasswordWhenSudoRequiresPasswordClosesBug21|TestDownDoesNotPromptWhenSudoIsPasswordless|TestDown_SudoProbeRunsEvenWhenSSHReachableFalse" -count=1 -race</automated>
  </verify>
  <acceptance_criteria>
    - `internal/cli/down.go` does NOT contain the literal pattern `if err != nil {\n\t\t// Non-fatal — fall through to the unwrapped path; if sudo is\n\t\t// genuinely broken the cleanup surface will surface it.\n\t\treturn false, nil\n\t}` anywhere inside `probeSudoNeedsPassword`. Verify via `awk '/^func probeSudoNeedsPassword/,/^}/' internal/cli/down.go | grep -c 'return false, nil$'` returning ≤ 3 (one for nil-executor guard, one for ExitCode==0, one for non-marker fallback) — i.e., NO early-err-guard return.
    - `grep -n "Bug 28" internal/cli/down.go` returns at least one match inside the probeSudoNeedsPassword body.
    - `go test -run TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper ./internal/cli/ -count=1` exits 0.
    - `go test ./internal/cli/ -count=1 -race` exits 0 (full down package + sibling tests still green).
    - `go vet ./internal/cli/...` clean.
    - Commit message follows pattern `fix(06-12): bug 28 — probeSudoNeedsPassword inspects ExitCode regardless of exit-status-N err wrapper`.
  </acceptance_criteria>
  <done>
    `runnerkit down --yes` against a real BYO host with password-protected
    sudo prompts for the password and threads it through `runner_files`
    cleanup; the regression test locks in the real-SSH-executor contract
    so future fakes that omit Errors[id] cannot mask this regression.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Bug 29 — cloud-init readiness step uses an explicit budget aligned with Hetzner host-key probe</name>
  <files>internal/cli/up.go, internal/cli/up_test.go</files>
  <read_first>
    - internal/cli/up.go lines 880-940 (waitCloudTargetReady + probeCloudHostKey)
    - internal/cli/up.go line 908 (the cloud.cloudinit.wait command without explicit Timeout)
    - internal/provider/hetzner/provision.go lines 410-488 (HostKeyProbeOptions + ProbeHostKeyWithRetry — defaults 60×5s = 300s; sleep injectable)
    - smoke-output.log lines 60-82 (live failure: 42s before cloud_readiness_failed; cloud-init typical 60-120s)
    - internal/testsupport/remote.go (RemoteExecutor doesn't honor Timeout — tests verify Timeout is SET on the Command, not that it actually elapsed)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md D-12 (RUNNERKIT_SMOKE_TIMEOUT default 300s — same precedent for cloud-budget env vars)
  </read_first>
  <behavior>
    Test cases for the cloud-init wait gate:
    - Test 1 (NEW — TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget):
      Default budget: with `deps.Env["RUNNERKIT_CLOUD_INIT_TIMEOUT"]` unset (or empty), the `cloud.cloudinit.wait` Command issued to the executor MUST have `Timeout >= 120 * time.Second` (target: 300s default). Assert by inspecting `exec.Commands` for the matching ID and checking `cmd.Timeout`.
    - Test 2 (NEW — same test, sub-case "override"):
      Override: `deps.Env["RUNNERKIT_CLOUD_INIT_TIMEOUT"] = "45s"` -> the Command's Timeout MUST equal `45 * time.Second`.
    - Test 3 (NEW — same test, sub-case "invalid_falls_back"):
      Invalid override: `deps.Env["RUNNERKIT_CLOUD_INIT_TIMEOUT"] = "not-a-duration"` -> the Command's Timeout MUST equal the default (300s); `time.ParseDuration` failures fall back to default rather than zero (zero would be "no timeout" — a regression).
    - Test 4 (NEW — same test, sub-case "rc_zero_succeeds"):
      Executor returns `Result{ExitCode: 0}` for cloud.cloudinit.wait -> waitCloudTargetReady proceeds past the cloud-init gate (no premature abort).
  </behavior>
  <action>
    Modify `internal/cli/up.go` `waitCloudTargetReady` (lines 891-923):

    1. Add a small helper near the bottom of the file (next to `probeCloudHostKey`):

       ```go
       const defaultCloudInitTimeout = 5 * time.Minute

       // cloudInitTimeoutFromEnv resolves the RUNNERKIT_CLOUD_INIT_TIMEOUT
       // env var into a usable Duration. Empty / unparseable values fall
       // back to defaultCloudInitTimeout (300s, aligned with
       // hetzner.HostKeyProbeOptions Attempts × Interval = 60 × 5s).
       // Bug 29 (Plan 06-12, 2026-05-06): the live attempt-17 smoke
       // aborted at 42s with cloud_readiness_failed because this command
       // had no explicit Timeout; default behavior was below Hetzner
       // cloud-init's typical 60-120s wall-clock on cpx22/ubuntu-24.04.
       func cloudInitTimeoutFromEnv(env map[string]string) time.Duration {
           raw := strings.TrimSpace(env["RUNNERKIT_CLOUD_INIT_TIMEOUT"])
           if raw == "" {
               return defaultCloudInitTimeout
           }
           parsed, err := time.ParseDuration(raw)
           if err != nil || parsed <= 0 {
               return defaultCloudInitTimeout
           }
           return parsed
       }
       ```

    2. Update the cloud-init wait Run call at line 908 to:

       ```go
       result, err := deps.RemoteExecutor.Run(ctx, target, remote.Command{
           ID:      "cloud.cloudinit.wait",
           Script:  "cloud-init status --wait || test -f /var/lib/cloud/instance/boot-finished",
           Timeout: cloudInitTimeoutFromEnv(deps.Env),
       })
       ```

       The `deps.Env` field already exists on `Dependencies` (used by
       update-check + Hetzner token discovery); follow the existing
       pattern. If `deps.Env` is nil, `cloudInitTimeoutFromEnv` should
       still return `defaultCloudInitTimeout` (handle nil map by
       reading the zero value `""` which falls back).

    3. Add an explanatory comment block at line 908 referencing Bug 29
       and the alignment with `hetzner.HostKeyProbeOptions` (Plan 06-10
       Bug 22) so future maintainers understand why both gates share
       the 300s budget.

    Add `TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget` in
    `internal/cli/up_test.go` (or new file `internal/cli/up_cloud_init_timeout_test.go`
    if up_test.go is bloated):

    - Build a `Dependencies` with `RemoteExecutor: &testsupport.RemoteExecutor{Results: ...}` and `Env: map[string]string{...}`.
    - Stub `cloud.cloudinit.wait` -> `Result{ExitCode: 0}` so the gate passes.
    - Stub the host-key probe to return a non-empty fingerprint.
    - Call `waitCloudTargetReady(ctx, deps, machine)` (or invoke the
      smaller helper directly if waitCloudTargetReady is not directly
      callable from the test — use the testsupport executor pattern
      from the existing cloud test files).
    - Walk `exec.Commands` for ID `"cloud.cloudinit.wait"` and assert
      `cmd.Timeout` per the four sub-cases listed in <behavior>.
    - For the "rc_zero_succeeds" sub-case, also assert
      `waitCloudTargetReady` returns nil error past the cloud-init
      gate (preflight may still fail on the test fixture; assert only
      the gate-passed property by checking that exec.Commands contains
      a preflight-related command after cloud.cloudinit.wait, OR by
      injecting a preflight stub that returns Passed=true).

    Test name: `TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget`.
  </action>
  <verify>
    <automated>cd /Users/salar/Projects/spool && go test ./internal/cli/ -run "TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget" -count=1 -race && go test ./internal/cli/ -count=1 -race</automated>
  </verify>
  <acceptance_criteria>
    - `grep -n "RUNNERKIT_CLOUD_INIT_TIMEOUT" internal/cli/up.go` returns at least one match.
    - `grep -n "cloudInitTimeoutFromEnv" internal/cli/up.go` returns at least 2 matches (definition + call site).
    - `grep -n "Bug 29" internal/cli/up.go` returns at least one match next to the cloud.cloudinit.wait Run call.
    - The remote.Command for ID "cloud.cloudinit.wait" carries a non-zero `Timeout` field (verified by the new test inspecting exec.Commands).
    - `go test -run TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget ./internal/cli/ -count=1` exits 0.
    - `go test ./internal/cli/ -count=1 -race` exits 0 (existing cloud-up tests still green; no regression in waitCloudTargetReady or probeCloudHostKey).
    - `go vet ./internal/cli/...` clean.
    - Commit message follows pattern `fix(06-12): bug 29 — cloud.cloudinit.wait honors RUNNERKIT_CLOUD_INIT_TIMEOUT budget aligned with host-key probe`.
  </acceptance_criteria>
  <done>
    `runnerkit up --cloud hetzner` against a fresh Hetzner cpx22 server
    succeeds in the typical 60-120s window without aborting at 42s; the
    300s default budget gives cloud-init headroom for slower images /
    regions; smoke harnesses can override via env var without code
    changes.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: Bug 30 — destroy skips DeletePrimaryIP on AutoDelete=true; retries 409 must_be_unassigned for legacy state</name>
  <files>internal/provider/hetzner/destroy.go, internal/provider/hetzner/destroy_test.go</files>
  <read_first>
    - internal/provider/hetzner/destroy.go (whole file — current Destroy + apply + isAlreadyAbsentError contract)
    - internal/provider/hetzner/client.go lines 124-147 (DeletePrimaryIP + UnassignPrimaryIP signatures + Bug 23/26 doc comments)
    - internal/provider/hetzner/destroy_test.go lines 100-210 (existing destroy tests + destroyFakeOrderedClient + destroyRefWithBothPrimaryIPs)
    - internal/state/schema.go (state.ProviderRef.Cloud struct shape — find PrimaryIPv4ID/PrimaryIPv6ID fields to know where to add AutoDelete booleans)
    - internal/provider/hetzner/provision.go lines 100-148 (ServerCreatePublicNet EnableIPv4=true ... — the create site that should record AutoDelete=true)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-11-status-down-sudoers-cloud-destroy-fixes-SUMMARY.md (Bug 26 cascade decision rationale — preserve, don't regress)
    - smoke-output.log lines 84-110 (live trace: provider_primary_ip pending must_be_unassigned 409 race vs cascade)
  </read_first>
  <behavior>
    Test cases for `Destroy` cascade-aware path:
    - Test 1 (NEW — TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade):
      ProviderRef has Cloud.PrimaryIPv4AutoDelete=true + PrimaryIPv6AutoDelete=true. Destroy call sequence MUST be `[detach:firewall, delete:server, delete:ssh_key, delete:firewall]` — explicitly NO `delete:primary_ipv4` / `delete:primary_ipv6`. Result.Partial=false; Pending is empty; both primary-IP ArtifactResults have Status="skipped" with Message="auto_delete cascade".
    - Test 2 (NEW — TestDestroy_RetriesPrimaryIPDeleteOn409MustBeUnassigned):
      Legacy ProviderRef (Cloud.PrimaryIPv4AutoDelete=false). Fake client's DeletePrimaryIP returns a custom error `&hcloudStubError{code: 409, msg: "primary IP must be unassigned (must_be_unassigned, abc123)"}` on the FIRST call and `nil` (or 404 stub) on the SECOND call. Destroy completes with Partial=false; the call sequence shows `delete:primary` recorded twice (or via call counter). Sleep is injected so the test runs in <100ms.
    - Test 3 (NEW — TestDestroy_RetryExhaustsBudgetThenSurfacesAsPartial):
      Same as Test 2 but the fake client returns 409 must_be_unassigned on EVERY call. With RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT=10ms (very small), the retry loop exits and Destroy returns Partial=true with `provider_primary_ip_pending` recorded.
    - Test 4 (NEW — TestIsCascadeInFlightError):
      Unit test: 409 + "must_be_unassigned" -> true; 404 + "not_found" -> false; 409 + other text -> false; nil -> false; non-status error containing "must_be_unassigned" -> true (substring fallback for test-fake errors).
    - Test 5 (EXISTING — TestDestroy_AutoDeleteCascadeNoUnassign): Plan 06-11's existing test uses `destroyRefWithBothPrimaryIPs()` which today has no AutoDelete flag. Update it OR add a new ref helper `destroyRefWithBothPrimaryIPsAutoDeleteFalse()` to keep the legacy-fallback path covered. Either way, the test must continue to assert NO unassign:* calls (Plan 06-11 contract preserved). The expected call sequence for legacy state is `[detach:firewall, delete:server, delete:ssh_key, delete:primary_ipv4, delete:primary_ipv6, delete:firewall]` — same as today, because the 404 cascade case is the happy path when AutoDelete is unset.
  </behavior>
  <action>
    Three coordinated changes:

    **A. Add AutoDelete fields to state.ProviderRef.Cloud.**

    In `internal/state/schema.go` (or wherever `ProviderRef.Cloud` is defined — discoverable via grep `Cloud\s*struct` in internal/state), add two fields:

    ```go
    PrimaryIPv4AutoDelete bool `json:"primary_ipv4_auto_delete,omitempty"`
    PrimaryIPv6AutoDelete bool `json:"primary_ipv6_auto_delete,omitempty"`
    ```

    No state migration required — missing field defaults to false, in
    which case the legacy 409-retry path handles the cleanup correctly.
    Use `omitempty` so existing fixtures (state.json snapshots in tests)
    don't need updating just because the field exists.

    **B. Record AutoDelete=true at provision time.**

    In `internal/provider/hetzner/provision.go`, after `client.CreateServer` returns successfully (around line 145 where `addPublicNetResourceIDs` runs), add a step that sets:

    ```go
    machine.Provider.Cloud.PrimaryIPv4AutoDelete = true
    machine.Provider.Cloud.PrimaryIPv6AutoDelete = true
    ```

    These match Hetzner's default behavior for IPs auto-allocated via
    `EnableIPv4: true, EnableIPv6: true` (the property Plan 06-11 Bug 26
    locked in). If `addPublicNetResourceIDs` already mirrors fields onto
    `machine.Provider.Cloud`, extend it; otherwise set them directly on
    the result struct in machineFromServer or buildCloudRepositoryState.
    Update / extend the existing `TestProvisionEnablesPublicIPsWithoutOverridingForBug26`
    test (or add a sibling) to assert AutoDelete=true is stored on the
    returned Machine struct.

    **C. Rewrite the destroy primary-IP path.**

    In `internal/provider/hetzner/destroy.go`:

    1. Add a new package-level constant + helper:

       ```go
       const defaultDestroyPrimaryIPTimeout = 30 * time.Second

       // destroyPrimaryIPTimeoutFromEnv resolves
       // RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT into a usable Duration for
       // the Bug 30 retry loop. Empty / unparseable -> default 30s.
       func destroyPrimaryIPTimeoutFromEnv(env map[string]string) time.Duration {
           raw := strings.TrimSpace(env["RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT"])
           if raw == "" {
               return defaultDestroyPrimaryIPTimeout
           }
           parsed, err := time.ParseDuration(raw)
           if err != nil || parsed <= 0 {
               return defaultDestroyPrimaryIPTimeout
           }
           return parsed
       }
       ```

       Note: the Hetzner Provider already has access to `p.Env` per the
       existing `p.client()` -> `ResolveToken(p.Env)` pattern; thread
       `p.Env` through to the helper. Add an injectable
       `Sleep func(time.Duration)` field + getter on Provider (default
       `time.Sleep`) so tests can fast-forward.

    2. Add `isCascadeInFlightError(err error) bool` next to `isAlreadyAbsentError`:

       ```go
       // isCascadeInFlightError returns true when err is a Hetzner 409
       // `must_be_unassigned` response, which indicates the auto_delete
       // cascade is still in flight on the server side. Bug 30 (Plan
       // 06-12, 2026-05-06): destroy retries on this signal until 404
       // (cascade complete -> isAlreadyAbsentError) or the bounded
       // timeout expires.
       func isCascadeInFlightError(err error) bool {
           if err == nil {
               return false
           }
           text := strings.ToLower(err.Error())
           if !strings.Contains(text, "must_be_unassigned") {
               return false
           }
           var status interface{ StatusCode() int }
           if errors.As(err, &status) {
               return status.StatusCode() == 409
           }
           // Test fakes that don't implement StatusCode but include the
           // canonical substring still match. Real hcloud-go responses
           // always implement StatusCode.
           return true
       }
       ```

    3. Replace the two `apply(artifactProviderPrimaryIP, ids["primary_ipv4"], client.DeletePrimaryIP, ...)` calls at destroy.go:95-96 with a new `applyPrimaryIPDelete` closure that takes the IP id, the AutoDelete flag, and the retry budget:

       ```go
       applyPrimaryIPDelete := func(idStr string, autoDelete bool) {
           if strings.TrimSpace(idStr) == "" {
               result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderPrimaryIP, Status: "skipped", Message: "not tracked"})
               return
           }
           if autoDelete {
               // Bug 30 (Plan 06-12, 2026-05-06): primary IP carries
               // AutoDelete=true; the server.Delete cascade handles it.
               // Skip the explicit DeletePrimaryIP call entirely so the
               // smoke trap does not race against the in-flight cascade
               // (Hetzner returns 409 must_be_unassigned during the
               // window, which isAlreadyAbsentError can't silence).
               result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderPrimaryIP, Status: "skipped", Message: "auto_delete cascade"})
               return
           }
           parsed, parseErr := parseID(idStr)
           if parseErr != nil {
               result.Partial = true
               result.Pending = append(result.Pending, "provider_primary_ip_pending")
               result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderPrimaryIP, Status: "pending", Message: parseErr.Error()})
               return
           }
           // Legacy fallback: retry 409 must_be_unassigned until 404.
           deadline := time.Now().Add(destroyPrimaryIPTimeoutFromEnv(p.Env))
           sleep := p.sleep // injectable; defaults to time.Sleep
           if sleep == nil {
               sleep = time.Sleep
           }
           for {
               err := client.DeletePrimaryIP(ctx, parsed)
               if err == nil || isAlreadyAbsentError(err) {
                   result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderPrimaryIP, Status: "done"})
                   return
               }
               if !isCascadeInFlightError(err) || time.Now().After(deadline) {
                   result.Partial = true
                   result.Pending = append(result.Pending, "provider_primary_ip_pending")
                   result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderPrimaryIP, Status: "pending", Message: err.Error()})
                   return
               }
               sleep(1 * time.Second)
           }
       }
       ```

       Replace lines 95-96 in destroy.go with:
       ```go
       applyPrimaryIPDelete(ids["primary_ipv4"], cloudAutoDeleteFlag(ref, "primary_ipv4"))
       applyPrimaryIPDelete(ids["primary_ipv6"], cloudAutoDeleteFlag(ref, "primary_ipv6"))
       ```

       where `cloudAutoDeleteFlag(ref, kind)` reads from
       `ref.Cloud.PrimaryIPv4AutoDelete` / `PrimaryIPv6AutoDelete` and
       defaults to false (legacy state).

    4. Add a `sleep func(time.Duration)` field on `Provider` (next to
       existing fields like `Env`, `Client`) and an option setter
       `WithSleep(sleep func(time.Duration)) Option` for tests. Default
       to nil; call sites use `time.Sleep` as fallback.

    Add the new tests + extend `destroy_test.go` per <behavior> Test 1-4. For Test 5 (legacy compatibility), update `destroyRefWithBothPrimaryIPs` (line 231) to explicitly set `Cloud.PrimaryIPv4AutoDelete: false, PrimaryIPv6AutoDelete: false` so its intent is clear. Add a sibling `destroyRefWithBothPrimaryIPsAutoDeleteTrue` for the new Test 1.

    Test fake error type for Tests 2-3:
    ```go
    type hcloudStubError struct {
        code int
        msg  string
    }
    func (e *hcloudStubError) Error() string  { return e.msg }
    func (e *hcloudStubError) StatusCode() int { return e.code }
    ```

    Test names:
    - `TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade`
    - `TestDestroy_RetriesPrimaryIPDeleteOn409MustBeUnassigned`
    - `TestDestroy_RetryExhaustsBudgetThenSurfacesAsPartial`
    - `TestIsCascadeInFlightError`

    Plan 06-11 cross-check: `TestDestroy_AutoDeleteCascadeNoUnassign`'s
    expected call sequence WILL change because the new code skips the
    explicit DeletePrimaryIP calls when AutoDelete=true. Update its
    `destroyRefWithBothPrimaryIPs()` helper to set AutoDelete=true (the
    new Plan 06-12 default — matches what provision.go now records), and
    update the `want` slice in the test to remove `delete:primary_ipv4`
    and `delete:primary_ipv6` entries. The "no unassign" invariant
    remains — the test name is still accurate. Add a new sibling test
    `TestDestroy_LegacyAutoDeleteFalseStillCallsDeletePrimaryIP` that
    uses AutoDelete=false to keep the legacy-fallback path covered with
    its full call sequence.
  </action>
  <verify>
    <automated>cd /Users/salar/Projects/spool && go test ./internal/provider/hetzner/ -run "TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade|TestDestroy_RetriesPrimaryIPDeleteOn409MustBeUnassigned|TestDestroy_RetryExhaustsBudgetThenSurfacesAsPartial|TestIsCascadeInFlightError|TestDestroy_AutoDeleteCascadeNoUnassign|TestDestroy_LegacyAutoDeleteFalseStillCallsDeletePrimaryIP|TestDestroyDeletesThenVerifyDescribesBeforeSuccess|TestDestroyTreatsAlreadyAbsentDetachAsSuccess" -count=1 -race && go test ./internal/provider/hetzner/ -count=1 -race</automated>
  </verify>
  <acceptance_criteria>
    - `grep -n "isCascadeInFlightError" internal/provider/hetzner/destroy.go` returns at least 2 matches (definition + usage in applyPrimaryIPDelete).
    - `grep -n "RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT" internal/provider/hetzner/destroy.go` returns at least one match.
    - `grep -n "PrimaryIPv4AutoDelete\|PrimaryIPv6AutoDelete" internal/state/schema.go internal/provider/hetzner/provision.go internal/provider/hetzner/destroy.go` returns matches in all three files (state field, provision write, destroy read).
    - `grep -n "auto_delete cascade" internal/provider/hetzner/destroy.go` returns at least one match (the skipped-status message).
    - `grep -n "Bug 30" internal/provider/hetzner/destroy.go` returns at least one match.
    - `go test -run "TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade|TestDestroy_RetriesPrimaryIPDeleteOn409MustBeUnassigned|TestDestroy_RetryExhaustsBudgetThenSurfacesAsPartial|TestIsCascadeInFlightError" ./internal/provider/hetzner/ -count=1` exits 0.
    - `go test ./internal/provider/hetzner/ -count=1 -race` exits 0 — full Hetzner package tests still green; Plan 06-11 Bug 26 contract preserved (no `unassign:*` calls in any test's expected sequence).
    - `go test ./... -count=1 -race` exits 0 — full repo green (state schema change must not break unrelated packages).
    - `go vet ./...` clean.
    - Commit message follows pattern `fix(06-12): bug 30 — destroy skips DeletePrimaryIP on AutoDelete=true and retries 409 must_be_unassigned for legacy state`.
  </acceptance_criteria>
  <done>
    `runnerkit destroy --yes --cloud hetzner` against a real Hetzner
    project provisioned post-Plan-06-11 exits 0 with `provider_primary_ip:
    status=done` (skipped — auto_delete cascade); the destroy_verify
    binary then polls each saved cloud ID to 404 within
    RUNNERKIT_SMOKE_TIMEOUT. Legacy state with AutoDelete=false retries
    against 409 must_be_unassigned until cascade completes (404) or the
    bounded timeout expires.
  </done>
</task>

</tasks>

<verification>
After all three tasks land:

1. **Repo-wide test green:**
   ```bash
   go test ./... -count=1 -race
   go vet ./...
   gofmt -l internal/cli/down.go internal/cli/up.go internal/provider/hetzner/destroy.go
   ```

2. **All five targeted regression tests green:**
   ```bash
   go test ./internal/cli/ -run "TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper|TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget" -count=1 -race
   go test ./internal/provider/hetzner/ -run "TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade|TestDestroy_RetriesPrimaryIPDeleteOn409MustBeUnassigned|TestIsCascadeInFlightError" -count=1 -race
   ```

3. **Plan 06-11 contract preserved (must NOT regress):**
   ```bash
   go test ./internal/cli/ -run "TestDown_SudoProbeRunsEvenWhenSSHReachableFalse|TestDownThreadsSudoPasswordWhenSudoRequiresPasswordClosesBug21|TestDownDoesNotPromptWhenSudoIsPasswordless" -count=1
   go test ./internal/provider/hetzner/ -run "TestDestroy_AutoDeleteCascadeNoUnassign|TestDestroyTreatsAlreadyAbsentDetachAsSuccess" -count=1
   go test ./internal/remote/ -count=1
   go test ./internal/bootstrap/ -count=1
   ```

4. **Frontmatter `must_haves.artifacts.contains` markers verified present:**
   ```bash
   grep -q "probeSudoNeedsPassword" internal/cli/down.go
   grep -q "TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper" internal/cli/down_test.go
   grep -q "RUNNERKIT_CLOUD_INIT_TIMEOUT" internal/cli/up.go
   grep -q "TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget" internal/cli/up_test.go
   grep -q "isCascadeInFlightError" internal/provider/hetzner/destroy.go
   grep -q "TestDestroy_SkipsDeletePrimaryIPWhenAutoDeleteCascade" internal/provider/hetzner/destroy_test.go
   ```

5. **Live re-smoke (post-plan, maintainer-only — Plan 06-07 attempt-18):**
   - Maintainer follows the post-landing recovery + re-run sequence in
     "Out-of-Plan Maintainer Action" above.
   - Plan 06-07 attempt-18 BYO+cloud smoke completes end-to-end without
     `runner_files: failed`, without `cloud_readiness_failed` at <120s,
     without `provider_primary_ip: pending must_be_unassigned`.
   - 06-VERIFICATION.md baseline filled with real numbers; maintainer
     signs and dates -> `smoke-green` resume signal -> v1.0.0 tag push
     per D-13.
</verification>

<success_criteria>
- All three regression-blocker bugs (28, 29, 30) are closed at code AND test level.
- The pre-Plan-06-12 smoke-output.log root-cause traces no longer reproduce: probe inspects ExitCode + Stderr regardless of err; cloud-init wait carries an explicit ≥120s budget; destroy completes without 409-induced partial.
- All Plan 06-11 (Bugs 24-27) fixes remain intact and verified live (truth #5).
- `go test ./... -count=1 -race` exits 0; `go vet ./...` clean.
- Plan 06-07 attempt-18 unblocked: BYO + cloud smoke green; 06-VERIFICATION.md fillable + signable.
- Phase 6 closes; v1.0.0 tag pushable per D-13.
</success_criteria>

<output>
After completion, create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-12-down-sudo-probe-cloud-init-and-destroy-cascade-fixes-SUMMARY.md` documenting:
- Each bug's root cause, fix shape, and the regression test that locks it in.
- Commit hashes for each task (one atomic commit per task per parallel-executor protocol).
- Verification of all `acceptance_criteria` items.
- Confirmation that Plan 06-11 contracts (Bugs 24-27) remain intact.
- Pre-smoke maintainer action checklist (BYO host install dir recovery + Hetzner empty re-verify + re-run instructions for Plan 06-07 attempt-18).
- Pointer to Plan 06-07 SUMMARY (created after attempt-18 smoke-green) for the final closure.
</output>
</content>
</invoke>