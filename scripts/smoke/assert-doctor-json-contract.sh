#!/usr/bin/env bash
# scripts/smoke/assert-doctor-json-contract.sh
# Validates `runnerkit doctor --json` output includes stable keys required for
# tooling and Phase 7 fields (`host_incident_hints`). Maintainer smoke only (D-11).
#
# Usage: ./scripts/smoke/assert-doctor-json-contract.sh <owner/repo>
#
# Requires: repository cwd or git checkout root, python3, go.
# Honors RUNNERKIT_STATE_DIR when set (BYO/cloud smokes export it).
#
# Env:
#   RUNNERKIT_SMOKE_SKIP_DOCTOR_DEEP=1 — skip the second pass with `--deep`
#     (journal collection); default is to run both baseline and --deep.

set -euo pipefail

REPO="${1:?owner/repo required}"

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "${ROOT}" ]]; then
	echo "FAIL: run from a RunnerKit git checkout (git rev-parse --show-toplevel failed)" >&2
	exit 2
fi
cd "${ROOT}"

command -v python3 >/dev/null || {
	echo "FAIL: python3 required for doctor JSON assertions" >&2
	exit 2
}

verify_payload() {
	local json_file="$1"
	local mode_label="$2"
	python3 - "${json_file}" "${mode_label}" <<'PY'
import json, sys

path, mode = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as f:
	raw = f.read().strip()
if not raw:
	print("FAIL: empty stdout from doctor --json", file=sys.stderr)
	sys.exit(1)
d = json.loads(raw)
required = (
	"ok",
	"command",
	"repo",
	"state_path",
	"health",
	"findings",
	"next_actions",
	"host_incident_hints",
	"redactions_applied",
	"schema_version",
	"stage",
)
for k in required:
	assert k in d, "missing key %r (mode=%s); keys=%r" % (k, mode, list(d.keys()))
assert d["ok"] is True, d
assert d["command"] == "doctor", d
assert isinstance(d["findings"], list), type(d["findings"])
assert isinstance(d["host_incident_hints"], list), type(d["host_incident_hints"])
assert d["redactions_applied"] is True, d
print("doctor JSON contract OK (%s)" % mode)
PY
}

run_doctor_json() {
	local mode_label="$1"
	shift
	local tmp err_rc
	tmp="$(mktemp)"
	set +e
	go run ./cmd/runnerkit doctor --repo "${REPO}" --json --no-color "$@" >"${tmp}" 2>/tmp/runnerkit-smoke-doctor.stderr
	err_rc=$?
	set -e
	if [[ "${err_rc}" -ne 0 ]]; then
		echo "FAIL: doctor --json ${mode_label} exited ${err_rc}" >&2
		cat /tmp/runnerkit-smoke-doctor.stderr >&2 || true
		rm -f "${tmp}"
		exit "${err_rc}"
	fi
	verify_payload "${tmp}" "${mode_label}"
	rm -f "${tmp}"
}

echo "===> [smoke-doctor-json] baseline (--json)"
run_doctor_json "baseline"

if [[ -z "${RUNNERKIT_SMOKE_SKIP_DOCTOR_DEEP:-}" ]]; then
	echo "===> [smoke-doctor-json] Phase 7 journal path (--deep --json)"
	run_doctor_json "deep" --deep
else
	echo "===> [smoke-doctor-json] skipping --deep (RUNNERKIT_SMOKE_SKIP_DOCTOR_DEEP=1)"
fi

# Bug 33-C (smoke-discovery 2026-05-18): the error-path JSON envelope previously
# omitted schema_version/stage/next_actions/host_incident_hints when no state
# existed for the inferred repo. Assert the contract holds even when doctor
# refuses to run end-to-end.
verify_error_payload() {
	local json_file="$1"
	python3 - "${json_file}" <<'PY'
import json, sys
path = sys.argv[1]
with open(path, encoding="utf-8") as f:
	raw = f.read().strip()
if not raw:
	print("FAIL: empty stdout from doctor --json (error path)", file=sys.stderr)
	sys.exit(1)
d = json.loads(raw)
required = (
	"ok",
	"command",
	"error",
	"next_actions",
	"host_incident_hints",
	"redactions_applied",
	"schema_version",
	"stage",
)
for k in required:
	assert k in d, "error envelope missing key %r; keys=%r" % (k, list(d.keys()))
assert d["ok"] is False, d
assert d["command"] == "doctor", d
assert isinstance(d["error"], dict), type(d["error"])
assert isinstance(d["next_actions"], list), type(d["next_actions"])
assert isinstance(d["host_incident_hints"], list), type(d["host_incident_hints"])
assert d["redactions_applied"] is True, d
print("doctor JSON error-envelope contract OK (state_not_found)")
PY
}

echo "===> [smoke-doctor-json] error path (no state for nonexistent repo)"
NONEXISTENT_REPO="runnerkit-smoke/nonexistent-$(date +%s)-$$"
ERR_TMP="$(mktemp)"
set +e
go run ./cmd/runnerkit doctor --repo "${NONEXISTENT_REPO}" --json --no-color >"${ERR_TMP}" 2>/tmp/runnerkit-smoke-doctor-err.stderr
ERR_RC=$?
set -e
# Expect non-zero exit (state_not_found returns ExitStateIO == 5) but valid JSON.
if [[ "${ERR_RC}" -eq 0 ]]; then
	echo "FAIL: doctor --json for nonexistent state should exit non-zero (got 0)" >&2
	cat "${ERR_TMP}" >&2
	rm -f "${ERR_TMP}"
	exit 1
fi
verify_error_payload "${ERR_TMP}"
rm -f "${ERR_TMP}"
