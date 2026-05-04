#!/usr/bin/env bash
# scripts/smoke/hetzner-empty-precheck.sh
# D-12 gate 1: refuse if Hetzner project contains any runnerkit-* resource.
# This is a hard gate — orphans from a previous failed smoke must be cleaned
# up via `runnerkit destroy --yes` (or the Hetzner Console) before re-running.
set -euo pipefail
: "${HCLOUD_TOKEN:?HCLOUD_TOKEN required}"

echo "===> [smoke-precheck] Hetzner empty-project precheck"
go run ./cmd/_smokebin/empty_precheck
echo "===> [smoke-precheck] OK — no orphan runnerkit-* resources"
