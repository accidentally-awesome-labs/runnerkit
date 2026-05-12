#!/usr/bin/env bash
# scripts/smoke/assert-list-json-contract.sh
# Validates `runnerkit list --json` includes stable keys for tooling (SEED-002).
#
# Usage: ./scripts/smoke/assert-list-json-contract.sh
#
# Requires: python3, go. Honors RUNNERKIT_STATE_DIR when set.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "${ROOT}" ]]; then
	echo "FAIL: run from a RunnerKit git checkout (git rev-parse --show-toplevel failed)" >&2
	exit 2
fi
cd "${ROOT}"

command -v python3 >/dev/null || {
	echo "FAIL: python3 required for list JSON assertions" >&2
	exit 2
}

tmp="$(mktemp)"
set +e
go run ./cmd/runnerkit list --json --no-color >"${tmp}" 2>/tmp/runnerkit-smoke-list.stderr
rc=$?
set -e
if [[ "${rc}" -ne 0 ]]; then
	echo "FAIL: list --json exited ${rc}" >&2
	cat /tmp/runnerkit-smoke-list.stderr >&2 || true
	rm -f "${tmp}"
	exit 1
fi

python3 - "${tmp}" <<'PY'
import json, sys
path = sys.argv[1]
with open(path, encoding="utf-8") as f:
	raw = f.read().strip()
if not raw:
	print("FAIL: empty stdout from list --json", file=sys.stderr)
	sys.exit(1)
d = json.loads(raw)
required = ("ok", "command", "schema_version", "state_path", "hosts", "next_actions")
for k in required:
	assert k in d, "missing key %r; keys=%r" % (k, list(d.keys()))
assert d["ok"] is True, d
assert d["command"] == "list", d
assert isinstance(d["hosts"], list), type(d["hosts"])
assert isinstance(d["next_actions"], list), type(d["next_actions"])
print("list JSON contract OK")
PY
rm -f "${tmp}"
