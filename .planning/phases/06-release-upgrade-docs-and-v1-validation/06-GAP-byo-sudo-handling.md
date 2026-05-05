---
status: open
source: live BYO smoke (Plan 06-04 Task 4); re-smoke (Plan 06-07 attempt 1)
discovered: 2026-05-04
updated: 2026-05-05
phase: 06-release-upgrade-docs-and-v1-validation
gap_closure_target: 06-05 (Bug 1+2 + Tasks A,E — CLOSED 2026-05-04 commit ee5c0a2); 06-06 (Tasks B,C,D — CLOSED 2026-05-05 commit 08b8708); 06-08 (Bug 3 + Task F — OPEN, mandatory before v1.0.0 tag)
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

## Cross-references

- Discovered while running Plan 06-04 Task 4 BYO smoke. The smoke
  workaround for v1.0.0: maintainer manually adds NOPASSWD on the test
  host. This is the documented v1.0.0 contract until this gap closes.
- Bug 3 discovered 2026-05-05 during Plan 06-07 attempt 1 against the
  same `salar@mckee-small-desktop` host AFTER Plans 06-05 + 06-06 landed.
  Cloud path unaffected because Hetzner cloud-init configures `(ALL) NOPASSWD: ALL`
  which covers `runas=runnerkit-runner`. BYO non-functional in v1 until
  Task F lands.
- Once closed: the Plan 06-07 BYO + Hetzner smoke can re-run end-to-end
  without any host-side preconfiguration AND `register_runner` no longer
  requires `(ALL)` runas in host sudoers. Plan 06-08 is the gap-closure
  target.
- Related decisions: Phase 2 context (service must not run as root —
  unaffected; this gap is about *bootstrap-time* sudo, not runtime),
  D-04 (live BYO smoke — directly affected), Plan 02-02 (bootstrap pinned
  runner — unaffected).
