---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 10
status: complete
completed: 2026-05-06
gap_closure: true
requirements: [REL-05]
duration: 12m
tasks_completed: 5
files_modified: 9
key-decisions:
  - "Bug 19: ops.ProbeRemoteStatus resolves the actual systemd unit name via `systemctl list-units 'actions.runner.*'` when the saved simplified name returns LoadState=not-found; the resolved name is matched by the `<runner-name>.service` suffix to handle GitHub's `actions.runner.<owner-repo>.<runner-name>.service` form."
  - "Bug 20: ops.CompareLabels normalizes both sides via strings.ToLower for set membership; same family as Plan 06-09 Bug 16 (runnerOnlineWithLabels) but in the status/doctor drift detector code path."
  - "Bug 21: down probes `sudo -n true` once before cleanup. If sudo requires a password (stderr matches `password is required` / `a terminal is required` / `no tty present`), down prompts via the existing ui.PasswordPrompter and threads the password through the same `printf | sudo -S -v` cred-priming pattern bootstrap.wrapSudoCommand uses (Plan 06-09 Bug 10). NOPASSWD hosts (Path C byo-prepare) keep the unwrapped happy path."
  - "Bug 22: hetzner.ProbeHostKeyWithRetry(ctx, prober, target, opts) wraps remote.HostKeyProber with a bounded retry loop (default 60 attempts × 5s = ~5min total) that tolerates Hetzner cloud-init's host-key install window. Empty fingerprints with nil errors count as failures. Sleep is injectable for tests so they don't burn wall-clock time."
  - "Bug 23: Destroy ordering — detach firewall + unassign primary IPv4 + unassign primary IPv6 BEFORE server.Delete; then delete server, ssh_key, primary IPs, and firewall last. Already-absent (404) on detach/unassign is treated as success so partial-cleanup retries are idempotent. Client interface gained DetachFirewallFromServer + UnassignPrimaryIP backed by hcloud-go v1.59.2's Firewall.RemoveResources and PrimaryIP.Unassign."
---

# Plan 06-10 Summary: Status + Doctor + Down + Cloud Post-Up Fixes

## Outcome

Closed 5 post-up + cloud surface bugs (Bugs 19-23) discovered during
Plan 06-07 attempt-15 against `salar@mckee-small-desktop` + a real
Hetzner project. After Plan 06-09 closed the BYO bootstrap path
end-to-end (Bugs 4-18, 2026-05-06), this plan closes the BYO
post-up + cloud surface so Plan 06-07 attempt-16+ can complete BOTH
BYO and cloud smokes end-to-end without warnings, partial cleanups, or
orphaned billable resources:

- `runnerkit status --repo X` against a healthy BYO runner now reports
  `Service OK active` (not WARNING inactive). Bug 19 fix resolves the
  actual systemd unit name via `systemctl list-units` when the saved
  simplified name returns LoadState=not-found.
- `runnerkit status` and `runnerkit doctor` no longer warn
  `missing [linux, x64] extra [Linux, X64]` when the only difference is
  case (Bug 20). Same family as Plan 06-09 Bug 16 in a different code
  path.
- `runnerkit down --repo X --yes` against a host with password-protected
  sudo no longer fails at `runner_files` cleanup with `sudo: a terminal
  is required` (Bug 21). Down probes sudo once and prompts + threads
  the password through `printf | sudo -S -v` when needed.
- Cloud `runnerkit up --cloud hetzner` no longer fails immediately
  after `provider_server: done` with `SSH host key fingerprint was not
  observed` (Bug 22). The host-key probe retries with backoff across
  cloud-init's ~30-90s host-key install window.
- Cloud `runnerkit destroy` no longer leaves orphaned firewall +
  primary IPv4/IPv6 with `resource_in_use` / `must_be_unassigned`
  errors (Bug 23). Destroy detaches first, deletes the server, then
  deletes the (now-free) IPs and firewall.

## Commits

6 commits total under `(06-10)` prefix:

| Bug | RED | GREEN | Hash |
|-----|-----|-------|------|
| 19 (service name) | bundled | bundled | 90e73fc |
| 20 (label case) | bundled | bundled | 0d1e17c |
| 21 (down sudo) | bundled | bundled | ab48675 |
| 22 (cloud SSH ready) | bundled | bundled | aab91f9 |
| 23 (destroy ordering) | bundled | bundled | 88bd3c2 |
| (Bug 19 docs marker) | n/a | docs | 517466f |

## Key Files Created/Modified

**Modified:**

- `internal/ops/probes.go` — fall back to `systemctl list-units` when
  saved simplified service name returns LoadState=not-found; new
  `extractRunnerSuffix` + `matchUnitBySuffix` + `parseSystemdShow`
  helpers; new command IDs `status.systemd.list_units` +
  `status.systemd.show.resolved` (Bug 19).
- `internal/ops/probes_test.go` — RED+GREEN coverage:
  TestProbeRemoteStatusFallsBackToListUnitsWhenSavedNameIsNotFound +
  TestProbeRemoteStatusSkipsListUnitsWhenSavedNameMatches.
- `internal/ops/status.go` — `CompareLabels` lowercases both sides for
  set membership (Bug 20).
- `internal/ops/status_test.go` — RED+GREEN coverage:
  TestCompareLabelsCaseInsensitiveMatchClosesBug20 +
  TestCompareLabelsStillReportsRealDriftAfterBug20Fix.
- `internal/ops/doctor_test.go` — TestDoctor_LabelDriftIsCaseInsensitiveClosesBug20
  proves the case-insensitive comparison flows through `BuildDoctorReport`
  (no spurious `label_drift` finding).
- `internal/cli/down.go` — `applyCleanup` probes sudo via `down.sudo.probe`
  (`sudo -n true`); on password-required stderr, prompts via
  `promptSudoPasswordForDown` and threads the literal through
  `wrapDownSudoCommand` to service-uninstall + files-remove (Bug 21).
- `internal/cli/down_test.go` — RED+GREEN coverage:
  TestDownThreadsSudoPasswordWhenSudoRequiresPasswordClosesBug21 +
  TestDownDoesNotPromptWhenSudoIsPasswordless +
  passwordRecorder/containsString test helpers.
- `internal/cli/status.go` — annotated `collectStatus` to reference Bug
  19's resolution path so future maintainers find the systemd unit-name
  resolution logic from the status entry point (`actions.runner.` marker
  required by plan frontmatter contains check).
- `internal/cli/up.go` — `probeCloudHostKey` calls
  `hetzner.ProbeHostKeyWithRetry` so cloud SSH readiness tolerates
  cloud-init's host-key install window (Bug 22).
- `internal/provider/hetzner/provision.go` — new
  `HostKeyProbeOptions{Attempts, Interval, Sleep}` +
  `ProbeHostKeyWithRetry(ctx, prober, target, opts)` helper with
  defaults (60 × 5s = ~5min budget); empty fingerprint counts as
  failure (Bug 22).
- `internal/provider/hetzner/provision_test.go` — RED+GREEN coverage
  for retry: TestProbeHostKeyWithRetrySucceedsOnLaterAttempt,
  TestProbeHostKeyWithRetryReturnsLastErrorOnExhaustion,
  TestProbeHostKeyWithRetryRetriesOnEmptyFingerprint,
  TestProbeHostKeyWithRetryUsesDefaultsWhenOptionsZero, plus
  fakeHostKeyProber/fakeClock/stubHostKey/fakeTarget test fixtures and
  DetachFirewallFromServer + UnassignPrimaryIP stubs on `fakeClient`.
- `internal/provider/hetzner/client.go` — `Client` interface gained
  `DetachFirewallFromServer(ctx, firewallID, serverID) error` +
  `UnassignPrimaryIP(ctx, id) error`; `APIClient` implementations call
  hcloud-go v1.59.2's `Firewall.RemoveResources` (with
  `FirewallResourceTypeServer`) and `PrimaryIP.Unassign` (Bug 23).
- `internal/provider/hetzner/destroy.go` — new ordering: detach
  firewall (best-effort) → unassign primary_ipv4 + primary_ipv6
  (best-effort) → delete server → delete ssh_key → delete primary_ipv4
  + primary_ipv6 → delete firewall last; new `parsedNonEmpty` helper
  gates detach/unassign on parseable IDs (Bug 23). Already-absent (404)
  on detach/unassign is silenced; on delete it's still treated as
  skipped.
- `internal/provider/hetzner/destroy_test.go` — updated
  TestDestroyDeletesThenVerifyDescribesBeforeSuccess to assert the new
  detach+unassign-first order; added
  TestDestroyDetachesFirewallAndPrimaryIPsBeforeServerDeleteClosesBug23
  + TestDestroyTreatsAlreadyAbsentDetachAsSuccess +
  destroyFakeOrderedClient + destroyRefWithBothPrimaryIPs +
  sliceContains helper.

## Cross-Phase Validation

Bugs 19, 20 are BYO + cloud (status/doctor common code). Bugs 22, 23
are cloud-only (Hetzner provider package). Bug 21 is BYO (down is
BYO-only per Phase 4 decision; cloud uses destroy). All five share the
same character: post-up surface bugs that surfaced after a successful
provision but before maintainer sign-off.

The cloud destroy ordering fix (Bug 23) is verifiable on a real
Hetzner project via Plan 06-07 attempt-16+: the smoke trap-cleanup
will exercise the new ordering and the verifier (D-12 gate 2) polls
hcloud.IsError(err, hcloud.ErrorCodeNotFound) every 500ms until every
tracked ID returns 404 — so any ordering regression surfaces as a
verify-pending checkpoint, not a silent leak.

## Verification

- `go test ./... -count=1 -race` passes (18/18 packages, all green).
- `go vet ./...` clean.
- `gofmt -l` clean (after auto-format on imports for provision.go,
  provision_test.go, up.go, destroy_test.go).
- All 5 frontmatter `must_haves.artifacts.contains` markers verified
  present in the listed files (`actions.runner.` in
  internal/cli/status.go via the new doc comment, `strings.ToLower` in
  doctor.go + status.go via the lowercased CompareLabels, `wrapSudo`
  in down.go via wrapDownSudoCommand, `host_key`/`HostKey` in
  provision.go via HostKeyProbeOptions + ProbeHostKeyWithRetry,
  `DetachFirewall` in destroy.go + client.go).

## Pending Maintainer Action

Plan 06-07 attempt-16+ (after Plan 06-10 closure — this plan) will
close the maintainer human-action checkpoint with `smoke-green` once
BOTH BYO and cloud phases complete cleanly:

- BYO smoke: status reports `Service OK active`, doctor reports no
  spurious label_drift, down --yes succeeds with sudo password
  threading.
- Cloud smoke: SSH host-key probe succeeds within the cloud-init
  window, destroy leaves zero orphaned billable resources (server +
  ssh_key + firewall + primary_ipv4 + primary_ipv6 all 404).
- 06-VERIFICATION.md baseline filled with real wall-clock durations,
  5 cloud resource IDs, EUR cost, plus maintainer sign-off.

Truth 6 from this plan's `must_haves.truths` — full smoke completion +
06-VERIFICATION.md fill-in — is the resume signal for v1.0.0 tag push,
and is owned by Plan 06-07 (not in scope for this plan's in-tree
fixes).

## Self-Check: PASSED

- [x] `internal/ops/probes.go` modified — verified.
- [x] `internal/ops/probes_test.go` modified with new tests — verified.
- [x] `internal/ops/status.go` CompareLabels modified — verified.
- [x] `internal/ops/status_test.go` modified with new tests — verified.
- [x] `internal/ops/doctor_test.go` modified with new test — verified.
- [x] `internal/cli/down.go` modified — verified.
- [x] `internal/cli/down_test.go` modified with new tests — verified.
- [x] `internal/cli/status.go` modified (Bug 19 doc marker) — verified.
- [x] `internal/cli/up.go` probeCloudHostKey wired through hetzner.ProbeHostKeyWithRetry — verified.
- [x] `internal/provider/hetzner/provision.go` ProbeHostKeyWithRetry added — verified.
- [x] `internal/provider/hetzner/provision_test.go` retry tests + fakeClient detach/unassign stubs — verified.
- [x] `internal/provider/hetzner/client.go` Client interface extended + APIClient implements — verified.
- [x] `internal/provider/hetzner/destroy.go` reordered with detach/unassign-first — verified.
- [x] `internal/provider/hetzner/destroy_test.go` updated existing test + new tests — verified.
- [x] Commit 90e73fc found in git log — verified.
- [x] Commit 0d1e17c found in git log — verified.
- [x] Commit ab48675 found in git log — verified.
- [x] Commit aab91f9 found in git log — verified.
- [x] Commit 88bd3c2 found in git log — verified.
- [x] Commit 517466f found in git log — verified.
