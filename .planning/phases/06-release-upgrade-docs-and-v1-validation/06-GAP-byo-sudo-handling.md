---
status: open
source: live BYO smoke (Plan 06-04 Task 4); re-smoke (Plan 06-07 attempt 1)
discovered: 2026-05-04
updated: 2026-05-05
phase: 06-release-upgrade-docs-and-v1-validation
gap_closure_target: 06-05 (Bug 1+2 + Tasks A,E — CLOSED 2026-05-04 commit ee5c0a2); 06-06 (Tasks B,C,D — CLOSED 2026-05-05 commit 08b8708); 06-08 (Bug 3 + Task F — CLOSED 2026-05-05 commit bdef940); 06-09 (Bugs 4+5+6+7+8 + Tasks G,H,I,J,K — OPEN, mandatory before v1.0.0 tag)
severity: high
type: bug + missing-feature
related_decisions: [D-04 (live BYO smoke), Phase 2 context (service must not run as root), 02-01 (preflight separate behind remote.Executor)]
---

# Gap: BYO bootstrap is unusable — sudo handling AND download permission bug

## What's broken

During the Plan 06-04 live BYO smoke against `salar@mckee-small-desktop`,
`runnerkit up` failed with the opaque error:

```
ERROR RunnerKit could not apply the BYO runner install plan.
NEXT Review the remote host output, fix the issue, and re-run runnerkit up.
exit status 4
```

Root cause: the BYO host's `salar` user has sudo access **with a password
prompt**, not NOPASSWD. The bootstrap commands run via SSH and cannot
respond to a sudo password prompt, so the very first sudo-prefixed
command (`fix_dependencies`) fails silently from the user's perspective.

**The preflight at `internal/preflight/checks.go:127` falsely passed**
because `host.privilege` only tests `probe.Commands["sudo"]` — i.e. "is
the `sudo` binary installed". It does not test whether the SSH user can
actually run sudo non-interactively.

Two compounding issues:

1. **Preflight bug:** sudo passwordless availability is not actually checked.
2. **Missing feature:** there is no documented path for users to either
   (a) prepare the host for passwordless bootstrap, or (b) interactively
   provide a sudo password during `runnerkit up`. Users who hit this
   today must manually edit `/etc/sudoers.d/` with no guidance.

## Decision (user, 2026-05-04)

Two complementary solutions: **Path B + Path C**.

- **Path C is the recommended default.** A new `runnerkit byo-prepare
  --host user@host` command runs once with one local sudo-password
  prompt, installs a scoped sudoers entry on the host, and from then on
  every `runnerkit up` against that host runs passwordlessly.
- **Path B is the fallback.** If `byo-prepare` has not been run on the
  host (i.e. the preflight detects sudo password is required),
  `runnerkit up` automatically falls back to interactive password
  prompting over SSH (TTY allocation + `sudo -S`). This keeps the
  zero-host-setup path working without making users edit sudoers
  manually.

**Rejected:** Path A (document NOPASSWD as the contract, force users to
edit sudoers manually) — too high-friction for solo developers who are
the target audience.

## Bug 2 — `download_runner` permission failure

After NOPASSWD sudo was configured on `salar@mckee-small-desktop` to
test past Bug 1, the bootstrap then failed at the `download_runner`
step with:

```
curl: (23) Failure writing output to destination
Warning: Permission denied
```

Root cause: the `download_runner` script in
`internal/bootstrap/install.go:74` creates the install directory
**owned by `runnerkit-runner` (mode 0755 default)** then runs plain
`curl`, `sha256sum`, and `tar` *without sudo* as the SSH user. The SSH
user does not own the directory and 0755 grants `other` read+execute
only — no write. The download cannot land.

The same buggy pattern exists in `internal/bootstrap/script.go`:
- `RenderInstallScript` (lines 35-48): `sudo install -d -o serviceUser`
  → `cd` → plain `curl` → plain `sha256sum` → plain `tar`.
- `RenderEphemeralInstallScript` (lines 65-84): same pattern.

This bug never surfaced before because every test in
`internal/bootstrap/` uses `fakeExecutor` which records commands but
does not execute them. There is no integration test that runs the
actual scripts against a real shell, so the permission contradiction
went undetected from Plan 02-02 through Phase 5.

Anyone running `runnerkit up` against a BYO host today, regardless of
sudo configuration, will hit this. **BYO is non-functional in v1
without a fix.**

### Bug 2 fix

In `internal/bootstrap/install.go::Apply` (and `ApplyEphemeral`) and
in `internal/bootstrap/script.go::RenderInstallScript` (and
`RenderEphemeralInstallScript`), the install-directory pre-stage and
the download/verify/extract sequence must run with consistent
ownership. Two acceptable approaches:

**Option 1 (preferred — minimal diff):** prefix `curl`, `sha256sum -c`,
and `tar xzf` with `sudo`. The directory remains owned by
`runnerkit-runner`; root writes the tarball into it; ownership stays
correct without an extra `chown` round-trip.

**Option 2:** `install -d` as the SSH user, then `chown -R
runnerkit-runner:runnerkit-runner` at the end of the configure step.
Slightly cleaner conceptually but adds an extra sudo call.

Either way the fix needs:

- An end-to-end test that exercises a real shell (or a tighter
  fakeExecutor that simulates filesystem permissions), so future
  regressions surface before live smoke.
- A scripted shell smoke (`scripts/smoke/byo-permission.sh` is the
  obvious extension point) that validates `download_runner` lands the
  tarball and the extraction succeeds, before the configure step.

## Bug 3 — `register_runner` runas mismatch (discovered 2026-05-05)

After Plans 06-05 + 06-06 landed (Bug 1 + Bug 2 + Tasks A-E closed), the
Plan 06-07 re-smoke against `salar@mckee-small-desktop` failed at the
`register_runner` step with:

```
Remote stderr (unknown): sudo: a terminal is required to read the password;
either use the -S option to read from standard input or configure an
askpass helper sudo: a password is required
```

The smoke host had system-wide sudoers:

```
salar ALL=(ALL : ALL) ALL          # password required
salar ALL=(root) NOPASSWD: ALL     # runas=root only
```

The bootstrap got past `download_runner` (Bug 2 fix verified — `config.sh`
extracted to `/opt/actions-runner/runnerkit-<owner>-<repo>-local/`), then
failed at `register_runner` because `internal/bootstrap/script.go:47, 83`
runs:

```bash
sudo -u runnerkit-runner RUNNERKIT_REGISTRATION_TOKEN="..." ./config.sh \
  --unattended --url ... --token ... --name ... --labels ... --work ... --replace
```

`sudo -u runnerkit-runner` runs as a **non-root** user. With `(root) NOPASSWD: ALL`,
only runas=root is passwordless; runas=runnerkit-runner matches `(ALL:ALL) ALL`
(password required). Same problem with the `byo-prepare` scoped sudoers
template (`internal/bootstrap/sudoers.go:23`) which uses `ALL=(root) NOPASSWD:`
— even with `byo-prepare` run, `register_runner` still requires a password.

### Why cloud is unaffected

Hetzner cloud-init (`internal/provider/hetzner/provision.go:241`) configures
`runnerkit-admin` with `sudo: ALL=(ALL) NOPASSWD:ALL` — `(ALL)` runas covers
runas=runnerkit-runner. Cloud path works; BYO path does not. v1.0.0 cannot
ship with BYO non-functional.

### Bug 3 fix options

**Option 2 (preferred — minimal blast radius):** drop the `sudo -u runnerkit-runner`
prefix from `register_runner`. Run `config.sh` via `su -s /bin/bash - runnerkit-runner -c '<cmd>'`
inside an existing `sudo` context. `su` runs from root → no `(ALL)` runas
needed → works on BYO host with only root NOPASSWD AND on cloud host with
broader NOPASSWD. The byo-prepare scoped sudoers entry needs no change.
Forms:

```bash
sudo su -s /bin/bash - runnerkit-runner -c \
  "RUNNERKIT_REGISTRATION_TOKEN='...' ./config.sh --unattended ..."
```

**Option 1 (rejected — high complexity):** expand byo-prepare scoped sudoers
to `(ALL) NOPASSWD:` runas plus a per-repo allowlist that includes the
runtime-derived install dir's `config.sh`. Path is repo-derived (`runnerkit-<owner>-<repo>-local`)
so the allowlist must be regenerated per `runnerkit up` invocation, defeating
the "one-time prepare" promise.

**Option 3 (rejected — semantics drift):** run `config.sh` as root with
`HOME=/home/runnerkit-runner`. Risk: GitHub runner registration may write
user-specific files (e.g. `.runner` ownership) differently when invoked
from root vs the target user. Untested upstream.

### Bug 3 acceptance

- `register_runner` step succeeds against a BYO host whose user has ONLY
  `(root) NOPASSWD: ALL` (or only the byo-prepare scoped sudoers entry).
- Cloud path remains green — same `register_runner` script path works on
  cloud where `runnerkit-admin` has broader `(ALL) NOPASSWD: ALL`.
- A bootstrap unit test asserts the new shell form does NOT contain the
  literal token `sudo -u`. Closes the regression-detection gap.
- The integration test from Task E is extended (or paralleled) with a
  fixture that simulates `runnerkit-runner` not being root → asserts the
  new approach works without `(ALL)` runas in the host's sudoers.

## Required work

### Task A — Fix preflight (the actual bug)

`internal/preflight/checks.go::CheckPrivilege`

Replace the binary-existence check with a real probe:

```go
// Pseudocode:
result := exec.Run(ctx, target, remote.Command{ID: "probe_sudo_n", Script: "sudo -n true"})
switch {
case result.ExitCode == 0:
    pass("host.privilege", "passwordless sudo available")
case strings.Contains(result.Stderr, "password is required"):
    pass("host.privilege.password_required", "sudo requires password — runnerkit will prompt or use byo-prepare")
    // emits the warning that triggers Path B fallback or Path C suggestion
case strings.Contains(result.Stderr, "may not run sudo"):
    failure("host.privilege.no_sudo", "user not in sudoers", "Add the SSH user to sudoers or pick a host where they are.")
default:
    failure("host.privilege", "sudo probe failed: "+result.Stderr, ...)
}
```

Tests live alongside `internal/preflight/checks_test.go`. Use the existing
`testsupport/remote` fakes.

**Surface remote stderr in the bootstrap_failed CLI message.** Today the
`internal/cli/up.go:224` error swallows the underlying `err` from
`bootstrap.Apply`. Even when Paths B/C are wired, surfacing the
remote command stderr (with redaction) on bootstrap failures makes
diagnosis 10× faster. Preserve redactor invariants when doing so.

### Task B — Path B: interactive sudo password fallback

When preflight reports `host.privilege.password_required`:

- If `--non-interactive`: fail with clear remediation pointing at
  `runnerkit byo-prepare`.
- Otherwise: prompt locally for the host's sudo password (TTY-only,
  never logged, never written to state).
- Wrap each remote command's script in `sudo -S` and pipe the password
  via stdin per command.
- Register the password with the redactor (`redact.SudoPassword`)
  immediately so any accidentally-logged stderr scrubs it.
- After bootstrap completes (success OR failure), zero the password
  buffer and unregister.

Files affected:
- `internal/cli/up.go` (interactive prompt + flag wiring)
- `internal/bootstrap/install.go` (commands accept a sudo-password
  channel; passes via `sudo -S`)
- `internal/redact/` (new redaction kind)
- `internal/cli/up_test.go` (table tests for `--non-interactive` + interactive paths)

Constraint: `--yes` does NOT imply non-interactive sudo. `--yes` accepts
runnerkit's safe defaults; sudo password is a separate human-input concern.

### Task C — Path C: `runnerkit byo-prepare`

New top-level command. Single responsibility: install the scoped sudoers
entry and verify passwordless sudo afterwards.

Behavior:

1. SSH to `--host user@host` with TTY allocation.
2. Prompt locally once for the sudo password.
3. Idempotent: read existing `/etc/sudoers.d/runnerkit-installer` and
   verify the contents match the desired set; if so, exit 0 with "already
   prepared".
4. Otherwise, write `/etc/sudoers.d/runnerkit-installer` (mode 0440):

   ```
   # /etc/sudoers.d/runnerkit-installer (managed by runnerkit byo-prepare)
   <user> ALL=(root) NOPASSWD: \
     /usr/bin/apt-get, /usr/bin/dnf, /usr/bin/yum, \
     /usr/sbin/useradd, \
     /usr/bin/install, \
     /bin/tar, /usr/bin/tar, \
     /bin/systemctl, /usr/bin/systemctl, \
     /opt/runnerkit-runner/svc.sh
   ```

5. `visudo -c` validation before persisting (write to a temp file,
   `visudo -cf <tmp>`, atomically rename only on pass — bad sudoers can
   lock the user out of sudo entirely).
6. Run the same `sudo -n true` probe to verify the install worked.
7. Exit 0 + print follow-up: "host is now prepared; run `runnerkit up`."

Reverse operation: `runnerkit byo-prepare --host user@host --remove`
deletes the sudoers entry and verifies removal.

Doctor integration: `runnerkit doctor` against a BYO host detects the
sudoers file presence and reports it as a "byo_host_prepared" finding.

Files:
- `internal/cli/byo_prepare.go` (new command)
- `internal/cli/byo_prepare_test.go`
- `internal/bootstrap/sudoers.go` (sudoers template + visudo runner)
- `internal/bootstrap/sudoers_test.go`
- `internal/cli/root.go` (register command)
- `docs/byo-quickstart.md` (Sudo Setup section + decision tree)
- `docs/troubleshooting/bootstrap.md` (RKD-BOOT-NNN code for the
  preflight password-required warning, Path B/C remediation)
- `internal/errcodes/codes.go` (new RKD code)

### Task D — Documentation

- `docs/byo-quickstart.md`: top-level "Sudo Setup" section with the
  Path C → Path B decision tree.
- `docs/troubleshooting/bootstrap.md`: new RKD-BOOT-NNN entry for "sudo
  password required" with remediation pointing at `byo-prepare`.
- README.md: one-liner under BYO install pointing at the sudo setup.

### Task E — Fix download_runner permission bug

`internal/bootstrap/install.go::Apply` step `download_runner` and
`internal/bootstrap/script.go::RenderInstallScript` /
`RenderEphemeralInstallScript`:

- Prefix `curl`, `sha256sum -c`, and `tar xzf` with `sudo` (Option 1
  above), OR restructure ownership (Option 2).
- Add an integration test that exercises the actual scripts against a
  real shell with a tmpfs sandbox and asserts the tarball + extraction
  land in a directory owned by `runnerkit-runner`. This closes the gap
  that fakeExecutor-based unit tests left.
- Extend `scripts/smoke/byo-permission.sh` to assert the install dir
  contains `config.sh` after the bootstrap apply (verifies the
  download landed, before going on to runner registration).

Files affected:
- `internal/bootstrap/install.go` (Apply + ApplyEphemeral)
- `internal/bootstrap/script.go` (RenderInstallScript + RenderEphemeralInstallScript)
- `internal/bootstrap/install_integration_test.go` (new — guarded by build tag, exercised in `make test-integration`)
- `scripts/smoke/byo-permission.sh` (additional assertion)
- `internal/bootstrap/script_test.go` (substring assertions updated to include `sudo curl`/`sudo tar` if Option 1 chosen)

### Task F — Fix register_runner runas mismatch (Bug 3)

`internal/bootstrap/script.go::RenderInstallScript` and
`RenderEphemeralInstallScript` (lines 47, 83):

- Replace `sudo -u runnerkit-runner ./config.sh ...` with
  `sudo su -s /bin/bash - runnerkit-runner -c '...'` so the registration
  call runs from a root sudo context with no `(ALL)` runas requirement.
- Update `internal/bootstrap/script_test.go` substring assertions: assert
  presence of `su -s /bin/bash` and absence of `sudo -u`.
- Extend the Task E integration test (`install_integration_test.go`) with
  a sub-case that asserts the new shell form succeeds when the SSH user's
  sudoers has only `(root) NOPASSWD: ALL` — i.e. byo-prepare scoped entry
  alone is sufficient with no `(ALL)` runas.
- Re-run the Plan 06-07 BYO smoke against the same `salar@mckee-small-desktop`
  host that exposed the bug; assert the registration step lands a runner
  ID before destroy.

Files affected:
- `internal/bootstrap/script.go` (RenderInstallScript + RenderEphemeralInstallScript registration line)
- `internal/bootstrap/script_test.go` (substring assertions)
- `internal/bootstrap/install_integration_test.go` (extend with no-ALL-runas fixture)
- `scripts/smoke/byo-permission.sh` (optional: assert runner ID file exists post-bootstrap)

Cloud path note: `internal/provider/hetzner/provision.go:241` cloud-init
keeps `(ALL) NOPASSWD: ALL` — broader than needed but harmless. Optional
downscope: tighten to match the byo-prepare scoped set + `su` form, but
out of scope for Bug 3 closure (no observed impact, no v1.0.0 blocker).

## Acceptance

- [ ] Preflight rejects "sudo binary present but password required"
      cleanly with a clear remediation message and a finding ID.
- [ ] `bootstrap_failed` errors surface the underlying remote command
      stderr (redacted) so users can self-diagnose.
- [ ] `runnerkit byo-prepare --host user@host` installs scoped sudoers
      idempotently, validates with `visudo -c`, supports `--remove`.
- [ ] `runnerkit up --host user@host` against a host that has NOT run
      `byo-prepare` automatically prompts for sudo password (Path B).
- [ ] `runnerkit up --non-interactive --host user@host` against an
      unprepared host fails with remediation pointing at `byo-prepare`.
- [ ] `runnerkit doctor --host user@host` reports byo-prepared status.
- [ ] `internal/preflight/checks.go::CheckPrivilege` test coverage:
      passwordless / password-required / not-in-sudoers / sudo-missing.
- [ ] BYO quickstart docs updated.
- [ ] No raw sudo password leaks into state files, logs, JSON output,
      or error messages — redactor coverage tested.
- [ ] `download_runner` (and `RenderInstallScript`,
      `RenderEphemeralInstallScript`) write the tarball with consistent
      ownership; salar (or any non-runnerkit-runner SSH user) runs
      `runnerkit up` against a fresh BYO host successfully end-to-end.
- [ ] New integration test (or shell-backed bootstrap test) runs the
      actual install scripts against a real shell — the
      fakeExecutor-only test suite gap that hid this bug since 02-02
      is closed.
- [ ] (Bug 3) `register_runner` step succeeds against a BYO host whose
      user has ONLY `(root) NOPASSWD: ALL` (or only the byo-prepare
      scoped sudoers entry). No `(ALL)` runas required in host sudoers.
- [ ] (Bug 3) Bootstrap script unit test asserts absence of `sudo -u`
      in `register_runner` shell form; presence of `su -s /bin/bash`.
- [ ] (Bug 3) Plan 06-07 re-smoke against `salar@mckee-small-desktop`
      lands a GitHub runner ID before destroy.

## Bug 4 — `ui.Prompter` interface has no concrete implementation (discovered 2026-05-05)

After Plan 06-08 landed (Bug 3 closed), Plan 06-07 attempt-2 was started
against `salar@mckee-small-desktop` with the attempt-1 workaround file
`/etc/sudoers.d/runnerkit-smoke-temp` removed. Both `runnerkit byo-prepare`
(Path C) and the implicit `runnerkit up` Path B fallback fail immediately
with the misleading error:

```
✗ RunnerKit needs a sudo password but no TTY is available.
→ Run runnerkit byo-prepare from an interactive terminal.
```

The error fires even on a real native terminal where `os.Stdin` IS a
TTY. Root cause: `internal/ui/prompt.go` defines only the `Prompter`
interface and the optional `PasswordPrompter` capability — there is no
concrete implementation. `cmd/runnerkit/main.go` lines 19-33 wire
`Dependencies` with `In/Out/Err/TTY/Clock/CommandRunner` but **never set
the `Prompts` field**, so `deps.Prompts == nil` in production binaries.

The branch at `internal/cli/byo_prepare.go:72` and
`internal/cli/up.go:2032` is:

```go
if !deps.TTY.StdinTTY || deps.Prompts == nil {
    _ = renderer.Error("input_required", "RunnerKit needs a sudo password but no TTY is available.", ...)
}
```

The condition triggers because `Prompts == nil`, but the rendered
message is "no TTY" — leading the user to believe their terminal is
broken. Both Path B and Path C are unreachable in real binaries; only
unit tests (which inject a fake `Prompter`) ever exercise these paths.

### Why cloud is unaffected (again)

Hetzner cloud-init configures the cloud host with `(ALL) NOPASSWD: ALL`
during provisioning, so the `runnerkit up --cloud` flow never triggers
the Path B preflight branch — `CheckPrivilegePasswordReq` reports the
host is passwordless and the prompter is never consulted.

### Bug 4 fix

Implement a concrete CLI prompter that satisfies both `ui.Prompter`
(Confirm/Select) and `ui.PasswordPrompter` (Password with terminal
echo disabled via `golang.org/x/term`), wire it in `cmd/runnerkit/main.go`,
and update tests. The package `golang.org/x/term v0.10.0` is already
present as an indirect dep through `golang.org/x/sys`; promote it to
direct.

### Bug 4 acceptance

- `runnerkit byo-prepare --host user@host` against a real BYO host with
  password-protected sudo prompts ONCE for the sudo password and
  installs `/etc/sudoers.d/runnerkit-installer` on success.
- `runnerkit up --host user@host` against the same host (after the
  scoped sudoers entry exists OR with no preconfiguration via Path B)
  prompts for the sudo password and threads it through to the
  bootstrap.
- Unit test asserts `cmd/runnerkit/main.go` wires `Prompts` to a non-nil
  `ui.Prompter` AND that the implementation also satisfies
  `ui.PasswordPrompter` (assignable via type assertion).
- Plan 06-07 attempt-3 against `salar@mckee-small-desktop` lands a
  GitHub runner ID before destroy.

### Task G — Wire concrete `ui.Prompter` in main.go (Bug 4)

1. Create `internal/ui/cli_prompter.go` with a `CLIPrompter` struct
   that implements `Confirm` (stdin y/N parsing), `Select` (numbered
   list with stdin), and `Password` (`golang.org/x/term.ReadPassword`
   on `os.Stdin`'s fd, with newline restore).
2. Add `internal/ui/cli_prompter_test.go` covering: empty input → default,
   y/Y/yes acceptance, n/N/no rejection, Select numeric parsing + bounds,
   Password echo-suppression precondition (assert `term.IsTerminal` is
   consulted before reading).
3. Wire `Prompts: ui.NewCLIPrompter(os.Stdin, os.Stdout)` in
   `cmd/runnerkit/main.go` Dependencies struct.
4. Run `go mod tidy` to promote `golang.org/x/term` from indirect to
   direct.
5. (Optional polish) Update `byo_prepare.go:73` and `up.go:2037` error
   message: distinguish "no TTY" vs "prompter unavailable" so future
   wiring regressions surface a clearer message.

## Bug 5 — `mktemp` staging file owned by SSH user → root EACCES under fs.protected_regular=2 (discovered 2026-05-05)

After Bug 4 (Plan 06-09 commit 1d1888e) wired a concrete `ui.Prompter`,
Plan 06-07 attempt-3 against `salar@mckee-small-desktop` (Ubuntu 24.04
LTS) progressed past the prompt and surfaced a 5th BYO blocker on the
remote sudoers install:

```
✗ RunnerKit could not install the scoped sudoers entry.
→ Remote stderr: tee: /tmp/runnerkit-installer.HtDsXL: Permission denied
```

Root cause: `internal/bootstrap/sudoers.go::RemoteVisudoCheckScript`
creates the staging tempfile with plain `mktemp /tmp/runnerkit-installer.XXXXXX`
(unsudoed). The resulting file is owned by the SSH user (mode 0600).
The next pipeline step `printf '%s' "$RUNNERKIT_SUDOERS_CONTENT" | sudo tee "$TMP"`
runs as root, but Ubuntu 24.04 LTS ships with the kernel hardening
default `fs.protected_regular=2`, which disallows O_CREAT-open of files
in world-writable sticky directories (e.g. `/tmp`) by any process whose
UID differs from the file owner — root is NOT exempt from this protection.
`tee` therefore fails with EACCES, and `byo-prepare` aborts.

The bug went undetected before now because the unit test for the
script only asserts `visudo -cf` runs before `mv`; the live BYO smoke
in v1 had previously been worked around with `(ALL) NOPASSWD: ALL`
(attempt 1), so byo-prepare itself never ran end-to-end against a
real Ubuntu 24.04 host.

### Why cloud is unaffected (yet again)

Hetzner cloud-init writes `/etc/sudoers.d/runnerkit-admin` directly
during provisioning; it never goes through `RemoteVisudoCheckScript`.
The cloud path bypasses the staging tempfile entirely.

### Bug 5 fix

One-line patch: change `TMP=$(mktemp /tmp/runnerkit-installer.XXXXXX)`
to `TMP=$(sudo mktemp /tmp/runnerkit-installer.XXXXXX)` in
`RemoteVisudoCheckScript`. After the wrapper substitutes `sudo ` →
`sudo -S `, the tempfile is created as root (which can re-open and
write any file under `/tmp` without tripping `fs.protected_regular`).
Subsequent `sudo tee/visudo/chmod/mv` operations then succeed end-to-end.

### Bug 5 acceptance

- `runnerkit byo-prepare --host user@host` against an Ubuntu 24.04 LTS
  host with `fs.protected_regular=2` (kernel default) successfully
  writes `/etc/sudoers.d/runnerkit-installer` after a single sudo
  password prompt.
- Unit test `TestRemoteVisudoCheckScript_MktempInvokedViaSudo` asserts
  the staging tempfile is created via `sudo mktemp`.
- Plan 06-07 attempt-3 against `salar@mckee-small-desktop` proceeds
  past byo-prepare into the smoke proper.

### Task H — `sudo mktemp` in RemoteVisudoCheckScript (Bug 5)

1. Update `internal/bootstrap/sudoers.go::RemoteVisudoCheckScript` line 58
   to invoke mktemp via sudo. Update the function doc comment to record
   the rationale (Ubuntu 24.04 + fs.protected_regular=2).
2. Add `internal/bootstrap/sudoers_test.go::TestRemoteVisudoCheckScript_MktempInvokedViaSudo`
   asserting the script contains the literal substring `sudo mktemp `.
3. Run full repo `go test ./... -count=1 -race` to confirm no regression
   in any sibling package.

## Bug 6 — naive `strings.ReplaceAll("sudo ", "sudo -S ")` mangles `visudo ` (discovered 2026-05-05)

After Bug 5 (Plan 06-09 commit 62cdd2a) made the staging tempfile
root-owned, Plan 06-07 attempt-4 against `salar@mckee-small-desktop`
got past `sudo mktemp` and `sudo tee` and aborted at the `visudo`
validation step:

```
✗ RunnerKit could not install the scoped sudoers entry.
→ Remote stderr: visudo: invalid option -- 'S'
                 usage: visudo [-chqsV] [[-f] sudoers ]
                 visudo -S validation failed; sudoers entry not installed
```

Root cause: both `internal/bootstrap/install.go::wrapSudoCommand`
(line 98) and `internal/cli/byo_prepare.go::runByoPrepareInstall`
(line 99) call `strings.ReplaceAll(script, "sudo ", "sudo -S ")` —
which is naive and matches anywhere in the string, including the
trailing 5 chars of `visudo `. After the rewrite,
`sudo visudo -cf "$TMP"` becomes
`sudo -S visudo -S -cf "$TMP"`, and visudo rejects `-S` as an
unknown option.

The same naive substitution also mangles the in-script error message
`echo "visudo validation failed..."` into
`echo "visudo -S validation failed..."` — visible in the stderr quoted
above as `visudo -S validation failed; sudoers entry not installed`.

The bug went undetected because (a) the unit test asserts that
`visudo -cf` appears before `mv` in the un-rewritten script — but it
does not exercise the wrapper path; and (b) the bootstrap install
scripts (cmd/runnerkit `up`) happen not to contain any embedded
`sudo` substrings (no `visudo`, no `pseudo-something`), so the latent
mangling bug never triggered there.

### Why cloud is unaffected

Hetzner cloud-init configures `(ALL) NOPASSWD: ALL` so neither
`runnerkit up` Path B nor `runnerkit byo-prepare` is invoked on the
cloud path. The wrapper is never engaged → the buggy substitution
never runs → cloud bootstrap is unaffected.

### Bug 6 fix

Introduce a single source of truth for the sudo rewrite that uses a
word-boundary regex (`\bsudo `) so embedded substrings are preserved.
Add `internal/bootstrap/sudo_rewrite.go` exporting
`RewriteSudoForPasswordPipe(script string) string`. Replace both
naive `strings.ReplaceAll` callsites with calls to the helper.

### Bug 6 acceptance

- `runnerkit byo-prepare --host user@host` against an Ubuntu 24.04 LTS
  host with password-protected sudo lands `/etc/sudoers.d/runnerkit-installer`
  end-to-end (no `visudo: invalid option` error).
- Unit test `TestRewriteSudoForPasswordPipe_VisudoIsNotMangled` asserts
  the rewriter leaves `visudo ` intact while still upgrading
  standalone `sudo ` to `sudo -S `.
- Plan 06-07 attempt-5 against `salar@mckee-small-desktop` proceeds
  past byo-prepare into the smoke proper.

### Task I — Word-boundary sudo rewrite (Bug 6)

1. Create `internal/bootstrap/sudo_rewrite.go` with
   `RewriteSudoForPasswordPipe(script string) string` using
   `regexp.MustCompile(\`\bsudo \`)`.
2. Add `internal/bootstrap/sudo_rewrite_test.go` covering: standalone
   sudo at line start / after newline / after semicolon all
   rewritten; `visudo ` and `sudoers` substrings preserved; full
   `RemoteVisudoCheckScript()` round-trip preserves visudo.
3. Replace `strings.ReplaceAll(c.Script, "sudo ", "sudo -S ")` in
   `internal/bootstrap/install.go::wrapSudoCommand` with
   `RewriteSudoForPasswordPipe(c.Script)`. Drop the now-unused
   `"strings"` import.
4. Replace `strings.ReplaceAll(bootstrap.RemoteVisudoCheckScript(), ...)`
   in `internal/cli/byo_prepare.go::runByoPrepareInstall` with
   `bootstrap.RewriteSudoForPasswordPipe(bootstrap.RemoteVisudoCheckScript())`.
5. Run full repo `go test ./... -count=1 -race` — no regression in any
   sibling package.

## Bug 7 — preflight `sudo -n true` switch ignores `*exec.ExitError` (discovered 2026-05-05)

After Bug 6 (Plan 06-09 commit b1ce1c1) closed the visudo mangling,
Plan 06-07 attempt-5 against `salar@mckee-small-desktop` got past
byo-prepare and aborted at preflight:

```
ERROR host.privilege: sudo probe failed: sudo: a password is required
```

Root cause: `internal/preflight/checks.go::Run` (lines 148-165) probes
`sudo -n true` and classifies the result with a switch where every
case requires `probeErr == nil`. The real `internal/remote/system.go::SystemExecutor.Run`
returns `*exec.ExitError` for any non-zero remote exit (line 81-90),
so the probe with stderr "sudo: a password is required" lands here
with `probeErr != nil` and falls through to the default
"sudo probe failed: …" branch — emitting FAILURE instead of the
intended WARNING.

The gap-classifier was added in Plan 06-05 Task A but only tested
against `fakePreflightExecutor` (`internal/preflight/checks_test.go:21`)
which returns nil err for canned results. The fake doesn't reflect
the production executor's err semantics → tests passed against an
unrealistic prod model. Same family of "test fake masks prod bug"
issue as Bug 4.

### Bug 7 fix

Drop the `probeErr == nil` guard from the password/no-sudo cases; use
the stderr content as the discriminator. Keep `probeErr` as a
last-resort signal in the default branch (so genuine SSH transport
failures still produce a useful "sudo probe failed: …" message).

### Bug 7 acceptance

- `runnerkit up --host user@host` against an Ubuntu host with
  password-protected sudo correctly emits `host.privilege.password_required`
  WARNING and proceeds to the Path B prompt (rather than aborting at
  preflight with FAILURE).
- Two unit tests assert the WARNING/no-sudoers branches fire even when
  the executor returns a non-nil err alongside the matching stderr.

### Task J — Stderr-based privilege classification (Bug 7)

1. Edit `internal/preflight/checks.go::Run` switch (lines 148-165):
   drop `probeErr == nil` from the password/no-sudo cases. Add a
   doc comment recording the exec.ExitError discriminator decision.
2. Add tests in `internal/preflight/checks_bugfix_test.go` covering
   ExitError + matching stderr → WARNING; ExitError + "may not run
   sudo" stderr → FAILURE (no_sudo).
3. Run full repo suite — no regression.

## Bug 8 — `curl -fsS` misclassifies anonymous github API rate-limit as connectivity failure (discovered 2026-05-05)

Plan 06-07 attempt-5 simultaneously surfaced:

```
ERROR host.network.github: Outbound HTTPS to GitHub failed.
NEXT  Allow HTTPS egress to https://github.com and https://api.github.com.
```

Root cause: `runNetworkCheck` calls
`curl -fsS https://api.github.com >/dev/null`. The `-f` flag makes
curl exit 22 on HTTP 4xx/5xx. `mckee-small-desktop`'s outbound IP
(99.241.162.100) had exhausted GitHub's anonymous API rate limit
(`x-ratelimit-remaining=0` → HTTP 403). The host's network IS
reachable; only the unauthenticated probe is rate-limited. The
preflight then misreports a connectivity problem.

The remediation message is also misleading — it tells the user to
"allow HTTPS egress" when egress is fine.

### Why cloud is unaffected

Hetzner cloud-init runs against a freshly-provisioned cloud server
whose IP has never hit GitHub's anonymous rate limit (each provision
gets a clean Hetzner IP). The bug is residential-network-specific.

### Bug 8 fix

Drop `-f` so `curl` exits 0 whenever the request completes at the
HTTP layer — distinguishing "network reachable" (any HTTP response)
from "transport error" (DNS, TLS, connection refused). Add `--max-time`
and `--connect-timeout` to keep the probe bounded.

### Bug 8 acceptance

- Preflight `host.network.github` PASSES against a host whose IP
  has hit the anonymous GitHub API rate limit (HTTP 403 response).
- Preflight still FAILS against a genuinely unreachable host (DNS
  failure, no route to host, TLS error).
- Regression test asserts the github + api probe scripts do NOT
  contain `-fsS` and DO contain `--connect-timeout`.

### Task K — Drop `-fsS` from network probes (Bug 8)

1. Edit `internal/preflight/checks.go` lines 179-180: change
   `curl -fsS` → `curl -sS --connect-timeout 5 --max-time 10 -o /dev/null`
   for both github.com and api.github.com probes. Add a doc comment.
2. Add `TestRunNetworkCheck_Script_DoesNotUseFailFlag` in
   `checks_bugfix_test.go` asserting the new flag set is in place.
3. Run full repo suite — no regression.

## Cross-references

- Discovered while running Plan 06-04 Task 4 BYO smoke. The smoke
  workaround for v1.0.0: maintainer manually adds NOPASSWD on the test
  host. This is the documented v1.0.0 contract until this gap closes.
- Bug 3 discovered 2026-05-05 during Plan 06-07 attempt 1 against the
  same `salar@mckee-small-desktop` host AFTER Plans 06-05 + 06-06 landed.
  Cloud path unaffected because Hetzner cloud-init configures `(ALL) NOPASSWD: ALL`
  which covers `runas=runnerkit-runner`. BYO non-functional in v1 until
  Task F lands.
- Once Bug 3 closed: the Plan 06-07 BYO + Hetzner smoke can re-run end-to-end
  without any host-side preconfiguration AND `register_runner` no longer
  requires `(ALL)` runas in host sudoers. Plan 06-08 is the gap-closure
  target. (CLOSED 2026-05-05 commit bdef940.)
- Bug 4 discovered 2026-05-05 during Plan 06-07 attempt 2 against the
  same host AFTER Plan 06-08 landed. Both `runnerkit byo-prepare` and
  `runnerkit up` Path B fail immediately because `cmd/runnerkit/main.go`
  never wires a concrete `ui.Prompter` — the production binary's
  `deps.Prompts` is always `nil`. Cloud path unaffected (cloud-init
  bypasses both prompts). BYO non-functional in v1 until Task G lands.
- Once Bug 4 closed: Plan 06-07 attempt-3 can finally run end-to-end on
  a host with password-protected sudo. Plan 06-09 is the gap-closure
  target.
- Bug 5 discovered 2026-05-05 during Plan 06-07 attempt-3 against the
  same host AFTER Bug 4 fix (Plan 06-09 commit 1d1888e) landed. The
  Ubuntu 24.04 default `fs.protected_regular=2` rejects root's O_CREAT
  on a non-root-owned file under `/tmp`. Cloud path unaffected (cloud-init
  writes sudoers.d directly, never uses the staging tempfile). Bundled
  with Bug 4 in Plan 06-09 closure since the smoke can't make end-to-end
  forward progress without both fixes.
- Once Bug 5 closed: byo-prepare progressed past mktemp/tee/chmod and
  reached the visudo gate, where Bug 6 surfaced.
- Bug 6 discovered 2026-05-05 during Plan 06-07 attempt-4 against the
  same host AFTER the Bug 5 fix landed. Affects both `runnerkit
  byo-prepare` (live trigger) and `runnerkit up` Path B install
  scripts (latent — happens not to contain any "Xsudo " sequences in
  the existing bootstrap scripts). Bundled with Bugs 4 + 5 in the Plan
  06-09 closure since the smoke can't make end-to-end forward progress
  without all three fixes.
- Once Bug 6 closed: byo-prepare runs to completion; Plan 06-07
  attempt-5 progressed to runnerkit up preflight where Bugs 7 + 8
  surfaced.
- Bugs 7 + 8 discovered 2026-05-05 during Plan 06-07 attempt-5 against
  the same host AFTER Bug 6 fix landed. Bug 7 was masked by an
  unrealistic test fake; Bug 8 is residential-network-specific
  (cloud paths see clean IPs and never trip the anonymous rate
  limit). Bundled with Bugs 4-6 in Plan 06-09 closure.
- Once Bugs 7 + 8 closed: preflight passes on byo-prepared Ubuntu 24.04
  hosts; Plan 06-07 attempt-6 proceeds to the actual smoke (BYO
  bootstrap → register_runner → status → doctor → down → Hetzner
  empty_precheck → cloud-end-to-end → destroy_verify).
- Related decisions: Phase 2 context (service must not run as root —
  unaffected; this gap is about *bootstrap-time* sudo, not runtime),
  D-04 (live BYO smoke — directly affected), Plan 02-02 (bootstrap pinned
  runner — unaffected).
