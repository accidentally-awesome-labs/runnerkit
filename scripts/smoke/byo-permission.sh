#!/usr/bin/env bash
# scripts/smoke/byo-permission.sh
# Phase 1 outstanding live GitHub permission smoke.
# Args: $1 = owner/repo, $2 = user@host
#
# Closes: STATE.md "Plan 01-02/01-04 validation note: ... a controlled live
# GitHub permission smoke remains recommended before public release."
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

# Verify the bootstrap actually landed the runner tarball and extracted
# config.sh into the install dir on the remote host. This catches Bug 2
# (download_runner permission failure) before the test moves on to
# runner registration. See gap doc 06-GAP-byo-sudo-handling.md Task E.
echo "===> [smoke-byo] Asserting install dir contains config.sh on the remote host"
ssh "${HOST}" 'sudo test -f /opt/actions-runner/runnerkit-*/config.sh' || {
  echo "FAIL: config.sh not found in /opt/actions-runner/runnerkit-*/ — bootstrap did not land the tarball"
  exit 3
}

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
