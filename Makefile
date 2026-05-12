# RunnerKit Makefile — solo developer + Claude execution.
# Live smoke targets are MAINTAINER-ONLY and must NOT be invoked from CI (D-11).

.PHONY: help test test-race test-integration vet lint smoke-live smoke-live-byo smoke-live-cloud smoke-stopwatch release-snapshot

help: ## Show this help.
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-22s %s\n", $$1, $$2}'

test: ## Run the unit/integration test suite.
	go test ./... -count=1

test-race: ## Run the full test suite with -race.
	go test ./... -count=1 -race

test-integration: ## Run real-shell integration tests (requires NOPASSWD sudo on local machine; gated by RUNNERKIT_INTEGRATION=1).
	RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -v

vet: ## go vet all packages.
	go vet ./...

release-snapshot: ## Local GoReleaser dry-run (validates the build matrix).
	goreleaser release --snapshot --skip=publish --clean --skip=sign

# ---------- LIVE SMOKES (maintainer-only — D-11) ----------
# These targets create REAL GitHub registrations and REAL Hetzner billable
# resources. Do NOT wire into any GitHub Actions workflow. Per D-11 the
# live smokes must be invoked manually by the maintainer before tagging
# a release; they are NOT scheduled, NOT triggered on tag push, and the
# CI environment must NOT hold the real GitHub PAT or HCLOUD_TOKEN
# secrets these targets depend on.

smoke-live: smoke-live-byo smoke-live-cloud smoke-stopwatch ## Run all live smokes (BYO + Hetzner + 10-minute stopwatch). Maintainer-only.

smoke-live-byo: ## Phase 1 outstanding: live GitHub permission smoke. Requires RUNNERKIT_SMOKE_BYO_HOST and RUNNERKIT_SMOKE_REPO.
	@test -n "$$RUNNERKIT_SMOKE_BYO_HOST" || { echo "RUNNERKIT_SMOKE_BYO_HOST=user@host required"; exit 2; }
	@test -n "$$RUNNERKIT_SMOKE_REPO"     || { echo "RUNNERKIT_SMOKE_REPO=owner/name required (must be a maintainer-controlled trusted repo)"; exit 2; }
	@command -v gh >/dev/null || { echo "gh CLI not installed; install from https://cli.github.com/"; exit 2; }
	@gh auth status >/dev/null 2>&1 || { echo "gh auth not present; run 'gh auth login' first"; exit 2; }
	@command -v python3 >/dev/null || { echo "python3 required for scripts/smoke/assert-doctor-json-contract.sh"; exit 2; }
	./scripts/smoke/byo-permission.sh "$$RUNNERKIT_SMOKE_REPO" "$$RUNNERKIT_SMOKE_BYO_HOST"

smoke-live-cloud: ## Phase 4 outstanding: live Hetzner end-to-end smoke. CREATES BILLABLE RESOURCES. Requires HCLOUD_TOKEN and RUNNERKIT_SMOKE_REPO.
	@test -n "$$HCLOUD_TOKEN"         || { echo "HCLOUD_TOKEN required"; exit 2; }
	@test -n "$$RUNNERKIT_SMOKE_REPO" || { echo "RUNNERKIT_SMOKE_REPO=owner/name required (maintainer-controlled trusted repo, NOT public)"; exit 2; }
	@command -v gh >/dev/null || { echo "gh CLI not installed"; exit 2; }
	@gh auth status >/dev/null 2>&1 || { echo "gh auth not present"; exit 2; }
	@command -v python3 >/dev/null || { echo "python3 required for scripts/smoke/assert-doctor-json-contract.sh"; exit 2; }
	@RUNNERKIT_SMOKE_STATE_DIR="$$(mktemp -d -t runnerkit-smoke-cloud-XXXXXXXX)" && \
		export RUNNERKIT_SMOKE_STATE_DIR && \
		echo "[smoke-cloud] state dir: $$RUNNERKIT_SMOKE_STATE_DIR" && \
		./scripts/smoke/hetzner-empty-precheck.sh && \
		./scripts/smoke/cloud-end-to-end.sh "$$RUNNERKIT_SMOKE_REPO" && \
		./scripts/smoke/hetzner-destroy-verify.sh "$${RUNNERKIT_SMOKE_TIMEOUT:-300}" && \
		rm -rf "$$RUNNERKIT_SMOKE_STATE_DIR"

smoke-stopwatch: ## 10-minute stopwatch checklist (D-13). Maintainer manually records into RELEASE-NOTES-vX.Y.Z.md.
	@echo "Open docs/release-process.md '## Stopwatch Checklist' and follow the BYO + Hetzner end-to-end timing."
	@echo "Record durations into RELEASE-NOTES-v$${VER:-1.0.0}.md and .planning/phases/06-release-upgrade-docs-and-v1-validation/06-VERIFICATION.md."
