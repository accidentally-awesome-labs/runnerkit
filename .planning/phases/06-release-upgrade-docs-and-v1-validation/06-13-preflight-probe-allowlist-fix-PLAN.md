---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 13
type: execute
wave: 1
depends_on: [12]
files_modified:
  - internal/preflight/checks.go
  - internal/preflight/checks_test.go
  - internal/preflight/checks_bugfix_test.go
autonomous: true
gap_closure: true
requirements: [REL-05]
must_haves:
  truths:
    - "Bug 31: `runnerkit byo-prepare --host user@host` followed by `runnerkit up --host user@host --yes --non-interactive` against a host whose sudo requires a password (no NOPASSWD ALL) bootstraps the runner end-to-end with exit 0, NO `sudo_password_required` error, and NO interactive Path B prompt. The preflight probe at `internal/preflight/checks.go:148` no longer runs `sudo -n true` (which is NOT in the byo-prepare scoped allowlist `/usr/bin/apt-get, /usr/bin/dnf, /usr/bin/yum, /usr/sbin/useradd, /usr/bin/install, /bin/tar, /usr/bin/tar, /bin/systemctl, /usr/bin/systemctl, /opt/actions-runner/runnerkit-*/svc.sh`). It runs `sudo -n install --version >/dev/null` instead, which IS in the allowlist (`/usr/bin/install`) and exits 0 on any Path-C-prepared host. Plan 06-07 attempt-19 evidence: `smoke-byo-attempt-19.log` line 3 shows the probe forced Path B's TTY prompt despite a successful prior `byo-prepare`."
    - "Bug 7 contract preserved: when the real SSH executor returns `(probeResult{ExitCode:1, Stderr:\"sudo: a password is required\"}, errors.New(\"ssh: command exited 1\"))` for the new probe command, the classification still emits `host.privilege.password_required` (SeverityWarning, NOT Failure). The stderr classifier at `internal/preflight/checks.go:160-172` (markers `password is required`, `a terminal is required`, `may not run sudo`) is unchanged — only the probe Script literal changes, the result-handling logic stays. `TestCheckPrivilege_PasswordRequired_WhenExecutorReturnsExitErr` and `TestCheckPrivilege_NoSudoers_WhenExecutorReturnsExitErr` (Bug 7 regression tests) stay green."
    - "Bug 8 contract preserved: `runNetworkCheck`'s `curl -sS --connect-timeout 5 --max-time 10` probes (lines 193-194) and the `TestRunNetworkCheck_Script_DoesNotUseFailFlag` regression guard are NOT touched. Plan 06-13 only edits the privilege probe Script literal."
    - "New regression test `TestCheckPrivilege_AllowsScopedSudoers` simulates a host whose ONLY NOPASSWD entry is the byo-prepare scoped allowlist (no broad `(root) NOPASSWD: ALL`). With the new probe command, the fake executor returns `Result{ExitCode:0, Stdout:\"install (GNU coreutils) 9.4\\n\"}` (matching the live evidence in `06-GAP-byo-sudo-handling.md` lines 1465-1467), and `CheckPrivilege` classifies as SeverityPass. This test fails on the pre-fix probe (`sudo -n true` would return ExitCode=1 because `true` is NOT in the allowlist) and passes on the post-fix probe."
    - "`go test ./internal/preflight/... -count=1 -race` exits 0. Existing tests `TestCheckPrivilege_Passwordless`, `TestCheckPrivilege_PasswordRequired`, `TestCheckPrivilege_NotInSudoers`, `TestCheckPrivilege_SudoMissing`, `TestCheckPrivilege_PasswordRequired_WhenExecutorReturnsExitErr`, `TestCheckPrivilege_NoSudoers_WhenExecutorReturnsExitErr`, `TestRunNetworkCheck_Script_DoesNotUseFailFlag`, `TestRunEmitsAllStableCheckIDs`, `TestUnknownLinuxBlocksUnlessAllowed`, `TestNormalizeArch` all stay green. The `probe_sudo_n` Command.ID is unchanged (only the Script literal flips), so all existing fakes that key on `runResults[\"probe_sudo_n\"]` keep working without changes."
    - "`scripts/smoke/byo-permission.sh` runs to completion under `tee` (no TTY) when the maintainer has previously run `runnerkit byo-prepare --host user@host` against the same host. The smoke harness does NOT assert the literal probe command anywhere (verified 2026-05-08: `grep -rn 'sudo -n true' scripts/smoke/` returns 0 matches), so no smoke-script change is required."
    - "Plan 06-07 attempt-20 (post-13 fix) reaches the `BYO_DURATION_SECONDS=NNN` line in `smoke-byo-attempt-20.log` and the cloud `[empty_precheck] OK` marker in `smoke-output.log` without falling back to a manual `/etc/sudoers.d/runnerkit-smoke-temp NOPASSWD ALL` workaround — closing the v1.0.0 release gate (D-13) for REL-05."
  artifacts:
    - path: "internal/preflight/checks.go"
      provides: "CheckPrivilege probe Script changed from `sudo -n true` to `sudo -n install --version >/dev/null`. The stderr-classification switch at lines 157-173 is unchanged: ExitCode==0 → pass; stderr contains `password is required`/`a terminal is required` → SeverityWarning + `host.privilege.password_required`; stderr contains `may not run sudo` → SeverityFailure + `host.privilege.no_sudo`; default → SeverityFailure with stderr surfaced. Command.ID stays `probe_sudo_n` so existing fakes keep working. A code comment cites Bug 31 + Plan 06-13 + cross-references `bootstrap.RenderSudoersEntry` so future maintainers know the Script literal is coupled to the byo-prepare allowlist contract."
      contains: "sudo -n install --version"
    - path: "internal/preflight/checks_test.go"
      provides: "New `TestCheckPrivilege_AllowsScopedSudoers` regression test simulating a Path-C-prepared host: fake executor returns `Result{ExitCode:0, Stdout:\"install (GNU coreutils) 9.4\\n\"}` for `probe_sudo_n`, asserts CheckPrivilege classifies as SeverityPass and report.Passed() is true. Existing tests `TestCheckPrivilege_Passwordless`, `TestCheckPrivilege_PasswordRequired`, `TestCheckPrivilege_NotInSudoers`, `TestCheckPrivilege_SudoMissing` keep their fixtures (they key on Command.ID which is unchanged) but doc-comments are updated to reference the new probe Script for clarity."
      contains: "TestCheckPrivilege_AllowsScopedSudoers"
    - path: "internal/preflight/checks_bugfix_test.go"
      provides: "Doc-comment at lines 13-22 updated to reference the new probe Script (Bug 31 superseded the `sudo -n true` literal). The test bodies for `TestCheckPrivilege_PasswordRequired_WhenExecutorReturnsExitErr` and `TestCheckPrivilege_NoSudoers_WhenExecutorReturnsExitErr` are NOT changed: they key on Command.ID `probe_sudo_n` and on the stderr marker, both of which are stable across this plan. `TestRunNetworkCheck_Script_DoesNotUseFailFlag` is NOT touched (Bug 8 contract is independent)."
      contains: "Bug 31"
  key_links:
    - from: "internal/preflight/checks.go::Run probe at line 148"
      to: "internal/bootstrap/sudoers.go::RenderSudoersEntry allowlist `/usr/bin/install`"
      via: "probe Script `sudo -n install --version >/dev/null` matches an entry inside the scoped sudoers allowlist that byo-prepare installs at /etc/sudoers.d/runnerkit-installer"
      pattern: "install --version"
    - from: "Plan 06-07 attempt-20 BYO smoke (Path C)"
      to: "v1.0.0 tag push per D-13"
      via: "post-byo-prepare `runnerkit up --yes --non-interactive` exits 0 on a password-protected-sudo host because the preflight probe now passes inside the scoped allowlist; `BYO_DURATION_SECONDS=NNN` is recorded in 06-VERIFICATION.md; SMOKE-GREEN signal unblocks v1.0.0"
      pattern: "BYO_DURATION_SECONDS"

tasks:
  - id: bug-31-failing-test-allows-scoped-sudoers
    name: "preflight: add failing test TestCheckPrivilege_AllowsScopedSudoers (Bug 31 RED)"
    autonomous: true
  - id: bug-31-rewrite-probe-to-allowlisted-command
    name: "preflight: replace `sudo -n true` probe with `sudo -n install --version >/dev/null` (Bug 31 GREEN)"
    autonomous: true
---

# Plan 06-13: Preflight Probe Allowlist Fix

## Context

Plan 06-07 attempt-19 (2026-05-08, see `smoke-byo-attempt-19.log`) re-ran
the BYO smoke against `salar@mckee-small-desktop` from a non-TTY
automation context AFTER Plan 06-12 (Bugs 28-30) landed. Path C
(`runnerkit byo-prepare`) had already installed the scoped sudoers
allowlist, but `runnerkit up --yes --non-interactive` immediately fell
through to Path B's TTY prompt and aborted because no terminal was
attached. Smoke evidence (3 lines, full file `smoke-byo-attempt-19.log`):

```
===> [smoke-byo] Setting up isolated state dir
===> [smoke-byo] runnerkit up --repo accidentally-awesome-labs/dat0 --host salar@mckee-small-desktop --mode persistent --yes
Sudo password for salar@mckee-small-desktop:22:
```

That third line is `promptSudoPasswordForPathB` (`internal/cli/up.go:2108`)
firing because `internal/preflight/checks.go::Run` emitted the
`host.privilege.password_required` warning, which Path B consumes as its
trigger.

**Root cause** (per `06-GAP-byo-sudo-handling.md` Bug 31 lines 1433-1538):

`internal/preflight/checks.go:148` probes:

```go
probeResult, probeErr := executor.Run(ctx, target, remote.Command{
    ID:     "probe_sudo_n",
    Script: "sudo -n true",
})
```

`true` is NOT in byo-prepare's scoped allowlist (per
`internal/bootstrap/sudoers.go::RenderSudoersEntry`, the allowlist is
`/usr/bin/apt-get, /usr/bin/dnf, /usr/bin/yum, /usr/sbin/useradd,
/usr/bin/install, /bin/tar, /usr/bin/tar, /bin/systemctl,
/usr/bin/systemctl, /opt/actions-runner/runnerkit-*/svc.sh`). So even on
a host that has been fully prepared by Path C, `sudo -n true` exits 1
with `sudo: a password is required` and the preflight emits
`host.privilege.password_required` unconditionally. The remediation
("Run `runnerkit byo-prepare`...") is then wrong: the user already did.

Live evidence the fix is correct (from the GAP file lines 1457-1472):

```
$ ssh salar@mckee-small-desktop 'sudo -n true; echo exit=$?'
sudo: a password is required
exit=1

$ ssh salar@mckee-small-desktop 'sudo -n install --version 2>&1 | head -1; echo exit=$?'
install (GNU coreutils) 9.4
exit=0
```

`install` IS in the allowlist (`/usr/bin/install`) and is also a
RunnerKit-required tool (`internal/preflight/checks.go::RequiredTools`
line 230 lists `install`), so it is guaranteed present on any host that
otherwise passes preflight. `sudo -n install --version >/dev/null` is
the smallest, safest, most-portable probe that exercises the
scoped-sudoers contract end-to-end.

## Bug Summary

| Bug | Description | Surface | Detected | Severity |
|-----|-------------|---------|----------|----------|
| 31 | Preflight `sudo -n true` probe is not in byo-prepare scoped allowlist; Path C never bypasses Path B's TTY prompt; v1.0.0 BYO smoke broken in non-TTY automation | `runnerkit up` preflight | Plan 06-07 attempt-19 BYO smoke 2026-05-08 | BLOCKER |

## Wave / Dependency Convention

This plan uses `wave: 1` with `depends_on: [12]` — matching the phase-6
gap-closure convention (06-10 deps [09], 06-11 deps [10], 06-12 deps
[11], 06-13 deps [12]). Each gap closure plan is filed AFTER the prior
plan's smoke attempt fails, so they are inherently serial.

## Approach

**Option A (recommended in the GAP file lines 1511-1538) — replace the
probe Script with an allowlisted command:**

- Change `internal/preflight/checks.go:148` Script literal from
  `"sudo -n true"` to `"sudo -n install --version >/dev/null"`.
- Keep `Command.ID = "probe_sudo_n"` UNCHANGED — every existing fake
  test (in `checks_test.go`, `checks_bugfix_test.go`, `up_test.go`)
  keys on this ID, so changing it would force a sweep of unrelated
  fixtures. The Script change is silent at the test fixture layer.
- Keep the entire stderr-classification switch at lines 157-173
  UNCHANGED. The Bug 7 fix (which made the switch tolerant of
  `*exec.ExitError` from the real SSH executor) and the marker
  substring matches (`password is required` / `a terminal is required` /
  `may not run sudo`) are independent of WHICH command ran. They
  classify based on stderr content only, so a host that returns
  `sudo: a password is required` for the new probe lands in the same
  warning branch as before — Path B is still reachable when truly
  needed.

**Why NOT Option B (file-existence detection):** Coupling preflight to
byo-prepare's exact sudoers content would force a recompile of preflight
whenever the allowlist changes. Option A validates the actual
scoped-sudoers contract end-to-end via a real `sudo -n` probe,
regardless of which file granted the access. (The user could have
hand-rolled a different sudoers entry that grants `install` NOPASSWD —
preflight still passes correctly.)

**Why `install --version` (not `systemctl --version` or `apt-get --version`):**

- `install` is in the byo-prepare allowlist as `/usr/bin/install` (the
  same path GNU coreutils ships at on Ubuntu/Debian/Fedora/Arch).
- `install` is in `RequiredTools()` (line 230) so it's already a
  preflight-checked invariant — preflight short-circuits to a tools
  failure if it's missing, before ever reaching the privilege probe.
- `install --version` is a no-op that exits 0 with a single-line stdout.
  Redirecting to /dev/null keeps the probe quiet.
- `systemctl --version` would also work but is bound to a systemd
  initsystem; `install --version` works on any Linux host with
  coreutils, leaving preflight robust against future support of
  non-systemd hosts.
- `apt-get --version` is Debian-family-only.

**TDD cadence (matches Plan 06-12 style — RED then GREEN, two atomic
commits):**

- Task 1: write `TestCheckPrivilege_AllowsScopedSudoers` against the
  CURRENT (pre-fix) code. The test simulates a Path-C-prepared host by
  setting `runResults["probe_sudo_n"] = Result{ExitCode:0, Stdout:"install (GNU coreutils) 9.4\n"}`
  and asserts CheckPrivilege classifies as SeverityPass. On the
  pre-fix code this passes (because the fake doesn't care what the
  Script literal is — it keys on Command.ID). So the test alone is
  not sufficient as a RED. To create a meaningful RED, the test ALSO
  asserts that `internal/preflight/checks.go` source contains the
  literal `sudo -n install --version` (proving the Script flipped) and
  does NOT contain the literal `Script: "sudo -n true"`. This is the
  RED gate: the source-code substring assertion fails on the pre-fix
  code, then passes after Task 2.
- Task 2: edit `internal/preflight/checks.go` to flip the probe Script
  literal. Run the suite — RED test now goes GREEN.

This is the same cadence as Plan 06-12 Task 1 (`TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper`):
land the test in a `test(06-13): ...` commit that documents the
behaviour, then land the fix in a `fix(06-13): ...` commit.

## Out of Scope

- `internal/cli/byo_prepare.go:133` post-install `sudo -n true` verify
  probe — this emits a "Note: post-install probe did not pass; this is
  expected" message after byo-prepare runs. Switching this probe to the
  new allowlisted command would turn the Note into a positive
  verification, but it's purely cosmetic and the user-facing "Note" was
  intentional per Plan 06-06 SUMMARY line 153 ("informational, not a
  warning"). Out-of-scope per gap_closure_scope rules. Leave as-is.
- `internal/cli/down.go:230, 423, 450` `sudo -n true` references —
  these are the down-cleanup probe (Plan 06-10 Bug 21), independent
  surface, not Bug 31. Out-of-scope.
- `internal/remote/system.go:40` `sudo -n true` reference — this is
  the SystemExecutor's command-precheck path, unrelated to preflight
  privilege classification. Out-of-scope.
- `internal/cli/up.go:2108` `promptSudoPasswordForPathB` — Path B
  prompt logic is correct as-is per Plan 06-06; only the upstream
  preflight classification needs fixing so Path B isn't entered
  unnecessarily. Out-of-scope.
- byo-prepare's scoped sudoers content (`internal/bootstrap/sudoers.go::RenderSudoersEntry`)
  — DO NOT add `true` or expand the allowlist. The probe must move to
  a command already inside it. Out-of-scope.
- New error code for Bug 31 — existing `host.privilege.password_required`
  emit-site (Plan 06-05) and RKD-BOOT-015 docs anchor still cover the
  password-required path. No `internal/errcodes/codes.go` change.
- Any architectural follow-up (SEED-001 bootstrap/lifecycle split) —
  deferred to v1.1+ per project memory `project_v1_release_posture.md`.
  Plan 06-13 is the cheap v1.0.0 unblock.

## Out-of-Plan Maintainer Action (post-landing)

After Plan 06-13 lands, before re-running `make smoke-live`:

1. **Re-run byo-prepare on the smoke host** (the existing scoped
   allowlist already contains `/usr/bin/install`, so this step is a
   no-op idempotency check — but explicit:

   ```bash
   go run ./cmd/runnerkit byo-prepare --host salar@mckee-small-desktop
   ```

   Confirm output `Host salar@mckee-small-desktop:22 is now prepared.`
   (no install required if content matches; `runnerkit byo-prepare`
   short-circuits via `bootstrap.SudoersIsPrepared`).

2. **Re-verify the probe works manually:**

   ```bash
   ssh salar@mckee-small-desktop 'sudo -n install --version >/dev/null; echo exit=$?'
   ```

   Must print `exit=0`.

3. **Re-run** `make smoke-live` per Plan 06-07 sequence (or run
   `scripts/smoke/byo-permission.sh` directly under `tee` to confirm
   non-TTY behavior). Expected outcomes:
   - BYO smoke: `runnerkit up --yes --non-interactive` proceeds past
     preflight without `sudo_password_required`, no Path B prompt,
     bootstrap completes, runner registers, exit 0. `BYO_DURATION_SECONDS=NNN`
     marker is captured.
   - Cloud smoke: unaffected by Plan 06-13 (cloud path uses cloud-init
     to install `(ALL) NOPASSWD: ALL` — preflight always passes).

4. **Resume signal:** signal SMOKE-GREEN to close Plan 06-07; v1.0.0
   tag push per D-13.

## Verification

- Full repo `go test ./... -count=1 -race` passes.
- New unit test:
  - `internal/preflight/checks_test.go::TestCheckPrivilege_AllowsScopedSudoers`
- Existing tests unchanged (must stay green):
  - `internal/preflight/checks_test.go::TestCheckPrivilege_Passwordless`
  - `internal/preflight/checks_test.go::TestCheckPrivilege_PasswordRequired`
  - `internal/preflight/checks_test.go::TestCheckPrivilege_NotInSudoers`
  - `internal/preflight/checks_test.go::TestCheckPrivilege_SudoMissing`
  - `internal/preflight/checks_test.go::TestRunEmitsAllStableCheckIDs`
  - `internal/preflight/checks_test.go::TestUnknownLinuxBlocksUnlessAllowed`
  - `internal/preflight/checks_test.go::TestNormalizeArch`
  - `internal/preflight/checks_bugfix_test.go::TestCheckPrivilege_PasswordRequired_WhenExecutorReturnsExitErr` (Bug 7)
  - `internal/preflight/checks_bugfix_test.go::TestCheckPrivilege_NoSudoers_WhenExecutorReturnsExitErr` (Bug 7)
  - `internal/preflight/checks_bugfix_test.go::TestRunNetworkCheck_Script_DoesNotUseFailFlag` (Bug 8)
  - `internal/cli/up_test.go::TestUp_SudoPasswordPrompt_Interactive` (Path B Plan 06-06)
- Plan 06-07 attempt-20 BYO smoke completes end-to-end under `tee` (no
  TTY); `06-VERIFICATION.md` becomes fillable + signable.

<objective>
Close Bug 31 surfaced by Plan 06-07 attempt-19 live BYO smoke
(2026-05-08, `smoke-byo-attempt-19.log`) so the v1.0.0 maintainer smoke
can complete end-to-end on attempt-20+ from a non-TTY automation
context (Bash tool, CI, `tee`-piped invocation):

- Bug 31: preflight `sudo -n true` probe is replaced with
  `sudo -n install --version >/dev/null` so Path C
  (`runnerkit byo-prepare`)'s scoped allowlist actually bypasses Path
  B's TTY prompt. The probe command is bound to byo-prepare's
  `RenderSudoersEntry` allowlist contract via a regression test that
  simulates a Path-C-prepared host returning ExitCode=0 for the new
  probe.

Purpose: Plan 06-07 attempt-20 produces SMOKE-GREEN; 06-VERIFICATION.md
gets filled with real durations + 5 cloud resource IDs + EUR cost +
maintainer signature; v1.0.0 tag is pushable per D-13. Closes the
re-verification cycle Plan 06-12 -> attempt-19 SMOKE-RED -> Plan 06-13.

Output: 3 file modifications committed across 2 atomic commits
(test → fix); full test suite green; ready for `/gsd:execute-phase 06
--gaps-only` re-run of Plan 06-07.
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
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-12-down-sudo-probe-cloud-init-and-destroy-cascade-fixes-PLAN.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-12-down-sudo-probe-cloud-init-and-destroy-cascade-fixes-SUMMARY.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-07-live-smoke-rerun-and-baseline-fillin-PLAN.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/smoke-byo-attempt-19.log
@internal/preflight/checks.go
@internal/preflight/checks_test.go
@internal/preflight/checks_bugfix_test.go
@internal/bootstrap/sudoers.go
@internal/cli/up.go
@internal/cli/byo_prepare.go
@scripts/smoke/byo-permission.sh

<interfaces>
<!-- Existing contracts the executor uses directly. -->

internal/preflight/checks.go (Bug 31 target — only line 148 Script literal flips):
- `Run(ctx context.Context, executor remote.Executor, target remote.Target, options Options) (Report, error)`
  at line 94. The privilege probe at line 148:
    probeResult, probeErr := executor.Run(ctx, target, remote.Command{
        ID:     "probe_sudo_n",
        Script: "sudo -n true",   // <-- THE LITERAL TO CHANGE
    })
  Switch at lines 157-173 classifies based on probeErr/ExitCode/Stderr;
  this whole switch is UNCHANGED. The Bug 7 fix at lines 149-156 (the
  comment block + the err-tolerant pattern) is UNCHANGED. Only the
  `Script:` value flips to `"sudo -n install --version >/dev/null"`.
- `Command.ID` MUST stay `"probe_sudo_n"` so existing fakes that key on
  it (checks_test.go lines 93/122/153, checks_bugfix_test.go lines
  58/61/88/91, up_test.go line 449) keep working without churn.

internal/bootstrap/sudoers.go (allowlist contract — read-only reference):
- `RenderSudoersEntry(user string) string` at line 49 produces:
    %s ALL=(root) NOPASSWD: \
      /usr/bin/apt-get, /usr/bin/dnf, /usr/bin/yum, \
      /usr/sbin/useradd, \
      /usr/bin/install, \    <-- THE PROBE'S TARGET
      /bin/tar, /usr/bin/tar, \
      /bin/systemctl, /usr/bin/systemctl, \
      /opt/actions-runner/runnerkit-*/svc.sh
  When sudoers lists a command without explicit args, sudo allows ANY
  args to that absolute-path command. So `sudo -n install --version`
  matches the `/usr/bin/install` entry as long as install resolves to
  `/usr/bin/install` (which it does on Ubuntu/Debian/Fedora/Arch — the
  GNU coreutils default).
- DO NOT modify this file. The probe must move to an entry already in
  the list, not the other way around.

internal/preflight/checks_test.go (existing fakes — keep them):
- `fakePreflightExecutor` at line 11 keys on `command.ID`. New test
  TestCheckPrivilege_AllowsScopedSudoers uses the same fake with
  `runResults: map[string]remote.Result{"probe_sudo_n": {ExitCode: 0,
  Stdout: "install (GNU coreutils) 9.4\n"}}`.
- `passingProbe(osID, arch)` at line 82 already includes
  `Commands["install"] = true` so RequiredTools doesn't short-circuit
  before the probe runs.

internal/preflight/checks_bugfix_test.go (Bug 7 + Bug 8 — preserve):
- Bug 7 tests at lines 53-102 key on Command.ID `probe_sudo_n` and
  Stderr markers; both are stable across this plan. Test bodies do NOT
  change. Doc comment at lines 13-22 references `sudo -n true`
  literally — update the comment to reference the new probe Script
  (Bug 31 superseded the literal) for future-archaeology clarity.
- `TestRunNetworkCheck_Script_DoesNotUseFailFlag` at line 104 is
  Bug 8's regression guard, unrelated to Bug 31. NOT touched.

scripts/smoke/byo-permission.sh (smoke harness — verified, no change):
- `grep -n 'sudo -n true' scripts/smoke/byo-permission.sh` returns 0
  matches. No literal probe assertion exists in the smoke script. NOT
  touched by this plan.

Plan 06-12 cross-context (preserve, do not regress):
- Plan 06-12 Bugs 28-30 (down probe, cloud-init timeout, destroy
  cascade) are VERIFIED in code; Bug 31 is a different surface
  (up preflight, not down/destroy/up-cloud). No cross-test
  interactions.
- Plan 06-12's commit cadence (`test(06-12): ...` then `fix(06-12): ...`
  per task) is the model for Plan 06-13's cadence
  (`test(06-13): ...` then `fix(06-13): ...`).

Plan 06-06 cross-context (preserve, do not regress):
- Path B fallback (`promptSudoPasswordForPathB` at up.go:2108) is
  correct as-is. Plan 06-13 only fixes the UPSTREAM preflight
  classification so Path B isn't entered unnecessarily on Path-C
  hosts. When Path B IS legitimately needed (no NOPASSWD, no
  byo-prepare), the new probe still returns the same
  `host.privilege.password_required` warning and Path B still fires.
  TestUp_SudoPasswordPrompt_Interactive (up_test.go:458) stays green.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Bug 31 (RED) — add failing test TestCheckPrivilege_AllowsScopedSudoers asserting the probe Script binds to byo-prepare's allowlist</name>
  <files>internal/preflight/checks_test.go</files>
  <read_first>
    - internal/preflight/checks_test.go (existing TestCheckPrivilege_Passwordless / TestCheckPrivilege_PasswordRequired / TestCheckPrivilege_NotInSudoers / TestCheckPrivilege_SudoMissing — mirror their fakePreflightExecutor wiring; passingProbe helper at line 82)
    - internal/preflight/checks.go (target — line 148 probe Script literal; RequiredTools at line 230 confirms `install` is a required tool already)
    - internal/preflight/checks_bugfix_test.go (Bug 7 doc comment lines 13-22 — references the OLD probe literal; needs a parallel update in Task 2)
    - internal/bootstrap/sudoers.go::RenderSudoersEntry (lines 49-59 — the byo-prepare allowlist contract this test pins; `/usr/bin/install` is the target entry)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md lines 1433-1554 (Bug 31 evidence: `sudo -n install --version` exits 0 on the live smoke host inside the byo-prepare allowlist; lines 1465-1467 show the literal `install (GNU coreutils) 9.4` stdout)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/smoke-byo-attempt-19.log (3-line live evidence Path B fired despite Path C prep)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-12-down-sudo-probe-cloud-init-and-destroy-cascade-fixes-PLAN.md Task 1 (style exemplar — the `<read_first>` / `<behavior>` / `<action>` / `<verify>` / `<acceptance_criteria>` / `<done>` shape; commit message form `test(06-12): bug 28 — ...`)
  </read_first>
  <behavior>
    Test cases for the probe Script binding (TestCheckPrivilege_AllowsScopedSudoers):

    - Test (NEW — TestCheckPrivilege_AllowsScopedSudoers, two sub-assertions in one Test func):
      Sub-assertion 1 (BEHAVIOR — passes pre+post fix because fake keys on Command.ID):
        passingProbe("ubuntu", "x86_64") + fakePreflightExecutor with
        runResults["probe_sudo_n"] = remote.Result{ExitCode: 0, Stdout: "install (GNU coreutils) 9.4\n"}
        → CheckPrivilege classifies as SeverityPass; report.Passed() == true; result.Message contains "passwordless sudo".
      Sub-assertion 2 (SOURCE-CODE BIND — fails pre-fix, passes post-fix; this is the RED gate):
        os.ReadFile("checks.go") → string contents → assert:
          - contents contains the literal "sudo -n install --version" (proving the new probe Script is present)
          - contents does NOT contain the literal `Script: "sudo -n true"` (proving the OLD literal was removed; `Script:` prefix avoids false matches against doc-comment occurrences of `sudo -n true` elsewhere in the file)
        Helper `readChecksGoSource()` already exists in checks_bugfix_test.go (line 133) — reuse it.
      Failure messages must reference Bug 31 + Plan 06-13 + the GAP file path so future archaeologists find the trail.

    Existing tests (must stay green; NOT modified, only mentioned for context):
    - TestCheckPrivilege_Passwordless (ExitCode=0 → SeverityPass) — unaffected; fake keys on Command.ID
    - TestCheckPrivilege_PasswordRequired (ExitCode=1 + "password is required" → SeverityWarning) — unaffected
    - TestCheckPrivilege_NotInSudoers (ExitCode=1 + "may not run sudo" → SeverityFailure) — unaffected
    - TestCheckPrivilege_SudoMissing (probe.Commands["sudo"]=false → bypass probe, SeverityFailure) — unaffected
  </behavior>
  <action>
    Add a new test function `TestCheckPrivilege_AllowsScopedSudoers` to `internal/preflight/checks_test.go`. Place it AFTER `TestCheckPrivilege_SudoMissing` (after line 197) so the file ordering is: NormalizeArch → UnknownLinuxBlocks → RunEmitsAllStableCheckIDs → passingProbe helper → CheckPrivilege_Passwordless → CheckPrivilege_PasswordRequired → CheckPrivilege_NotInSudoers → CheckPrivilege_SudoMissing → CheckPrivilege_AllowsScopedSudoers (new).

    The test body — VERBATIM (place at end of file, before any trailing helpers):

    ```go
    // TestCheckPrivilege_AllowsScopedSudoers asserts that the probe at
    // internal/preflight/checks.go:148 uses a Script literal that is
    // present in byo-prepare's scoped sudoers allowlist (per
    // internal/bootstrap/sudoers.go::RenderSudoersEntry). Bug 31
    // (Plan 06-13, 2026-05-08): the prior literal `sudo -n true` was
    // NOT in the allowlist, so a Path-C-prepared host (byo-prepare ran
    // successfully) still fell through to Path B's TTY prompt during
    // `runnerkit up`. The fix swaps the Script to
    // `sudo -n install --version >/dev/null` because /usr/bin/install
    // IS in the byo-prepare allowlist (and is also a RequiredTools
    // member, so it is guaranteed present on hosts that pass earlier
    // preflight steps).
    //
    // Test has two assertions:
    //   1. Behavioral: a fake executor returning ExitCode=0 + the
    //      live-smoke-confirmed install --version stdout classifies as
    //      SeverityPass (passwordless sudo). (See gap doc Bug 31
    //      lines 1465-1467 for the exact `install (GNU coreutils)
    //      9.4` evidence.) This branch alone passes pre+post fix
    //      because the fake keys on Command.ID rather than Script.
    //   2. Source-code binding: the checks.go source contains the new
    //      literal `sudo -n install --version` AND does NOT contain
    //      the old literal `Script: "sudo -n true"`. This is the RED
    //      gate that fails on the pre-fix code and passes after Task 2.
    //
    // See: .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md
    // (Bug 31 lines 1433-1554) and Plan 06-13.
    func TestCheckPrivilege_AllowsScopedSudoers(t *testing.T) {
        // Sub-assertion 1: behavioral (independent of Script literal)
        probe := passingProbe("ubuntu", "x86_64")
        exec := fakePreflightExecutor{probe: probe, runResults: map[string]remote.Result{
            "probe_sudo_n": {ExitCode: 0, Stdout: "install (GNU coreutils) 9.4\n"},
        }}
        target := remote.Target{User: "salar", Host: "mckee-small-desktop", Port: 22}
        report, err := Run(context.Background(), exec, target, Options{})
        if err != nil {
            t.Fatalf("Run returned error: %v", err)
        }
        result, ok := report.Result(CheckPrivilege)
        if !ok {
            t.Fatalf("report missing %q result (Bug 31 / Plan 06-13): %#v", CheckPrivilege, report.Results)
        }
        if result.Severity != SeverityPass {
            t.Fatalf("Path-C-prepared host probe should classify as SeverityPass; got %q (Bug 31 / Plan 06-13)", result.Severity)
        }
        if !report.Passed() {
            t.Fatalf("report.Passed() should be true on Path-C-prepared host (Bug 31 / Plan 06-13): %#v", report.Results)
        }
        if !strings.Contains(strings.ToLower(result.Message), "passwordless sudo") {
            t.Fatalf("message should mention passwordless sudo: %q (Bug 31 / Plan 06-13)", result.Message)
        }

        // Sub-assertion 2: source-code binding to byo-prepare allowlist
        // (RED gate — fails pre-fix because checks.go still has
        // `Script: "sudo -n true"`).
        src, srcErr := readChecksGoSource()
        if srcErr != nil {
            t.Fatalf("read checks.go source: %v", srcErr)
        }
        if !strings.Contains(src, "sudo -n install --version") {
            t.Fatalf("checks.go missing new probe literal `sudo -n install --version` — Bug 31 (Plan 06-13) requires the privilege probe to use a command in byo-prepare's scoped allowlist. See .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md Bug 31.")
        }
        if strings.Contains(src, `Script: "sudo -n true"`) {
            t.Fatalf("checks.go still uses old probe literal `Script: \"sudo -n true\"` — Bug 31 (Plan 06-13) replaced this with `sudo -n install --version >/dev/null` because `true` is NOT in byo-prepare's scoped sudoers allowlist (see internal/bootstrap/sudoers.go::RenderSudoersEntry).")
        }
    }
    ```

    `readChecksGoSource()` is already defined in `checks_bugfix_test.go` line 133 (same package), so the new test can call it directly without redefinition.

    Imports already present in `checks_test.go`: `context`, `strings`, `testing`, and the `remote` package. No new imports needed for this test.

    DO NOT modify any other tests in `checks_test.go`. DO NOT modify
    `passingProbe`. DO NOT modify `fakePreflightExecutor`. DO NOT
    touch `checks_bugfix_test.go` in this commit (its doc comment
    update lands in Task 2 alongside the source change).

    Run the suite to confirm RED: the source-code-binding sub-assertion
    must fail because `checks.go` still contains `Script: "sudo -n true"`
    and does NOT contain `sudo -n install --version`.

    Commit the test ALONE in a single commit (matches Plan 06-12 TDD
    cadence — failing test commit precedes fix commit):
    ```
    test(06-13): bug 31 — add TestCheckPrivilege_AllowsScopedSudoers binding probe Script to byo-prepare allowlist
    ```

    Reference: Plan 06-12 Task 1's commit
    `test(06-12): bug 28 — TestDown_ProbeUsesExitCodeWhenExecutorReturnsExitErrorWrapper` is the style exemplar.
  </action>
  <verify>
    <automated>cd /Users/salar/Projects/spool && go test ./internal/preflight/ -run TestCheckPrivilege_AllowsScopedSudoers -count=1 -v 2>&1 | tee /tmp/06-13-task-1-verify.txt && grep -q "FAIL" /tmp/06-13-task-1-verify.txt && echo "RED CONFIRMED (test fails on pre-fix code as required by TDD cadence)" || (echo "UNEXPECTED PASS — test must fail before Task 2 lands" && exit 1)</automated>
  </verify>
  <acceptance_criteria>
    - `internal/preflight/checks_test.go` contains the literal `func TestCheckPrivilege_AllowsScopedSudoers(t *testing.T)`. Verify via `grep -c 'func TestCheckPrivilege_AllowsScopedSudoers' internal/preflight/checks_test.go` returns 1.
    - `internal/preflight/checks_test.go` test body contains all three required substrings: `install (GNU coreutils) 9.4`, `sudo -n install --version`, and `Script: "sudo -n true"`. Verify via three separate `grep -F` calls inside the test function body.
    - `go test ./internal/preflight/ -run TestCheckPrivilege_AllowsScopedSudoers -count=1` exits NON-ZERO (RED gate — test fails on pre-fix code; this is the TDD invariant). Specifically the failure message must reference `sudo -n install --version` (the missing-literal sub-assertion) — verify via `go test ... 2>&1 | grep -F 'sudo -n install --version'`.
    - All other tests in `internal/preflight/` still pass: `go test ./internal/preflight/ -run '^TestCheckPrivilege_(Passwordless|PasswordRequired|NotInSudoers|SudoMissing|PasswordRequired_WhenExecutorReturnsExitErr|NoSudoers_WhenExecutorReturnsExitErr)$|^TestRunEmitsAllStableCheckIDs$|^TestUnknownLinuxBlocksUnlessAllowed$|^TestNormalizeArch$|^TestRunNetworkCheck_Script_DoesNotUseFailFlag$' -count=1` exits 0.
    - `go vet ./internal/preflight/...` clean.
    - `gofmt -l internal/preflight/checks_test.go` empty (no formatting issues).
    - Commit message follows pattern `test(06-13): bug 31 — add TestCheckPrivilege_AllowsScopedSudoers binding probe Script to byo-prepare allowlist`. Verify via `git log -1 --pretty=format:%s` after commit.
    - The single commit modifies ONLY `internal/preflight/checks_test.go` (no other files changed in the same commit). Verify via `git show --stat HEAD` showing exactly 1 file.
  </acceptance_criteria>
  <done>
    A failing regression test exists that simultaneously (a) documents
    the expected behavior (Path-C-prepared host classifies as
    passwordless sudo) and (b) source-code-binds the probe Script to
    byo-prepare's scoped allowlist. The test fails RED on the current
    code, providing the safety rail Task 2 will turn GREEN. Future
    refactors that revert the probe Script to a non-allowlisted command
    will trip this test before reaching CI.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Bug 31 (GREEN) — replace probe Script `sudo -n true` with `sudo -n install --version >/dev/null` and update Bug 7 doc comment</name>
  <files>internal/preflight/checks.go, internal/preflight/checks_bugfix_test.go</files>
  <read_first>
    - internal/preflight/checks.go (target — lines 140-174; ONLY the Script literal at line 148 changes; the comment block at 142-156 gains a Bug 31 reference; the switch at 157-173 is UNCHANGED)
    - internal/preflight/checks_test.go (after Task 1 lands — TestCheckPrivilege_AllowsScopedSudoers must turn GREEN with this commit)
    - internal/preflight/checks_bugfix_test.go (lines 13-22 doc comment references the old probe literal; update parenthetical to also cite Bug 31)
    - internal/bootstrap/sudoers.go::RenderSudoersEntry (lines 49-59 — the contract the new Script binds to; `/usr/bin/install` entry at line 54)
    - internal/cli/byo_prepare.go line 133 (the verify_sudo_n probe — DO NOT touch in this plan; out-of-scope per the gap_closure_scope. The user-facing Note message at line 135 stays.)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md Bug 31 lines 1509-1538 (Option A spec — the recommended fix)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-12-down-sudo-probe-cloud-init-and-destroy-cascade-fixes-PLAN.md Task 1 action block (commit-message style exemplar — `fix(06-12): bug 28 — ...`)
  </read_first>
  <behavior>
    Behavioral contract after this commit:

    - `internal/preflight/checks.go` line 148 Script value is `"sudo -n install --version >/dev/null"` (NOT `"sudo -n true"`).
    - `Command.ID` stays `"probe_sudo_n"` (no change). All existing fakes that key on this ID keep working without churn (verified by the existing test list staying green).
    - The stderr-classification switch at lines 157-173 is byte-identical to pre-fix (no logic change).
    - The comment block at lines 142-156 cites Bug 31 + Plan 06-13 explaining WHY the Script literal was changed (binding to byo-prepare's allowlist) so future archaeologists understand the coupling.
    - `internal/preflight/checks_bugfix_test.go` doc comment at lines 13-22 is updated: the parenthetical example `(e.g. \`sudo -n true\` exit 1 with stderr ...)` becomes `(e.g. \`sudo -n install --version\` exit 1 with stderr ...)` so future maintainers reading the Bug 7 explanation see the current probe literal, not the historical one. The test bodies are NOT changed.

    All previously-failing assertions in TestCheckPrivilege_AllowsScopedSudoers
    (introduced by Task 1) now pass:
    - `strings.Contains(src, "sudo -n install --version")` → true
    - `!strings.Contains(src, \`Script: "sudo -n true"\`)` → true

    All previously-passing tests stay passing.
  </behavior>
  <action>
    Step 1 — edit `internal/preflight/checks.go` lines 142-156 + 148.

    Replace the existing comment block + probe call (lines 143-156):

    ```go
        // Probe whether the SSH user can run sudo non-interactively. The
        // bootstrap path runs over a non-interactive channel and cannot
        // answer a sudo password prompt, so a host with sudo-binary present
        // but a password requirement must NOT pass preflight as if it were
        // passwordless. See gap doc 06-GAP-byo-sudo-handling.md Task A.
        probeResult, probeErr := executor.Run(ctx, target, remote.Command{ID: "probe_sudo_n", Script: "sudo -n true"})
    ```

    With (NEW comment + NEW Script literal — preserving the Bug 7 comment intact below this block):

    ```go
        // Probe whether the SSH user can run sudo non-interactively. The
        // bootstrap path runs over a non-interactive channel and cannot
        // answer a sudo password prompt, so a host with sudo-binary present
        // but a password requirement must NOT pass preflight as if it were
        // passwordless. See gap doc 06-GAP-byo-sudo-handling.md Task A.
        //
        // Bug 31 (Plan 06-13, 2026-05-08): the probe Script MUST be a
        // command that is inside `runnerkit byo-prepare`'s scoped sudoers
        // allowlist, otherwise a Path-C-prepared host (where byo-prepare
        // installed /etc/sudoers.d/runnerkit-installer with NOPASSWD only
        // for the bootstrap commands) still trips the password-required
        // warning and the up command falls through to Path B's TTY
        // prompt — defeating the entire one-time-prepare purpose. The
        // prior probe `sudo -n true` was NOT in the allowlist (only
        // apt-get/dnf/yum/useradd/install/tar/systemctl/svc.sh are; see
        // internal/bootstrap/sudoers.go::RenderSudoersEntry). The new
        // probe `sudo -n install --version >/dev/null` IS in the
        // allowlist (`/usr/bin/install`) and is also a RequiredTools
        // member, so /usr/bin/install is guaranteed present on any host
        // that otherwise passes preflight. The Command.ID stays
        // `probe_sudo_n` so all existing test fakes keep working.
        // Regression test: TestCheckPrivilege_AllowsScopedSudoers.
        probeResult, probeErr := executor.Run(ctx, target, remote.Command{ID: "probe_sudo_n", Script: "sudo -n install --version >/dev/null"})
    ```

    Lines 149-156 (the existing Bug 7 comment block beginning `// Bug 7 fix: classify based on the remote stderr regardless of`) and lines 157-173 (the switch) are UNCHANGED.

    Step 2 — edit `internal/preflight/checks_bugfix_test.go` lines 14-22 doc comment block.

    Replace the existing block (lines 14-22):

    ```go
    // Bug 7: internal/remote/system.go::SystemExecutor.Run returns
    // *exec.ExitError for any non-zero remote exit. The preflight switch
    // at internal/preflight/checks.go::Run requires probeErr == nil for
    // every classification branch, so a real ssh probe that exits non-zero
    // (e.g. `sudo -n true` exit 1 with stderr "sudo: a password is
    // required") never reaches the WARNING branch — it falls into the
    // default ERROR branch ("sudo probe failed: ..."). Plan 06-07 attempt-5
    // surfaced this as a hard preflight failure.
    ```

    With (Bug 7 narrative preserved; example literal updated to current probe Script per Bug 31 / Plan 06-13):

    ```go
    // Bug 7: internal/remote/system.go::SystemExecutor.Run returns
    // *exec.ExitError for any non-zero remote exit. The preflight switch
    // at internal/preflight/checks.go::Run previously required
    // probeErr == nil for every classification branch, so a real ssh
    // probe that exits non-zero (e.g. `sudo -n install --version` exit 1
    // with stderr "sudo: a password is required") would never reach the
    // WARNING branch — it would fall into the default ERROR branch
    // ("sudo probe failed: ..."). Plan 06-07 attempt-5 surfaced this as
    // a hard preflight failure; the fix made the switch tolerant of
    // *exec.ExitError. The probe Script literal was later changed from
    // `sudo -n true` to `sudo -n install --version >/dev/null` (Plan
    // 06-13, Bug 31) so the probe binds to byo-prepare's scoped sudoers
    // allowlist; the Bug 7 stderr-classification contract is unchanged.
    ```

    Test bodies in `checks_bugfix_test.go` (lines 53-102) are NOT changed — they key on Command.ID `probe_sudo_n` and on stderr markers, both stable. The Bug 8 test (lines 104-131) and `readChecksGoSource()` helper (line 133) are NOT touched.

    Step 3 — DO NOT touch `internal/cli/byo_prepare.go` line 133 (`verify_sudo_n` probe). Out-of-scope per the gap_closure_scope. Leave the user-facing Note message ("`sudo -n true` probe did not pass; this is expected — RunnerKit only allows-list the bootstrap commands") AS-IS. The Note is informational and Plan 06-06 SUMMARY line 153 calls it out as intentional.

    Step 4 — run the suite. Expected outcomes:
    - `TestCheckPrivilege_AllowsScopedSudoers` (introduced in Task 1) GOES GREEN — both sub-assertions now hold.
    - All other preflight tests stay green (they don't depend on Script content; Command.ID is stable).
    - `go test ./internal/cli/...` stays green (up_test.go's `passwordRequiredProbe` keys on Command.ID and stderr; no change needed).
    - `go test ./...` overall stays green.

    Step 5 — commit. Single commit modifying both files (the doc comment update belongs with the source change for atomicity):

    ```
    fix(06-13): bug 31 — preflight probe uses sudo -n install --version >/dev/null inside byo-prepare scoped allowlist
    ```

    Reference: Plan 06-12 Task 1's `fix(06-12): bug 28 — probeSudoNeedsPassword inspects ExitCode regardless of exit-status-N err wrapper` is the style exemplar.
  </action>
  <verify>
    <automated>cd /Users/salar/Projects/spool && go test ./internal/preflight/ -count=1 -race && go test ./internal/cli/ -run 'TestUp_SudoPasswordPrompt' -count=1 -race && go vet ./internal/preflight/... && gofmt -l internal/preflight/checks.go internal/preflight/checks_bugfix_test.go</automated>
  </verify>
  <acceptance_criteria>
    - `internal/preflight/checks.go` contains the literal `Script: "sudo -n install --version >/dev/null"`. Verify via `grep -c 'Script: "sudo -n install --version >/dev/null"' internal/preflight/checks.go` returns exactly 1.
    - `internal/preflight/checks.go` does NOT contain `Script: "sudo -n true"`. Verify via `grep -c 'Script: "sudo -n true"' internal/preflight/checks.go` returns 0.
    - `internal/preflight/checks.go` Command.ID stays `probe_sudo_n`. Verify via `grep -c '"probe_sudo_n"' internal/preflight/checks.go` returns at least 1 (i.e., the ID literal is preserved).
    - `internal/preflight/checks.go` contains the `Bug 31` annotation inside the privilege probe block. Verify via `awk '/Probe whether the SSH user can run sudo/,/probeResult, probeErr := executor.Run/' internal/preflight/checks.go | grep -c 'Bug 31'` returns at least 1.
    - `internal/preflight/checks_bugfix_test.go` doc comment references the new probe literal. Verify via `grep -c 'sudo -n install --version' internal/preflight/checks_bugfix_test.go` returns at least 1, and `grep -c 'Bug 31' internal/preflight/checks_bugfix_test.go` returns at least 1.
    - `internal/preflight/checks_bugfix_test.go` test bodies are unchanged (semantic check): `grep -c '"probe_sudo_n"' internal/preflight/checks_bugfix_test.go` returns at least 4 (same as pre-fix — 4 occurrences in runResults+runErrors maps for the two Bug 7 tests).
    - `go test ./internal/preflight/... -count=1 -race` exits 0 — including `TestCheckPrivilege_AllowsScopedSudoers` now GREEN.
    - `go test ./internal/preflight/ -run TestCheckPrivilege_AllowsScopedSudoers -count=1 -v` exits 0 (the test that was RED in Task 1 is now GREEN).
    - `go test ./internal/cli/ -run 'TestUp_SudoPasswordPrompt|TestUpRequiresHostFlag|TestUp_PreflightWiring' -count=1 -race` exits 0 (Plan 06-06 Path B contract preserved).
    - `go test ./... -count=1 -race` exits 0 (full repo green; no cross-package regression).
    - `go vet ./internal/preflight/...` clean.
    - `gofmt -l internal/preflight/checks.go internal/preflight/checks_bugfix_test.go` empty.
    - `internal/bootstrap/sudoers.go` is NOT modified (allowlist contract preserved). Verify via `git diff --name-only HEAD~1 HEAD | grep -c 'internal/bootstrap/sudoers.go'` returns 0.
    - `internal/cli/byo_prepare.go` is NOT modified (verify_sudo_n probe out-of-scope). Verify via `git diff --name-only HEAD~1 HEAD | grep -c 'internal/cli/byo_prepare.go'` returns 0.
    - `internal/cli/up.go` is NOT modified (Path B logic preserved). Verify via `git diff --name-only HEAD~1 HEAD | grep -c 'internal/cli/up.go'` returns 0.
    - `scripts/smoke/byo-permission.sh` is NOT modified. Verify via `git diff --name-only HEAD~1 HEAD | grep -c 'scripts/smoke/'` returns 0.
    - Commit message follows pattern `fix(06-13): bug 31 — preflight probe uses sudo -n install --version >/dev/null inside byo-prepare scoped allowlist`. Verify via `git log -1 --pretty=format:%s`.
    - The single commit modifies EXACTLY 2 files: `internal/preflight/checks.go` and `internal/preflight/checks_bugfix_test.go`. Verify via `git show --stat HEAD | grep -c '|'` returning 2 (two file lines in the diff stat).
  </acceptance_criteria>
  <done>
    The preflight privilege probe now exercises a command inside
    byo-prepare's scoped sudoers allowlist, so a Path-C-prepared host
    classifies as passwordless sudo and `runnerkit up --yes
    --non-interactive` proceeds past preflight without falling through
    to Path B's TTY prompt. Bug 31 closed end-to-end at the unit-test
    layer; the maintainer can now re-run `make smoke-live` (Plan 06-07
    attempt-20) under `tee` and reach SMOKE-GREEN, unblocking the v1.0.0
    tag push per D-13.
  </done>
</task>

</tasks>

<verification>

After Task 2 commits, the full repo verification is:

```bash
cd /Users/salar/Projects/spool
go test ./... -count=1 -race
go vet ./...
gofmt -l internal/preflight/
git log --oneline -2
```

Expected:
- All tests pass.
- `go vet` clean.
- `gofmt -l` empty.
- Last 2 commits, in order:
  - `fix(06-13): bug 31 — preflight probe uses sudo -n install --version >/dev/null inside byo-prepare scoped allowlist`
  - `test(06-13): bug 31 — add TestCheckPrivilege_AllowsScopedSudoers binding probe Script to byo-prepare allowlist`

Cross-package regression checks:
- `internal/cli/up_test.go::passwordRequiredProbe` (line 448) keys on `probe_sudo_n` Command.ID — UNAFFECTED (ID unchanged).
- `internal/cli/up_test.go::TestUp_SudoPasswordPrompt_Interactive` (line 458) — Path B still fires correctly when stderr contains marker, regardless of which probe Script ran.
- `internal/cli/byo_prepare.go::verify_sudo_n` probe (line 133) — UNCHANGED, still emits its informational "Note: post-install probe did not pass" message (out-of-scope cosmetic, intentional per Plan 06-06).
- Plan 06-12 surfaces (down probe, cloud-init timeout, destroy cascade) — UNAFFECTED, no shared code paths.
- Plan 06-11 surfaces (host-key fingerprint, down sudo gate, cloud destroy cascade, scoped sudoers svc.sh glob) — UNAFFECTED, no shared code paths.

Manual maintainer verification (post-landing, pre-attempt-20):

```bash
# 1. Verify the new probe works inside byo-prepare's allowlist on the smoke host
ssh salar@mckee-small-desktop 'sudo -n install --version >/dev/null; echo exit=$?'
# Expected: exit=0

# 2. Re-run byo-prepare (idempotent — should short-circuit since allowlist already current)
go run ./cmd/runnerkit byo-prepare --host salar@mckee-small-desktop
# Expected: "Host salar@mckee-small-desktop:22 is now prepared." OR "already prepared"

# 3. Run BYO smoke under tee (non-TTY) — the original failure mode
make smoke-live 2>&1 | tee /tmp/smoke-byo-attempt-20.log
# Expected: BYO_DURATION_SECONDS=NNN line; no "Sudo password for" prompt;
# no "sudo_password_required" error; runner registers; exit 0.
```

</verification>

<success_criteria>

- [ ] `internal/preflight/checks.go:148` Script literal is `"sudo -n install --version >/dev/null"` (verified via grep).
- [ ] `internal/preflight/checks.go:148` Command.ID is unchanged (`"probe_sudo_n"`).
- [ ] `internal/preflight/checks.go` contains a `Bug 31` annotation inside the privilege probe comment block.
- [ ] `internal/preflight/checks_test.go::TestCheckPrivilege_AllowsScopedSudoers` exists, exercises a Path-C-prepared host fixture, and passes.
- [ ] `internal/preflight/checks_bugfix_test.go` doc comment references the new probe literal and `Bug 31`.
- [ ] All existing preflight tests stay green (10+ tests across `checks_test.go` and `checks_bugfix_test.go`).
- [ ] `internal/bootstrap/sudoers.go::RenderSudoersEntry` is NOT modified — the allowlist contract is preserved as-is.
- [ ] `internal/cli/up.go` and `internal/cli/byo_prepare.go` are NOT modified.
- [ ] `scripts/smoke/byo-permission.sh` is NOT modified.
- [ ] Two atomic commits land in TDD cadence (`test(06-13): ...` then `fix(06-13): ...`), matching Plan 06-12's style.
- [ ] `go test ./... -count=1 -race` exits 0 across the full repo.
- [ ] `go vet ./...` clean; `gofmt -l ...` empty.
- [ ] Plan 06-07 attempt-20 (post-13 fix) reaches `BYO_DURATION_SECONDS=NNN` in `smoke-byo-attempt-20.log` under `tee` (no TTY) — the v1.0.0 release gate (D-13) for REL-05 closes.

</success_criteria>

<output>
After completion, create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-13-preflight-probe-allowlist-fix-SUMMARY.md` per the standard summary template, capturing:

- Bug 31 root cause (`sudo -n true` not in byo-prepare allowlist)
- Fix approach (Option A — replace probe Script with `sudo -n install --version >/dev/null`)
- Files modified (3 — `checks.go`, `checks_test.go`, `checks_bugfix_test.go`)
- Commits landed (2 — TDD cadence)
- Tests added (1 — `TestCheckPrivilege_AllowsScopedSudoers`)
- Tests preserved (10+ — all preflight + Bug 7 + Bug 8 + Path B Plan 06-06)
- Cross-plan invariants preserved (Plans 06-05, 06-06, 06-08..06-12)
- Out-of-scope items left untouched (byo_prepare verify probe, up.go Path B, sudoers.go allowlist content, smoke harness)
- Resume signal for Plan 06-07 attempt-20 maintainer action
</output>
