# RunnerKit v1.2.0 Release Notes

Date: 2026-05-12

## Highlights (SEED-002 — multi-repo on one BYO host)

- **Shared runner cache:** Versioned GitHub Actions runner tarballs under **`/opt/actions-runner/runnerkit-shared-bin/<version>/`**, linked into per-repo install dirs; **`upgrade-runner`** refreshes the shared tree.
- **`runnerkit register`:** Add another repo on an already-prepared host; fails with **`lifecycle_foundation_missing`** if the shared **`runnerkit-runner`** user is not present (use **`up`** / **`init`** first).
- **`runnerkit list`** / **`list --json`** / **`list --host`:** Local inventory grouped by canonical **`user@host:port`**; JSON for automation and smoke contracts.
- **`runnerkit unregister`:** Alias of **`runnerkit down`** (same behavior).
- **`doctor`:** Extra finding when multiple RunnerKit install dirs share one host (operator heads-up).
- **Live smoke:** BYO and cloud scripts assert **`list --json`** via **`scripts/smoke/assert-list-json-contract.sh`**. Optional BYO path: **`RUNNERKIT_SMOKE_MULTI_REPO=1`** + **`RUNNERKIT_SMOKE_REPO2`** runs a second **`register`**, **`scripts/smoke/assert-list-host-repo-count.sh`**, doctor JSON for repo2, then ordered teardown. **`make smoke-live-byo`** checks repo2 when the gate is on.

## Docs

- [`docs/troubleshooting/multi-repo.md`](docs/troubleshooting/multi-repo.md) — commands, PAT scope, isolation caveats, optional maintainer smoke.

## Stopwatch / Live Smoke (maintainer, D-13)

Fill timings for this tag in **`RELEASE-NOTES-v1.0.0.md`** / **`06-VERIFICATION.md`** when you run **`make smoke-live`** (and optional multi-repo BYO leg) on the release commit.

## Upgrade Path

From [docs/upgrade.md](docs/upgrade.md): CLI via **`runnerkit upgrade`**; bundled runner pin via **`runnerkit upgrade-runner`**. Existing single-repo installs are unchanged; adding a second repo uses **`register`** and the shared cache on new installs.
