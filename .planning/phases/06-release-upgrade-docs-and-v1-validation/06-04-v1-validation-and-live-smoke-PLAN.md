---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 04
type: execute
wave: C
depends_on: [01, 02, 03]
files_modified:
  - Makefile
  - cmd/_smokebin/empty_precheck/main.go
  - cmd/_smokebin/destroy_verify/main.go
  - cmd/_smokebin/empty_precheck/main_test.go
  - cmd/_smokebin/destroy_verify/main_test.go
  - scripts/smoke/byo-permission.sh
  - scripts/smoke/cloud-end-to-end.sh
  - scripts/smoke/hetzner-empty-precheck.sh
  - scripts/smoke/hetzner-destroy-verify.sh
  - docs/release-process.md
  - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md
  - RELEASE-NOTES-v1.0.0.md
  - .gitignore
autonomous: false
requirements: [REL-05, DOC-04]
must_haves:
  truths:
    - "`make smoke-live-byo` runs against a maintainer-provided real GitHub repo + BYO host, exits 0, captures duration, and refuses to run if `RUNNERKIT_SMOKE_BYO_HOST`/`RUNNERKIT_SMOKE_REPO`/`gh auth status` are missing."
    - "`make smoke-live-cloud` runs Hetzner empty-project precheck (D-12 gate 1) AS A HARD GATE — refuses to proceed if any pre-existing `runnerkit-*` server, ssh-key, primary-ip, or firewall is found."
    - "After `runnerkit destroy --yes` completes, the destroy-verify polling loop (D-12 gate 2) asserts every saved resource ID returns 404 within `RUNNERKIT_SMOKE_TIMEOUT` seconds (default 300); fails loudly with non-zero exit if any resource lingers."
    - "The 10-minute stopwatch checklist in `docs/release-process.md` records BYO and Hetzner end-to-end durations into `RELEASE-NOTES-v1.0.0.md` and `06-VERIFICATION.md`."
    - "`make smoke-live` is NOT triggered by any GitHub Actions workflow (D-11): grep `.github/workflows/*.yml` for `smoke-live` returns zero hits."
    - "Smoke binaries in `cmd/_smokebin/` are excluded from `go build ./...` default outputs by being under a `_`-prefixed directory; their unit tests pass against fake hcloud.Client fixtures."
  artifacts:
    - path: "Makefile"
      provides: "smoke-live, smoke-live-byo, smoke-live-cloud, smoke-stopwatch targets with env-var precondition checks"
      contains: "smoke-live-byo:"
      contains_also: "smoke-live-cloud:"
      contains_also2: "RUNNERKIT_SMOKE_BYO_HOST"
      contains_also3: "HCLOUD_TOKEN"
    - path: "cmd/_smokebin/empty_precheck/main.go"
      provides: "Hetzner empty-project precheck binary; refuses if any runnerkit-* resource exists"
      contains: "runnerkit-"
      contains_also: "hcloud.NewClient"
    - path: "cmd/_smokebin/destroy_verify/main.go"
      provides: "Hetzner 404-poll destroy-verify binary; reads saved RepositoryState; polls each resource ID"
      contains: "ErrorCodeNotFound"
      contains_also: "RUNNERKIT_SMOKE_TIMEOUT"
    - path: "scripts/smoke/cloud-end-to-end.sh"
      provides: "Full cloud smoke orchestration with trap-on-EXIT runnerkit destroy --yes guard"
      contains: "trap"
      contains_also: "runnerkit destroy --yes"
      contains_also2: "RUNNERKIT_STATE_DIR"
    - path: "docs/release-process.md"
      provides: "10-minute stopwatch checklist appended to existing maintainer doc from Plan 06-01"
      contains: "Stopwatch"
      contains_also: "BYO"
      contains_also2: "Hetzner"
    - path: ".planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md"
      provides: "v1.0.0 baseline verification matrix; durations + runner IDs + Hetzner cost recorded by maintainer post-smoke"
      contains: "v1.0.0"
      contains_also: "### BYO smoke"
      contains_also2: "### Hetzner smoke"
    - path: "RELEASE-NOTES-v1.0.0.md"
      provides: "First release notes with stopwatch durations, bundled runner pin, install/verify reminders"
      contains: "RunnerKit v1.0.0"
      contains_also: "2.334.0"
      contains_also2: "cosign verify-blob"
  key_links:
    - from: "scripts/smoke/cloud-end-to-end.sh"
      to: "cmd/_smokebin/empty_precheck (run BEFORE runnerkit up --cloud hetzner)"
      via: "go run ./cmd/_smokebin/empty_precheck"
    - from: "scripts/smoke/cloud-end-to-end.sh"
      to: "cmd/_smokebin/destroy_verify (run AFTER runnerkit destroy --yes)"
      via: "go run ./cmd/_smokebin/destroy_verify"
    - from: "scripts/smoke/cloud-end-to-end.sh trap EXIT INT TERM"
      to: "runnerkit destroy --yes (cleanup orphans on Ctrl-C)"
      via: "Bash trap (Pitfall 7 mitigation)"
    - from: "Makefile smoke-live"
      to: "smoke-live-byo + smoke-live-cloud + smoke-stopwatch"
      via: ".PHONY chain — ALL three required for v1 sign-off (D-10)"
---

<objective>
Land the live validation harness that proves RunnerKit v1's promises end-to-end and closes the two outstanding live-smoke notes from STATE.md (Phase 1 GitHub permission smoke, Phase 4 Hetzner billable smoke). Add `make smoke-live*` Makefile targets, `cmd/_smokebin/` Go programs for empty-project precheck and destroy-verify polling, `scripts/smoke/` shell wrappers with trap-on-Ctrl-C cleanup, the 10-minute stopwatch checklist in `docs/release-process.md`, the `06-VERIFICATION.md` baseline file, and the first `RELEASE-NOTES-v1.0.0.md`.

Implements **D-10..D-13** from CONTEXT.md.

Purpose: Phase 6 success criterion 4 — "A fresh user can complete at least one supported setup path in about 10 minutes, run a GitHub Actions job on RunnerKit labels, and clean up confidently."

Output: A maintainer who has resolved the Plan 06-01 tap-repo + secret prerequisites can run `make smoke-live` on their machine, watch BYO and Hetzner end-to-end smokes pass with the destroy-verify gates, capture durations into `RELEASE-NOTES-v1.0.0.md` and `06-VERIFICATION.md`, and then push the `v1.0.0` tag.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/REQUIREMENTS.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md
@.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VALIDATION.md
@docs/release-process.md
@internal/state/schema.go
@internal/provider/hetzner/provision.go
@internal/cli/destroy.go
@internal/bootstrap/package.go

<interfaces>
<!-- Existing contracts the smoke harness must integrate with. -->

State directory convention (internal/state/store.go):
- `$XDG_STATE_HOME/runnerkit/state.json` if set, else `$HOME/.local/state/runnerkit/state.json`
- Smoke MUST set `RUNNERKIT_STATE_DIR` to a tempdir (or equivalent) to isolate from maintainer's real state.

Hetzner client (internal/provider/hetzner/):
- Module: `github.com/hetznercloud/hcloud-go/hcloud` (v1.59.2 — pinned, do NOT use v2)
- Pattern: `client := hcloud.NewClient(hcloud.WithToken(token))`
- 404 detection: `hcloud.IsError(err, hcloud.ErrorCodeNotFound)` returns true when the resource is gone.
- Resources to check for empty-project precheck and destroy-verify: Server, SSHKey, PrimaryIP, Firewall (the four resources Phase 4 creates).

State.RepositoryState fields needed by destroy-verify (from internal/state/schema.go):
- `Provider.Cloud.ServerID` (string)
- `Provider.Cloud.SSHKeyID` (string)
- `Provider.Cloud.PrimaryIPv4ID` (string), `Provider.Cloud.PrimaryIPv6ID` (string)
- `Provider.Cloud.FirewallID` (string)

Resource naming convention (Phase 4 / Plan 04-02):
- All Hetzner resources are tagged with the `runnerkit-` prefix in their Name field, e.g., `runnerkit-owner-repo-server`, `runnerkit-owner-repo-ssh-key`, etc. The empty-project precheck uses the Name prefix to identify orphans.

Existing destroy semantics (internal/cli/destroy.go):
- `runnerkit destroy --yes` runs GitHub deregistration → remote cleanup → provider cleanup → state removal in order. The provider cleanup invokes `hetzner.Provisioner.Destroy` then `VerifyDestroyed` (the 404 check). On partial failure, state retains checkpoints. Plan 06-04 ADDS an EXTERNAL polling loop that runs AFTER `destroy --yes` returns, asserting that the IDs are 404 from the maintainer's perspective (independent of internal state).

Bundled runner pin (internal/bootstrap/package.go):
- `RunnerVersion = "2.334.0"` — recorded in RELEASE-NOTES-v1.0.0.md.

Module path: `github.com/salar/runnerkit`. Go 1.22.

CI exclusion (D-11):
- `make smoke-live*` targets MUST NOT be referenced by any `.github/workflows/*.yml` file. Verified by grep at end of plan.

Phase 1 / Phase 4 outstanding live smokes this plan closes:
- Phase 1: "controlled live GitHub permission smoke" (STATE.md Blockers/Concerns line). Closed by `make smoke-live-byo`.
- Phase 4: "controlled live Hetzner smoke" (STATE.md Blockers/Concerns line). Closed by `make smoke-live-cloud`.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="false">
  <name>Task 1: Makefile + scripts/smoke/ shell wrappers with env-var preconditions and trap-on-EXIT cleanup</name>
  <files>Makefile, scripts/smoke/byo-permission.sh, scripts/smoke/cloud-end-to-end.sh, scripts/smoke/hetzner-empty-precheck.sh, scripts/smoke/hetzner-destroy-verify.sh, .gitignore</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (Pattern 9 — full Makefile sketch + script orchestration; Pitfall 7 — trap on Ctrl-C)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-10, D-11, D-12 gates 1+2)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VALIDATION.md (lines 74-77: live smoke contracts)
    - docs/release-process.md (Plan 06-01 output — referenced by `make smoke-stopwatch`; Task 3 of THIS plan extends it)
    - .gitignore (verify dist/ is gitignored from Plan 06-01; smoke binaries under cmd/_smokebin/ are auto-excluded by `_` prefix from `go build ./...`)
  </read_first>
  <action>
**Step 1: Create `Makefile`** at repo root with `.PHONY` targets:

```makefile
# RunnerKit Makefile — solo developer + Claude execution.
# Live smoke targets are MAINTAINER-ONLY and must NOT be invoked from CI (D-11).

.PHONY: help test test-race vet lint smoke-live smoke-live-byo smoke-live-cloud smoke-stopwatch release-snapshot

help: ## Show this help.
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-22s %s\n", $$1, $$2}'

test: ## Run the unit/integration test suite.
	go test ./... -count=1

test-race: ## Run the full test suite with -race.
	go test ./... -count=1 -race

vet: ## go vet all packages.
	go vet ./...

release-snapshot: ## Local GoReleaser dry-run (validates the build matrix).
	goreleaser release --snapshot --skip=publish --clean

# ---------- LIVE SMOKES (maintainer-only — D-11) ----------
# These targets create REAL GitHub registrations and REAL Hetzner billable
# resources. Do NOT wire into any GitHub Actions workflow.

smoke-live: smoke-live-byo smoke-live-cloud smoke-stopwatch ## Run all live smokes (BYO + Hetzner + 10-minute stopwatch). Maintainer-only.

smoke-live-byo: ## Phase 1 outstanding: live GitHub permission smoke. Requires RUNNERKIT_SMOKE_BYO_HOST and RUNNERKIT_SMOKE_REPO.
	@test -n "$$RUNNERKIT_SMOKE_BYO_HOST" || { echo "RUNNERKIT_SMOKE_BYO_HOST=user@host required"; exit 2; }
	@test -n "$$RUNNERKIT_SMOKE_REPO"     || { echo "RUNNERKIT_SMOKE_REPO=owner/name required (must be a maintainer-controlled trusted repo)"; exit 2; }
	@command -v gh >/dev/null || { echo "gh CLI not installed; install from https://cli.github.com/"; exit 2; }
	@gh auth status >/dev/null 2>&1 || { echo "gh auth not present; run 'gh auth login' first"; exit 2; }
	./scripts/smoke/byo-permission.sh "$$RUNNERKIT_SMOKE_REPO" "$$RUNNERKIT_SMOKE_BYO_HOST"

smoke-live-cloud: ## Phase 4 outstanding: live Hetzner end-to-end smoke. CREATES BILLABLE RESOURCES. Requires HCLOUD_TOKEN and RUNNERKIT_SMOKE_REPO.
	@test -n "$$HCLOUD_TOKEN"         || { echo "HCLOUD_TOKEN required"; exit 2; }
	@test -n "$$RUNNERKIT_SMOKE_REPO" || { echo "RUNNERKIT_SMOKE_REPO=owner/name required (maintainer-controlled trusted repo, NOT public)"; exit 2; }
	@command -v gh >/dev/null || { echo "gh CLI not installed"; exit 2; }
	@gh auth status >/dev/null 2>&1 || { echo "gh auth not present"; exit 2; }
	# D-12 gate 1: empty-project precheck — refuse if any runnerkit-* resource exists.
	./scripts/smoke/hetzner-empty-precheck.sh
	# Run end-to-end up + status + workflow + destroy.
	./scripts/smoke/cloud-end-to-end.sh "$$RUNNERKIT_SMOKE_REPO"
	# D-12 gate 2: destroy-verify polling — assert 404 within timeout.
	./scripts/smoke/hetzner-destroy-verify.sh "$${RUNNERKIT_SMOKE_TIMEOUT:-300}"

smoke-stopwatch: ## 10-minute stopwatch checklist (D-13). Maintainer manually records into RELEASE-NOTES-vX.Y.Z.md.
	@echo "Open docs/release-process.md '## Stopwatch checklist' and follow the BYO + Hetzner end-to-end timing."
	@echo "Record durations into RELEASE-NOTES-v$${VER:-1.0.0}.md and .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md."
```

**Step 2: Create `scripts/smoke/byo-permission.sh`** — Phase 1 outstanding live smoke. Wraps `runnerkit up --mode persistent --host` against a maintainer host:

```bash
#!/usr/bin/env bash
# scripts/smoke/byo-permission.sh
# Phase 1 outstanding live GitHub permission smoke.
# Args: $1 = owner/repo, $2 = user@host
set -euo pipefail

REPO="${1:?repo required}"
HOST="${2:?host required}"

echo "===> [smoke-byo] Setting up isolated state dir"
SMOKE_DIR="$(mktemp -d -t runnerkit-smoke-byo-XXXXXXXX)"
export RUNNERKIT_STATE_DIR="${SMOKE_DIR}"
trap 'rm -rf "${SMOKE_DIR}"' EXIT

START_EPOCH=$(date +%s)

echo "===> [smoke-byo] runnerkit up --repo ${REPO} --host ${HOST} --mode persistent --yes"
go run ./cmd/runnerkit up --repo "${REPO}" --host "${HOST}" --mode persistent --yes

echo "===> [smoke-byo] runnerkit status --repo ${REPO}"
go run ./cmd/runnerkit status --repo "${REPO}"

echo "===> [smoke-byo] runnerkit doctor --repo ${REPO}"
go run ./cmd/runnerkit doctor --repo "${REPO}" || true

echo "===> [smoke-byo] runnerkit down --repo ${REPO} --yes"
go run ./cmd/runnerkit down --repo "${REPO}" --yes

END_EPOCH=$(date +%s)
DURATION=$((END_EPOCH - START_EPOCH))
echo "===> [smoke-byo] OK — duration: ${DURATION}s"
echo "[smoke-byo] BYO_DURATION_SECONDS=${DURATION}" >&2
```

**Step 3: Create `scripts/smoke/cloud-end-to-end.sh`** — Phase 4 outstanding live smoke. Includes the trap-on-EXIT cleanup (Pitfall 7):

```bash
#!/usr/bin/env bash
# scripts/smoke/cloud-end-to-end.sh
# Phase 4 outstanding live Hetzner billable-resource smoke.
# Args: $1 = owner/repo
# Pre: empty-project precheck has already run (called by Makefile).
# Post: hetzner-destroy-verify.sh runs after this exits (called by Makefile).
set -euo pipefail

REPO="${1:?repo required}"
: "${HCLOUD_TOKEN:?HCLOUD_TOKEN required}"

echo "===> [smoke-cloud] Setting up isolated state dir"
SMOKE_DIR="$(mktemp -d -t runnerkit-smoke-cloud-XXXXXXXX)"
export RUNNERKIT_STATE_DIR="${SMOKE_DIR}"

# Pitfall 7 mitigation: trap Ctrl-C / abnormal exit and run runnerkit destroy
# --yes BEFORE removing the state dir. This guarantees no orphan billable resources
# even if the maintainer interrupts the smoke mid-run.
cleanup() {
    rc=$?
    echo "===> [smoke-cloud] cleanup trap fired (rc=$rc)"
    if [ -f "${SMOKE_DIR}/state.json" ]; then
        go run ./cmd/runnerkit destroy --repo "${REPO}" --yes || echo "[smoke-cloud] WARN: destroy --yes failed during trap; check Hetzner Console manually"
    fi
    rm -rf "${SMOKE_DIR}"
    exit $rc
}
trap cleanup EXIT INT TERM

START_EPOCH=$(date +%s)

echo "===> [smoke-cloud] runnerkit up --repo ${REPO} --cloud hetzner --mode persistent --yes"
go run ./cmd/runnerkit up --repo "${REPO}" --cloud hetzner --mode persistent --yes

echo "===> [smoke-cloud] runnerkit status --repo ${REPO}"
go run ./cmd/runnerkit status --repo "${REPO}"

echo "===> [smoke-cloud] runnerkit doctor --repo ${REPO}"
go run ./cmd/runnerkit doctor --repo "${REPO}" || true

echo "===> [smoke-cloud] runnerkit destroy --repo ${REPO} --yes"
go run ./cmd/runnerkit destroy --repo "${REPO}" --yes

# DO NOT remove the state file before the destroy-verify wrapper reads it.
# We move it to a sibling location so the verifier can read the saved cloud IDs.
cp "${SMOKE_DIR}/state.json" "${SMOKE_DIR}/state-after-destroy.json" 2>/dev/null || true

END_EPOCH=$(date +%s)
DURATION=$((END_EPOCH - START_EPOCH))
echo "===> [smoke-cloud] OK — duration: ${DURATION}s"
echo "[smoke-cloud] CLOUD_DURATION_SECONDS=${DURATION}" >&2

# Disarm the trap on success — Makefile will run hetzner-destroy-verify.sh next,
# which needs the state file to read saved IDs.
trap - EXIT INT TERM
# But still clean up on signal:
trap 'rm -rf "${SMOKE_DIR}"' INT TERM

# Hand the state path to the next stage via env (the verify script reads it).
echo "${SMOKE_DIR}" > "${SMOKE_DIR}/.smoke_dir"
export RUNNERKIT_SMOKE_STATE_DIR="${SMOKE_DIR}"
echo "[smoke-cloud] State dir handed to destroy-verify: ${SMOKE_DIR}"
```

NOTE on the env hand-off: env vars don't propagate from a child script back to make. The Makefile target needs to know the state dir for the `hetzner-destroy-verify.sh` step. Solution: write a stable path file at `/tmp/runnerkit-smoke.last-state-dir` or use a fixed `RUNNERKIT_SMOKE_STATE_DIR` from the start. Implement option B for simplicity — set `RUNNERKIT_SMOKE_STATE_DIR` in the Makefile target itself, pass it through to both scripts:

Update Makefile `smoke-live-cloud` block:
```makefile
smoke-live-cloud: ## ...
	@test -n "$$HCLOUD_TOKEN" || { ...; exit 2; }
	@test -n "$$RUNNERKIT_SMOKE_REPO" || { ...; exit 2; }
	@gh auth status >/dev/null 2>&1 || { ...; exit 2; }
	@RUNNERKIT_SMOKE_STATE_DIR="$$(mktemp -d -t runnerkit-smoke-cloud-XXXXXXXX)" && \
		export RUNNERKIT_SMOKE_STATE_DIR && \
		echo "[smoke-cloud] state dir: $$RUNNERKIT_SMOKE_STATE_DIR" && \
		./scripts/smoke/hetzner-empty-precheck.sh && \
		./scripts/smoke/cloud-end-to-end.sh "$$RUNNERKIT_SMOKE_REPO" && \
		./scripts/smoke/hetzner-destroy-verify.sh "$${RUNNERKIT_SMOKE_TIMEOUT:-300}" && \
		rm -rf "$$RUNNERKIT_SMOKE_STATE_DIR"
```

And update `cloud-end-to-end.sh` to use `${RUNNERKIT_SMOKE_STATE_DIR:?...}` instead of `mktemp` so the state dir is shared with the verify step.

**Step 4: Create `scripts/smoke/hetzner-empty-precheck.sh`** — wraps the `cmd/_smokebin/empty_precheck` Go program (Task 2):

```bash
#!/usr/bin/env bash
# D-12 gate 1: refuse if Hetzner project contains any runnerkit-* resource.
set -euo pipefail
: "${HCLOUD_TOKEN:?HCLOUD_TOKEN required}"

echo "===> [smoke-precheck] Hetzner empty-project precheck"
go run ./cmd/_smokebin/empty_precheck
echo "===> [smoke-precheck] OK — no orphan runnerkit-* resources"
```

**Step 5: Create `scripts/smoke/hetzner-destroy-verify.sh`** — wraps the `cmd/_smokebin/destroy_verify` Go program (Task 2):

```bash
#!/usr/bin/env bash
# D-12 gate 2: poll Hetzner for 404 on every saved resource ID.
# Args: $1 = timeout seconds (default 300).
set -euo pipefail
: "${HCLOUD_TOKEN:?HCLOUD_TOKEN required}"
: "${RUNNERKIT_SMOKE_STATE_DIR:?RUNNERKIT_SMOKE_STATE_DIR required (set by Makefile)}"

TIMEOUT="${1:-300}"
STATE_FILE="${RUNNERKIT_SMOKE_STATE_DIR}/state.json"

if [ ! -f "${STATE_FILE}" ]; then
    # destroy may have already removed state.json — try the sibling backup.
    STATE_FILE="${RUNNERKIT_SMOKE_STATE_DIR}/state-after-destroy.json"
fi

echo "===> [smoke-verify] Polling Hetzner for 404 on saved IDs (timeout: ${TIMEOUT}s)"
RUNNERKIT_SMOKE_TIMEOUT="${TIMEOUT}" \
RUNNERKIT_SMOKE_STATE_FILE="${STATE_FILE}" \
    go run ./cmd/_smokebin/destroy_verify
echo "===> [smoke-verify] OK — every saved resource ID returns 404"
```

**Step 6: Make all 4 shell scripts executable:**

```bash
chmod +x scripts/smoke/byo-permission.sh scripts/smoke/cloud-end-to-end.sh scripts/smoke/hetzner-empty-precheck.sh scripts/smoke/hetzner-destroy-verify.sh
```

**Step 7: Confirm `.gitignore`** does not exclude `scripts/`. Verify by `cat .gitignore`. Add `cmd/_smokebin/empty_precheck/empty_precheck` and `cmd/_smokebin/destroy_verify/destroy_verify` lines if go-build artifacts could land there (defensive; `go run` doesn't write binaries to source dir, but `go build` might).
  </action>
  <verify>
    <automated>test -f Makefile && grep -q "smoke-live-byo:" Makefile && grep -q "smoke-live-cloud:" Makefile && grep -q "RUNNERKIT_SMOKE_BYO_HOST" Makefile && grep -q "HCLOUD_TOKEN" Makefile && grep -q "smoke-stopwatch" Makefile && test -x scripts/smoke/byo-permission.sh && test -x scripts/smoke/cloud-end-to-end.sh && test -x scripts/smoke/hetzner-empty-precheck.sh && test -x scripts/smoke/hetzner-destroy-verify.sh && grep -q "trap" scripts/smoke/cloud-end-to-end.sh && grep -q "runnerkit destroy --yes\|runnerkit destroy --repo" scripts/smoke/cloud-end-to-end.sh && grep -q "RUNNERKIT_SMOKE_STATE_DIR\|RUNNERKIT_STATE_DIR" scripts/smoke/cloud-end-to-end.sh && ! grep -rq "smoke-live" .github/workflows/ 2>/dev/null && bash -n scripts/smoke/byo-permission.sh && bash -n scripts/smoke/cloud-end-to-end.sh && bash -n scripts/smoke/hetzner-empty-precheck.sh && bash -n scripts/smoke/hetzner-destroy-verify.sh</automated>
  </verify>
  <acceptance_criteria>
    - `Makefile` exists at repo root with `.PHONY: smoke-live smoke-live-byo smoke-live-cloud smoke-stopwatch` declarations.
    - `make smoke-live-byo` precondition checks include: `RUNNERKIT_SMOKE_BYO_HOST` non-empty, `RUNNERKIT_SMOKE_REPO` non-empty, `gh auth status` succeeds. (verified by reading the Makefile body — each `test -n` or `command -v` line is present).
    - `make smoke-live-cloud` precondition checks include: `HCLOUD_TOKEN` non-empty, `RUNNERKIT_SMOKE_REPO` non-empty, `gh auth status` succeeds.
    - `make smoke-live-cloud` invokes empty-precheck BEFORE cloud-end-to-end AND destroy-verify AFTER (ordering enforced via && chain).
    - `scripts/smoke/cloud-end-to-end.sh` includes a `trap cleanup EXIT INT TERM` line that calls `runnerkit destroy --yes` (Pitfall 7).
    - All 4 shell scripts pass `bash -n <script>` syntax check and are executable (`test -x`).
    - No GitHub Actions workflow file mentions `smoke-live` (D-11): `grep -rq "smoke-live" .github/workflows/` returns 1 (not found) — i.e., the negation condition.
    - All four shell scripts use `set -euo pipefail` (search each script for the literal).
    - `Makefile` `help` target exists and renders comments after `##`.
  </acceptance_criteria>
  <done>Makefile has all required smoke targets with env-var preconditions; 4 shell scripts exist, are executable, syntactically valid, with trap-on-EXIT cleanup in cloud-end-to-end.sh; no CI workflow references make smoke-live.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: cmd/_smokebin/empty_precheck and destroy_verify Go programs + unit tests with fake hcloud client</name>
  <files>cmd/_smokebin/empty_precheck/main.go, cmd/_smokebin/empty_precheck/main_test.go, cmd/_smokebin/destroy_verify/main.go, cmd/_smokebin/destroy_verify/main_test.go</files>
  <read_first>
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (Pattern 9 hcloud-go API patterns; "Code Examples" section for the IsError snippet; D-12 gates 1+2)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VALIDATION.md (lines 76, 77: TestEmptyPrecheck_RefusesOnExisting, TestDestroyVerify_Timeout)
    - internal/state/schema.go (Provider.Cloud field shape — what destroy_verify reads)
    - internal/provider/hetzner/provision.go (existing hcloud client usage as a reference)
    - go.mod (confirm hcloud-go v1.59.2 is already a dependency from Phase 4)
  </read_first>
  <behavior>
    - Test 1: `TestEmptyPrecheck_RefusesOnExisting` — fake hcloud client where `Server.All` returns one server with Name `runnerkit-test-server`. Call empty_precheck.run(ctx, client). Assert it returns a non-nil error whose message names the offending resource. Then test where Server.All returns servers ONLY with names not starting with `runnerkit-` (e.g., `unrelated-server`); assert run returns nil.
    - Test 2: `TestDestroyVerify_Timeout` — fake hcloud client. Set fixture state.json with a server ID. Configure the fake to return the server (not 404) for the first 3 polls, then return ErrorCodeNotFound. Assert run returns nil, and that the poll attempted >= 3 times. Then test the failure case: fake always returns the server (never 404); assert run returns non-nil error within `RUNNERKIT_SMOKE_TIMEOUT=2` seconds (not the default 300; use a short timeout for the test to keep it fast).
    - Test 3: `TestEmptyPrecheck_AllResourceTypes` — fake client where SSHKey.All returns one key named `runnerkit-test-key`; assert refuse. Then PrimaryIP.All has one named `runnerkit-test-ip`; refuse. Then Firewall.All has one named `runnerkit-test-fw`; refuse. (The precheck must scan ALL FOUR resource types Phase 4 creates.)
  </behavior>
  <action>
**Step 1: Create `cmd/_smokebin/empty_precheck/main.go`:**

```go
// Command empty_precheck implements D-12 gate 1: refuse the live cloud smoke
// if the configured Hetzner project contains any pre-existing `runnerkit-*`
// managed servers, ssh-keys, primary-ips, or firewalls.
//
// Phase 4 creates exactly these four resource types per `runnerkit up --cloud
// hetzner`; finding any with the `runnerkit-` Name prefix means a previous
// smoke leaked or a previous up failed mid-provision.
//
// This binary is excluded from `go build ./...` by the `_smokebin` directory's
// `_` prefix (Go convention).
package main

import (
    "context"
    "fmt"
    "os"
    "strings"
    "time"

    hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

func main() {
    if err := run(context.Background(), nil); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

// hcloudClient is the subset of *hcloud.Client we use; lets tests inject a fake.
type hcloudClient interface {
    AllServers(ctx context.Context) ([]*hcloud.Server, error)
    AllSSHKeys(ctx context.Context) ([]*hcloud.SSHKey, error)
    AllPrimaryIPs(ctx context.Context) ([]*hcloud.PrimaryIP, error)
    AllFirewalls(ctx context.Context) ([]*hcloud.Firewall, error)
}

type realClient struct{ c *hcloud.Client }

func (r realClient) AllServers(ctx context.Context) ([]*hcloud.Server, error) {
    return r.c.Server.All(ctx)
}
func (r realClient) AllSSHKeys(ctx context.Context) ([]*hcloud.SSHKey, error) {
    return r.c.SSHKey.All(ctx)
}
func (r realClient) AllPrimaryIPs(ctx context.Context) ([]*hcloud.PrimaryIP, error) {
    return r.c.PrimaryIP.All(ctx)
}
func (r realClient) AllFirewalls(ctx context.Context) ([]*hcloud.Firewall, error) {
    return r.c.Firewall.All(ctx)
}

const namePrefix = "runnerkit-"

func run(ctx context.Context, client hcloudClient) error {
    if client == nil {
        token := os.Getenv("HCLOUD_TOKEN")
        if token == "" {
            return fmt.Errorf("HCLOUD_TOKEN required")
        }
        client = realClient{c: hcloud.NewClient(hcloud.WithToken(token))}
    }
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    var orphans []string
    if servers, err := client.AllServers(ctx); err != nil {
        return fmt.Errorf("list servers: %w", err)
    } else {
        for _, s := range servers {
            if strings.HasPrefix(s.Name, namePrefix) {
                orphans = append(orphans, fmt.Sprintf("server: %s (id %d)", s.Name, s.ID))
            }
        }
    }
    if keys, err := client.AllSSHKeys(ctx); err != nil {
        return fmt.Errorf("list ssh keys: %w", err)
    } else {
        for _, k := range keys {
            if strings.HasPrefix(k.Name, namePrefix) {
                orphans = append(orphans, fmt.Sprintf("ssh-key: %s (id %d)", k.Name, k.ID))
            }
        }
    }
    if ips, err := client.AllPrimaryIPs(ctx); err != nil {
        return fmt.Errorf("list primary ips: %w", err)
    } else {
        for _, p := range ips {
            if strings.HasPrefix(p.Name, namePrefix) {
                orphans = append(orphans, fmt.Sprintf("primary-ip: %s (id %d)", p.Name, p.ID))
            }
        }
    }
    if fws, err := client.AllFirewalls(ctx); err != nil {
        return fmt.Errorf("list firewalls: %w", err)
    } else {
        for _, f := range fws {
            if strings.HasPrefix(f.Name, namePrefix) {
                orphans = append(orphans, fmt.Sprintf("firewall: %s (id %d)", f.Name, f.ID))
            }
        }
    }
    if len(orphans) > 0 {
        return fmt.Errorf("D-12 gate 1: Hetzner project contains %d pre-existing runnerkit-* resources; refuse to run live smoke. Resources:\n  - %s\nClean up with `runnerkit destroy --yes` for each, or via Hetzner Console.", len(orphans), strings.Join(orphans, "\n  - "))
    }
    return nil
}
```

**Step 2: Create `cmd/_smokebin/empty_precheck/main_test.go`** with `TestEmptyPrecheck_RefusesOnExisting` and `TestEmptyPrecheck_AllResourceTypes`. Use a fake `hcloudClient` interface implementation:

```go
type fakeClient struct {
    servers   []*hcloud.Server
    sshKeys   []*hcloud.SSHKey
    primaryIPs []*hcloud.PrimaryIP
    firewalls []*hcloud.Firewall
}
func (f *fakeClient) AllServers(_ context.Context) ([]*hcloud.Server, error) { return f.servers, nil }
// ... etc
```

Test cases:
1. `client.servers = []*hcloud.Server{{Name: "runnerkit-test-server", ID: 1}}` → `run()` returns error containing "runnerkit-test-server".
2. `client.servers = []*hcloud.Server{{Name: "unrelated-server", ID: 2}}` → `run()` returns nil.
3. Same checks for SSHKey, PrimaryIP, Firewall.

**Step 3: Create `cmd/_smokebin/destroy_verify/main.go`:**

```go
// Command destroy_verify implements D-12 gate 2: after `runnerkit destroy --yes`,
// poll Hetzner for 404 on every saved resource ID. Fail loudly if any resource
// lingers within the timeout.
//
// State file path is read from RUNNERKIT_SMOKE_STATE_FILE (set by the calling
// shell script). Timeout is read from RUNNERKIT_SMOKE_TIMEOUT (seconds; default
// 300).
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "strconv"
    "time"

    hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

const pollInterval = 5 * time.Second

func main() {
    if err := run(context.Background(), nil); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

type cloudIDs struct {
    ServerID      int64
    SSHKeyID      int64
    PrimaryIPv4ID int64
    PrimaryIPv6ID int64
    FirewallID    int64
}

type verifierClient interface {
    GetServerByID(ctx context.Context, id int64) (*hcloud.Server, error)
    GetSSHKeyByID(ctx context.Context, id int64) (*hcloud.SSHKey, error)
    GetPrimaryIPByID(ctx context.Context, id int64) (*hcloud.PrimaryIP, error)
    GetFirewallByID(ctx context.Context, id int64) (*hcloud.Firewall, error)
}

type realVerifier struct{ c *hcloud.Client }

func (r realVerifier) GetServerByID(ctx context.Context, id int64) (*hcloud.Server, error) {
    s, _, err := r.c.Server.GetByID(ctx, int(id))
    return s, err
}
// ...similar for SSHKey, PrimaryIP, Firewall

func run(ctx context.Context, client verifierClient) error {
    timeoutSec, _ := strconv.Atoi(os.Getenv("RUNNERKIT_SMOKE_TIMEOUT"))
    if timeoutSec <= 0 {
        timeoutSec = 300
    }
    statePath := os.Getenv("RUNNERKIT_SMOKE_STATE_FILE")
    if statePath == "" {
        return fmt.Errorf("RUNNERKIT_SMOKE_STATE_FILE required")
    }
    raw, err := os.ReadFile(statePath)
    if err != nil {
        return fmt.Errorf("read state file: %w", err)
    }
    ids, err := extractCloudIDs(raw)
    if err != nil {
        return err
    }
    if client == nil {
        token := os.Getenv("HCLOUD_TOKEN")
        if token == "" {
            return fmt.Errorf("HCLOUD_TOKEN required")
        }
        client = realVerifier{c: hcloud.NewClient(hcloud.WithToken(token))}
    }
    deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)

    return pollUntilGone(ctx, client, ids, deadline)
}

// extractCloudIDs reads state.json bytes and pulls out the saved Hetzner resource IDs.
// Mirrors the structure of internal/state/schema.go::ProviderRef.Cloud (best-effort
// JSON unmarshal — we only need the integer ID fields).
func extractCloudIDs(raw []byte) (cloudIDs, error) {
    var partial struct {
        Repositories []struct {
            Provider struct {
                Cloud struct {
                    ServerID      string `json:"server_id"`
                    SSHKeyID      string `json:"ssh_key_id"`
                    PrimaryIPv4ID string `json:"primary_ipv4_id"`
                    PrimaryIPv6ID string `json:"primary_ipv6_id"`
                    FirewallID    string `json:"firewall_id"`
                } `json:"cloud"`
            } `json:"provider"`
        } `json:"repositories"`
    }
    if err := json.Unmarshal(raw, &partial); err != nil {
        return cloudIDs{}, fmt.Errorf("parse state json: %w", err)
    }
    if len(partial.Repositories) == 0 {
        // After a successful destroy --yes, the repo entry is removed from
        // state. That is the SUCCESS signal — nothing to verify.
        return cloudIDs{}, nil
    }
    c := partial.Repositories[0].Provider.Cloud
    out := cloudIDs{}
    out.ServerID, _ = strconv.ParseInt(c.ServerID, 10, 64)
    out.SSHKeyID, _ = strconv.ParseInt(c.SSHKeyID, 10, 64)
    out.PrimaryIPv4ID, _ = strconv.ParseInt(c.PrimaryIPv4ID, 10, 64)
    out.PrimaryIPv6ID, _ = strconv.ParseInt(c.PrimaryIPv6ID, 10, 64)
    out.FirewallID, _ = strconv.ParseInt(c.FirewallID, 10, 64)
    return out, nil
}

func pollUntilGone(ctx context.Context, client verifierClient, ids cloudIDs, deadline time.Time) error {
    if ids == (cloudIDs{}) {
        // No IDs to verify — destroy already removed the state entry. Success.
        return nil
    }
    for {
        if time.Now().After(deadline) {
            return fmt.Errorf("D-12 gate 2: timeout waiting for Hetzner resources to return 404 (deadline=%s, ids=%+v)", deadline.Format(time.RFC3339), ids)
        }
        remaining, err := checkRemaining(ctx, client, ids)
        if err != nil {
            return err
        }
        if len(remaining) == 0 {
            return nil // every resource gone
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(pollInterval):
        }
        ids = remaining
    }
}

func checkRemaining(ctx context.Context, client verifierClient, ids cloudIDs) (cloudIDs, error) {
    out := cloudIDs{}
    if ids.ServerID > 0 {
        s, err := client.GetServerByID(ctx, ids.ServerID)
        if err != nil && !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
            return cloudIDs{}, fmt.Errorf("get server %d: %w", ids.ServerID, err)
        }
        if s != nil {
            out.ServerID = ids.ServerID
        }
    }
    if ids.SSHKeyID > 0 {
        k, err := client.GetSSHKeyByID(ctx, ids.SSHKeyID)
        if err != nil && !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
            return cloudIDs{}, fmt.Errorf("get ssh key %d: %w", ids.SSHKeyID, err)
        }
        if k != nil {
            out.SSHKeyID = ids.SSHKeyID
        }
    }
    if ids.PrimaryIPv4ID > 0 {
        p, err := client.GetPrimaryIPByID(ctx, ids.PrimaryIPv4ID)
        if err != nil && !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
            return cloudIDs{}, fmt.Errorf("get primary ipv4 %d: %w", ids.PrimaryIPv4ID, err)
        }
        if p != nil {
            out.PrimaryIPv4ID = ids.PrimaryIPv4ID
        }
    }
    if ids.PrimaryIPv6ID > 0 {
        p, err := client.GetPrimaryIPByID(ctx, ids.PrimaryIPv6ID)
        if err != nil && !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
            return cloudIDs{}, fmt.Errorf("get primary ipv6 %d: %w", ids.PrimaryIPv6ID, err)
        }
        if p != nil {
            out.PrimaryIPv6ID = ids.PrimaryIPv6ID
        }
    }
    if ids.FirewallID > 0 {
        f, err := client.GetFirewallByID(ctx, ids.FirewallID)
        if err != nil && !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
            return cloudIDs{}, fmt.Errorf("get firewall %d: %w", ids.FirewallID, err)
        }
        if f != nil {
            out.FirewallID = ids.FirewallID
        }
    }
    return out, nil
}

var _ = errors.Is // keep imports stable
```

**Step 4: Create `cmd/_smokebin/destroy_verify/main_test.go`** with `TestDestroyVerify_Timeout`. Use a fake `verifierClient` that returns the resource for the first N polls, then ErrorCodeNotFound:

```go
type fakeVerifier struct {
    serverHits int
    return404After int
}

func (f *fakeVerifier) GetServerByID(_ context.Context, _ int64) (*hcloud.Server, error) {
    f.serverHits++
    if f.serverHits > f.return404After {
        return nil, hcloud.Error{Code: hcloud.ErrorCodeNotFound}
    }
    return &hcloud.Server{ID: 1}, nil
}
// ... other methods return 404 always
```

Test the success path (404 after 3 polls) AND the failure path (always returns server) with a short `RUNNERKIT_SMOKE_TIMEOUT=2`. Use `t.Setenv("RUNNERKIT_SMOKE_TIMEOUT", "2")` and `t.Setenv("RUNNERKIT_SMOKE_STATE_FILE", path)` where `path` is a fixture state.json with a server ID.

The test names MUST be exactly `TestEmptyPrecheck_RefusesOnExisting` and `TestDestroyVerify_Timeout` per `06-VALIDATION.md` lines 76-77.
  </action>
  <verify>
    <automated>test -f cmd/_smokebin/empty_precheck/main.go && test -f cmd/_smokebin/destroy_verify/main.go && go vet ./cmd/_smokebin/... && go test ./cmd/_smokebin/empty_precheck -run 'TestEmptyPrecheck_RefusesOnExisting|TestEmptyPrecheck_AllResourceTypes' -count=1 && go test ./cmd/_smokebin/destroy_verify -run TestDestroyVerify_Timeout -count=1 && grep -q "namePrefix = \"runnerkit-\"" cmd/_smokebin/empty_precheck/main.go && grep -q "ErrorCodeNotFound" cmd/_smokebin/destroy_verify/main.go && grep -q "RUNNERKIT_SMOKE_TIMEOUT" cmd/_smokebin/destroy_verify/main.go && grep -q "go build" go.sum 2>/dev/null || true && go list ./cmd/_smokebin/... | wc -l | grep -q 2</automated>
  </verify>
  <acceptance_criteria>
    - `cmd/_smokebin/empty_precheck/main.go` exists and exports `main()` + private `run(ctx, client)`. Scans Server, SSHKey, PrimaryIP, Firewall (all 4 resource types) for `runnerkit-` Name prefix.
    - `cmd/_smokebin/destroy_verify/main.go` exists; reads state.json from `$RUNNERKIT_SMOKE_STATE_FILE`; honors `$RUNNERKIT_SMOKE_TIMEOUT` (default 300s); polls every 5s; uses `hcloud.IsError(err, hcloud.ErrorCodeNotFound)` to detect 404.
    - The `_smokebin` directory is excluded from `go build ./...` (verified: `go list ./...` does not include `cmd/_smokebin/...` because Go skips directories starting with `_`).
    - Tests `TestEmptyPrecheck_RefusesOnExisting` and `TestEmptyPrecheck_AllResourceTypes` pass.
    - Test `TestDestroyVerify_Timeout` passes (success-after-N-polls AND timeout-failure paths).
    - `go vet ./cmd/_smokebin/...` passes.
    - All validation matrix rows for D-12 (lines 76, 77) are green.
  </acceptance_criteria>
  <done>cmd/_smokebin/empty_precheck and destroy_verify Go programs exist with hcloud-go v1.59.2 integration, fake-client unit tests covering all 4 resource types and both success/timeout paths, `_` prefix excludes them from default builds, all 3 tests green.</done>
</task>

<task type="auto" tdd="false">
  <name>Task 3: Add 10-minute stopwatch checklist to docs/release-process.md + create 06-VERIFICATION.md baseline + RELEASE-NOTES-v1.0.0.md template</name>
  <files>docs/release-process.md, .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md, RELEASE-NOTES-v1.0.0.md</files>
  <read_first>
    - docs/release-process.md (Plan 06-01 output — append the stopwatch section to it)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-CONTEXT.md (D-13)
    - .planning/phases/06-release-upgrade-docs-and-v1-validation/06-RESEARCH.md (D-13 baseline content)
    - internal/bootstrap/package.go (RunnerVersion constant — recorded in RELEASE-NOTES)
    - .planning/PROJECT.md (the load-bearing 10-minute claim from Core Value)
  </read_first>
  <action>
**Step 1: Append a stopwatch checklist section** to `docs/release-process.md` (existing file from Plan 06-01). Add at the END:

```markdown
## Stopwatch Checklist (D-13)

This is the 10-minute reliable-runner promise from PROJECT.md Core Value.
Run this on a CLEAN machine (fresh laptop, fresh VM, clean
`$HOME/.local/state/runnerkit/`) before tagging each release. The
maintainer's wall-clock numbers go into `RELEASE-NOTES-vX.Y.Z.md`.

### BYO path (target: ≤ 10 minutes)

| Step | Description | T0 | T_now | Δ |
|---|---|---|---|---|
| 1 | `gh auth login` (if not already authed) | | | |
| 2 | `runnerkit up --repo $REPO --host user@host --mode persistent` | | | |
| 3 | Trigger a workflow targeting the `runnerkit-...` label | | | |
| 4 | Observe job runs on the new runner | | | |
| 5 | `runnerkit down --repo $REPO --yes` | | | |

Total wall-clock: __ minutes __ seconds.

### Hetzner cloud path (target: ≤ 10 minutes)

| Step | Description | T0 | T_now | Δ |
|---|---|---|---|---|
| 1 | `gh auth login` (if not already authed) | | | |
| 2 | `export HCLOUD_TOKEN=...` (one-time) | | | |
| 3 | `runnerkit up --repo $REPO --cloud hetzner --mode persistent` | | | |
| 4 | Trigger a workflow targeting the `runnerkit-...` label | | | |
| 5 | Observe job runs on the new runner | | | |
| 6 | `runnerkit destroy --repo $REPO --yes` | | | |
| 7 | Verify Hetzner Console shows 0 `runnerkit-*` resources | | | |

Total wall-clock: __ minutes __ seconds.
Hetzner cost (from project billing dashboard): __ EUR.

### Recording

After running both paths, copy the totals into:
1. `RELEASE-NOTES-v$VERSION.md` (per-release file, committed at tag time).
2. `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md`
   for the v1.0.0 baseline (ONE-TIME — overwritten only if the baseline
   methodology changes).

If either path exceeds 10 minutes, do NOT tag the release. Investigate the
slow step, fix it, and re-run the stopwatch.
```

**Step 2: Create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md`** — v1.0.0 baseline (skeleton; maintainer fills the numbers when running smoke):

```markdown
***
phase: 06-release-upgrade-docs-and-v1-validation
type: verification
status: pending
created: 2026-05-02
***

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
- Resource IDs created (server/ssh-key/primary-ip(v4)/primary-ip(v6)/firewall): `____ / ____ / ____ / ____ / ____`
- D-12 gate 1 (empty-project precheck) status: `____` (PASS / FAIL)
- D-12 gate 2 (destroy-verify 404 within timeout) status: `____` (PASS / FAIL)
- Empty-project precheck final ID list size: `0` (must be exactly zero on a successful smoke)

### 10-minute stopwatch (D-13)

- [ ] BYO total: `____ minutes ____ seconds` (target ≤ 10 minutes)
- [ ] Hetzner total: `____ minutes ____ seconds` (target ≤ 10 minutes)

## Bundled Versions

- runner pin (internal/bootstrap/package.go::RunnerVersion): `2.334.0`
- GoReleaser CI version: `v2.15.4`
- cosign CI version: `v3.0.6`
- hcloud-go (Phase 4 pinned): `v1.59.2`

## Sign-Off

- [ ] Maintainer signature: `____________ (date)`
- [ ] All gates green; ready to push `git tag -a v1.0.0`
```

**Step 3: Create `RELEASE-NOTES-v1.0.0.md`** at repo root (template; maintainer fills numbers post-smoke):

```markdown
# RunnerKit v1.0.0

First public release.

## What This Is

RunnerKit is a CLI-first tool that helps solo developers create and manage
self-hosted GitHub Actions runners without becoming infrastructure operators.

## Bundled Versions

- GitHub Actions runner pin: **2.334.0**
- Built with Go 1.22
- Released by GoReleaser v2.15.4 + cosign v3.0.6 (keyless OIDC signature on
  `runnerkit_v1.0.0_checksums.txt`)

## Supported CLI Host Platforms

- macOS arm64
- macOS amd64
- Linux amd64
- Linux arm64

Windows, 32-bit Linux, and 32-bit ARM are NOT supported.

## Install

See [README.md](README.md#install) for the full install matrix.

```bash
# Homebrew
brew install salar/runnerkit/runnerkit

# Or download from GitHub Releases and verify with cosign:
TAG=v1.0.0
cosign verify-blob \
  --bundle  runnerkit_${TAG#v}_checksums.txt.sigstore.json \
  --certificate-identity   "https://github.com/salar/runnerkit/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  runnerkit_${TAG#v}_checksums.txt
```

## 10-Minute Stopwatch (D-13)

Measured by the maintainer on a clean machine before tagging.

| Path | Wall-clock | Notes |
|---|---|---|
| BYO persistent | `____ s` | (maintainer fills) |
| Hetzner cloud persistent | `____ s` | Hetzner cost: `__.__ EUR` |

## Outstanding Live Smokes Closed

- Phase 1: live GitHub permission smoke — closed by `make smoke-live-byo`. STATE.md note resolved.
- Phase 4: live Hetzner billable smoke — closed by `make smoke-live-cloud` with D-12 gate 1 (empty-project precheck) and D-12 gate 2 (destroy-verify 404 polling within 300s timeout). STATE.md note resolved.

## Upgrade Path

This is the first release; nothing to upgrade from. Future releases follow
[docs/upgrade.md](docs/upgrade.md):

- CLI: `runnerkit upgrade` prints the right command for your install channel
  (Homebrew or Releases binary). Does NOT self-replace.
- Bundled runner: `runnerkit upgrade-runner` re-applies bootstrap with the new
  pin. Refuses without `--force` when an ephemeral runner is currently
  waiting/busy.
- State: forward-only auto migrations with side-by-side backup
  (`state.json.backup-vN-<timestamp>`). Refuses to mutate on newer schema with
  exit code 7.

## Troubleshooting

If the CLI prints a `RKD-<COMPONENT>-NNN` code, see
[docs/troubleshooting/](docs/troubleshooting/README.md). Override the URL
prefix with `RUNNERKIT_DOCS_BASE`.

## Acknowledgements

Built end-to-end through the GSD planning + execution workflow.
```
  </action>
  <verify>
    <automated>grep -q "Stopwatch Checklist" docs/release-process.md && grep -q "BYO path" docs/release-process.md && grep -q "Hetzner cloud path" docs/release-process.md && grep -q "10-minute" docs/release-process.md && test -f .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md && grep -q "BYO smoke" .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md && grep -q "Hetzner smoke" .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md && grep -q "D-12 gate 1" .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md && grep -q "D-12 gate 2" .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md && test -f RELEASE-NOTES-v1.0.0.md && grep -q "RunnerKit v1.0.0" RELEASE-NOTES-v1.0.0.md && grep -q "2.334.0" RELEASE-NOTES-v1.0.0.md && grep -q "cosign verify-blob" RELEASE-NOTES-v1.0.0.md && grep -q "make smoke-live-byo" RELEASE-NOTES-v1.0.0.md && grep -q "make smoke-live-cloud" RELEASE-NOTES-v1.0.0.md</automated>
  </verify>
  <acceptance_criteria>
    - `docs/release-process.md` (already exists from Plan 06-01) now includes a "Stopwatch Checklist (D-13)" section with BYO and Hetzner stopwatch tables having T0/T_now/Δ columns and a Total wall-clock line.
    - `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md` exists with frontmatter (`phase`, `type: verification`, `status: pending`, `created`).
    - `06-VERIFICATION.md` lists: automated test checklist (8 entries — full suite + GoReleaser checks + 5 errcodes + 4 migration + 6 update + 7 upgrade tests), BYO smoke fields (host, repo, duration, runner ID), Hetzner smoke fields (repo, project, duration, cost, 5 resource IDs, D-12 gate 1 status, D-12 gate 2 status, precheck final size=0), 10-minute stopwatch totals, bundled versions table.
    - `RELEASE-NOTES-v1.0.0.md` exists at repo root with: title `RunnerKit v1.0.0`, supported platforms (4), Homebrew install command, cosign verify-blob snippet (matching README), 10-minute stopwatch table with placeholder rows, "Outstanding Live Smokes Closed" section explicitly mentioning Phase 1 and Phase 4 STATE.md notes, upgrade path summary, troubleshooting forward link.
    - Bundled runner pin `2.334.0` is referenced in RELEASE-NOTES-v1.0.0.md.
    - All validation matrix rows for D-13 (line 78) are wired.
  </acceptance_criteria>
  <done>docs/release-process.md has the 10-minute Stopwatch Checklist appended; 06-VERIFICATION.md baseline file exists with full automated + manual checklist for v1.0.0; RELEASE-NOTES-v1.0.0.md template exists with all required sections and explicit Phase 1/Phase 4 closure notes.</done>
</task>

<task type="checkpoint:human-action" gate="blocking">
  <name>Task 4: Maintainer runs `make smoke-live` and fills 06-VERIFICATION.md + RELEASE-NOTES-v1.0.0.md durations</name>
  <files>.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md, RELEASE-NOTES-v1.0.0.md</files>
  <action>See `<what-built>` and `<how-to-verify>` below for the full 9-step maintainer procedure. Summary: (1) verify Plan 06-01 prerequisites (tap repo + HOMEBREW_TAP_GITHUB_TOKEN); (2) export RUNNERKIT_SMOKE_BYO_HOST, RUNNERKIT_SMOKE_REPO, HCLOUD_TOKEN; (3) verify Hetzner project empty; (4) `time make smoke-live 2>&1 | tee smoke-output.log`; (5) run 10-minute stopwatch on a CLEAN machine following docs/release-process.md Stopwatch Checklist; (6) fill 06-VERIFICATION.md (8 automated test ticks + BYO smoke fields + Hetzner smoke fields including 5 resource IDs + D-12 gates 1+2 status + cost + stopwatch totals + maintainer signature); (7) fill RELEASE-NOTES-v1.0.0.md durations + cost; (8) re-verify Hetzner project empty post-smoke; (9) commit both files. Maintainer cannot delegate this to Claude because it requires real `gh auth login`, real billable Hetzner resource creation, and a real human stopwatch on a clean laptop.</action>
  <verify><automated>echo "checkpoint:human-action — verified by maintainer resume-signal (smoke-green / smoke-red), not automation"</automated></verify>
  <done>06-VERIFICATION.md committed with all 8 automated checklist items ticked AND BYO+Hetzner smoke fields filled (host, repo, duration, runner ID, project, resource IDs, cost, D-12 gate 1 PASS, D-12 gate 2 PASS, stopwatch totals, maintainer signature). RELEASE-NOTES-v1.0.0.md committed with wall-clock durations and Hetzner cost. Maintainer typed "smoke-green" (or "smoke-red <reason>" triggering gap-closure plan).</done>
  <what-built>Plans 06-01..06-03 produced the release pipeline, upgrade flow, troubleshooting docs, and RKD-coded CLI emit sites. Tasks 1-3 of THIS plan produced the live smoke harness (Makefile targets, smoke binaries, shell wrappers, stopwatch checklist) and skeleton verification + release-notes files. The remaining gate is the maintainer's manual `make smoke-live` run on a real machine with real GitHub auth and real Hetzner project — Claude cannot execute this because (a) it requires `gh auth login` against a real account, (b) it creates billable Hetzner resources, (c) the wall-clock stopwatch needs a human running the steps on a clean laptop. This is the explicit "we have run out of fakes" boundary per Phase 6 RESEARCH §"Fake/Real Boundary".</what-built>
  <how-to-verify>
    Maintainer steps (estimated 30-45 minutes total):

    1. **Resolve Plan 06-01 prerequisites if not already done:** Verify `salar/homebrew-runnerkit` repo exists, default branch `main`, `Casks/` directory present. Verify `HOMEBREW_TAP_GITHUB_TOKEN` is in `salar/runnerkit` repo secrets. (These are the deferred items from Plan 06-01 Task 5.)

    2. **Set up environment:**
       ```bash
       export RUNNERKIT_SMOKE_BYO_HOST=user@maintainer-host
       export RUNNERKIT_SMOKE_REPO=salar/runnerkit-smoke-test   # or any maintainer-controlled trusted repo
       export HCLOUD_TOKEN=<from Hetzner project: Security → API tokens → Read & Write>
       gh auth login                                            # if not already
       ```

    3. **Verify Hetzner project is empty:** Open <https://console.hetzner.cloud/projects> for the project the HCLOUD_TOKEN belongs to. Confirm zero servers, zero ssh-keys named `runnerkit-*`, zero firewalls named `runnerkit-*`. (The empty-precheck D-12 gate 1 will refuse if any exist.)

    4. **Run `make smoke-live`:**
       ```bash
       cd /path/to/spool
       time make smoke-live 2>&1 | tee smoke-output.log
       ```
       This runs BYO smoke, Hetzner smoke (with D-12 gates), and prints the stopwatch reminder. Watch for:
       - BYO smoke prints `[smoke-byo] BYO_DURATION_SECONDS=NNN` on success.
       - Empty-precheck prints `OK — no orphan runnerkit-* resources` and exits 0.
       - Cloud-end-to-end prints `[smoke-cloud] CLOUD_DURATION_SECONDS=NNN` on success.
       - Destroy-verify prints `OK — every saved resource ID returns 404` and exits 0.

    5. **Run the 10-minute stopwatch on a CLEAN machine** (a fresh VM or a different laptop, NOT the development machine). Follow `docs/release-process.md` Stopwatch Checklist for both BYO and Hetzner paths. Record T0/T_now/Δ for each step.

    6. **Fill in `06-VERIFICATION.md`** at `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md`:
       - Tick all 8 automated test checklist items (after running `go test ./... -count=1 -race`, `goreleaser check`, `goreleaser release --snapshot --skip=publish --clean`).
       - Fill BYO smoke section: host, repo, duration, runner ID.
       - Fill Hetzner smoke section: repo, project, duration, cost (from Hetzner project dashboard 24h after the smoke), 5 resource IDs (from `state.json.backup-*` file or smoke output), D-12 gate 1 PASS, D-12 gate 2 PASS, precheck final size 0.
       - Fill 10-minute stopwatch totals.
       - Sign and date at the bottom.

    7. **Fill in `RELEASE-NOTES-v1.0.0.md`** with the same wall-clock numbers and Hetzner cost.

    8. **Verify Hetzner project is empty AGAIN** (post-smoke). The destroy-verify gate already proved it via API, but eyeball the Console as belt-and-suspenders.

    9. **Resume signal:** When `06-VERIFICATION.md` is filled in and committed AND `RELEASE-NOTES-v1.0.0.md` is filled in, type "smoke-green" to indicate the phase is complete and ready to tag. If any gate failed, type "smoke-red <reason>" and the plan stays open for a /gsd:plan-phase --gaps follow-up.
  </how-to-verify>
  <resume-signal>Type "smoke-green" when both files are filled in and committed. Type "smoke-red <reason>" if any gate failed (D-12 gate 1 or 2, or 10-min target exceeded) — this triggers a gap-closure plan.</resume-signal>
</task>

</tasks>

<verification>
Phase-level checks for Plan 06-04 completion:

1. `Makefile` exists with `smoke-live`, `smoke-live-byo`, `smoke-live-cloud`, `smoke-stopwatch` targets.
2. `cmd/_smokebin/empty_precheck/` and `cmd/_smokebin/destroy_verify/` Go programs exist with passing unit tests.
3. `scripts/smoke/*.sh` shell scripts exist, executable, syntactically valid (`bash -n`).
4. `docs/release-process.md` includes "Stopwatch Checklist (D-13)" section.
5. `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md` exists as skeleton.
6. `RELEASE-NOTES-v1.0.0.md` exists at repo root as template.
7. No `.github/workflows/*.yml` references `make smoke-live` or `smoke-live-` (D-11 verified).
8. The maintainer human-action checkpoint resolves to `smoke-green` (phase complete) or `smoke-red <reason>` (gap closure required).
9. Tests `TestEmptyPrecheck_RefusesOnExisting`, `TestEmptyPrecheck_AllResourceTypes`, `TestDestroyVerify_Timeout` all pass.

Validation matrix coverage (`06-VALIDATION.md`):
- Line 74 (Live BYO smoke): closed by checkpoint resolution, scaffold by Tasks 1-3.
- Line 75 (Live Hetzner smoke): closed by checkpoint, scaffold by Tasks 1-3.
- Line 76 (D-12 gate 1 unit + live): unit test green from Task 2; live closed by checkpoint.
- Line 77 (D-12 gate 2 unit + live): unit test green from Task 2; live closed by checkpoint.
- Line 78 (10-min stopwatch checklist): scaffolded by Task 3; closed by checkpoint.
</verification>

<success_criteria>
- Makefile + 4 shell scripts in `scripts/smoke/` produce a working `make smoke-live` orchestration with env-var preconditions, trap-on-EXIT cleanup (Pitfall 7), and the D-12 gate 1 + gate 2 invocation order (precheck → smoke → verify).
- `cmd/_smokebin/empty_precheck` and `destroy_verify` Go programs exist with hcloud-go v1.59.2 integration; 3 unit tests green covering both gates and all 4 Hetzner resource types.
- `docs/release-process.md` has the appended Stopwatch Checklist; `06-VERIFICATION.md` baseline + `RELEASE-NOTES-v1.0.0.md` template exist with all required sections.
- No CI workflow file references `smoke-live*` (D-11 enforced).
- Maintainer human-action checkpoint resolves to `smoke-green` (or `smoke-red <reason>` triggers gap closure).
- Closes Phase 1 and Phase 4 STATE.md outstanding live-smoke notes.
- All hard rules from `<phase_specific_guidance>` Hard rules 8, 9, 11 are satisfied.
- Validation matrix rows 74-78 are wired (unit tests green from Tasks 1-2; live rows green at checkpoint resolution).
</success_criteria>

<output>
After completion, create `.planning/phases/06-release-upgrade-docs-and-v1-validation/06-04-SUMMARY.md` summarizing:
- Files added (`Makefile`, `cmd/_smokebin/{empty_precheck, destroy_verify}/main.go`+tests, 4 `scripts/smoke/*.sh`, `06-VERIFICATION.md`, `RELEASE-NOTES-v1.0.0.md`).
- Files modified (`docs/release-process.md` — Stopwatch Checklist appended).
- Locked decisions implemented (D-10, D-11, D-12 gates 1+2, D-13).
- Validation matrix rows wired (lines 74-78 of `06-VALIDATION.md`).
- Outstanding live smokes from Phase 1 and Phase 4 STATE.md Blockers/Concerns now have executable harnesses; closure status depends on the maintainer checkpoint resolution.
- Files NOT modified by this plan but updated by maintainer in checkpoint: `06-VERIFICATION.md` (durations, runner ID, resource IDs, cost), `RELEASE-NOTES-v1.0.0.md` (wall-clock numbers).
- Next step after `smoke-green`: run `/gsd:verify-work` for Phase 6 sign-off; then push `git tag -a v1.0.0` per `docs/release-process.md`.
</output>
