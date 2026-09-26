#!/usr/bin/env bash
# scripts/testkit/fallback-probe.sh: validation V-4, falling back to
# GitHub-hosted runners by flipping a repository variable.
# Runbook: docs/testkit/validations.md.
#
# Uses .github/workflows/rk-fallback.yml, whose job runs on
#   ${{ fromJSON(vars.RUNS_ON || '["ubuntu-latest"]') }}
# and checks:
#   F1 RUNS_ON = the RunnerKit labels -> the RunnerKit runner takes the job;
#   F2 RUNS_ON = ["ubuntu-latest"]    -> a GitHub-hosted runner takes it
#      (set with a Variables-only token when you pass --flip-token);
#   F3 RUNS_ON deleted                 -> a GitHub-hosted runner takes it;
#   F4 with the runner service stopped, a job queued for the RunnerKit
#      labels stays queued after RUNS_ON flips to ["ubuntu-latest"];
#   F5 cancelling that run and re-running it sends the new attempt to
#      GitHub-hosted.
# F4 and F5 stop and restart the runner service over SSH with
# `sudo -n systemctl` (granted by install.sh); skip them with --skip-queued.
# At exit the script starts the service again and restores RUNS_ON.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=scripts/testkit/lib.sh
. "$HERE/lib.sh"

usage() {
	cat <<'EOF'
Usage: scripts/testkit/fallback-probe.sh --repo OWNER/NAME --host USER@HOST[:PORT] [options]

Options:
  --ssh-key PATH       SSH private key for the host
  --runs-on JSON       RunnerKit runner labels (default: from `runnerkit status --json`)
  --runner-name NAME   RunnerKit runner name (default: from `runnerkit status --json`)
  --flip-token         use a fine-grained PAT with only Variables: read and
                       write for the F2 flip (asked for, or RK_TOKEN_variables_only)
  --skip-queued        skip F4 and F5 (they stop the runner service for a few minutes)
EOF
}

runs_on=""
runner_name=""
flip_token=no
skip_queued=no
while [ $# -gt 0 ]; do
	case "$1" in
	--repo)
		rk_set_repo "${2:?--repo needs a value}"
		shift 2
		;;
	--host)
		rk_set_host "${2:?--host needs a value}"
		shift 2
		;;
	--ssh-key)
		RK_SSH_KEY="${2:?--ssh-key needs a value}"
		shift 2
		;;
	--runs-on)
		runs_on="${2:?--runs-on needs a value}"
		shift 2
		;;
	--runner-name)
		runner_name="${2:?--runner-name needs a value}"
		shift 2
		;;
	--flip-token)
		flip_token=yes
		shift
		;;
	--skip-queued)
		skip_queued=yes
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
if [ -z "$RK_REPO" ] || [ -z "$RK_SSH_TARGET" ]; then
	usage >&2
	exit 2
fi

rk_need curl jq ssh
rk_default_token
default_auth="$RK_AUTH_FILE"
rk_require_private_repo
rk_require_workflow rk-fallback.yml
ev="$(rk_evidence_dir fallback-probe)"
rk_info "evidence directory: $ev"

if [ -z "$runner_name" ] || [ -z "$runs_on" ]; then
	rk_runnerkit status --repo "$RK_REPO" --json --no-color >"$ev/runnerkit-status.json" 2>/dev/null || true
	[ -n "$runner_name" ] || runner_name="$(jq -r '.runner.name // empty' "$ev/runnerkit-status.json" 2>/dev/null || true)"
	[ -n "$runs_on" ] || runs_on="$(jq -c '.runner.labels // empty' "$ev/runnerkit-status.json" 2>/dev/null || true)"
fi
[ -n "$runner_name" ] && [ -n "$runs_on" ] || rk_die "pass --runner-name and --runs-on (runnerkit status gave neither)"
hosted='["ubuntu-latest"]'

results="$ev/results.txt"
: >"$results"
failed=0
record() {
	printf '%s|%s|%s|%s\n' "$1" "$2" "$3" "$(printf '%s' "$4" | tr '|\n' '/ ')" >>"$results"
	if [ "$2" != PASS ]; then
		failed=1
	fi
	printf '%-3s %-4s %s: %s\n' "$1" "$2" "$3" "$4" >&2
}

# --- RUNS_ON helpers -----------------------------------------------------------
var_path="/repos/$RK_REPO/actions/variables/RUNS_ON"
rk_api GET "$var_path"
case "$RK_STATUS" in
200) original_var="$(jq -r '.value' "$RK_BODY")" ;;
404) original_var="" ;;
*) rk_die "GET RUNS_ON: HTTP $RK_STATUS $(rk_api_message) (the token needs Variables: read and write)" ;;
esac

# var_set VALUE: creates or updates RUNS_ON with the current token; the
# HTTP status stays in RK_STATUS.
var_set() {
	local body
	body="$(jq -nc --arg v "$1" '{name: "RUNS_ON", value: $v}')"
	rk_api GET "$var_path"
	if [ "$RK_STATUS" = 200 ]; then
		rk_api PATCH "$var_path" "$body"
	else
		rk_api POST "/repos/$RK_REPO/actions/variables" "$body"
	fi
	case "$RK_STATUS" in
	201 | 204) return 0 ;;
	*) return 1 ;;
	esac
}
var_delete() {
	rk_api DELETE "$var_path"
	case "$RK_STATUS" in
	204 | 404) return 0 ;;
	*) return 1 ;;
	esac
}
restore_var() {
	RK_AUTH_FILE="$default_auth"
	if [ -n "$original_var" ]; then
		var_set "$original_var" || rk_warn "could not restore RUNS_ON (HTTP $RK_STATUS)"
	else
		var_delete || rk_warn "could not delete RUNS_ON (HTTP $RK_STATUS)"
	fi
}
rk_at_exit restore_var

# run_probe LABEL: dispatches rk-fallback, waits, saves the jobs JSON as
# jobs-LABEL.json and sets probe_run, probe_runner, probe_group, probe_labels.
run_probe() {
	local nonce conclusion
	nonce="$(rk_new_nonce)"
	probe_run="$(rk_dispatch rk-fallback.yml "$(jq -nc --arg n "$nonce" '{nonce: $n}')" "$nonce")"
	conclusion="$(rk_wait_run "$probe_run" 900)"
	rk_run_jobs "$probe_run" "$ev/jobs-$1.json"
	probe_runner="$(jq -r '.jobs[0].runner_name // "none"' "$ev/jobs-$1.json")"
	probe_group="$(jq -r '.jobs[0].runner_group_name // "none"' "$ev/jobs-$1.json")"
	probe_labels="$(jq -c '.jobs[0].labels // []' "$ev/jobs-$1.json")"
	rk_info "$1: run $probe_run $conclusion on $probe_runner ($probe_group) labels $probe_labels"
}

is_hosted() {
	[ "$probe_runner" != "$runner_name" ] && [ "$probe_runner" != none ] &&
		[ "$probe_labels" = "$hosted" ]
}

# --- F1 ------------------------------------------------------------------------
rk_wait_runner "$runner_name" online 120 || rk_die "$runner_name is not online; start it before this probe"
var_set "$runs_on" || rk_die "set RUNS_ON: HTTP $RK_STATUS $(rk_api_message)"
run_probe f1
if [ "$probe_runner" = "$runner_name" ]; then
	record F1 PASS "RUNS_ON = RunnerKit labels routes the job to the RunnerKit runner" "$probe_runner"
else
	record F1 FAIL "RUNS_ON = RunnerKit labels routes the job to the RunnerKit runner" "ran on $probe_runner"
fi

# --- F2 ------------------------------------------------------------------------
flip_detail="flipped with GH_TOKEN"
if [ "$flip_token" = yes ]; then
	if [ -n "${RK_TOKEN_variables_only:-}" ]; then
		rk_use_token variables-only "$RK_TOKEN_variables_only"
	else
		rk_use_token variables-only "$(rk_read_secret "Fine-grained PAT with only Variables: read and write (input hidden): ")"
	fi
	if var_set "$hosted"; then
		flip_detail="flipped with the Variables-only token (HTTP $RK_STATUS)"
	else
		flip_detail="Variables-only token refused: HTTP $RK_STATUS, needs $(rk_accepted_permissions)"
		RK_AUTH_FILE="$default_auth"
		var_set "$hosted" || rk_die "set RUNS_ON: HTTP $RK_STATUS"
	fi
	RK_AUTH_FILE="$default_auth"
else
	var_set "$hosted" || rk_die "set RUNS_ON: HTTP $RK_STATUS $(rk_api_message)"
fi
run_probe f2
if is_hosted; then
	record F2 PASS "RUNS_ON = [\"ubuntu-latest\"] routes the job to GitHub-hosted" "$probe_runner ($probe_group); $flip_detail"
else
	record F2 FAIL "RUNS_ON = [\"ubuntu-latest\"] routes the job to GitHub-hosted" "ran on $probe_runner with labels $probe_labels; $flip_detail"
fi

# --- F3 ------------------------------------------------------------------------
var_delete || rk_die "delete RUNS_ON: HTTP $RK_STATUS"
run_probe f3
if is_hosted; then
	record F3 PASS "Without RUNS_ON the job runs on GitHub-hosted (workflow default)" "$probe_runner ($probe_group)"
else
	record F3 FAIL "Without RUNS_ON the job runs on GitHub-hosted (workflow default)" "ran on $probe_runner with labels $probe_labels"
fi

# --- F4 and F5 -------------------------------------------------------------------
if [ "$skip_queued" = no ]; then
	unit="$(rk_ssh "systemctl list-units --all --plain --no-legend 'actions.runner.*'" | awk '{print $1}' | grep -F ".$runner_name.service" | head -n1 || true)"
	[ -n "$unit" ] || rk_die "no actions.runner.*.$runner_name.service unit on the host"
	restart_runner() {
		rk_ssh "sudo -n systemctl start '$unit'" || rk_warn "could not start $unit; run: sudo systemctl start '$unit'"
	}
	rk_at_exit restart_runner
	rk_ssh "sudo -n systemctl stop '$unit'" || rk_die "could not stop $unit with sudo -n (install.sh grants systemctl)"
	rk_wait_runner "$runner_name" offline 180 || rk_die "$runner_name did not go offline after stopping $unit"

	var_set "$runs_on" || rk_die "set RUNS_ON: HTTP $RK_STATUS"
	nonce="$(rk_new_nonce)"
	queued_run="$(rk_dispatch rk-fallback.yml "$(jq -nc --arg n "$nonce" '{nonce: $n}')" "$nonce")"
	job_status=""
	i=0
	while [ "$i" -lt 12 ]; do
		rk_run_jobs "$queued_run" "$ev/jobs-f4-before.json"
		job_status="$(jq -r '.jobs[0].status // "none"' "$ev/jobs-f4-before.json")"
		[ "$job_status" = queued ] && break
		i=$((i + 1))
		sleep 5
	done
	[ "$job_status" = queued ] || rk_die "the job for the stopped runner is $job_status, not queued"
	var_set "$hosted" || rk_die "set RUNS_ON: HTTP $RK_STATUS"
	rk_info "RUNS_ON flipped; waiting 90 s to see whether the queued job moves"
	sleep 90
	rk_run_jobs "$queued_run" "$ev/jobs-f4-after.json"
	job_status="$(jq -r '.jobs[0].status // "none"' "$ev/jobs-f4-after.json")"
	job_runner="$(jq -r '.jobs[0].runner_name // "none"' "$ev/jobs-f4-after.json")"
	if [ "$job_status" = queued ] && [ "$job_runner" = none ]; then
		record F4 PASS "A job queued for the stopped runner stays queued after the flip" "status $job_status, labels $(jq -c '.jobs[0].labels' "$ev/jobs-f4-after.json")"
	else
		record F4 FAIL "A job queued for the stopped runner stays queued after the flip" "status $job_status on $job_runner"
	fi

	rk_api POST "/repos/$RK_REPO/actions/runs/$queued_run/cancel"
	cancelled="$(rk_wait_run "$queued_run" 300)"
	attempt="$(rk_run_field "$queued_run" '.run_attempt')"
	rk_api POST "/repos/$RK_REPO/actions/runs/$queued_run/rerun"
	[ "$RK_STATUS" = 201 ] || rk_die "re-run $queued_run: HTTP $RK_STATUS $(rk_api_message)"
	# Wait for the new attempt before reading its result.
	i=0
	while [ "$(rk_run_field "$queued_run" '.run_attempt')" = "$attempt" ]; do
		i=$((i + 1))
		[ "$i" -lt 30 ] || rk_die "run $queued_run shows no new attempt after the re-run request"
		sleep 2
	done
	rerun_conclusion="$(rk_wait_run "$queued_run" 900)"
	rk_run_jobs "$queued_run" "$ev/jobs-f5.json"
	probe_runner="$(jq -r '.jobs[0].runner_name // "none"' "$ev/jobs-f5.json")"
	probe_group="$(jq -r '.jobs[0].runner_group_name // "none"' "$ev/jobs-f5.json")"
	probe_labels="$(jq -c '.jobs[0].labels // []' "$ev/jobs-f5.json")"
	if is_hosted; then
		record F5 PASS "Cancel and re-run sends the job to GitHub-hosted" "first attempt $cancelled; re-run $rerun_conclusion on $probe_runner"
	else
		record F5 FAIL "Cancel and re-run sends the job to GitHub-hosted" "re-run $rerun_conclusion on $probe_runner with labels $probe_labels"
	fi
	restart_runner
	rk_wait_runner "$runner_name" online 180 || rk_warn "$runner_name is not back online yet"
fi

# --- Report --------------------------------------------------------------------
{
	echo "# V-4 fallback probe: $RK_REPO"
	echo
	echo "- Date (UTC): $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo "- RunnerKit runner: $runner_name, labels $runs_on"
	echo "- Workflow: rk-fallback.yml, \`runs-on: \${{ fromJSON(vars.RUNS_ON || '[\"ubuntu-latest\"]') }}\`"
	echo
	echo "| # | Check | Result | Detail |"
	echo "| --- | --- | --- | --- |"
	while IFS='|' read -r id res crit detail; do
		echo "| $id | $crit | $res | $detail |"
	done <"$results"
} >"$ev/fallback-probe.md"
cat "$ev/fallback-probe.md"
rk_info "wrote $ev/fallback-probe.md"
[ "$failed" -eq 0 ]
