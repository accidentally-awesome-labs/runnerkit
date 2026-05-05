---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 08
subsystem: bootstrap
tags: [byo, sudo, sudoers, register_runner, su, runas, gap-closure, tdd]

# Dependency graph
requires:
  - phase: 06-release-upgrade-docs-and-v1-validation/05
    provides: shellExecutor + buildFakeRunnerTarball + httptest harness in install_integration_test.go (build-tag-guarded)
  - phase: 06-release-upgrade-docs-and-v1-validation/06
    provides: byo-prepare scoped sudoers template (RenderSudoersEntry) — works untouched with the new su form
provides:
  - register_runner step renders `sudo su -s /bin/bash - runnerkit-runner -c "..."` instead of `sudo -u runnerkit-runner ./config.sh ...` so a host with only `(root) NOPASSWD: ALL` (or only the byo-prepare scoped sudoers entry) is sufficient — no `(ALL)` runas required in host sudoers
  - Unit tests asserting absence of `sudo -u runnerkit-runner ./config.sh` and presence of `sudo su -s /bin/bash` in both RenderInstallScript and RenderEphemeralInstallScript
  - Build-tag-guarded integration test (`TestApply_RegisterRunner_RootOnlyNopasswd`) that extracts the register_runner line and asserts the new shell form, closing the test gap that hid Bug 3 from Plans 06-05 + 06-06 verification
  - Smoke harness `.runner` registration sentinel assertion (exit 4 distinct from existing config.sh exit 3) for Plan 06-07 attempt-2 hard pass/fail signal
affects: [06-07 (live-smoke re-run unblocked), v1.0.0 milestone (final BYO blocker closed)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Privilege escalation pattern: when a sudo-context script must drop to a different user, prefer `sudo su -s /bin/bash - <user> -c '<cmd>'` over `sudo -u <user> <cmd>` so the call runs from a root sudo context (works with `(root) NOPASSWD:` scope) instead of requiring `(ALL)` runas in sudoers."
    - "Quote handling for nested shell wrapping: outer `bash -c \"...\"` uses double quotes so the OUTER shell expands `\\\"$ENV_VAR\\\"` to literal value before `su` invokes the inner shell; preserves env-var indirection (no token leak in rendered Script)."

key-files:
  created: []
  modified:
    - internal/bootstrap/script.go (RenderInstallScript + RenderEphemeralInstallScript register_runner line rewritten with su wrap; rationale comment added in both)
    - internal/bootstrap/script_test.go (2 new tests for Bug 3 + 2 existing tests' expected substrings updated for the escaped-quote form)
    - internal/bootstrap/install_integration_test.go (1 new test + `strings` import added)
    - scripts/smoke/byo-permission.sh (.runner sentinel assertion + exit code 4)

key-decisions:
  - "Option 2 (preferred) from gap doc lines 165-176: `sudo su -s /bin/bash - <user> -c \"...\"` chosen over Option 1 (per-repo allowlist regen — defeats one-time prepare promise) and Option 3 (run config.sh as root with HOME override — untested upstream semantics drift)."
  - "Outer `bash -c \"...\"` uses DOUBLE quotes so the OUTER SSH-user shell expands `\\\"$RUNNERKIT_REGISTRATION_TOKEN\\\"` to the literal token string BEFORE `su` invokes the inner shell — keeps the diff minimal vs `sudo --preserve-env=` which adds new env-passthrough surface area."
  - "RenderRemoveConfigScript and RenderReconfigureScript still use `sudo -u %s` — out of scope for Bug 3 per gap doc Task F bounds; recovery flows run after bootstrap completes so byo-prepare or Path B has already established a working sudo path."
  - "Cloud path (internal/provider/hetzner/provision.go cloud-init `(ALL) NOPASSWD: ALL`) intentionally untouched — broader than needed but harmless; tightening to match byo-prepare scope is a non-blocking optional follow-up per gap doc lines 362-365."

patterns-established:
  - "TDD discipline mirroring Plans 06-05 + 06-06: discrete RED commit (failing tests proving the bug exists in the renderer) followed by GREEN commit (production fix turning RED tests green AND surfacing any pre-existing test that still asserts the old shape so test expectations evolve with renderer changes)."
  - "Test gap closure via shape extraction: when an integration test cannot exercise a real call (e.g. real GitHub registration), assert on the rendered shell shape extracted from the renderer output — gives strong negative + positive guarantees without requiring full live infrastructure."
  - "Exit-code differentiation in smoke scripts: distinct numeric exit codes per failure mode (3=config.sh missing, 4=.runner missing) so downstream re-smoke automation can branch on cause without parsing stderr."

requirements-completed: [REL-05]

# Metrics
duration: 5min
completed: 2026-05-05
---

# Phase 06 Plan 08: byo-register-runner-runas-fix Summary

**Replaced `sudo -u runnerkit-runner ./config.sh` with `sudo su -s /bin/bash - runnerkit-runner -c "..."` in both bootstrap renderers to close Bug 3 (register_runner runas mismatch), restoring BYO functionality for v1.0.0 against hosts whose sudoers grants only `(root) NOPASSWD: ALL`.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-05-05T16:24:08Z
- **Completed:** 2026-05-05T16:28:59Z
- **Tasks:** 2 (RED + GREEN, TDD pair)
- **Files modified:** 4 (1 production source, 2 test files, 1 smoke script)

## Accomplishments

- Closed the third and final BLOCKER bug from `06-GAP-byo-sudo-handling.md` (Bug 3 / Task F) discovered during Plan 06-07 attempt-1 re-smoke against `salar@mckee-small-desktop` on 2026-05-05.
- `register_runner` step now works against any sudoers configuration that grants `runas=root` NOPASSWD — including the minimal byo-prepare scoped template — instead of requiring `(ALL)` runas.
- Closed the test gap that hid Bug 3 from Plans 06-05 + 06-06 verification: script_test.go substring assertions now ALSO assert ABSENCE of `sudo -u <non-root>` in register_runner; install_integration_test.go now exercises the rendered register_runner shell form (not only download_runner).
- Smoke harness extended with `.runner` sentinel assertion so Plan 06-07 attempt-2 has a hard pass/fail signal beyond `runnerkit up exited 0`.
- Cloud path verifiably unchanged: `internal/provider/hetzner` package not modified; its tests stay green.

## Task Commits

Each task was committed atomically per the plan's TDD RED/GREEN discipline:

1. **Task 1: RED — failing unit + integration tests for Bug 3** — `2dc17d3` (test)
   - Added `TestRenderInstallScriptUsesSuForRegisterRunner` and `TestRenderEphemeralInstallScriptUsesSuForRegisterRunner` to `internal/bootstrap/script_test.go`.
   - Added `TestApply_RegisterRunner_RootOnlyNopasswd` to `internal/bootstrap/install_integration_test.go` (build-tag `integration` + `RUNNERKIT_INTEGRATION=1` env gate, mirroring `TestApply_DownloadRunner_RealShell`).
   - All three tests FAIL on the unmodified `script.go`:
     - Unit tests fail with `RenderInstallScript missing sudo su -s /bin/bash - runnerkit-runner -c form for register_runner` (because the unmodified Sprintf still emits `sudo -u runnerkit-runner RUNNERKIT_REGISTRATION_TOKEN=... ./config.sh ...`).
     - Integration test fails under `RUNNERKIT_INTEGRATION=1` with `register_runner line still uses sudo -u <non-root> (Bug 3 not closed): "sudo -u salar RUNNERKIT_REGISTRATION_TOKEN=..."`.
   - Without the env var, the integration test SKIPs cleanly (`ok ... (cached)`).

2. **Task 2: GREEN — replace `sudo -u <user> ./config.sh` with `sudo su -s /bin/bash` in both renderers; extend smoke harness** — `895490a` (fix)
   - `internal/bootstrap/script.go` lines 47 + 83 (within `RenderInstallScript` and `RenderEphemeralInstallScript` Sprintf format strings) rewritten to:
     ```
     sudo su -s /bin/bash - %[1]s -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh --unattended --url %[7]s --token \"$RUNNERKIT_REGISTRATION_TOKEN\" --name %[8]s --labels %[9]s --work %[3]s --replace[ --ephemeral]"
     ```
   - Code comment added immediately above the modified line in BOTH renderers explaining the rationale (root-context rationale, byo-prepare-template compatibility, gap-doc back-reference).
   - All three RED tests now PASS.
   - Two existing tests' expected substrings updated for deviation reason below.
   - `scripts/smoke/byo-permission.sh` extended with `.runner` sentinel assertion (exit code 4 distinct from existing config.sh exit 3).

**Plan metadata commit:** Pending (final commit after STATE.md + ROADMAP.md updates).

## Files Created/Modified

- `internal/bootstrap/script.go` — `RenderInstallScript` Sprintf line and `RenderEphemeralInstallScript` Sprintf line rewritten to use `sudo su -s /bin/bash - %[1]s -c "..."` form. Rationale comment block added in both renderers. `RenderRemoveConfigScript` and `RenderReconfigureScript` UNCHANGED (out of scope per Task F bounds).
- `internal/bootstrap/script_test.go` — 2 new test functions added (`TestRenderInstallScriptUsesSuForRegisterRunner`, `TestRenderEphemeralInstallScriptUsesSuForRegisterRunner`); 2 existing tests updated (`TestRenderInstallAndServiceScripts`, `TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken`) to use the escaped-quote form `\\\"$RUNNERKIT_REGISTRATION_TOKEN\\\"` so substring matches survive the renderer wrap. Token-leak invariant assertions preserved.
- `internal/bootstrap/install_integration_test.go` — 1 new test function added (`TestApply_RegisterRunner_RootOnlyNopasswd`); `strings` package added to import block. Re-uses `shellExecutor`, `buildFakeRunnerTarball`, `httptest.NewServer` from the Plan 06-05 harness.
- `scripts/smoke/byo-permission.sh` — `.runner` sentinel assertion appended after the existing `config.sh` assertion. Comment block explains gap doc Task F connection. Distinct exit code 4.

## Decisions Made

- **Option 2 (`sudo su -s /bin/bash - <user> -c "..."`) chosen** — Options 1 (per-repo sudoers allowlist regen) and 3 (run config.sh as root with HOME override) explicitly rejected per gap doc lines 178-187. Option 2 is the minimal-blast-radius fix.
- **Double-quoted outer `bash -c "..."` body** — picked over `sudo --preserve-env=RUNNERKIT_REGISTRATION_TOKEN` because env-passthrough via OUTER-shell variable interpolation matches the existing renderer pattern (line 47 already double-quotes `"$RUNNERKIT_REGISTRATION_TOKEN"`) and keeps the diff minimal. Inside Go raw-string literals (backticks), the `\"` sequences are literal characters bash will interpret as escape sequences.
- **RenderRemoveConfigScript and RenderReconfigureScript NOT modified** — gap doc Task F explicitly bounds the fix to `RenderInstallScript` line 47 and `RenderEphemeralInstallScript` line 83. Recovery flows (`runnerkit recover`) run AFTER bootstrap, by which point byo-prepare has either landed scoped sudoers or Path B has typed a sudo password — those paths still hit `(ALL)` runas semantics but operate in established sudo contexts. Follow-up todo for consistency may be filed.
- **Cloud path intentionally untouched** — Hetzner cloud-init keeps `(ALL) NOPASSWD: ALL` (broader than needed, harmless). Tightening is non-blocking optional follow-up per gap doc lines 362-365.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated existing test expected substrings to reflect the new escaped-quote form**

- **Found during:** Task 2 (GREEN — full bootstrap test run after the renderer change)
- **Issue:** Two existing tests asserted on the substring `./config.sh --unattended --url https://github.com/owner/repo --token "$RUNNERKIT_REGISTRATION_TOKEN"` (with literal `"` chars in the rendered output). After the Bug 3 fix, the rendered output is now wrapped in `sudo su -s /bin/bash - <user> -c "..."` and the inner reference is backslash-escaped (`\"$RUNNERKIT_REGISTRATION_TOKEN\"`) so the OUTER shell expands the env var before `su` invokes the inner shell. The plan's Step 2.3 prediction that the substring would match unchanged was incorrect — the escape sequences differ, so `strings.Contains` fails. Without this fix the bootstrap test suite stays RED and Task 2's GREEN claim cannot be verified.
- **Fix:** Updated the expected substring in `TestRenderInstallAndServiceScripts` (line 20) and `TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken` (line 184) from `--token \"$RUNNERKIT_REGISTRATION_TOKEN\"` (Go-source literal `"`) to `--token \\\"$RUNNERKIT_REGISTRATION_TOKEN\\\"` (Go-source literal `\"`). Added a code comment in both tests explaining the Plan 06-08 rationale so the next reader doesn't re-revert.
- **Files modified:** `internal/bootstrap/script_test.go` (lines 19-29 and lines 180-191 expectations only; test logic + token-leak invariant unchanged).
- **Verification:** Both tests PASS after the update; `go test ./internal/bootstrap/... -count=1` exits 0; full repo `go test ./... -count=1 -race` exits 0 across all 17+ packages.
- **Committed in:** `895490a` (Task 2 GREEN commit — bundled with the renderer change since the test-update + production-change must commit together to avoid a transient broken commit).

---

**Total deviations:** 1 auto-fixed (Rule 3 — blocking issue: existing tests had to reflect the new wrapped form to verify the GREEN claim).
**Impact on plan:** Necessary for correctness; the existing tests now correctly assert that the env-var reference reaches `config.sh` through the `su` wrap with proper shell escaping. No scope creep — the fix is mechanical (substring-update only) and bound to the same two tests the plan acknowledged would be impacted (Step 2.3 lines 437-438 explicitly named these tests).

## Issues Encountered

- **Pre-existing env constraint surfaced (not a regression):** `TestApply_DownloadRunner_RealShell` (Plan 06-05) skips/fails on the local Mac because `sudo` requires a password on this host (the test's own skip comment documents the requirement: "requires NOPASSWD sudo on the test machine"). This is a Plan 06-05 environmental constraint, not a Plan 06-08 regression. The new `TestApply_RegisterRunner_RootOnlyNopasswd` was deliberately designed to be string-shape-only (no real sudo invocation) so it PASSES under `RUNNERKIT_INTEGRATION=1` regardless of host sudoers. On Linux CI with NOPASSWD configured, both tests pass.

## User Setup Required

None — no external service configuration required.

## Verification Results

- `go test ./... -count=1 -race` — PASSES across all 17+ packages including `internal/provider/hetzner` (cloud path verifiably unchanged).
- `grep -cF 'sudo su -s /bin/bash - %[1]s -c' internal/bootstrap/script.go` returns `2` (one each for RenderInstallScript + RenderEphemeralInstallScript).
- `grep -cF 'sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN' internal/bootstrap/script.go` returns `0` (Bug 3 Sprintf pattern gone).
- `grep -cF 'sudo -u runnerkit-runner ./config.sh' internal/bootstrap/script.go` returns `0` (literal-rendered Bug 3 form gone).
- `grep -nF 'sudo -u ' internal/bootstrap/script.go` returns 4 lines: 2 in code comments referencing the OLD pattern (rationale prose) + 2 in `RenderRemoveConfigScript` and `RenderReconfigureScript` (out of scope per Task F bounds). Production grep count for the bug pattern is 0.
- `RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -run TestApply_RegisterRunner_RootOnlyNopasswd` — PASSES.
- `grep -nF '.runner' scripts/smoke/byo-permission.sh` returns matches in the new sentinel-assertion block.
- `git diff HEAD~2 -- internal/provider/hetzner/` is empty — cloud path verifiably unchanged.

## Quote-Handling Note (for future readers)

The rendered script ends up at the rendered shell as:
```
sudo su -s /bin/bash - runnerkit-runner -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh --unattended --url <repo> --token \"$RUNNERKIT_REGISTRATION_TOKEN\" --name <name> --labels <labels> --work <work> --replace"
```

Mechanism:
1. `remote.Command.Env["RUNNERKIT_REGISTRATION_TOKEN"]` injects the token into the OUTER SSH-user shell environment.
2. The OUTER shell expands `\"$RUNNERKIT_REGISTRATION_TOKEN\"` to `"<actual-token>"` (the backslashes are literal in the bash source; bash interprets them as escapes that produce literal `"` chars in the resulting string). The outer `bash -c "..."` body therefore becomes a string containing the actual token wrapped in literal `"` chars.
3. `sudo` runs from the SSH-user shell context with `runas=root` (matched by `(root) NOPASSWD: ALL` or the byo-prepare scoped entry).
4. `su -s /bin/bash - runnerkit-runner -c '<body>'` then drops to the runnerkit-runner user with a clean login shell, executing the body that contains the literal token.
5. `config.sh --unattended --token "<actual-token>"` receives the registration token via argv as expected.

Token-leak invariant preserved: the env-var reference (NOT the literal token) appears in the rendered `Script` string the renderer returns; the literal token is materialized only inside the live SSH session at command-execution time, never stored in state files, logs, JSON output, or error messages.

## Next Phase Readiness

- **Plan 06-07 attempt-2 unblocked.** The maintainer can now re-run `make smoke-live` against `salar@mckee-small-desktop` (the host that exposed Bug 3 on 2026-05-05) WITHOUT manual `/etc/sudoers.d/runnerkit-smoke-temp NOPASSWD ALL` workaround AND WITHOUT modifying byo-prepare. Expected outcome: BYO smoke lands a GitHub runner ID before destroy.
- **06-VERIFICATION.md gap status update:** gaps[1] (Bug 3) status moves from `failed` to `closed`; gaps[0] (BYO bootstrap completes end-to-end) moves from `partial` to `closed` once Plan 06-07 attempt-2 confirms live. gaps[2] (10-minute stopwatch + Hetzner cost + resource IDs) remains the last open item, owned by Plan 06-07 attempt-2.
- **Final v1.0.0 BYO blocker closed.** Per `06-VERIFICATION.md` Verifier Verdict (2026-05-05 update), this was the LAST blocker before v1.0.0 tag push. After Plan 06-07 attempt-2 confirms live, v1.0.0 tag push is unblocked.

## Self-Check: PASSED

Verified post-write:
- `internal/bootstrap/script.go` — sudo su -s /bin/bash - %[1]s -c pattern present 2 times; sudo -u <user> RUNNERKIT_REGISTRATION_TOKEN pattern absent; sudo -u runnerkit-runner ./config.sh pattern absent.
- `internal/bootstrap/script_test.go` — `TestRenderInstallScriptUsesSuForRegisterRunner` and `TestRenderEphemeralInstallScriptUsesSuForRegisterRunner` present; existing tests pass with updated expectations.
- `internal/bootstrap/install_integration_test.go` — `TestApply_RegisterRunner_RootOnlyNopasswd` present; `strings` import added.
- `scripts/smoke/byo-permission.sh` — `.runner` sentinel block present with exit code 4.
- Commits — `2dc17d3` (RED) + `895490a` (GREEN) both present in `git log --oneline -3`.
- Cloud path — `git diff HEAD~2 -- internal/provider/hetzner/` is empty (no changes).

---
*Phase: 06-release-upgrade-docs-and-v1-validation*
*Completed: 2026-05-05*
