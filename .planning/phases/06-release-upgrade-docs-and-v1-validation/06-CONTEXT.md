# Phase 6: Release, Upgrade, Docs, and v1 Validation - Context

**Gathered:** 2026-05-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 6 makes RunnerKit shippable to early external users. It delivers official multi-platform release distribution, a documented upgrade path that prevents pinned-runner-version rot, cleanup/troubleshooting documentation tied to CLI diagnostics, and a live end-to-end validation that proves the 10-minute reliable-runner promise across BYO and the recommended Hetzner cloud path.

This phase stays inside packaging/distribution, the upgrade lifecycle, troubleshooting documentation, and v1 validation. It does NOT add a hosted control plane, autoscaling, organization-level runner management, broader provider matrix, Windows runner-host support, telemetry, automatic `doctor --fix`, or any new runtime capability beyond what already exists at end of Phase 5.

</domain>

<decisions>
## Implementation Decisions

### Distribution and install (Plan 06-01 surface)

- **D-01:** Ship through two install channels: official GitHub Releases binaries (always) plus a Homebrew tap (`brew install runnerkit`). Do not provide `go install` or native `.deb`/`.rpm` packages in v1 — they add packaging surface without serving the solo-developer-on-macOS/Linux audience better than the two chosen channels.
- **D-02:** Supported CLI host platforms for v1 are: macOS arm64, macOS amd64, Linux amd64, Linux arm64. Windows, Linux 386, and 32-bit ARM are explicitly out of scope. The runner host stays Linux/systemd as established in Phase 2.
- **D-03:** Releases are produced by GoReleaser running in GitHub Actions, triggered by pushing a `vX.Y.Z` tag. One GoReleaser config produces all platform binaries, the checksums file, the cosign signature, the GitHub Release notes, and the Homebrew formula update in a single tag-driven run. No local manual cuts and no hand-rolled per-platform CI matrix.
- **D-04:** Each release ships, at minimum, the per-platform binaries, a `runnerkit_vX.Y.Z_checksums.txt` SHA256 file, and a cosign keyless signature signed via the GitHub Actions OIDC token. GPG signing and SBOM generation are out of scope for v1; revisit if downstream demand appears.
- **D-05:** Install instructions in README and the troubleshooting docs must show the verification step for both checksums and `cosign verify-blob` so users can prove integrity before installing.

### Upgrade detection and execution (Plan 06-02 surface)

- **D-06:** The CLI performs a lazy update check on user-relevant commands (`runnerkit up`, `runnerkit status`, `runnerkit doctor`) using a HEAD against the GitHub Releases API or equivalent. The check is cached for ~24 hours, skipped when the network is unreachable, skipped when the active output mode is JSON, and surfaced as a single non-blocking notice line. No always-on per-invocation network call.
- **D-07:** `runnerkit upgrade` does not self-replace the binary. It detects the install channel (Homebrew vs GitHub Releases binary) and prints exact upgrade instructions for that channel (`brew upgrade runnerkit` or the release-asset download flow). This avoids self-update logic, signature re-verification on update, and rollback complexity.
- **D-08:** The GitHub Actions runner version (currently 2.334.0) is pinned per RunnerKit release. Each RunnerKit release pins a known-good runner version in bootstrap. `runnerkit doctor` warns when the installed runner is stale relative to the bundled pin or when GitHub starts rejecting the installed version. `runnerkit upgrade-runner` re-runs the bootstrap path against the host to roll the runner forward to the bundled pin without re-running full setup.
- **D-09:** State migrations are forward-only and automatic for older `schema_version` values, with a side-by-side backup of the original state file before migration runs (atomic-write semantics from Phase 1). When state is read with a newer `schema_version` than the running CLI knows about, RunnerKit refuses to mutate, prints exact "upgrade RunnerKit to read this state" guidance, and exits non-zero. There is no interactive migration prompt.

### Live v1 validation (Plan 06-04 surface)

- **D-10:** Plan 06-04 finally lands the two outstanding live smoke validations (Phase 1 GitHub permission smoke, Phase 4 Hetzner billable-resource smoke) AND a clean-machine 10-minute stopwatch checklist that exercises the BYO and Hetzner happy paths. All three are required for v1 sign-off.
- **D-11:** Live smokes are triggered manually by the maintainer via a `make smoke-live` target (or equivalent documented procedure) before tagging a release. They are NOT wired into a scheduled or automatic CI workflow, do NOT run on `vX.Y.Z` tag push, and do NOT require GitHub Actions to hold real Hetzner / GitHub PAT secrets.
- **D-12:** The Hetzner live smoke must include two billable-risk gates: (1) a hard pre-check that the configured Hetzner project is empty of any pre-existing `runnerkit-*` managed servers, volumes, or SSH keys, refusing to run if it is not, and (2) a deferred destroy verification with timeout — after the smoke completes, `runnerkit destroy --yes` runs and the smoke asserts every created resource ID returns 404 from the provider within N minutes, failing loudly if any resource lingers. Phase 4's destroy semantics back this up; Phase 6 turns it into an executable contract.
- **D-13:** Validation results live in `06-VERIFICATION.md` (the v1.0.0 baseline: timestamps, durations, runner IDs, Hetzner cost) AND in a `RELEASE-NOTES-vX.Y.Z.md` file shipped per release, which re-runs the same checklist and records measured durations and the runner version pinned in that release. This makes the 10-minute claim auditable per release.

### Troubleshooting docs and CLI integration (Plan 06-03 surface, also DOC-04)

- **D-14:** Troubleshooting documentation is split across `docs/troubleshooting/` per component: `auth.md`, `ssh.md`, `bootstrap.md`, `github.md`, `provider.md`, `cleanup.md`, plus a `docs/troubleshooting/README.md` index. The structure mirrors the internal package layout users already see in CLI error output.
- **D-15:** Each `runnerkit doctor` finding and each user-facing failure has a stable error code of the form `RKD-<COMPONENT>-NNN` (e.g., `RKD-AUTH-001`, `RKD-SSH-003`). The CLI output prints `See: <docs URL>/troubleshooting/<component>#rkd-<component>-NNN` for every emitted code. Doc anchors must be stable across renames.
- **D-16:** The v1 docs must cover all four failure surfaces from prior phases: setup (auth scope, public-repo block, SSH host-key, preflight), bootstrap and service (systemd unit, runner user, package install, online-verification timeout), operations (status drift, doctor findings, recover, down), and cloud and cleanup (Hetzner quota/credentials, partial destroy, billable-resource verification). Nothing is deferred.
- **D-17:** Each troubleshooting entry follows a `Symptom → Diagnosis → Fix` structure with copyable commands wherever a fix is executable. Entries are short, structured, and optimized for a user who is currently stuck. No long-form prose tutorials, no flat FAQ list.

### Claude's Discretion

- Exact file names, GoReleaser config layout, and Homebrew tap repo structure, provided the published artifact set matches D-01..D-05.
- Exact CLI flag/command spelling for `runnerkit upgrade` and `runnerkit upgrade-runner`, provided behavior matches D-07/D-08.
- Exact wording, polling cadence (within "lazy / ~24h cached"), and cache file location for the update notice, provided D-06 holds.
- Exact RKD error-code numbering scheme per component, provided codes are stable, prefixed `RKD-`, and URL-anchorable per D-15.
- Exact `make smoke-live` script implementation, environment variable names, and host-side scratch directory, provided the gates in D-12 are enforced and the deferred destroy verification fails loudly on lingering resources.
- Documentation hosting choice (GitHub repo only vs `runnerkit.dev` static site) — pick whichever is cheapest to maintain, provided D-15's stable anchor URLs work in either world.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements

- `.planning/ROADMAP.md` §"Phase 6: Release, Upgrade, Docs, and v1 Validation" — Fixed phase goal, four success criteria, and the four planned plans (06-01 release packaging/checksums/install/smoke, 06-02 upgrade workflow/state migrations/rollback, 06-03 troubleshooting/cleanup/recovery docs, 06-04 end-to-end v1 validation).
- `.planning/REQUIREMENTS.md` §"Reliability and Operations" — `REL-05` (upgrade path that prevents immediate runner rot) is the binding requirement for Plans 06-01/06-02.
- `.planning/REQUIREMENTS.md` §"Documentation and Safety" — `DOC-04` (cleanup and troubleshooting guidance for common failure modes) is the binding requirement for Plan 06-03.
- `.planning/REQUIREMENTS.md` §"v2 Requirements" — `COST-01..03` are explicitly v2; release docs may reference cost guidance for Hetzner but must not implement cost-control features.
- `.planning/REQUIREMENTS.md` §"Out of Scope" — Hosted dashboard, multi-CI, Kubernetes/ARC, autoscaling fleet manager, organization-level management, automatic workflow-YAML edits, and `doctor --fix` are all excluded from v1 and must not creep into Phase 6.
- `.planning/PROJECT.md` §"What This Is", §"Constraints", §"Out of Scope", and §"Key Decisions" — Solo-developer focus, CLI-only, GitHub-Actions-only, ~10-min setup target, persistent default for trusted private, ephemeral explicit (Phase 5), `runnerkit down` for BYO + `runnerkit destroy` for cloud, GoReleaser-style release expectation implicit in shippable v1.
- `.planning/STATE.md` §"Accumulated Context" §"Blockers/Concerns" — Open notes that the live GitHub permission smoke (Phase 1/2) and live Hetzner smoke (Phase 4) are recommended before public release. Phase 6 explicitly closes both.

### Prior phase context that constrains Phase 6 implementation

- `.planning/phases/01-cli-auth-state-and-safety-foundation/01-CONTEXT.md` — Versioned non-secret JSON state with atomic writes, `SchemaVersion` already at `"1"`, redaction policy, and CLI/JSON output conventions that release docs and upgrade flow must continue to honor.
- `.planning/phases/02-byo-persistent-runner-happy-path/02-CONTEXT.md` — Pinned runner version `2.334.0`, non-root `runnerkit-runner` user, systemd service shape, and BYO quickstart pattern that the upgrade-runner flow rebuilds against.
- `.planning/phases/03-operations-diagnostics-and-byo-cleanup/03-CONTEXT.md` — `runnerkit doctor` finding/recovery model, `runnerkit down` cleanup boundaries, and the "stable error code" precedent that Phase 6 formalizes as `RKD-*`.
- `.planning/phases/04-recommended-cloud-path-and-billable-cleanup/04-CONTEXT.md` — Hetzner provider details, billable destroy semantics, `ProviderRef` inventory in state, and the cost-caveat language that the live-smoke gates and release notes must echo.
- Phase 5 plans (`.planning/phases/05-scoped-ephemeral-mode-and-safety-profiles/05-01..05-03-SUMMARY.md`) — `runnerkit upgrade-runner` must not regress the ephemeral lifecycle, `_diag` log preservation, or mode-aware `status`/`logs`/`doctor`/`down`/`destroy` semantics.

### Existing code that release/upgrade work must integrate with

- `internal/state/schema.go` — `SchemaVersion = "1"`, `RunnerKitVersion`, `RunnerTemplateVersion`, `ServiceTemplateVersion` fields. Migration code attaches here.
- `internal/bootstrap/install.go` and `internal/bootstrap/script.go` — Pinned runner version, `Apply` and `ApplyEphemeral` paths. `runnerkit upgrade-runner` re-enters these with the new pin.
- `internal/cli/up.go`, `internal/cli/status.go`, `internal/cli/doctor.go` — Lazy update-check call sites for `runnerkit X.Y.Z available` notice (D-06).
- `internal/cli/destroy.go` and `internal/provider/hetzner/provision.go` — Live-smoke deferred destroy verification path; resources must be inspectable post-destroy.
- `internal/redact/` — Release flow must not regress redaction; release notes / smoke output must continue to flow through the redactor.
- `cmd/runnerkit/main.go` — Single binary entry point; version string injection point for build-time `-ldflags -X`.

### External docs the release flow depends on

- GoReleaser docs (https://goreleaser.com) — config schema, GitHub Actions integration, Homebrew tap publishing, cosign integration. Researcher should query Context7 for the current API rather than relying on training data.
- Sigstore cosign docs — keyless signing via GitHub Actions OIDC. Same Context7 caveat.
- GitHub Releases API and `workflow_dispatch` semantics (only insofar as 06-04's manual smoke procedure references them).
- Hetzner Cloud API (already used in Phase 4) — for the deferred destroy 404-verification timeout in the live smoke.

### User-facing docs to extend

- `README.md` — Add install matrix (Homebrew + Releases), checksums and cosign verification snippet, link to docs/troubleshooting/, link to RELEASE-NOTES.
- `docs/byo-quickstart.md`, `docs/cloud-quickstart.md`, `docs/safety.md` — Cross-link to the relevant troubleshooting/component pages from any "if this fails" call-outs.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- **Atomic state writes + `SchemaVersion`** (`internal/state/schema.go`): The migration framework already exists in skeleton — Phase 1 added the version field, atomic write, and migration hook points. Plan 06-02 just needs to add the actual forward migrations and the "newer schema, refuse to mutate" branch.
- **Bootstrap re-entry path** (`internal/bootstrap/install.go::Apply`, `ApplyEphemeral`): Both persistent and ephemeral install paths are already idempotent against an existing host. `runnerkit upgrade-runner` is mostly a thin CLI wrapper that calls `Apply` with the new pin and updates state's `RunnerTemplateVersion`/`ServiceTemplateVersion`.
- **Doctor finding model** (`internal/cli/doctor.go`, `internal/ops/doctor.go`): Already produces structured findings with severity. RKD error codes attach to existing finding shapes — no new diagnostic engine needed.
- **Destroy + provider verification** (`internal/cli/destroy.go`, `internal/provider/hetzner/provision.go`): The "verify resource absent or non-billable" loop from Phase 4 is the same primitive the live smoke uses for its deferred destroy assertion.
- **Redactor** (`internal/redact/`): Release-notes generation, smoke-script output, and any logs published in VERIFICATION.md must flow through it. Already integrated into the renderer.

### Established Patterns

- **Plan-before-mutation** (Phases 1, 2, 4): Every mutating command shows what it will do before doing it. `runnerkit upgrade` and `runnerkit upgrade-runner` follow the same shape — print the plan, then act on `--yes` or interactive confirmation.
- **JSON-mode output suppresses interactive UX** (Phase 1): The lazy update notice (D-06) must follow this — silent in JSON mode.
- **Pinned third-party versions** (Phase 2: runner 2.334.0, Phase 4: hcloud-go v1.59.2): Phase 6 keeps the same pinning discipline; the new pin is the GoReleaser version and the cosign version in CI.
- **Fake adapters for tests** (Phases 1–5): The live smoke is the explicit "we have run out of fakes" boundary. Everything else in Phase 6 (release dry-run, migration tests, doctor code routing) stays on fakes.

### Integration Points

- `cmd/runnerkit/main.go` — `-ldflags -X main.version=...` injection during GoReleaser build feeds the lazy update check and the version line in `runnerkit status` / `runnerkit doctor`.
- `internal/state/schema.go::Load` — Migration dispatch lives here; backup-then-migrate-then-write is the contract.
- `internal/bootstrap/script.go` — Pinned runner version constant; bumping this constant + re-running `Apply` is the implementation of `upgrade-runner`.
- `internal/cli/*` doctor/up/status — Update notice integration points (D-06).
- New: `Makefile` target `smoke-live` orchestrating the live BYO + Hetzner runs and the destroy-verification timeout.
- New: `.github/workflows/release.yml` invoking GoReleaser on tag push.

</code_context>

<specifics>
## Specific Ideas

- The "10-minute" claim is the load-bearing promise from PROJECT.md core value. Plan 06-04's stopwatch checklist is the only place RunnerKit measures itself against that claim with real numbers — the captured durations are a feature, not a footnote.
- The `RKD-<COMPONENT>-NNN` error code precedent should be designed so future components (workflow, state, redact) can be added without renumbering existing codes.
- Cosign keyless signing depends on the GitHub Actions OIDC token, so the release workflow must not be invoked from forks or PR contexts where OIDC scope is restricted.
- The Homebrew tap is its own repository (e.g., `homebrew-runnerkit`); GoReleaser commits formula updates to it on each tag. Researcher should confirm current GoReleaser+Homebrew flow before planner locks the tap layout.

</specifics>

<deferred>
## Deferred Ideas

- **Windows CLI host support** — Out of scope for v1 per D-02. Revisit when a real user asks; runner host stays Linux either way.
- **`runnerkit upgrade` self-replacing binary** — Considered and rejected for v1 (D-07). Reconsider if Homebrew-vs-Releases install detection becomes ambiguous in practice.
- **GPG signatures + SBOM** — Considered and rejected for v1 (D-04). Add when a downstream packager or auditor specifically needs them.
- **`go install` channel** — Excluded for v1 (D-01). Revisit if Go-tooled developers report friction installing via Releases binaries.
- **Linux .deb / .rpm packages** — Excluded for v1 (D-01). Revisit when there is demand for runner-host package management.
- **Scheduled / pre-release CI gate for live smokes** — Rejected for v1 (D-11) to avoid storing real Hetzner / GitHub PAT in repo secrets and to avoid recurring billable cost. Consider once an external maintainer team forms.
- **Anonymous telemetry / usage analytics** — Not raised by user; would belong in a future phase if it ever lands.
- **Doctor `--fix` automatic remediation** — Already excluded in REQUIREMENTS Out of Scope; Phase 6 surface intentionally stops at "diagnose + link to docs".
- **Cost-control features (`COST-01..03`)** — v2; Phase 6 docs may mention cost but must not implement idle-shutdown, cost ceilings, or orphan detection.

### Reviewed Todos (not folded)

None — `gsd-tools todo match-phase 6` returned zero matches.

</deferred>

---

*Phase: 06-release-upgrade-docs-and-v1-validation*
*Context gathered: 2026-05-02*
