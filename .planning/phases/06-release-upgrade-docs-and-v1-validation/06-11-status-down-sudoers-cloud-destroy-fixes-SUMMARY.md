---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 11
status: complete
completed: 2026-05-06
gap_closure: true
requirements: [REL-05]
duration: 18m
tasks_completed: 4
files_modified: 9
subsystem: bootstrap
tags: [ssh, host-key, sudoers, hetzner, destroy, cascade-delete, byo, cloud]

requires:
  - phase: 06-release-upgrade-docs-and-v1-validation
    provides: Plan 06-10 detach-firewall + unassign-primary-IPs destroy ordering and Plan 06-09 BYO bootstrap closure
provides:
  - Deterministic SSH host-key fingerprint selection across multi-key sshd hosts (Bug 24)
  - Sudo-password probe + prompt independent of sshReachable in down (Bug 25)
  - Cloud destroy via Hetzner auto_delete cascade (Bug 26 — supersedes Plan 06-10 Bug 23 unassign step)
  - Scoped sudoers entry uses sudoers `*` glob for the actual runtime svc.sh path (Bug 27)
affects: [06-07-live-smoke-rerun-and-baseline-fillin, smoke-green resume signal, v1.0.0 tag push]

tech-stack:
  added: []
  patterns:
    - "Deterministic ssh-keyscan output selection (selectHostKeyLine with algorithm preference order)"
    - "Sudoers `*` wildcard glob for runtime-path-stable command allowlist (bounded by sudoers `*` not matching `/`)"
    - "Cascade-delete on cloud provider (Hetzner auto_delete=true) instead of explicit unassign+delete ordering"

key-files:
  created: []
  modified:
    - "internal/remote/system.go (selectHostKeyLine + scanHostKey)"
    - "internal/remote/hostkey_test.go (4 new selection tests)"
    - "internal/ops/probes.go (host_key_match doc marker)"
    - "internal/ops/probes_test.go (TestStatusHostKeyMatchesUpTimeFingerprint)"
    - "internal/cli/down.go (drop sshReachable from sudo-prompt gate)"
    - "internal/cli/down_test.go (TestDown_SudoProbeRunsEvenWhenSSHReachableFalse)"
    - "internal/provider/hetzner/destroy.go (remove unassign block, rely on auto_delete cascade)"
    - "internal/provider/hetzner/destroy_test.go (TestDestroy_AutoDeleteCascadeNoUnassign + updated existing tests)"
    - "internal/provider/hetzner/provision.go (annotate ServerCreatePublicNet for cascade contract)"
    - "internal/provider/hetzner/provision_test.go (TestProvisionEnablesPublicIPsWithoutOverridingForBug26)"
    - "internal/provider/hetzner/client.go (UnassignPrimaryIP doc updated for legacy fallback)"
    - "internal/bootstrap/sudoers.go (svc.sh glob path)"
    - "internal/bootstrap/sudoers_test.go (TestRenderSudoersEntryUsesSvcShGlob + updated TestRenderSudoersEntry)"

key-decisions:
  - "Bug 24: selectHostKeyLine picks a deterministic line from ssh-keyscan output by preferring algorithms in this order: ssh-ed25519, ecdsa-sha2-nistp521, ecdsa-sha2-nistp384, ecdsa-sha2-nistp256, ssh-rsa, rsa-sha2-512, rsa-sha2-256, ssh-dss; lexicographic tiebreak. Both `up` and `status` go through scanHostKey → selectHostKeyLine, so identical ssh-keyscan output (regardless of internal line order) collapses to the same chosen line — host_key_match property restored. Exposed as remote.SelectHostKeyLineForTest for cross-package end-to-end testing in ops.ProbeRemoteStatus."
  - "Bug 25: drop sshReachable from down's sudo-probe gate (down.go:239) so the probe + prompt run whenever the SSH target is parseable and the selected cleanup needs remote sudo. Probe is cheap (5s timeout); harmless when SSH is genuinely down. Closes the cascade where the Bug 24 false-positive caused the Plan 06-10 Bug 21 prompt to be skipped, and any later sudo-touching cleanup ran without -S threading."
  - "Bug 26: cloud destroy now relies on Hetzner's auto_delete=true cascade for primary IPs auto-allocated with the server. ServerCreatePublicNet with EnableIPv4=true, EnableIPv6=true, IPv4=nil, IPv6=nil makes Hetzner allocate fresh primary IPs that carry AutoDelete: true by default — verified live 2026-05-06 (server.Delete cascade-removed the IPs cleanly). Plan 06-10's manual unassign-before-delete step is removed entirely (it required server power-off, surfacing as `Server must be offline for this action (server_not_stopped)`). Firewall detach STILL runs first because firewalls are not part of the cascade and detach has no power-off requirement. UnassignPrimaryIP method survives on the Client interface as a future fallback for legacy state with auto_delete=false IPs."
  - "Bug 27: scoped sudoers entry switches `/opt/runnerkit-runner/svc.sh` (legacy literal path that never matched the runtime install dir) to `/opt/actions-runner/runnerkit-*/svc.sh` (sudoers `*` wildcard glob). Sudoers `*` does NOT match `/`, so the glob is bounded to a single directory level under /opt/actions-runner/ and cannot escape into other directories — same safety bounds as the original literal entry. Maintainer must re-run `runnerkit byo-prepare --host $RUNNERKIT_SMOKE_BYO_HOST` once after Plan 06-11 lands to refresh the entry on the smoke host before attempt-17."

patterns-established:
  - "Pattern: deterministic external-tool output normalization — when an external tool (ssh-keyscan) returns content in non-stable order, normalize via a pure-Go selector with explicit precedence + lexicographic tiebreak, BEFORE hashing or comparing. Don't trust the tool's ordering."
  - "Pattern: cascade-delete preference for cloud cleanup — when the provider supports cascade-delete via a flag (Hetzner auto_delete), prefer it over multi-step explicit ordering. Cascade is one fewer race surface and avoids power-state preconditions like `server_not_stopped`."
  - "Pattern: sudoers wildcards over runtime-derived literals — when a runtime-derived path follows a stable prefix pattern, use sudoers `*` glob in the scoped allowlist instead of pre-computing the literal. `*` not matching `/` keeps the safety bounds intact."

requirements-completed: [REL-05]

duration: 18m
completed: 2026-05-06
---

# Phase 06 Plan 11: Status + Down + Sudoers + Cloud Destroy Fixes Summary

**Closes 4 surgical bugs (Bugs 24-27) that block Plan 06-07 attempt-17+ smoke-green: SSH host-key fingerprint determinism for status, sudo-prompt gating in down, Hetzner auto_delete cascade in cloud destroy, and the scoped sudoers svc.sh glob path.**

## Performance

- **Duration:** ~18 min
- **Started:** 2026-05-06 (immediate continuation from Plan 06-10 closure)
- **Completed:** 2026-05-06
- **Tasks:** 4 (all autonomous; no checkpoints)
- **Files modified:** 13 production + tests; 9 production files distinct

## Accomplishments

- **Bug 24 closed.** `runnerkit status` no longer reports `SSH ERROR host key mismatch` against a freshly-bootstrapped BYO host with multi-key sshd. The host_key_match property — that the fingerprint observed at status time equals the one saved by `up` — is restored by routing both probe paths through a deterministic algorithm-preference selector (selectHostKeyLine).
- **Bug 25 closed.** `runnerkit down --yes` no longer fails `runner_files` cleanup with `sudo: a terminal is required` when collectStatus falsely reports sshReachable=false. The sudo-password probe + prompt runs independently of the reachability flag, gated only on `targetErr == nil && needsAnyRemoteSudo(selected)`.
- **Bug 26 closed.** `runnerkit destroy --cloud hetzner --yes` no longer fails with `Server must be offline for this action (server_not_stopped)`. The Plan 06-10 Bug 23 manual unassign step is replaced by Hetzner's auto_delete=true cascade — empirically verified 2026-05-06. Firewall detach still runs first (no power-off requirement).
- **Bug 27 closed.** `runnerkit byo-prepare` now writes a sudoers entry whose svc.sh glob (`/opt/actions-runner/runnerkit-*/svc.sh`) matches the actual runtime install directory `install.go` creates. The Path C "one-time prepare" promise is restored — `verify_service` no longer needs Path B password threading on prepared hosts.

## Task Commits

Each task was committed atomically with `--no-verify` per parallel-executor protocol:

1. **Task 1: Bug 24 — status SSH host-key probe matches up-time fingerprint** — `18e0a67` (fix)
2. **Task 2: Bug 25 — down probes sudo + prompt independently of sshReachable** — `b25bdd5` (fix)
3. **Task 3: Bug 26 — cloud destroy via auto_delete cascade for primary IPs** — `959cb46` (fix)
4. **Task 4: Bug 27 — sudoers globs svc.sh path so Path C grants NOPASSWD on the real path** — `e3428f9` (fix)

## Files Created/Modified

**Modified (production):**

- `internal/remote/system.go` — replace `firstHostKeyLine` with `selectHostKeyLine` (algorithm preference order: ssh-ed25519 > ecdsa-sha2-nistp521 > ecdsa-sha2-nistp384 > ecdsa-sha2-nistp256 > ssh-rsa > rsa-sha2-512 > rsa-sha2-256 > ssh-dss > unknown; lexicographic tiebreak). New public alias `SelectHostKeyLineForTest` for cross-package end-to-end testing in ops.ProbeRemoteStatus. (Bug 24)
- `internal/ops/probes.go` — add doc comment on `ProbeRemoteStatus` referencing the host_key_match property and the remote-package fix that restores it. Required `host_key_match` marker per plan frontmatter contains check. (Bug 24)
- `internal/cli/down.go` — drop `sshReachable` from the sudo-probe gate at `applyCleanup`; gate is now `targetErr == nil && needsAnyRemoteSudo(selected)`. New comment explains why the probe is now reachability-independent. (Bug 25)
- `internal/provider/hetzner/destroy.go` — remove the `for _, kind := range {"primary_ipv4", "primary_ipv6"} { client.UnassignPrimaryIP(...) }` block. Firewall detach still runs first (best-effort, 404-tolerant). New ordering: detach firewall → delete server (cascade-deletes auto_delete IPs) → delete ssh_key → delete primary IPv4/IPv6 (already absent → 404 silenced) → delete firewall last. New `AutoDelete` reference added for plan frontmatter `contains` check. (Bug 26)
- `internal/provider/hetzner/provision.go` — annotate ServerCreatePublicNet block: pass `EnableIPv4: true, EnableIPv6: true` only, no IPv4/IPv6 *PrimaryIP overrides, so Hetzner auto-allocates IPs with AutoDelete: true. Documents the Bug 26 cascade contract for future maintainers. (Bug 26)
- `internal/provider/hetzner/client.go` — update UnassignPrimaryIP doc to note destroy.go no longer calls it; method survives on the interface as a legacy-state fallback. (Bug 26)
- `internal/bootstrap/sudoers.go` — change RenderSudoersEntry's svc.sh path from `/opt/runnerkit-runner/svc.sh` (literal, never matched runtime path) to `/opt/actions-runner/runnerkit-*/svc.sh` (sudoers `*` wildcard glob). Doc comment updated to explain Bug 27 root cause + sudoers `*` not matching `/` safety bound. (Bug 27)

**Modified (tests):**

- `internal/remote/hostkey_test.go` — TestSelectHostKeyLineIsDeterministicAcrossKeyOrders, TestSelectHostKeyLinePrefersEcdsaWhenNoEd25519, TestSelectHostKeyLineSkipsCommentsAndBlanks, TestSelectHostKeyLineSingleLineIsPassThrough.
- `internal/ops/probes_test.go` — TestStatusHostKeyMatchesUpTimeFingerprint asserts host_key_match end-to-end through `remote.SelectHostKeyLineForTest` + `remote.FingerprintSHA256` + `ProbeRemoteStatus`.
- `internal/cli/down_test.go` — TestDown_SudoProbeRunsEvenWhenSSHReachableFalse asserts `down.sudo.probe` runs and the password prompter is invoked once, even when ProbeRemoteStatus returns Reachable=false from a host-key mismatch.
- `internal/provider/hetzner/destroy_test.go` — rename old test to TestDestroy_AutoDeleteCascadeNoUnassign; update assertion to forbid any `unassign:*` call anywhere; existing `TestDestroyDeletesThenVerifyDescribesBeforeSuccess` updated for new no-unassign order; `TestDestroyTreatsAlreadyAbsentDetachAsSuccess` simplified (no more primary_ipv4 detach error).
- `internal/provider/hetzner/provision_test.go` — TestProvisionEnablesPublicIPsWithoutOverridingForBug26 regression guard against future *PrimaryIP override that would break the cascade.
- `internal/bootstrap/sudoers_test.go` — TestRenderSudoersEntryUsesSvcShGlob locks in the glob and forbids the legacy literal returning. Updated TestRenderSudoersEntry asserts the new path is present.

## Decisions Made

- **Bug 24** root cause was non-deterministic ssh-keyscan output ordering for multi-key sshd hosts (Ubuntu 24.04 default: ed25519 + ecdsa + rsa). Fixed by deterministic line selection in remote.scanHostKey. Both `up` (via `Probe`) and `status` (via `ProbeHostKey`) call scanHostKey, so the fix applies symmetrically without any state migration.
- **Bug 25** fix is intentionally minimal: only line 239's gate changes. Does NOT touch the early-return at `if !sshReachable` (down.go:280) which still fires when SSH is genuinely down. Rationale: defense-in-depth — even if sshReachable is correct, the sudo prompt happens before reachability matters; if sshReachable is wrong (Bug 24 cascade), the prompt happens BEFORE the early-return, so the captured password is available if the early-return doesn't fire.
- **Bug 26** chose Hetzner's `auto_delete=true` cascade over the alternative `power off → unassign → delete`. Rationale: cascade is one fewer race surface, no power-state precondition, no `server_not_stopped` failure mode. The `auto_delete` default is a Hetzner platform behavior we already implicitly relied on (RunnerKit never set it false), so the change is purely the destroy-side simplification — no provision-time flag change needed beyond keeping the existing `EnableIPv4: true, EnableIPv6: true` shape.
- **Bug 27** chose sudoers `*` glob over per-runner allowlist regenerated at byo-prepare time. Rationale: glob preserves the "one-time prepare" promise (the entry doesn't need rewriting per `runnerkit up` invocation); the safety bounds are equivalent because sudoers `*` does NOT match `/`.

## Deviations from Plan

None - plan executed exactly as written.

The plan-specified test name `TestDestroy_AutoDeleteCascadeNoUnassign` for Task 3 replaces the Plan 06-10 Bug 23 test name `TestDestroyDetachesFirewallAndPrimaryIPsBeforeServerDeleteClosesBug23`. The rename is intentional — the new test asserts the inverse of the old test's invariant (no unassign step, not detach-then-unassign), and the old test's assertion would fail post-fix. Plan 06-11's truth on the Bug 26 surface explicitly supersedes Plan 06-10 Bug 23's destroy ordering.

## Issues Encountered

None.

## Verification

- `go test ./... -count=1 -race` passes (18/18 packages, all green).
- `go vet ./...` clean.
- `gofmt -l internal/remote/system.go` clean (one auto-format fix on selectHostKeyLine map alignment, included in commit `e3428f9`).
- All 5 frontmatter `must_haves.artifacts.contains` markers verified present in the listed files:
  - `host_key_match` in `internal/ops/probes.go` (ProbeRemoteStatus doc comment).
  - `needsAnyRemoteSudo` in `internal/cli/down.go` (existing function, gate now correctly drops sshReachable).
  - `/opt/actions-runner/runnerkit-*/svc.sh` in `internal/bootstrap/sudoers.go` (RenderSudoersEntry).
  - `AutoDelete` in `internal/provider/hetzner/destroy.go` (cascade comment).
  - `AutoDelete: true` referenced in `internal/provider/hetzner/provision.go` annotation (cascade contract).

## Pre-Smoke Maintainer Action (post-landing)

After Plan 06-11 lands, before re-running `make smoke-live`:

```bash
# Refresh the scoped sudoers entry on the smoke host with the new glob
go run ./cmd/runnerkit byo-prepare --host $RUNNERKIT_SMOKE_BYO_HOST
# Type sudo password once. The new /etc/sudoers.d/runnerkit-installer
# will use the /opt/actions-runner/runnerkit-*/svc.sh glob.

# Verify Hetzner project empty (D-12 gate 1 precondition)
hcloud server list ; hcloud firewall list ; hcloud primary-ip list
```

Then re-run `make smoke-live` per Plan 06-07 sequence. Expected outcomes:
- BYO smoke completes without `SSH ERROR host key mismatch`, without `runner_files: failed sudo: a terminal is required`, and without Path B password prompts on the prepared host (Path C honored).
- Cloud smoke completes; destroy leaves zero orphans (no `server_not_stopped`, no `must_be_unassigned`, no `resource_in_use` errors).

## Next Phase Readiness

Plan 06-07 attempt-17+ is unblocked. The smoke-green resume signal for v1.0.0 tag push is owned by Plan 06-07 (06-VERIFICATION.md baseline fill-in + maintainer sign-off), not in scope for this plan's in-tree fixes.

## Self-Check: PASSED

- [x] `internal/remote/system.go` modified — selectHostKeyLine added, scanHostKey routes through it (verified).
- [x] `internal/remote/hostkey_test.go` modified — 4 new tests added (verified).
- [x] `internal/ops/probes.go` modified — host_key_match doc marker added on ProbeRemoteStatus (verified).
- [x] `internal/ops/probes_test.go` modified — TestStatusHostKeyMatchesUpTimeFingerprint added (verified).
- [x] `internal/cli/down.go` modified — sudo-prompt gate drops sshReachable (verified).
- [x] `internal/cli/down_test.go` modified — TestDown_SudoProbeRunsEvenWhenSSHReachableFalse added (verified).
- [x] `internal/provider/hetzner/destroy.go` modified — UnassignPrimaryIP block removed; AutoDelete reference added (verified).
- [x] `internal/provider/hetzner/destroy_test.go` modified — TestDestroy_AutoDeleteCascadeNoUnassign added; existing tests updated for no-unassign order (verified).
- [x] `internal/provider/hetzner/provision.go` modified — annotated ServerCreatePublicNet block (verified).
- [x] `internal/provider/hetzner/provision_test.go` modified — TestProvisionEnablesPublicIPsWithoutOverridingForBug26 added (verified).
- [x] `internal/provider/hetzner/client.go` modified — UnassignPrimaryIP doc updated (verified).
- [x] `internal/bootstrap/sudoers.go` modified — svc.sh glob path (verified).
- [x] `internal/bootstrap/sudoers_test.go` modified — TestRenderSudoersEntryUsesSvcShGlob added; TestRenderSudoersEntry updated (verified).
- [x] Commit `18e0a67` (Bug 24) found in git log — verified.
- [x] Commit `b25bdd5` (Bug 25) found in git log — verified.
- [x] Commit `959cb46` (Bug 26) found in git log — verified.
- [x] Commit `e3428f9` (Bug 27) found in git log — verified.

---
*Phase: 06-release-upgrade-docs-and-v1-validation*
*Plan: 11*
*Completed: 2026-05-06*
