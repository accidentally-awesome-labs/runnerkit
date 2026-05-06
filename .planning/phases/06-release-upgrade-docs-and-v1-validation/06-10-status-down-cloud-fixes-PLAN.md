---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 10
type: execute
wave: 1
depends_on: [05, 06, 08, 09]
files_modified:
  - internal/cli/status.go
  - internal/cli/status_test.go
  - internal/cli/down.go
  - internal/cli/down_test.go
  - internal/ops/doctor.go
  - internal/ops/doctor_test.go
  - internal/provider/hetzner/provision.go
  - internal/provider/hetzner/provision_test.go
  - internal/provider/hetzner/destroy.go
  - internal/provider/hetzner/destroy_test.go
autonomous: true
gap_closure: true
requirements: [REL-05]
must_haves:
  truths:
    - "`runnerkit status --repo X` against a healthy BYO runner reports `Service OK active` (not WARNING inactive). Bug 19: status looks up the actual systemd unit name (which includes the GitHub-mangled repo segment, e.g. `actions.runner.<owner-repo>.<runner-name>.service`), not the simplified `actions.runner.<runner-name>.service` it currently constructs."
    - "`runnerkit status` and `runnerkit doctor` label-drift detection treat `Linux`/`linux` and `X64`/`x64` as equal (Bug 20). The drift report does not warn `missing [linux, x64] extra [Linux, X64]` when the only difference is case. Same family as Plan 06-09 Bug 16 but in different code paths (status / doctor vs up online-check)."
    - "`runnerkit down --repo X --yes` against a BYO host with password-protected sudo + a Path B-style flow (or with byo-prepared scoped sudoers) does NOT fail at the `runner_files` cleanup step with `sudo: a terminal is required`. Bug 21: down's remote cleanup must use the same SudoPassword threading that bootstrap does (or a equivalent path that doesn't require a tty)."
    - "Cloud `runnerkit up --cloud hetzner --mode persistent` against a freshly-provisioned Hetzner server does NOT fail at SSH-readiness with `SSH host key fingerprint was not observed`. Bug 22: the host-key probe path needs sufficient retry / backoff / observability to reliably capture the host key during the cloud-init window."
    - "Cloud `runnerkit destroy --repo X --yes` (or the cloud destroy that fires on cleanup) deletes resources in the correct order so firewall + primary IPv4 + primary IPv6 are detached from the server BEFORE the server destroy completes — preventing the `resource_in_use` / `must_be_unassigned` orphans observed in Plan 06-07 attempt-15. Bug 23: ordering + waits."
    - "Plan 06-07 attempt-16 (or subsequent) against `salar@mckee-small-desktop` and a real Hetzner project completes BOTH BYO and cloud smokes end-to-end with no warnings, partial cleanups, or orphaned billable resources, recording real wall-clock durations + 5 cloud resource IDs + EUR cost into `06-VERIFICATION.md`."
  artifacts:
    - path: "internal/cli/status.go"
      provides: "Status command resolves the actual systemd unit name from runner registration (.runner JSON contains agentName + serverUrl which can derive the unit pattern), or queries `systemctl list-units` and matches by `*runnerkit*<runner-name>*` pattern."
      contains: "actions.runner."
    - path: "internal/ops/doctor.go"
      provides: "doctor's label-drift finding does case-insensitive comparison (matches Plan 06-09 Bug 16's runnerOnlineWithLabels approach)."
      contains: "strings.ToLower"
    - path: "internal/cli/down.go"
      provides: "down's remote cleanup wraps sudo invocations with the existing wrapSudoCommand-style password threading when SudoPassword is provided, OR explicitly uses byo-prepare's scoped sudoers (preferred) for the rm/uninstall commands."
      contains: "wrapSudo"
    - path: "internal/provider/hetzner/provision.go"
      provides: "Cloud SSH host-key readiness probe with retry budget + backoff + diagnostic on failure."
      contains: "host_key"
    - path: "internal/provider/hetzner/destroy.go"
      provides: "Destroy ordering: detach firewall + primary IPv4 + primary IPv6 from server BEFORE server delete; wait for detachment confirmation; delete primary IPs after server gone; firewall last (free, no ordering risk)."
      contains: "DetachFirewall"
  key_links:
    - from: "Plan 06-07 attempt-16+ BYO+cloud smoke"
      to: "smoke-green resume signal → v1.0.0 tag push"
      via: "All 5 in-tree fixes (Bugs 19-23) clear the post-up + cloud surface; 06-VERIFICATION.md baseline filled with real numbers + maintainer sign-off"
      pattern: "smoke-green"

tasks:
  - id: bug-19-service-name
    name: "status: resolve actual systemd unit name (Bug 19)"
    autonomous: true
  - id: bug-20-label-case
    name: "status + doctor: case-insensitive label drift (Bug 20)"
    autonomous: true
  - id: bug-21-down-sudo
    name: "down: thread sudo password through remote cleanup (Bug 21)"
    autonomous: true
  - id: bug-22-cloud-ssh-ready
    name: "cloud up: robust SSH host-key probe (Bug 22)"
    autonomous: true
  - id: bug-23-destroy-ordering
    name: "cloud destroy: detach firewall + IPs before server delete (Bug 23)"
    autonomous: true
---

# Plan 06-10: Status + Doctor + Down + Cloud Post-Up Fixes

## Context

Plan 06-09 closed the BYO bootstrap path end-to-end (Bugs 4-18,
2026-05-05/06). Plan 06-07 attempt-15 then validated BYO bootstrap
(`BYO_DURATION_SECONDS=125`, runner id 24 online) but surfaced 5
NEW bugs in the post-up + cloud surface:

1. `runnerkit status` reports `Service WARNING inactive` even though
   the actual systemd unit IS in `active running` state. The status
   command constructs the unit name as
   `actions.runner.<runner-name>.service` but GitHub's runner naming
   convention puts the repo segment first:
   `actions.runner.<owner-repo>.<runner-name>.service`.

2. `runnerkit status` and `runnerkit doctor` both report a label
   drift `missing [linux, x64] extra [Linux, X64]` even though the
   labels are semantically identical. Same root cause as Plan 06-09
   Bug 16 (case-sensitive set membership against GitHub's CamelCase
   auto-labels) but in a different code path that wasn't touched by
   Bug 16's fix.

3. `runnerkit down --repo X --yes` against the same host fails at
   the `runner_files` cleanup step with
   `sudo: a terminal is required to read the password`. Down's
   remote cleanup invokes sudo without threading the password the
   way bootstrap does (Plan 06-09 Bug 10's wrapSudoCommand approach).

4. Cloud `runnerkit up --cloud hetzner` provisions a Hetzner server
   successfully (`provider_server: done`) but immediately fails the
   SSH-readiness check with
   `SSH host key fingerprint was not observed`. cloud-init takes
   ~30-90s to install ssh-host-keys on a fresh Ubuntu image; the
   current probe doesn't tolerate that.

5. Cloud `runnerkit destroy` (fired on cleanup trap when up failed)
   deletes the server but cannot delete the firewall or primary
   IPs because they're still attached:
   - firewall 10939017: `resource_in_use`
   - primary-ipv4 4a48c7f: `must_be_unassigned`
   - primary-ipv6 9b11cdf: `must_be_unassigned`
   - primary-ipv6 60b2566: `must_be_unassigned`

## Bug Summary

| Bug | Description | Surface |
|-----|-------------|---------|
| 19 | `runnerkit status` constructs simplified systemd unit name; misses GitHub's `<owner-repo>.<runner-name>` form | status |
| 20 | `runnerkit status` + `runnerkit doctor` label-drift detector is case-sensitive (same family as Plan 06-09 Bug 16) | status + doctor |
| 21 | `runnerkit down` remote cleanup uses sudo without password threading (same family as Plan 06-09 Bug 10) | down |
| 22 | Cloud SSH host-key readiness probe lacks retry/backoff for cloud-init's host-key install window | provider/hetzner provision |
| 23 | Cloud destroy ordering: server deleted before firewall + primary IPs detached → orphaned billable resources | provider/hetzner destroy |

## Approach

- **Bug 19**: read the actual unit name from `.runner` registration
  metadata (agentName + repo from serverUrl) OR query
  `systemctl list-units 'actions.runner.*' --no-pager` and match
  by `runnerkit-<runner-name>` substring.
- **Bug 20**: lowercase both sides in label-drift comparison.
  Possibly extract a shared helper between Bug 16's
  `runnerOnlineWithLabels` and the status/doctor drift check.
- **Bug 21**: thread SudoPassword through down's remote cleanup
  the same way Plan 06-09 Bug 10's wrapSudoCommand does. Or rely
  on byo-prepare's scoped sudoers if rm/uninstall commands are
  in the allowlist.
- **Bug 22**: increase SSH host-key probe retry budget; surface
  cloud-init progress in stderr; fall back to passive
  `ssh-keyscan` after N seconds.
- **Bug 23**: pre-destroy detachment sequence:
  1. Detach firewall from server
  2. Detach primary IPs from server
  3. Wait for confirmations
  4. Delete server
  5. Delete primary IPs
  6. Delete firewall
  Add idempotency for partial-cleanup retries.

## Out of Scope

- Plan 06-09 work (Bugs 4-18) — already complete.
- Maintainer human-action checkpoint (Plan 06-07) — closes after
  Plan 06-10 lands and full smoke passes.
- Token-permission verification (existing Plan 02 scope).

## Verification

- Full repo `go test ./... -count=1 -race` passes.
- Plan 06-07 attempt-16+ BYO+cloud smoke completes without warnings;
  cloud destroy leaves zero orphaned billable resources.
- 06-VERIFICATION.md baseline filled with real numbers + maintainer
  sign-off → `smoke-green` resume signal → v1.0.0 tag push.
