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
# Bug 18 (Plan 06-09, 2026-05-06): use plain `test -f` (no sudo). The
# install dir is mode 0755 and config.sh is mode 0755 (per
# RenderInstallScript's `sudo install -d -o <serviceUser>` + the
# tarball's preserved mode), so the SSH user can stat both without
# elevation. `sudo test -f` over a non-tty SSH session fails with
# "a terminal is required" because `test` is not in the byo-prepare
# scoped sudoers allowlist.
ssh "${HOST}" 'test -f /opt/actions-runner/runnerkit-*/config.sh' || {
  echo "FAIL: config.sh not found in /opt/actions-runner/runnerkit-*/ — bootstrap did not land the tarball"
  exit 3
}

# Verify the GitHub-runner registration sentinel `.runner` exists in the
# install dir — this is the file that `config.sh --unattended` writes on
# successful registration. Catches Bug 3 (register_runner runas mismatch)
# regression: if `runnerkit up` exits 0 but `.runner` is absent, the smoke
# fails with a distinct exit code so Plan 06-07 attempt-2 has a hard
# pass/fail signal beyond `runnerkit up exited 0`. See gap doc Task F.
echo "===> [smoke-byo] Asserting GitHub-runner registration sentinel .runner exists"
# Bug 18: same rationale as the config.sh assertion above. .runner is
# mode 0664 (world-readable) so plain `test -f` works.
ssh "${HOST}" 'test -f /opt/actions-runner/runnerkit-*/.runner' || {
  echo "FAIL: .runner sentinel not found in /opt/actions-runner/runnerkit-*/ — config.sh --unattended did not complete registration (Bug 3 regression?)"
  exit 4
}

echo "===> [smoke-byo] runnerkit status --repo ${REPO}"
go run ./cmd/runnerkit status --repo "${REPO}"

echo "===> [smoke-byo] runnerkit doctor --repo ${REPO}"
go run ./cmd/runnerkit doctor --repo "${REPO}" || true

echo "===> [smoke-byo] doctor JSON contract (Phase 7: host_incident_hints + --deep)"
./scripts/smoke/assert-doctor-json-contract.sh "${REPO}"

echo "===> [smoke-byo] list JSON contract (SEED-002)"
./scripts/smoke/assert-list-json-contract.sh

# Optional second repo on the same BYO host (SEED-002). Gated so default
# smoke stays single-repo / single-registration cost.
if [[ "${RUNNERKIT_SMOKE_MULTI_REPO:-}" == "1" ]]; then
	: "${RUNNERKIT_SMOKE_REPO2:?RUNNERKIT_SMOKE_MULTI_REPO=1 requires RUNNERKIT_SMOKE_REPO2=owner/other (trusted private repo, different from primary)}"
	if [[ "${RUNNERKIT_SMOKE_REPO2}" == "${REPO}" ]]; then
		echo "FAIL: RUNNERKIT_SMOKE_REPO2 must differ from primary RUNNERKIT_SMOKE_REPO / script arg repo" >&2
		exit 2
	fi
	echo "===> [smoke-byo] multi-repo: runnerkit register second repo ${RUNNERKIT_SMOKE_REPO2} on ${HOST}"
	go run ./cmd/runnerkit register --repo "${RUNNERKIT_SMOKE_REPO2}" --host "${HOST}" --mode persistent --yes
	echo "===> [smoke-byo] multi-repo: assert list shows two repos on this host"
	./scripts/smoke/assert-list-host-repo-count.sh 2 "${HOST}"
	echo "===> [smoke-byo] multi-repo: doctor JSON contract for second repo"
	./scripts/smoke/assert-doctor-json-contract.sh "${RUNNERKIT_SMOKE_REPO2}"
	echo "===> [smoke-byo] multi-repo: list JSON contract after second registration"
	./scripts/smoke/assert-list-json-contract.sh
fi

if [[ "${RUNNERKIT_SMOKE_MULTI_REPO:-}" == "1" ]]; then
	echo "===> [smoke-byo] runnerkit down second repo ${RUNNERKIT_SMOKE_REPO2} --yes"
	go run ./cmd/runnerkit down --repo "${RUNNERKIT_SMOKE_REPO2}" --yes
fi
echo "===> [smoke-byo] runnerkit down primary repo ${REPO} --yes"
go run ./cmd/runnerkit down --repo "${REPO}" --yes

END_EPOCH=$(date +%s)
DURATION=$((END_EPOCH - START_EPOCH))
echo "===> [smoke-byo] OK — duration: ${DURATION}s"
echo "[smoke-byo] BYO_DURATION_SECONDS=${DURATION}" >&2
