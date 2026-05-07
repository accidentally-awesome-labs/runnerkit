---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 11
type: execute
wave: 1
depends_on: [05, 06, 08, 09, 10]
files_modified:
  - internal/cli/status.go
  - internal/cli/status_test.go
  - internal/cli/down.go
  - internal/cli/down_test.go
  - internal/ops/probes.go
  - internal/ops/probes_test.go
  - internal/bootstrap/sudoers.go
  - internal/bootstrap/sudoers_test.go
  - internal/provider/hetzner/destroy.go
  - internal/provider/hetzner/destroy_test.go
autonomous: true
gap_closure: true
requirements: [REL-05]
must_haves:
  truths:
    - "Bug 24: `runnerkit status --repo X` against a freshly-bootstrapped BYO host with an isolated `RUNNERKIT_STATE_DIR` does NOT report `SSH ERROR host key mismatch` when the host has not changed since `runnerkit up` saved its fingerprint. Plan 06-07 attempt-16 (2026-05-06) showed `up` saving the fingerprint successfully, then `status` reconnecting and reporting mismatch despite same host/port/key — runtime drift in the status SSH probe vs the up-time probe."
    - "Bug 25: `runnerkit down --repo X --yes` against a BYO host where `runnerkit status` reports SSH unreachable (legitimately or due to Bug 24) STILL prompts for the sudo password and threads it through `runner_files` cleanup. The current Plan 06-10 Bug 21 fix in `down.go:239` gates on `sshReachable && targetErr == nil && needsAnyRemoteSudo(selected)`. If `sshReachable=false` (false-positive from Bug 24, or genuinely flaky network), the sudo-password prompt is skipped — the subsequent SSH-based cleanup invocations (which DO connect successfully because SSH actually works at the executor level) run sudo without `-S` threading and fail with `sudo: a terminal is required`. The probe + prompt must run independently of `sshReachable`, gated only on `targetErr == nil && needsAnyRemoteSudo(selected)`."
    - "Bug 26: `runnerkit destroy --repo X --yes` (and the cloud-end-to-end smoke destroy trap) deletes Hetzner resources cleanly without `Server must be offline for this action (server_not_stopped)` errors. Plan 06-10 Bug 23 specified detach-firewall + unassign-primary-IPs BEFORE server delete, but `hcloud primary-ip unassign` requires the server to be powered off first — verified live 2026-05-06 against server 129595285. Two acceptable resolutions: (a) power off server before unassign + delete, OR (b) rely on Hetzner's `auto_delete=true` flag on primary IPs so they cascade-delete with the server (verified live 2026-05-06: server delete cascaded primary-IPs cleanly when `auto_delete=true`). Implementation should pick (b) and assert `auto_delete=true` is set at provision time."
    - "Bug 27: scoped sudoers entry at `/etc/sudoers.d/runnerkit-installer` (rendered by `internal/bootstrap/sudoers.go:42`) grants `/opt/runnerkit-runner/svc.sh` but the actual svc.sh path at runtime is `/opt/actions-runner/<RunnerName>/svc.sh` (e.g., `/opt/actions-runner/runnerkit-accidentally-awesome-labs-dat0-local/svc.sh` per `install.go:236`). The scoped entry never matches the real path, so `verify_service` step at `install.go:144` (`cd $InstallPath && sudo ./svc.sh status`) needs password threading at runtime instead of running passwordless under Path C as the scoped entry was designed to support. Fix: change the entry to glob `/opt/actions-runner/runnerkit-*/svc.sh` (sudoers supports `*` wildcards in command paths), OR make the byo-prepare command derive the runner name and write a per-runner allowlist at byo-prepare time — glob is simpler and keeps the one-time prepare promise."
    - "Plan 06-07 attempt-17+ BYO+cloud smoke against `salar@mckee-small-desktop` and a real Hetzner project completes BOTH BYO and cloud smokes end-to-end with: status reporting healthy (no host key mismatch), down completing without `runner_files: failed sudo: a terminal is required`, cloud destroy leaving zero Hetzner orphans (no `server_not_stopped` errors), recording real wall-clock durations + 5 cloud resource IDs + EUR cost into `06-VERIFICATION.md`."
  artifacts:
    - path: "internal/ops/probes.go"
      provides: "Status SSH host-key probe that returns the SAME fingerprint as the one saved by up's probe path (or, if the saved-vs-observed mismatch is real, an explicit reason in the diagnostic, not just a generic mismatch flag). Likely root cause: status uses ssh-keyscan/dial mode that produces different fingerprint encoding than up's path; reconciliation makes them equivalent."
      contains: "host_key_match"
    - path: "internal/cli/down.go"
      provides: "Sudo-password probe + prompt block runs independently of `sshReachable` gating; the gate is reduced to `targetErr == nil && needsAnyRemoteSudo(selected)`. SSH reachability is rechecked at the moment cleanup actually fires; if it succeeds at that point, the threaded password authorizes the rm calls. If SSH genuinely fails, the cleanup surfaces the real SSH error, not a sudo error."
      contains: "needsAnyRemoteSudo"
    - path: "internal/bootstrap/sudoers.go"
      provides: "Scoped sudoers template uses `/opt/actions-runner/runnerkit-*/svc.sh` glob (sudoers `*` wildcard) instead of the literal `/opt/runnerkit-runner/svc.sh` path that never matches. visudo-validates as a real wildcard expansion. Note: sudoers `*` does NOT match `/`, so `runnerkit-*/svc.sh` cannot escape into other dirs — same safety bounds as the original literal."
      contains: "/opt/actions-runner/runnerkit-*/svc.sh"
    - path: "internal/provider/hetzner/destroy.go"
      provides: "Destroy flow asserts `auto_delete=true` was recorded for primary IPs at provision time; relies on the cascade-delete behavior so `Server must be offline` is never hit. If a saved-state primary-IP shows `auto_delete=false` (legacy state from a prior runnerkit version), destroy logs a warning and falls back to power-off + unassign + delete. Firewall detach-from-server still runs first (no power-off needed for firewall detach)."
      contains: "AutoDelete"
    - path: "internal/provider/hetzner/provision.go"
      provides: "Provision sets `auto_delete=true` on both primary IPv4 and primary IPv6 at create time — required for the destroy.go cascade-delete approach above. (May already be set; this plan's task is to verify + add a regression test.)"
      contains: "AutoDelete: true"
  key_links:
    - from: "Plan 06-07 attempt-17+ smoke against fresh BYO + Hetzner project"
      to: "smoke-green resume signal → v1.0.0 tag push"
      via: "Bugs 24+25 close the BYO post-up surface; Bug 26 closes the cloud destroy ordering surface; Bug 27 closes the Path C scoped-sudoers passwordless-svc.sh promise; status/down/destroy all green end-to-end without operator intervention"
      pattern: "smoke-green"

tasks:
  - id: bug-24-status-host-key
    name: "status: SSH host-key probe matches up-time fingerprint (Bug 24)"
    autonomous: true
  - id: bug-25-down-sudo-gate
    name: "down: probe sudo + prompt independently of sshReachable (Bug 25)"
    autonomous: true
  - id: bug-26-destroy-cascade
    name: "cloud destroy: rely on auto_delete cascade for primary IPs (Bug 26)"
    autonomous: true
  - id: bug-27-sudoers-glob
    name: "sudoers: glob svc.sh path so Path C actually grants NOPASSWD on the real path (Bug 27)"
    autonomous: true
---

# Plan 06-11: Status + Down + Cloud Destroy + Sudoers Path Fixes

## Context

Plan 06-07 attempt-16 (2026-05-06) re-ran `make smoke-live` against
`salar@mckee-small-desktop` and a real Hetzner `dat0` project after
Plan 06-10 (Bugs 19-23) landed. BYO bootstrap completed in 107s and
registered runner ID 25 successfully. Cloud-up provisioned 4 Hetzner
resources cleanly. The harness then surfaced four NEW bugs that block
the v1.0.0 `smoke-green` resume signal:

1. **Bug 24** — `runnerkit status` reports `SSH ERROR host key mismatch`
   despite the host being unchanged since `runnerkit up` saved the
   fingerprint. The status SSH probe path produces a different
   fingerprint encoding/format than the up-time probe path. False
   positive cascades into `Service WARNING skipped because SSH host
   key mismatch`.

2. **Bug 25** — `runnerkit down --yes` against the same host fails
   `runner_files` cleanup with
   `sudo: a terminal is required to read the password`. Plan 06-10
   Bug 21 fix added a sudo-password probe + Path B prompt to down,
   but gated it on `sshReachable && targetErr == nil` (down.go:239).
   `sshReachable` was set false by the preceding `collectStatus` call
   because of Bug 24, so the prompt block was skipped entirely.
   The subsequent cleanup commands DO connect over SSH (the
   reachability flag is a higher-level health summary, not the actual
   executor state), invoke sudo without `-S` threading, and fail.

3. **Bug 26** — Cloud destroy ordering still produces orphans. Plan
   06-10 Bug 23 specified `detach firewall → unassign primary IPs →
   delete server → delete primary IPs → delete firewall`. Live test
   2026-05-06 (manual hcloud CLI cleanup of Plan 06-07 attempt-16
   orphans): `hcloud primary-ip unassign` fails with
   `Server must be offline for this action (server_not_stopped)`.
   Hetzner requires the server to be powered off OR primary IPs
   to have `auto_delete=true` for cascade-delete on server delete.
   The latter is simpler and was empirically verified working at
   cleanup time (server delete with assigned IPs that had
   `auto_delete=true` cascaded cleanly).

4. **Bug 27** — scoped sudoers entry from `runnerkit byo-prepare`
   grants NOPASSWD for `/opt/runnerkit-runner/svc.sh`, but the
   ACTUAL svc.sh path at runtime is
   `/opt/actions-runner/<RunnerName>/svc.sh` (e.g., the dat0 host
   ended up with
   `/opt/actions-runner/runnerkit-accidentally-awesome-labs-dat0-local/svc.sh`).
   The scoped entry never matches; `verify_service` step in
   bootstrap.Apply (install.go:144) needs the Path B password
   threading at runtime even on a Path C-prepared host. Either fix
   the path mismatch in the sudoers template (sudoers supports `*`
   wildcards in command paths, and `*` does not match `/`, so
   `runnerkit-*/svc.sh` is safe) or re-render the entry per-runner
   at byo-prepare time. Glob is simpler.

## Bug Summary

| Bug | Description | Surface | Detected |
|-----|-------------|---------|----------|
| 24 | status SSH host-key probe returns mismatch despite unchanged host | `runnerkit status` | Plan 06-07 attempt-16 BYO smoke 2026-05-06 |
| 25 | down's Plan 06-10 sudo-prompt fix gated on `sshReachable`; skipped when status falsely flags SSH down | `runnerkit down` cleanup | Same — cascaded from Bug 24 |
| 26 | cloud destroy `primary-ip unassign` requires server offline; Plan 06-10 ordering missed this | `runnerkit destroy --cloud hetzner` | Manual hcloud cleanup of attempt-16 orphans 2026-05-06 |
| 27 | scoped sudoers grants `/opt/runnerkit-runner/svc.sh` but real path is `/opt/actions-runner/runnerkit-*/svc.sh` | `runnerkit byo-prepare` → bootstrap.Apply verify_service | Plan 06-07 attempt-16 BYO smoke 2026-05-06 |

## Approach

- **Bug 24:** trace where `runnerkit status` collects the current
  fingerprint and compare against the up-time path
  (`internal/bootstrap` or `internal/remote`). Most likely candidates:
  (a) status uses `ssh-keyscan` while up uses dial-time `host-key
  callback`, producing different encodings (Base64 SHA256 vs MD5
  hex); (b) status falls through to a different default port
  (mckee-small-desktop:22 vs whatever was saved). Reconcile so both
  paths produce byte-equal fingerprints. Add a unit test fixture
  that asserts equality.

- **Bug 25:** drop `sshReachable` from the gate at `down.go:239`.
  The probe + prompt can run regardless — if SSH is genuinely down,
  the probe times out (5s) and we fall through; if SSH is up
  (which is what we observe in practice), the password is captured
  and threaded. Risk: probe wastes 5s when SSH genuinely down. Cost
  acceptable for cleanup robustness.

- **Bug 26:** verify provision.go sets `AutoDelete: true` on both
  primary IPs at create time. Update destroy.go to skip the
  `unassign` step entirely when state shows `auto_delete=true` on
  the saved IPs; rely on server delete to cascade. Keep firewall
  detach (no power-off requirement). Add unit test asserting destroy
  order: detach firewall → delete server → delete firewall (no
  unassign step in the auto_delete path).

- **Bug 27:** change sudoers template line `/opt/runnerkit-runner/svc.sh`
  to `/opt/actions-runner/runnerkit-*/svc.sh`. Update
  RenderSudoersEntry's commentary block and update sudoers_test.go
  fixture. Add a regression test asserting the rendered content
  contains the glob, not the legacy literal. After Plan 06-11 lands,
  the maintainer must re-run `runnerkit byo-prepare --host
  $RUNNERKIT_SMOKE_BYO_HOST` once to refresh the entry on the smoke
  host before attempt-17.

## Out of Scope

- Plan 06-10 work (Bugs 19-23) — already complete.
- Maintainer human-action checkpoint (Plan 06-07) — closes after
  Plan 06-11 lands and full smoke passes attempt-17.
- New error codes for Bugs 24/25/26/27 — existing RKD-CLEAN-003 +
  RKD-PROV-006 + RKD-BOOT-015 cover these surfaces.
- Re-architecting the status/down split or the byo-prepare flow —
  the four bugs are surgical fixes within existing structure.

## Verification

- Full repo `go test ./... -count=1 -race` passes.
- New unit tests:
  - `probes_test.go::TestStatusHostKeyMatchesUpTimeFingerprint`
  - `down_test.go::TestDown_SudoProbeRunsEvenWhenSSHReachableFalse`
  - `destroy_test.go::TestDestroy_AutoDeleteCascadeNoUnassign`
  - `sudoers_test.go::TestRenderSudoersEntryUsesSvcShGlob`
- Plan 06-07 attempt-17+ BYO+cloud smoke completes without warnings;
  cloud destroy leaves zero orphans (verify with hcloud server list
  + firewall list + primary-ip list returning empty).
- 06-VERIFICATION.md baseline filled with real numbers + maintainer
  sign-off → `smoke-green` resume signal → v1.0.0 tag push.

## Pre-Smoke Maintainer Action (post-landing)

After Plan 06-11 lands, before re-running `make smoke-live`:

```bash
# Refresh the scoped sudoers entry on the smoke host with the new glob
go run ./cmd/runnerkit byo-prepare --host $RUNNERKIT_SMOKE_BYO_HOST
# Type sudo password once. The new /etc/sudoers.d/runnerkit-installer
# will use the /opt/actions-runner/runnerkit-*/svc.sh glob.

# Verify Hetzner project empty
hcloud server list ; hcloud firewall list ; hcloud primary-ip list
```

Then re-run `make smoke-live` per Plan 06-07 sequence.
