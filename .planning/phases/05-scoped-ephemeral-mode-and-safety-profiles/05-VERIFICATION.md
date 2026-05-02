---
phase: 05-scoped-ephemeral-mode-and-safety-profiles
verified: 2026-05-02T16:15:00Z
status: passed
score: 4/4 must-haves verified
---

# Phase 5: Scoped Ephemeral Mode and Safety Profiles Verification Report

**Phase Goal:** RunnerKit offers an explicit stronger-isolation ephemeral option without pretending to be a full autoscaling fleet manager, and helps developers choose the right mode for their workload.

**Verified:** 2026-05-02T16:15:00Z
**Status:** passed
**Re-verification:** No (initial verification)

## Goal Achievement

### Observable Truths (Success Criteria)

| #   | Truth                                                                                                                                              | Status     | Evidence                                                                                                                                                                                                                          |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Developer can understand the cost, isolation, cleanup, and operational tradeoffs between persistent and ephemeral modes before selecting a mode. | VERIFIED | `internal/runmode/mode.go` defines `Tradeoffs` struct with Cost/Isolation/Cleanup/Operations/Logs and `Evaluate` returns full Decision. `internal/cli/up.go` calls `runmode.Evaluate` and `renderModeTradeoffs` BEFORE side effects. JSON output includes `tradeoffs`, `recommended_for`, `not_recommended_for`, `warnings`. |
| 2   | Developer can choose an explicit ephemeral runner option/profile when they want stronger isolation.                                              | VERIFIED | `runnerkit up` exposes `--mode persistent|ephemeral`, `--ephemeral-ttl 24h`, `--allow-ephemeral-byo-risk` flags. Interactive prompt presents three explicit options (persistent-byo, ephemeral-byo, ephemeral-cloud) BEFORE mutation. |
| 3   | Ephemeral runs have bounded lifecycle behavior with cleanup finalizers, TTL-style safeguards, and useful log preservation.                       | VERIFIED | `bootstrap.RenderEphemeralServiceScript` writes `Restart=no` + `ExecStopPost=<finalizer> completed`. `RenderEphemeralTTLTimerScript` writes `OnActiveSec=24h`. `RenderEphemeralFinalizerScript` preserves `Runner_*.log`/`Worker_*.log`/`systemd-journal.log`. `cli/down.go` and `cli/destroy.go` invoke `ephemeral.logs.preserve` BEFORE file/provider deletion. |
| 4   | Developer can read safety guidance explaining when persistent runners are unsafe and when ephemeral mode is recommended.                         | VERIFIED | `docs/safety.md` exists with all 10 required UI-SPEC headings. README links to safety guide. BYO/cloud quickstarts no longer say "deferred" and recommend ephemeral cloud for risky workloads. `TestSafetyDocsGrepContract` enforces docs regression. |

**Score:** 4/4 truths verified

### Required Artifacts

#### Plan 05-01 Artifacts

| Artifact                          | Expected                                                                | Status     | Details                                                                                                                                                  |
| --------------------------------- | ----------------------------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/runmode/mode.go`        | Mode constants, profile constants, Tradeoffs/Decision, Normalize/Evaluate, DefaultEphemeralTTL=24h, ProfileEphemeralCloud | VERIFIED | All constants present; Normalize/Evaluate exported; ProfileEphemeralCloud constant confirmed                                                              |
| `internal/cli/up.go`              | Mode flags, "Choose runner mode for `owner/name`:" prompt, tradeoff rendering, ephemeral cloud lifecycle | VERIFIED | All three flags registered; prompt copy present; `runmode.Evaluate` called BEFORE setup-path mutation; `bootstrap.ApplyEphemeral` invoked after readiness gates |
| `internal/github/safety.go`       | Updated public/fork persistent risk copy and safer ephemeral cloud recommendation | VERIFIED | `PublicRepoRiskBody`, `PublicRepoRiskNextAction`, and `DangerousPersistentOverrideCopy` constants present with exact UI-SPEC strings                       |
| `internal/labels/labels.go`       | Persistent and ephemeral label/snippet helpers, EphemeralRunnerName | VERIFIED | `ModeEphemeral`, `RepoScopedLabel`, `EphemeralRunnerName` exported; persistent name `runnerkit-owner-repo-local` preserved                                  |

#### Plan 05-02 Artifacts

| Artifact                              | Expected                                                                | Status     | Details                                                                                                                                                  |
| ------------------------------------- | ----------------------------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/bootstrap/script.go`        | Ephemeral install/finalizer/service/TTL/log-preservation render funcs | VERIFIED | All five `RenderEphemeral*` functions present; contains `--ephemeral`, `Restart=no`, `ExecStopPost=`, `OnActiveSec=24h`, `Runner_*.log`, `Worker_*.log`, `systemd-journal.log` |
| `internal/bootstrap/install.go`       | ApplyEphemeral with command sequence | VERIFIED | `ApplyEphemeral` exported; eight command IDs in correct order: `fix_dependencies`, `create_runner_user`, `download_runner`, `configure_ephemeral_runner`, `install_ephemeral_finalizer`, `install_ephemeral_service`, `install_ephemeral_ttl_timer`, `verify_ephemeral_service` |
| `internal/cli/up.go` (ephemeral path) | `buildEphemeralRepositoryState`, ephemeral completion output | VERIFIED | Helpers `ephemeralSuffix`, `ephemeralLogArchivePath`, `ephemeralServiceName`, `ephemeralCleanupCommand` present; "Ephemeral runner ready" rendered with TTL/finalizer/cleanup sentences |
| `internal/state/schema.go`            | EphemeralMetadata struct on RepositoryState | VERIFIED | Field `Ephemeral EphemeralMetadata` with `json:"ephemeral,omitempty"`; struct fields `Enabled`, `TTL`, `ExpiresAt`, `LogArchivePath`, `FinalizerStatus`, `CleanupCommand` plus `SafetyMetadata.SafetyProfile` |
| `internal/ops/status.go`              | Mode-aware ephemeral terminal/TTL health states | VERIFIED | `EphemeralFact`, reason constants `ephemeral_waiting/busy/completed/ttl_expired/cleanup_pending`, `classifyEphemeral` runs BEFORE persistent `github_runner_missing` |

#### Plan 05-03 Artifacts

| Artifact                                         | Expected                                                                | Status     | Details                                                                                                                                          |
| ------------------------------------------------ | ----------------------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `docs/safety.md`                                 | Self-hosted Runner Safety Guide with all 10 UI-SPEC headings | VERIFIED | All headings present (`# Self-hosted Runner Safety Guide`, `## Quick recommendation`, `## Persistent vs ephemeral tradeoffs`, etc.); tradeoffs table; non-goal bullets |
| `README.md`                                      | Top-level link to safety guide and ephemeral cloud commands | VERIFIED | Links `docs/safety.md`; contains ephemeral cloud commands, exact Hetzner cost caveat, both runs-on snippets, lower-case validation phrase |
| `internal/cli/up_ephemeral_e2e_test.go`          | Five fake E2E tests | VERIFIED | All 5 tests present: `TestPersistentPrivateDefaultStillUsesPersistentProfile`, `TestPublicPersistentBlocksAndRecommendsEphemeralCloud`, `TestEphemeralCloudRecommendedForPublicRepo`, `TestEphemeralBYOPublicRequiresAcknowledgement`, `TestEphemeralBYOTrustedPrivateUsesDownCleanup` |
| `internal/cli/docs_test.go`                      | Docs grep contract assertions | VERIFIED | Both `TestSafetyGuideDocsContainRequiredCopy` and `TestSafetyDocsGrepContract` present (4 grep occurrences total) |

### Key Link Verification

| From                                              | To                                          | Via                                                                                | Status | Details                                                                            |
| ------------------------------------------------- | ------------------------------------------- | ---------------------------------------------------------------------------------- | ------ | ---------------------------------------------------------------------------------- |
| `internal/cli/up.go`                              | `internal/runmode/mode.go`                  | `runmode.Evaluate(repo, runmode.Options{...})` BEFORE setup mutation              | WIRED  | Line 358 in up.go calls `runmode.Evaluate`; line 312 calls `runmode.Normalize`     |
| `internal/cli/up.go`                              | `internal/labels/labels.go`                 | `labels.EphemeralRunnerName` and ephemeral label set construction                  | WIRED  | `ModeEphemeral` referenced in up.go; ephemeral name and labels built via labels pkg |
| `internal/state/schema.go`                        | `internal/runmode/mode.go`                  | `SafetyMetadata.SafetyProfile` records profile name from runmode                   | WIRED  | State writes `safety-profile` from `decision.SafetyProfile`; ephemeral path stores `ephemeral-byo` or `ephemeral-cloud` |
| `internal/cli/up.go`                              | `internal/bootstrap/install.go`             | `bootstrap.ApplyEphemeral(ctx, executor, target, opts)` after readiness/preflight | WIRED  | Two callsites in up.go (BYO and cloud) at lines 202 and 704                        |
| `internal/cli/up.go`                              | `internal/provider/profile.go`              | `provider.ProvisionInput.Mode` drives `HetznerOwnershipTags` mode tag             | WIRED  | profile.go uses `strings.TrimSpace(input.Mode)` to set `mode=ephemeral` tag        |
| `internal/cli/logs.go`                            | `internal/ops/logs.go`                      | logs.ephemeral.archive.list/tail collect from `state.Ephemeral.LogArchivePath`     | WIRED  | Both command IDs present; sections `ephemeral_runner_diag` and `ephemeral_systemd_journal` |
| `docs/safety.md`                                  | `internal/cli/docs_test.go`                 | TestSafetyGuideDocsContainRequiredCopy / TestSafetyDocsGrepContract                | WIRED  | Both test functions present and run as part of full `go test ./...` suite          |
| `internal/cli/up_ephemeral_e2e_test.go`           | `internal/cli/up.go`                        | Fake E2E tests exercise mode flag, safety gates, lifecycle output                 | WIRED  | Five tests reference `--mode ephemeral` and exercise full Cobra command path       |
| `internal/cli/destroy_test.go`                    | `internal/cli/destroy.go` (logs preserve)   | `ephemeral.logs.preserve` runs before provider Destroy                            | WIRED  | Test asserts ordering; destroy.go has command at line 162                          |
| `internal/cli/down_test.go`                       | `internal/cli/down.go` (logs preserve)      | `ephemeral.logs.preserve` runs before `down.files.remove`                         | WIRED  | Test asserts ordering; down.go has command at line 266                             |

### Data-Flow Trace (Level 4)

| Artifact                       | Data Variable                       | Source                                                              | Produces Real Data | Status   |
| ------------------------------ | ----------------------------------- | ------------------------------------------------------------------- | ------------------ | -------- |
| `runmode.Decision`             | `Tradeoffs`, `Warnings`             | `runmode.Evaluate(repo, opts)` returns Decision with full tradeoffs | Yes                | FLOWING  |
| `bootstrap.Options.RunnerToken`| Registration token                  | Just-in-time `CreateRegistrationToken` after readiness gates        | Yes (redacted)     | FLOWING  |
| `state.RepositoryState.Ephemeral` | EphemeralMetadata with TTL/log_archive/cleanup_command | `buildEphemeralBYORepositoryState` / `buildEphemeralCloudRepositoryState` | Yes                | FLOWING  |
| `ops.ObservedRunner.Ephemeral` | Live finalizer status               | `status.ephemeral.state` remote sentinel read + saved state         | Yes                | FLOWING  |
| `provider.ProvisionInput.Mode` | "ephemeral" tag                     | `buildCloudProvisionInput` passes `modeDecision.Mode`               | Yes                | FLOWING  |

### Behavioral Spot-Checks

| Behavior                                                 | Command                                                                                  | Result                                                  | Status |
| -------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------- | ------ |
| Full Go test suite passes                                | `go test ./... -count=1`                                                                  | All packages OK (bootstrap, cli, github, labels, ops, runmode, state, ...) | PASS   |
| Phase 5 docs grep contract: persistent self-hosted phrase | `grep -c "persistent self-hosted runners" README.md docs/safety.md`                       | README:1, safety.md:1 (also Capital-P variants in BYO and safety) | PASS   |
| Phase 5 docs grep contract: not-fleet-manager            | `grep -c "Ephemeral mode is not a fleet manager" README.md docs/safety.md`                | README:1, safety.md:1                                   | PASS   |
| Phase 5 docs grep contract: Hetzner cost caveat          | `grep -c "Estimated cost is approximate. Hetzner pricing varies" README.md docs/safety.md docs/cloud-quickstart.md` | README:1, safety.md:2, cloud-quickstart.md:1 | PASS   |
| Phase 5 ephemeral cloud command in docs                  | `grep -c "runnerkit up --repo owner/name --mode ephemeral --cloud hetzner" README.md docs/*.md` | README:4, safety.md:6, byo-quickstart.md:1, cloud-quickstart.md:4 | PASS   |
| Phase 5 commits present                                  | `git log` for all 16 task commits (871f9fb..98ec323)                                     | All 16 commits found in history                         | PASS   |

### Requirements Coverage

| Requirement | Source Plan       | Description                                                                                                         | Status    | Evidence                                                                                                                                                              |
| ----------- | ----------------- | ------------------------------------------------------------------------------------------------------------------- | --------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| RUN-02      | 05-01, 05-02, 05-03 | Developer can choose an explicit ephemeral runner option/profile when they want stronger isolation.               | SATISFIED | `--mode ephemeral` flag, `ProfileEphemeralBYO`/`ProfileEphemeralCloud`, `bootstrap.ApplyEphemeral` lifecycle, EphemeralMetadata persisted, fake E2E tests prove flow. |
| RUN-04      | 05-01, 05-03     | Developer can understand the tradeoff between persistent and ephemeral modes before selecting a mode.              | SATISFIED | `runmode.Tradeoffs` with five fields, `renderModeTradeoffs` renders before mutation, JSON exposes `tradeoffs`/`recommended_for`/`not_recommended_for`, docs guide.   |
| DOC-03      | 05-03             | Developer can read safety guidance explaining when persistent self-hosted runners are unsafe and when ephemeral mode is recommended. | SATISFIED | `docs/safety.md` Self-hosted Runner Safety Guide; README/BYO/cloud quickstarts updated; `TestSafetyDocsGrepContract` regression contract.                            |

All three requirement IDs from PLAN frontmatter (RUN-02, RUN-04, DOC-03) are accounted for and SATISFIED. REQUIREMENTS.md confirms all three marked Complete and mapped to Phase 5. No orphaned requirement IDs.

### Anti-Patterns Found

| File                  | Line | Pattern              | Severity | Impact                                                                                          |
| --------------------- | ---- | -------------------- | -------- | ----------------------------------------------------------------------------------------------- |
| `internal/cli/up.go`  | 813  | `return ""` fallback | Info     | Benign: fallback in public-key file lookup helper after iterating candidates; not a stub.       |

No TODO/FIXME/HACK markers in any phase 5 file. No empty handlers. No hardcoded empty placeholder data. No console-log-only implementations.

### Human Verification Required

None. Phase 5 is automatable and was fully covered by deterministic Go tests and docs grep contracts. The optional live cloud-ephemeral one-job smoke (real GitHub repo + `HCLOUD_TOKEN`) is intentionally deferred to Phase 6 release validation per `05-03-SUMMARY.md`'s "User Setup Required" notes; it is not required to declare Phase 5 goal achieved because the fake E2E tests exercise the full Cobra command path with realistic dependencies.

### Gaps Summary

No gaps. All four success criteria met, all three requirements satisfied, full uncached `go test ./...` exits 0, every key link wired, every artifact substantive, and the safety docs regression contract is encoded in test code so future copy edits cannot silently regress.

The phase delivers exactly what the goal asked for:
1. Explicit `--mode ephemeral` option with both BYO and cloud paths.
2. Real RunnerKit-managed one-job lifecycle (config.sh `--ephemeral`, one-shot systemd unit, finalizer, 24h TTL timer, log preservation, cleanup ordering).
3. Tradeoff transparency before mutation (human + JSON output exposing cost/isolation/cleanup/operations/logs).
4. Safety documentation explicitly stating RunnerKit is not a fleet manager / autoscaler / ARC and that BYO ephemeral is not a clean VM.

---

_Verified: 2026-05-02T16:15:00Z_
_Verifier: Claude (gsd-verifier)_
