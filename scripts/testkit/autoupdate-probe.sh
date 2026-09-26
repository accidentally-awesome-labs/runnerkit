#!/usr/bin/env bash
# scripts/testkit/autoupdate-probe.sh: validation V-3, does a runner that
# RunnerKit v1.3.3 installed (actions/runner 2.334.0) update itself?
# Runbook: docs/testkit/validations.md.
#
# install  Registers a probe runner at 2.334.0 on the host, laid out the way
#          RunnerKit lays out its runners (internal/bootstrap/script.go):
#          /opt/actions-runner/runnerkit-autoupdate-probe owned by
#          runnerkit-runner, configured through `su` as that user, run by
#          svc.sh as that user. It never passes --disableupdate. Every host
#          command runs with `sudo -n` and needs only what install.sh grants.
# run      Dispatches rk-probe.yml at the probe's label and reports:
#            U1 the probe registered at 2.334.0;
#            U2 the probe job succeeded;
#            U3 the runner now reports a newer version;
#            U4 bin and externals point at the new version and the
#               self-update log ends in .succeed;
#            U5 runsvc.sh is not empty and matches bin/runsvc.sh (runner
#               issue #4421);
#            U6 after `systemctl restart` the runner is online again.
# remove   Stops and uninstalls the service, deletes the runner on GitHub
#          and removes its directories.
# all      install, run, remove (the default).
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=scripts/testkit/lib.sh
. "$HERE/lib.sh"

# RunnerKit v1.3.3's pin (internal/bootstrap/package.go at v1.3.3).
OLD_VERSION=2.334.0
OLD_TARBALL=actions-runner-linux-x64-2.334.0.tar.gz
OLD_URL=https://github.com/actions/runner/releases/download/v2.334.0/actions-runner-linux-x64-2.334.0.tar.gz
OLD_SHA256=048024cd2c848eb6f14d5646d56c13a4def2ae7ee3ad12122bee960c56f3d271
PROBE_NAME=runnerkit-autoupdate-probe
PROBE_LABEL=rk-autoupdate-probe
PROBE_DIR=/opt/actions-runner/$PROBE_NAME
PROBE_WORK=/var/lib/runnerkit/work/$PROBE_NAME

usage() {
	cat <<'EOF'
Usage: scripts/testkit/autoupdate-probe.sh --repo OWNER/NAME --host USER@HOST[:PORT] [--ssh-key PATH] [install|run|remove|all]

Run it on the gate host after `runnerkit up`, before the revocation steps:
it needs the runnerkit-runner user and the passwordless sudo that
install.sh grants. It registers a second runner, runnerkit-autoupdate-probe,
and removes it again.
EOF
}

action=all
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
	install | run | remove | all)
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
if [ -z "$RK_REPO" ] || [ -z "$RK_SSH_TARGET" ]; then
	usage >&2
	exit 2
fi

rk_need curl jq ssh
rk_default_token
rk_require_private_repo
ev="$(rk_evidence_dir "autoupdate-$action")"
rk_info "evidence directory: $ev"

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

runner_version() {
	local json
	json="$(rk_runner "$PROBE_NAME")"
	if [ -n "$json" ]; then
		printf '%s' "$json" | jq -r '.version // "unset"'
	fi
}

do_install() {
	local token
	rk_ssh true || rk_die "cannot SSH to $RK_SSH_TARGET without a prompt"
	rk_ssh 'sudo -n -l' >/dev/null 2>&1 ||
		rk_die "the SSH user has no passwordless sudo: run this before the revocation steps, on a host prepared by install.sh"
	rk_api POST "/repos/$RK_REPO/actions/runners/registration-token"
	[ "$RK_STATUS" = 201 ] || rk_die "registration token: HTTP $RK_STATUS $(rk_api_message) (the token needs Administration: write)"
	token="$(jq -r '.token' "$RK_BODY")"
	printf '%s' "$token" | grep -Eq '^[A-Za-z0-9_-]+$' || rk_die "unexpected registration token format"
	rk_info "installing $PROBE_NAME at $OLD_VERSION on $RK_SSH_TARGET"
	# The token reaches config.sh's argv on the host, as it does in RunnerKit
	# (SEC-6); it expires after about an hour.
	rk_ssh bash -s >"$ev/install.log" 2>&1 <<EOF || rk_die "install failed; see $ev/install.log"
set -euo pipefail
id -u runnerkit-runner >/dev/null 2>&1 || { echo "runnerkit-runner does not exist: run runnerkit up first" >&2; exit 3; }
if [ -e '$PROBE_DIR' ]; then echo '$PROBE_DIR exists: run the remove action first' >&2; exit 4; fi
sudo -n install -d -o runnerkit-runner -g runnerkit-runner '$PROBE_DIR' '$PROBE_WORK'
cd '$PROBE_DIR'
sudo -n curl -fL --retry 3 --connect-timeout 10 -o '$OLD_TARBALL' '$OLD_URL'
printf '%s  %s\n' '$OLD_SHA256' '$OLD_TARBALL' | sudo -n sha256sum -c -
sudo -n tar xzf '$OLD_TARBALL' --skip-old-files
sudo -n chown -R runnerkit-runner:runnerkit-runner '$PROBE_DIR' '$PROBE_WORK'
sudo -n su -s /bin/bash - runnerkit-runner -c "cd '$PROBE_DIR' && ./config.sh --unattended --url 'https://github.com/$RK_REPO' --token '$token' --name '$PROBE_NAME' --labels '$PROBE_LABEL' --work '$PROBE_WORK' --replace"
sudo -n ./svc.sh install runnerkit-runner
sudo -n ./svc.sh start
EOF
	rk_wait_runner "$PROBE_NAME" online 300 || rk_die "$PROBE_NAME did not come online"
	installed_version="$(runner_version)"
	printf '%s\n' "$installed_version" >"$ev/installed-version.txt"
	rk_info "$PROBE_NAME is online, version $installed_version"
}

host_facts() { # FILE
	rk_ssh bash -s >"$1" 2>&1 <<EOF || true
cd '$PROBE_DIR' || exit 0
echo "bin -> \$(readlink bin || echo 'a directory')"
echo "externals -> \$(readlink externals || echo 'a directory')"
echo "versioned: \$(ls -d bin.* externals.* 2>/dev/null | tr '\n' ' ')"
echo "self-update logs: \$(ls _diag 2>/dev/null | grep '^SelfUpdate' | tr '\n' ' ')"
echo "runsvc.sh bytes: \$(wc -c < runsvc.sh)"
if cmp -s runsvc.sh bin/runsvc.sh; then echo 'runsvc.sh matches bin/runsvc.sh'; else echo 'runsvc.sh differs from bin/runsvc.sh'; fi
echo "unit: \$(cat .service) \$(systemctl is-active "\$(cat .service)")"
EOF
}

do_run() {
	local installed latest nonce run_id conclusion job_id log_version after unit bytes
	installed="$(cat "$ev/installed-version.txt" 2>/dev/null || runner_version)"
	[ -n "$installed" ] || rk_die "no $PROBE_NAME runner in $RK_REPO; run the install action first"
	rk_require_workflow rk-probe.yml
	latest="$(rk_latest_runner_version || true)"
	if [ "$installed" = "$OLD_VERSION" ]; then
		record U1 PASS "The probe registered at $OLD_VERSION" "version $installed before any job"
	else
		record U1 FAIL "The probe registered at $OLD_VERSION" "GitHub reports $installed before any job (it may have updated on connect)"
	fi

	nonce="$(rk_new_nonce)"
	run_id="$(rk_dispatch rk-probe.yml "$(jq -nc --arg r "[\"self-hosted\",\"$PROBE_LABEL\"]" --arg n "$nonce" '{runs_on: $r, nonce: $n}')" "$nonce")"
	rk_info "rk-probe run: $(rk_run_field "$run_id" .html_url)"
	conclusion="$(rk_wait_run "$run_id" 1800)"
	rk_run_jobs "$run_id" "$ev/jobs.json"
	job_id="$(jq -r '.jobs[0].id // empty' "$ev/jobs.json")"
	: >"$ev/job.log"
	if [ -n "$job_id" ]; then
		rk_job_log "$job_id" "$ev/job.log" || rk_warn "job log not available"
	fi
	log_version="$(grep -o "Current runner version: '[^']*'" "$ev/job.log" | head -n1 | sed "s/.*: '//; s/'\$//" || true)"
	if [ "$conclusion" = success ]; then
		record U2 PASS "The probe job succeeded" "job ran on runner version ${log_version:-unknown}"
	else
		record U2 FAIL "The probe job succeeded" "conclusion $conclusion"
	fi

	after="$(runner_version)"
	if [ -n "$after" ] && rk_version_lt "$OLD_VERSION" "$after"; then
		record U3 PASS "The runner updated itself" "$OLD_VERSION -> $after (latest release ${latest:-unknown})"
	else
		record U3 FAIL "The runner updated itself" "GitHub reports ${after:-no runner}; latest release ${latest:-unknown}"
	fi

	host_facts "$ev/host-after-job.txt"
	if grep -q "^bin -> .*bin\.$after\$" "$ev/host-after-job.txt" &&
		grep -q "^externals -> .*externals\.$after\$" "$ev/host-after-job.txt" &&
		grep -q 'SelfUpdate.*\.succeed' "$ev/host-after-job.txt" &&
		! grep -q 'SelfUpdate.*\.failed' "$ev/host-after-job.txt"; then
		record U4 PASS "bin and externals switched and the self-update log ends in .succeed" "$(grep -E '^(bin|externals) ->' "$ev/host-after-job.txt" | tr '\n' ' ')"
	else
		record U4 FAIL "bin and externals switched and the self-update log ends in .succeed" "see host-after-job.txt"
	fi
	bytes="$(sed -n 's/^runsvc.sh bytes: *//p' "$ev/host-after-job.txt")"
	if [ "${bytes:-0}" -gt 0 ] && grep -q 'runsvc.sh matches bin/runsvc.sh' "$ev/host-after-job.txt"; then
		record U5 PASS "runsvc.sh survived the update (runner issue #4421)" "$bytes bytes, matches bin/runsvc.sh"
	else
		record U5 FAIL "runsvc.sh survived the update (runner issue #4421)" "${bytes:-?} bytes; see host-after-job.txt"
	fi

	# A 0-byte runsvc.sh (#4421) only shows after a restart: the unit then
	# exits at once and the runner stays offline.
	unit="$(rk_ssh "cat '$PROBE_DIR/.service'")"
	rk_ssh "sudo -n systemctl restart '$unit'" || rk_warn "systemctl restart failed"
	sleep 20
	host_facts "$ev/host-after-restart.txt"
	if rk_wait_runner "$PROBE_NAME" online 240 && grep -q "^unit: .* active\$" "$ev/host-after-restart.txt"; then
		record U6 PASS "The runner is online again after systemctl restart" "version $(runner_version); unit active"
	else
		record U6 FAIL "The runner is online again after systemctl restart" "see host-after-restart.txt"
	fi
}

do_remove() {
	local id
	rk_info "removing $PROBE_NAME"
	rk_ssh bash -s >"$ev/remove.log" 2>&1 <<EOF || rk_warn "host cleanup reported errors; see $ev/remove.log"
set -u
if [ -d '$PROBE_DIR' ]; then
	cd '$PROBE_DIR'
	sudo -n ./svc.sh stop || true
	sudo -n ./svc.sh uninstall || true
fi
sudo -n rm -rf '$PROBE_DIR' '$PROBE_WORK'
EOF
	if rk_list_runners; then
		id="$(jq -r --arg n "$PROBE_NAME" '[.runners[] | select(.name == $n)][0].id // empty' "$RK_BODY")"
		if [ -n "$id" ]; then
			rk_api DELETE "/repos/$RK_REPO/actions/runners/$id"
			[ "$RK_STATUS" = 204 ] || rk_warn "delete runner $id: HTTP $RK_STATUS; remove it in Settings > Actions > Runners"
		fi
	fi
}

case "$action" in
install) do_install ;;
run) do_run ;;
remove) do_remove ;;
all)
	do_install
	rk_at_exit do_remove
	do_run
	;;
esac

if [ -s "$results" ]; then
	{
		echo "# V-3 runner auto-update probe: $RK_REPO"
		echo
		echo "- Date (UTC): $(date -u +%Y-%m-%dT%H:%M:%SZ)"
		echo "- Probe: $PROBE_NAME on $RK_SSH_TARGET, installed at $OLD_VERSION (RunnerKit v1.3.3's pin) without --disableupdate"
		echo
		echo "| # | Check | Result | Detail |"
		echo "| --- | --- | --- | --- |"
		while IFS='|' read -r id res crit detail; do
			echo "| $id | $crit | $res | $detail |"
		done <"$results"
		echo
		echo "Host facts after the job:"
		echo
		echo '```text'
		cat "$ev/host-after-job.txt" 2>/dev/null || true
		echo '```'
	} >"$ev/autoupdate-probe.md"
	cat "$ev/autoupdate-probe.md"
	rk_info "wrote $ev/autoupdate-probe.md"
fi
[ "$failed" -eq 0 ]
