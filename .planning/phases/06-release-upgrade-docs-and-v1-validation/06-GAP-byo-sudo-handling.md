---
status: open
source: live BYO smoke (Plan 06-04 Task 4)
discovered: 2026-05-04
phase: 06-release-upgrade-docs-and-v1-validation
gap_closure_target: 06-05 (or v1.1 milestone)
severity: medium
type: bug + missing-feature
related_decisions: [D-04 (live BYO smoke), Phase 2 context (service must not run as root), 02-01 (preflight separate behind remote.Executor)]
---

# Gap: BYO host sudo handling

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

## Cross-references

- Discovered while running Plan 06-04 Task 4 BYO smoke. The smoke
  workaround for v1.0.0: maintainer manually adds NOPASSWD on the test
  host. This is the documented v1.0.0 contract until this gap closes.
- Once closed: the Plan 06-04 BYO smoke can be re-run end-to-end without
  any host-side preconfiguration (Path B will prompt automatically), and
  the v1.0.0 preflight bug stops biting first-time BYO users.
- Related decisions: Phase 2 context (service must not run as root —
  unaffected; this gap is about *bootstrap-time* sudo, not runtime),
  D-04 (live BYO smoke — directly affected), Plan 02-02 (bootstrap pinned
  runner — unaffected).
