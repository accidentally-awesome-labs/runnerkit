---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 08
type: execute
wave: 1
depends_on: [05, 06]
files_modified:
  - internal/bootstrap/script.go
  - internal/bootstrap/script_test.go
  - internal/bootstrap/install_integration_test.go
  - scripts/smoke/byo-permission.sh
autonomous: true
gap_closure: true
requirements: [REL-05]
must_haves:
  truths:
    - "`register_runner` step succeeds against a BYO host whose user has ONLY `(root) NOPASSWD: ALL` (or only the byo-prepare scoped sudoers entry from `RenderSudoersEntry`). No `(ALL)` runas required in host sudoers."
    - "Bootstrap script unit test asserts ABSENCE of `sudo -u runnerkit-runner ./config.sh` in `register_runner` shell form for both `RenderInstallScript` and `RenderEphemeralInstallScript`; PRESENCE of `sudo su -s /bin/bash - runnerkit-runner -c` in both renderers."
    - "The Plan 06-05 build-tag-guarded integration test (`internal/bootstrap/install_integration_test.go`) is extended with a sub-case that exercises the rendered `register_runner` form against a real bash shell with a sudoers configuration consisting only of `(root) NOPASSWD: ALL` and asserts no `sudo -u <non-root>` command line appears."
    - "Cloud path is verifiably unchanged: `internal/provider/hetzner/provision.go` is NOT modified; existing hetzner package tests stay green; `runnerkit-admin` cloud-init keeps `(ALL) NOPASSWD: ALL` (broader than needed but harmless — out of scope per Bug 3 closure note)."
    - "Re-running Plan 06-07 BYO smoke against `salar@mckee-small-desktop` (the host that exposed Bug 3 on 2026-05-05) lands a GitHub runner ID before destroy, with NO manual `/etc/sudoers.d/runnerkit-smoke-temp NOPASSWD ALL` workaround."
    - "`scripts/smoke/byo-permission.sh` (optional but recommended) asserts that after `runnerkit up` succeeds the install dir contains `.runner` (the GitHub-runner registration sentinel file produced by `config.sh --unattended`), giving Plan 06-07 re-smoke a hard pass/fail signal beyond `runnerkit up exited 0`."
  artifacts:
    - path: "internal/bootstrap/script.go"
      provides: "RenderInstallScript line 47 and RenderEphemeralInstallScript line 83 invoke `config.sh` via `sudo su -s /bin/bash - runnerkit-runner -c '...'` (instead of `sudo -u runnerkit-runner ./config.sh ...`). `su` runs from a root sudo context — no `(ALL)` runas required in host sudoers."
      contains: "sudo su -s /bin/bash - runnerkit-runner -c"
      contains_also: "RUNNERKIT_REGISTRATION_TOKEN"
      contains_also2: "config.sh --unattended"
      contains_also3: "--replace"
    - path: "internal/bootstrap/script_test.go"
      provides: "New test functions assert (a) absence of `sudo -u runnerkit-runner ./config.sh` and (b) presence of `sudo su -s /bin/bash - runnerkit-runner -c` in BOTH RenderInstallScript and RenderEphemeralInstallScript outputs. Token redaction invariant from existing tests preserved."
      contains: "TestRenderInstallScriptUsesSuForRegisterRunner"
      contains_also: "TestRenderEphemeralInstallScriptUsesSuForRegisterRunner"
      contains_also2: "su -s /bin/bash"
      contains_also3: "sudo -u"
    - path: "internal/bootstrap/install_integration_test.go"
      provides: "New build-tag-guarded sub-case `TestApply_RegisterRunner_RootOnlyNopasswd` that runs the rendered register_runner shell form through the existing `shellExecutor` from Plan 06-05 and asserts no `sudo -u <non-root>` command line is invoked. Uses the existing `buildFakeRunnerTarball` helper for the upstream tarball."
      contains: "TestApply_RegisterRunner_RootOnlyNopasswd"
      contains_also: "shellExecutor"
      contains_also2: "sudo su -s /bin/bash"
    - path: "scripts/smoke/byo-permission.sh"
      provides: "Optional post-bootstrap assertion that the install dir contains `.runner` (GitHub-runner registration sentinel) after `runnerkit up` exits 0. Adjacent to the Plan 06-05 `config.sh` assertion; gives Plan 06-07 re-smoke a hard pass/fail signal for the registration step."
      contains: ".runner"
  key_links:
    - from: "internal/bootstrap/script.go::RenderInstallScript register_runner line"
      to: "Remote shell with `sudo su -s /bin/bash - runnerkit-runner -c '...'` form"
      via: "Sprintf-rendered Script field; `su` runs from existing root sudo context (matches `(root) NOPASSWD: ALL`); no `(ALL)` runas required in target sudoers"
      pattern: "sudo su -s /bin/bash - runnerkit-runner -c"
    - from: "internal/bootstrap/script.go::RenderEphemeralInstallScript register_runner line"
      to: "Same `sudo su -s /bin/bash` form (identical pattern, ephemeral flag only differs)"
      via: "Sprintf-rendered Script field; same fix mirrored"
      pattern: "sudo su -s /bin/bash - runnerkit-runner -c"
    - from: "internal/bootstrap/install_integration_test.go::TestApply_RegisterRunner_RootOnlyNopasswd"
      to: "Plan 06-05 `shellExecutor` + `buildFakeRunnerTarball` helpers"
      via: "Same package; integration test re-uses existing harness; new test only adds the register_runner-specific assertion"
      pattern: "shellExecutor\\{workingDir"
    - from: "Plan 06-07 BYO smoke re-run"
      to: "BYO host with only `(root) NOPASSWD: ALL` or byo-prepare scoped sudoers"
      via: "Once 06-08 lands, register_runner no longer requires `(ALL)` runas — both system-wide root NOPASSWD and the byo-prepare template suffice; Plan 06-07 attempt-2 succeeds"
      pattern: "su -s /bin/bash"
---

<objective>
Close the third and final BLOCKER bug from `06-GAP-byo-sudo-handling.md` (Task F, Bug 3) discovered during Plan 06-07 attempt-1 re-smoke against `salar@mckee-small-desktop` on 2026-05-05.

After Plans 06-05 + 06-06 closed Bugs 1 + 2 (preflight `sudo -n true` probe + sudo-prefixed `download_runner` curl/sha256sum/tar), the live re-smoke proved those fixes work — `config.sh` extracted to `/opt/actions-runner/runnerkit-<owner>-<repo>-local/` successfully — and then exposed Bug 3 at the next step:

**Bug 3 (register_runner runas mismatch):** `internal/bootstrap/script.go:47` (RenderInstallScript) and `:83` (RenderEphemeralInstallScript) invoke
```bash
sudo -u runnerkit-runner RUNNERKIT_REGISTRATION_TOKEN="..." ./config.sh --unattended ...
```
to register the GitHub runner as the unprivileged service user. Linux sudoers semantics: `(root) NOPASSWD: ALL` only covers `runas=root`. Running as a non-root user (`runas=runnerkit-runner`) matches a different sudoers rule and triggers a password prompt. Both system-wide `(root) NOPASSWD: ALL` AND the byo-prepare scoped template `<user> ALL=(root) NOPASSWD:` (`internal/bootstrap/sudoers.go::RenderSudoersEntry` line 23) cover only `runas=root`. Result: `sudo: a terminal is required to read the password sudo: a password is required` — bootstrap fails after `download_runner` succeeds; BYO non-functional in v1 at registration step.

Cloud path unaffected: `internal/provider/hetzner/provision.go:241` cloud-init configures `runnerkit-admin` with `sudo: ALL=(ALL) NOPASSWD:ALL` — `(ALL)` runas covers `runas=runnerkit-runner`. Cloud is OUT OF SCOPE for this plan.

**Fix (Option 2 from gap doc, lines 165-176 — preferred; Options 1 + 3 explicitly REJECTED in gap doc):** replace `sudo -u runnerkit-runner ./config.sh ...` with `sudo su -s /bin/bash - runnerkit-runner -c '<config.sh ...>'`. `su` runs from a root sudo context (matches `(root) NOPASSWD: ALL`) → no `(ALL)` runas needed → works on BYO host with only root NOPASSWD AND on cloud host with broader NOPASSWD. The byo-prepare scoped sudoers template needs no change (the existing `(root)` runas in `RenderSudoersEntry` remains sufficient).

This plan is small and focused: 2 tasks (RED test + GREEN fix; smoke harness extension folded into GREEN). One renderer file modified; assertions added; integration test extended with a no-`(ALL)`-runas sub-case. Plan 06-07 (the live-smoke re-run human-action checkpoint) is the downstream consumer — it re-runs only after 06-08 lands.

Implements gap doc `## Required work` Task F + Bug 3 Acceptance bullets (lines 189-199; new acceptance bullets added 2026-05-05 at lines 393-399).

Purpose: unblock Plan 06-07 attempt-2 re-smoke; close the final v1.0.0 BYO blocker; restore Phase 6 success criterion 4 — "A fresh user can complete at least one supported setup path in about 10 minutes" — for BYO without manual sudoers preconfiguration. Per `06-VERIFICATION.md` Verifier Verdict (2026-05-05 update), this is the LAST blocker before v1.0.0 tag push.

Output: `register_runner` step works against any sudoers configuration that grants `runas=root` NOPASSWD (including the minimal byo-prepare scoped template). Test gap that hid Bug 3 from Plans 06-05 + 06-06 verification (script_test.go substring assertions never asserted absence of `sudo -u <non-root>`, install_integration_test.go only fixtured download_runner not register_runner) is closed.
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
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-06-byo-prepare-and-sudo-prompt-PLAN.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-07-live-smoke-rerun-and-baseline-fillin-PLAN.md
@internal/bootstrap/script.go
@internal/bootstrap/script_test.go
@internal/bootstrap/install_integration_test.go
@internal/bootstrap/sudoers.go
@internal/bootstrap/install.go
@scripts/smoke/byo-permission.sh

<interfaces>
<!-- Key contracts the executor must integrate with. Extracted from current source. -->

CURRENT internal/bootstrap/script.go RenderInstallScript line 47 (THE BUG):
```go
sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN="$RUNNERKIT_REGISTRATION_TOKEN" ./config.sh --unattended --url %[7]s --token "$RUNNERKIT_REGISTRATION_TOKEN" --name %[8]s --labels %[9]s --work %[3]s --replace
```
The fix: replace the `sudo -u %[1]s` prefix with `sudo su -s /bin/bash - %[1]s -c '...'` wrapping the entire `RUNNERKIT_REGISTRATION_TOKEN=... ./config.sh ...` invocation.

CURRENT internal/bootstrap/script.go RenderEphemeralInstallScript line 83 (SAME BUG, ephemeral flag added):
```go
sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN="$RUNNERKIT_REGISTRATION_TOKEN" ./config.sh --unattended --url %[7]s --token "$RUNNERKIT_REGISTRATION_TOKEN" --name %[8]s --labels %[9]s --work %[3]s --replace --ephemeral
```
Same fix.

REJECTED forms (gap doc lines 178-187):
- Option 1: expand byo-prepare scoped sudoers to `(ALL) NOPASSWD:` runas with per-repo allowlist — install dir name is repo-derived (`runnerkit-<owner>-<repo>-local`) so the allowlist must be regenerated per `runnerkit up` invocation, defeating the "one-time prepare" promise.
- Option 3: run `config.sh` as root with `HOME=/home/runnerkit-runner` — GitHub runner registration may write user-specific files (`.runner` ownership) differently when invoked from root vs the target user. Untested upstream.

Quote-handling guidance for `sudo su -s /bin/bash - runnerkit-runner -c '...'`:
- The outer wrapper passes a SINGLE-QUOTED string to `bash -c` so $SHELL expansion happens inside `runnerkit-runner`'s subshell, not in the outer SSH-user shell.
- The `RUNNERKIT_REGISTRATION_TOKEN` env var must remain visible to `config.sh`. Two options:
  (a) Pass it through the `su` environment via the inner command: `sudo su -s /bin/bash - runnerkit-runner -c 'RUNNERKIT_REGISTRATION_TOKEN="$RUNNERKIT_REGISTRATION_TOKEN" ./config.sh ...'`. Inside the single quotes, `$RUNNERKIT_REGISTRATION_TOKEN` is NOT expanded by the SSH-user shell — but `su -` resets the environment, so this only works if `su` preserves env vars in its `-c` body. With `su - <user> -c '<cmd>'`, the OUTER `$RUNNERKIT_REGISTRATION_TOKEN` is interpolated by the OUTER shell BEFORE `su` runs only if the shell script reaches the line via the SSH executor's environment. Confirm experimentally: the `remote.Command.Env` field already injects `RUNNERKIT_REGISTRATION_TOKEN` into the outer SSH shell env (see how Apply at install.go line 71 sets `Env: map[string]string{"RUNNERKIT_REGISTRATION_TOKEN": opts.RunnerToken}`). The OUTER shell expands `$RUNNERKIT_REGISTRATION_TOKEN` to the literal token BEFORE the single-quoted `su -c` body is constructed only if we use DOUBLE quotes around the outer body, not single. Use double quotes for the OUTER `-c` argument and escape internal double quotes: `sudo su -s /bin/bash - runnerkit-runner -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh ..."`. The escaped inner `\"$RUNNERKIT_REGISTRATION_TOKEN\"` is interpolated by the OUTER shell to the actual token string before `su` invokes its inner shell. NB: the existing line 47 already double-quotes `"$RUNNERKIT_REGISTRATION_TOKEN"` so this pattern is consistent with current code.
  (b) Alternative: use `sudo --preserve-env=RUNNERKIT_REGISTRATION_TOKEN su -s /bin/bash - runnerkit-runner -c '...'`. More explicit but requires sudo >= 1.8.x (universally available on supported distros). NOT preferred — env-passthrough via the outer shell expansion (option a) keeps the diff minimal.

PREFERRED form (use option a; matches existing line 47/83 pattern; minimal diff):
```bash
sudo su -s /bin/bash - runnerkit-runner -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh --unattended --url <repo_url> --token \"$RUNNERKIT_REGISTRATION_TOKEN\" --name <runner_name> --labels <labels> --work <work_dir> --replace"
```
And the ephemeral variant only differs by the `--ephemeral` flag at the end.

Existing test pattern in script_test.go (lines 39-56 — TestRenderInstallScriptUsesSudoForCurlSha256SumTar from Plan 06-05) is the model: build an Options literal, call the renderer, then `strings.Contains` assertions over the rendered string. New tests follow this pattern.

Existing integration test pattern in install_integration_test.go (Plan 06-05 commit, lines 63-111 — TestApply_DownloadRunner_RealShell) provides:
- `shellExecutor{workingDir: tmp}` — actually runs commands via `exec.Command("bash", "-c", c.Script)`.
- `buildFakeRunnerTarball(t, tarballPath)` — produces a real .tar.gz with a `config.sh` body.
- `httptest.NewServer(http.FileServer(http.Dir(tmp)))` — local URL the curl line can reach.
- `os.Getenv("RUNNERKIT_INTEGRATION") == ""` skip — already gates the test out of `go test ./...`.
- `Options{ServiceUser: currentUser, ...}` — uses `os.Getenv("USER")` so the test machine's current user can be the "service user" for the chmod/install -d -o lines.

The new sub-case `TestApply_RegisterRunner_RootOnlyNopasswd` re-uses ALL of these helpers; only the per-test fixture and assertion change. To simulate "host sudoers has only `(root) NOPASSWD: ALL`": the test does NOT need to actually mutate the local /etc/sudoers — it only needs to assert that the rendered shell does not contain `sudo -u <non-root>`. The "real-shell" portion of the test exercises the substituted `sudo su -s /bin/bash` form with the test machine's existing NOPASSWD sudo (the same RUNNERKIT_INTEGRATION=1 gate as TestApply_DownloadRunner_RealShell).

Substitution helper for the integration test: the rendered `register_runner` line is INSIDE the `RenderInstallScript` output. The integration test extracts this line by splitting on `\n` and asserting on the line containing `config.sh --unattended` (or the surrounding shell block).

renderer.Redactor invariant: existing tests TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken (lines 87-117) and TestRenderInstallAndServiceScripts (lines 8-32) ALREADY assert that `opts.RunnerToken` does NOT appear in rendered output (the renderer reads from $RUNNERKIT_REGISTRATION_TOKEN env var, never interpolating the literal). NEW tests must preserve this invariant — the `sudo su -s` rewrite cannot leak the token. Add an explicit no-leak assertion to the new tests for safety.

Numbering rule (gap doc 2026-05-05 acceptance bullets at lines 393-399):
- `(Bug 3) register_runner step succeeds against a BYO host whose user has ONLY (root) NOPASSWD: ALL (or only the byo-prepare scoped sudoers entry). No (ALL) runas required in host sudoers.`
- `(Bug 3) Bootstrap script unit test asserts absence of sudo -u in register_runner shell form; presence of su -s /bin/bash.`
- `(Bug 3) Plan 06-07 re-smoke against salar@mckee-small-desktop lands a GitHub runner ID before destroy.`
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: RED — failing unit + integration tests for Bug 3 (sudo -u absent, sudo su -s present)</name>
  <files>internal/bootstrap/script_test.go, internal/bootstrap/install_integration_test.go</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md (the Bug 3 spec — lines 122-199; Task F at lines 338-365; acceptance bullets at lines 393-399)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md (gaps[1] entry — lines 29-43; Anti-Patterns Found table at lines 226-234 showing the test gap that allowed Bug 3 to escape)
    - internal/bootstrap/script.go (CURRENT RenderInstallScript line 47 register_runner — the bug; CURRENT RenderEphemeralInstallScript line 83 same pattern)
    - internal/bootstrap/script_test.go (existing TestRenderInstallScriptUsesSudoForCurlSha256SumTar at lines 39-56 — the test pattern to mirror; existing TestRenderInstallAndServiceScripts at lines 8-32 — token-leak invariant to preserve)
    - internal/bootstrap/install_integration_test.go (existing TestApply_DownloadRunner_RealShell at lines 63-111 — the harness to extend; shellExecutor at lines 34-55; buildFakeRunnerTarball at lines 116-152)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-05-byo-bootstrap-blocker-fixes-PLAN.md (Task 2 RED→GREEN cycle pattern at Step 2.8 — the discipline this plan follows)
  </read_first>
  <behavior>
    - Test 1 (`TestRenderInstallScriptUsesSuForRegisterRunner`): build an Options literal identical to the existing TestRenderInstallScriptUsesSudoForCurlSha256SumTar; call `RenderInstallScript(opts)`; assert (a) `strings.Contains(script, "sudo su -s /bin/bash - runnerkit-runner -c")` is true, (b) `strings.Contains(script, "sudo -u runnerkit-runner ./config.sh")` is FALSE (NEGATIVE assertion — the bug pattern must be GONE), (c) `strings.Contains(script, "sudo -u runnerkit-runner RUNNERKIT_REGISTRATION_TOKEN")` is FALSE (the literal pre-fix line). Token-leak invariant preserved: `strings.Contains(script, opts.RunnerToken)` MUST be false.
    - Test 2 (`TestRenderEphemeralInstallScriptUsesSuForRegisterRunner`): same pattern as Test 1 but for `RenderEphemeralInstallScript`. The `--ephemeral` flag must remain at the end of the wrapped command (verify `strings.Contains(script, "--replace --ephemeral\"")` or `strings.Contains(script, "--ephemeral")` after the `--replace` token — wrap in the same single substring assertion as the existing TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken pattern). Same negative + no-leak invariants as Test 1.
    - Test 3 — INTEGRATION (`TestApply_RegisterRunner_RootOnlyNopasswd` in `install_integration_test.go` with the existing `//go:build integration` tag): re-use `shellExecutor`, `buildFakeRunnerTarball`, and `httptest.NewServer` from the Plan 06-05 harness. Build an Options for the persistent install path; call `RenderInstallScript(opts)` and EXTRACT the `register_runner` line (the substring starting at `sudo` and ending before the next `\n`, that contains `config.sh --unattended`). Assert: (a) the extracted line does NOT contain `sudo -u runnerkit-runner` (negative — Bug 3 pattern absent), (b) the extracted line DOES contain `sudo su -s /bin/bash - runnerkit-runner -c` (positive). To preserve the existing "real-shell" character of the test file, ALSO write the rendered script to a temp file and execute the lines BEFORE `register_runner` (download + extract the fake tarball) via the shellExecutor — but stop before `register_runner` because the fake tarball's `config.sh` is a placeholder that exits 0; the assertion is on the rendered shape, not on a real GitHub registration. Skip cleanly when `RUNNERKIT_INTEGRATION` is unset (matches existing TestApply_DownloadRunner_RealShell skip pattern).
    - All three tests fail RED (do NOT pass) BEFORE Task 2 lands, because the current `script.go` lines 47 and 83 still use `sudo -u runnerkit-runner ./config.sh ...`. Verify RED by running the tests on a clean tree before Task 2 production change. Commit RED first.
  </behavior>
  <action>
    **Step 1.1 — Append two new test functions to `internal/bootstrap/script_test.go`** (after the existing TestRenderEphemeralInstallScriptUsesSudoForCurlSha256SumTar at line 78):

    ```go
    // TestRenderInstallScriptUsesSuForRegisterRunner asserts the
    // renderer-side fix for Bug 3 of gap doc 06-GAP-byo-sudo-handling.md:
    // register_runner must invoke config.sh via `sudo su -s /bin/bash -
    // runnerkit-runner -c '...'` instead of `sudo -u runnerkit-runner
    // ./config.sh ...`. The su form runs from a root sudo context so the
    // host's sudoers needs only (root) NOPASSWD — no (ALL) runas required.
    // See gap doc lines 122-199 (Bug 3 description) and lines 338-365
    // (Task F) for the rationale.
    func TestRenderInstallScriptUsesSuForRegisterRunner(t *testing.T) {
        opts := Options{
            RunnerName:  "runnerkit-owner-repo-local",
            RepoURL:     "https://github.com/owner/repo",
            Labels:      []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "persistent"},
            InstallPath: "/opt/actions-runner/runnerkit-owner-repo-local",
            WorkDir:     "/var/lib/runnerkit/work/runnerkit-owner-repo-local",
            ServiceUser: "runnerkit-runner",
            RunnerToken: "registration-token-secret-bug3-12345",
            Package:     RunnerPackage{Filename: "actions-runner-linux-x64-2.334.0.tar.gz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
        }
        script := RenderInstallScript(opts)
        // PRESENCE assertion: the new su form must be in the rendered script.
        if !strings.Contains(script, "sudo su -s /bin/bash - runnerkit-runner -c") {
            t.Fatalf("RenderInstallScript missing sudo su -s /bin/bash - runnerkit-runner -c form for register_runner:\n%s", script)
        }
        // NEGATIVE assertion: the buggy `sudo -u runnerkit-runner ./config.sh` form must be GONE.
        for _, forbidden := range []string{
            "sudo -u runnerkit-runner ./config.sh",
            "sudo -u runnerkit-runner RUNNERKIT_REGISTRATION_TOKEN",
        } {
            if strings.Contains(script, forbidden) {
                t.Fatalf("RenderInstallScript still contains buggy %q (Bug 3 not closed):\n%s", forbidden, script)
            }
        }
        // Token-leak invariant: $RUNNERKIT_REGISTRATION_TOKEN must remain the env-var reference; the literal token must NOT appear.
        if strings.Contains(script, opts.RunnerToken) {
            t.Fatalf("RenderInstallScript leaked registration token literal (Bug 3 fix must preserve redaction invariant):\n%s", script)
        }
    }

    // TestRenderEphemeralInstallScriptUsesSuForRegisterRunner is the
    // parallel assertion for the ephemeral renderer.
    func TestRenderEphemeralInstallScriptUsesSuForRegisterRunner(t *testing.T) {
        opts := Options{
            RunnerName:  "runnerkit-owner-repo-ephemeral-bug3test",
            RepoURL:     "https://github.com/owner/repo",
            Labels:      []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "ephemeral"},
            InstallPath: "/opt/actions-runner/runnerkit-owner-repo-ephemeral-bug3test",
            WorkDir:     "/var/lib/runnerkit/work/runnerkit-owner-repo-ephemeral-bug3test",
            ServiceUser: "runnerkit-runner",
            RunnerToken: "registration-token-ephemeral-bug3-secret",
            Mode:        "ephemeral",
            Package:     RunnerPackage{Filename: "actions-runner-linux-x64-2.334.0.tar.gz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
        }
        script := RenderEphemeralInstallScript(opts)
        if !strings.Contains(script, "sudo su -s /bin/bash - runnerkit-runner -c") {
            t.Fatalf("RenderEphemeralInstallScript missing sudo su -s /bin/bash - runnerkit-runner -c form for register_runner:\n%s", script)
        }
        for _, forbidden := range []string{
            "sudo -u runnerkit-runner ./config.sh",
            "sudo -u runnerkit-runner RUNNERKIT_REGISTRATION_TOKEN",
        } {
            if strings.Contains(script, forbidden) {
                t.Fatalf("RenderEphemeralInstallScript still contains buggy %q (Bug 3 not closed):\n%s", forbidden, script)
            }
        }
        // Ephemeral-specific: --ephemeral flag must remain at the end of the wrapped command.
        if !strings.Contains(script, "--replace --ephemeral") {
            t.Fatalf("RenderEphemeralInstallScript missing --replace --ephemeral flag tail in wrapped command:\n%s", script)
        }
        if strings.Contains(script, opts.RunnerToken) {
            t.Fatalf("RenderEphemeralInstallScript leaked registration token literal:\n%s", script)
        }
    }
    ```

    **Step 1.2 — Extend `internal/bootstrap/install_integration_test.go` with the new sub-case.** Append after the existing `buildFakeRunnerTarball` helper (after line 152). The new test re-uses `shellExecutor`, `buildFakeRunnerTarball`, and the build-tag gate. It is INTENTIONALLY focused on the rendered shell form (string-shape assertion + RUNNERKIT_INTEGRATION=1 skip), NOT on a real GitHub registration call (the fake tarball's config.sh is a placeholder; we don't need a real GitHub repo for this test):

    ```go
    // TestApply_RegisterRunner_RootOnlyNopasswd asserts that the
    // rendered register_runner shell form is acceptable to a sudoers
    // configuration consisting only of `(root) NOPASSWD: ALL` — i.e. the
    // byo-prepare scoped sudoers entry alone is sufficient with no `(ALL)`
    // runas required. This closes the test gap that hid Bug 3 from Plans
    // 06-05 + 06-06 verification (gap doc 06-GAP-byo-sudo-handling.md
    // lines 122-199 + 338-365).
    //
    // Strategy: render the install script via RenderInstallScript, extract
    // the register_runner line, and assert (a) absence of `sudo -u
    // <non-root>` (Bug 3 pattern), (b) presence of `sudo su -s /bin/bash`
    // (Task F fix). The real-shell harness (shellExecutor +
    // buildFakeRunnerTarball + httptest) is set up so future extensions
    // can exercise the full sequence; this test stops at the shell-form
    // assertion because a real registration call requires a real GitHub
    // runner registration token.
    func TestApply_RegisterRunner_RootOnlyNopasswd(t *testing.T) {
        if os.Getenv("RUNNERKIT_INTEGRATION") == "" {
            t.Skip("set RUNNERKIT_INTEGRATION=1 to run; mirrors TestApply_DownloadRunner_RealShell skip pattern")
        }

        // Reuse the Plan 06-05 fake-tarball harness so the install dir
        // structure is realistic — we don't ACTUALLY register, but the
        // setup exercises the full path leading up to register_runner.
        tmp := t.TempDir()
        tarballPath := filepath.Join(tmp, "fake-runner.tgz")
        _ = buildFakeRunnerTarball(t, tarballPath)
        server := httptest.NewServer(http.FileServer(http.Dir(tmp)))
        defer server.Close()

        installPath := filepath.Join(tmp, "install")
        currentUser := os.Getenv("USER")
        if currentUser == "" {
            currentUser = "runnerkit-runner"
        }
        opts := Options{
            RunnerName:  "runnerkit-it-bug3test",
            RepoURL:     "https://github.com/owner/repo",
            Labels:      []string{"self-hosted"},
            InstallPath: installPath,
            WorkDir:     filepath.Join(tmp, "work"),
            ServiceUser: currentUser,
            RunnerToken: "registration-token-itest-bug3",
            Package:     RunnerPackage{Filename: "fake-runner.tgz", URL: server.URL + "/fake-runner.tgz", SHA256: "ignored-shape-only-test"},
        }

        script := RenderInstallScript(opts)
        // Extract the line containing `config.sh --unattended` — the register_runner invocation.
        var registerLine string
        for _, line := range strings.Split(script, "\n") {
            if strings.Contains(line, "config.sh --unattended") {
                registerLine = line
                break
            }
        }
        if registerLine == "" {
            t.Fatalf("rendered install script does not contain config.sh --unattended invocation:\n%s", script)
        }

        // NEGATIVE: Bug 3 pattern absent.
        if strings.Contains(registerLine, "sudo -u "+currentUser) || strings.Contains(registerLine, "sudo -u runnerkit-runner") {
            t.Fatalf("register_runner line still uses sudo -u <non-root> (Bug 3 not closed): %q\nfull script:\n%s", registerLine, script)
        }
        // POSITIVE: Task F fix present.
        if !strings.Contains(registerLine, "sudo su -s /bin/bash") {
            t.Fatalf("register_runner line missing sudo su -s /bin/bash form (Task F): %q\nfull script:\n%s", registerLine, script)
        }
    }
    ```

    Note the new imports needed in install_integration_test.go: `strings` (extract the line). Already imported via the existing test file? Verify by reading the existing imports block at line 12-26 — `strings` is NOT currently imported (the existing test only uses io/os/exec/etc.). Add `"strings"` to the import block.

    **Step 1.3 — RED commit.** Run the new tests on a clean tree (BEFORE Task 2's production fix):
    ```bash
    go test ./internal/bootstrap/... -count=1 -run 'TestRenderInstallScriptUsesSuForRegisterRunner|TestRenderEphemeralInstallScriptUsesSuForRegisterRunner'
    ```
    Expect FAIL with messages like:
    ```
    --- FAIL: TestRenderInstallScriptUsesSuForRegisterRunner
        script_test.go:NN: RenderInstallScript missing sudo su -s /bin/bash - runnerkit-runner -c form for register_runner:
            ...sudo -u runnerkit-runner RUNNERKIT_REGISTRATION_TOKEN=...
    ```
    AND:
    ```bash
    RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -run TestApply_RegisterRunner_RootOnlyNopasswd
    ```
    Expect FAIL with the negative-assertion message about `sudo -u`.

    Commit RED:
    ```bash
    git add internal/bootstrap/script_test.go internal/bootstrap/install_integration_test.go
    git commit -m "test(06-08): add failing tests for Bug 3 register_runner runas mismatch"
    ```
  </action>
  <verify>
    <automated>go test ./internal/bootstrap/... -count=1 -run 'TestRenderInstallScriptUsesSuForRegisterRunner|TestRenderEphemeralInstallScriptUsesSuForRegisterRunner' 2>&1 | grep -E '(FAIL|PASS|ok)'</automated>
    <secondary>grep -n "TestRenderInstallScriptUsesSuForRegisterRunner" internal/bootstrap/script_test.go</secondary>
    <secondary>grep -n "TestRenderEphemeralInstallScriptUsesSuForRegisterRunner" internal/bootstrap/script_test.go</secondary>
    <secondary>grep -n "TestApply_RegisterRunner_RootOnlyNopasswd" internal/bootstrap/install_integration_test.go</secondary>
  </verify>
  <acceptance_criteria>
    - `internal/bootstrap/script_test.go` contains test functions named exactly `TestRenderInstallScriptUsesSuForRegisterRunner` and `TestRenderEphemeralInstallScriptUsesSuForRegisterRunner` (verified by `grep -c 'func TestRenderInstallScriptUsesSuForRegisterRunner\|func TestRenderEphemeralInstallScriptUsesSuForRegisterRunner' internal/bootstrap/script_test.go` returns exactly `2`).
    - Both new unit tests CURRENTLY FAIL on the unmodified `script.go` — this is the RED commit. Verified by running `go test ./internal/bootstrap/... -count=1 -run 'TestRenderInstallScriptUsesSuForRegisterRunner|TestRenderEphemeralInstallScriptUsesSuForRegisterRunner'` and observing FAIL output mentioning `missing sudo su -s /bin/bash`. Sample expected: `script_test.go:NN: RenderInstallScript missing sudo su -s /bin/bash`.
    - `internal/bootstrap/install_integration_test.go` contains a function named exactly `TestApply_RegisterRunner_RootOnlyNopasswd` (verified by `grep -c 'func TestApply_RegisterRunner_RootOnlyNopasswd' internal/bootstrap/install_integration_test.go` returns exactly `1`). The test file's import block now includes `"strings"`.
    - When RUNNERKIT_INTEGRATION is unset, `go test -tags=integration ./internal/bootstrap/... -count=1 -run TestApply_RegisterRunner_RootOnlyNopasswd` outputs `--- SKIP` (the existing skip pattern from TestApply_DownloadRunner_RealShell is mirrored).
    - When RUNNERKIT_INTEGRATION=1, the new integration test runs but FAILS on the unmodified script.go (RED).
    - RED commit message follows the convention: `test(06-08): add failing tests for Bug 3 register_runner runas mismatch`.
    - No production source files (`internal/bootstrap/script.go`, `internal/bootstrap/install.go`) modified in this commit — verified by `git diff HEAD~1 --name-only` showing only the two test files.
    - Token-leak invariant assertions present: each new unit test contains a `strings.Contains(script, opts.RunnerToken)` MUST-NOT-MATCH check.
  </acceptance_criteria>
  <done>
    Two new unit tests + one new integration sub-case exist; all three FAIL on the unmodified renderer (RED); the test gap that hid Bug 3 from Plans 06-05 + 06-06 verification is now reproducibly demonstrated by the test suite.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: GREEN — replace `sudo -u runnerkit-runner ./config.sh` with `sudo su -s /bin/bash` in both renderers; extend smoke harness</name>
  <files>internal/bootstrap/script.go, scripts/smoke/byo-permission.sh</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md (Task F at lines 338-365; Bug 3 fix preferred form at lines 165-176; cloud unaffected note at lines 158-162)
    - internal/bootstrap/script.go (CURRENT lines 47 + 83 — exact text to replace; pkg.URL/RepoURL/RunnerName/labels/workDir field substitution patterns at lines 35-48)
    - internal/bootstrap/script_test.go (the RED tests from Task 1 — lines added above; existing TestRenderInstallAndServiceScripts at lines 8-32 and TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken at lines 87-117 must STAY GREEN — token-leak invariant + ephemeral-flag substring assertions are preserved by the new su form)
    - internal/bootstrap/install_integration_test.go (the RED integration sub-case from Task 1)
    - scripts/smoke/byo-permission.sh (existing structure; the optional `.runner` sentinel assertion attaches AFTER the existing `runnerkit up` line)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-07-live-smoke-rerun-and-baseline-fillin-PLAN.md (the downstream consumer — confirms the smoke harness extension is welcome but NOT required for re-smoke success; folded into this plan as best-effort)
  </read_first>
  <behavior>
    - All RED tests from Task 1 turn GREEN: `TestRenderInstallScriptUsesSuForRegisterRunner`, `TestRenderEphemeralInstallScriptUsesSuForRegisterRunner`, `TestApply_RegisterRunner_RootOnlyNopasswd` (under RUNNERKIT_INTEGRATION=1).
    - All EXISTING tests remain GREEN: `TestRenderInstallAndServiceScripts`, `TestRenderInstallScriptUsesSudoForCurlSha256SumTar`, `TestRenderEphemeralInstallScriptUsesSudoForCurlSha256SumTar`, `TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken`, `TestRenderRecoveryScriptsUseEnvironmentTokens`, `TestApply_DownloadRunner_RealShell`. NO regression.
    - Cloud path verifiably unchanged: `internal/provider/hetzner/provision.go` is NOT modified; `internal/provider/hetzner/...` package tests remain green.
    - Full Go test suite green: `go test ./... -count=1 -race`.
    - Optional: smoke script asserts `.runner` exists in the install dir after `runnerkit up` succeeds (gives Plan 06-07 re-smoke a hard registration sentinel beyond `runnerkit up exited 0`).
  </behavior>
  <action>
    **Step 2.1 — Fix `RenderInstallScript` register_runner line in `internal/bootstrap/script.go`.** Locate line 47 of `script.go` (inside the Sprintf for `RenderInstallScript` at lines 35-48). Current text:
    ```
    sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN="$RUNNERKIT_REGISTRATION_TOKEN" ./config.sh --unattended --url %[7]s --token "$RUNNERKIT_REGISTRATION_TOKEN" --name %[8]s --labels %[9]s --work %[3]s --replace
    ```

    Replace with (note the entire `RUNNERKIT_REGISTRATION_TOKEN=... ./config.sh ...` becomes the body of `bash -c "..."`, wrapped in DOUBLE quotes so the OUTER shell expands `$RUNNERKIT_REGISTRATION_TOKEN`; inner double quotes are escaped as `\"`):
    ```
    sudo su -s /bin/bash - %[1]s -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh --unattended --url %[7]s --token \"$RUNNERKIT_REGISTRATION_TOKEN\" --name %[8]s --labels %[9]s --work %[3]s --replace"
    ```

    Concretely the diff in the Sprintf format string (line 47) is:
    - BEFORE: `sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN="$RUNNERKIT_REGISTRATION_TOKEN" ./config.sh --unattended --url %[7]s --token "$RUNNERKIT_REGISTRATION_TOKEN" --name %[8]s --labels %[9]s --work %[3]s --replace`
    - AFTER:  `sudo su -s /bin/bash - %[1]s -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh --unattended --url %[7]s --token \"$RUNNERKIT_REGISTRATION_TOKEN\" --name %[8]s --labels %[9]s --work %[3]s --replace"`

    Inside a Go raw string literal (backticks), the inner backslash-escaped quotes `\"` are literal characters — Go's raw-string syntax does not interpret backslashes. So the line ends up in the rendered script as `sudo su -s /bin/bash - runnerkit-runner -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh --unattended ..."` — which is exactly what bash needs (the `\"` sequences are bash escapes, not Go escapes).

    **Step 2.2 — Apply the IDENTICAL substitution to `RenderEphemeralInstallScript` line 83 in `internal/bootstrap/script.go`.** Same pattern, only the trailing `--replace --ephemeral` differs from `--replace`:
    - BEFORE: `sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN="$RUNNERKIT_REGISTRATION_TOKEN" ./config.sh --unattended --url %[7]s --token "$RUNNERKIT_REGISTRATION_TOKEN" --name %[8]s --labels %[9]s --work %[3]s --replace --ephemeral`
    - AFTER:  `sudo su -s /bin/bash - %[1]s -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh --unattended --url %[7]s --token \"$RUNNERKIT_REGISTRATION_TOKEN\" --name %[8]s --labels %[9]s --work %[3]s --replace --ephemeral"`

    NB: `RenderRemoveConfigScript` (line 243) and `RenderReconfigureScript` (line 253) ALSO use the `sudo -u %s` pattern. Are they in scope? Read the gap doc — the Bug 3 acceptance bullets at lines 393-399 and Task F at lines 338-365 mention ONLY RenderInstallScript line 47 and RenderEphemeralInstallScript line 83. RenderRemoveConfigScript and RenderReconfigureScript are USED only by recovery flows (`runnerkit recover`) which run AFTER bootstrap — by which point `byo-prepare` has either been run (scoped sudoers exists) OR the user has typed a sudo password (Path B). However, those flows STILL hit the same `(ALL)` runas semantics. To stay narrowly scoped on Bug 3 closure (per the user-supplied gap-closure plan boundaries), DO NOT modify RenderRemoveConfigScript or RenderReconfigureScript in this plan. File a follow-up todo if the maintainer wants them updated for consistency.

    Add a code comment immediately above the modified line in BOTH RenderInstallScript and RenderEphemeralInstallScript explaining WHY the `su` form replaces `sudo -u`:
    ```
    // register_runner: invoke config.sh via `su` from a root sudo context
    // so the host's sudoers needs only (root) NOPASSWD — no (ALL) runas
    // required. Closes Bug 3 from 06-GAP-byo-sudo-handling.md.
    // sudo -u <non-root> would match (ALL) runas, which neither the
    // byo-prepare scoped template nor a typical (root) NOPASSWD: ALL host
    // sudoers covers. See gap doc lines 122-199 for the full rationale.
    ```

    **Step 2.3 — Run the test suite to confirm GREEN.** First the targeted tests:
    ```bash
    go test ./internal/bootstrap/... -count=1 -run 'TestRenderInstallScriptUsesSuForRegisterRunner|TestRenderEphemeralInstallScriptUsesSuForRegisterRunner'
    ```
    Expect PASS. Then the full bootstrap suite to catch regressions:
    ```bash
    go test ./internal/bootstrap/... -count=1
    ```
    Expect PASS — note specifically that `TestRenderInstallAndServiceScripts` (line 20 wanted-substring `./config.sh --unattended --url https://github.com/owner/repo --token \"$RUNNERKIT_REGISTRATION_TOKEN\"`) STILL passes because the new su-wrapped form contains the same `./config.sh --unattended --url ... --token "$RUNNERKIT_REGISTRATION_TOKEN"` substring (the wrap adds prefix `sudo su -s /bin/bash - runnerkit-runner -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ` before, NOT a substring break). Same logic for `TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken` at line 100-107: the wanted substring `./config.sh --unattended --url https://github.com/owner/repo --token \"$RUNNERKIT_REGISTRATION_TOKEN\"` is preserved verbatim.

    Then the integration test under RUNNERKIT_INTEGRATION=1:
    ```bash
    RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -run TestApply_RegisterRunner_RootOnlyNopasswd
    ```
    Expect PASS.

    Finally the full repository regression:
    ```bash
    go test ./... -count=1 -race
    ```
    Expect PASS across all 17+ packages, with no impact on `internal/provider/hetzner/...` or `internal/cli/...` tests.

    **Step 2.4 — Extend `scripts/smoke/byo-permission.sh` with `.runner` sentinel assertion (optional but recommended).** Locate the existing `config.sh` assertion block (added by Plan 06-05 per `internal/bootstrap/install_integration_test.go` Step 2.6). Append AFTER the `runnerkit up` invocation succeeds and AFTER the existing `config.sh` assertion (so the order is: (1) up succeeds, (2) tarball + config.sh extracted, (3) registration sentinel `.runner` exists):

    ```bash
    echo "===> [smoke-byo] Asserting GitHub-runner registration sentinel .runner exists"
    ssh "${HOST}" 'sudo test -f /opt/actions-runner/runnerkit-*/.runner' || {
      echo "FAIL: .runner sentinel not found in /opt/actions-runner/runnerkit-*/ — config.sh --unattended did not complete registration (Bug 3 regression?)"
      exit 4
    }
    ```

    The `.runner` file is the GitHub-runner registration sentinel that `config.sh --unattended` writes on success. This gives Plan 06-07 attempt-2 re-smoke a HARD pass/fail signal: if `runnerkit up` exits 0 but `.runner` is absent, the smoke fails with a specific error (exit 4, distinct from the existing exit 3 for missing config.sh) instead of falsely succeeding. NB: variable name in the existing script is `${HOST}` if that's what Plan 06-05 used; verify when reading the script. If the existing variable is `$RUNNERKIT_SMOKE_BYO_HOST` instead of `${HOST}`, match the existing pattern.

    **Step 2.5 — GREEN commit.**
    ```bash
    git add internal/bootstrap/script.go scripts/smoke/byo-permission.sh
    git commit -m "fix(06-08): replace sudo -u <user> ./config.sh with sudo su -s /bin/bash to close Bug 3 register_runner runas mismatch"
    ```
    NB: do NOT amend the RED commit — the RED + GREEN pair is the audit trail.

    **Step 2.6 — Sanity grep verification (post-commit).** All four greps below MUST return the expected results:
    ```bash
    grep -F 'sudo su -s /bin/bash - %[1]s -c' internal/bootstrap/script.go    # exactly 2 matches (RenderInstallScript + RenderEphemeralInstallScript)
    grep -cF 'sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN' internal/bootstrap/script.go   # exactly 0 matches (the bug pattern in Sprintf must be GONE)
    grep -cF 'sudo -u runnerkit-runner ./config.sh' internal/bootstrap/script.go         # exactly 0 matches (the literal-rendered form must be GONE)
    grep -F '.runner' scripts/smoke/byo-permission.sh   # at least 1 match in the new sentinel assertion
    ```
    Note: RenderRemoveConfigScript line 243 and RenderReconfigureScript line 253 still use `sudo -u %s` — that's intentional (out of scope for Bug 3 per Step 2.2 note). The `grep -F 'sudo -u'` count for the whole file is therefore 2, NOT 0 — the test in Step 2.6 second/third bullet uses the more specific grep patterns to avoid false positives.
  </action>
  <verify>
    <automated>go test ./... -count=1 -race</automated>
    <secondary>go test ./internal/bootstrap/... -count=1 -run 'TestRenderInstallScriptUsesSuForRegisterRunner|TestRenderEphemeralInstallScriptUsesSuForRegisterRunner|TestRenderInstallAndServiceScripts|TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken|TestRenderRecoveryScriptsUseEnvironmentTokens'</secondary>
    <secondary>RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -run 'TestApply_RegisterRunner_RootOnlyNopasswd|TestApply_DownloadRunner_RealShell'</secondary>
    <secondary>grep -F 'sudo su -s /bin/bash - %[1]s -c' internal/bootstrap/script.go</secondary>
    <secondary>grep -cF 'sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN' internal/bootstrap/script.go</secondary>
    <secondary>grep -cF 'sudo -u runnerkit-runner ./config.sh' internal/bootstrap/script.go</secondary>
    <secondary>grep -F '.runner' scripts/smoke/byo-permission.sh</secondary>
  </verify>
  <acceptance_criteria>
    - `internal/bootstrap/script.go` contains the literal substring `sudo su -s /bin/bash - %[1]s -c` exactly 2 times (RenderInstallScript line ~47 + RenderEphemeralInstallScript line ~83). Verified by `grep -cF 'sudo su -s /bin/bash - %[1]s -c' internal/bootstrap/script.go` returns `2`.
    - `internal/bootstrap/script.go` does NOT contain the substring `sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN` ANYWHERE — the Bug 3 Sprintf pattern is gone. Verified by `grep -cF 'sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN' internal/bootstrap/script.go` returns `0`.
    - `internal/bootstrap/script.go` does NOT contain the literal-rendered form `sudo -u runnerkit-runner ./config.sh`. Verified by `grep -cF 'sudo -u runnerkit-runner ./config.sh' internal/bootstrap/script.go` returns `0`.
    - `internal/bootstrap/script.go` STILL contains `sudo -u ` (used by RenderRemoveConfigScript line 243 and RenderReconfigureScript line 253) — those are out of scope for Bug 3 closure per the gap doc Task F bounds. Verified by `grep -cF 'sudo -u ' internal/bootstrap/script.go` returns `2`.
    - All RED tests from Task 1 now PASS: `go test ./internal/bootstrap/... -count=1 -run 'TestRenderInstallScriptUsesSuForRegisterRunner|TestRenderEphemeralInstallScriptUsesSuForRegisterRunner'` exits 0 with both tests in `--- PASS`.
    - Existing tests STAY GREEN — no regression: `go test ./internal/bootstrap/... -count=1` exits 0; `TestRenderInstallAndServiceScripts`, `TestRenderInstallScriptUsesSudoForCurlSha256SumTar`, `TestRenderEphemeralInstallScriptUsesSudoForCurlSha256SumTar`, `TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken`, `TestRenderRecoveryScriptsUseEnvironmentTokens` all PASS.
    - Token-leak invariant preserved: existing assertions `if strings.Contains(install, opts.RunnerToken)` (script_test.go line 25) and `if strings.Contains(script, fakeToken)` (line 114) still PASS — the new `\"$RUNNERKIT_REGISTRATION_TOKEN\"` form still uses the env-var reference, not the literal token.
    - Integration test passes under RUNNERKIT_INTEGRATION=1: `RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -run TestApply_RegisterRunner_RootOnlyNopasswd` exits 0.
    - Cloud path UNTOUCHED: `git diff HEAD~1 -- internal/provider/hetzner/` returns empty (no diffs in the hetzner package). The cloud cloud-init still uses `(ALL) NOPASSWD: ALL` (broader than needed, harmless — gap doc Task F closure note).
    - Full repository regression GREEN: `go test ./... -count=1 -race` exits 0 across all 17+ packages.
    - `scripts/smoke/byo-permission.sh` contains a NEW assertion mentioning `.runner` (verified by `grep -nF '.runner' scripts/smoke/byo-permission.sh` returns at least 1 match in the new ssh-test block AFTER the existing `config.sh` assertion). The new block exits with a distinct exit code (`exit 4`) so Plan 06-07 re-smoke can distinguish "registration sentinel missing" from "config.sh missing" (existing exit 3).
    - GREEN commit message follows the convention: `fix(06-08): replace sudo -u <user> ./config.sh with sudo su -s /bin/bash to close Bug 3 register_runner runas mismatch`.
    - Renderer change is provable by inspection: a fresh BYO host whose user has ONLY `(root) NOPASSWD: ALL` (or only the byo-prepare scoped sudoers entry from `RenderSudoersEntry`) can now run `runnerkit up` end-to-end without the `sudo: a password is required` failure observed in Plan 06-07 attempt-1 (live verification deferred to Plan 06-07 attempt-2).
  </acceptance_criteria>
  <done>
    `register_runner` step uses `sudo su -s /bin/bash - runnerkit-runner -c '...'` form in both persistent and ephemeral renderers; no `(ALL)` runas required in host sudoers; cloud path untouched; smoke harness asserts `.runner` registration sentinel; full test suite green; Plan 06-07 attempt-2 re-smoke unblocked.
  </done>
</task>

</tasks>

<verification>
After both tasks:
1. `go test ./... -count=1 -race` — full suite green; no regressions in any package, including `internal/provider/hetzner/...` (cloud path unchanged).
2. `grep -cF 'sudo su -s /bin/bash - %[1]s -c' internal/bootstrap/script.go` returns exactly `2` (one each for RenderInstallScript + RenderEphemeralInstallScript).
3. `grep -cF 'sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN' internal/bootstrap/script.go` returns exactly `0` (the Bug 3 Sprintf pattern is gone).
4. `grep -cF 'sudo -u runnerkit-runner ./config.sh' internal/bootstrap/script.go` returns exactly `0` (the literal-rendered Bug 3 form is gone).
5. `RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1` — both `TestApply_DownloadRunner_RealShell` (Plan 06-05) AND new `TestApply_RegisterRunner_RootOnlyNopasswd` pass; without RUNNERKIT_INTEGRATION=1, both skip cleanly.
6. `grep -nF '.runner' scripts/smoke/byo-permission.sh` returns at least one match in the new sentinel-assertion block.
7. RED + GREEN commits exist as a discrete pair: `git log --oneline -3` shows `test(06-08): add failing tests for Bug 3 ...` followed by `fix(06-08): replace sudo -u <user> ./config.sh with sudo su -s /bin/bash ...`.
8. `git diff HEAD~2 -- internal/provider/hetzner/` is empty — cloud path verifiably unchanged.
9. Manual sanity (deferred to Plan 06-07 attempt-2): a fresh BYO host with ONLY `(root) NOPASSWD: ALL` (no `(ALL)` runas) runs `runnerkit up` end-to-end through `register_runner` without `sudo: a password is required`. This is the must-haves truth #1 + #5 — Plan 06-07 owns the live verification.
</verification>

<success_criteria>
- Bug 3 from `06-GAP-byo-sudo-handling.md` (`register_runner` runas mismatch) is fixed in code AND covered by tests (unit + integration).
- The test gap that hid Bug 3 from Plans 06-05 + 06-06 verification (script_test.go substring assertions never asserted absence of `sudo -u <non-root>`; install_integration_test.go only fixtured download_runner) is closed.
- Cloud path verifiably unchanged: hetzner provider source + tests untouched; cloud-init still works (broader `(ALL) NOPASSWD: ALL` covers both `runas=root` and `runas=runnerkit-runner` — harmless redundancy).
- Smoke harness extended with `.runner` sentinel assertion so Plan 06-07 attempt-2 has a hard pass/fail signal beyond `runnerkit up exited 0`.
- Plan 06-07 (the live-smoke re-run human-action checkpoint) is unblocked. Once Plan 06-08 lands, the maintainer re-runs `make smoke-live` against `salar@mckee-small-desktop` (the host that exposed Bug 3) without manual sudoers preconfiguration AND without modifying byo-prepare.
- RED + GREEN commits land as a discrete pair, mirroring the TDD discipline of Plans 06-05 + 06-06.
- Existing 06-01..06-07 PLAN.md files are NOT modified or renumbered.
- After this plan: `06-VERIFICATION.md` gaps[1] (Bug 3) status moves from `failed` to `closed`; gaps[0] (BYO bootstrap completes end-to-end) moves from `partial` to `closed`. Plan 06-07 attempt-2 then closes gaps[2] (10-minute stopwatch + Hetzner cost + resource IDs).
</success_criteria>

<output>
After completion, create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-08-byo-register-runner-runas-fix-SUMMARY.md` documenting:
- RED commit hash + which tests fail and why (the `sudo -u runnerkit-runner ./config.sh` literal in the unmodified Sprintf format string).
- GREEN commit hash + diff summary of the two-line change in `script.go` (lines 47 + 83).
- Confirmation that all RED tests now PASS and all existing tests STAY GREEN.
- Confirmation that `internal/provider/hetzner/...` is unmodified and its tests stay green.
- Confirmation that `RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/...` passes both `TestApply_DownloadRunner_RealShell` and the new `TestApply_RegisterRunner_RootOnlyNopasswd`.
- Note that RenderRemoveConfigScript and RenderReconfigureScript STILL use `sudo -u %s` — out of scope for Bug 3 per the gap doc Task F bounds; flag whether to file a follow-up todo for consistency.
- Confirmation that the smoke harness extension (`.runner` sentinel) landed in `scripts/smoke/byo-permission.sh`.
- Pointer to Plan 06-07 attempt-2 — the live-smoke re-run human-action checkpoint that this plan unblocks.
- Quote-handling note: the rendered shell uses double-quoted `bash -c "..."` body so `$RUNNERKIT_REGISTRATION_TOKEN` is interpolated by the OUTER SSH-user shell BEFORE `su` invokes the inner shell; the env-var reference (NOT the literal token) reaches `config.sh`. Token-leak invariant preserved.
</output>
