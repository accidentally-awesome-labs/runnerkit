---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 06
subsystem: cli
tags: [byo, bootstrap, sudo, sudoers, visudo, redactor, prompter, doctor, docs, errcodes]

# Dependency graph
requires:
  - phase: 06-release-upgrade-docs-and-v1-validation/06-05
    provides: preflight.CheckPrivilegePasswordReq stable warning ID + errcodes.BootSudoPasswordRequired (RKD-BOOT-015) + redacted remote stderr surfacing in bootstrap_failed
  - phase: 02-byo-runner-setup
    provides: bootstrap.Apply / ApplyEphemeral, remote.Executor, ui.Prompter pattern
  - phase: 06-release-upgrade-docs-and-v1-validation/06-03
    provides: errcodes.FormatLine docs-anchor pattern
provides:
  - "redact.SudoPassword Kind + replacement marker + sensitiveKindForKey('sudo_password') matcher"
  - "bootstrap.Options.SudoPassword field + private wrapSudoCommand() helper that wraps each sudo command with `printf | { sudo -S ... }`; literal flows via Env (never Script) and is appended to RedactArgs"
  - "ui.PasswordPrompter optional capability interface (type-asserted; legacy Prompter implementations remain compatible)"
  - "cli/up.go promptSudoPasswordForPathB: preflight.CheckPrivilegePasswordReq → TTY+!nonInteractive prompt → redact.SudoPassword register → bootstrapOpts.SudoPassword set; --non-interactive fails fast with byo-prepare + RKD-BOOT-015 remediation"
  - "internal/cli/byo_prepare.go: new top-level `runnerkit byo-prepare --host` command with --remove inverse + idempotent SudoersIsPrepared skip"
  - "internal/bootstrap/sudoers.go: SudoersFilePath, RenderSudoersEntry, RemoteVisudoCheckScript, RemoteSudoersReadScript, RemoteSudoersRemoveScript, SudoersIsPrepared"
  - "doctor finding `byo_host_prepared` (severity=pass/info) emitted when /etc/sudoers.d/runnerkit-installer present on remote host; new ops.DeepChecks.BYOHostPrepared field; cli/doctor.go probes via `test -f` script"
  - "docs: byo-quickstart.md ## Sudo Setup section with Path C → Path B decision tree; troubleshooting/bootstrap.md rkd-boot-015 Fix references real implemented commands (placeholder text removed); README BYO section links to Sudo Setup"
affects: [06-07-live-smoke-rerun-and-baseline-fillin]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "PasswordPrompter optional capability: legacy ui.Prompter implementations stay compatible because callers type-assert via interface{ Password(...) } before calling, allowing existing tests/fakes to satisfy Prompter without implementing Password"
    - "Sudo password transport via remote.Command.Env (RUNNERKIT_SUDO_PASSWORD) + RedactArgs append: literal value never appears in Script, only the env-var name. Script wrapper `printf | { sudo -S ...; }` reads env via stdin pipe to sudo per command"
    - "Idempotent host-state probe: SudoersIsPrepared compares trimmed remote-cat output against trimmed RenderSudoersEntry output before any mutation. Same helper drives both `byo-prepare` skip-on-match AND doctor's byo_host_prepared finding"
    - "Lockout-prevention atomic install: RemoteVisudoCheckScript writes to /tmp tempfile → `sudo visudo -cf <tmp>` → `sudo mv` ONLY on visudo exit 0. visudo failure short-circuits with `exit 21` so /etc/sudoers.d/runnerkit-installer is never overwritten with malformed content"

key-files:
  created:
    - "internal/bootstrap/sudoers.go"
    - "internal/bootstrap/sudoers_test.go"
    - "internal/cli/byo_prepare.go"
    - "internal/cli/byo_prepare_test.go"
  modified:
    - "internal/redact/redact.go"
    - "internal/redact/redact_test.go"
    - "internal/bootstrap/install.go"
    - "internal/bootstrap/install_test.go"
    - "internal/ui/prompt.go"
    - "internal/cli/up.go"
    - "internal/cli/up_test.go"
    - "internal/cli/root.go"
    - "internal/cli/doctor.go"
    - "internal/ops/doctor.go"
    - "internal/ops/doctor_test.go"
    - "docs/byo-quickstart.md"
    - "docs/troubleshooting/bootstrap.md"
    - "README.md"

key-decisions:
  - "Plan 06-06: sudo password flows via remote.Command.Env['RUNNERKIT_SUDO_PASSWORD'] + RedactArgs append, NOT via a new SudoPassword field on remote.Command. This keeps remote.Command's contract minimal AND ensures the literal password never serializes through the remote.Executor abstraction — the script wrapper reads the env var via printf|sudo -S so the rendered Script string only references the env-var name."
  - "Plan 06-06: ui.PasswordPrompter is an OPTIONAL capability interface, not a required Prompter method. Callers type-assert (`deps.Prompts.(ui.PasswordPrompter)`) before using Password(), so all existing fake Prompter implementations in test files remain compatible without modification."
  - "Plan 06-06: byo_host_prepared doctor finding uses ops.SeverityPass (informational/positive) because ops.Severity has only pass/warning/error — no SeverityInfo. The plan suggested 'severity: info'; pass is the closest analog and correctly conveys that path C is a positive observation, not an error or warning. Absence of the finding is NOT itself an error since Path B is a valid alternative."
  - "Plan 06-06: Visudo validation runs on the remote host (sudo visudo -cf <tmp>) inside RemoteVisudoCheckScript, NOT locally. Local validation would require pulling the rendered content back through the SSH channel and would not catch host-specific syntax differences. The atomic-rename pattern (tmp → visudo → mv) guarantees lockout-prevention regardless: even if visudo segfaults, exit 21 short-circuits before mv."
  - "Plan 06-06: --yes does NOT imply --non-interactive for sudo password input (per gap doc lines 177-178). --yes accepts safe defaults (registration token reuse, plan confirmation); the sudo password is a separate human-input concern that always requires a TTY prompt unless --non-interactive is explicitly set. Test TestUp_SudoPasswordPrompt_YesDoesNotImplyNonInteractive locks this contract in."

patterns-established:
  - "Pattern: TDD RED commit asserts new test names + new struct fields BEFORE production change; GREEN commit makes them pass. Used 4 commits (RED + GREEN per task) plus 1 docs commit (Task 3 has no test surface)."
  - "Pattern: Optional capability interface (PasswordPrompter) layered on top of base interface (Prompter) so legacy implementations remain compatible. Same approach as remote.HostKeyProber on remote.Executor."
  - "Pattern: Sensitive value transport via Env + RedactArgs (NOT Script interpolation): the literal flows through remote.Command.Env['NAME'], RedactArgs catches it for stderr scrubbing, and the Script only references $NAME. Provides defense in depth: both the script-string AND any captured stderr are safe to log."

requirements-completed: [REL-05, DOC-04]

# Metrics
duration: 11m
completed: 2026-05-05
---

# Phase 6 Plan 06: BYO Prepare and Sudo Prompt Summary

**Path B (interactive sudo password fallback in `runnerkit up`) + Path C (`runnerkit byo-prepare` scoped sudoers installer with visudo lockout-prevention) close the BYO gap-doc 2026-05-04 user decision so a fresh user can complete BYO setup against a sudo-with-password host without manually editing /etc/sudoers.d/.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-05-05T02:07:08Z
- **Completed:** 2026-05-05T02:18:23Z
- **Tasks:** 3
- **Files modified:** 15 (4 created, 11 modified)

## Accomplishments

- **Path B wired end-to-end:** `internal/cli/up.go` now consults `report.Result(preflight.CheckPrivilegePasswordReq)` after preflight passes; when present AND TTY available AND `--non-interactive` not set, `promptSudoPasswordForPathB` collects the password via `ui.PasswordPrompter`, registers it with `redact.SudoPassword`, threads it into `bootstrap.Options.SudoPassword`, and zeros the in-process buffer in a deferred cleanup. `--non-interactive` (or no TTY) fails fast with remediation explicitly pointing at `runnerkit byo-prepare` AND the `RKD-BOOT-015` docs link, returning `ExitInputRequired`.
- **Sudo password transport via Env (NOT Script):** `bootstrap.wrapSudoCommand` rewrites each sudo-prefixed command's `Script` to use `sudo -S` and reads the password from `$RUNNERKIT_SUDO_PASSWORD`. The literal value flows through `remote.Command.Env` and is appended to `RedactArgs` — it never appears in the rendered Script string OR in any captured stderr. The `printf '%s\n' "$RUNNERKIT_SUDO_PASSWORD" | { sudo -S ...; }` pattern feeds the password to sudo's stdin from a brace-grouped subshell that exports the env var.
- **`runnerkit byo-prepare` is a registered top-level command:** `internal/cli/byo_prepare.go` plus `root.AddCommand(newByoPrepareCommand(...))`. Supports `--host user@host[:port]` (required), `--remove` (inverse), `--yes`. Idempotent: `bootstrap.SudoersIsPrepared` short-circuits with "already prepared" when the on-disk content matches `RenderSudoersEntry(user)` byte-for-byte. Prompts ONCE for the sudo password (via `ui.PasswordPrompter`), registers with the redactor, threads through Env. `runnerkit byo-prepare --help` is discoverable via cobra and shows the full flag matrix.
- **Lockout-prevention atomic install:** `RemoteVisudoCheckScript` writes the rendered content to `/tmp/runnerkit-installer.XXXXXX` (mode 0440), runs `sudo visudo -cf <tmp>`, and atomically renames into `/etc/sudoers.d/runnerkit-installer` ONLY on visudo exit 0. visudo failure short-circuits with `exit 21` so a malformed sudoers file can never be persisted. `TestRemoteVisudoCheckScript_RunsVisudoBeforeMv` asserts this ordering invariant in CI; `TestVisudoValidates_BadSudoersFails` proves the local visudo binary rejects malformed content (skipped when visudo is not on $PATH, e.g. macOS dev box).
- **Sudoers template is SCOPED, not blanket NOPASSWD:** `RenderSudoersEntry("alice")` produces:
  ```
  # /etc/sudoers.d/runnerkit-installer (managed by runnerkit byo-prepare)
  alice ALL=(root) NOPASSWD: \
    /usr/bin/apt-get, /usr/bin/dnf, /usr/bin/yum, \
    /usr/sbin/useradd, \
    /usr/bin/install, \
    /bin/tar, /usr/bin/tar, \
    /bin/systemctl, /usr/bin/systemctl, \
    /opt/runnerkit-runner/svc.sh
  ```
  `TestRenderSudoersEntry` asserts this exact command set, the `# managed by` header, the trailing newline (visudo requires it), AND the absence of `ALL=(ALL) NOPASSWD: ALL` / `ALL: ALL`.
- **`runnerkit doctor` reports byo-prepared status:** `cli/doctor.go::collectDoctorChecks` probes `test -f /etc/sudoers.d/runnerkit-installer` over SSH and feeds the result into `ops.DeepChecks.BYOHostPrepared`. `BuildDoctorReport` emits a `byo_host_prepared` finding (severity=pass) when true; absent otherwise (no false-positive). `TestDoctor_ByoHostPreparedFinding` asserts both branches.
- **`redact.SudoPassword` Kind:** new `Kind = "sudo-password"` with `<redacted:sudo-password>` replacement. Auto-redacts JSON fields whose key matches `sudo_password` regardless of prior registration (`sensitiveKindForKey` extension placed BEFORE the broader password/secret/credential matcher so it routes to `SudoPassword` not `ProviderCredential`).
- **DOC-04 fully restored:** `docs/byo-quickstart.md` has a top-level `## Sudo Setup` section with Path C → Path B decision-tree table covering one-time / always-prompt / CI / pre-NOPASSWD scenarios. `docs/troubleshooting/bootstrap.md`'s `rkd-boot-015` Fix subsection now references the real implemented commands — the placeholder text "Plan 06-06 lands the CLI surface for both" / "Until Plan 06-06 lands, the v1.0.0 documented workaround is to add a NOPASSWD sudoers entry..." is fully removed. README BYO section gains a one-liner pointing at the new Sudo Setup anchor.

## Task Commits

1. **Task 1 (TDD): Path B — interactive sudo password fallback in `runnerkit up`**
   - RED: `77f163d` — `test(06-06): add failing tests for Path B sudo password fallback`
   - GREEN: `de8ab96` — `feat(06-06): Path B interactive sudo password fallback in runnerkit up`
2. **Task 2 (TDD): Path C — `runnerkit byo-prepare` command + sudoers template + doctor finding**
   - RED: `e034a8b` — `test(06-06): add failing tests for byo-prepare command + sudoers template + doctor finding`
   - GREEN: `c05b823` — `feat(06-06): runnerkit byo-prepare command + scoped sudoers + doctor finding`
3. **Task 3: Documentation — byo-quickstart Sudo Setup + RKD-BOOT-015 update + README link**
   - DOCS: `23eb584` — `docs(06-06): byo-quickstart Sudo Setup section + RKD-BOOT-015 update + README link`

**Plan metadata:** (final docs commit appended below after STATE/ROADMAP updates)

## Files Created/Modified

### Created
- `internal/bootstrap/sudoers.go` — Sudoers template renderer, remote scripts (visudo+atomic mv, read, remove), SudoersIsPrepared idempotency probe.
- `internal/bootstrap/sudoers_test.go` — RenderSudoersEntry shape; visudo good/bad validation (skipped when visudo absent); RemoteVisudoCheckScript ordering invariant; SudoersIsPrepared missing-file and matching-content cases.
- `internal/cli/byo_prepare.go` — New `runnerkit byo-prepare` cobra command. runByoPrepareInstall (idempotent → prompt → install → verify), runByoPrepareRemove (try sudo -n then prompt fallback).
- `internal/cli/byo_prepare_test.go` — Command registration, idempotent skip, visudo-failure error path, --remove inverse, --host required, no-TTY fails fast.

### Modified
- `internal/redact/redact.go` — Added `SudoPassword Kind = "sudo-password"`, `<redacted:sudo-password>` replacement, `sensitiveKindForKey('sudo_password')` matcher placed BEFORE the broader provider-credential matcher so the kind routes correctly.
- `internal/redact/redact_test.go` — `TestRedact_SudoPasswordRegistration` covers (a) Register + String redaction and (b) JSON key auto-redaction (without prior registration).
- `internal/bootstrap/install.go` — Added `Options.SudoPassword string` field with full doc comment; added private `wrapSudoCommand(c, opts)` helper that no-ops on empty SudoPassword AND on non-sudo commands; both `Apply` and `ApplyEphemeral` now call `wrapSudoCommand` per command before exec.Run; new `strings` import.
- `internal/bootstrap/install_test.go` — `TestApply_WithSudoPassword_UsesSudoMinusSPipedFromHeredoc` (asserts sudo -S + Env + RedactArgs + literal-not-in-Script) and `TestApply_WithoutSudoPassword_BehaviorUnchangedFromPlan0605` (asserts Plan 06-05 baseline unchanged when SudoPassword=="").
- `internal/ui/prompt.go` — New `PasswordPrompter` optional capability interface with `Password(ctx, prompt) (string, error)`. Callers type-assert; legacy Prompter implementations stay compatible.
- `internal/cli/up.go` — Plan 06-06 Path B: `promptSudoPasswordForPathB` helper handles --non-interactive / no-TTY / no-PasswordPrompter / empty-password rejection paths and prompts via the optional capability when available; runUp consults `report.Result(preflight.CheckPrivilegePasswordReq)` between bootstrapOpts construction and dryRun branch; sets bootstrapOpts.SudoPassword and zeroes it in a deferred cleanup.
- `internal/cli/up_test.go` — `recordingPasswordPrompter` test helper; `passwordRequiredProbe` shared fixture wires fakeRemoteExecutor to return password-required stderr for the probe_sudo_n command; three new tests: TestUp_SudoPasswordPrompt_Interactive, TestUp_SudoPasswordPrompt_NonInteractive_Fails, TestUp_SudoPasswordPrompt_YesDoesNotImplyNonInteractive.
- `internal/cli/root.go` — `root.AddCommand(newByoPrepareCommand(deps, &jsonOutput, &noColor))` registered after newUpgradeRunnerCommand.
- `internal/cli/doctor.go` — Imported `bootstrap`; collectDoctorChecks now runs an additional `test -f /etc/sudoers.d/runnerkit-installer` probe and feeds the result into the new `ops.DeepChecks.BYOHostPrepared` field.
- `internal/ops/doctor.go` — Added `DeepChecks.BYOHostPrepared bool` field; `BuildDoctorReport` emits `byo_host_prepared` finding (SeverityPass, source="remote") with remediation `runnerkit byo-prepare --host {hostRef} --remove # to revert` when checks.BYOHostPrepared is true.
- `internal/ops/doctor_test.go` — `TestDoctor_ByoHostPreparedFinding` asserts both branches (present when true, absent when false) and the canonical evidence/severity values.
- `docs/byo-quickstart.md` — New `## Sudo Setup` section between `## Safety warning` and `## Run setup`. Documents Recommended (`runnerkit byo-prepare`) and Fallback (interactive prompt) paths with explicit no-leak guarantees and a 4-row decision-tree Markdown table.
- `docs/troubleshooting/bootstrap.md` — rkd-boot-015 Fix subsection rewritten: removed all "Plan 06-06 lands" / "Until Plan 06-06 lands" placeholder copy AND the manual `/etc/sudoers.d/runnerkit-temporary` workaround. Added cross-link to docs/byo-quickstart.md#sudo-setup.
- `README.md` — One-liner under BYO install pointing at `docs/byo-quickstart.md#sudo-setup` so first-time users find `runnerkit byo-prepare` immediately.

## Decisions Made

- **Sudo password transport: Env + RedactArgs, not remote.Command.SudoPassword field.** The plan body suggested possibly adding a `SudoPassword` field on `remote.Command`. The cleaner factoring (per the plan's NOTE block) flows the literal through `remote.Command.Env['RUNNERKIT_SUDO_PASSWORD']` and appends it to `RedactArgs`, then the rendered `Script` only references the env-var name. This keeps `remote.Command`'s contract minimal AND provides defense in depth: both the script-string AND any captured stderr are scrubbed.
- **PasswordPrompter as optional capability interface, not new Prompter method.** Adding `Password(...)` to `Prompter` would have broken every fake Prompter in `*_test.go` files (denyingRepoPrompter, recordingSetupPathPrompter, destroyInputPrompter, etc.). Layering `PasswordPrompter` as a separate optional interface lets the CLI type-assert at the call site, keeps legacy fakes compatible, and is the same pattern remote.HostKeyProber uses on remote.Executor.
- **byo_host_prepared severity = SeverityPass (informational), not new SeverityInfo.** The plan suggested "severity: info" but `ops.Severity` has only `pass`/`warning`/`error`. Adding a fourth severity would have ripple effects across the doctor renderer, status renderer, and several errcodes.Code Severity values. Using `SeverityPass` correctly conveys the finding as a positive observation (host has been prepared) and renders correctly in `cli/doctor.go::renderDoctorHuman` which already special-cases pass findings with `ui.Success` styling.
- **Visudo runs on the REMOTE host inside the install script, not locally before sending.** Local visudo invocation would require pulling parsed content back through SSH for round-trip validation, would not catch host-specific syntax differences (e.g. distros with custom sudoers includedir patterns), and would add complexity to the byo-prepare CLI. The remote `RemoteVisudoCheckScript` runs `sudo visudo -cf <tmp>` on the actual host AND short-circuits with `exit 21` on failure BEFORE the atomic mv, so the lockout-prevention guarantee is preserved with simpler factoring.
- **byo-prepare emits a `note` (not warning) when post-install `sudo -n true` fails.** The scoped sudoers template only allow-lists the bootstrap commands (apt-get, useradd, install, tar, systemctl, svc.sh). It does NOT allow-list `true`, so the post-install `sudo -n true` probe will typically fail unless the user has unrelated NOPASSWD config. Treating this as informational (not an install failure) prevents byo-prepare from reporting false-negative install failures while still surfacing the non-result to the user.

## Deviations from Plan

- **[Rule 1 - Bug fix during testing] Visudo failure remediation copy adjustment.** First test run revealed `TestByoPrepare_VisudoValidationFails_DoesNotMoveFile` was checking for the literal string `byo_prepare_failed` in human-rendered output, but the human renderer prints the user-facing message ("could not install the scoped sudoers entry") rather than the machine-readable code. Updated the test to check for either the code OR the message AND additionally assert that the remote stderr (`parse error...`) surfaces — a stronger assertion than the original. The test now locks in BOTH the surfacing invariant AND the message contract. Production code unchanged.

The remaining tasks (sudoers template shape, visudo ordering, idempotent re-runs, --remove inverse, --host required, no-TTY fail-fast, docs cross-links, RKD-BOOT-015 cleanup) translated 1:1 from plan body to implementation. The plan's pseudocode for `wrapSudoCommand`, `RenderSudoersEntry`, `RemoteVisudoCheckScript`, `runByoPrepareInstall`, and the cli/up.go Path B block were used as-is with minor tightening (e.g. the `printf | { ...; }` brace-group wrapper instead of the plan's `{ read; export; ... }` variant — both valid; the brace-group flavor reads more naturally for shell readers).

## Issues Encountered

None blocking. The visudo-failure test message mismatch (described above) was caught immediately by running the targeted test and resolved in one edit.

## User Setup Required

None — the new `runnerkit byo-prepare` command is fully self-contained. Users on existing BYO hosts can opt into Path C by running `runnerkit byo-prepare --host user@host` once, OR continue using Path B (interactive prompt) by running `runnerkit up` directly.

## Next Phase Readiness

- **Plan 06-07 fully unblocked:** Live BYO smoke can now be re-run end-to-end against a fresh host using EITHER Path B (just run `runnerkit up`, type password when prompted) OR Path C (`runnerkit byo-prepare` once, then `runnerkit up --yes` runs passwordlessly). The maintainer no longer needs to manually add `/etc/sudoers.d/runnerkit-smoke-temp` before testing — the smoke can validate the user-facing flows directly.
- **v1.0.0 BYO contract change:** From "host MUST have NOPASSWD sudo configured manually" → "host has password-protected sudo; RunnerKit handles it via Path B prompt or Path C `byo-prepare`." This restores Phase 6 Success Criterion 4 ("A fresh user can complete at least one supported setup path in about 10 minutes") for the realistic case of a fresh Linux host.
- **Verification status:** `go test ./... -count=1 -race` green across all 18 packages; `go run ./cmd/runnerkit byo-prepare --help` shows the new command + all three flags. The errcodes invariant tests (`TestEveryCodeHasDocAnchor`, `TestEntriesFollowSymptomDiagnosisFix`, `TestEachComponentHasMinimumOneEntry`) all pass after the rkd-boot-015 Fix-subsection rewrite.
- **DOC-04 restored to fully satisfied:** `docs/troubleshooting/bootstrap.md` rkd-boot-015 entry no longer references future plan numbers; `docs/byo-quickstart.md` has the Sudo Setup section first-time users see before `## Run setup`; README links into it. No user-facing doc references "Plan 06-06" anywhere.

## Self-Check: PASSED

Per `<self_check>` step in execute-plan workflow:

**Files created exist:**
- FOUND: `internal/bootstrap/sudoers.go`
- FOUND: `internal/bootstrap/sudoers_test.go`
- FOUND: `internal/cli/byo_prepare.go`
- FOUND: `internal/cli/byo_prepare_test.go`
- FOUND: `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-06-byo-prepare-and-sudo-prompt-SUMMARY.md` (this file)

**Commit hashes verified (in `git log --oneline`):**
- FOUND: `77f163d` (Task 1 RED)
- FOUND: `de8ab96` (Task 1 GREEN)
- FOUND: `e034a8b` (Task 2 RED)
- FOUND: `c05b823` (Task 2 GREEN)
- FOUND: `23eb584` (Task 3 docs)

**Key substrings verified in production files:**
- `SudoPassword Kind = "sudo-password"` in `internal/redact/redact.go`
- `<redacted:sudo-password>` in `internal/redact/redact.go`
- `SudoPassword string` field in `internal/bootstrap/install.go::Options`
- `sudo -S` in both `wrapSudoCommand` (install.go) and the byo_prepare install script
- `RUNNERKIT_SUDO_PASSWORD` env var in install.go and byo_prepare.go
- `redact.SudoPassword` registered in BOTH `internal/cli/up.go::promptSudoPasswordForPathB` AND `internal/cli/byo_prepare.go::runByoPrepareInstall`
- `newByoPrepareCommand` registered in `internal/cli/root.go` line 127
- `SudoersFilePath = "/etc/sudoers.d/runnerkit-installer"` in `internal/bootstrap/sudoers.go`
- `visudo -cf` in `internal/bootstrap/sudoers.go::RemoteVisudoCheckScript` BEFORE the `mv` command (lockout-prevention assertion locked in by `TestRemoteVisudoCheckScript_RunsVisudoBeforeMv`)
- `byo_host_prepared` finding emit in `internal/ops/doctor.go`
- `## Sudo Setup` heading in `docs/byo-quickstart.md`
- `runnerkit byo-prepare` references in `docs/byo-quickstart.md`, `docs/troubleshooting/bootstrap.md`, `README.md`
- `byo-quickstart.md#sudo-setup` cross-link in `README.md` AND `docs/troubleshooting/bootstrap.md`

**Placeholder text confirmed scrubbed:**
- `grep "Plan 06-06" docs/troubleshooting/bootstrap.md docs/byo-quickstart.md` returns no matches
- `grep "Until Plan 06-06" docs/troubleshooting/bootstrap.md` returns no matches

**Test status:**
- `go test ./... -count=1 -race` — GREEN (all 18 packages, including 14 new tests added by this plan)
- `go run ./cmd/runnerkit byo-prepare --help` — shows command + --host/--remove/--yes flags
- `go test ./internal/errcodes/... -count=1 -run 'TestEveryCodeHasDocAnchor|TestEntriesFollowSymptomDiagnosisFix|TestEachComponentHasMinimumOneEntry'` — GREEN (rkd-boot-015 entry edits did not break the docs-anchor invariants)

---
*Phase: 06-release-upgrade-docs-and-v1-validation*
*Completed: 2026-05-05*
