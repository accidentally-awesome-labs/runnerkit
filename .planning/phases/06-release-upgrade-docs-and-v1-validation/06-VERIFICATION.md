---
phase: 06-release-upgrade-docs-and-v1-validation
type: verification
status: pending
created: 2026-05-04
---

# Phase 06 — v1.0.0 Verification Baseline

> Source-of-truth for the v1.0.0 ten-minute reliable-runner promise. Filled in
> by the maintainer running `make smoke-live` before tagging v1.0.0.
> See `docs/release-process.md` Stopwatch Checklist (D-13).

## Test Suite (automated)

- [ ] `go test ./... -count=1 -race` green
- [ ] `goreleaser check` green
- [ ] `goreleaser release --snapshot --skip=publish --clean` green; `dist/` contains 4 platform tarballs + checksums.txt
- [ ] All 5 errcodes tests green (TestEveryCodeHasDocAnchor, TestCodesAreUnique, TestURL_RespectsEnvOverride, TestEachComponentHasMinimumOneEntry, TestEntriesFollowSymptomDiagnosisFix)
- [ ] All 4 state migration tests green (TestMigrate_V1ToV2_ForwardOnly, TestMigrate_WritesBackupBeforeMutation, TestMigrate_RefusesNewerSchema, TestMigrate_Atomic)
- [ ] All 6 update-check tests green (TestMaybePrint_*)
- [ ] All 7 upgrade/upgrade-runner/doctor-stale tests green
- [ ] All 3 _smokebin unit tests green (TestEmptyPrecheck_RefusesOnExisting, TestEmptyPrecheck_AllResourceTypes, TestDestroyVerify_Timeout)

## Live Smoke (manual, maintainer-only)

### BYO smoke (closes Phase 1 outstanding)

- [ ] `make smoke-live-byo` succeeds end-to-end
- BYO host: `user@________`
- Repo (maintainer-controlled, trusted): `________/________`
- Wall-clock duration: `____ seconds`
- Runner ID assigned by GitHub: `____`

### Hetzner smoke (closes Phase 4 outstanding)

- [ ] `make smoke-live-cloud` succeeds end-to-end
- Repo (maintainer-controlled, trusted, NOT public): `________/________`
- Hetzner project: `________`
- Wall-clock duration (up → workflow → destroy): `____ seconds`
- Hetzner cost (from project dashboard, EUR): `__.__`
- Resource IDs created (server / ssh-key / primary-ip(v4) / primary-ip(v6) / firewall): `____ / ____ / ____ / ____ / ____`
- D-12 gate 1 (empty-project precheck) status: `____` (PASS / FAIL)
- D-12 gate 2 (destroy-verify 404 within timeout) status: `____` (PASS / FAIL)
- Empty-project precheck final ID list size: `0` (must be exactly zero on a successful smoke)

### 10-minute stopwatch (D-13)

- [ ] BYO total: `____ minutes ____ seconds` (target ≤ 10 minutes)
- [ ] Hetzner total: `____ minutes ____ seconds` (target ≤ 10 minutes)

## Bundled Versions

- runner pin (`internal/bootstrap/package.go::RunnerVersion`): `2.334.0`
- GoReleaser CI version: `v2.15.4`
- cosign CI version: `v3.0.6`
- hcloud-go (Phase 4 pinned): `v1.59.2`

## Sign-Off

- [ ] Maintainer signature: `____________ (date)`
- [ ] All gates green; ready to push `git tag -a v1.0.0`
