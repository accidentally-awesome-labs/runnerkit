#!/usr/bin/env bash
# scripts/testkit/lib.sh: helpers shared by the release-gate and validation
# scripts in this directory. Source it; do not run it.
#
# Needs bash 3.2 or newer (the macOS default works), curl 7.55+, jq and, for
# host checks, ssh. GitHub token: GH_TOKEN, else GITHUB_TOKEN, else
# `gh auth token`. Use a fine-grained PAT limited to the throwaway test
# repository (docs/testkit/README.md lists its permissions). Tokens go to
# curl through a 0600 header file, never on a command line.
#
# Environment:
#   GITHUB_API_URL    API base (default https://api.github.com)
#   RK_TESTKIT_OUT    evidence directory (default ~/runnerkit-testkit)
#   RK_POLL_SECONDS   polling interval (default 10)
#   RUNNERKIT_BIN     the RunnerKit binary under test (default: runnerkit)

set -euo pipefail

RK_API="${GITHUB_API_URL:-https://api.github.com}"
RK_API="${RK_API%/}"
RK_POLL_SECONDS="${RK_POLL_SECONDS:-10}"
RK_TMP="$(mktemp -d "${TMPDIR:-/tmp}/rk-testkit.XXXXXX")"
RK_BODY="$RK_TMP/body"
RK_HEADERS="$RK_TMP/headers"
RK_AUTH_FILE=""
RK_STATUS=""
RK_REPO=""
RK_DEFAULT_BRANCH=""
RK_SSH_TARGET=""
RK_SSH_PORT=""
RK_SSH_KEY=""
RK_EXIT_HOOKS=""

rk__exit() {
	local status=$? hook
	if [ -n "$RK_EXIT_HOOKS" ]; then
		while IFS= read -r hook; do
			[ -n "$hook" ] || continue
			# </dev/null: a hook that runs ssh must not eat the other hooks.
			eval "$hook" </dev/null || true
		done <<EOF
$RK_EXIT_HOOKS
EOF
	fi
	rm -rf "$RK_TMP"
	exit "$status"
}
trap rk__exit EXIT

# rk_at_exit CMD: run CMD (a shell string) when the script exits, first
# registered first.
rk_at_exit() {
	RK_EXIT_HOOKS="$RK_EXIT_HOOKS
$1"
}

rk_die() {
	printf 'FAIL: %s\n' "$*" >&2
	exit 1
}
rk_info() { printf '==> %s\n' "$*" >&2; }
rk_warn() { printf 'WARN: %s\n' "$*" >&2; }

rk_need() {
	local c
	for c in "$@"; do
		command -v "$c" >/dev/null 2>&1 || rk_die "missing required command: $c"
	done
}

# rk_use_token LABEL TOKEN: later rk_api calls authenticate with TOKEN.
rk_use_token() {
	local file
	file="$RK_TMP/auth.$(printf '%s' "$1" | tr -c 'A-Za-z0-9_-' '_')"
	(
		umask 077
		printf 'Authorization: Bearer %s\n' "$2" >"$file"
	)
	RK_AUTH_FILE="$file"
}

# rk_default_token: authenticate with GH_TOKEN, GITHUB_TOKEN or gh.
rk_default_token() {
	local t="${GH_TOKEN:-${GITHUB_TOKEN:-}}"
	if [ -z "$t" ] && command -v gh >/dev/null 2>&1; then
		t="$(gh auth token 2>/dev/null || true)"
	fi
	[ -n "$t" ] || rk_die "no GitHub token: export GH_TOKEN (a fine-grained PAT limited to the test repository)"
	rk_use_token default "$t"
}

# rk_read_secret PROMPT: reads a value from the terminal without echo.
rk_read_secret() {
	local v=""
	printf '%s' "$1" >/dev/tty
	IFS= read -rs v </dev/tty || rk_die "could not read from the terminal"
	printf '\n' >/dev/tty
	printf '%s' "$v"
}

# rk_api METHOD PATH [JSON]: body in $RK_BODY, response headers in
# $RK_HEADERS, HTTP status in $RK_STATUS ("000" when curl itself failed).
rk_api() {
	local method="$1" path="$2" data="${3-}"
	local -a args
	[ -n "$RK_AUTH_FILE" ] || rk_die "internal: no token selected"
	args=(-sS -X "$method" -o "$RK_BODY" -D "$RK_HEADERS" -w '%{http_code}'
		-H @"$RK_AUTH_FILE"
		-H 'Accept: application/vnd.github+json'
		-H 'X-GitHub-Api-Version: 2022-11-28')
	if [ -n "$data" ]; then
		args+=(-H 'Content-Type: application/json' --data-binary "$data")
	fi
	: >"$RK_BODY"
	: >"$RK_HEADERS"
	RK_STATUS="$(curl "${args[@]}" "$RK_API$path" 2>"$RK_TMP/curl.err")" || RK_STATUS="000"
}

# rk_api_message: the "message" field of the last response, if any.
rk_api_message() {
	jq -r '.message // empty' "$RK_BODY" 2>/dev/null || true
}

# rk_accepted_permissions: the X-Accepted-GitHub-Permissions header of the
# last response ("-" when absent).
rk_accepted_permissions() {
	local v
	v="$(grep -i '^x-accepted-github-permissions:' "$RK_HEADERS" 2>/dev/null | head -n1 | cut -d: -f2- | tr -d '\r' | sed 's/^ *//' || true)"
	printf '%s' "${v:--}"
}

rk_set_repo() {
	case "$1" in
	*/*/* | /* | */) rk_die "--repo must be owner/name, got: $1" ;;
	*/*) RK_REPO="$1" ;;
	*) rk_die "--repo must be owner/name, got: $1" ;;
	esac
}

# rk_require_private_repo: stops unless $RK_REPO is a private repository.
rk_require_private_repo() {
	rk_api GET "/repos/$RK_REPO"
	case "$RK_STATUS" in
	200) ;;
	401) rk_die "GitHub rejected the token (HTTP 401)" ;;
	404) rk_die "$RK_REPO not found, or the token cannot see it (HTTP 404)" ;;
	*) rk_die "GET /repos/$RK_REPO returned HTTP $RK_STATUS $(rk_api_message)" ;;
	esac
	if [ "$(jq -r '.private' "$RK_BODY")" != "true" ]; then
		rk_die "$RK_REPO is not private. Use a throwaway PRIVATE repository: self-hosted runners must never serve public repositories."
	fi
	RK_DEFAULT_BRANCH="$(jq -r '.default_branch' "$RK_BODY")"
}

# rk_require_workflow FILE: stops unless .github/workflows/FILE is active.
rk_require_workflow() {
	rk_api GET "/repos/$RK_REPO/actions/workflows/$1"
	if [ "$RK_STATUS" != 200 ]; then
		rk_die "workflow $1 not found in $RK_REPO (HTTP $RK_STATUS). Copy scripts/testkit/workflows/$1 to .github/workflows/ on the default branch and push."
	fi
	if [ "$(jq -r '.state' "$RK_BODY")" != active ]; then
		rk_die "workflow $1 is $(jq -r '.state' "$RK_BODY") in $RK_REPO; enable it in the Actions tab"
	fi
}

rk_new_nonce() {
	printf 'rk-%s-%s%s' "$(date -u +%Y%m%dT%H%M%SZ)" "$$" "$RANDOM"
}

# rk_dispatch FILE INPUTS_JSON NONCE: dispatches the workflow on the default
# branch (or $RK_REF) and prints the run id. The workflows put the nonce in
# their run-name, so the run can be found when the API returns no run id.
rk_dispatch() {
	local wf="$1" inputs="$2" nonce="$3" ref body i id
	ref="${RK_REF:-$RK_DEFAULT_BRANCH}"
	[ -n "$ref" ] || rk_die "internal: default branch unknown (call rk_require_private_repo first)"
	body="$(jq -nc --arg ref "$ref" --argjson inputs "$inputs" '{ref: $ref, inputs: $inputs, return_run_details: true}')"
	rk_api POST "/repos/$RK_REPO/actions/workflows/$wf/dispatches" "$body"
	if [ "$RK_STATUS" = 422 ] && grep -q return_run_details "$RK_BODY"; then
		body="$(jq -nc --arg ref "$ref" --argjson inputs "$inputs" '{ref: $ref, inputs: $inputs}')"
		rk_api POST "/repos/$RK_REPO/actions/workflows/$wf/dispatches" "$body"
	fi
	case "$RK_STATUS" in
	200)
		id="$(jq -r '.workflow_run_id // empty' "$RK_BODY")"
		if [ -n "$id" ]; then
			printf '%s\n' "$id"
			return 0
		fi
		;;
	204) ;;
	*) rk_die "dispatch $wf: HTTP $RK_STATUS $(rk_api_message) $(rk_accepted_permissions)" ;;
	esac
	i=0
	while [ "$i" -lt 30 ]; do
		rk_api GET "/repos/$RK_REPO/actions/workflows/$wf/runs?event=workflow_dispatch&per_page=30"
		if [ "$RK_STATUS" = 200 ]; then
			id="$(jq -r --arg n "$nonce" '[.workflow_runs[] | select((.display_title // "") | contains($n))][0].id // empty' "$RK_BODY")"
			if [ -n "$id" ]; then
				printf '%s\n' "$id"
				return 0
			fi
		fi
		i=$((i + 1))
		sleep 2
	done
	rk_die "dispatched $wf but found no run titled with $nonce"
}

# rk_run_field RUN_ID JQ_FILTER: prints a field of the workflow run.
rk_run_field() {
	rk_api GET "/repos/$RK_REPO/actions/runs/$1"
	[ "$RK_STATUS" = 200 ] || rk_die "GET run $1: HTTP $RK_STATUS $(rk_api_message)"
	jq -r "$2" "$RK_BODY"
}

# rk_wait_run RUN_ID [TIMEOUT_SECONDS]: waits for the run to complete and
# prints its conclusion, or "timeout".
rk_wait_run() {
	local id="$1" timeout="${2:-1800}" start="$SECONDS" status=""
	while :; do
		rk_api GET "/repos/$RK_REPO/actions/runs/$id"
		if [ "$RK_STATUS" = 200 ]; then
			status="$(jq -r '.status' "$RK_BODY")"
			if [ "$status" = completed ]; then
				jq -r '.conclusion // "none"' "$RK_BODY"
				return 0
			fi
		fi
		if [ $((SECONDS - start)) -ge "$timeout" ]; then
			printf 'timeout\n'
			return 0
		fi
		rk_info "run $id: ${status:-HTTP $RK_STATUS} ($((SECONDS - start))s)"
		sleep "$RK_POLL_SECONDS"
	done
}

# rk_run_jobs RUN_ID FILE: saves the run's jobs JSON.
rk_run_jobs() {
	rk_api GET "/repos/$RK_REPO/actions/runs/$1/jobs?per_page=100"
	[ "$RK_STATUS" = 200 ] || rk_die "GET jobs for run $1: HTTP $RK_STATUS $(rk_api_message)"
	cp "$RK_BODY" "$2"
}

# rk_job_log JOB_ID FILE: saves the job's log (retries while GitHub
# finishes uploading it). Returns non-zero when the log is unavailable.
rk_job_log() {
	local code i=0
	while [ "$i" -lt 6 ]; do
		code="$(curl -sS -L -o "$2" -w '%{http_code}' -H @"$RK_AUTH_FILE" \
			-H 'Accept: application/vnd.github+json' -H 'X-GitHub-Api-Version: 2022-11-28' \
			"$RK_API/repos/$RK_REPO/actions/jobs/$1/logs" 2>/dev/null)" || code=000
		[ "$code" = 200 ] && return 0
		i=$((i + 1))
		sleep 5
	done
	return 1
}

# rk_list_runners: the repository's runners JSON in $RK_BODY. Returns
# non-zero on an HTTP error.
rk_list_runners() {
	rk_api GET "/repos/$RK_REPO/actions/runners?per_page=100"
	[ "$RK_STATUS" = 200 ]
}

# rk_runner NAME: prints the runner object (compact JSON), or nothing.
rk_runner() {
	rk_list_runners || rk_die "list runners: HTTP $RK_STATUS $(rk_api_message) (the token needs Administration: read)"
	jq -c --arg n "$1" '[.runners[] | select(.name == $n)][0] // empty' "$RK_BODY"
}

# rk_wait_runner NAME STATUS [TIMEOUT_SECONDS]: waits until the runner's
# GitHub status is STATUS (online, offline or absent).
rk_wait_runner() {
	local name="$1" want="$2" timeout="${3:-300}" start="$SECONDS" got
	while :; do
		if rk_list_runners; then
			got="$(jq -r --arg n "$name" '[.runners[] | select(.name == $n)][0].status // "absent"' "$RK_BODY")"
		else
			got="HTTP $RK_STATUS"
		fi
		[ "$got" = "$want" ] && return 0
		[ $((SECONDS - start)) -lt "$timeout" ] || return 1
		rk_info "runner $name: $got, waiting for $want ($((SECONDS - start))s)"
		sleep "$RK_POLL_SECONDS"
	done
}

# rk_labels_json_lower JSON_ARRAY: lower-cases a JSON array of labels.
rk_labels_json_lower() {
	printf '%s' "$1" | jq -c 'map(ascii_downcase)'
}

# rk_set_host user@host[:port]
rk_set_host() {
	case "$1" in
	*@*:*)
		RK_SSH_TARGET="${1%:*}"
		RK_SSH_PORT="${1##*:}"
		;;
	*@*)
		RK_SSH_TARGET="$1"
		RK_SSH_PORT=""
		;;
	*) rk_die "--host must be user@host or user@host:port, got: $1" ;;
	esac
}

# rk_ssh ARGS...: runs a command on the host without a TTY. The host key
# must already be in ~/.ssh/known_hosts (you accepted it when you ran
# install.sh there).
rk_ssh() {
	local -a args
	[ -n "$RK_SSH_TARGET" ] || rk_die "internal: no --host"
	args=(-o BatchMode=yes -o ConnectTimeout=15)
	if [ -n "$RK_SSH_PORT" ]; then
		args+=(-p "$RK_SSH_PORT")
	fi
	if [ -n "$RK_SSH_KEY" ]; then
		args+=(-i "$RK_SSH_KEY")
	fi
	# shellcheck disable=SC2029 # callers pass commands for the host
	ssh "${args[@]}" "$RK_SSH_TARGET" "$@"
}

# rk_repo_dir: creates and prints the evidence directory of $RK_REPO.
rk_repo_dir() {
	local dir
	dir="${RK_TESTKIT_OUT:-$HOME/runnerkit-testkit}/$(printf '%s' "$RK_REPO" | tr '/' '-')"
	mkdir -p "$dir"
	printf '%s\n' "$dir"
}

# rk_evidence_dir LABEL: creates and prints a new evidence directory for
# one script run.
rk_evidence_dir() {
	local dir
	dir="$(rk_repo_dir)/$(date -u +%Y%m%dT%H%M%SZ)-$1"
	mkdir -p "$dir"
	printf '%s\n' "$dir"
}

rk_runnerkit() {
	"${RUNNERKIT_BIN:-runnerkit}" "$@"
}

# rk_latest_runner_version: the newest actions/runner release (e.g. 2.337.0),
# or nothing when the API call fails.
rk_latest_runner_version() {
	rk_api GET "/repos/actions/runner/releases/latest"
	if [ "$RK_STATUS" = 200 ]; then
		jq -r '.tag_name // empty' "$RK_BODY" | sed 's/^v//'
	fi
}

# rk_runner_pin: the actions/runner version this checkout's RunnerKit
# installs (RunnerVersion in internal/bootstrap/package.go), or nothing.
rk_runner_pin() {
	sed -n 's/^const RunnerVersion = "\([0-9][0-9.]*\)"$/\1/p' \
		"$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/internal/bootstrap/package.go"
}

# rk_sha256 FILE: the file's SHA-256 in hex (sha256sum on Linux, shasum on
# macOS).
rk_sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	else
		shasum -a 256 "$1" | cut -d' ' -f1
	fi
}

# rk_version_lt A B: true when version A sorts before version B.
rk_version_lt() {
	[ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "$1" "$2" | sort -t. -k1,1n -k2,2n -k3,3n | head -n1)" = "$1" ]
}
