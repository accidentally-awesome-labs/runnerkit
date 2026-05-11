#!/usr/bin/env bash
# Smoke: bash syntax-check install.sh on common distro bases (requires Docker).
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
for img in ubuntu:22.04 ubuntu:24.04 debian:12 fedora:40; do
	echo "== ${img} =="
	docker run --rm -v "${root}/install.sh:/install.sh:ro" "${img}" bash -n /install.sh
done
echo "OK"
