---
phase: 04-recommended-cloud-path-and-billable-cleanup
plan: "01"
subsystem: cloud-provider-foundation
tags: [cloud, hetzner, provider-interface, cli, provisioning-plan]
requires:
  - phase: 03-operations-diagnostics-and-byo-cleanup
    provides: BYO runner lifecycle, GitHub runner operations, state safety, redaction, and CLI renderer patterns
provides:
  - Provider interface and fake provider contracts for Phase 4 cloud lifecycle work
  - Hetzner default profile, cost caveat, resource naming, ownership tags, and env-only credential discovery
  - Explicit runnerkit up cloud intent flags and setup-path selection between BYO and Hetzner cloud
  - Cloud provisioning plan human/JSON output with cost, resources, tags, labels, and future destroy command
  - Non-token GitHub runner-management read check before cloud provisioning plans
  - Redaction fix preserving placeholder credential setup copy while masking real provider tokens
affects: [phase-04-02, phase-04-03, phase-04-04, cloud, provider, cleanup]
tech-stack:
  added: []
  patterns: [provider registry, plan-before-mutation, non-token cloud preflight, CLI JSON parity]
key-files:
  created:
    - internal/provider/provider.go
    - internal/provider/profile.go
    - internal/provider/profile_test.go
    - internal/provider/hetzner/credentials.go
    - internal/provider/hetzner/credentials_test.go
    - internal/provider/fake.go
    - internal/cli/up_cloud_test.go
  modified:
    - internal/cli/root.go
    - internal/cli/root_test.go
    - internal/cli/up.go
    - internal/cli/up_byo_test.go
    - internal/cli/up_test.go
    - internal/github/service.go
    - internal/testsupport/github.go
    - internal/redact/redact.go
    - internal/redact/redact_test.go
key-decisions:
  - "Hetzner is the only registered Phase 4 cloud provider path for now, behind an injectable provider registry."
  - "Cloud setup requires explicit --cloud hetzner intent plus --yes for non-interactive billable-resource creation; missing --host plus --yes is not enough."
  - "Cloud pre-provisioning uses a non-token runner-management read check and does not mint GitHub registration tokens before future cloud readiness/preflight gates."
patterns-established:
  - "Provider contracts return normalized plans, machines, status, destroy, and verification results while CLI owns GitHub/bootstrap orchestration."
  - "Cloud plan output uses the existing ui.Renderer primitives and mirrors human-visible facts into JSON cloud_plan fields."
requirements-completed: [MACH-03, CLEAN-01]
duration: 9 min
completed: 2026-05-01
---

# Phase 04 Plan 01: Provider Interface, Cloud Intent, and Provisioning Plan Summary

**Hetzner cloud-provider planning foundation with explicit `runnerkit up --cloud hetzner` intent, non-token GitHub read checks, and cost-aware human/JSON provisioning plans**

## Performance

- **Duration:** 9 min
- **Started:** 2026-04-30T23:52:17Z
- **Completed:** 2026-05-01T00:01:42Z
- **Tasks:** 3
- **Files modified:** 16

## Accomplishments

- Added `internal/provider` contracts, Hetzner profile defaults, env-only credential discovery, resource naming/tags, and fake provider support.
- Wired `runnerkit up` with explicit cloud flags, provider registry injection, interactive BYO-vs-cloud selection, and safe missing-intent behavior.
- Rendered cloud provisioning plans in human and JSON modes with approximate cost caveat, resources, tags, labels, workflow snippet, and future `runnerkit destroy` guidance.
- Added a non-token GitHub runner-management read check so cloud planning does not mint registration tokens before future readiness/preflight gates.

## Task Commits

1. **Task 04-01-01: Add provider contracts, Hetzner defaults, credential discovery, and fakes** - `75509b9` (feat)
2. **Task 04-01-02: Add cloud intent flags and setup-path selection without mutating cloud resources** - `5c81a34` (feat)
3. **Task 04-01-03: Render cloud provisioning plans with cost caveats, resource identity, JSON parity, and future destroy copy** - `689fb2b` (feat)

## Files Created/Modified

- `internal/provider/provider.go` - Provider interface, registry, provision plan/machine/status/destroy/verification structs, and `ProvisionError`.
- `internal/provider/profile.go` - Hetzner default profile, cost caveat, resource names, tags, and plan-only provider implementation.
- `internal/provider/hetzner/credentials.go` - `HCLOUD_TOKEN` / `HETZNER_CLOUD_TOKEN` discovery with setup remediation and no JSON token persistence.
- `internal/provider/fake.go` - Configurable fake provider and call counters for CLI tests.
- `internal/cli/root.go` / `internal/cli/up.go` - Provider registry injection, cloud flags, setup-path selection, non-token read check, plan rendering, and confirmation gate.
- `internal/cli/up_cloud_test.go` - Cloud intent safety, dry-run, JSON parity, confirmation, credential, and no-token-call coverage.
- `internal/github/service.go` / `internal/testsupport/github.go` - Non-mutating runner-management read check and fake/testsupport coverage.
- `internal/redact/redact.go` / `internal/redact/redact_test.go` - Provider token redaction refined to preserve placeholder setup copy.

## Decisions Made

- Hetzner remains the only selectable provider for Phase 4 Plan 01, registered through an injectable provider registry.
- Non-interactive billable cloud setup must include explicit `--cloud hetzner` and `--yes`; RunnerKit will not infer cloud creation from a missing BYO host.
- Cloud pre-provisioning uses `VerifyRunnerManagementRead` rather than `VerifyAuth`, preserving the just-in-time registration-token timing contract.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Preserve placeholder provider-token setup copy**
- **Found during:** Task 04-01-03 (cloud plan and credential output tests)
- **Issue:** The existing provider credential redaction regex masked the required remediation placeholder `HCLOUD_TOKEN=<token from Hetzner Cloud Console>`, making setup guidance unreadable.
- **Fix:** Narrowed provider env-var pattern redaction so real `HCLOUD_TOKEN=secret` values are still redacted while placeholder documentation remains visible.
- **Files modified:** `internal/redact/redact.go`, `internal/redact/redact_test.go`
- **Verification:** `go test ./internal/redact`, `go test ./...`
- **Committed in:** `689fb2b` (Task 04-01-03)

---

**Total deviations:** 1 auto-fixed (1 bug).
**Impact on plan:** Required UI copy is now preserved without weakening real provider credential redaction. No scope expansion beyond planned cloud output/redaction safety.

## Issues Encountered

None beyond the auto-fixed redaction bug documented above.

## User Setup Required

None - no USER-SETUP.md was generated. Hetzner token setup is documented in command remediation only; live cloud provisioning is deferred to later Phase 4 plans.

## Next Phase Readiness

Plan 04-02 can implement Hetzner VM/SSH-key/firewall creation behind the `internal/provider.Provider` contract, reuse the cloud plan input/profile/tags, persist pending checkpoints after billable resource creation, and continue to keep GitHub registration-token minting after cloud readiness and BYO preflight.

## Self-Check: PASSED

- Key files exist: `internal/provider/provider.go`, `internal/provider/profile.go`, `internal/provider/hetzner/credentials.go`, `internal/provider/fake.go`, `internal/cli/up_cloud_test.go`.
- Task commits are present for `04-01`: `75509b9`, `5c81a34`, `689fb2b`.
- Focused and full verification commands passed, including `go test ./...`.

---

_Phase: 04-recommended-cloud-path-and-billable-cleanup_
_Completed: 2026-05-01_
