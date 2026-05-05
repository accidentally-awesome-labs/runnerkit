---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 06
type: execute
wave: 2
depends_on: [05]
files_modified:
  - internal/cli/up.go
  - internal/cli/up_test.go
  - internal/cli/byo_prepare.go
  - internal/cli/byo_prepare_test.go
  - internal/cli/root.go
  - internal/bootstrap/install.go
  - internal/bootstrap/sudoers.go
  - internal/bootstrap/sudoers_test.go
  - internal/redact/redact.go
  - internal/redact/redact_test.go
  - internal/ops/doctor.go
  - internal/ops/doctor_test.go
  - internal/preflight/checks.go
  - docs/byo-quickstart.md
  - docs/troubleshooting/bootstrap.md
  - README.md
autonomous: true
gap_closure: true
requirements: [REL-05, DOC-04]
must_haves:
  truths:
    - "`runnerkit byo-prepare --host user@host` is a registered top-level cobra command that installs a scoped `/etc/sudoers.d/runnerkit-installer` entry (mode 0440) idempotently, validates with `visudo -c` before persisting, and supports `--remove` for the inverse operation."
    - "`runnerkit up --host user@host` against a host where preflight returns the new `host.privilege.password_required` warning (from Plan 06-05) automatically falls back to interactive sudo password prompting when stdin is a TTY and `--non-interactive` is NOT set; the prompted password is registered with `redact.SudoPassword` and never leaks into state, logs, JSON, or error messages."
    - "`runnerkit up --non-interactive --host user@host` against a host requiring sudo password fails fast with remediation explicitly pointing at `runnerkit byo-prepare` AND the new `RKD-BOOT-015` docs link."
    - "`runnerkit doctor --host user@host` (or against saved state) detects the presence of `/etc/sudoers.d/runnerkit-installer` and reports a `byo_host_prepared` finding (info severity), surfacing whether Path C has been applied."
    - "Path C scoped sudoers template grants NOPASSWD for the minimum command set required by bootstrap (apt-get/dnf/yum, useradd, install, tar, systemctl, /opt/runnerkit-runner/svc.sh) — NOT a blanket `ALL: ALL` NOPASSWD."
    - "`docs/byo-quickstart.md` has a top-level `## Sudo Setup` section with the Path C → Path B decision tree; `docs/troubleshooting/bootstrap.md`'s `RKD-BOOT-015` entry (added by Plan 06-05) is updated to reference the now-implemented `runnerkit byo-prepare` command (no longer 'Plan 06-06 lands the CLI surface'); README.md BYO install section has a one-liner pointing at `docs/byo-quickstart.md#sudo-setup`."
    - "No raw sudo password leaks into state files, logs, JSON output, or error messages — `redact.SudoPassword` is registered immediately upon prompt and zeroed in a deferred cleanup before bootstrap returns."
  artifacts:
    - path: "internal/cli/byo_prepare.go"
      provides: "New top-level `runnerkit byo-prepare` cobra command — installs scoped sudoers entry idempotently with visudo validation; --remove inverse; --host required."
      contains: "byo-prepare"
      contains_also: "--remove"
      contains_also2: "--host"
      contains_also3: "newByoPrepareCommand"
    - path: "internal/cli/byo_prepare_test.go"
      provides: "Tests for byo-prepare: idempotent re-run, visudo validation failure handling, --remove inverse, --non-interactive failure path."
      contains: "TestByoPrepare_Idempotent"
      contains_also: "TestByoPrepare_VisudoValidationFails"
      contains_also2: "TestByoPrepare_Remove"
    - path: "internal/bootstrap/sudoers.go"
      provides: "Sudoers template renderer + visudo runner. Renders the scoped NOPASSWD entry for apt-get/dnf/yum/useradd/install/tar/systemctl/svc.sh."
      contains: "NOPASSWD"
      contains_also: "visudo -c"
      contains_also2: "/etc/sudoers.d/runnerkit-installer"
    - path: "internal/bootstrap/sudoers_test.go"
      provides: "Tests for sudoers template (correct command set, mode 0440, atomic temp+rename pattern); visudo runner test."
      contains: "TestRenderSudoers"
      contains_also: "TestVisudoValidates"
    - path: "internal/cli/root.go"
      provides: "newByoPrepareCommand registered via root.AddCommand; appears in `runnerkit --help` output."
      contains: "newByoPrepareCommand"
    - path: "internal/cli/up.go"
      provides: "Plan 06-05 redacted-stderr surfacing extended with Path B fallback: when preflight emits host.privilege.password_required AND stdin is TTY AND not --non-interactive, prompt for sudo password and pass via Sudo + sudo -S in remote.Command."
      contains: "host.privilege.password_required"
      contains_also: "redact.SudoPassword"
      contains_also2: "sudo -S"
    - path: "internal/bootstrap/install.go"
      provides: "Apply / ApplyEphemeral accept an optional sudo password channel via Options.SudoPassword; when set, sudo-prefixed commands use `sudo -S` and pipe the password from a stdin redirect; when unset, behavior is unchanged from Plan 06-05."
      contains: "SudoPassword"
      contains_also: "sudo -S"
    - path: "internal/redact/redact.go"
      provides: "New SudoPassword Kind + replacement marker + sensitive-key matcher (registers when prompted; redactor strips raw password from any output)."
      contains: "SudoPassword"
      contains_also: "sudo-password"
    - path: "internal/redact/redact_test.go"
      provides: "Tests for SudoPassword registration + redaction (string and JSON paths) + sensitiveKindForKey matcher for `sudo_password` key."
      contains: "TestSudoPassword"
    - path: "internal/ops/doctor.go"
      provides: "New `byo_host_prepared` finding emitted when /etc/sudoers.d/runnerkit-installer exists on the target host (probed via remote shell `test -f`)."
      contains: "byo_host_prepared"
      contains_also: "/etc/sudoers.d/runnerkit-installer"
    - path: "internal/preflight/checks.go"
      provides: "After Plan 06-05, the new password_required warning's remediation now references the actual `runnerkit byo-prepare` command (not 'will be added in Plan 06-06')."
      contains: "runnerkit byo-prepare"
    - path: "docs/byo-quickstart.md"
      provides: "Top-level `## Sudo Setup` section above `## Run setup`, with Path C → Path B decision tree and exact byo-prepare invocation."
      contains: "## Sudo Setup"
      contains_also: "runnerkit byo-prepare"
    - path: "docs/troubleshooting/bootstrap.md"
      provides: "RKD-BOOT-015 entry updated — Path C and Path B sections now point at the implemented commands (not 'Plan 06-06 lands the CLI surface')."
      contains: "rkd-boot-015"
      contains_also: "runnerkit byo-prepare"
    - path: "README.md"
      provides: "One-liner under BYO install referencing the new Sudo Setup section."
      contains: "byo-quickstart.md#sudo-setup"
  key_links:
    - from: "internal/preflight/checks.go::CheckPrivilege (password_required branch)"
      to: "internal/cli/up.go (Path B prompt fallback)"
      via: "report.Result(CheckPrivilegePasswordReq) lookup → if hit AND TTY AND !nonInteractive → prompt via deps.Prompts.Password → register redact.SudoPassword → bootstrap.Options.SudoPassword set"
      pattern: "CheckPrivilegePasswordReq"
    - from: "internal/cli/up.go (sudo password prompt)"
      to: "redact.Redactor (registers + zeroes after bootstrap returns)"
      via: "renderer.Redactor().Register(redact.SudoPassword, password); defer renderer.Redactor().Forget(redact.SudoPassword) (or equivalent) so the password is no longer redacted/retained after bootstrap returns"
      pattern: "redact.SudoPassword"
    - from: "internal/cli/up.go (sudo password set)"
      to: "internal/bootstrap/install.go::Apply / ApplyEphemeral"
      via: "bootstrap.Options.SudoPassword string; when non-empty, the rendered Script uses `sudo -S` and reads the password from a Heredoc-piped stdin per command"
      pattern: "sudo -S"
    - from: "internal/cli/byo_prepare.go::runByoPrepare"
      to: "internal/bootstrap/sudoers.go::RenderSudoersEntry + RunVisudoCheck + WriteSudoersFile"
      via: "Render → Write to /tmp/runnerkit-installer-${pid} on remote → visudo -cf <tmp> → atomic mv to /etc/sudoers.d/runnerkit-installer (mode 0440) → re-probe sudo -n true to verify"
      pattern: "visudo -cf"
    - from: "internal/ops/doctor.go::byo_host_prepared finding"
      to: "remote shell `test -f /etc/sudoers.d/runnerkit-installer`"
      via: "deps.RemoteExecutor.Run with Script: 'test -f /etc/sudoers.d/runnerkit-installer'; ExitCode 0 → finding present, ExitCode 1 → finding absent"
      pattern: "test -f /etc/sudoers.d/runnerkit-installer"
    - from: "docs/byo-quickstart.md ## Sudo Setup"
      to: "docs/troubleshooting/bootstrap.md#rkd-boot-015"
      via: "Markdown link in the decision-tree paragraph"
      pattern: "rkd-boot-015"
---

<objective>
Land the user-facing features that complete the gap doc's Path B + Path C decision (user, 2026-05-04). Plan 06-05 fixed the BLOCKER bugs and emitted the `host.privilege.password_required` preflight warning + the `RKD-BOOT-015` docs anchor. This plan wires the CLI surfaces that consume them:

1. **Task B — Path B (interactive sudo password fallback).** When preflight emits `host.privilege.password_required` AND stdin is a TTY AND `--non-interactive` is NOT set, `runnerkit up` prompts locally once for the sudo password, registers it with the redactor, threads it through `bootstrap.Options.SudoPassword`, and uses `sudo -S` + stdin-piped password per remote command. With `--non-interactive`, fails fast with remediation pointing at `runnerkit byo-prepare`.

2. **Task C — Path C (`runnerkit byo-prepare` command).** New top-level cobra command. SSH to `--host user@host`, prompt locally once for sudo password (TTY-allocated), idempotently install a SCOPED `/etc/sudoers.d/runnerkit-installer` entry (mode 0440) granting NOPASSWD only for apt-get/dnf/yum/useradd/install/tar/systemctl/svc.sh, validate with `visudo -cf <tmp>` BEFORE atomically renaming into place (a malformed sudoers file can lock the user out), supports `--remove` for the inverse, and re-probes `sudo -n true` to verify. Also adds a `byo_host_prepared` doctor finding so `runnerkit doctor` surfaces whether Path C has been applied.

3. **Task D — Documentation.** `docs/byo-quickstart.md` gains a `## Sudo Setup` section above `## Run setup` with the Path C → Path B decision tree; `docs/troubleshooting/bootstrap.md`'s `RKD-BOOT-015` entry (added by Plan 06-05 with placeholders) is updated to reference the now-implemented commands; `README.md` BYO install section gets a one-liner pointing at the new Sudo Setup section.

Implements gap doc `## Required work` Tasks B + C + D. Closes verification truth: "BYO bootstrap completes end-to-end against a real host without manual sudoers preconfiguration." (now fully achievable — Path B handles zero-host-setup, Path C handles persistent passwordless setup, Plan 06-05 handles the preflight + bug fix that gates them).

Purpose: Restore Phase 6 success criterion 4 — "A fresh user can complete at least one supported setup path in about 10 minutes" — for hosts that DON'T already have NOPASSWD sudo configured. The maintainer's documented v1.0.0 contract changes from "configure NOPASSWD manually" to "run `runnerkit byo-prepare` once OR let `runnerkit up` prompt".

Output: A solo developer with a fresh Linux host (sudo with password) can run `runnerkit up` and either (a) be prompted once for the password and complete bootstrap end-to-end (Path B), or (b) run `runnerkit byo-prepare` once for persistent passwordless future invocations (Path C). DOC-04 is restored to fully satisfied (the new RKD-BOOT-015 entry and Sudo Setup section both reference real implemented commands).
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
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-05-byo-bootstrap-blocker-fixes-PLAN.md
@internal/cli/root.go
@internal/cli/up.go
@internal/cli/upgrade_runner.go
@internal/bootstrap/install.go
@internal/bootstrap/script.go
@internal/preflight/checks.go
@internal/redact/redact.go
@internal/ops/doctor.go
@internal/errcodes/codes.go
@internal/remote/executor.go
@docs/byo-quickstart.md
@docs/troubleshooting/bootstrap.md
@README.md

<interfaces>
<!-- Key contracts from existing source. Plan 06-05 must land first; this plan consumes its outputs. -->

cli command registration pattern (internal/cli/root.go:117-126):
```go
root.AddCommand(newUpCommand(deps, &jsonOutput, &noColor))
root.AddCommand(newUpgradeRunnerCommand(deps, &jsonOutput, &noColor))
// Add: root.AddCommand(newByoPrepareCommand(deps, &jsonOutput, &noColor))
```

cli command file template (internal/cli/upgrade_runner.go is the closest analog — single command, --host implicit via saved state but byo-prepare requires explicit --host since no state exists yet, --yes confirmation, opts struct with cobra.Flags binding). Pattern:
```go
func newByoPrepareCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
    opts := &byoPrepareOptions{}
    cmd := &cobra.Command{Use: "byo-prepare"}
    cmd.Short = "Install a scoped sudoers entry on a BYO host so runnerkit up runs passwordlessly"
    cmd.Flags().StringVar(&opts.host, "host", "", "BYO SSH target as user@host or user@host:port (required)")
    cmd.Flags().BoolVar(&opts.remove, "remove", false, "remove the runnerkit-installer sudoers entry instead of installing it")
    cmd.Flags().BoolVar(&opts.yes, "yes", false, "skip confirmation prompts")
    cmd.RunE = func(_ *cobra.Command, _ []string) error { return runByoPrepare(deps, *jsonOutput, *noColor, opts) }
    return cmd
}
```

remote.Command struct (internal/remote/executor.go:18-25):
```go
type Command struct {
    ID         string
    Script     string
    Sudo       bool        // existing field; flips a bool but doesn't carry password
    Timeout    time.Duration
    Env        map[string]string
    RedactArgs []string
}
```
NOTE: this plan does NOT add a `SudoPassword` field to `remote.Command` — instead, Plan 06-06 threads the password through `bootstrap.Options.SudoPassword`, and the bootstrap script renderers / Apply build the `sudo -S` invocation that pipes the password from a Heredoc to sudo's stdin per command. This keeps remote.Command's contract minimal AND ensures the password never serializes through the remote.Executor abstraction.

bootstrap.Options struct (internal/bootstrap/install.go:13-49) — extend with:
```go
type Options struct {
    // ... existing 12 fields ...
    // SudoPassword, when set, causes Apply / ApplyEphemeral to wrap each
    // sudo-prefixed command in `sudo -S <cmd>` with the password piped
    // via a Heredoc from a generated stdin redirect. Empty means use the
    // existing NOPASSWD-style sudo invocation. The caller (CLI) MUST
    // register the value with redact.SudoPassword before passing it here
    // and zero the buffer in a deferred cleanup after Apply returns.
    SudoPassword string
}
```

redact pattern (internal/redact/redact.go:14-21, 154-171):
```go
type Kind string
const (
    GitHubToken             Kind = "github-token"
    // ... existing 5 ...
)
// Add: SudoPassword Kind = "sudo-password"
func replacement(kind Kind) string {
    switch kind {
    case SudoPassword: return "<redacted:sudo-password>"
    // ...
    }
}
// Extend sensitiveKindForKey at line 142:
//   if strings.Contains(lower, "sudo") && strings.Contains(lower, "password") { return SudoPassword, true }
```

doctor finding emit (internal/ops/doctor.go) — pattern: append to findings slice with a Code, severity, evidence, and remediation. The new `byo_host_prepared` is informational (severity: info), not an error or warning. Look at how `runner_version_stale` is emitted (Plan 06-02) as the closest analog.

ui.Prompter (internal/ui — used by deps.Prompts in cli/up.go:337) needs a `Password(ctx, prompt) (string, error)` method. If it doesn't exist yet, ADD it to the Prompter interface AND provide a no-op default implementation in the test helper so existing tests don't break. Search `internal/ui/` for the existing Prompter definition before adding.

Plan 06-05 outputs that 06-06 consumes:
- `internal/preflight/checks.go::CheckPrivilegePasswordReq` (new const) — Path B uses `report.Result(CheckPrivilegePasswordReq)` to detect when to prompt.
- `internal/errcodes/codes.go::BootSudoPasswordRequired` (RKD-BOOT-015) — Task D updates docs and `--non-interactive` failure remediation reference this.
- `docs/troubleshooting/bootstrap.md#rkd-boot-015` (new entry) — Task D updates the entry to point at real commands instead of "Plan 06-06 lands ..." placeholder text.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Path B — interactive sudo password fallback in `runnerkit up`</name>
  <files>internal/cli/up.go, internal/cli/up_test.go, internal/bootstrap/install.go, internal/bootstrap/install_test.go, internal/redact/redact.go, internal/redact/redact_test.go, internal/preflight/checks.go</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md (Task B at lines 154-178; constraint at lines 177-178: `--yes` does NOT imply non-interactive sudo)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-05-byo-bootstrap-blocker-fixes-PLAN.md (Plan 06-05 — defines CheckPrivilegePasswordReq const, BootSudoPasswordRequired code, and the redacted-stderr surfacing helper `lastCommandStderr`)
    - internal/cli/up.go (CURRENT runUp at line 94; preflight call at line 159; existing renderer.Redactor().Register pattern at line 205 for runner registration token)
    - internal/cli/up_test.go (existing pattern for testing runUp with fake remote.Executor + fake Prompts + recording renderer)
    - internal/bootstrap/install.go (Apply at line 66, ApplyEphemeral at line 107; how Sudo: true commands work; Plan 06-05 already prefixed download_runner internals with sudo)
    - internal/redact/redact.go (Kind enum at lines 14-21; replacement at lines 154-171; sensitiveKindForKey at lines 124-152)
    - internal/preflight/checks.go (CheckPrivilegePasswordReq const added by Plan 06-05; how the Result is added to report.Results)
    - internal/ui/ (look for the Prompter interface — Select/Confirm exist; need to add Password if missing)
  </read_first>
  <behavior>
    - Test 1 (`TestRedact_SudoPasswordRegistration`): registering a value via `r.Register(redact.SudoPassword, "p@ssw0rd!")` causes `r.String("the password is p@ssw0rd! today")` to return `"the password is <redacted:sudo-password> today"`. JSON path with key `"sudo_password"` value `"p@ssw0rd!"` returns `"<redacted:sudo-password>"`.
    - Test 2 (`TestUp_SudoPasswordPrompt_Interactive`): drive runUp with a fake preflight that emits a `host.privilege.password_required` warning AND `deps.TTY.StdinTTY = true` AND `opts.nonInteractive = false`. Provide a fake `deps.Prompts` whose `Password(...)` returns `"correct horse battery staple"`. Assert: (a) renderer.Redactor() now contains the registered SudoPassword value, (b) `bootstrap.Apply` is called with `Options.SudoPassword == "correct horse battery staple"`, (c) the password value does NOT appear in any captured renderer output (Step lines, Error lines, JSON). Use a recordingExecutor that captures the rendered remote.Command Scripts to verify (b) by examining whether `sudo -S` appears in the script body for sudo-prefixed commands.
    - Test 3 (`TestUp_SudoPasswordPrompt_NonInteractive_Fails`): same fake preflight emits password_required, but `opts.nonInteractive = true`. Assert: runUp returns a non-nil error with the renderer's captured Error containing both `"runnerkit byo-prepare"` AND `"RKD-BOOT-015"` (or the URL anchor) in the remediation slice. NEVER prompts (deps.Prompts.Password is never called).
    - Test 4 (`TestUp_SudoPasswordPrompt_YesDoesNotImplyNonInteractive`): `opts.yes = true` AND `opts.nonInteractive = false` AND TTY available — prompt STILL fires. Per gap doc constraint at lines 177-178.
    - Test 5 (`TestApply_WithSudoPassword_UsesSudoMinusSPipedFromHeredoc`): unit test on bootstrap.Apply with Options.SudoPassword non-empty. Assert each rendered Command.Script for sudo-prefixed commands now uses `sudo -S` AND reads from a Heredoc that includes the password VARIABLE `$RUNNERKIT_SUDO_PASSWORD` (NOT the literal password — it flows via Env field instead, so the renderer never embeds the literal). The literal password lands in `command.Env["RUNNERKIT_SUDO_PASSWORD"]` and `command.RedactArgs` so the executor handles redaction. The Script uses `printf '%s\n' "$RUNNERKIT_SUDO_PASSWORD" | sudo -S <cmd>` pattern.
    - Test 6 (`TestApply_WithoutSudoPassword_BehaviorUnchangedFromPlan0605`): Options.SudoPassword == "" — rendered scripts match Plan 06-05's output exactly (no `sudo -S`, no `RUNNERKIT_SUDO_PASSWORD` env). Backward-compat verification.
  </behavior>
  <action>
    **Step 1.1 — Add SudoPassword Kind to redact.** In `internal/redact/redact.go`:
    - Add `SudoPassword Kind = "sudo-password"` to the const block at line 21.
    - Add `case SudoPassword: return "<redacted:sudo-password>"` in `replacement()` at line 156.
    - Extend `sensitiveKindForKey()` at line 142 — add BEFORE the `if strings.Contains(lower, "hcloud") ...` line:
      ```go
      if strings.Contains(lower, "sudo") && strings.Contains(lower, "password") {
          return SudoPassword, true
      }
      ```

    **Step 1.2 — Add SudoPassword field to bootstrap.Options.** In `internal/bootstrap/install.go::Options` (line 13), add a new field at the end of the struct:
    ```go
    // SudoPassword, when non-empty, causes Apply / ApplyEphemeral to render
    // sudo-prefixed commands as `printf '%s\n' "$RUNNERKIT_SUDO_PASSWORD" | sudo -S <cmd>`.
    // The literal password is passed via the remote.Command.Env field
    // (NOT interpolated into Script) and registered for redaction. Empty
    // means use the existing NOPASSWD-style sudo invocation.
    SudoPassword string
    ```
    DO NOT modify the existing Apply/ApplyEphemeral command construction yet — Step 1.4 introduces a renderer helper that consumes SudoPassword.

    **Step 1.3 — Add Prompter.Password method.** Find the Prompter interface in `internal/ui/` (likely `prompts.go` or similar). If `Password(ctx, prompt)` doesn't exist, add it to the interface AND implement a default `survey`-backed (or stdlib-backed) version. For tests, ensure the existing fake Prompter in `internal/cli/*_test.go` is extended to satisfy the new method. If the Prompter interface lives in a separate package, this step also requires updating that package's tests; use grep to locate all Prompter implementations BEFORE making the interface change.

    **Step 1.4 — Add sudo-S renderer helper to bootstrap.** In `internal/bootstrap/install.go`, add a new private helper near the top of the file:
    ```go
    // wrapSudoCommand wraps a single command's Script with `sudo -S`
    // semantics if opts.SudoPassword is non-empty, also setting the env
    // var that the script reads. When SudoPassword is empty the input
    // command is returned unchanged so Plan 06-05's behavior is preserved.
    func wrapSudoCommand(c remote.Command, opts Options) remote.Command {
        if opts.SudoPassword == "" {
            return c
        }
        // Replace bare `sudo ` invocations in the script with `sudo -S ` AND
        // pipe the password from a Heredoc-equivalent printf. We do NOT
        // interpolate the password into the script; it flows via Env.
        // The script reads $RUNNERKIT_SUDO_PASSWORD from the environment
        // each command remote.Executor sets up.
        c.Script = strings.ReplaceAll(c.Script, "sudo ", "sudo -S ")
        c.Script = "printf '%s\\n' \"$RUNNERKIT_SUDO_PASSWORD\" | { read -r RUNNERKIT_SUDO_PASSWORD; export RUNNERKIT_SUDO_PASSWORD; " + c.Script + " }"
        // NB: the simple replace above also catches `sudo -u xxxx` which is fine
        // because `sudo -S -u xxxx` is the supported form; verify in tests.
        if c.Env == nil {
            c.Env = map[string]string{}
        }
        c.Env["RUNNERKIT_SUDO_PASSWORD"] = opts.SudoPassword
        c.RedactArgs = append(c.RedactArgs, opts.SudoPassword)
        return c
    }
    ```
    NOTE: the `printf | { read; export; ... }` shell pattern feeds the password to the env-var-reading subshell so the password is NEVER in the rendered script string itself — only `$RUNNERKIT_SUDO_PASSWORD` appears, which is safe to log. Implementer: verify the exact shell pattern compiles + executes correctly in test before committing.

    Then in `Apply` (line 66), wrap each command before appending:
    ```go
    out := Result{Commands: make([]remote.Result, 0, len(commands))}
    for _, command := range commands {
        command = wrapSudoCommand(command, opts)  // Path B sudo password injection
        result, err := exec.Run(ctx, target, command)
        // ... rest unchanged ...
    }
    ```
    Apply the IDENTICAL change in `ApplyEphemeral` (line 107).

    **Step 1.5 — Wire Path B in cli/up.go.** AFTER preflight (currently line 159) and BEFORE bootstrap.Apply (line 218), add:
    ```go
    // Path B: if preflight emitted host.privilege.password_required AND we
    // have a TTY AND --non-interactive is not set, prompt locally for the
    // sudo password and thread it through bootstrap.Options.SudoPassword.
    if pw, ok := report.Result(preflight.CheckPrivilegePasswordReq); ok {
        _ = pw
        if opts.nonInteractive {
            remediation := []string{
                "Run `runnerkit byo-prepare --host " + target.Display() + "` to install a scoped sudoers entry, then re-run `runnerkit up`.",
                errcodes.FormatLine(errcodes.BootSudoPasswordRequired),
            }
            _ = renderer.Error("sudo_password_required", "RunnerKit can't prompt for sudo password in non-interactive mode.", remediation)
            return NewExitError(ExitInputRequired, errors.New("sudo password required but --non-interactive set"))
        }
        if !deps.TTY.StdinTTY || deps.Prompts == nil {
            remediation := []string{
                "Run `runnerkit byo-prepare --host " + target.Display() + "` first; this terminal does not support interactive prompting.",
                errcodes.FormatLine(errcodes.BootSudoPasswordRequired),
            }
            _ = renderer.Error("sudo_password_required", "RunnerKit needs a sudo password but no TTY is available.", remediation)
            return NewExitError(ExitInputRequired, errors.New("sudo password required but no TTY"))
        }
        password, err := deps.Prompts.Password(ctx, ui.Prompt{Message: "Sudo password for " + target.Display() + ":"})
        if err != nil {
            return err
        }
        renderer.Redactor().Register(redact.SudoPassword, password)
        bootstrapOpts.SudoPassword = password
        defer func() {
            // Best-effort scrub: the SudoPassword value is no longer needed
            // after Apply returns; the redactor entry stays so any deferred
            // log writes still scrub it from output, but the in-process
            // bootstrapOpts copy is zeroed.
            bootstrapOpts.SudoPassword = ""
        }()
    }
    ```
    Place this block between line 175 (`bootstrapOpts := buildBootstrapOptions(...)`) and line 181 (`if opts.dryRun ...`). Ensure `redact` is already imported (yes, line 20).

    **Step 1.6 — Update preflight remediation copy.** In `internal/preflight/checks.go::Run` — the password_required branch added by Plan 06-05 — update the remediation string from "Run `runnerkit byo-prepare --host user@host` to install a scoped sudoers entry, or re-run `runnerkit up` interactively to be prompted for the sudo password." to remove the "to install a scoped sudoers entry" qualifier since Plan 06-06 makes the command real (the docs in Task 4 of THIS plan link it). Functionally trivial copy edit; surface so the user sees a working command name.

    **Step 1.7 — TDD cycle.** Write Tests 1-6 BEFORE production changes. Tests 1 + 5 + 6 are pure unit tests on redact and bootstrap. Tests 2-4 require careful test fixture wiring — look at `internal/cli/up_byo_test.go` for the existing patterns of testing runUp with a fake preflight result + recording renderer + recording remote.Executor. RED commit first.
  </action>
  <verify>
    <automated>go test ./internal/redact/... ./internal/bootstrap/... ./internal/cli/... -count=1 -run 'TestRedact_SudoPasswordRegistration|TestUp_SudoPasswordPrompt_Interactive|TestUp_SudoPasswordPrompt_NonInteractive_Fails|TestUp_SudoPasswordPrompt_YesDoesNotImplyNonInteractive|TestApply_WithSudoPassword_UsesSudoMinusSPipedFromHeredoc|TestApply_WithoutSudoPassword_BehaviorUnchangedFromPlan0605'</automated>
    <secondary>grep -n "redact.SudoPassword" internal/cli/up.go</secondary>
    <secondary>grep -n "sudo -S" internal/bootstrap/install.go</secondary>
    <secondary>grep -n "Password(" internal/cli/up.go</secondary>
    <secondary>grep -n "SudoPassword" internal/bootstrap/install.go internal/redact/redact.go</secondary>
    <secondary>go test ./... -count=1   # full regression — no other package breaks on Prompter interface extension</secondary>
  </verify>
  <acceptance_criteria>
    - `internal/redact/redact.go` exports `SudoPassword Kind = "sudo-password"` and `replacement(SudoPassword)` returns `"<redacted:sudo-password>"` and `sensitiveKindForKey("sudo_password")` returns `(SudoPassword, true)`.
    - `internal/bootstrap/install.go::Options` has the new `SudoPassword string` field; `Apply` and `ApplyEphemeral` both call `wrapSudoCommand(command, opts)` before `exec.Run`.
    - `internal/bootstrap/install.go` contains the literal `sudo -S` (proves the wrap helper inserts the flag).
    - `internal/cli/up.go` contains the literal `redact.SudoPassword` (proves password is registered before bootstrap call) AND `deps.Prompts.Password(` (proves Path B prompts) AND the remediation paths reference both `runnerkit byo-prepare` AND `errcodes.BootSudoPasswordRequired`.
    - All 6 new tests pass green.
    - Full Go test suite green: `go test ./... -count=1`.
    - No raw password value appears in any captured test output (assert via `strings.Contains(captured, "correct horse battery staple")` returning false in Test 2).
  </acceptance_criteria>
  <done>
    Path B interactive sudo password fallback is wired end-to-end: preflight detects, CLI prompts, redactor registers, bootstrap renders `sudo -S` per command, password never leaks. `--non-interactive` fails fast with byo-prepare remediation.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Path C — `runnerkit byo-prepare` command + sudoers template + visudo validation + doctor finding</name>
  <files>internal/cli/byo_prepare.go, internal/cli/byo_prepare_test.go, internal/cli/root.go, internal/bootstrap/sudoers.go, internal/bootstrap/sudoers_test.go, internal/ops/doctor.go, internal/ops/doctor_test.go</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md (Task C at lines 180-225 — sudoers template at lines 193-202; visudo step at lines 204-206; doctor finding at line 213-214)
    - internal/cli/upgrade_runner.go (full file — closest analog of a single-purpose cobra command with --host implicit via state, --yes, --force flag pattern; byo_prepare adapts this with --host EXPLICIT, --remove inverse, --yes confirmation)
    - internal/cli/root.go (newUpgradeRunnerCommand registration at line 126 — newByoPrepareCommand registers identically)
    - internal/remote/executor.go (Command + Result types; how byo_prepare uses deps.RemoteExecutor.Run to drive remote shell)
    - internal/preflight/checks.go (the sudo -n true probe pattern from Plan 06-05 — byo-prepare re-uses the same probe at end of run to verify)
    - internal/ops/doctor.go (existing finding emit pattern — look for runner_version_stale per Plan 06-02; byo_host_prepared follows the same shape but with severity: info)
    - internal/cli/doctor.go (CLI-level doctor command; if it accepts --host directly, byo_host_prepared is checked there; if it loads from saved state, the finding is checked against the host saved in state — VERIFY which by reading the file)
  </read_first>
  <behavior>
    - Test 1 (`TestRenderSudoersEntry`): given a user `alice`, returns a string containing exactly the lines from gap doc lines 194-202 (with `<user>` substituted). Mode + ownership comments at top match `# /etc/sudoers.d/runnerkit-installer (managed by runnerkit byo-prepare)`. Trailing newline present (sudoers requires it). Asserts the command list contains `apt-get`, `dnf`, `yum`, `useradd`, `install`, `tar`, `systemctl`, `/opt/runnerkit-runner/svc.sh`. Does NOT contain `ALL: ALL` or any unscoped wildcard.
    - Test 2 (`TestVisudoValidates_GoodSudoersPasses`): a syntactically-valid sudoers content writes to a tmp file, `visudo -cf <tmp>` returns ExitCode 0. Skips when `which visudo` returns nothing (CI environments without sudo package).
    - Test 3 (`TestVisudoValidates_BadSudoersFails`): a deliberately malformed sudoers content (e.g. missing `ALL=` entirely) returns non-zero exit code from `visudo -cf` AND the function returns a non-nil error containing the visudo stderr. PROVES we don't move a bad file into /etc/sudoers.d/.
    - Test 4 (`TestByoPrepare_Idempotent`): two consecutive `runByoPrepare` invocations against the same target with a fake remote.Executor that captures commands. First invocation: writes sudoers, validates, atomic-renames into place, re-probes sudo -n true (fake returns ExitCode 0). Second invocation: reads existing /etc/sudoers.d/runnerkit-installer, sees content matches, exits with "already prepared" message, NO write, NO visudo call (assert via captured commands).
    - Test 5 (`TestByoPrepare_VisudoValidationFails_DoesNotMoveFile`): fake remote.Executor returns ExitCode 1 + stderr "syntax error in sudoers" for the visudo step. Assert: runByoPrepare returns a non-nil error AND NO subsequent `mv` command is recorded (the visudo failure short-circuits before atomic rename). The user is NOT locked out of sudo.
    - Test 6 (`TestByoPrepare_Remove`): runByoPrepare with `opts.remove = true` against a target where /etc/sudoers.d/runnerkit-installer exists. Asserts a `rm /etc/sudoers.d/runnerkit-installer` command is sent AND a final `sudo -n true` probe verifies the entry is gone (post-remove probe should report password_required again unless other sudoers entries exist).
    - Test 7 (`TestDoctor_ByoHostPreparedFinding`): in `internal/ops/doctor_test.go`, drive Doctor against a fake remote.Executor that returns ExitCode 0 for `Script: "test -f /etc/sudoers.d/runnerkit-installer"`. Assert the returned findings slice contains a finding with ID == `byo_host_prepared` and severity == info. When ExitCode is 1, the finding is absent (no false-positive). NB: if `internal/cli/doctor.go` is the actual emit site, the test lives in `internal/cli/doctor_test.go` instead.
    - Test 8 (`TestByoPrepareCommandRegistered`): `cmd := NewRootCommand(deps)` — `cmd.Find([]string{"byo-prepare"})` returns the registered command (NOT an error). Smoke test that root.go registration landed.
  </behavior>
  <action>
    **Step 2.1 — Create `internal/bootstrap/sudoers.go`.** New file:
    ```go
    package bootstrap

    import (
        "context"
        "fmt"
        "strings"

        "github.com/accidentally-awesome-labs/runnerkit/internal/remote"
    )

    // SudoersFilePath is the canonical absolute path of the scoped sudoers
    // entry that `runnerkit byo-prepare` installs.
    const SudoersFilePath = "/etc/sudoers.d/runnerkit-installer"

    // RenderSudoersEntry renders the scoped sudoers entry that grants the
    // SSH user passwordless sudo for the minimum command set required by
    // bootstrap (apt-get/dnf/yum, useradd, install, tar, systemctl,
    // svc.sh). The output is suitable for writing to SudoersFilePath after
    // visudo validation. user MUST be the SSH user; no input sanitization
    // is done here — callers MUST use a previously-validated remote.Target.
    //
    // The exact line set comes from gap doc lines 194-202.
    func RenderSudoersEntry(user string) string {
        return fmt.Sprintf(`# /etc/sudoers.d/runnerkit-installer (managed by runnerkit byo-prepare)
%s ALL=(root) NOPASSWD: \
  /usr/bin/apt-get, /usr/bin/dnf, /usr/bin/yum, \
  /usr/sbin/useradd, \
  /usr/bin/install, \
  /bin/tar, /usr/bin/tar, \
  /bin/systemctl, /usr/bin/systemctl, \
  /opt/runnerkit-runner/svc.sh
`, user)
    }

    // RemoteVisudoCheckScript renders the script that writes the proposed
    // sudoers content to a tempfile (mode 0440), runs `visudo -cf <tmp>`,
    // and on success ATOMICALLY renames into SudoersFilePath. Reads the
    // proposed content from $RUNNERKIT_SUDOERS_CONTENT (passed via Env).
    // On visudo failure, the tempfile is removed and the script exits
    // non-zero so the caller surfaces the validation failure (and the
    // user is NOT locked out of sudo).
    func RemoteVisudoCheckScript() string {
        return `set -euo pipefail
TMP=$(mktemp /tmp/runnerkit-installer.XXXXXX)
printf '%s' "$RUNNERKIT_SUDOERS_CONTENT" | sudo tee "$TMP" >/dev/null
sudo chmod 0440 "$TMP"
if ! sudo visudo -cf "$TMP"; then
  sudo rm -f "$TMP"
  echo "visudo validation failed; sudoers entry not installed" >&2
  exit 21
fi
sudo mv "$TMP" ` + SudoersFilePath + `
sudo chmod 0440 ` + SudoersFilePath + `
`
    }

    // RemoteSudoersReadScript reads the existing sudoers entry (if any)
    // and emits its content on stdout. Exit 0 means the file exists; exit
    // 1 means it doesn't. The caller uses the stdout to compare against
    // RenderSudoersEntry for idempotent re-runs.
    func RemoteSudoersReadScript() string {
        return `set -euo pipefail
if [ -f ` + SudoersFilePath + ` ]; then
  sudo cat ` + SudoersFilePath + `
else
  exit 1
fi
`
    }

    // RemoteSudoersRemoveScript removes the scoped sudoers entry.
    func RemoteSudoersRemoveScript() string {
        return `set -euo pipefail
sudo rm -f ` + SudoersFilePath + `
`
    }

    // SudoersIsPrepared returns true when the remote sudoers file exists
    // AND its content is byte-identical to what RenderSudoersEntry(user)
    // would produce (idempotent check). Used by both byo-prepare (skip
    // re-write) and doctor (emit byo_host_prepared finding).
    func SudoersIsPrepared(ctx context.Context, exec remote.Executor, target remote.Target, user string) (bool, error) {
        result, err := exec.Run(ctx, target, remote.Command{ID: "read_sudoers", Script: RemoteSudoersReadScript()})
        if err != nil || result.ExitCode != 0 {
            return false, nil // file missing is not an error
        }
        return strings.TrimSpace(result.Stdout) == strings.TrimSpace(RenderSudoersEntry(user)), nil
    }
    ```

    **Step 2.2 — Create `internal/bootstrap/sudoers_test.go`.** Tests 1, 2, 3 from the behavior block. For Test 2 + 3 (visudo invocation), use `os/exec` to invoke `visudo` locally — gate with `t.Skip` if `exec.LookPath("visudo")` fails (Linux-only; macOS dev box may not have it). The test writes a temp file with rendered content, runs `visudo -cf <tmp>`, asserts ExitCode 0 for a good file and non-zero for a bad file.

    **Step 2.3 — Create `internal/cli/byo_prepare.go`.** Pattern follows `internal/cli/upgrade_runner.go`:
    ```go
    package cli

    import (
        "context"
        "errors"
        "fmt"
        "strings"

        "github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
        "github.com/accidentally-awesome-labs/runnerkit/internal/redact"
        "github.com/accidentally-awesome-labs/runnerkit/internal/remote"
        "github.com/accidentally-awesome-labs/runnerkit/internal/ui"
        "github.com/spf13/cobra"
    )

    type byoPrepareOptions struct {
        host   string
        remove bool
        yes    bool
    }

    func newByoPrepareCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
        opts := &byoPrepareOptions{}
        cmd := &cobra.Command{Use: "byo-prepare"}
        cmd.Short = "Install a scoped sudoers entry on a BYO host so runnerkit up runs passwordlessly"
        cmd.Long = "Installs /etc/sudoers.d/runnerkit-installer (mode 0440) with a NOPASSWD entry for the minimum command set required by runnerkit up. Validated by visudo before atomic rename. Idempotent. Use --remove to delete the entry."
        cmd.Flags().StringVar(&opts.host, "host", "", "BYO SSH target as user@host or user@host:port (required)")
        cmd.Flags().BoolVar(&opts.remove, "remove", false, "remove the runnerkit-installer sudoers entry instead of installing it")
        cmd.Flags().BoolVar(&opts.yes, "yes", false, "skip the local sudo password prompt confirmation")
        cmd.RunE = func(_ *cobra.Command, _ []string) error {
            return runByoPrepare(deps, *jsonOutput, *noColor, opts)
        }
        return cmd
    }

    func runByoPrepare(deps Dependencies, jsonOutput bool, noColor bool, opts *byoPrepareOptions) error {
        renderer := newRenderer(deps, jsonOutput, noColor)
        ctx := context.Background()
        if strings.TrimSpace(opts.host) == "" {
            _ = renderer.Error("input_required", "RunnerKit can't continue because --host is required.", []string{"Pass --host user@host."})
            return NewExitError(ExitInputRequired, errors.New("--host required"))
        }
        target, err := remote.ParseTarget(opts.host)
        if err != nil {
            _ = renderer.Error("invalid_host", "RunnerKit can't parse --host.", []string{err.Error()})
            return NewExitError(ExitInvalidInput, err)
        }
        if opts.remove {
            return runByoPrepareRemove(ctx, deps, renderer, target)
        }
        return runByoPrepareInstall(ctx, deps, renderer, target, opts)
    }

    func runByoPrepareInstall(ctx context.Context, deps Dependencies, renderer *ui.Renderer, target remote.Target, opts *byoPrepareOptions) error {
        // Idempotency check.
        if prepared, _ := bootstrap.SudoersIsPrepared(ctx, deps.RemoteExecutor, target, target.User); prepared {
            fmt.Fprintf(deps.Out, "Host %s is already prepared (sudoers entry matches expected content).\n", target.Display())
            return nil
        }
        // Prompt for sudo password (one-time; fed to remote sudo via stdin).
        if !deps.TTY.StdinTTY || deps.Prompts == nil {
            _ = renderer.Error("input_required", "RunnerKit needs a sudo password but no TTY is available.", []string{"Run runnerkit byo-prepare from an interactive terminal."})
            return NewExitError(ExitInputRequired, errors.New("no TTY"))
        }
        password, err := deps.Prompts.Password(ctx, ui.Prompt{Message: "Sudo password for " + target.Display() + ":"})
        if err != nil {
            return err
        }
        renderer.Redactor().Register(redact.SudoPassword, password)
        // Render sudoers content + run visudo validation + atomic rename in one remote command.
        content := bootstrap.RenderSudoersEntry(target.User)
        cmd := remote.Command{
            ID:         "install_sudoers",
            Script:     bootstrap.RemoteVisudoCheckScript(),
            Sudo:       true,
            Env:        map[string]string{"RUNNERKIT_SUDOERS_CONTENT": content, "RUNNERKIT_SUDO_PASSWORD": password},
            RedactArgs: []string{password},
        }
        // For first install, sudo requires the password — wrap with sudo -S like Path B does.
        cmd.Script = "printf '%s\\n' \"$RUNNERKIT_SUDO_PASSWORD\" | sudo -S -v && " + cmd.Script
        result, err := deps.RemoteExecutor.Run(ctx, target, cmd)
        if err != nil || result.ExitCode != 0 {
            stderr := strings.TrimSpace(result.Stderr)
            _ = renderer.Error("byo_prepare_failed", "RunnerKit could not install the scoped sudoers entry.", []string{
                "Remote stderr: " + renderer.Redactor().String(stderr),
                "Verify the sudo password is correct and that the SSH user has sudo access.",
            })
            return NewExitError(ExitSafetyGate, err)
        }
        // Re-probe sudo -n true to verify the entry took effect.
        verify, err := deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "verify_sudo_n", Script: "sudo -n true"})
        if err != nil || verify.ExitCode != 0 {
            _ = renderer.Error("byo_prepare_failed", "Installed sudoers entry did not pass post-install verification (`sudo -n true` failed).", []string{"Inspect /etc/sudoers.d/runnerkit-installer manually on the host."})
            return NewExitError(ExitSafetyGate, fmt.Errorf("post-install verification failed"))
        }
        fmt.Fprintf(deps.Out, "Host %s is now prepared. Run `runnerkit up --host %s` to install the runner.\n", target.Display(), target.Display())
        return nil
    }

    func runByoPrepareRemove(ctx context.Context, deps Dependencies, renderer *ui.Renderer, target remote.Target) error {
        // Removal still needs sudo; if the entry is already what gives passwordless sudo,
        // sudo -n rm should work. If not, prompt for password.
        cmd := remote.Command{ID: "remove_sudoers", Script: bootstrap.RemoteSudoersRemoveScript(), Sudo: true}
        result, err := deps.RemoteExecutor.Run(ctx, target, cmd)
        if (err != nil || result.ExitCode != 0) && deps.TTY.StdinTTY && deps.Prompts != nil {
            password, perr := deps.Prompts.Password(ctx, ui.Prompt{Message: "Sudo password for " + target.Display() + ":"})
            if perr != nil {
                return perr
            }
            renderer.Redactor().Register(redact.SudoPassword, password)
            cmd.Env = map[string]string{"RUNNERKIT_SUDO_PASSWORD": password}
            cmd.RedactArgs = []string{password}
            cmd.Script = "printf '%s\\n' \"$RUNNERKIT_SUDO_PASSWORD\" | sudo -S rm -f " + bootstrap.SudoersFilePath
            result, err = deps.RemoteExecutor.Run(ctx, target, cmd)
        }
        if err != nil || result.ExitCode != 0 {
            _ = renderer.Error("byo_prepare_remove_failed", "RunnerKit could not remove the sudoers entry.", []string{"Remote stderr: " + renderer.Redactor().String(result.Stderr)})
            return NewExitError(ExitSafetyGate, err)
        }
        fmt.Fprintf(deps.Out, "Removed %s from host %s.\n", bootstrap.SudoersFilePath, target.Display())
        return nil
    }
    ```

    **Step 2.4 — Register in root.go.** In `internal/cli/root.go` line 126 area:
    ```go
    root.AddCommand(newUpgradeRunnerCommand(deps, &jsonOutput, &noColor))
    root.AddCommand(newByoPrepareCommand(deps, &jsonOutput, &noColor))   // NEW
    ```

    **Step 2.5 — Add doctor finding.** In `internal/ops/doctor.go` (or `internal/cli/doctor.go` — verify which is the emit site by reading both before editing), add a new finding emit. The pattern (per Plan 06-02's `runner_version_stale`) is:
    ```go
    // Check whether Path C has been applied on the host. Emits an INFO
    // finding either way (presence is informational; absence is not an
    // error since Path B is also valid).
    if exec != nil && target.Host != "" {
        result, err := exec.Run(ctx, target, remote.Command{ID: "doctor_byo_host_prepared", Script: "test -f " + bootstrap.SudoersFilePath})
        if err == nil && result.ExitCode == 0 {
            findings = append(findings, ops.Finding{
                ID:       "byo_host_prepared",
                Severity: ops.SeverityInfo,
                Message:  "Host has /etc/sudoers.d/runnerkit-installer (runnerkit byo-prepare applied).",
                Evidence: bootstrap.SudoersFilePath + " exists on remote host",
            })
        }
    }
    ```
    NB: ops.SeverityInfo may not exist — if only error/warning/pass are defined, use the closest analog (likely SeverityPass or add SeverityInfo to ops). Verify by reading internal/ops/finding.go (or wherever Severity lives) BEFORE adding the new severity. If a new severity is needed, also add the corresponding errcodes/Code (skip — info severity does NOT need an RKD code per the registry pattern).

    **Step 2.6 — Tests.** Write Tests 1-8 BEFORE production code (RED commit). Tests 4-6 + 8 use the existing `recordingExecutor` pattern from `internal/bootstrap/install_test.go` adapted for the cli package. Test 7 follows the doctor test pattern in `internal/ops/doctor_test.go`. Test 8 uses `cobra.Command.Find` to assert registration.

    **Step 2.7 — TDD cycle.** RED commit: write tests + stub functions returning errors. GREEN commit: implement runByoPrepareInstall + runByoPrepareRemove + sudoers helpers + doctor finding. Verify all 8 tests + full regression suite pass green.
  </action>
  <verify>
    <automated>go test ./internal/bootstrap/... ./internal/cli/... ./internal/ops/... -count=1 -run 'TestRenderSudoersEntry|TestVisudoValidates_GoodSudoersPasses|TestVisudoValidates_BadSudoersFails|TestByoPrepare_Idempotent|TestByoPrepare_VisudoValidationFails_DoesNotMoveFile|TestByoPrepare_Remove|TestDoctor_ByoHostPreparedFinding|TestByoPrepareCommandRegistered'</automated>
    <secondary>grep -n "newByoPrepareCommand" internal/cli/root.go internal/cli/byo_prepare.go</secondary>
    <secondary>grep -n "visudo -c" internal/bootstrap/sudoers.go</secondary>
    <secondary>grep -n "/etc/sudoers.d/runnerkit-installer" internal/bootstrap/sudoers.go</secondary>
    <secondary>grep -n "byo_host_prepared" internal/ops/doctor.go internal/cli/doctor.go</secondary>
    <secondary>grep -n "NOPASSWD" internal/bootstrap/sudoers.go</secondary>
    <secondary>go test ./... -count=1   # full regression — no other package breaks on the new severity / new command registration</secondary>
    <secondary>go run ./cmd/runnerkit byo-prepare --help   # verify command is discoverable</secondary>
  </verify>
  <acceptance_criteria>
    - `internal/cli/byo_prepare.go` exists; defines `newByoPrepareCommand` and `runByoPrepare`; cobra Use string is `"byo-prepare"`; supports `--host`, `--remove`, `--yes` flags.
    - `internal/cli/root.go` includes `root.AddCommand(newByoPrepareCommand(deps, &jsonOutput, &noColor))` in NewRootCommand.
    - `internal/bootstrap/sudoers.go` exists; exports `SudoersFilePath = "/etc/sudoers.d/runnerkit-installer"`, `RenderSudoersEntry(user)`, `RemoteVisudoCheckScript()`, `RemoteSudoersReadScript()`, `RemoteSudoersRemoveScript()`, `SudoersIsPrepared(...)`.
    - `internal/bootstrap/sudoers.go` `RenderSudoersEntry` output contains `NOPASSWD`, `apt-get`, `dnf`, `yum`, `useradd`, `install`, `tar`, `systemctl`, `/opt/runnerkit-runner/svc.sh`. Does NOT contain `ALL: ALL` or `ALL=(ALL) NOPASSWD: ALL`.
    - `internal/bootstrap/sudoers.go` `RemoteVisudoCheckScript()` contains `visudo -cf` AND atomic rename (`mv`) only AFTER visudo passes.
    - `internal/ops/doctor.go` (or `internal/cli/doctor.go`) emits a `byo_host_prepared` finding when `test -f /etc/sudoers.d/runnerkit-installer` returns ExitCode 0.
    - All 8 new tests pass green.
    - Full Go test suite green: `go test ./... -count=1`.
    - `runnerkit byo-prepare --help` runs successfully and shows the new command.
    - The visudo validation test (Test 5) PROVES that a malformed sudoers content causes the function to return non-nil error AND no `mv` command appears in the captured executor commands. This is the "user is NOT locked out of sudo" guarantee.
  </acceptance_criteria>
  <done>
    `runnerkit byo-prepare` is a registered, tested top-level command. Sudoers template is scoped (NOT blanket NOPASSWD). visudo validates BEFORE atomic rename. Idempotent. --remove inverse works. Doctor finds and reports byo_host_prepared.
  </done>
</task>

<task type="auto">
  <name>Task 3: Documentation — byo-quickstart Sudo Setup section + RKD-BOOT-015 update + README one-liner</name>
  <files>docs/byo-quickstart.md, docs/troubleshooting/bootstrap.md, README.md</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md (Task D at lines 227-233)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-05-byo-bootstrap-blocker-fixes-PLAN.md (Step 1.4 — the placeholder text in the rkd-boot-015 entry that THIS task replaces)
    - docs/byo-quickstart.md (CURRENT structure: Prerequisites → Safety warning → Run setup → What RunnerKit does → Add the workflow labels — Sudo Setup must go BETWEEN Safety warning and Run setup so the user reads it before running `runnerkit up`)
    - docs/troubleshooting/bootstrap.md (Plan 06-05 added rkd-boot-015 entry with placeholder text "Plan 06-06 lands the CLI surface for both" — Task 3 here removes that placeholder)
    - README.md (CURRENT line 116 references docs/byo-quickstart.md — Task D adds a sibling line under that mentioning Sudo Setup)
  </read_first>
  <action>
    **Step 3.1 — Add Sudo Setup section to docs/byo-quickstart.md.** Insert a new `## Sudo Setup` section between the existing `## Safety warning` (currently around line 13-21) and `## Run setup` (currently line 23). Content:

    ```markdown
    ## Sudo Setup

    RunnerKit's bootstrap commands run as the SSH user with sudo. RunnerKit supports two paths so the SSH user does NOT need to manually edit `/etc/sudoers.d/`:

    ### Recommended: `runnerkit byo-prepare` (one-time, persistent)

    Run this once per BYO host. RunnerKit prompts locally for the sudo password, installs a SCOPED `/etc/sudoers.d/runnerkit-installer` entry granting NOPASSWD only for the bootstrap commands (`apt-get`/`dnf`/`yum`, `useradd`, `install`, `tar`, `systemctl`, `svc.sh`), validates with `visudo -c` before atomically renaming into place, and verifies passwordless sudo works:

    ```bash
    runnerkit byo-prepare --host user@host
    ```

    After this completes, every subsequent `runnerkit up --host user@host` runs passwordlessly.

    To revert:

    ```bash
    runnerkit byo-prepare --host user@host --remove
    ```

    ### Fallback: interactive password prompt during `runnerkit up`

    If you have NOT run `runnerkit byo-prepare`, `runnerkit up` detects the password requirement during preflight and prompts you locally for the sudo password. The password is registered with RunnerKit's redactor immediately so it never leaks into logs, JSON output, or error messages, and is zeroed from process memory after bootstrap returns:

    ```bash
    runnerkit up --repo owner/name --host user@host
    # RunnerKit: "Sudo password for user@host:" → type password → bootstrap proceeds
    ```

    `--non-interactive` (or piping stdin) disables this fallback. In that case, `runnerkit up` exits with `RKD-BOOT-015` and remediation pointing at `runnerkit byo-prepare`. See [docs/troubleshooting/bootstrap.md#rkd-boot-015](troubleshooting/bootstrap.md#rkd-boot-015).

    ### Decision tree

    | Scenario | Recommended path |
    | --- | --- |
    | One-time setup; want passwordless from now on | `runnerkit byo-prepare --host user@host` |
    | Don't want host-side state changes; happy to type password | `runnerkit up --host user@host` (prompts) |
    | CI / automation / no TTY | `runnerkit byo-prepare` first, then `runnerkit up --non-interactive` |
    | Already configured `/etc/sudoers.d/` manually with NOPASSWD ALL | `runnerkit up` works without prompt (but prefer scoped `byo-prepare`) |
    ```

    **Step 3.2 — Update RKD-BOOT-015 entry in docs/troubleshooting/bootstrap.md.** Plan 06-05 added a placeholder version. Replace the Fix section so it no longer says "Plan 06-06 lands the CLI surface for both":

    Find the section (added by Plan 06-05) starting with `<a name="rkd-boot-015"></a>` and replace the entire Fix subsection with:

    ```markdown
    ### Fix

    **Recommended — `runnerkit byo-prepare`** (one-time scoped sudoers entry):

    ```bash
    runnerkit byo-prepare --host user@host
    ```

    Installs `/etc/sudoers.d/runnerkit-installer` (mode 0440) with NOPASSWD only for the bootstrap command set. Validated with `visudo -c` before atomic rename. Idempotent — safe to re-run. Use `--remove` to revert.

    **Fallback — interactive password prompt** (no host-side preconfiguration):

    ```bash
    runnerkit up --repo owner/name --host user@host
    ```

    RunnerKit prompts locally for the sudo password during preflight. The password is registered with the redactor and zeroed after bootstrap returns. `--non-interactive` disables this fallback (use `runnerkit byo-prepare` for non-TTY contexts).

    See [docs/byo-quickstart.md#sudo-setup](../byo-quickstart.md#sudo-setup) for the full decision tree.
    ```

    Verify the placeholder text from Plan 06-05 ("Plan 06-06 lands the CLI surface for both", "Until Plan 06-06 lands, the v1.0.0 documented workaround is to add a NOPASSWD sudoers entry...") is FULLY REMOVED. The user-facing docs must not reference internal plan numbers.

    **Step 3.3 — Add README.md one-liner.** In `README.md`, find line 116 (`See [docs/byo-quickstart.md](docs/byo-quickstart.md) for prerequisites...`). Insert a new sibling line directly after it:

    ```markdown
    First-time BYO setup against a sudo-with-password host? See [Sudo Setup](docs/byo-quickstart.md#sudo-setup) — `runnerkit byo-prepare --host user@host` installs a scoped sudoers entry once, then every `runnerkit up` runs passwordlessly.
    ```

    **Step 3.4 — Verify the docs-anchor invariant still holds.** Run `go test ./internal/errcodes/...` to confirm `TestEveryCodeHasDocAnchor` and `TestEntriesFollowSymptomDiagnosisFix` still pass after the rkd-boot-015 entry edit. The Symptom + Diagnosis sections from Plan 06-05 should be untouched; only Fix changed.
  </action>
  <verify>
    <automated>go test ./internal/errcodes/... -count=1 -run 'TestEveryCodeHasDocAnchor|TestEntriesFollowSymptomDiagnosisFix|TestEachComponentHasMinimumOneEntry'</automated>
    <secondary>grep -n "## Sudo Setup" docs/byo-quickstart.md</secondary>
    <secondary>grep -n "runnerkit byo-prepare" docs/byo-quickstart.md docs/troubleshooting/bootstrap.md README.md</secondary>
    <secondary>grep -n "byo-quickstart.md#sudo-setup" README.md docs/troubleshooting/bootstrap.md</secondary>
    <secondary>! grep -n "Plan 06-06" docs/troubleshooting/bootstrap.md   # placeholder text MUST be gone</secondary>
    <secondary>! grep -n "Until Plan 06-06" docs/troubleshooting/bootstrap.md</secondary>
  </verify>
  <acceptance_criteria>
    - `docs/byo-quickstart.md` contains the literal heading `## Sudo Setup` AND the literal substring `runnerkit byo-prepare`. Position: between `## Safety warning` and `## Run setup`.
    - `docs/troubleshooting/bootstrap.md` rkd-boot-015 section's Fix subsection contains `runnerkit byo-prepare` AND links to `byo-quickstart.md#sudo-setup`.
    - `docs/troubleshooting/bootstrap.md` does NOT contain `Plan 06-06` or `Until Plan 06-06` anywhere (placeholder text from Plan 06-05 fully removed).
    - `README.md` contains `byo-quickstart.md#sudo-setup` (proves the cross-link landed).
    - `TestEveryCodeHasDocAnchor`, `TestEntriesFollowSymptomDiagnosisFix`, `TestEachComponentHasMinimumOneEntry` all green (the Symptom/Diagnosis structure is preserved).
    - The decision tree table renders as a Markdown table (visual check via the rendered version of byo-quickstart.md OR via a markdownlint pass if one is configured).
  </acceptance_criteria>
  <done>
    docs/byo-quickstart.md has Sudo Setup section with Path C → Path B decision tree; troubleshooting/bootstrap.md rkd-boot-015 entry references real implemented commands; README.md links to Sudo Setup. Plan 06-05's placeholder text is fully replaced.
  </done>
</task>

</tasks>

<verification>
After all three tasks:
1. `go test ./... -count=1 -race` — full suite green; new tests pass; existing tests not regressed by Prompter interface extension or new Severity (if added).
2. `go run ./cmd/runnerkit byo-prepare --help` — command discoverable, flags documented.
3. `go run ./cmd/runnerkit up --help` — `--non-interactive` flag still works (existing behavior preserved).
4. Manual sanity: `grep -n "runnerkit byo-prepare" docs/byo-quickstart.md docs/troubleshooting/bootstrap.md README.md` returns matches in all three files; `grep "Plan 06-06" docs/` returns no matches (placeholder text scrubbed).
5. Live verification (Plan 06-07): `runnerkit byo-prepare --host salar@mckee-small-desktop` succeeds against a fresh host; `runnerkit up --host salar@mckee-small-desktop --yes` then runs without password prompt; revert via `runnerkit byo-prepare --host ... --remove`.
</verification>

<success_criteria>
- Path B (interactive sudo password) AND Path C (`runnerkit byo-prepare`) both wired and tested per the gap doc 2026-05-04 user decision.
- All 14+ new tests pass green; no regressions in any existing package.
- Sudoers template is SCOPED (apt-get/dnf/yum/useradd/install/tar/systemctl/svc.sh ONLY) — NOT blanket NOPASSWD ALL.
- visudo -c validates BEFORE atomic rename (user can never be locked out of sudo).
- Sudo password is registered with `redact.SudoPassword` immediately on prompt; never leaks into state, logs, JSON, or error messages.
- `--non-interactive` fails fast with remediation pointing at `runnerkit byo-prepare` AND the RKD-BOOT-015 docs link.
- `runnerkit doctor` emits `byo_host_prepared` finding when sudoers entry present (info severity).
- DOC-04 docs are now self-consistent: byo-quickstart.md has Sudo Setup section, troubleshooting/bootstrap.md rkd-boot-015 references real commands, README.md links to it.
- Plan 06-07 (maintainer live smoke re-run) is now fully unblocked: BYO works end-to-end without manual sudoers preconfiguration via either Path B (prompt) or Path C (byo-prepare).
</success_criteria>

<output>
After completion, create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-06-byo-prepare-and-sudo-prompt-SUMMARY.md` documenting:
- Each task's RED + GREEN commits.
- The exact contents of `RenderSudoersEntry("alice")` (so future renames of the sudoers file path are auditable).
- Confirmation that visudo validation step is BEFORE atomic rename (the lockout-prevention guarantee).
- Confirmation that `redact.SudoPassword` registration happens in BOTH Path B (cli/up.go) AND Path C (cli/byo_prepare.go) emit sites.
- Confirmation that `go test ./... -count=1 -race` passes green AND `go run ./cmd/runnerkit byo-prepare --help` shows the new command.
- Updated note for Plan 06-07: the maintainer can now re-run live smoke against a fresh BYO host with EITHER Path B OR Path C and skip the manual `/etc/sudoers.d/runnerkit-smoke-temp` workaround used in attempt 1.
</output>
