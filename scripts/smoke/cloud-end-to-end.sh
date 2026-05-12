#!/usr/bin/env bash
# scripts/smoke/cloud-end-to-end.sh
# Phase 4 outstanding live Hetzner billable-resource smoke.
# Args: $1 = owner/repo
# Pre: empty-project precheck has already run (called by Makefile).
# Post: hetzner-destroy-verify.sh runs after this exits (called by Makefile).
#
# Closes: STATE.md "Phase 4 validation note: a controlled live Hetzner
# smoke remains recommended before public release because it creates
# billable resources and needs real credentials."
#
# Pitfall 7 mitigation: a trap on EXIT/INT/TERM runs `runnerkit destroy
# --yes` BEFORE removing the state dir, so a Ctrl-C mid-smoke does not
# leave billable Hetzner resources behind.
set -euo pipefail

REPO="${1:?repo required}"
: "${HCLOUD_TOKEN:?HCLOUD_TOKEN required}"
: "${RUNNERKIT_SMOKE_STATE_DIR:?RUNNERKIT_SMOKE_STATE_DIR required (set by Makefile)}"

# Use the maintainer-isolated tempdir set by the Makefile target so the
# destroy-verify script can read state.json after this returns.
export RUNNERKIT_STATE_DIR="${RUNNERKIT_SMOKE_STATE_DIR}"
SMOKE_DIR="${RUNNERKIT_SMOKE_STATE_DIR}"

cleanup() {
    rc=$?
    echo "===> [smoke-cloud] cleanup trap fired (rc=$rc)"
    if [ -f "${SMOKE_DIR}/state.json" ]; then
        # Best-effort destroy. If it fails, the maintainer must verify
        # by hand via the Hetzner Console.
        go run ./cmd/runnerkit destroy --repo "${REPO}" --yes \
            || echo "[smoke-cloud] WARN: destroy --yes failed during trap; check Hetzner Console manually"
    fi
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

echo "===> [smoke-cloud] doctor JSON contract (Phase 7: host_incident_hints + --deep)"
./scripts/smoke/assert-doctor-json-contract.sh "${REPO}"

# Snapshot the state file before destroy mutates/removes it. The
# destroy-verify wrapper reads state-after-destroy.json to recover the
# saved cloud IDs in case `runnerkit destroy --yes` removed the repo
# entry from state.json on success.
cp "${SMOKE_DIR}/state.json" "${SMOKE_DIR}/state-after-destroy.json" 2>/dev/null || true

echo "===> [smoke-cloud] runnerkit destroy --repo ${REPO} --yes"
go run ./cmd/runnerkit destroy --repo "${REPO}" --yes

END_EPOCH=$(date +%s)
DURATION=$((END_EPOCH - START_EPOCH))
echo "===> [smoke-cloud] OK — duration: ${DURATION}s"
echo "[smoke-cloud] CLOUD_DURATION_SECONDS=${DURATION}" >&2

# Disarm the on-success trap so the Makefile can run hetzner-destroy-verify.sh
# next without a redundant destroy invocation. INT/TERM still purge state on
# signal so an abort during the verify phase does not leak the tempdir.
trap - EXIT
trap 'rc=$?; rm -rf "${SMOKE_DIR}"; exit $rc' INT TERM
