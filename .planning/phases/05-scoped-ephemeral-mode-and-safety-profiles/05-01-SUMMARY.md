---
phase: 05-scoped-ephemeral-mode-and-safety-profiles
plan: "01"
subsystem: cli-mode-and-safety-profiles
tags: [cli, mode, ephemeral, persistent, safety, ux]
requires:
  - phase: 04-recommended-cloud-path-and-billable-cleanup
    provides: BYO and Hetzner cloud setup paths, provider Validate/Plan/Provision lifecycle, persistent labels and runner naming, and runnerkit destroy cleanup contract
provides:
  - internal/runmode package with ModePersistent/ModeEphemeral, persistent-trusted/persistent-risky/ephemeral-byo/ephemeral-cloud safety profiles, DefaultEphemeralTTL, Tradeoffs, Decision, Normalize, and Evaluate
  - internal/labels ModeEphemeral/ModePersistent, RepoScopedLabel, EphemeralRunnerName helpers preserving persistent backwards compatibility
  - internal/state SafetyMetadata.SafetyProfile field with safety_profile,omitempty serialization
  - runnerkit up --mode persistent|ephemeral, --ephemeral-ttl 24h, --allow-ephemeral-byo-risk flags
  - Interactive Choose runner mode prompt with the three UI-SPEC mode/profile options
  - renderModeTradeoffs human and JSON tradeoff payloads (mode, safety_profile, ephemeral, ttl, tradeoffs, recommended_for, not_recommended_for, warnings, redactions_applied)
  - enforceModeSafetyDecision side-effect-safe gate for persistent risky vs ephemeral byo vs ephemeral cloud
  - Updated public/fork persistent risk copy recommending runnerkit up --repo owner/name --mode ephemeral --cloud hetzner
affects: [phase-05-02, phase-05-03, cli, github, labels, state]
tech-stack:
  added: []
  patterns:
    [
      mode/profile decision before setup-path mutation,
      mode-aware safety gates with typed acknowledgement and non-interactive override,
      ephemeral runner naming with short-id collision avoidance,
    ]
key-files:
  created:
    - internal/runmode/mode.go
    - internal/runmode/mode_test.go
    - internal/cli/up_modes_test.go
    - internal/github/safety_test.go
  modified:
    - internal/labels/labels.go
    - internal/labels/labels_test.go
    - internal/state/schema.go
    - internal/state/state_test.go
    - internal/cli/up.go
    - internal/cli/up_cloud_test.go
    - internal/github/safety.go
key-decisions:
  - "Mode and safety-profile decisions live in a new internal/runmode package so the CLI, labels, state, and tests share one typed Decision."
  - "The runner-mode prompt replaces the previous setup-path prompt; selecting persistent-byo or ephemeral-byo lets resolveBYOTarget collect a host while ephemeral-cloud selects --cloud hetzner."
  - "Public/fork persistent setup blocks with the new UI-SPEC body and DangerousPersistentOverrideCopy before any GitHub auth, registration token, remote, provider, or state mutation."
  - "Ephemeral cloud is the recommended public/fork path; the safety gate appends 'Use ephemeral cloud runner: runnerkit up --repo owner/name --mode ephemeral --cloud hetzner' so docs greps and human dry-run output both succeed."
  - "Ephemeral BYO on public/fork requires either typed 'use ephemeral byo for owner/name' acknowledgement or non-interactive --allow-ephemeral-byo-risk --yes; otherwise it blocks before remote probe and registration token."
patterns-established:
  - "resolveModeDecision runs before resolveSetupPath/cloud planning so users see persistent-vs-ephemeral tradeoffs before any mutation."
  - "Mode-selection JSON keys (mode, safety_profile, ephemeral, ttl, tradeoffs, recommended_for, not_recommended_for, warnings) are produced by modeSelectionPayload and shared by every mode-fronting JSON path."
  - "Ephemeral runner names use runnerkit-owner-repo-ephemeral-<short-id> while persistent keeps runnerkit-owner-repo-local for full backwards compatibility."
requirements-completed: [RUN-02, RUN-04]
duration: 16 min
completed: 2026-05-02
---

# Phase 05 Plan 01: Mode/Profile Selection and Safety Policy Summary

**runnerkit up now explains and gates persistent vs ephemeral runner mode tradeoffs before any GitHub, remote, provider, or state side effect.**

## Performance

- **Duration:** 16 min
- **Started:** 2026-05-02T14:52:07Z
- **Completed:** 2026-05-02T15:08:23Z
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments

- Added the new `internal/runmode` package with `ModePersistent`/`ModeEphemeral` constants, four safety profile constants (`persistent-trusted`, `persistent-risky`, `ephemeral-byo`, `ephemeral-cloud`), `DefaultEphemeralTTL = 24 * time.Hour`, typed `Tradeoffs`/`Decision`/`Options`, `Normalize` parser, and `Evaluate` so the CLI receives a typed decision with exact UI-SPEC tradeoff strings.
- Added `internal/labels.ModePersistent`/`ModeEphemeral`, `RepoScopedLabel`, and `EphemeralRunnerName` helpers so persistent stays backwards-compatible (`runnerkit-owner-repo-local`) while ephemeral derives `runnerkit-owner-repo-ephemeral-<short-id>` with length capping.
- Added `internal/state.SafetyMetadata.SafetyProfile` (`json:"safety_profile,omitempty"`) so saved state can record the chosen profile without breaking older states.
- Wired `runnerkit up --mode persistent|ephemeral`, `--ephemeral-ttl 24h`, and `--allow-ephemeral-byo-risk` flags. The interactive selection now prompts `Choose runner mode for owner/name:` with the three UI-SPEC mode options before any GitHub auth or state mutation.
- Implemented `renderModeTradeoffs` so human output prints `Mode`, `Safety profile`, `Cost`, `Isolation`, `Cleanup`, `Operations`, `Logs`, `Recommended for`, `Not recommended for`, the BYO clean-VM caveat, the exact ephemeral cloud cost caveat, the `TTL safeguard` copy, and the not-fleet-manager warning. JSON output exposes `mode`, `safety_profile`, `ephemeral`, `ttl`, `tradeoffs`, `recommended_for`, `not_recommended_for`, `warnings`, plus `redactions_applied: true`.
- Updated `internal/github/safety.go` with the new public/fork persistent body, ephemeral cloud `PublicRepoRiskNextAction`, and `DangerousPersistentOverrideCopy`, and replaced `enforceSafetyDecision` with mode-aware `enforceModeSafetyDecision` so persistent risky blocks with the new copy, ephemeral cloud surfaces the ephemeral cloud command without blocking public/fork, and ephemeral BYO on public/fork requires typed acknowledgement or `--allow-ephemeral-byo-risk --yes` before remote probe or registration token creation.

## Task Commits

1. **Task 05-01-01: Add mode, tradeoff, safety-profile, label, and state primitives** — `871f9fb` (test) + `682e857` (feat)
2. **Task 05-01-02: Wire runnerkit up mode flags, interactive profile selection, and tradeoff rendering** — `b9544cc` (test) + `946fd1f` (feat)
3. **Task 05-01-03: Enforce persistent-risk and BYO-ephemeral caveats with side-effect-safe safety policy** — `d28817b` (test) + `cdab99a` (feat)

## Files Created/Modified

- `internal/runmode/mode.go` — Mode/profile constants, `DefaultEphemeralTTL`, `Tradeoffs`, `Decision`, `Options`, `Normalize`, `Evaluate`, plus shared warning constants used by the CLI safety gate.
- `internal/runmode/mode_test.go` — Constants/normalize/evaluate tradeoff coverage with exact UI-SPEC strings.
- `internal/labels/labels.go` — Adds `ModePersistent`/`ModeEphemeral`, `RepoScopedLabel`, `EphemeralRunnerName` (length-capped, short-id-preserving) while preserving `DefaultMode = persistent`.
- `internal/labels/labels_test.go` — Mode constants assertion, ephemeral runs-on snippet, repo-scoped label slugging, ephemeral runner name format and length cap.
- `internal/state/schema.go` — `SafetyMetadata.SafetyProfile` with `safety_profile,omitempty`.
- `internal/state/state_test.go` — Serialization assertion plus backwards-compatible load test for older states without `safety_profile`.
- `internal/cli/up.go` — `--mode/--ephemeral-ttl/--allow-ephemeral-byo-risk` flags, UI-SPEC copy constants, `resolveModeDecision`, `renderModeTradeoffs`, `modeSelectionPayload`, `shortEphemeralID`, `buildModeLabelSet`, mode-aware `renderDryRun`/`renderCloudProvisionPlan`, and `enforceModeSafetyDecision` with persistent/ephemeral byo/ephemeral cloud branches.
- `internal/cli/up_cloud_test.go` — Updated the previous setup-path prompt expectation to the new `Choose runner mode for owner/repo:` prompt with the three UI-SPEC labels.
- `internal/cli/up_modes_test.go` — New tests for interactive prompt copy, ephemeral BYO/cloud dry-run rendering, JSON mode payload, invalid-mode error, persistent backwards-compat, public-persistent block timing, public ephemeral cloud recommendation, and public ephemeral BYO acknowledgement gate.
- `internal/github/safety.go` — Tightened `PublicRepoRiskBody`, ephemeral cloud `PublicRepoRiskNextAction`, and added `DangerousPersistentOverrideCopy`.
- `internal/github/safety_test.go` — Pins each constant and asserts the public evaluate decision warnings include the ephemeral cloud command.

## Decisions Made

- The previous setup-path prompt is replaced by the explicit mode/profile prompt; existing `TestUpInteractiveNoHostOffersBYOAndCloudChoices` is retitled to assert the new prompt copy and labels.
- `enforceModeSafetyDecision` mutates the supplied `*runmode.Decision` so the appended public/fork warnings flow through `renderModeTradeoffs` and JSON `warnings` arrays without re-running enforcement.
- Ephemeral BYO acknowledgement copy follows the plan exactly: typed `use ephemeral byo for owner/name` interactive input, or non-interactive `--allow-ephemeral-byo-risk --yes`. The error remediation points users at `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner` for stronger isolation.
- The public/fork ephemeral cloud "Use ephemeral cloud runner" warning includes the full `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner` command so docs greps and human dry-run assertions both succeed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Mode tradeoff warnings appended by safety gate did not flow to JSON `warnings` array**

- **Found during:** Task 05-01-03 (public ephemeral cloud dry-run test).
- **Issue:** `enforceModeSafetyDecision` appended public/fork ephemeral cloud warnings to the `runmode.Decision`, but the function received the decision by value, so warnings were lost before `renderModeTradeoffs` saw them.
- **Fix:** Changed the signature to take `*runmode.Decision` and updated the caller in `runUp`. The renderer now de-duplicates appended warnings against the canonical per-profile copy so the same sentence does not appear twice.
- **Files modified:** `internal/cli/up.go`.
- **Verification:** `go test ./internal/cli/... ./internal/github/...`.
- **Committed in:** `cdab99a`.

### Test-Tolerance Adjustments

**2. [Test ergonomics] Long UI-SPEC sentences wrap at 80 columns**

- **Found during:** Tasks 05-01-02 and 05-01-03.
- **Issue:** The `internal/ui` renderer caps wrap width at 100 and tests that assert the literal long sentences (BYO clean-VM caveat, exact Hetzner cost caveat, public-fork persistent body, ephemeral cloud command) saw them split across lines in the rendered output.
- **Fix:** Tests assert short prefixes verbatim and flatten whitespace before checking the long full sentences (`strings.Join(strings.Fields(out), " ")`). The renderer is unchanged; the human sentence content remains exactly as required by the UI-SPEC.
- **Files modified:** `internal/cli/up_modes_test.go`.
- **Committed in:** `946fd1f`, `cdab99a`.

---

**Total deviations:** 1 auto-fixed (1 bug); 1 test-tolerance adjustment.
**Impact on plan:** Minor; deviations strengthen the plan-required behavior and stay within the 05-01 scope.

## Issues Encountered

- The `internal/cli` test infrastructure does not let `executeForTest` inject a fake provider, so cloud-related tests construct `Dependencies` directly with `provider.NewRegistry(&provider.FakeProvider{})`.
- Existing `TestUpInteractiveNoHostOffersBYOAndCloudChoices` had to be updated to the new mode/profile prompt copy because Phase 5 deliberately replaces the previous setup-path prompt.

## User Setup Required

None — no `USER-SETUP.md` was generated. Live verification of the public/fork ephemeral cloud recommendation still depends on a real GitHub repo plus `HCLOUD_TOKEN`, but Plan 05-02 wires the live ephemeral lifecycle.

## Next Phase Readiness

- 05-02 can build the ephemeral lifecycle on top of `runmode.Mode/Profile` constants, ephemeral labels/runner names, the new safety enforcement gate, and the `--ephemeral-ttl` flag.
- 05-03 will use the new `PublicRepoRiskBody`/`PublicRepoRiskNextAction`/`DangerousPersistentOverrideCopy` strings, the ephemeral runs-on snippet, and the mode/profile JSON payload for docs and safety-guide assertions.

## Verification

- `go test ./internal/runmode/... ./internal/labels/... ./internal/state/...` exits 0.
- `go test ./internal/github/... ./internal/cli/... ./internal/runmode/... ./internal/labels/...` exits 0.
- `go test ./...` exits 0.
- `grep -R "ModeEphemeral" internal/runmode internal/labels internal/cli` returns matches.
- `grep -R "Choose runner mode for \`owner/name\`:" internal/cli/up.go internal/cli/up_modes_test.go` returns matches.
- `grep -R "Ephemeral mode is not a fleet manager" internal/cli internal/runmode` returns matches.
- `grep -R "runnerkit up --repo owner/name --mode ephemeral --cloud hetzner" internal/github internal/cli` returns matches.
- `grep -R "SafetyProfile" internal/state internal/cli` returns matches.
- `grep -R "Estimated cost is approximate. Hetzner pricing varies by region and time" internal/cli` returns matches.

## Self-Check: PASSED

- All key files exist (`internal/runmode/mode.go`, `internal/runmode/mode_test.go`, `internal/cli/up_modes_test.go`, `internal/github/safety_test.go`, `internal/labels/labels.go`, `internal/labels/labels_test.go`, `internal/state/schema.go`, `internal/state/state_test.go`, `internal/cli/up.go`, `internal/cli/up_cloud_test.go`, `internal/github/safety.go`).
- All task commits are present in git history: `871f9fb`, `682e857`, `b9544cc`, `946fd1f`, `d28817b`, `cdab99a`.
- Required verification commands pass: `go test ./internal/runmode/... ./internal/labels/... ./internal/state/...`, `go test ./internal/github/... ./internal/cli/...`, `go test ./...`. Required grep checks confirm `ModeEphemeral`, the mode prompt copy, the not-fleet-manager warning, the ephemeral cloud command, `SafetyProfile`, and the exact Hetzner cost caveat all appear in the expected files.

---

_Phase: 05-scoped-ephemeral-mode-and-safety-profiles_
_Completed: 2026-05-02_
