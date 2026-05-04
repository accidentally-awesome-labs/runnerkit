#!/usr/bin/env bash
# scripts/smoke/hetzner-destroy-verify.sh
# D-12 gate 2: poll Hetzner for 404 on every saved resource ID.
# Args: $1 = timeout seconds (default 300).
#
# Reads cloud resource IDs from the state file written during the
# cloud-end-to-end smoke run; polls each ID until either every resource
# returns 404 (success) or the timeout elapses (loud failure).
set -euo pipefail
: "${HCLOUD_TOKEN:?HCLOUD_TOKEN required}"
: "${RUNNERKIT_SMOKE_STATE_DIR:?RUNNERKIT_SMOKE_STATE_DIR required (set by Makefile)}"

TIMEOUT="${1:-300}"
STATE_FILE="${RUNNERKIT_SMOKE_STATE_DIR}/state.json"

if [ ! -f "${STATE_FILE}" ]; then
    # destroy may have already removed state.json — try the sibling backup
    # written by cloud-end-to-end.sh before destroy executed.
    STATE_FILE="${RUNNERKIT_SMOKE_STATE_DIR}/state-after-destroy.json"
fi

echo "===> [smoke-verify] Polling Hetzner for 404 on saved IDs (timeout: ${TIMEOUT}s)"
RUNNERKIT_SMOKE_TIMEOUT="${TIMEOUT}" \
RUNNERKIT_SMOKE_STATE_FILE="${STATE_FILE}" \
    go run ./cmd/_smokebin/destroy_verify
echo "===> [smoke-verify] OK — every saved resource ID returns 404"
