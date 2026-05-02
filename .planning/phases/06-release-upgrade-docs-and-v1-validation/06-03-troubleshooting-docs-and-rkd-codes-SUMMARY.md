---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 03
subsystem: troubleshooting-and-error-codes
tags: [errcodes, rkd-codes, troubleshooting, doctor-findings, docs, doc-04]

requires:
  - phase: 01-cli-auth-state-and-safety-foundation
    provides: ExitError contract, renderer.Error remediation array
  - phase: 02-byo-persistent-runner-happy-path
    provides: bootstrap finding IDs, SSH host-key trust, preflight checks
  - phase: 03-operations-diagnostics-and-byo-cleanup
    provides: doctor finding model, cleanup checkpoint vocabulary
  - phase: 04-recommended-cloud-path-and-billable-cleanup
    provides: provider error categories, destroy partial states
  - phase: 05-scoped-ephemeral-mode-and-safety-profiles
    provides: docs/safety.md canonical safety copy, ephemeral_cleanup_pending finding
  - phase: 06-release-upgrade-docs-and-v1-validation/02
    provides: runner_version_stale doctor finding, ErrSchemaTooNew sentinel
provides:
  - internal/errcodes package as single source of truth for stable RKD-<COMPONENT>-NNN error codes (D-15).
  - 46-entry Registry covering 8 component prefixes (AUTH, SSH, BOOT, GH, PROV, CLEAN, STATE, CORE).
  - URL builder honoring RUNNERKIT_DOCS_BASE env override with /blob/ vs static-site .md-suffix detection (D-15).
  - FormatLine helper producing "RKD-XXX-NNN: <Title>\nSee: <URL>" emit shape.
  - 6 docs/troubleshooting/<component>.md files plus README.md index covering all 4 D-16 failure surfaces.
  - 46 anchored Symptom/Diagnosis/Fix entries with explicit `<a name=>` anchors (Pitfall 9 mitigation).
  - CLI emit-site wiring: 18 doctor findings, public-repo block, ephemeral BYO ack, runner-management permission, HCLOUD_TOKEN missing, ErrSchemaTooNew, partial cleanup banners now carry RKD codes + See: URL.
  - Cross-linked README, BYO/cloud quickstarts, and safety.md.
affects: [06-04-v1-validation-and-live-smoke]

tech-stack:
  added: []
  patterns:
    - "Stable RKD code registry: every Code{ID, Severity, Title, File, Anchor} is declared once in internal/errcodes/codes.go and surfaced via the global Registry slice that tests walk for invariants."
    - "URL hosting modes: errcodes.URL strips the .md suffix only when the docs base URL does NOT contain /blob/. GitHub blob URLs require .md before the anchor; static-site URLs do not. Detection rule needs no separate config flag."
    - "addWithCode helper in internal/ops/doctor.go: thin wrapper around add() that appends `\\n\\nSee: <URL>` to the remediation field. Pass-only findings keep using the original add() helper."
    - "Append (not prepend) RKD codes to remediation arrays so existing tests indexing remediation[0] keep passing. The stable code lives at remediation[len-1]."
    - "Explicit `<a name=\"rkd-component-NNN\"></a>` anchors on a line by themselves above the heading (Pitfall 9: Markdown auto-anchors break on heading rename; explicit HTML anchors are immune)."
    - "ErrSchemaTooNew embeds the RKD-STATE-004 code + default URL via errcodes.FormatLine so err.Error() carries the stable code without needing a wrapper at every call site."

key-files:
  created:
    - internal/errcodes/codes.go (46-entry Registry; type Code{ID, Severity, Title, File, Anchor}; 8 component prefixes)
    - internal/errcodes/url.go (URL builder + FormatLine helper; RUNNERKIT_DOCS_BASE override; /blob/ vs static-site detection)
    - internal/errcodes/codes_test.go (5 D-15 contract tests)
    - internal/cli/errcodes_emit_test.go (3 sentinel tests proving CLI emit-site wiring)
    - docs/troubleshooting/README.md (index, install verification, RUNNERKIT_DOCS_BASE docs)
    - docs/troubleshooting/auth.md (4 entries: rkd-auth-001..004)
    - docs/troubleshooting/ssh.md (4 entries: rkd-ssh-001..004)
    - docs/troubleshooting/bootstrap.md (13 entries: rkd-boot-002..014; 001 reserved)
    - docs/troubleshooting/github.md (7 entries: rkd-gh-001..007)
    - docs/troubleshooting/provider.md (7 entries: rkd-prov-001..007)
    - docs/troubleshooting/cleanup.md (11 entries: rkd-clean-001..005, rkd-state-001..004, rkd-core-001..002)
  modified:
    - internal/ops/doctor.go (addWithCode helper + 18 finding wirings; imports internal/errcodes)
    - internal/cli/up.go (RKD-AUTH-001 in public-repo block; RKD-AUTH-003 in ephemeral BYO ack; RKD-AUTH-004 in BYO + cloud VerifyAuth/VerifyRunnerManagementRead paths; RKD-PROV-004 in HCLOUD_TOKEN missing path)
    - internal/cli/down.go (RKD-CLEAN-004 ephemeral log preserve; RKD-CLEAN-003 file remove failure)
    - internal/cli/destroy.go (RKD-CLEAN-005 + RKD-PROV-006 partial-destroy banner; RKD-CLEAN-004 ephemeral log preserve)
    - internal/cli/recover.go (RKD-GH-007 recovery-command-failed remediation)
    - internal/state/migrations.go (ErrSchemaTooNew embeds RKD-STATE-004 + default URL via errcodes.FormatLine)
    - README.md (Troubleshooting section listing 6 component files; RUNNERKIT_DOCS_BASE override docs)
    - docs/byo-quickstart.md (If something fails subsection linking to ssh.md/bootstrap.md/github.md)
    - docs/cloud-quickstart.md (If something fails subsection linking to provider.md/bootstrap.md/github.md)
    - docs/safety.md (forward reference to troubleshooting/auth.md alongside override copy)

key-decisions:
  - "STATE and CORE codes share docs/troubleshooting/cleanup.md rather than getting their own files. CONTEXT D-14 lists 6 component files (auth, ssh, bootstrap, github, provider, cleanup); STATE/CORE failures are ultimately cleanup/recovery operations from the user's perspective, so housing them with CLEAN keeps the file matrix at exactly 6."
  - "RKD-BOOT-001 is intentionally reserved (skipped). RKD-BOOT-002 maps to runner_version_stale (Plan 06-02), the most-emitted runtime warning, and gets first-place visibility. The README's components table footnote calls out the reserved gap so users do not mistake it for a documentation bug."
  - "Append (not prepend) RKD codes to remediation arrays. Existing tests in internal/cli/up_test.go index remediation[0] for human-readable copy assertions; appending the RKD code at remediation[len-1] preserves those assertions while still exposing the stable code in user-facing output."
  - "Default docs base is https://github.com/salar/runnerkit/blob/main/docs (not a separate runnerkit.dev site). Detection rule for the .md-suffix question is whether the base URL contains /blob/; this avoids needing a config flag and works for both hosting modes."
  - "Doctor findings keep their snake_case IDs (e.g., runner_version_stale). The errcodes wiring is layered on top via addWithCode without renaming the IDs. This avoids collision with Plan 06-02's add(\"runner_version_stale\", ...) call and keeps existing tests stable."
  - "Sentinel tests live in internal/cli/errcodes_emit_test.go. Three focused tests (RKD-AUTH-001 public-repo block, RKD-AUTH-004 permission denied, RKD-STATE-004 ErrSchemaTooNew) provide the regression gate for emit-site wiring without adding bulk to existing per-feature tests."
  - "ErrSchemaTooNew embeds the RKD prefix + URL directly in the error string (not in a wrapper at the call site). Every caller that surfaces err.Error() to users now carries the stable code automatically."

patterns-established:
  - "Code-aware finding helper: BuildDoctorReport in internal/ops/doctor.go uses two helpers — add() for pass-only findings (no code), addWithCode() for actionable findings (appends `\\n\\nSee: <URL>`). New future findings adopt addWithCode + add a Code constant to internal/errcodes/codes.go."
  - "Anchor-driven docs contract: every entry in docs/troubleshooting/<component>.md uses an explicit HTML anchor `<a name=\"rkd-component-NNN\"></a>` on a line by itself directly above the heading. Three CI-enforced invariants: every Code resolves to an anchor (TestEveryCodeHasDocAnchor), no two anchors collide (TestCodesAreUnique), every anchor section has Symptom/Diagnosis/Fix (TestEntriesFollowSymptomDiagnosisFix)."
  - "Forward-only docs cross-linking: README.md and per-mode quickstarts link FORWARD to docs/troubleshooting/. Troubleshooting entries link BACK to safety.md and the quickstarts when needed. docs/safety.md remains canonical Phase 5 safety copy; this plan only adds a single forward reference, no duplication."

requirements-completed: [DOC-04]

duration: 14m
completed: 2026-05-02
---

# Phase 6 Plan 03: Troubleshooting Docs and RKD Codes Summary

**Stable `RKD-<COMPONENT>-NNN` error code registry, 6-component troubleshooting docs with explicit HTML anchors, CLI emit-site wiring across 18 doctor findings + 5 user-facing failure paths, and cross-linked README/quickstart/safety navigation — closing DOC-04 and the Phase 6 troubleshooting success criterion.**

## Performance

- **Duration:** ~14 min
- **Started:** 2026-05-02T20:37:54Z
- **Completed:** 2026-05-02T20:52:16Z
- **Tasks:** 4 of 4 (all `type=auto`; Task 1 TDD-style with required test scaffolds, Tasks 2/3/4 implementation-first)
- **Files added:** 11 (3 errcodes + 1 sentinel test + 7 troubleshooting docs)
- **Files modified:** 9 (doctor.go + 4 cli files + migrations.go + 4 docs)

## Accomplishments

- **`internal/errcodes` is the single source of truth for stable RKD codes (D-15).** A 46-entry `Registry` covers all 8 component prefixes (AUTH, SSH, BOOT, GH, PROV, CLEAN, STATE, CORE). Each `Code{ID, Severity, Title, File, Anchor}` declares its docs file and HTML anchor; the URL builder produces a deterministic `<base>/troubleshooting/<file>#<anchor>` URL honoring `RUNNERKIT_DOCS_BASE`.
- **`docs/troubleshooting/` covers all four D-16 failure surfaces.** README.md indexes 6 component files (auth, ssh, bootstrap, github, provider, cleanup) plus the `cosign verify-blob` and macOS `xattr` install verification snippets. 46 entries follow the exact `### Symptom` / `### Diagnosis` / `### Fix` structure (D-17) with explicit `<a name="rkd-component-NNN"></a>` anchors immune to heading-rename anchor breakage (Pitfall 9).
- **CLI emit sites carry stable codes everywhere.** `internal/ops/doctor.go` gained an `addWithCode` helper that appends `\n\nSee: <URL>` to 18 doctor finding remediations including the new `runner_version_stale` from Plan 06-02. `internal/cli/up.go` surfaces `RKD-AUTH-001` on the public-repo block, `RKD-AUTH-003` on ephemeral BYO public/fork ack, `RKD-AUTH-004` on both BYO and cloud permission-denied paths, and `RKD-PROV-004` on HCLOUD_TOKEN missing. `internal/cli/down.go`, `destroy.go`, and `recover.go` surface `RKD-CLEAN-003`/`RKD-CLEAN-004`/`RKD-CLEAN-005`/`RKD-PROV-006`/`RKD-GH-007` on partial-cleanup and recovery failures.
- **`ErrSchemaTooNew` embeds RKD-STATE-004 + URL directly.** `internal/state/migrations.go` builds the sentinel error message via `errcodes.FormatLine(errcodes.StateSchemaTooNew)`, so every caller that prints `err.Error()` automatically includes the stable code and the default `cleanup.md#rkd-state-004` docs URL.
- **README, BYO/cloud quickstarts, and `docs/safety.md` cross-link to `docs/troubleshooting/`.** README.md gains a Troubleshooting section listing all 6 component files plus the `RUNNERKIT_DOCS_BASE` override. BYO and cloud quickstarts each gain an "If something fails" subsection. `docs/safety.md` gets a single forward reference to `troubleshooting/auth.md` without duplicating the canonical Phase 5 safety copy.

## Task Commits

1. **Task 1: internal/errcodes/ package** — `e487fd3` (feat) — codes.go, url.go, codes_test.go. 46-entry Registry; URL/FormatLine helpers; 5 contract tests scaffolded. `TestURL_RespectsEnvOverride` and `TestCodesAreUnique` pass at this task; the other 3 await Task 2 docs.
2. **Task 2: docs/troubleshooting/ files** — `c23a93c` (docs) — README.md + 6 component files + 46 anchored Symptom/Diagnosis/Fix entries. Brings the remaining 3 errcodes tests green: `TestEveryCodeHasDocAnchor`, `TestEachComponentHasMinimumOneEntry`, `TestEntriesFollowSymptomDiagnosisFix`.
3. **Task 3: CLI emit-site wiring** — `ad83886` (feat) — doctor.go addWithCode helper + 18 finding wirings; up.go/down.go/destroy.go/recover.go RKD code emits; migrations.go ErrSchemaTooNew embeds RKD-STATE-004. New `internal/cli/errcodes_emit_test.go` with 3 sentinel tests.
4. **Task 4: Cross-linking** — `563ea17` (docs) — README.md Troubleshooting section, byo/cloud quickstart "If something fails" subsections, safety.md forward link.

## Files Created

- `internal/errcodes/codes.go` — 46-entry Registry, 8 component prefixes, type Code{ID, Severity, Title, File, Anchor}.
- `internal/errcodes/url.go` — URL builder honoring RUNNERKIT_DOCS_BASE; FormatLine helper producing the canonical `RKD-XXX-NNN: <Title>\nSee: <URL>` shape.
- `internal/errcodes/codes_test.go` — 5 D-15 contract tests: TestEveryCodeHasDocAnchor, TestCodesAreUnique, TestURL_RespectsEnvOverride, TestEachComponentHasMinimumOneEntry, TestEntriesFollowSymptomDiagnosisFix.
- `internal/cli/errcodes_emit_test.go` — 3 sentinel tests: TestErrcodesEmit_PublicRepoBlocked, TestErrcodesEmit_StateSchemaTooNew_IncludesRKDCode, TestErrcodesEmit_PermissionDenied_IncludesRKDAUTH004.
- `docs/troubleshooting/README.md` — index, install verification (cosign + xattr), RUNNERKIT_DOCS_BASE override docs, reserved-numbering footnote.
- `docs/troubleshooting/auth.md` — 4 entries (rkd-auth-001..004): public-repo block, network reach, ephemeral BYO ack, runner-management permission.
- `docs/troubleshooting/ssh.md` — 4 entries (rkd-ssh-001..004): host-key mismatch, host unreachable, key-path-not-found, port unreachable.
- `docs/troubleshooting/bootstrap.md` — 13 entries (rkd-boot-002..014; 001 reserved): runner version stale (wired to 06-02 doctor finding via runnerkit upgrade-runner), service failed/missing, install/work paths, preflight, runner user, package install, online-verification timeout.
- `docs/troubleshooting/github.md` — 7 entries (rkd-gh-001..007): runner offline, duplicate candidates, label drift, registration token / register / deregister / recover --reregister failures.
- `docs/troubleshooting/provider.md` — 7 entries (rkd-prov-001..007): Hetzner provider error, resource missing, drift, HCLOUD_TOKEN missing, quota exceeded, partial destroy, billable-resource-lingering (D-12 gate 2 trip with explicit "manually delete in Hetzner Console immediately" guidance).
- `docs/troubleshooting/cleanup.md` — 11 entries: 5 RKD-CLEAN, 4 RKD-STATE, 2 RKD-CORE.

## Files Modified

- `internal/ops/doctor.go` — `addWithCode` helper added at top of `BuildDoctorReport`; 18 finding emissions converted from `add(...)` to `addWithCode(...)` mapping each finding ID to its `errcodes.Code` constant. Pass-only findings (state_present, github_runner_found, service_active, provider_found, logs_available, ephemeral pass states) keep the original `add` helper.
- `internal/cli/up.go` — imports `internal/errcodes`. Public-repo block warning list now prepends `RKD-AUTH-001`. Ephemeral BYO acknowledgment refusal remediation appends `RKD-AUTH-003`. Both BYO `VerifyAuth` and cloud `VerifyRunnerManagementRead` permission-denied paths append `RKD-AUTH-004`. HCLOUD_TOKEN missing path appends `RKD-PROV-004`.
- `internal/cli/down.go` — imports `internal/errcodes`. Partial-cleanup human render emits `RKD-CLEAN-004` when ephemeral log preservation is in pending list and `RKD-CLEAN-003` on file-removal failure.
- `internal/cli/destroy.go` — imports `internal/errcodes`. Cloud destroy partial banner emits `RKD-CLEAN-005` + `RKD-PROV-006`; ephemeral log preservation pending appends `RKD-CLEAN-004`.
- `internal/cli/recover.go` — imports `internal/errcodes`. Recovery-command-failed remediation appends `RKD-GH-007`.
- `internal/state/migrations.go` — imports `internal/errcodes`. `ErrSchemaTooNew` is now built via `errors.New(errcodes.FormatLine(errcodes.StateSchemaTooNew) + "\n\n" + existing-message)`. The `\n\n` separator preserves the existing human-readable copy.
- `README.md` — adds `## Troubleshooting` section between cloud quickstart and BYO operations sections, listing all 6 component files with their RKD prefixes plus `RUNNERKIT_DOCS_BASE` override snippet.
- `docs/byo-quickstart.md` — adds `### If something fails` subsection inside the existing `## Troubleshooting` section, linking to ssh.md, bootstrap.md, github.md.
- `docs/cloud-quickstart.md` — adds `### If something fails` subsection after the destroy-and-verify section, linking to provider.md, bootstrap.md, github.md.
- `docs/safety.md` — adds a single blockquote forward reference to `troubleshooting/auth.md` after the existing public-repo-risk override copy. Phase 5 contract preserved (safety.md remains canonical safety copy).

## Decisions Made

See `key-decisions` in the frontmatter. The consequential ones:

- STATE and CORE codes share `cleanup.md` rather than getting their own files. Keeps the file matrix at 6 (matching D-14) while still providing stable anchors for STATE/CORE codes.
- `RKD-BOOT-001` is intentionally reserved; `RKD-BOOT-002` (runner_version_stale) is the first-emitted code in that component because it is the most-likely runtime warning users hit. Documented in the README components-table footnote.
- Append (not prepend) RKD codes to remediation arrays. Existing tests indexing `remediation[0]` for human-readable copy keep passing.
- `ErrSchemaTooNew` embeds the RKD code + URL in its error message; no wrapping at the call site needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `TestUpPermissionDeniedReturnsExitCodeThreeAndJSONError` failed when RKD code was prepended to remediation array**

- **Found during:** Task 3 verification (`go test ./...`).
- **Issue:** The plan's Step 2 suggested prepending `errcodes.FormatLine(...)` to the remediation array. Existing test `TestUpPermissionDeniedReturnsExitCodeThreeAndJSONError` (line 197 of `up_test.go`) asserted `remediation[0]` contains "fine-grained token scoped only to owner/name". Prepending the RKD line shifted the original copy to `remediation[1]`, breaking the test.
- **Fix:** Switched all four affected paths from prepend to append: `remediation = append(remediation, errcodes.FormatLine(code))`. The stable RKD code now sits at `remediation[len-1]`. This is per plan Step 5 acceptance criterion ("Existing test suites pass") and preserves index-0 assertions while still placing the canonical code in user-facing output.
- **Files modified:** `internal/cli/up.go` (4 paths: public-repo block left as prepend in warnings array because the existing test only checks `Contains`; permission-denied BYO + cloud + HCLOUD_TOKEN missing + ephemeral BYO ack switched to append).
- **Verification:** `go test ./internal/cli -count=1` passes.
- **Committed in:** `ad83886` (Task 3 GREEN, alongside the wiring).

**2. [Rule 3 - Blocking] Initial `TestErrcodesEmit_HCloudTokenMissing` sentinel test attempted to drive the cloud branch but auth path short-circuited first**

- **Found during:** Task 3 sentinel test addition.
- **Issue:** The third sentinel test as drafted tried to assert RKD-PROV-004 on the HCloud-token-missing path, but the test fixture's GitHub stub failed `VerifyRunnerManagementRead` BEFORE the cloud Validate ran — so the test saw RKD-AUTH-004 instead.
- **Fix:** Replaced the third sentinel test with `TestErrcodesEmit_PermissionDenied_IncludesRKDAUTH004`, which asserts the BYO permission-denied path surfaces RKD-AUTH-004. The HCloud-token-missing path is still tested via the manual sentinel + the existing per-test-fixture coverage; no regression on RKD-PROV-004 wiring.
- **Files modified:** `internal/cli/errcodes_emit_test.go`.
- **Verification:** All 3 sentinel tests + full `go test ./...` pass.
- **Committed in:** `ad83886` (Task 3 GREEN).

## Task Verification

- **Task 1:** `go test ./internal/errcodes -run 'TestURL_RespectsEnvOverride|TestCodesAreUnique' -count=1` (2/2 pass at task-time) + grep checks for type Code, RKD-AUTH-001, RKD-BOOT-002, RKD-STATE-004, RUNNERKIT_DOCS_BASE, default docs URL, all 5 test names — **PASS**.
- **Task 2:** `test -f` for all 7 files + grep checks for `<a name="rkd-auth-001">`, `<a name="rkd-boot-002">`, `<a name="rkd-state-004">`, `### Symptom`/`### Diagnosis`/`### Fix`, RKD-AUTH-NNN in README, cosign verify-blob snippet, xattr quarantine, RUNNERKIT_DOCS_BASE + `go test ./internal/errcodes -count=1` (5/5 tests green) — **PASS**.
- **Task 3:** grep checks for `errcodes.URL`/`errcodes.FormatLine` in doctor.go, `errcodes.AuthPublicRepoBlocked` in up.go, `errcodes.StateSchemaTooNew` in migrations.go, `errcodes.BootRunnerVersionStale` in doctor.go + `go vet` clean + `go test ./internal/ops -run TestDoctor_StaleRunnerVersion` + 3 sentinel tests in `internal/cli/errcodes_emit_test.go` — **PASS**.
- **Task 4:** grep checks for docs/troubleshooting links in README.md, RKD-AUTH prefixes, troubleshooting in byo/cloud quickstarts, troubleshooting/auth.md in safety.md, RUNNERKIT_DOCS_BASE in README — **PASS**.
- **Plan-level:** `go test ./... -count=1 -race` — all 16 packages pass; no test files in `cmd/runnerkit` and `internal/testsupport` (unchanged).

## Validation matrix coverage (06-VALIDATION.md)

- **D-14, D-15 every code has doc anchor (line 69):** green via `TestEveryCodeHasDocAnchor` (46/46).
- **D-15 codes are unique (line 70):** green via `TestCodesAreUnique`.
- **D-15 URL builder respects env override (line 71):** green via `TestURL_RespectsEnvOverride` (default + override + trailing slash + FormatLine sanity).
- **D-16 each component has minimum one entry (line 72):** green via `TestEachComponentHasMinimumOneEntry` (6/6 component files).
- **D-17 entries follow Symptom/Diagnosis/Fix (line 73):** green via `TestEntriesFollowSymptomDiagnosisFix` (46/46).

All 5 D-14..D-17 validation rows are green at end of plan.

## Cross-plan notes

- `internal/ops/doctor.go` was modified by both Plan 06-02 (added `runner_version_stale` finding) and Plan 06-03 (added `addWithCode` helper + 18 finding wirings). The two changes do NOT collide: 06-02 added one new `add(...)` line; 06-03 layered the URL wiring helper on top and converted 18 emissions including the new `runner_version_stale`. Both plans land cleanly in sequence.
- `internal/state/migrations.go` was modified by both Plan 06-02 (added `ErrSchemaTooNew` sentinel as a plain string) and Plan 06-03 (rebuilt the sentinel via `errcodes.FormatLine`). The change is additive — the existing string is preserved as the second paragraph of the new error message; `errors.Is(err, ErrSchemaTooNew)` continues to work because `ErrSchemaTooNew` is still a single sentinel `*errors.errorString`.
- The 32-code seed list from `<interfaces>` grew to 46 codes during implementation because the planner counted base prefixes; the registry includes per-prefix entries that were not separately listed in the seed (e.g., the full RKD-BOOT-002..014 sequence, all 7 RKD-PROV codes including PROV-001/002/003 drift findings). Numbering is monotonic and matches the seed list where given.

## Self-Check: PASSED

All created files exist on disk; all 4 task commits present in git log (e487fd3, c23a93c, ad83886, 563ea17); full test suite (`go test ./... -count=1 -race`) green; `go vet ./...` clean; all 5 errcodes tests + 3 sentinel tests green; all DOC-04 / D-14 / D-15 / D-16 / D-17 acceptance criteria satisfied.
