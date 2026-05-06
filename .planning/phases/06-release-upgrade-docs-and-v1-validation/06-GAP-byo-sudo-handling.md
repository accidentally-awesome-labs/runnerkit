---
status: open
source: live BYO smoke (Plan 06-04 Task 4); re-smoke (Plan 06-07 attempt 1)
discovered: 2026-05-04
updated: 2026-05-05
phase: 06-release-upgrade-docs-and-v1-validation
gap_closure_target: 06-05 (Bug 1+2 + Tasks A,E — CLOSED 2026-05-04 commit ee5c0a2); 06-06 (Tasks B,C,D — CLOSED 2026-05-05 commit 08b8708); 06-08 (Bug 3 + Task F — CLOSED 2026-05-05 commit bdef940); 06-09 (Bugs 4..17 + Tasks G..T — OPEN, mandatory before v1.0.0 tag)
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

## Bug 9 — `configure_runner` + `configure_ephemeral_runner` missing `Sudo: true` (discovered 2026-05-05)

After Bugs 7 + 8 (Plan 06-09 commit 9a08b01) closed, Plan 06-07
attempt-6 against `salar@mckee-small-desktop` got past preflight, the
Path B password prompt fired and accepted a password, and bootstrap
aborted at the `configure_runner` step:

```
sudo: a terminal is required to read the password; either use the
-S option to read from standard input or configure an askpass helper
sudo: a password is required
```

Root cause: `internal/bootstrap/install.go::Apply` line 118 and
`ApplyEphemeral` line 160 declare two `remote.Command` literals
without `Sudo: true`:

```go
{ID: "configure_runner", Script: RenderInstallScript(opts), Env: ..., RedactArgs: ...},
{ID: "configure_ephemeral_runner", Script: RenderEphemeralInstallScript(opts), Env: ..., RedactArgs: ...},
```

`wrapSudoCommand` short-circuits when `c.Sudo == false` (gating
"Path B should never wrap non-sudo commands"). The configure scripts
DO contain `sudo curl`, `sudo sha256sum`, `sudo tar`, `sudo chown`,
and `sudo su -s` (per Plan 06-05 Bug 2 fix and Plan 06-08 Bug 3 fix
in `script.go`), so the rendered script gets dispatched verbatim and
each raw sudo tries `/dev/tty` over the non-tty SSH session and fails.

Same family of bug as the original Bug 1 (Plan 06-05) — preflight
classification was tightened, but the bootstrap commands themselves
were left under-annotated. Subsequent fixes that added sudo-prefixed
calls into the configure step (Bug 2 / Bug 3) didn't update the
Command flag, and unit tests didn't assert it.

### Why cloud is unaffected

Hetzner cloud-init configures `(ALL) NOPASSWD: ALL`, so the
SudoPassword option is empty, so `wrapSudoCommand` returns the
command unchanged in both branches — Sudo flag has no effect. Cloud
bootstrap passes through.

### Bug 9 fix

Add `Sudo: true` to the configure_runner literal in `Apply` and the
configure_ephemeral_runner literal in `ApplyEphemeral`. Two-line
edit. Add unit tests covering both literals so future commands that
add sudo invocations to the configure scripts can't regress this.

### Bug 9 acceptance

- `runnerkit up --host user@host` against an Ubuntu host with
  password-protected sudo + Plan 06-06 Path B prompt threads the
  password through the configure_runner step (no `sudo: a terminal
  is required` aborts).
- Two unit tests assert `Sudo: true` on both configure commands.

### Task L — `Sudo: true` on configure commands (Bug 9)

1. Edit `internal/bootstrap/install.go::Apply` line 118 and
   `ApplyEphemeral` line 160: add `Sudo: true` to both
   `remote.Command` literals.
2. Add `TestApplyConfigureRunnerCommand_HasSudoTrue` and
   `TestApplyEphemeralConfigureCommand_HasSudoTrue` in
   `install_test.go`.
3. Run full repo `go test ./... -count=1 -race` — no regression.

## Bug 10 — `wrapSudoCommand` outer brace-pipe breaks inner `printf | sudo X` patterns (discovered 2026-05-05)

After Bug 9 (Plan 06-09 commit f195a83) closed the configure_runner
Sudo flag, Plan 06-07 attempt-7 against `salar@mckee-small-desktop`
got past the Path B prompt and failed at bootstrap with:

```
Sorry, try again.
sudo: no password was provided
sudo: 1 incorrect password attempt
```

Root cause: `internal/bootstrap/install.go::wrapSudoCommand` rendered
the wrapped script as:

```bash
printf '%s\n' "$RUNNERKIT_SUDO_PASSWORD" | { rewritten_script }
```

Inside the brace group, `RenderInstallScript` and
`RenderEphemeralInstallScript` (Plan 06-05 + 06-08) emit the inline
checksum-verify pattern:

```bash
printf '%%s  %%s\n' '<sha256>' '<file>' | sudo -S sha256sum -c -
```

The inner pipe opens its own stdin to `sudo -S`. On Ubuntu 24.04 LTS,
`/etc/sudoers` defaults include `use_pty` and a tty-scoped timestamp
cache; cred priming via `sudo -S X` (where X is the previous bootstrap
command in the same brace) did not reliably cache across the SSH
session, so the next sudo re-prompted. Without a fresh cache hit,
sudo's `-S` consumed the inner printf's checksum string as the
password attempt — emitting "Sorry, try again", then EOF, then
"no password was provided".

### Why byo-prepare worked but install.go did not

`internal/cli/byo_prepare.go::runByoPrepareInstall` uses a different
wrapper structure:

```bash
printf '%s\n' "$RUNNERKIT_SUDO_PASSWORD" | sudo -S -v
<rewritten_script>
```

byo-prepare primes cred ONCE via a dedicated `sudo -S -v` invocation
that consumes the password line, then runs the rewritten script
WITHOUT an outer pipe. Each subsequent `sudo -S` in the rewritten
script hits the freshly-primed cred and does not read its stdin —
so inner `printf | sudo X` patterns reach X correctly. The structure
asymmetry between byo-prepare (works) and install.go (fails) was the
asymmetry that hid Bug 10.

### Why cloud is unaffected

Hetzner cloud-init configures `(ALL) NOPASSWD: ALL`, so SudoPassword
stays empty, wrapSudoCommand returns commands unchanged, no outer
pipe is introduced. Cloud bootstrap is unaffected.

### Bug 10 fix

Align `wrapSudoCommand` with byo-prepare's structure: prime cred via
`printf | sudo -S -v` once, then run the rewritten script directly.
No outer brace-group pipe. Each subsequent `sudo -S` benefits from
the freshly-primed cache and does not consume its stdin.

### Bug 10 acceptance

- `runnerkit up --host user@host` against an Ubuntu 24.04 LTS host
  with password-protected sudo + Plan 06-06 Path B prompt threads
  the password through configure_runner's checksum verify step (no
  "Sorry, try again" / "no password was provided" aborts).
- Unit test asserts: inner `printf | sudo X` pipe preserved
  end-to-end; no outer brace-group pipe; dedicated `sudo -S -v`
  cred-priming invocation present.
- Plan 06-07 attempt-8 against `salar@mckee-small-desktop` lands a
  GitHub runner ID before destroy.

### Task M — Cred-priming wrap structure (Bug 10)

1. Edit `internal/bootstrap/install.go::wrapSudoCommand` to render
   the prime-then-run structure that mirrors byo-prepare. Update the
   doc comment.
2. Add `TestWrapSudoCommand_InnerStdinPipePreserved` in
   `install_test.go`.
3. Run full repo `go test ./... -count=1 -race` — no regression.

## Bug 11 — `sudo su -s /bin/bash -` login shell drops cwd, breaks `./config.sh` (discovered 2026-05-06)

After Bug 10 (Plan 06-09 commit 281966d) closed the password-pipe
structure, Plan 06-07 attempt-8 against `salar@mckee-small-desktop`
got past every previous gate and aborted at config.sh:

```
-bash: line 1: ./config.sh: No such file or directory
```

Root cause: Plan 06-08's Bug 3 fix changed the register_runner form
from `sudo -u runnerkit-runner ./config.sh ...` to:

```
sudo su -s /bin/bash - runnerkit-runner -c "RUNNERKIT_REGISTRATION_TOKEN=... ./config.sh --unattended ..."
```

The literal `-` argument to `su` requests a LOGIN shell, which
re-establishes the user's HOME as cwd by design. The outer script
DOES `cd /opt/actions-runner/runnerkit-<name>` earlier, but the
inner `sudo su -` re-anchors cwd to runnerkit-runner's HOME (which
useradd --create-home placed at `/home/runnerkit-runner`). When the
inner shell runs `./config.sh`, it looks for it under HOME, fails.

Bug 3's correctness for runas was preserved; the cwd-inheritance
contract from the prior `sudo -u user` form was inadvertently
dropped.

### Why cloud is unaffected

Same reason as Bugs 9 + 10: cloud-init grants full NOPASSWD and
runs the bootstrap fresh via cloud-init. The `sudo su -` form fires
on cloud too, but cloud cloud-init doesn't use the bootstrap
`runnerkit up` codepath — it constructs its own register_runner
command directly with the correct cwd. The bug is BYO-only.

### Bug 11 fix

Prepend `cd <installPath> && ` to the inner `-c` argument of both
`sudo su -s /bin/bash -` invocations in `RenderInstallScript` and
`RenderEphemeralInstallScript`. The su login shell still resets HOME
correctly, but the explicit cd inside the -c arg ensures
`./config.sh` resolves against the install dir. install dir is
already `chown`ed to runnerkit-runner so the user has access.

### Bug 11 acceptance

- `runnerkit up --host user@host` against an Ubuntu host runs
  config.sh under runnerkit-runner from the correct cwd; .runner
  sentinel file lands; `.runner` assertion in
  `scripts/smoke/byo-permission.sh` passes.
- Two unit tests assert the inner -c arg starts with
  `cd <installPath> && ` for both renderers.

### Task N — `cd <installPath>` inside `-c` arg (Bug 11)

1. Edit `internal/bootstrap/script.go::RenderInstallScript` line 53
   and `RenderEphemeralInstallScript` line 95: prepend `cd %[2]s && `
   to the -c argument body.
2. Add `TestRenderInstallScriptCdsBeforeConfigSh` and
   `TestRenderEphemeralInstallScriptCdsBeforeConfigSh` in
   `script_test.go`.
3. Run full repo `go test ./... -count=1 -race` — no regression.

## Bug 12 — `ServiceNotActiveError` swallows remote stderr (discovered 2026-05-06)

After Bug 11 (Plan 06-09 commit beef841) closed the config.sh cwd
issue, Plan 06-07 attempt-9 against `salar@mckee-small-desktop`
aborted with the bare-bones message:

```
ERROR RunnerKit installed the runner but the service is not active.
NEXT  Run sudo ./svc.sh status in the runner directory or re-run runnerkit up after fixing the service.
```

The actual remote stderr from the failing svc.sh / install_service /
verify_service step was never surfaced. `internal/bootstrap/install.go`
returned `ServiceNotActiveError{Err: err}` — the struct only carried
the wrapped err, not the failing command's ID or stderr. `up.go`'s
handler then emitted a generic remediation that left the user
unable to diagnose root cause without a separate SSH session.

This is a UX bug, not a behavioral one — RunnerKit's bootstrap was
working correctly enough to land a runner unit (the actual systemd
service IS in `active running` state when probed directly). Whatever
the failing command actually reported was the only signal that could
have explained the false negative.

### Why cloud is unaffected

Cloud bootstrap uses the same ServiceNotActiveError plumbing (see
up.go cloud branch around line 749), but cloud-init's install +
service-start runs synchronously inside cloud-init before runnerkit
takes over, so verify_service tends to find the service already
active. The bug is dormant on the cloud path; nevertheless the same
remediation enrichment is applied to keep both paths symmetric.

### Bug 12 fix

Extend `ServiceNotActiveError` with `CommandID` and `Stderr` fields.
`Apply` and `ApplyEphemeral` populate both when a service-related
step exits non-zero. `up.go`'s four ServiceNotActiveError handlers
(BYO/cloud × persistent/ephemeral) append the stderr and CommandID
to the remediation list when present.

### Bug 12 acceptance

- When install_service / verify_service / install_ephemeral_service /
  install_ephemeral_ttl_timer / verify_ephemeral_service exits
  non-zero on a real host, the user sees the failing step's actual
  remote stderr in the CLI output (subject to redactor scrubbing).
- Two unit tests assert: ServiceNotActiveError exposes CommandID +
  Stderr; Apply populates both when install_service exits non-zero.

### Task O — Surface remote stderr in service errors (Bug 12)

1. Edit `internal/bootstrap/install.go::ServiceNotActiveError` to
   add CommandID + Stderr fields. Update Apply + ApplyEphemeral to
   populate both at the failing-step branch.
2. Edit `internal/cli/up.go` ServiceNotActiveError handlers (BYO
   persistent + BYO ephemeral + cloud persistent + cloud ephemeral)
   to append "Remote stderr (CmdID): ..." to the remediation list
   when serviceErr.Stderr is non-empty.
3. Add `TestServiceNotActiveError_CarriesCommandIDAndStderr` and
   `TestApply_ServiceFailureSurfacesStderrInError` in
   `install_test.go`.
4. Run full repo `go test ./... -count=1 -race` — no regression.

## Bug 13 — stale `.runner` blocks re-registration (discovered 2026-05-06)

After Bug 12 (Plan 06-09 commit 248c68b) made remote stderr visible
in service errors, Plan 06-07 attempt-10 against
`salar@mckee-small-desktop` revealed the actual cause of attempt-9's
service abort:

```
Cannot configure the runner because it is already configured.
To reconfigure the runner, run 'config.cmd remove' or
'./config.sh remove' first.
```

Root cause: bootstrap's register_runner step is not idempotent
against re-registration. When the install dir already contains the
`.runner` sentinel + `.credentials` + `.credentials_rsaparams` from
a prior successful registration, config.sh refuses to re-register
even with `--replace`. The `--replace` flag only removes the
GitHub-side runner record (preventing 409 on duplicate name); local
state must be removed separately.

This was hidden across attempts 1-9 because every prior attempt
failed BEFORE config.sh ran successfully even once — so `.runner`
was never written. Once Bugs 4-11 closed and config.sh actually ran
cleanly in attempt 9, attempt 10 saw the leftover state and
config.sh aborted on the second pass.

### Why cloud is unaffected

Each Hetzner cloud server is freshly provisioned per `runnerkit up`
invocation; there's no persistent state between runs. Bug 13 is
exclusively a BYO concern.

### Bug 13 fix

Insert `sudo rm -f .runner .credentials .credentials_rsaparams`
after `sudo chown` and BEFORE the `sudo su` config.sh invocation
in both `RenderInstallScript` and `RenderEphemeralInstallScript`.
The script already `cd %[2]s` for the chown, but explicitly cd
again before the rm to make the script's cwd contract explicit
under `set -euo pipefail`. `rm -f` is idempotent (no error on
absent files), so the line is a no-op on first install.

### Bug 13 acceptance

- `runnerkit up --host user@host` is idempotent against a host
  whose install dir already contains `.runner` + `.credentials`
  from a prior successful registration. config.sh re-registers
  cleanly and the smoke harness's `.runner` sentinel re-appears.
- Two unit tests assert the rm line is present and ordered before
  config.sh in both renderers.

### Task P — Idempotent re-registration (Bug 13)

1. Edit `internal/bootstrap/script.go::RenderInstallScript` and
   `RenderEphemeralInstallScript` to add a cd-then-rm step removing
   `.runner`, `.credentials`, `.credentials_rsaparams` before the
   `sudo su` config.sh invocation.
2. Add `TestRenderInstallScriptRemovesStaleRunnerStateBeforeConfig`
   and `TestRenderEphemeralInstallScriptRemovesStaleRunnerStateBeforeConfig`
   in `script_test.go`.
3. Run full repo `go test ./... -count=1 -race` — no regression.

## Bug 14 — `svc.sh install` refuses to overwrite stale systemd unit (discovered 2026-05-06)

After Bug 13 (Plan 06-09 commit ae1a702) made register_runner
idempotent, Plan 06-07 attempt-11 against `salar@mckee-small-desktop`
got past config.sh and aborted at `install_service` with the
now-visible remote stderr (Bug 12 fix):

```
Failed: error: exists
/etc/systemd/system/actions.runner.<repo>.<runner>.service
```

Root cause: `internal/bootstrap/script.go::RenderServiceScript`
invokes `sudo ./svc.sh install runnerkit-runner` without first
removing the existing systemd unit file. svc.sh install refuses to
overwrite. Re-runs of `runnerkit up` against a host where the unit
already exists abort at install_service.

Same family as Bug 13 (stale .runner state) but at the systemd
layer. Both are idempotency gaps that surfaced once attempts started
making forward progress — every prior attempt failed before reaching
service-install, so stale unit-file state was never observed.

### Why cloud is unaffected

Cloud servers are freshly provisioned per `runnerkit up`. No prior
unit can exist.

### Bug 14 fix

Insert idempotent stop + uninstall before install:

```bash
sudo ./svc.sh stop || true
sudo ./svc.sh uninstall || true
sudo ./svc.sh install runnerkit-runner
sudo ./svc.sh start
sudo ./svc.sh status
```

Each pre-step is `|| true`-suffixed so a first install (where no
unit/service exists) is not blocked by the stop/uninstall failing
on absent state.

### Bug 14 acceptance

- `runnerkit up` is idempotent against a host whose
  `/etc/systemd/system/actions.runner.<...>.service` already exists
  from a prior install. svc.sh install runs cleanly; smoke proceeds
  to verify_service.
- Unit test asserts: stop + uninstall both present; ordered before
  install; both `|| true`-suffixed.

### Task Q — Idempotent svc.sh install (Bug 14)

1. Edit `internal/bootstrap/script.go::RenderServiceScript` to
   prepend `sudo ./svc.sh stop || true` and
   `sudo ./svc.sh uninstall || true` before
   `sudo ./svc.sh install runnerkit-runner`.
2. Add `TestRenderServiceScriptIdempotentInstall` in `script_test.go`.
3. Run full repo `go test ./... -count=1 -race` — no regression.

## Bug 15 — `verify_service` Command lacks `cd <installPath>` (discovered 2026-05-06)

After Bug 14 (Plan 06-09 commit 7bc9b25) made svc.sh install
idempotent, Plan 06-07 attempt-12 against `salar@mckee-small-desktop`
got past install_service and aborted at verify_service:

```
sudo: ./svc.sh: command not found
```

Root cause: `internal/bootstrap/install.go::Apply` line 144 declares:

```go
{ID: "verify_service", Script: "set -euo pipefail\nsudo ./svc.sh status\n", Sudo: true}
```

Each `remote.Command` runs in a fresh SSH session whose default cwd
is the SSH user's HOME, NOT installPath. `./svc.sh` is relative to
cwd, so the lookup fails. install_service does NOT have this bug
because `RenderServiceScript` emits an explicit `cd <installPath>`
at the top; verify_service was inlined as a one-liner and the cd
was never added.

This was hidden across attempts 1-11 because install_service kept
failing first, so verify_service never ran. Once Bug 14 made
install_service pass, attempt-12 reached verify_service and exposed
the missing cd.

### Why cloud is unaffected

The cloud verify_service runs against the same install.go source,
but the cloud bootstrap path (cloud-init) handles install + start
synchronously and verify_service typically observes an already-active
state via `is-active` shortcuts. Even so the cd fix is BYO/cloud
agnostic and applies uniformly.

### Bug 15 fix

Add `cd <installPath>` (with `defaultString(opts.InstallPath, ...)`
default) to verify_service's Script before `sudo ./svc.sh status`,
mirroring RenderServiceScript's structure.

### Bug 15 acceptance

- `runnerkit up` against an Ubuntu host runs `sudo ./svc.sh status`
  from /opt/actions-runner/runnerkit-<name>/, so svc.sh is found
  and the service-active check returns 0.
- Unit test asserts the verify_service Command's Script contains
  `cd <installPath>` before `./svc.sh`.

### Task R — `cd` in verify_service Command (Bug 15)

1. Edit `internal/bootstrap/install.go::Apply` verify_service Command
   literal to prepend `cd <installPath>` to its Script.
2. Add `TestApply_VerifyService_CdsIntoInstallPathBeforeSvcSh` in
   `install_test.go`.
3. Run full repo `go test ./... -count=1 -race` — no regression.

## Bug 16 — `runnerOnlineWithLabels` case-sensitive match against GitHub auto-labels (discovered 2026-05-06)

After Bug 15 (Plan 06-09 commit ec19486) closed verify_service,
Plan 06-07 attempt-13 against `salar@mckee-small-desktop` completed
the bootstrap end-to-end but `waitForRunnerOnline` polled for 6
minutes and timed out:

```
ERROR RunnerKit could not verify the GitHub runner came online with the expected labels.
```

`gh api repos/accidentally-awesome-labs/dat0/actions/runners`
confirmed the runner WAS online with id 24 and labels:

```
["self-hosted", "Linux", "X64", "runnerkit",
 "runnerkit-accidentally-awesome-labs-dat0", "persistent"]
```

— exactly what RunnerKit registered, plus GitHub's auto-added
"Linux" + "X64" (CamelCase). RunnerKit's expected label set from
`labels.Build` includes lowercase "linux" + "x64" (slug() applies
strings.ToLower). `runnerOnlineWithLabels` does case-sensitive set
membership, so "linux" never matched "Linux" → poll timeout.

### Why cloud is unaffected

Cloud also hits this code path. The bug is universal — affects any
runner whose labels include OS or arch tokens. Hetzner cloud-init
runs Linux x64 servers, so cloud bootstrap had the same latent bug,
but Plan 06-04 attempt-1 didn't reach the online-check step (BYO
failed first), so the case mismatch was never observed. The fix
applies uniformly to both code paths since they share
`runnerOnlineWithLabels`.

### Bug 16 fix

Lowercase both sides before set membership in
`runnerOnlineWithLabels`. GitHub always emits OS + arch auto-labels
in canonical CamelCase; RunnerKit always lowercases. Both are
correct in their own world; the matching layer must normalize.

### Bug 16 acceptance

- `runnerkit up` against any host whose runner registers successfully
  exits the polling loop on the first iteration after the runner
  flips online, regardless of label case.
- Unit test asserts a runner with labels in CamelCase form
  (`Linux`, `X64`) matches expected lowercase labels.

### Task S — Case-insensitive label match (Bug 16)

1. Edit `internal/cli/up.go::runnerOnlineWithLabels` to lowercase
   both actual and expected labels before set membership.
2. Add `internal/cli/runner_online_test.go::TestRunnerOnlineWithLabels_CaseInsensitiveMatch`.
3. Run full repo `go test ./... -count=1 -race` — no regression.

## Bug 17 — `runnerNameConflict` pre-check refuses re-runs against our own runner (discovered 2026-05-06)

After Bug 16 (Plan 06-09 commit 91c45ff) fixed the case-insensitive
label match, Plan 06-07 attempt-14 against
`salar@mckee-small-desktop` failed at the up.go pre-bootstrap check:

```
ERROR RunnerKit can't continue because a GitHub runner named
      runnerkit-accidentally-awesome-labs-dat0-local already exists.
NEXT  Remove or rename the existing GitHub runner ...
```

Root cause: `internal/cli/up.go` line 214 (BYO) + 743 (cloud) calls
`gh.FindRunnerByName(runners, labelSet.RunnerName)` and refuses to
proceed if any runner with the deterministic name already exists.
The runner name is `runnerkit-<repo>-local` (or `-<random>` for
ephemeral) — deterministic per (repo, host, mode). Every re-run of
`runnerkit up` against the same target sees its own prior runner.
The pre-check is too strict for the idempotent re-run path that
Bugs 13 + 14 enabled at the bootstrap layer.

`config.sh --replace` (already in RenderInstallScript) handles the
GitHub-side runner-record replacement during registration. The
pre-check should only fire when the existing runner is unrelated
(no `runnerkit` label).

### Why cloud is unaffected

The cloud branch has the same code path (line 743) but cloud-init
provisions a fresh server every `up` — there's no persistent
GitHub-side state from prior runs because each cloud server has a
fresh hostname. The pre-check never fires on the cloud path.

### Bug 17 fix

Extract `isRunnerKitManagedRunner(r gh.Runner) bool` — returns true
iff the runner's label set contains `runnerkit` (case-insensitive).
Both BYO and cloud pre-bootstrap checks call it; only when the
existing runner is NOT ours do they invoke `runnerNameConflict`.

### Bug 17 acceptance

- `runnerkit up` is idempotent against a host whose deterministic
  runner is already registered on GitHub. Bootstrap proceeds and
  config.sh --replace handles the registration cleanup.
- A genuine name collision with an unrelated user-managed runner
  (no `runnerkit` label) still triggers the conflict error.
- Two unit tests assert: runners with `runnerkit` label are ours;
  runners without the label are not.

### Task T — Skip self-collision in pre-bootstrap check (Bug 17)

1. Edit `internal/cli/up.go`: extract `isRunnerKitManagedRunner`,
   call it at both pre-bootstrap conflict checks (BYO + cloud) so
   the runnerNameConflict only fires for foreign runners.
2. Add `TestIsRunnerKitManagedRunner_DetectsOurOwnRunner` and
   `TestIsRunnerKitManagedRunner_RejectsForeignRunner` in
   `runner_online_test.go`.
3. Run full repo `go test ./... -count=1 -race` — no regression.

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
- Once Bugs 7 + 8 closed: preflight passed; Path B prompt fired;
  bootstrap aborted at configure_runner where Bug 9 surfaced.
- Bug 9 discovered 2026-05-05 during Plan 06-07 attempt-6 against the
  same host AFTER Bugs 7 + 8 fixes landed. Affects only Path B
  bootstrap on hosts with password-protected sudo. Cloud path
  unaffected (cloud-init grants full NOPASSWD; SudoPassword stays
  empty; wrapSudoCommand returns commands unchanged).
- Once Bug 9 closed: Plan 06-07 attempt-7 progressed to bootstrap
  configure_runner where Bug 10 surfaced.
- Bug 10 discovered 2026-05-05 during Plan 06-07 attempt-7 against the
  same host AFTER Bug 9 fix landed. Affects only Path B bootstrap on
  hosts where sudo's tty-scoped cache does not persist across SSH
  sessions (Ubuntu 24 default with use_pty). The asymmetry between
  byo-prepare's working structure and install.go's broken structure
  hid Bug 10 from earlier reviews.
- Once Bug 10 closed: Plan 06-07 attempt-8 progressed to config.sh
  invocation where Bug 11 surfaced.
- Bug 11 discovered 2026-05-06 during Plan 06-07 attempt-8 against the
  same host AFTER Bug 10 fix landed. Affects both BYO bootstrap
  paths (persistent + ephemeral). Cloud path unaffected (cloud-init
  uses a different register_runner construction). Latent regression
  introduced by Plan 06-08's Bug 3 fix; not caught by unit tests
  because they only asserted the new su form's presence, not the
  cwd contract.
- Once Bug 11 closed: Plan 06-07 attempt-9 progressed to service
  install/verify and aborted with the generic ServiceNotActiveError
  message — Bug 12 surfaced because the user-facing copy hid the
  actual remote stderr.
- Bug 12 discovered 2026-05-06 during Plan 06-07 attempt-9 against the
  same host AFTER Bug 11 fix landed. UX-only fix (no behavioral
  change). Required to unblock diagnosis of whatever real failure
  attempt-9 hit at the service step.
- Once Bug 12 closed: Plan 06-07 attempt-10's stderr surface revealed
  Bug 13 — config.sh refused to re-register because `.runner`
  sentinel from attempt-9 still existed in the install dir.
- Bug 13 discovered 2026-05-06 during Plan 06-07 attempt-10 against
  the same host AFTER Bug 12 fix landed. BYO-only (cloud always
  freshly provisioned).
- Once Bug 13 closed: Plan 06-07 attempt-11 progressed to
  install_service and aborted with the now-visible "exists"
  stderr — Bug 14 surfaced.
- Bug 14 discovered 2026-05-06 during Plan 06-07 attempt-11 against
  the same host AFTER Bug 13 fix landed. Same idempotency family at
  the systemd layer.
- Once Bug 14 closed: Plan 06-07 attempt-12 progressed past
  install_service and hit verify_service which had no cd — Bug 15
  surfaced.
- Bug 15 discovered 2026-05-06 during Plan 06-07 attempt-12 against
  the same host AFTER Bug 14 fix landed.
- Once Bug 15 closed: Plan 06-07 attempt-13 cleared bootstrap
  end-to-end and aborted at runner-online polling — Bug 16
  surfaced.
- Bug 16 discovered 2026-05-06 during Plan 06-07 attempt-13 against
  the same host AFTER Bug 15 fix landed. Affects both BYO and cloud
  paths but cloud never reached this stage in attempt-1; smokes have
  never validated the online-check end-to-end before.
- Once Bug 16 closed: Plan 06-07 attempt-14 hit the
  runner-name-conflict pre-check immediately because the runner
  registered by attempt-13 was still on GitHub — Bug 17 surfaced.
- Bug 17 discovered 2026-05-06 during Plan 06-07 attempt-14 against
  the same host AFTER Bug 16 fix landed. Pre-existing latent bug
  in idempotent re-run path; surfaced once attempts started
  reaching online-check successfully.
- Once Bug 17 closed: Plan 06-07 attempt-15 expected to skip the
  conflict, re-register via config.sh --replace, clear online-check
  with case-insensitive label match, and proceed to the smoke
  harness's downstream sequence.
- Related decisions: Phase 2 context (service must not run as root —
  unaffected; this gap is about *bootstrap-time* sudo, not runtime),
  D-04 (live BYO smoke — directly affected), Plan 02-02 (bootstrap pinned
  runner — unaffected).
