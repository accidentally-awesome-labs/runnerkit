---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 07
type: execute
wave: 3
depends_on: [05, 06]
files_modified:
  - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md
  - RELEASE-NOTES-v1.0.0.md
autonomous: false
gap_closure: true
requirements: [REL-05, DOC-04]
must_haves:
  truths:
    - "Maintainer runs `make smoke-live` end-to-end against a fresh BYO host AND a real Hetzner project AFTER Plans 06-05 + 06-06 land. The BYO smoke completes WITHOUT manual `/etc/sudoers.d/runnerkit-smoke-temp NOPASSWD ALL` workaround — instead, the maintainer either runs `runnerkit byo-prepare --host user@host` first (Path C) OR lets `runnerkit up` prompt interactively for the sudo password (Path B)."
    - "10-minute stopwatch BYO and Hetzner total durations are captured with real wall-clock numbers (NOT placeholder text) into `06-VERIFICATION.md` lines 67 and 73 (BYO + Hetzner durations) and `RELEASE-NOTES-v1.0.0.md` 10-Minute Stopwatch table."
    - "5 Hetzner cloud resource IDs (server / ssh-key / primary-ipv4 / primary-ipv6 / firewall) are recorded in 06-VERIFICATION.md line 75; D-12 gate 1 (empty-project precheck) status is PASS; D-12 gate 2 (destroy-verify within RUNNERKIT_SMOKE_TIMEOUT) status is PASS; precheck final ID list size is 0; Hetzner cost in EUR is recorded."
    - "Maintainer signs and dates the verification baseline at 06-VERIFICATION.md line 94 — the v1.0.0 baseline is now a sealed audit document."
    - "Maintainer types resume signal `smoke-green` (per `docs/release-process.md` D-13 / `06-04-v1-validation-and-live-smoke-SUMMARY.md` Pending Maintainer Checkpoint section) which triggers the v1.0.0 tag push (`git tag -a v1.0.0`)."
    - "If anything in the smoke fails, the maintainer types `smoke-red <reason>` and a new gap closure plan is filed; the v1.0.0 tag is NOT pushed and 06-VERIFICATION.md remains unsigned."
  artifacts:
    - path: ".planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md"
      provides: "v1.0.0 baseline FILLED with maintainer's real numbers (no longer skeleton); all 8 automated test items checked; BYO + Hetzner smoke fields complete; stopwatch totals recorded; sign-off block dated."
      contains: "smoke-live-byo` succeeds"
      contains_also: "smoke-live-cloud` succeeds"
      contains_also2: "Maintainer signature"
      contains_also3: "v1.0.0"
    - path: "RELEASE-NOTES-v1.0.0.md"
      provides: "10-Minute Stopwatch table FILLED with real wall-clock durations from the maintainer run; placeholder rows replaced with measured numbers."
      contains: "RunnerKit v1.0.0"
      contains_also: "10-Minute Stopwatch"
  key_links:
    - from: "Plan 06-05 + 06-06 (gap closure code) merged"
      to: "make smoke-live works end-to-end without manual sudoers preconfiguration"
      via: "Path C (`runnerkit byo-prepare`) OR Path B (interactive prompt) handles BYO sudo; download_runner now uses sudo for curl/sha256sum/tar so the install dir owned by runnerkit-runner receives the tarball"
      pattern: "smoke-green"
    - from: "06-VERIFICATION.md filled + signed"
      to: "docs/release-process.md Stopwatch Checklist (D-13) → v1.0.0 git tag push"
      via: "Resume signal `smoke-green` → maintainer runs `git tag -a v1.0.0 -m '...'` → tag-triggered release.yml workflow → GoReleaser produces signed multi-platform release"
      pattern: "git tag -a v1.0.0"
    - from: "Failed smoke (smoke-red) → new gap closure plan"
      to: ".planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-*.md"
      via: "/gsd:verify-work 6 → /gsd:plan-phase 06 --gaps cycle"
      pattern: "smoke-red"
---

<objective>
Re-execute the live maintainer smoke (Plan 06-04 Task 4) now that Plans 06-05 + 06-06 have closed the BYO blockers identified in `06-GAP-byo-sudo-handling.md`. Fill the verification baseline with real numbers from a fresh BYO host + real Hetzner project run, and trip the `smoke-green` resume signal that authorizes the v1.0.0 tag push per `docs/release-process.md` D-13.

This plan has ONE task: a `checkpoint:human-action` block that the maintainer executes against real GitHub + Hetzner credentials. Claude cannot execute this — it requires real `gh auth login`, real `HCLOUD_TOKEN`, real billable Hetzner resource creation, and a wall-clock stopwatch on a clean machine. Plan 06-04's SUMMARY already documents the maintainer-side sequence (Pending Maintainer Checkpoint section, lines 209-228); this plan just unblocks that sequence and adds the gap-closure context.

Implements: closes verification truth #2 from `06-VERIFICATION.md` ("10-minute stopwatch durations + Hetzner cost + Hetzner resource IDs are recorded ... from a real maintainer run on a clean machine").

Purpose: REL-05 (release pipeline + upgrade lifecycle promise) and DOC-04 (cleanup/troubleshooting docs) both move from PARTIAL to SATISFIED in REQUIREMENTS.md. The 10-minute reliable-runner v1 promise is empirically validated for the first time and recorded in an auditable per-release document.

Output: `06-VERIFICATION.md` is fully filled in and signed by the maintainer; `RELEASE-NOTES-v1.0.0.md` 10-Minute Stopwatch table contains real measured numbers; the maintainer is cleared to run `git tag -a v1.0.0 -m 'RunnerKit v1.0.0'` per `docs/release-process.md`.
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
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-04-v1-validation-and-live-smoke-PLAN.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-04-v1-validation-and-live-smoke-SUMMARY.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-05-byo-bootstrap-blocker-fixes-PLAN.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-06-byo-prepare-and-sudo-prompt-PLAN.md
@docs/release-process.md
@RELEASE-NOTES-v1.0.0.md
@Makefile

<interfaces>
<!-- This plan does NOT modify code; it executes the existing live-smoke harness against real services and fills audit fields. -->

Existing harness (Plan 06-04 outputs — DO NOT change):
- `make smoke-live` — chains smoke-live-byo + smoke-live-cloud + smoke-stopwatch (Makefile lines 38-60).
- `scripts/smoke/byo-permission.sh` — Phase 1 outstanding GitHub permission smoke (Plan 06-05 added the `config.sh` post-bootstrap assertion).
- `scripts/smoke/cloud-end-to-end.sh` — Phase 4 outstanding Hetzner billable smoke with `trap cleanup EXIT INT TERM` Pitfall 7 mitigation.
- `scripts/smoke/hetzner-empty-precheck.sh` — D-12 gate 1 wrapper (cmd/_smokebin/empty_precheck).
- `scripts/smoke/hetzner-destroy-verify.sh` — D-12 gate 2 wrapper (cmd/_smokebin/destroy_verify with RUNNERKIT_SMOKE_TIMEOUT default 300s).
- `06-VERIFICATION.md` skeleton (Plan 06-04 Task 3 commit 140cb06) — fields awaiting fill-in detailed in lines 60-83 + 92-95.
- `RELEASE-NOTES-v1.0.0.md` — 10-Minute Stopwatch table awaiting fill-in.

Sequence the maintainer follows (per `06-04-v1-validation-and-live-smoke-SUMMARY.md` lines 219-228, ADAPTED for the post-gap-closure world — Step 3.5 added):
1. Resolve Plan 06-01 prerequisites (already 'tap-ready' as of 2026-05-02).
2. Export `RUNNERKIT_SMOKE_BYO_HOST=user@host`, `RUNNERKIT_SMOKE_REPO=accidentally-awesome-labs/runnerkit-smoke-test` (or any maintainer-controlled trusted repo), `HCLOUD_TOKEN=<from Hetzner project>`; run `gh auth login` if not already.
3. Verify Hetzner project is empty of `runnerkit-*` resources (Console eyeball; the precheck will also refuse if any exist).
3.5. **NEW (post-gap-closure):** Choose Path B or Path C for the BYO host:
     - **Path C (recommended):** `go run ./cmd/runnerkit byo-prepare --host $RUNNERKIT_SMOKE_BYO_HOST` — type sudo password once; persistent passwordless future invocations.
     - **Path B (alternative):** Skip byo-prepare. `make smoke-live-byo` will trigger Path B via preflight; `runnerkit up` will prompt for the password during the smoke. The smoke script must allocate a TTY for this to work — verify by running `runnerkit up` manually from the maintainer terminal once before kicking off `make smoke-live` to confirm the prompt path works against the smoke host.
     - **Forbidden:** the `/etc/sudoers.d/runnerkit-smoke-temp NOPASSWD ALL` manual workaround used in attempt 1 (06-04-SUMMARY lines 240-244) is no longer needed; if used, the smoke does NOT validate the gap closure and `smoke-green` MUST NOT be signaled.
4. `time make smoke-live 2>&1 | tee smoke-output.log`. Watch for `BYO_DURATION_SECONDS=NNN`, empty-precheck OK, `CLOUD_DURATION_SECONDS=NNN`, destroy-verify OK.
5. Run the 10-minute stopwatch on a CLEAN machine (fresh laptop or VM). Follow `docs/release-process.md` Stopwatch Checklist.
6. Fill `06-VERIFICATION.md`: tick all 8 automated test items, fill BYO + Hetzner smoke fields, fill stopwatch totals, sign and date.
7. Fill `RELEASE-NOTES-v1.0.0.md` 10-Minute Stopwatch table.
8. Verify Hetzner project is empty AGAIN (post-smoke; belt-and-suspenders).
9. Commit both files. Resume signal `smoke-green` triggers `/gsd:verify-work` for Phase 6 sign-off; then push `git tag -a v1.0.0` per `docs/release-process.md`.

If smoke FAILS at any step: signal `smoke-red <reason>`. The orchestrator opens a new gap closure plan (06-08+) and the v1.0.0 tag is NOT pushed.

Resume signal source-of-truth: `docs/release-process.md` Stopwatch Checklist (D-13) section + `06-04-v1-validation-and-live-smoke-SUMMARY.md` lines 211-212 ("Resume signal: smoke-green or smoke-red <reason>").
</interfaces>
</context>

<tasks>

<task type="checkpoint:human-action" gate="blocking">
  <name>Task 1: Maintainer runs `make smoke-live` end-to-end and fills 06-VERIFICATION + RELEASE-NOTES baselines</name>
  <files>.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md, RELEASE-NOTES-v1.0.0.md</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-04-v1-validation-and-live-smoke-SUMMARY.md (Pending Maintainer Checkpoint section at lines 209-228 — already specifies the maintainer-side sequence; this plan just adds Step 3.5 for Path B/C choice)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md (current skeleton; the fields you fill are at lines 49-95)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md (the gap that Plans 06-05 + 06-06 closed; reference if smoke fails to identify whether the failure is in the closure or a NEW issue)
    - docs/release-process.md (Stopwatch Checklist section at line 120; the maintainer-only release procedure)
    - RELEASE-NOTES-v1.0.0.md (10-Minute Stopwatch table — the fields you fill)
  </read_first>
  <what-built>
    Plans 06-01 through 06-06 have landed:
    - **06-01:** GoReleaser v2 + cosign keyless + Homebrew Cask + tag-triggered release workflow + PR CI gate + README install matrix + maintainer release-process docs.
    - **06-02:** Lazy 24h-cached update notice + `runnerkit upgrade` (channel detect, print-only) + `runnerkit upgrade-runner` + forward-only state migration + `ExitStateSchemaTooNew` + doctor stale-runner finding.
    - **06-03:** `internal/errcodes/` stable RKD-<COMPONENT>-NNN registry + 6 docs/troubleshooting/ component files + CLI emit-site wiring.
    - **06-04:** `make smoke-live` Makefile targets + cmd/_smokebin/empty_precheck + cmd/_smokebin/destroy_verify + 10-min stopwatch checklist + `06-VERIFICATION.md` skeleton + `RELEASE-NOTES-v1.0.0.md` template.
    - **06-05 (gap closure):** Preflight `sudo -n true` probe (Bug 1 fix); download_runner sudo prefix on curl/sha256sum/tar (Bug 2 fix); RKD-BOOT-015 docs entry; redacted-stderr surfacing on bootstrap_failed; build-tag-guarded integration test; smoke script asserts config.sh landed; new `make test-integration` target.
    - **06-06 (gap closure):** Path B interactive sudo password fallback in `runnerkit up` + `redact.SudoPassword` registration + zeroing; Path C `runnerkit byo-prepare` command with scoped sudoers template + visudo validation + atomic rename + `--remove` inverse + idempotent re-run; `byo_host_prepared` doctor finding; docs/byo-quickstart.md `## Sudo Setup` section; docs/troubleshooting/bootstrap.md rkd-boot-015 entry now references real implemented commands; README.md one-liner under BYO install.

    What this task does: re-runs the live smoke (last attempted 2026-05-04 to 2026-05-05, blocked by Bugs 1 + 2 from the gap doc) end-to-end against a fresh BYO host + real Hetzner project, captures the wall-clock numbers, and fills the verification baseline.
  </what-built>
  <how-to-verify>
    **Maintainer-side sequence** (estimated 30-45 minutes total):

    1. **Pre-flight check.** Confirm Plans 06-05 + 06-06 are merged. Run:
       ```bash
       git log --oneline -20
       go test ./... -count=1 -race    # full suite green
       grep -n "sudo curl" internal/bootstrap/install.go     # Plan 06-05 Bug 2 fix landed
       grep -n "RKD-BOOT-015" internal/errcodes/codes.go     # Plan 06-05 docs anchor landed
       grep -n "newByoPrepareCommand" internal/cli/root.go   # Plan 06-06 Path C landed
       grep -n "redact.SudoPassword" internal/cli/up.go      # Plan 06-06 Path B landed
       ```
       If any of these grep commands return nothing, abort — the gap closure didn't land properly.

    2. **Environment setup.**
       ```bash
       export RUNNERKIT_SMOKE_BYO_HOST=user@host         # your fresh BYO target
       export RUNNERKIT_SMOKE_REPO=owner/repo            # maintainer-controlled trusted repo
       export HCLOUD_TOKEN=$(<your-hetzner-token-file>)  # from Hetzner Console
       gh auth status                                    # if not, run `gh auth login`
       ```

    3. **Empty Hetzner project check (eyeball + automated).**
       Visit Hetzner Console → confirm the project has zero `runnerkit-*` resources. The smoke's D-12 gate 1 will also enforce this; the eyeball check is belt-and-suspenders.

    4. **Choose Path B or Path C for BYO sudo (NEW step — gap closure unblocks both).**

       **Recommended — Path C (one-time persistent setup):**
       ```bash
       go run ./cmd/runnerkit byo-prepare --host $RUNNERKIT_SMOKE_BYO_HOST
       # Type sudo password once when prompted.
       # Output should end with: "Host user@host is now prepared. Run `runnerkit up --host user@host` to install the runner."
       ```
       Verify it installed scoped (NOT blanket NOPASSWD):
       ```bash
       ssh $RUNNERKIT_SMOKE_BYO_HOST 'sudo cat /etc/sudoers.d/runnerkit-installer'
       # Should show the scoped command list (apt-get/dnf/yum/useradd/install/tar/systemctl/svc.sh), NOT "ALL".
       ```

       **OR — Path B (interactive prompt during smoke):** skip `byo-prepare`. The smoke will prompt for the sudo password when `runnerkit up` runs. Note: `byo-permission.sh` runs `go run ./cmd/runnerkit up ... --yes` non-interactively under tee, which may not allocate a TTY for the prompt. If you choose Path B, manually run `go run ./cmd/runnerkit up --repo $RUNNERKIT_SMOKE_REPO --host $RUNNERKIT_SMOKE_BYO_HOST` once first to verify the prompt path works against the smoke host, then run `make smoke-live`.

       **Forbidden:** Do NOT add a manual `/etc/sudoers.d/runnerkit-smoke-temp NOPASSWD ALL` workaround. If you do, the smoke does NOT validate the gap closure and you MUST NOT signal `smoke-green`.

    5. **Run the live smoke.**
       ```bash
       time make smoke-live 2>&1 | tee smoke-output.log
       ```
       Watch for:
       - `===> [smoke-byo] Asserting install dir contains config.sh on the remote host` (NEW — Plan 06-05 Step 2.6 assertion).
       - `BYO_DURATION_SECONDS=NNN`
       - `[empty_precheck] OK` (D-12 gate 1)
       - `CLOUD_DURATION_SECONDS=NNN`
       - `[destroy_verify] all resources gone within NNNs` (D-12 gate 2)

    6. **Run the 10-minute stopwatch on a CLEAN machine.** Follow `docs/release-process.md` Stopwatch Checklist. Two clean-machine runs (BYO + Hetzner). Record real wall-clock numbers.

    7. **Fill `06-VERIFICATION.md`.** Open `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md` and:
       - Tick all 8 automated test items in `## Test Suite (automated)` (lines 49-56). Run each and confirm green: `go test ./... -count=1 -race`, `goreleaser check`, `goreleaser release --snapshot --skip=publish --clean`, the 5 errcodes test names, the 4 state migration test names, the 6 update-check test names, the 7 upgrade tests, the 3 _smokebin tests.
       - Fill BYO smoke fields (lines 60-65): host, repo, duration, runner ID.
       - Fill Hetzner smoke fields (lines 67-78): repo, project, duration, cost EUR, 5 resource IDs (server / ssh-key / primary-ipv4 / primary-ipv6 / firewall), gate 1 PASS, gate 2 PASS, precheck final size = 0.
       - Fill 10-minute stopwatch totals (lines 80-83).
       - Sign + date the maintainer signature (line 94).

    8. **Fill `RELEASE-NOTES-v1.0.0.md` 10-Minute Stopwatch table.** Update the wall-clock numbers in the table to match what you measured in Step 6.

    9. **Belt-and-suspenders Hetzner empty check (post-smoke).**
       ```bash
       go run ./cmd/_smokebin/empty_precheck   # exits 0 if project is empty of runnerkit-* resources
       ```
       Or: visit Hetzner Console → confirm zero `runnerkit-*` resources remain.

    10. **Commit + signal.**
        ```bash
        git add .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md RELEASE-NOTES-v1.0.0.md
        git commit -m "docs(06): fill v1.0.0 verification baseline + release notes from live smoke"
        ```
        Then signal `smoke-green` to the orchestrator. This triggers `/gsd:verify-work 6` for Phase 6 sign-off, then `git tag -a v1.0.0 -m 'RunnerKit v1.0.0'` per `docs/release-process.md`.

    **If anything fails at any step:** signal `smoke-red <reason>`. Examples:
    - `smoke-red byo-prepare visudo validation failed against host X` → file new gap; do NOT proceed to v1.0.0.
    - `smoke-red Path B prompt did not allocate TTY under tee` → file gap to enhance smoke harness or `runnerkit up` TTY detection.
    - `smoke-red destroy_verify timed out at 600s waiting for primary-ipv6 deletion` → file gap; investigate Hetzner provider timing.
    - `smoke-red BYO total 14 minutes exceeds 10-minute target` → file gap; investigate bootstrap step that took longest.
  </how-to-verify>
  <resume-signal>
    `smoke-green` (all gates passed; v1.0.0 baseline filled and signed; ready for tag push) OR `smoke-red <reason>` (a new gap closure plan is needed; v1.0.0 is NOT tagged).

    See `docs/release-process.md` Stopwatch Checklist (D-13) for the resume-signal contract.
  </resume-signal>
  <action>See `<what-built>` and `<how-to-verify>` above for the full 10-step maintainer procedure. Summary: (1) verify Plans 06-05 + 06-06 merged via grep checks for `sudo curl`, `RKD-BOOT-015`, `newByoPrepareCommand`, `redact.SudoPassword`; (2) export RUNNERKIT_SMOKE_BYO_HOST + RUNNERKIT_SMOKE_REPO + HCLOUD_TOKEN; (3) eyeball Hetzner project empty; (4) NEW — choose Path C (`runnerkit byo-prepare --host $HOST`) OR Path B (interactive prompt during smoke), DO NOT use the manual `/etc/sudoers.d/runnerkit-smoke-temp NOPASSWD ALL` workaround; (5) `time make smoke-live 2>&1 | tee smoke-output.log`; (6) run 10-minute stopwatch on a CLEAN machine; (7) fill 06-VERIFICATION.md (8 automated test ticks + BYO + Hetzner fields including 5 resource IDs + D-12 gates 1+2 + cost + stopwatch totals + signature); (8) fill RELEASE-NOTES-v1.0.0.md durations; (9) re-verify Hetzner project empty post-smoke; (10) commit + signal smoke-green. Maintainer cannot delegate this to Claude — real `gh auth login`, real billable Hetzner resources, real wall-clock stopwatch on a clean laptop are required. This is the Phase 6 "we have run out of fakes" boundary.</action>
  <verify><automated>echo "checkpoint:human-action — verified by maintainer resume-signal (smoke-green / smoke-red), not automation"</automated></verify>
  <acceptance_criteria>
    - `06-VERIFICATION.md` line 60-65 BYO smoke fields contain real values (NOT `____`): `user@________`, repo, duration in seconds, runner ID integer.
    - `06-VERIFICATION.md` line 67-78 Hetzner smoke fields contain real values: repo, project name, duration in seconds, cost EUR (decimal), 5 resource IDs, both D-12 gates marked PASS, precheck final size exactly `0`.
    - `06-VERIFICATION.md` line 80-83 10-minute stopwatch totals contain real values (NOT `____ minutes ____ seconds`).
    - `06-VERIFICATION.md` line 94 sign-off block contains a maintainer name + date in ISO format.
    - `RELEASE-NOTES-v1.0.0.md` 10-Minute Stopwatch table contains real wall-clock numbers (NOT placeholder text).
    - `grep "____" .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md` returns no matches OR only matches inside instructional/example text (the fillable fields are filled).
    - Commit landed: `git log --oneline -1` shows the verification baseline + release notes update.
    - Maintainer typed `smoke-green` to the orchestrator (or `smoke-red <reason>` if anything failed).
  </acceptance_criteria>
  <done>
    `06-VERIFICATION.md` is the sealed v1.0.0 baseline; `RELEASE-NOTES-v1.0.0.md` has real durations; resume signal `smoke-green` clears the v1.0.0 tag push; REL-05 + DOC-04 move from PARTIAL to SATISFIED.
  </done>
</task>

</tasks>

<verification>
Verification is the maintainer's wall-clock smoke. There is no automated `<verify>` for this plan because the gates are real billable Hetzner resources, real GitHub registration, and real wall-clock measurement on a clean machine — none of which Claude can produce.

Post-checkpoint verification (Claude-executable):
1. `grep "____" .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md` returns no matches in fillable fields.
2. `git log --oneline -1` shows the verification commit.
3. The orchestrator runs `/gsd:verify-work 6` and the verifier verdict reports PHASE COMPLETE (no gaps).
</verification>

<success_criteria>
- Maintainer ran `make smoke-live` end-to-end against fresh BYO + real Hetzner WITHOUT the manual `/etc/sudoers.d/runnerkit-smoke-temp NOPASSWD ALL` workaround used in attempt 1 (uses Path C `runnerkit byo-prepare` OR Path B interactive prompt instead).
- `06-VERIFICATION.md` is fully filled, signed, dated; serves as the v1.0.0 audit baseline.
- `RELEASE-NOTES-v1.0.0.md` 10-Minute Stopwatch table contains real wall-clock measurements.
- `smoke-green` resume signal received → v1.0.0 tag push authorized per `docs/release-process.md`.
- Verifier (`/gsd:verify-work 6`) confirms Phase 6 has zero remaining gaps; REQUIREMENTS.md REL-05 + DOC-04 status moves from PARTIAL to SATISFIED.
- Phase 6 closes; v1.0.0 ships.
</success_criteria>

<output>
After the maintainer completes the checkpoint, create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-07-live-smoke-rerun-and-baseline-fillin-SUMMARY.md` documenting:
- Whether Path B or Path C was used for BYO sudo (and why).
- Wall-clock numbers from the smoke (BYO total, Hetzner total, 10-minute stopwatch totals).
- 5 Hetzner resource IDs that were created and then verified destroyed.
- Hetzner cost in EUR.
- Maintainer signature + date.
- Commit hash of the verification baseline + release notes update.
- Resume signal sent (`smoke-green` or `smoke-red <reason>`).
- If `smoke-green`: pointer to the v1.0.0 tag commit (after `git tag -a v1.0.0` is pushed).
- If `smoke-red`: pointer to the new gap closure plan filed.
</output>
