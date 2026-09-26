#!/usr/bin/env bash
# scripts/testkit/token-probe.sh: validation V-2, the smallest token that can
# read runner health for a personal repository.
# Runbook: docs/testkit/validations.md.
#
# For every LABEL you give, the script asks for a token (input hidden) and
# calls, with that token only:
#   GET /repos/{repo}                                        Metadata: read
#   GET /repos/{repo}/actions/runners                        Administration: read
#   GET /repos/{repo}/actions/runners/deprecations/{version} Administration: read
#   GET /repos/{repo}/actions/runs?per_page=1                Actions: read
#   GET /repos/{repo}/actions/jobs/{id}/logs                 Actions: read
# and records the HTTP status, the X-Accepted-GitHub-Permissions header and,
# for the runner list, whether name, status, busy and version are populated.
# Labels `env` and `gh` use $GH_TOKEN and `gh auth token` without asking;
# RK_TOKEN_<LABEL> in the environment answers the prompt for LABEL.
# --in-workflow also dispatches rk-token-probe.yml to test the workflow's
# own GITHUB_TOKEN (and the RK_PROBE_PAT secret, if set).
# Tokens are never printed or written to disk outside a 0600 temp file.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=scripts/testkit/lib.sh
. "$HERE/lib.sh"

usage() {
	cat <<'EOF'
Usage: scripts/testkit/token-probe.sh --repo OWNER/NAME [--in-workflow] LABEL...

Example (three fine-grained PATs you created for the test repository):
  scripts/testkit/token-probe.sh --repo you/rk-gate --in-workflow \
    admin-read actions-read metadata-only

LABEL is a short name for a token you paste when asked, or that you put
in RK_TOKEN_<LABEL> (non-alphanumerics become _, e.g. RK_TOKEN_admin_read).
`env` uses $GH_TOKEN and `gh` uses `gh auth token`.
EOF
}

in_workflow=no
labels=""
while [ $# -gt 0 ]; do
	case "$1" in
	--repo)
		rk_set_repo "${2:?--repo needs a value}"
		shift 2
		;;
	--in-workflow)
		in_workflow=yes
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	-*)
		usage >&2
		exit 2
		;;
	*)
		labels="$labels $1"
		shift
		;;
	esac
done
if [ -z "$RK_REPO" ] || { [ -z "$labels" ] && [ "$in_workflow" = no ]; }; then
	usage >&2
	exit 2
fi

rk_need curl jq sed
rk_default_token
rk_require_private_repo
ev="$(rk_evidence_dir token-probe)"
report="$ev/token-probe.md"
rk_info "evidence directory: $ev"

# Facts the probes need, read with the default token: the newest runner
# versions to ask the deprecation endpoint about and one job id for the
# log probe.
versions="2.334.0"
if rk_list_runners; then
	versions="$(jq -r '[.runners[].version // empty] + ["2.334.0"] | unique | join(" ")' "$RK_BODY")"
fi
latest="$(rk_latest_runner_version || true)"
job_id=""
rk_api GET "/repos/$RK_REPO/actions/runs?per_page=5&status=completed"
if [ "$RK_STATUS" = 200 ]; then
	run_id="$(jq -r '.workflow_runs[0].id // empty' "$RK_BODY")"
	if [ -n "$run_id" ]; then
		rk_api GET "/repos/$RK_REPO/actions/runs/$run_id/jobs?per_page=1"
		if [ "$RK_STATUS" = 200 ]; then
			job_id="$(jq -r '.jobs[0].id // empty' "$RK_BODY")"
		fi
	fi
fi

{
	echo "# V-2 token probe: $RK_REPO"
	echo
	echo "- Date (UTC): $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo "- Latest actions/runner release: ${latest:-unknown}"
	echo "- Runner versions asked about: $versions"
	echo
	echo "| Token | Endpoint | HTTP | X-Accepted-GitHub-Permissions | Result |"
	echo "| --- | --- | --- | --- | --- |"
} >"$report"

row() { # token endpoint result
	# shellcheck disable=SC2016 # the backticks are Markdown
	printf '| %s | `%s` | %s | %s | %s |\n' "$1" "$2" "$RK_STATUS" "$(rk_accepted_permissions)" \
		"$(printf '%s' "$3" | tr '|\n' '/ ')" >>"$report"
}

# outcome: "ok" for a 2xx response, else the API's error message.
outcome() {
	case "$RK_STATUS" in
	2??) printf 'ok' ;;
	*) rk_api_message ;;
	esac
}

probe_token() {
	local label="$1" result version path
	rk_api GET "/repos/$RK_REPO"
	row "$label" "GET /repos/{repo}" "$(outcome)"

	rk_api GET "/repos/$RK_REPO/actions/runners?per_page=100"
	result="$(outcome)"
	if [ "$RK_STATUS" = 200 ]; then
		result="$(jq -r '[.runners[] | "\(.name): status=\(.status) busy=\(.busy) version=\(.version // "NULL")"] | if length == 0 then "no runners" else join("; ") end' "$RK_BODY")"
	fi
	row "$label" "GET /repos/{repo}/actions/runners" "$result"

	for version in $versions; do
		path="/repos/$RK_REPO/actions/runners/deprecations/$version"
		rk_api GET "$path"
		result="$(outcome)"
		if [ "$RK_STATUS" = 200 ]; then
			result="$(jq -r '"registration_deprecates_at=\(.registration_deprecates_at // "null") runtime_deprecates_at=\(.runtime_deprecates_at // "null")"' "$RK_BODY")"
		fi
		row "$label" "GET .../runners/deprecations/$version" "$result"
	done

	rk_api GET "/repos/$RK_REPO/actions/runs?per_page=1"
	row "$label" "GET /repos/{repo}/actions/runs" "$(outcome)"

	if [ -n "$job_id" ]; then
		RK_STATUS="$(curl -sS -L -o "$RK_TMP/log" -D "$RK_HEADERS" -w '%{http_code}' -H @"$RK_AUTH_FILE" \
			-H 'Accept: application/vnd.github+json' -H 'X-GitHub-Api-Version: 2022-11-28' \
			"$RK_API/repos/$RK_REPO/actions/jobs/$job_id/logs" 2>/dev/null)" || RK_STATUS=000
		if [ "$RK_STATUS" = 200 ]; then
			result="$(grep -o "Current runner version: '[^']*'" "$RK_TMP/log" | head -n1 || true)"
			result="${result:-no runner version line}"
		else
			result="$(jq -r '.message // empty' "$RK_TMP/log" 2>/dev/null || true)"
		fi
		row "$label" "GET /repos/{repo}/actions/jobs/{id}/logs" "$result"
	fi
}

for label in $labels; do
	case "$label" in
	env)
		[ -n "${GH_TOKEN:-}" ] || rk_die "label env: GH_TOKEN is not set"
		rk_use_token env "$GH_TOKEN"
		;;
	gh)
		rk_need gh
		rk_use_token gh "$(gh auth token)"
		;;
	*)
		var="RK_TOKEN_$(printf '%s' "$label" | tr -c 'A-Za-z0-9_' '_')"
		if [ -n "${!var:-}" ]; then
			rk_use_token "$label" "${!var}"
		else
			rk_use_token "$label" "$(rk_read_secret "Token for '$label' (input hidden): ")"
		fi
		;;
	esac
	rk_info "probing with $label"
	probe_token "$label"
done

if [ "$in_workflow" = yes ]; then
	rk_default_token
	rk_require_workflow rk-token-probe.yml
	nonce="$(rk_new_nonce)"
	run_id="$(rk_dispatch rk-token-probe.yml "$(jq -nc --arg n "$nonce" '{nonce: $n}')" "$nonce")"
	rk_info "rk-token-probe run: $(rk_run_field "$run_id" .html_url)"
	conclusion="$(rk_wait_run "$run_id" 600)"
	rk_run_jobs "$run_id" "$ev/workflow-jobs.json"
	job="$(jq -r '.jobs[0].id // empty' "$ev/workflow-jobs.json")"
	: >"$ev/workflow-job.log"
	if [ -n "$job" ]; then
		rk_job_log "$job" "$ev/workflow-job.log" || rk_warn "log for job $job not available"
	fi
	{
		echo
		echo "## Inside Actions (rk-token-probe run, conclusion: $conclusion)"
		echo
		echo '```text'
		grep -E '^[^ ]* RK-PROBE ' "$ev/workflow-job.log" | sed 's/^[^ ]* //' || true
		echo '```'
	} >>"$report"
fi

{
	echo
	echo "## Reading the table"
	echo
	echo "- The smallest token is the one with the fewest permissions whose runner list row shows status, busy and a version for every runner."
	echo "- A \`version=NULL\` means GitHub has no version for that runner (it never connected)."
	echo "- The deprecation rows are the dates after which GitHub stops registering, or stops sending jobs to, that runner version."
} >>"$report"

cat "$report"
rk_info "wrote $report"
