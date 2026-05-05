---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 05
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/preflight/checks.go
  - internal/preflight/checks_test.go
  - internal/cli/up.go
  - internal/cli/up_test.go
  - internal/bootstrap/install.go
  - internal/bootstrap/install_test.go
  - internal/bootstrap/install_integration_test.go
  - internal/bootstrap/script.go
  - internal/bootstrap/script_test.go
  - internal/errcodes/codes.go
  - internal/errcodes/codes_test.go
  - docs/troubleshooting/bootstrap.md
  - scripts/smoke/byo-permission.sh
  - Makefile
autonomous: true
gap_closure: true
requirements: [REL-05]
must_haves:
  truths:
    - "Preflight rejects 'sudo binary present but password required' cleanly with a stable finding ID (CheckPrivilege now probes `sudo -n true` rather than checking only `probe.Commands[\"sudo\"]`)."
    - "When `bootstrap.Apply` returns a `RemoteError`, `internal/cli/up.go`'s `bootstrap_failed` message surfaces the underlying remote command stderr (redacted via `renderer.Redactor()`), not just the generic 'Review the remote host output' line."
    - "`download_runner` (and `RenderInstallScript`, `RenderEphemeralInstallScript`) write the runner tarball with consistent ownership: a fresh BYO host with NOPASSWD sudo + non-`runnerkit-runner` SSH user successfully runs `runnerkit up` end-to-end without the `curl: (23) Failure writing output to destination, Permission denied` failure observed in smoke attempt 1."
    - "An integration test (build-tag guarded) drives the actual download/verify/extract shell commands against a real shell with a tmpfs sandbox and proves the tarball + extracted `config.sh` land in a directory owned by the configured service user — closing the fakeExecutor-only test gap that hid Bug 2 since Plan 02-02."
    - "`scripts/smoke/byo-permission.sh` asserts the install dir contains `config.sh` after bootstrap apply (verifies the download landed before runner registration runs)."
    - "A new `RKD-BOOT-015` error code (`BootSudoPasswordRequired`) appears in `internal/errcodes/codes.go::Registry` and resolves to a stable anchor in `docs/troubleshooting/bootstrap.md`."
  artifacts:
    - path: "internal/preflight/checks.go"
      provides: "CheckPrivilege now executes a real `sudo -n true` probe against the SSH target with passwordless / password-required / not-in-sudoers / sudo-missing branches."
      contains: "sudo -n true"
      contains_also: "host.privilege.password_required"
      contains_also2: "host.privilege.no_sudo"
    - path: "internal/preflight/checks_test.go"
      provides: "Table tests covering all four sudo probe branches via fake remote.Executor."
      contains: "TestCheckPrivilege_PasswordRequired"
      contains_also: "TestCheckPrivilege_NotInSudoers"
      contains_also2: "TestCheckPrivilege_Passwordless"
      contains_also3: "TestCheckPrivilege_SudoMissing"
    - path: "internal/cli/up.go"
      provides: "bootstrap_failed and ephemeral bootstrap_failed messages now surface the underlying redacted remote stderr in the remediation list."
      contains: "renderer.Redactor()"
    - path: "internal/bootstrap/install.go"
      provides: "Apply + ApplyEphemeral download_runner step prefixes curl, sha256sum -c, and tar xzf with sudo so the install dir owned by serviceUser receives the tarball without 'Permission denied'."
      contains: "sudo curl"
      contains_also: "sudo sha256sum"
      contains_also2: "sudo tar xzf"
    - path: "internal/bootstrap/script.go"
      provides: "RenderInstallScript and RenderEphemeralInstallScript prefix curl, sha256sum -c, and tar xzf with sudo (mirrors install.go fix)."
      contains: "sudo curl"
      contains_also: "sudo sha256sum"
      contains_also2: "sudo tar xzf"
    - path: "internal/bootstrap/install_integration_test.go"
      provides: "Build-tag-guarded integration test that exercises real shell against a tmpfs sandbox to assert tarball + extraction land in the service-user-owned dir."
      contains: "//go:build integration"
      contains_also: "TestApply_DownloadRunner_RealShell"
    - path: "internal/errcodes/codes.go"
      provides: "New BootSudoPasswordRequired Code (RKD-BOOT-015) with title + docs anchor."
      contains: "RKD-BOOT-015"
      contains_also: "BootSudoPasswordRequired"
    - path: "docs/troubleshooting/bootstrap.md"
      provides: "New rkd-boot-015 entry with Symptom/Diagnosis/Fix structure for sudo password required failures."
      contains: "rkd-boot-015"
      contains_also: "RKD-BOOT-015"
    - path: "scripts/smoke/byo-permission.sh"
      provides: "Post-bootstrap assertion that the install dir contains config.sh on the remote host."
      contains: "config.sh"
    - path: "Makefile"
      provides: "New `test-integration` target that runs `go test -tags=integration ./internal/bootstrap/...`."
      contains: "test-integration:"
      contains_also: "-tags=integration"
  key_links:
    - from: "internal/preflight/checks.go::CheckPrivilege"
      to: "remote.Executor.Run with Script: 'sudo -n true'"
      via: "executor.Run(ctx, target, remote.Command{ID: \"probe_sudo_n\", Script: \"sudo -n true\"})"
      pattern: "sudo -n true"
    - from: "internal/cli/up.go bootstrap_failed branch"
      to: "remote command stderr (already captured in bootstrap.Result.Commands)"
      via: "renderer.Redactor().String(lastFailingResult.Stderr) appended to remediation slice"
      pattern: "Redactor\\(\\)\\.String"
    - from: "internal/bootstrap/install.go download_runner step"
      to: "Remote shell with sudo curl/sha256sum/tar prefixes"
      via: "Sprintf-rendered Script field with literal `sudo curl` / `sudo sha256sum` / `sudo tar`"
      pattern: "sudo (curl|sha256sum|tar)"
    - from: "internal/errcodes/codes.go::BootSudoPasswordRequired"
      to: "docs/troubleshooting/bootstrap.md#rkd-boot-015"
      via: "Code{File: \"bootstrap.md\", Anchor: \"rkd-boot-015\"} → TestEveryCodeHasDocAnchor verifies the link"
      pattern: "rkd-boot-015"
---

<objective>
Close the two BLOCKER bugs that made BYO bootstrap unusable in v1 (live smoke attempt 1, 2026-05-04 — see `06-GAP-byo-sudo-handling.md` Tasks A + E):

1. **Bug 1 (preflight false-positive)** — `internal/preflight/checks.go::CheckPrivilege` only checks whether the `sudo` binary exists on the host (`probe.Commands["sudo"]`). It never tests whether the SSH user can actually run sudo non-interactively. Hosts requiring a sudo password pass preflight, then bootstrap fails opaquely with `bootstrap_failed` while remote stderr is swallowed. Replace with a real `sudo -n true` probe; surface the underlying redacted remote stderr in the bootstrap_failed CLI error.

2. **Bug 2 (download_runner permission failure)** — `internal/bootstrap/install.go::Apply` (and `ApplyEphemeral`, and `script.go::RenderInstallScript` + `RenderEphemeralInstallScript`) creates the install directory owned by `runnerkit-runner` (mode 0755) via `sudo install -d -o`, then runs plain `curl`/`sha256sum -c -`/`tar xzf` *without* sudo as the SSH user. The SSH user has no write permission; curl fails with `(23) Failure writing output to destination, Permission denied`. Apply Option 1 from the gap doc (minimal diff): prefix `curl`, `sha256sum -c`, and `tar xzf` with `sudo`. Add a build-tag-guarded integration test that exercises the real shell so the fakeExecutor-only test gap (latent since Plan 02-02) is closed.

These two bugs together prevent ANY BYO bootstrap from completing on a fresh host in v1. Once this plan lands, Plan 06-06 (Path B + Path C user-facing features) and Plan 06-07 (maintainer live smoke re-run) become unblocked.

Implements gap doc `## Required work` Task A + Task E. Closes verification truth: "BYO bootstrap completes end-to-end against a real host without manual sudoers preconfiguration." (downstream of this plan plus 06-06's Path B/C fallback wiring).

Purpose: Restore Phase 6 success criterion 4 — "A fresh user can complete at least one supported setup path in about 10 minutes" — by making the BYO setup path actually executable end-to-end on real hosts.

Output: BYO bootstrap completes successfully against a real host with NOPASSWD sudo configured (Path A from the gap doc — the existing v1.0.0 "documented contract" workaround), with redacted remote stderr surfaced when failures DO occur, and a real-shell integration test pinned in CI.
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
@internal/preflight/checks.go
@internal/preflight/checks_test.go
@internal/bootstrap/install.go
@internal/bootstrap/install_test.go
@internal/bootstrap/script.go
@internal/bootstrap/script_test.go
@internal/cli/up.go
@internal/errcodes/codes.go
@internal/remote/executor.go
@docs/troubleshooting/bootstrap.md

<interfaces>
<!-- Key contracts the executor must integrate with. Extracted from current source. -->

remote.Executor (internal/remote/executor.go):
```go
type Executor interface {
    Probe(ctx context.Context, target Target) (ProbeResult, error)
    Run(ctx context.Context, target Target, command Command) (Result, error)
}
type Command struct {
    ID         string
    Script     string
    Sudo       bool
    Timeout    time.Duration
    Env        map[string]string
    RedactArgs []string
}
type Result struct {
    Stdout   string
    Stderr   string
    ExitCode int
}
```

preflight stable check IDs (internal/preflight/checks.go):
```go
const (
    CheckPrivilege  = "host.privilege"
    // (siblings: CheckSSHConnectivity, CheckSSHHostKey, CheckOSRelease, CheckArch, CheckSystemd, CheckDisk, CheckTools, CheckNetworkGitHub, CheckTime, CheckRunnerConflict)
)
type Result struct {
    Check       Check
    ID          string         // stable identifier — must extend with `host.privilege.password_required` and `host.privilege.no_sudo`
    Severity    Severity        // SeverityPass | SeverityWarning | SeverityFailure
    Message     string
    Remediation string
    Fixable     bool
}
```

bootstrap.Apply / ApplyEphemeral signature (internal/bootstrap/install.go):
```go
func Apply(ctx context.Context, exec remote.Executor, target remote.Target, opts Options) (Result, error)
func ApplyEphemeral(ctx context.Context, exec remote.Executor, target remote.Target, opts Options) (Result, error)
type Result struct {
    Commands []remote.Result   // stderr of the failing command lives here
}
```

bootstrap script renderers (internal/bootstrap/script.go):
```go
func RenderInstallScript(opts Options) string         // persistent — line 35-48 of script.go
func RenderEphemeralInstallScript(opts Options) string // ephemeral — line 65-84 of script.go
```

errcodes pattern (internal/errcodes/codes.go):
```go
type Code struct {
    ID       string  // "RKD-BOOT-015"
    Severity Severity
    Title    string
    File     string  // "bootstrap.md"
    Anchor   string  // "rkd-boot-015"
}
// Append to var Registry slice; TestEveryCodeHasDocAnchor walks Registry and verifies docs/<File>#<Anchor> exists.
// BOOT prefix already in use; current highest is RKD-BOOT-014; next free is RKD-BOOT-015.
```

renderer redactor (internal/ui/, used in cli/up.go):
```go
renderer := newRenderer(...)              // already constructed at top of runUp
renderer.Redactor().String(stderr)        // redacts before display; preserves tokens
```

CURRENT bootstrap line 74 (download_runner — the bug):
```go
{ID: "download_runner", Script: fmt.Sprintf("set -euo pipefail\nsudo install -d -o %s -g %s %s\ncd %s\ncurl -fL --retry 3 --connect-timeout 10 -o %s %s\nprintf '%%s  %%s\n' '%s' '%s' | sha256sum -c -\ntar xzf %s --skip-old-files\n", opts.ServiceUser, opts.ServiceUser, opts.InstallPath, opts.InstallPath, opts.Package.Filename, opts.Package.URL, opts.Package.SHA256, opts.Package.Filename, opts.Package.Filename), Sudo: true},
```
The fix: prefix `curl`, the `sha256sum -c -` invocation, and `tar xzf` with `sudo`. Same fix in line 115 (ApplyEphemeral).

CURRENT script.go RenderInstallScript lines 41-45 (the bug):
```
cd %[2]s
if [ ! -f %[4]s ]; then
  curl -fL --retry 3 --connect-timeout 10 -o %[4]s %[5]s
fi
printf '%%s  %%s\n' '%[6]s' '%[4]s' | sha256sum -c -
tar xzf %[4]s --skip-old-files
```
Same fix: prefix curl/sha256sum/tar with sudo. Same fix in RenderEphemeralInstallScript lines 71-81.

CURRENT internal/cli/up.go line 224 (the swallowed-stderr bug):
```go
_ = renderer.Error("bootstrap_failed", "RunnerKit could not apply the BYO runner install plan.", []string{"Review the remote host output, fix the issue, and re-run runnerkit up."})
return NewExitError(ExitSafetyGate, err)
```
The fix: extract last command's stderr from the bootstrap.Result returned by Apply, redact it via renderer.Redactor().String(), and append it to the remediation slice (after the existing "Review" line). Same fix at line ~214 for the ephemeral branch.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Real `sudo -n true` preflight probe + redacted-stderr surfacing on bootstrap_failed</name>
  <files>internal/preflight/checks.go, internal/preflight/checks_test.go, internal/cli/up.go, internal/cli/up_test.go, internal/errcodes/codes.go, internal/errcodes/codes_test.go, docs/troubleshooting/bootstrap.md</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md (the spec — Task A pseudocode at lines 124-152; Surface remote stderr requirement at lines 150-152)
    - internal/preflight/checks.go (CURRENT CheckPrivilege at line 127 — only `probe.Commands["sudo"]` check)
    - internal/preflight/checks_test.go (existing fakePreflightExecutor pattern — lines 10-17; passingProbe helper at line 71)
    - internal/cli/up.go (bootstrap_failed branch at line 224; ephemeral branch at line 214; renderer construction at line 96)
    - internal/errcodes/codes.go (BOOT block at lines 59-72; Registry slice at lines 114-127; numbering rule comment at lines 6-12)
    - docs/troubleshooting/bootstrap.md (existing rkd-boot-002 entry as the Symptom/Diagnosis/Fix template at lines 9-44)
    - internal/remote/executor.go (Command and Result types — for understanding what the probe sends and what stderr looks like)
  </read_first>
  <behavior>
    - Test 1 (`TestCheckPrivilege_Passwordless`): fake executor returns `Result{ExitCode: 0}` for `Script: "sudo -n true"` → report contains a SeverityPass result with `ID == "host.privilege"` and message mentioning "passwordless sudo".
    - Test 2 (`TestCheckPrivilege_PasswordRequired`): fake executor returns `Result{ExitCode: 1, Stderr: "sudo: a password is required"}` → report contains a SeverityWarning (NOT failure) result with `ID == "host.privilege.password_required"` and remediation referencing `runnerkit byo-prepare` AND interactive password prompt fallback. Warning (not failure) so Path B can take over in Plan 06-06; report.Passed() must still return true so the bootstrap path is reachable for Path B fallback.
    - Test 3 (`TestCheckPrivilege_NotInSudoers`): fake executor returns `Result{ExitCode: 1, Stderr: "user alice may not run sudo on host"}` → report contains a SeverityFailure result with `ID == "host.privilege.no_sudo"` and remediation "Add the SSH user to sudoers or pick a host where they are."
    - Test 4 (`TestCheckPrivilege_SudoMissing`): probe.Commands["sudo"] = false (sudo binary not installed) — bypasses the probe entirely → report contains a SeverityFailure result with `ID == "host.privilege"` and the existing "sudo is required for setup commands." remediation. Existing fallback path; preserved for backward compatibility.
    - Test 5 (`TestUp_BootstrapFailed_SurfacesRedactedRemoteStderr`): table-driven — when bootstrap.Apply returns a non-nil error AND the bootstrap.Result.Commands slice contains a failing entry with non-empty Stderr containing a token-shaped string `"ghp_secrettoken12345"`, the renderer.Error remediation slice for `bootstrap_failed` includes a line containing `<redacted:github-token>` (NOT the raw token) AND the original failing command ID. Verifies both the surfacing (no longer swallowed) AND the redaction invariant (Phase 1 contract preserved).
    - Test 6 (`TestEveryCodeHasDocAnchor` already exists in errcodes_test.go) must continue to pass with the new RKD-BOOT-015 entry — implies docs/troubleshooting/bootstrap.md MUST contain `<a name="rkd-boot-015"></a>` and a Symptom/Diagnosis/Fix block for the new code.
  </behavior>
  <action>
    **Step 1.1 — Add new preflight result IDs.** In `internal/preflight/checks.go`, add two new exported const stable IDs alongside the existing `CheckPrivilege`:
    ```go
    const (
        // ... existing ...
        CheckPrivilege               = "host.privilege"
        CheckPrivilegePasswordReq    = "host.privilege.password_required"
        CheckPrivilegeNoSudo         = "host.privilege.no_sudo"
        // ... existing ...
    )
    ```

    **Step 1.2 — Replace binary-existence check with real probe.** In `internal/preflight/checks.go::Run`, replace the existing block at lines 127-131:
    ```go
    if probe.Commands["sudo"] {
        report.Results = append(report.Results, pass(CheckPrivilege, "sudo is available for setup commands."))
    } else {
        report.Results = append(report.Results, failure(CheckPrivilege, "sudo is required for setup commands.", "Grant sudo for installation or use a host where sudo is available."))
    }
    ```
    with the real probe per gap doc Task A pseudocode (lines 124-143):
    ```go
    if !probe.Commands["sudo"] {
        report.Results = append(report.Results, failure(CheckPrivilege, "sudo is required for setup commands.", "Grant sudo for installation or use a host where sudo is available."))
    } else {
        result, err := executor.Run(ctx, target, remote.Command{ID: "probe_sudo_n", Script: "sudo -n true"})
        switch {
        case err == nil && result.ExitCode == 0:
            report.Results = append(report.Results, pass(CheckPrivilege, "Passwordless sudo available for setup commands."))
        case err == nil && (strings.Contains(result.Stderr, "password is required") || strings.Contains(result.Stderr, "a terminal is required")):
            report.Results = append(report.Results, warning(CheckPrivilegePasswordReq, "sudo requires a password — RunnerKit will prompt or use byo-prepare.", "Run `runnerkit byo-prepare --host user@host` to install a scoped sudoers entry, or re-run `runnerkit up` interactively to be prompted for the sudo password."))
        case err == nil && strings.Contains(result.Stderr, "may not run sudo"):
            report.Results = append(report.Results, failure(CheckPrivilegeNoSudo, "User is not in sudoers on the remote host.", "Add the SSH user to sudoers or pick a host where they are."))
        default:
            // Probe failed for an unexpected reason; treat as a failure but include the stderr for diagnosis.
            stderr := strings.TrimSpace(result.Stderr)
            if stderr == "" && err != nil {
                stderr = err.Error()
            }
            report.Results = append(report.Results, failure(CheckPrivilege, "sudo probe failed: "+stderr, "Verify SSH access and that sudo is installed on the host."))
        }
    }
    ```
    Note: `strings` is already imported. The password_required path emits a WARNING (not failure) so `report.Passed()` returns true and the bootstrap path runs — Plan 06-06 Path B will then prompt for the password. For this plan (where Path B is not yet wired), the maintainer's documented v1.0.0 contract is NOPASSWD sudo, so the warning is the correct severity to allow the existing happy path to proceed.

    **Step 1.3 — Add the new RKD code.** In `internal/errcodes/codes.go`, append to the BOOT block (after BootRunnerOnlineVerifyTimeout at line 72):
    ```go
    BootSudoPasswordRequired = Code{ID: "RKD-BOOT-015", Severity: SeverityWarning, Title: "Remote sudo requires password — bootstrap needs scoped sudoers or interactive prompt", File: "bootstrap.md", Anchor: "rkd-boot-015"}
    ```
    Append `BootSudoPasswordRequired` to the Registry slice at line 117 (BOOT row). Numbering rule from package doc: append, never renumber; RKD-BOOT-015 is the next free integer after RKD-BOOT-014.

    **Step 1.4 — Add the docs entry.** In `docs/troubleshooting/bootstrap.md`, append at the end of the file:
    ```markdown
    ***

    <a name="rkd-boot-015"></a>
    ## RKD-BOOT-015: Remote sudo requires password

    **Severity:** warning
    **Component:** bootstrap

    ### Symptom

    `runnerkit up --host user@host` warns during preflight:

    ```
    [warning] host.privilege.password_required: sudo requires a password — RunnerKit will prompt or use byo-prepare
    ```

    Or the bootstrap step fails with `bootstrap_failed` and the surfaced remote stderr contains `sudo: a password is required` / `sudo: a terminal is required`.

    ### Diagnosis

    The SSH user can run `sudo` but only after entering their password. RunnerKit's bootstrap commands run over a non-interactive SSH channel and cannot answer a sudo prompt, so the very first sudo-prefixed command fails.

    ### Fix

    Two recommended paths (Plan 06-06 lands the CLI surface for both):

    1. **Recommended — `runnerkit byo-prepare`** (one-time scoped sudoers entry):
       ```bash
       runnerkit byo-prepare --host user@host
       ```
    2. **Fallback — interactive password prompt** (no host-side preconfiguration):
       ```bash
       runnerkit up --repo owner/name --host user@host
       # RunnerKit prompts for the host's sudo password locally; never logged or stored.
       ```

    Until Plan 06-06 lands, the v1.0.0 documented workaround is to add a NOPASSWD sudoers entry for the SSH user manually:

    ```bash
    ssh user@host 'echo "user ALL=(root) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/runnerkit-temporary && sudo chmod 0440 /etc/sudoers.d/runnerkit-temporary'
    ```

    ***
    ```

    Update the section header at the top of `docs/troubleshooting/bootstrap.md` line 3 from "Stable codes for this component: `RKD-BOOT-002`..`RKD-BOOT-014`" to "Stable codes for this component: `RKD-BOOT-002`..`RKD-BOOT-015`".

    **Step 1.5 — Surface redacted remote stderr in bootstrap_failed.** In `internal/cli/up.go`, modify both bootstrap_failed branches (line 224 for persistent, line 214 for ephemeral). Capture the bootstrap.Result.Commands slice and extract the last command's stderr if it indicates failure. Pseudocode for the persistent branch (line 218 onwards):
    ```go
    } else {
        result, err := bootstrap.Apply(ctx, deps.RemoteExecutor, target, bootstrapOpts)
        if err != nil {
            var serviceErr bootstrap.ServiceNotActiveError
            if errors.As(err, &serviceErr) {
                _ = renderer.Error("runner_service_not_active", ...)
                return NewExitError(ExitSafetyGate, err)
            }
            remediation := []string{"Review the remote host output, fix the issue, and re-run runnerkit up."}
            if stderr := lastCommandStderr(result); stderr != "" {
                remediation = append(remediation, "Remote stderr: "+renderer.Redactor().String(stderr))
            }
            _ = renderer.Error("bootstrap_failed", "RunnerKit could not apply the BYO runner install plan.", remediation)
            return NewExitError(ExitSafetyGate, err)
        }
    }
    ```
    Add a helper at the bottom of `internal/cli/up.go`:
    ```go
    // lastCommandStderr extracts the trailing command's stderr from a
    // bootstrap.Result for surfacing in user-facing error messages. It
    // returns "" if no command was recorded or no stderr was captured.
    func lastCommandStderr(result bootstrap.Result) string {
        if len(result.Commands) == 0 {
            return ""
        }
        return strings.TrimSpace(result.Commands[len(result.Commands)-1].Stderr)
    }
    ```
    Apply the same pattern to the ephemeral branch (line 208-216): replace `if _, err := bootstrap.ApplyEphemeral(...)` with `result, err := bootstrap.ApplyEphemeral(...)` and surface `lastCommandStderr(result)` via the redactor.

    **Step 1.6 — TDD cycle.** Write `internal/preflight/checks_test.go` tests 1-4 BEFORE implementing the probe (RED commit), confirm they fail, then implement (GREEN commit). For test 5, extend `internal/cli/up_test.go` (look at existing patterns there for how `runUp` is tested with a fake remote.Executor and a recording renderer) — the test should drive `bootstrap.Apply` to return an error AND a Result whose last command has non-empty Stderr containing a token shape, then assert the renderer's captured Error remediation slice includes the redacted form.
  </action>
  <verify>
    <automated>go test ./internal/preflight/... ./internal/cli/... ./internal/errcodes/... -count=1 -run 'TestCheckPrivilege_PasswordRequired|TestCheckPrivilege_NotInSudoers|TestCheckPrivilege_Passwordless|TestCheckPrivilege_SudoMissing|TestUp_BootstrapFailed_SurfacesRedactedRemoteStderr|TestEveryCodeHasDocAnchor|TestCodesAreUnique'</automated>
    <secondary>grep -n "sudo -n true" internal/preflight/checks.go</secondary>
    <secondary>grep -n "RKD-BOOT-015" internal/errcodes/codes.go</secondary>
    <secondary>grep -n "rkd-boot-015" docs/troubleshooting/bootstrap.md</secondary>
    <secondary>grep -n "renderer.Redactor()" internal/cli/up.go</secondary>
  </verify>
  <acceptance_criteria>
    - `internal/preflight/checks.go` contains the literal substring `sudo -n true` (proves the new probe is wired).
    - `internal/preflight/checks.go` contains both `host.privilege.password_required` and `host.privilege.no_sudo` (proves the new stable IDs are defined).
    - `internal/preflight/checks_test.go` contains test functions named `TestCheckPrivilege_Passwordless`, `TestCheckPrivilege_PasswordRequired`, `TestCheckPrivilege_NotInSudoers`, `TestCheckPrivilege_SudoMissing`. All four pass green.
    - `internal/cli/up.go` contains the substring `renderer.Redactor()` somewhere AFTER the bootstrap_failed branch (proves stderr surfacing is wired AND goes through the redactor).
    - `internal/cli/up_test.go` contains a test named `TestUp_BootstrapFailed_SurfacesRedactedRemoteStderr` that passes green AND asserts the redacted token (`<redacted:github-token>` or similar) appears in the captured remediation, AND asserts the raw token does NOT appear.
    - `internal/errcodes/codes.go` Registry slice (lines ~114-127) includes `BootSudoPasswordRequired` (the var name); the exported var has `ID: "RKD-BOOT-015"` and `File: "bootstrap.md"` and `Anchor: "rkd-boot-015"`.
    - `docs/troubleshooting/bootstrap.md` contains both `<a name="rkd-boot-015"></a>` and `## RKD-BOOT-015:` headings AND a Symptom/Diagnosis/Fix section structure (validated indirectly by `TestEntriesFollowSymptomDiagnosisFix` in errcodes_test.go remaining green).
    - `TestEveryCodeHasDocAnchor` and `TestCodesAreUnique` (existing in `internal/errcodes/codes_test.go`) remain green with the new RKD-BOOT-015 entry.
    - Full Go test suite (`go test ./... -count=1`) remains green — no regressions in other packages.
  </acceptance_criteria>
  <done>
    Preflight rejects sudo-with-password hosts cleanly with a stable warning ID; bootstrap_failed messages now include the underlying redacted remote stderr; new RKD-BOOT-015 code anchored in docs.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Sudo-prefixed download_runner + script renderers + real-shell integration test + smoke assertion</name>
  <files>internal/bootstrap/install.go, internal/bootstrap/install_test.go, internal/bootstrap/install_integration_test.go, internal/bootstrap/script.go, internal/bootstrap/script_test.go, scripts/smoke/byo-permission.sh, Makefile</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md (the spec — Task E at lines 235-257; Bug 2 root cause at lines 62-93; Option 1 fix at lines 102-110)
    - internal/bootstrap/install.go (CURRENT lines 71-78 — Apply commands; line 74 download_runner with the bug; CURRENT lines 112-121 — ApplyEphemeral commands; line 115 same bug)
    - internal/bootstrap/install_test.go (recordingExecutor pattern at lines 11-19; existing test names + assertions at lines 21-78 — they assert command ID order and presence; substring-match patterns will need `sudo curl`/`sudo tar` updates)
    - internal/bootstrap/script.go (CURRENT RenderInstallScript lines 29-49 with the bug at lines 41-45; CURRENT RenderEphemeralInstallScript lines 65-85 with the same bug at lines 71-81)
    - internal/bootstrap/script_test.go (TestRenderInstallAndServiceScripts pattern at lines 8-32 — substring-match assertions; same for TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken at lines 41-71)
    - scripts/smoke/byo-permission.sh (existing structure — extension point per gap doc Task E lines 247-249)
    - Makefile (existing test target at line 11; pattern for adding test-integration target)
  </read_first>
  <behavior>
    - Test 1 (`TestApplyDownloadRunnerCommandUsesSudoForCurlSha256SumTar`): existing recordingExecutor captures Apply's commands; assert the `download_runner` step's Script contains `sudo curl`, `sudo sha256sum -c -`, AND `sudo tar xzf` (NOT plain `curl`/`sha256sum`/`tar`). Same assertion for ApplyEphemeral via `TestApplyEphemeralDownloadRunnerCommandUsesSudoForCurlSha256SumTar`.
    - Test 2 (`TestRenderInstallScriptUsesSudoForCurlSha256SumTar`): script.go's RenderInstallScript output contains `sudo curl`, `sudo sha256sum`, `sudo tar xzf`. Same for `TestRenderEphemeralInstallScriptUsesSudoForCurlSha256SumTar`.
    - Test 3 — INTEGRATION (`TestApply_DownloadRunner_RealShell` in `install_integration_test.go` with `//go:build integration`): create a `t.TempDir()` sandbox; serve a fake runner tarball over an `httptest.Server`; build an Options with `InstallPath: filepath.Join(tmpdir, "install")` and `Package: RunnerPackage{URL: server.URL+"/runner.tgz", Filename: "runner.tgz", SHA256: <computed>}`; use a `shellExecutor` (defined in the test file) that actually invokes `bash -c` for each `remote.Command` with the script piped to stdin. Assert: (a) `<install_path>/runner.tgz` exists, (b) `<install_path>/config.sh` exists after extraction (the tarball must contain a fake `config.sh` so this assertion is meaningful), (c) `Apply` returns nil error. The test runs with the SSH-user equivalent of the local OS user; sudo prefix means `sudo` is invoked locally, so the test machine must have NOPASSWD sudo configured OR the test must use a `sudo` shim. To avoid requiring NOPASSWD on CI, the integration test sets the `Sudo` field to false via a custom Options helper AND uses `t.Skip("integration test requires NOPASSWD sudo or sudo shim — set RUNNERKIT_INTEGRATION=1 to run")` when `os.Getenv("RUNNERKIT_INTEGRATION") == ""`. The build tag ensures it never runs in `go test ./...`.
    - Test 4 (existing tests in install_test.go and script_test.go must continue to pass with substring assertions updated for `sudo curl` / `sudo tar` per Option 1 — these are auto-fix updates inside the same task).
  </behavior>
  <action>
    **Step 2.1 — Fix download_runner in install.go.** In `internal/bootstrap/install.go`, modify the `download_runner` Command at line 74. Replace the current Sprintf:
    ```go
    Script: fmt.Sprintf("set -euo pipefail\nsudo install -d -o %s -g %s %s\ncd %s\ncurl -fL --retry 3 --connect-timeout 10 -o %s %s\nprintf '%%s  %%s\n' '%s' '%s' | sha256sum -c -\ntar xzf %s --skip-old-files\n", opts.ServiceUser, opts.ServiceUser, opts.InstallPath, opts.InstallPath, opts.Package.Filename, opts.Package.URL, opts.Package.SHA256, opts.Package.Filename, opts.Package.Filename)
    ```
    with (note `sudo` prefixes added to curl, sha256sum -c -, and tar xzf):
    ```go
    Script: fmt.Sprintf("set -euo pipefail\nsudo install -d -o %s -g %s %s\ncd %s\nsudo curl -fL --retry 3 --connect-timeout 10 -o %s %s\nprintf '%%s  %%s\n' '%s' '%s' | sudo sha256sum -c -\nsudo tar xzf %s --skip-old-files\n", opts.ServiceUser, opts.ServiceUser, opts.InstallPath, opts.InstallPath, opts.Package.Filename, opts.Package.URL, opts.Package.SHA256, opts.Package.Filename, opts.Package.Filename)
    ```
    Apply the IDENTICAL substitution to line 115 (ApplyEphemeral's `download_runner`).

    **Step 2.2 — Fix RenderInstallScript and RenderEphemeralInstallScript in script.go.** In `internal/bootstrap/script.go::RenderInstallScript` (lines 35-48), modify the format string:
    ```
    cd %[2]s
    if [ ! -f %[4]s ]; then
      curl -fL --retry 3 --connect-timeout 10 -o %[4]s %[5]s
    fi
    printf '%%s  %%s\n' '%[6]s' '%[4]s' | sha256sum -c -
    tar xzf %[4]s --skip-old-files
    ```
    becomes (sudo prefixes on curl, sha256sum, tar):
    ```
    cd %[2]s
    if [ ! -f %[4]s ]; then
      sudo curl -fL --retry 3 --connect-timeout 10 -o %[4]s %[5]s
    fi
    printf '%%s  %%s\n' '%[6]s' '%[4]s' | sudo sha256sum -c -
    sudo tar xzf %[4]s --skip-old-files
    ```
    Apply IDENTICAL substitution to `RenderEphemeralInstallScript` at lines 71-81 of script.go.

    **Step 2.3 — Update existing substring assertions.** In `internal/bootstrap/script_test.go` line 20 add `"sudo curl"`, `"sudo sha256sum"`, `"sudo tar xzf"` to the wanted substring slice (in addition to any existing entries that should remain). Same for `TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken` at line 55. Use grep to locate any other tests that previously asserted plain `curl `/`tar xzf` and update them. NB: `sha256sum -c -` already in existing assertion — change to `sudo sha256sum -c -`.

    **Step 2.4 — Add new install_test.go assertions.** Add two new test functions to `internal/bootstrap/install_test.go`:
    ```go
    func TestApplyDownloadRunnerCommandUsesSudoForCurlSha256SumTar(t *testing.T) {
        exec := &recordingExecutor{}
        opts := Options{RunnerName: "runnerkit-owner-repo", RepoURL: "https://github.com/owner/repo", Labels: []string{"x"}, ServiceUser: "runnerkit-runner", RunnerToken: "registration-token-x", Package: RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc"}}
        if _, err := Apply(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, opts); err != nil {
            t.Fatalf("Apply returned error: %v", err)
        }
        var dl remote.Command
        for _, c := range exec.commands {
            if c.ID == "download_runner" {
                dl = c
                break
            }
        }
        for _, want := range []string{"sudo curl", "sudo sha256sum -c -", "sudo tar xzf"} {
            if !strings.Contains(dl.Script, want) {
                t.Fatalf("download_runner script missing %q:\n%s", want, dl.Script)
            }
        }
    }
    ```
    Add a parallel `TestApplyEphemeralDownloadRunnerCommandUsesSudoForCurlSha256SumTar` that calls `ApplyEphemeral` with `Mode: "ephemeral"` and applies the same assertions.

    **Step 2.5 — Add real-shell integration test.** Create `internal/bootstrap/install_integration_test.go` with the build tag at the top:
    ```go
    //go:build integration

    package bootstrap

    import (
        "archive/tar"
        "compress/gzip"
        "context"
        "crypto/sha256"
        "encoding/hex"
        "fmt"
        "net/http"
        "net/http/httptest"
        "os"
        "os/exec"
        "path/filepath"
        "strings"
        "testing"

        "github.com/accidentally-awesome-labs/runnerkit/internal/remote"
    )

    // shellExecutor invokes commands via a local bash shell. It is gated to
    // the integration build tag and only used to prove that real-shell
    // semantics match the renderer expectations. Sudo-prefixed commands
    // require NOPASSWD sudo on the test machine (or a sudo shim).
    type shellExecutor struct{ workingDir string }

    func (s *shellExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
        return remote.ProbeResult{}, nil
    }

    func (s *shellExecutor) Run(_ context.Context, _ remote.Target, c remote.Command) (remote.Result, error) {
        cmd := exec.Command("bash", "-c", c.Script)
        cmd.Dir = s.workingDir
        out, err := cmd.CombinedOutput()
        result := remote.Result{Stdout: string(out), Stderr: ""}
        if exitErr, ok := err.(*exec.ExitError); ok {
            result.ExitCode = exitErr.ExitCode()
            result.Stderr = string(out)
        } else if err != nil {
            return result, err
        }
        return result, nil
    }

    func TestApply_DownloadRunner_RealShell(t *testing.T) {
        if os.Getenv("RUNNERKIT_INTEGRATION") == "" {
            t.Skip("set RUNNERKIT_INTEGRATION=1 to run; requires NOPASSWD sudo on the test machine")
        }
        // Build a minimal fake tarball containing a config.sh file.
        tmp := t.TempDir()
        tarballPath := filepath.Join(tmp, "fake-runner.tgz")
        sha := buildFakeRunnerTarball(t, tarballPath)

        // Serve it over httptest.
        server := httptest.NewServer(http.FileServer(http.Dir(tmp)))
        defer server.Close()

        installPath := filepath.Join(tmp, "install")
        opts := Options{
            RunnerName:  "runnerkit-it-test",
            RepoURL:     "https://github.com/owner/repo",
            Labels:      []string{"self-hosted"},
            InstallPath: installPath,
            WorkDir:     filepath.Join(tmp, "work"),
            ServiceUser: os.Getenv("USER"), // run as current user so sudo install -d -o $USER works
            RunnerToken: "registration-token-itest",
            Package:     RunnerPackage{Filename: "fake-runner.tgz", URL: server.URL + "/fake-runner.tgz", SHA256: sha},
        }

        // Inject a synthetic "configure_runner" exit-0 by stopping after download_runner.
        // Run only the first 3 commands (fix_dependencies, create_runner_user, download_runner)
        // by intercepting; the remaining steps require a real systemd which is out of scope here.
        exec := &shellExecutor{workingDir: tmp}
        // Build commands inline by calling Apply but stopping after download_runner.
        // For test simplicity: invoke the rendered command sequence directly.
        // We exercise the download_runner script by calling exec.Run with the same
        // Script that Apply would produce.
        cmds := buildDownloadRunnerCommandForTest(opts) // helper added in install.go (NOT exported in production; use a test helper file in same package)
        result, err := exec.Run(context.Background(), remote.Target{}, cmds)
        if err != nil || result.ExitCode != 0 {
            t.Fatalf("download_runner shell exec failed: exit=%d err=%v\nstdout/stderr:\n%s", result.ExitCode, err, result.Stdout)
        }

        // Assert tarball + extracted config.sh landed in installPath.
        if _, err := os.Stat(filepath.Join(installPath, "fake-runner.tgz")); err != nil {
            t.Fatalf("tarball not found at %s: %v", installPath, err)
        }
        if _, err := os.Stat(filepath.Join(installPath, "config.sh")); err != nil {
            t.Fatalf("extracted config.sh not found at %s: %v", installPath, err)
        }
    }

    // buildFakeRunnerTarball creates a minimal .tar.gz containing a single
    // config.sh file at the archive root, returns its SHA-256 hex digest.
    func buildFakeRunnerTarball(t *testing.T, path string) string {
        t.Helper()
        f, err := os.Create(path)
        if err != nil {
            t.Fatalf("create tarball: %v", err)
        }
        gzw := gzip.NewWriter(f)
        tw := tar.NewWriter(gzw)
        body := []byte("#!/bin/bash\necho fake config.sh\n")
        hdr := &tar.Header{Name: "config.sh", Mode: 0755, Size: int64(len(body))}
        if err := tw.WriteHeader(hdr); err != nil {
            t.Fatalf("write tar header: %v", err)
        }
        if _, err := tw.Write(body); err != nil {
            t.Fatalf("write tar body: %v", err)
        }
        if err := tw.Close(); err != nil {
            t.Fatalf("close tar: %v", err)
        }
        if err := gzw.Close(); err != nil {
            t.Fatalf("close gzip: %v", err)
        }
        if err := f.Close(); err != nil {
            t.Fatalf("close file: %v", err)
        }
        // Compute sha256.
        f2, _ := os.Open(path)
        defer f2.Close()
        h := sha256.New()
        if _, err := io.Copy(h, f2); err != nil {
            t.Fatalf("hash: %v", err)
        }
        return hex.EncodeToString(h.Sum(nil))
    }

    _ = strings.Contains // keep imports stable across edits; remove if unused
    _ = fmt.Sprintf
    ```
    AND add a non-build-tagged helper file `internal/bootstrap/install_test_helpers.go` with the exported helper `BuildDownloadRunnerCommandForTest(opts Options) remote.Command` that returns the same `remote.Command` literal that `Apply` emits at line 74 (factor out the literal so the integration test re-uses it; the same factoring also de-duplicates the literal between Apply and ApplyEphemeral so future renderer edits stay in one place). Specifically: extract the `download_runner` Command construction into a private helper `downloadRunnerCommand(opts Options) remote.Command` in install.go; the integration test calls a thin exported wrapper.

    **Step 2.6 — Extend scripts/smoke/byo-permission.sh.** After the `runnerkit up` invocation, add:
    ```bash
    echo "===> [smoke-byo] Asserting install dir contains config.sh on the remote host"
    ssh "${HOST}" 'sudo test -f /opt/actions-runner/runnerkit-*/config.sh' || {
      echo "FAIL: config.sh not found in /opt/actions-runner/runnerkit-*/ — bootstrap did not land the tarball"
      exit 3
    }
    ```
    Place this assertion BEFORE the `runnerkit status` line so failure is attributed to bootstrap not registration.

    **Step 2.7 — Add Makefile test-integration target.** After the existing `test-race` target, add:
    ```makefile
    test-integration: ## Run integration tests (requires NOPASSWD sudo on local machine; gated by RUNNERKIT_INTEGRATION=1).
    	RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -v
    ```
    Add `test-integration` to the `.PHONY` line at the top of the Makefile (line 4).

    **Step 2.8 — TDD cycle.** Write all three test changes (Steps 2.3 + 2.4 + 2.5) BEFORE the production fix in Steps 2.1 + 2.2 (RED commit), confirm RED for tests 1+2 (existing tests already fail with old plain-curl scripts? — actually existing test in script_test.go line 20 asserts `"sha256sum -c -"` which still substring-matches `"sudo sha256sum -c -"`, so existing tests stay green; the NEW assertions in Steps 2.3-2.4 are the RED). Then GREEN by Steps 2.1 + 2.2.
  </action>
  <verify>
    <automated>go test ./internal/bootstrap/... -count=1 -run 'TestApplyDownloadRunnerCommandUsesSudoForCurlSha256SumTar|TestApplyEphemeralDownloadRunnerCommandUsesSudoForCurlSha256SumTar|TestRenderInstallScriptUsesSudoForCurlSha256SumTar|TestRenderEphemeralInstallScriptUsesSudoForCurlSha256SumTar|TestRenderInstallAndServiceScripts|TestApplyRunsBootstrapCommandsInOrderAndRedactsToken|TestApplyEphemeralRunsCommandsInOrderRedactsTokenAndAvoidsSvcSh'</automated>
    <secondary>go test ./... -count=1   # full regression — ensure no other test breaks on `sudo curl`/`sudo tar` substring drift</secondary>
    <secondary>grep -n "sudo curl" internal/bootstrap/install.go internal/bootstrap/script.go</secondary>
    <secondary>grep -n "sudo tar xzf" internal/bootstrap/install.go internal/bootstrap/script.go</secondary>
    <secondary>grep -n "sudo sha256sum" internal/bootstrap/install.go internal/bootstrap/script.go</secondary>
    <secondary>test -f internal/bootstrap/install_integration_test.go && head -1 internal/bootstrap/install_integration_test.go   # verify build tag is on line 1</secondary>
    <secondary>grep -n "config.sh" scripts/smoke/byo-permission.sh   # verify the smoke assertion landed</secondary>
    <secondary>grep -n "test-integration" Makefile   # verify the new target landed</secondary>
  </verify>
  <acceptance_criteria>
    - `internal/bootstrap/install.go` `download_runner` step (in BOTH Apply at line 74 area AND ApplyEphemeral at line 115 area) contains the literal substrings `sudo curl`, `sudo sha256sum -c -`, AND `sudo tar xzf`. NO bare `curl ` (with trailing space) or bare `tar xzf` (without preceding `sudo `) remains in download_runner.
    - `internal/bootstrap/script.go` `RenderInstallScript` AND `RenderEphemeralInstallScript` outputs contain `sudo curl`, `sudo sha256sum`, `sudo tar xzf`. (Verified by grep AND by `TestRenderInstallScriptUsesSudoForCurlSha256SumTar` / `TestRenderEphemeralInstallScriptUsesSudoForCurlSha256SumTar`.)
    - `internal/bootstrap/install_integration_test.go` exists with `//go:build integration` as line 1, defines `TestApply_DownloadRunner_RealShell`, AND uses `httptest.NewServer` + `t.TempDir()` + an `exec.Command("bash", "-c", ...)` shell executor. The test skips cleanly when `RUNNERKIT_INTEGRATION` env var is unset; when set AND NOPASSWD sudo is available, the test passes and asserts `config.sh` extracted into `installPath`.
    - `make test-integration` runs `go test -tags=integration ./internal/bootstrap/... -count=1 -v` — verified by `grep "test-integration" Makefile`.
    - `scripts/smoke/byo-permission.sh` contains the new `config.sh` assertion (`grep -n "config.sh" scripts/smoke/byo-permission.sh` returns at least one match in the new ssh-test block).
    - All new and existing tests in `internal/bootstrap/`, `internal/preflight/`, `internal/cli/`, and `internal/errcodes/` pass green via `go test ./... -count=1` (full regression suite).
    - The bug fix is provable by inspection: a fresh BYO host with NOPASSWD sudo and a non-`runnerkit-runner` SSH user can now run `runnerkit up` without the `curl: (23) Failure writing output to destination, Permission denied` failure (live verification deferred to Plan 06-07).
  </acceptance_criteria>
  <done>
    download_runner step writes the tarball with consistent ownership; integration test exists and is gated by build tag; smoke script asserts config.sh landed; the fakeExecutor-only test gap that hid Bug 2 since 02-02 is closed.
  </done>
</task>

</tasks>

<verification>
After both tasks:
1. `go test ./... -count=1 -race` — full suite green, no regressions.
2. `go test -tags=integration ./internal/bootstrap/... -count=1` (without RUNNERKIT_INTEGRATION=1) — integration test compiles and skips cleanly.
3. Manual sanity: with a fresh BYO host (NOPASSWD sudo configured), `runnerkit up --repo owner/name --host user@host --yes` reaches the runner registration step (no `Permission denied` from curl). Without NOPASSWD sudo, preflight emits a SeverityWarning with the new `host.privilege.password_required` ID and bootstrap_failed (when triggered by the missing Path B in this plan) surfaces the underlying redacted remote stderr.
</verification>

<success_criteria>
- Both BLOCKER bugs from `06-GAP-byo-sudo-handling.md` (Bug 1 preflight + Bug 2 download_runner permission) are fixed in code and covered by tests.
- New `RKD-BOOT-015` error code is in the registry, documented in `bootstrap.md`, and resolvable via `errcodes.URL(code)`.
- Build-tag-guarded integration test exercises real shell semantics (closes the fakeExecutor-only gap).
- Smoke script (`byo-permission.sh`) asserts `config.sh` landed before continuing to runner registration.
- `make test-integration` target exists and runs the integration test via the build tag.
- Plan 06-06 (Path B + Path C user-facing features) and Plan 06-07 (maintainer live smoke re-run) are now unblocked.
</success_criteria>

<output>
After completion, create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-05-byo-bootstrap-blocker-fixes-SUMMARY.md` documenting:
- Each task's RED + GREEN commits.
- Substring assertions added to existing tests.
- The new RKD-BOOT-015 entry's anchor + docs URL.
- Confirmation that `go test ./... -count=1 -race` passes green.
- Note that Path B (interactive sudo prompt) and Path C (`runnerkit byo-prepare`) are NOT in this plan — they land in Plan 06-06 and consume the new `host.privilege.password_required` warning that Plan 06-05 emits.
</output>
