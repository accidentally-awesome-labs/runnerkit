#!/usr/bin/env bash
# scripts/smoke/assert-list-host-repo-count.sh
# Asserts `runnerkit list --json` has exactly N repos under the canonical host
# bucket matching HOST (user@host or user@host:port; port defaults to 22).
#
# Usage: ./scripts/smoke/assert-list-host-repo-count.sh <expected_count> <user@host[:port]>
#
# Honors RUNNERKIT_STATE_DIR (required for isolated smoke state).

set -euo pipefail

WANT="${1:?expected repo count required}"
HOST_ARG="${2:?host pattern required}"

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "${ROOT}" ]]; then
	echo "FAIL: run from a RunnerKit git checkout" >&2
	exit 2
fi
cd "${ROOT}"

command -v python3 >/dev/null || {
	echo "FAIL: python3 required" >&2
	exit 2
}

OUT="$(mktemp)"
trap 'rm -f "${OUT}"' EXIT
go run ./cmd/runnerkit list --json --no-color >"${OUT}"

python3 - "${WANT}" "${HOST_ARG}" "${OUT}" <<'PY'
import json, sys

want = int(sys.argv[1])
host_arg = sys.argv[2].strip()
path = sys.argv[3]


def canon(h: str) -> str:
    if "@" not in h:
        return h
    user, rest = h.rsplit("@", 1)
    if ":" not in rest:
        return f"{user}@{rest}:22"
    return h


key = canon(host_arg)

with open(path, encoding="utf-8") as f:
    d = json.load(f)
if not d.get("ok"):
    print("FAIL: list --json ok=false", d, file=sys.stderr)
    sys.exit(1)
hosts = d.get("hosts")
if not isinstance(hosts, list):
    print("FAIL: hosts not a list", file=sys.stderr)
    sys.exit(1)
found = None
for h in hosts:
    if h.get("host_ref") == key:
        found = h
        break
if found is None:
    refs = [h.get("host_ref") for h in hosts]
    print("FAIL: no host bucket for", key, "have", refs, file=sys.stderr)
    sys.exit(1)
repos = found.get("repos")
if not isinstance(repos, list):
    print("FAIL: repos not a list", file=sys.stderr)
    sys.exit(1)
n = len(repos)
if n != want:
    print(f"FAIL: want {want} repos on {key}, got {n}: {repos!r}", file=sys.stderr)
    sys.exit(1)
print(f"list host repo count OK: {n} on {key}")
PY
