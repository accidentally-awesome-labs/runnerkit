#!/usr/bin/env bash
# scripts/testkit/offline-watch.sh: validation V-1, what GitHub does about a
# runner that stays offline past the 14-day auto-removal.
# Runbook: docs/testkit/validations.md.
#
# start  Run right after you stop the runner (or delete its host). Records
#        the runner's id, status and version and the start time; with
#        --queue-job it also dispatches rk-probe.yml at the runner's labels,
#        so a job waits for it (GitHub fails a job that stays queued for
#        24 hours; note whether that failure notifies you).
# check  Run on days 1, 13, 14, 15 and 16. Looks the runner up again,
#        appends one line to offline-watch.log and prints what to look for
#        in your email and GitHub notifications.
# State lives in ~/runnerkit-testkit/<owner>-<name>/offline-watch.json.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=scripts/testkit/lib.sh
. "$HERE/lib.sh"

usage() {
	cat <<'EOF'
Usage: scripts/testkit/offline-watch.sh --repo OWNER/NAME --runner NAME [--queue-job] start
       scripts/testkit/offline-watch.sh --repo OWNER/NAME check
EOF
}

action=""
runner_name=""
queue_job=no
while [ $# -gt 0 ]; do
	case "$1" in
	--repo)
		rk_set_repo "${2:?--repo needs a value}"
		shift 2
		;;
	--runner)
		runner_name="${2:?--runner needs a value}"
		shift 2
		;;
	--queue-job)
		queue_job=yes
		shift
		;;
	start | check)
		action="$1"
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		usage >&2
		exit 2
		;;
	esac
done
if [ -z "$RK_REPO" ] || [ -z "$action" ]; then
	usage >&2
	exit 2
fi

rk_need curl jq
rk_default_token
rk_require_private_repo
dir="$(rk_repo_dir)"
state="$dir/offline-watch.json"
log="$dir/offline-watch.log"
now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

checklist() {
	cat <<'EOF'

Look for a warning from GitHub and note each one (date, channel, wording):
  - your email inbox and spam folder: search for "runner", "self-hosted", "offline", "removed";
  - https://github.com/notifications (including "Done");
  - the repository's Settings > Actions > Runners page (any banner or notice);
  - the queued rk-probe run, if you started one: did its failure after
    24 hours notify you, and what did the message say?
EOF
}

if [ "$action" = start ]; then
	[ -n "$runner_name" ] || rk_die "start needs --runner NAME (the RunnerKit runner name)"
	[ ! -f "$state" ] || rk_die "$state exists: this watch was already started (delete it to restart)"
	runner="$(rk_runner "$runner_name")"
	[ -n "$runner" ] || rk_die "no runner named $runner_name in $RK_REPO"
	status="$(printf '%s' "$runner" | jq -r .status)"
	if [ "$status" != offline ]; then
		rk_warn "$runner_name is $status; stop its service first (sudo systemctl stop 'actions.runner.*' on the host), then run start again"
		exit 1
	fi
	queued_run=""
	if [ "$queue_job" = yes ]; then
		rk_require_workflow rk-probe.yml
		nonce="$(rk_new_nonce)"
		labels="$(printf '%s' "$runner" | jq -c '[.labels[].name]')"
		queued_run="$(rk_dispatch rk-probe.yml "$(jq -nc --arg r "$labels" --arg n "$nonce" '{runs_on: $r, nonce: $n}')" "$nonce")"
		rk_info "queued rk-probe run: $(rk_run_field "$queued_run" .html_url)"
	fi
	printf '%s' "$runner" | jq --arg at "$now" --arg repo "$RK_REPO" --arg run "$queued_run" \
		'{repo: $repo, runner: .name, id: .id, version: .version, labels: [.labels[].name], started_at: $at, queued_run: $run}' >"$state"
	jq -c --arg at "$now" '{at: $at, status: .status, version: .version}' <<EOF >>"$log"
$runner
EOF
	removal="$(jq -r '.started_at | fromdateiso8601 + 14 * 86400 | todate' "$state")"
	echo "Started watching $runner_name (id $(jq -r .id "$state")) at $now."
	echo "GitHub removes a runner that has not connected for more than 14 days: expect removal after $removal."
	echo "Run 'check' on days 1, 13, 14, 15 and 16. Record the dates in docs/testkit/validations.md, V-1."
	checklist
	exit 0
fi

[ -f "$state" ] || rk_die "no watch started for $RK_REPO ($state missing); run start first"
name="$(jq -r .runner "$state")"
id="$(jq -r .id "$state")"
rk_list_runners || rk_die "list runners: HTTP $RK_STATUS $(rk_api_message)"
runner="$(jq -c --argjson id "$id" '[.runners[] | select(.id == $id)][0] // empty' "$RK_BODY")"
days="$(jq -r --arg now "$now" '(($now | fromdateiso8601) - (.started_at | fromdateiso8601)) / 86400 | . * 10 | floor / 10' "$state")"
if [ -n "$runner" ]; then
	status="$(printf '%s' "$runner" | jq -r .status)"
	jq -nc --arg at "$now" --argjson r "$runner" '{at: $at, status: $r.status, version: $r.version}' >>"$log"
	echo "Day $days: $name (id $id) is still registered, status $status."
else
	jq -nc --arg at "$now" '{at: $at, status: "absent"}' >>"$log"
	last_present="$(jq -r 'select(.status != "absent") | .at' "$log" | tail -n1)"
	echo "Day $days: $name (id $id) is gone: GitHub removed it between ${last_present:-the start} and $now."
fi
queued_run="$(jq -r '.queued_run // empty' "$state")"
if [ -n "$queued_run" ]; then
	rk_api GET "/repos/$RK_REPO/actions/runs/$queued_run"
	if [ "$RK_STATUS" = 200 ]; then
		echo "Queued rk-probe run $queued_run: $(jq -r '"\(.status) \(.conclusion // "")"' "$RK_BODY") ($(jq -r .html_url "$RK_BODY"))"
	fi
fi
echo
echo "History ($log):"
cat "$log"
checklist
