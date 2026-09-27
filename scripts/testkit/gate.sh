#!/usr/bin/env bash
# scripts/testkit/gate.sh: the real-job BYO release gate (A-20).
# Runbook: docs/testkit/release-gate.md.
#
# For the RunnerKit runner of a throwaway private repository it checks:
#   G1 the runner is online on GitHub with the RunnerKit labels;
#   G2 a dispatched rk-gate job succeeds on that runner;
#   G3 gcc builds and runs hello.c in the job;
#   G4 `docker run hello-world` works in the job (is refused, with
#      --after-revocation);
#   G5 `id -nG runnerkit-runner` on the host contains docker (lacks it,
#      with --after-revocation);
#   G6 the host is Ubuntu 24.04 x86_64;
#   G7 the SSH user's passwordless sudo is exactly the fragment the
#      candidate install.sh writes and never NOPASSWD: ALL (with
#      --after-revocation: the SSH user has no passwordless sudo at all);
#   G8 with --after-revocation only: the job can neither rename svc.sh or
#      bin nor create files in the install directory, which is root-owned.
#   G9 without --after-revocation only: the runner took the job at the
#      version this checkout installs (RunnerVersion in
#      internal/bootstrap/package.go) without updating itself first.
# Writes EVIDENCE.md plus the raw material to
# ~/runnerkit-testkit/<owner>-<name>/<time>-gate-<mode>/ and exits 0 only
# when every criterion passes.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=scripts/testkit/lib.sh
. "$HERE/lib.sh"

usage() {
	cat <<'EOF'
Usage: scripts/testkit/gate.sh --repo OWNER/NAME --host USER@HOST[:PORT] [options]

Options:
  --ssh-key PATH        SSH private key for the host
  --after-revocation    check the host after the revocation steps
                        (docs/testkit/release-gate.md, part 3)
  --runs-on JSON        runner labels (default: from `runnerkit status --json`)
  --runner-name NAME    runner name (default: from `runnerkit status --json`)
  --install-sh PATH     the candidate install.sh (default: this checkout's)
  --timeout SECONDS     how long to wait for the job (default 1800)

Environment: GH_TOKEN (fine-grained PAT for the test repository),
RUNNERKIT_BIN and RUNNERKIT_STATE_DIR (the candidate binary and its state).
EOF
}

mode=gate
runs_on=""
runner_name=""
install_sh="$HERE/../../install.sh"
timeout=1800
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
	--after-revocation)
		mode=revoked
		shift
		;;
	--runs-on)
		runs_on="${2:?--runs-on needs a value}"
		shift 2
		;;
	--runner-name)
		runner_name="${2:?--runner-name needs a value}"
		shift 2
		;;
	--install-sh)
		install_sh="${2:?--install-sh needs a value}"
		shift 2
		;;
	--timeout)
		timeout="${2:?--timeout needs a value}"
		shift 2
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

rk_need curl jq ssh git sed awk
[ -f "$install_sh" ] || rk_die "candidate install.sh not found: $install_sh"
rk_default_token
rk_require_private_repo
rk_require_workflow rk-gate.yml
ev="$(rk_evidence_dir "gate-$mode")"
rk_info "evidence directory: $ev"

results="$ev/results.txt"
: >"$results"
failed=0
# record ID RESULT CRITERION DETAIL
record() {
	local detail
	detail="$(printf '%s' "$4" | tr '|\n' '/ ')"
	printf '%s|%s|%s|%s\n' "$1" "$2" "$3" "$detail" >>"$results"
	if [ "$2" != PASS ]; then
		failed=1
	fi
	printf '%-3s %-4s %s: %s\n' "$1" "$2" "$3" "$detail" >&2
}

# log_has MARKER: the job printed MARKER as a line of its own (the log also
# echoes each step's source, where the marker follows `echo "`).
log_has() {
	grep -Eq "^[^ ]* $1[[:space:]]*\$" "$ev/job.log" 2>/dev/null
}

# --- Candidate ---------------------------------------------------------------
root="$(cd "$HERE/../.." && pwd)"
commit="$(git -C "$root" rev-parse HEAD 2>/dev/null || echo unknown)"
if [ -n "$(git -C "$root" status --porcelain 2>/dev/null | head -n1)" ]; then
	tree_state="checkout has local changes: not a valid gate run for a tag"
else
	tree_state="clean checkout"
fi
rk_runnerkit --version >"$ev/runnerkit-version.txt" 2>&1 || true

# --- Runner identity from RunnerKit state -----------------------------------
rk_runnerkit status --repo "$RK_REPO" --json --no-color >"$ev/runnerkit-status.json" 2>"$ev/runnerkit-status.stderr" || true
if [ -z "$runner_name" ]; then
	runner_name="$(jq -r '.runner.name // empty' "$ev/runnerkit-status.json" 2>/dev/null || true)"
fi
if [ -z "$runs_on" ]; then
	runs_on="$(jq -c '.runner.labels // empty' "$ev/runnerkit-status.json" 2>/dev/null || true)"
fi
if [ -z "$runner_name" ] || [ -z "$runs_on" ]; then
	rk_die "could not read the runner name and labels from \`runnerkit status --repo $RK_REPO --json\` (see $ev/runnerkit-status.stderr); pass --runner-name and --runs-on"
fi
printf '%s' "$runs_on" | jq -e 'type == "array" and length > 0 and all(type == "string")' >/dev/null ||
	rk_die "--runs-on must be a JSON array of strings, got: $runs_on"
rk_runnerkit doctor --repo "$RK_REPO" --json --no-color >"$ev/runnerkit-doctor.json" 2>"$ev/runnerkit-doctor.stderr" || true
rk_runnerkit list --json --no-color >"$ev/runnerkit-list.json" 2>"$ev/runnerkit-list.stderr" || true

# --- G1: GitHub runner record -----------------------------------------------
runner_json="$(rk_runner "$runner_name")"
printf '%s\n' "${runner_json:-null}" >"$ev/github-runner.json"
r_id="" r_version="" r_status=""
if [ -z "$runner_json" ]; then
	record G1 FAIL "Runner online on GitHub with the RunnerKit labels" "no runner named $runner_name in $RK_REPO"
else
	r_id="$(printf '%s' "$runner_json" | jq -r '.id')"
	r_status="$(printf '%s' "$runner_json" | jq -r '.status')"
	r_version="$(printf '%s' "$runner_json" | jq -r '.version // "unset"')"
	missing="$(jq -nc --argjson want "$(rk_labels_json_lower "$runs_on")" \
		--argjson have "$(printf '%s' "$runner_json" | jq -c '[.labels[].name | ascii_downcase]')" '$want - $have')"
	if [ "$r_status" = online ] && [ "$missing" = "[]" ]; then
		record G1 PASS "Runner online on GitHub with the RunnerKit labels" "$runner_name id $r_id, status $r_status, version $r_version"
	else
		record G1 FAIL "Runner online on GitHub with the RunnerKit labels" "status $r_status, missing labels $missing"
	fi
fi

# --- Host facts over SSH (no sudo password needed) --------------------------
rk_ssh true || rk_die "cannot SSH to $RK_SSH_TARGET without a prompt; accept the host key first (ssh ${RK_SSH_TARGET}) and use key authentication"
# shellcheck disable=SC2016 # expands on the host
os_line="$(rk_ssh '. /etc/os-release && printf "%s %s %s\n" "$ID" "$VERSION_ID" "$(uname -m)"' 2>&1)" || os_line="error: $os_line"
printf '%s\n' "$os_line" >"$ev/host-os.txt"
groups="$(rk_ssh 'id -nG runnerkit-runner' 2>&1)" || groups="error: $groups"
printf '%s\n' "$groups" >"$ev/host-runner-groups.txt"
ssh_user="$(rk_ssh 'id -un')"
rk_ssh "stat -L -c '%U:%G %a %n' /opt/actions-runner/$runner_name /opt/actions-runner/$runner_name/svc.sh /opt/actions-runner/$runner_name/bin" \
	>"$ev/host-install-dir.txt" 2>&1 || true
rk_ssh 'cat /var/lib/runnerkit/image-setup.json' >"$ev/host-image-setup.json" 2>&1 || true
rk_ssh "systemctl list-units --all --plain --no-legend 'actions.runner.*'" >"$ev/host-units.txt" 2>&1 || true
rk_ssh 'df -h /' >"$ev/host-df.txt" 2>&1 || true

# --- Dispatch the job ------------------------------------------------------
nonce="$(rk_new_nonce)"
inputs="$(jq -nc --arg r "$runs_on" --arg m "$mode" --arg n "$nonce" '{runs_on: $r, mode: $m, nonce: $n}')"
run_id="$(rk_dispatch rk-gate.yml "$inputs" "$nonce")"
run_url="$(rk_run_field "$run_id" .html_url)"
rk_info "run: $run_url"
conclusion="$(rk_wait_run "$run_id" "$timeout")"
if [ "$conclusion" = timeout ]; then
	rk_api POST "/repos/$RK_REPO/actions/runs/$run_id/cancel"
	rk_warn "cancelled run $run_id after ${timeout}s"
fi
rk_run_jobs "$run_id" "$ev/jobs.json"
job_id="$(jq -r '.jobs[0].id // empty' "$ev/jobs.json")"
job_url="$(jq -r '.jobs[0].html_url // empty' "$ev/jobs.json")"
job_runner="$(jq -r '.jobs[0].runner_name // "none"' "$ev/jobs.json")"
job_conclusion="$(jq -r '.jobs[0].conclusion // "none"' "$ev/jobs.json")"
: >"$ev/job.log"
if [ -n "$job_id" ] && ! rk_job_log "$job_id" "$ev/job.log"; then
	rk_warn "the job log is not available yet; re-download it from $job_url"
fi
log_version="$(grep -o "Current runner version: '[^']*'" "$ev/job.log" | head -n1 | sed "s/.*: '//; s/'\$//" || true)"
grep -E '^[^ ]* RK-GATE' "$ev/job.log" | sed 's/^[^ ]* //' >"$ev/job-markers.txt" || true

# A runner that updates itself replaces bin with a link to bin.<version>
# and leaves a SelfUpdate log in _diag.
rk_ssh bash -s >"$ev/host-runner-update.txt" 2>&1 <<EOF || true
cd '/opt/actions-runner/$runner_name' || exit 1
if [ -L bin ]; then echo "bin -> \$(readlink bin)"; else echo 'bin is a directory'; fi
if ls _diag >/dev/null; then ls _diag | grep '^SelfUpdate' || echo 'no self-update logs'; fi
EOF

# --- G2..G8 --------------------------------------------------------------------
if [ "$job_conclusion" = success ] && [ "$job_runner" = "$runner_name" ]; then
	record G2 PASS "rk-gate job succeeds on the RunnerKit runner" "run conclusion $conclusion; job ran on $job_runner"
else
	record G2 FAIL "rk-gate job succeeds on the RunnerKit runner" "run conclusion $conclusion; job conclusion $job_conclusion on $job_runner"
fi

if log_has 'RK-GATE-PASS gcc'; then
	record G3 PASS "gcc builds and runs hello.c in the job" "hello from gcc"
else
	record G3 FAIL "gcc builds and runs hello.c in the job" "no RK-GATE-PASS gcc line in the job log"
fi

if [ "$mode" = gate ]; then
	if log_has 'RK-GATE-PASS docker'; then
		record G4 PASS "docker run hello-world works in the job" "as $(grep -o 'user=[^ ]*' "$ev/job-markers.txt" | head -n1)"
	else
		record G4 FAIL "docker run hello-world works in the job" "no RK-GATE-PASS docker line in the job log"
	fi
	case " $groups " in
	*" docker "*) record G5 PASS "runnerkit-runner is in the docker group (SEC-5)" "$groups" ;;
	*) record G5 FAIL "runnerkit-runner is in the docker group (SEC-5)" "$groups" ;;
	esac
else
	if log_has 'RK-GATE-PASS docker-refused'; then
		record G4 PASS "docker is refused in the job after revocation step 3" "docker run hello-world failed as expected"
	else
		record G4 FAIL "docker is refused in the job after revocation step 3" "no RK-GATE-PASS docker-refused line in the job log"
	fi
	case " $groups " in
	*" docker "*) record G5 FAIL "runnerkit-runner is no longer in the docker group" "$groups" ;;
	*) record G5 PASS "runnerkit-runner is no longer in the docker group" "$groups" ;;
	esac
fi

if [ "$os_line" = "ubuntu 24.04 x86_64" ]; then
	record G6 PASS "Host is Ubuntu 24.04 x86_64" "$os_line"
else
	record G6 FAIL "Host is Ubuntu 24.04 x86_64" "$os_line"
fi

sudo_l="$(rk_ssh 'sudo -n -l' 2>&1)" && sudo_l_rc=0 || sudo_l_rc=$?
printf '%s\n' "$sudo_l" >"$ev/host-sudo-l.txt"
if [ "$mode" = gate ]; then
	# shellcheck disable=SC2016 # $1 and $2 belong to the inner bash
	expected="$(bash -c 'set -euo pipefail; eval "$(sed -n "/^render_sudoers() {/,/^}/p" "$1")"; render_sudoers "$2"' rk "$install_sh" "$ssh_user")"
	actual="$(rk_ssh 'sudo -n cat /etc/sudoers.d/runnerkit-installer' 2>&1)" || actual="error: $actual"
	printf '%s\n' "$expected" >"$ev/sudoers-expected.txt"
	printf '%s\n' "$actual" >"$ev/sudoers-actual.txt"
	nopasswd_all=no
	if ! printf '%s\n' "$sudo_l" | awk '/NOPASSWD:/ {
			sub(/.*NOPASSWD:[[:space:]]*/, "")
			n = split($0, cmds, ",")
			for (i = 1; i <= n; i++) {
				gsub(/^[[:space:]]+|[[:space:]]+$/, "", cmds[i])
				if (cmds[i] == "ALL") bad = 1
			}
		}
		END { exit bad ? 1 : 0 }'; then
		nopasswd_all=yes
	fi
	if [ "$sudo_l_rc" -ne 0 ]; then
		record G7 FAIL "Passwordless sudo is exactly the install.sh fragment" "sudo -n -l failed for $ssh_user: install.sh has not run on this host"
	elif [ "$nopasswd_all" = yes ]; then
		record G7 FAIL "Passwordless sudo is exactly the install.sh fragment" "$ssh_user has NOPASSWD: ALL (for example from cloud-init); this is not a password-sudo host"
	elif [ "$expected" != "$actual" ]; then
		record G7 FAIL "Passwordless sudo is exactly the install.sh fragment" "/etc/sudoers.d/runnerkit-installer differs from the candidate install.sh (compare sudoers-expected.txt and sudoers-actual.txt)"
	else
		record G7 PASS "Passwordless sudo is exactly the install.sh fragment" "$ssh_user: fragment matches the candidate install.sh byte for byte; no NOPASSWD: ALL"
	fi
else
	if [ "$sudo_l_rc" -ne 0 ]; then
		record G7 PASS "The SSH user has no passwordless sudo (revocation step 1)" "sudo -n -l: $(printf '%s' "$sudo_l" | head -n1)"
	else
		record G7 FAIL "The SSH user has no passwordless sudo (revocation step 1)" "sudo -n -l still lists passwordless commands"
	fi
	owners="$(awk '{print $1}' "$ev/host-install-dir.txt" | sort -u | tr '\n' ' ')"
	if log_has 'RK-GATE-PASS install-dir-refused' && [ "$owners" = "root:root " ]; then
		record G8 PASS "The job cannot replace svc.sh or bin (revocation step 2)" "install directory, svc.sh and bin are root:root; renames refused in the job"
	else
		record G8 FAIL "The job cannot replace svc.sh or bin (revocation step 2)" "owners: $owners; see job-markers.txt and host-install-dir.txt"
	fi
fi

if [ "$mode" = gate ]; then
	pin="$(rk_runner_pin)"
	if [ -n "$pin" ] && [ "$r_version" = "$pin" ] && [ "$log_version" = "$pin" ] &&
		grep -qx 'bin is a directory' "$ev/host-runner-update.txt" &&
		grep -qx 'no self-update logs' "$ev/host-runner-update.txt"; then
		record G9 PASS "The runner took the job at RunnerKit's pin without updating itself" "$pin from the API and the job log; bin is the installed directory; no _diag/SelfUpdate log"
	else
		record G9 FAIL "The runner took the job at RunnerKit's pin without updating itself" "pin ${pin:-unreadable}, API ${r_version:-?}, job log ${log_version:-?}; see host-runner-update.txt. If a newer actions/runner release came out, bump RunnerVersion and rebuild the candidate"
	fi
fi

# --- EVIDENCE.md ---------------------------------------------------------------
overall=PASS
if [ "$failed" -ne 0 ]; then
	overall=FAIL
fi
{
	echo "# RunnerKit real-job gate ($mode): $overall"
	echo
	echo "- Date (UTC): $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo "- Repository: $RK_REPO (private)"
	echo "- Candidate commit: $commit ($tree_state)"
	echo "- RunnerKit binary: $(head -n1 "$ev/runnerkit-version.txt")"
	echo "- Host: $RK_SSH_TARGET${RK_SSH_PORT:+:$RK_SSH_PORT} ($os_line)"
	echo "- Runner: $runner_name (GitHub id ${r_id:-?}; runner version ${r_version:-?} from the API, ${log_version:-?} from the job log)"
	echo "- Run: $run_url"
	echo "- Job: ${job_url:-none}"
	echo
	echo "| # | Criterion | Result | Detail |"
	echo "| --- | --- | --- | --- |"
	while IFS='|' read -r id res crit detail; do
		echo "| $id | $crit | $res | $detail |"
	done <"$results"
	echo
	echo "## Job output"
	echo
	echo '```text'
	cat "$ev/job-markers.txt"
	echo '```'
	echo
	echo "## Manual checks"
	echo
	if [ "$mode" = gate ]; then
		echo "- [ ] The host was created for this test: provider, image and creation time: ..."
		echo "- [ ] Before install.sh ran, \`sudo -n true\` failed for $ssh_user (sudo needed a password)."
		echo "- [ ] install.sh was copied from commit $commit with scp and run once with sudo."
		echo "- [ ] \`runnerkit up\` output is saved as up.log in $(dirname "$ev") (duration: ...)."
		echo
		echo "## CHANGELOG line (only when every row is PASS)"
		echo
		echo "Real-job gate ($(date -u +%Y-%m-%d)): $run_url; runner ${log_version:-$r_version} on Ubuntu 24.04 x86_64, a password-sudo host prepared only by install.sh at ${commit:0:12}."
	else
		echo "- [ ] Revocation steps 1-3 of docs/security-posture.md ran exactly as written; \`sudo visudo -c\` parsed OK."
		echo "- [ ] After step 2, \`sudo -u runnerkit-runner mv <install>/svc.sh <install>/svc.sh.orig\` and the same for bin failed with Permission denied."
		echo "- [ ] After step 3, the runner came back online."
	fi
} >"$ev/EVIDENCE.md"

rk_info "wrote $ev/EVIDENCE.md"
printf '\nGate (%s): %s\n' "$mode" "$overall"
[ "$failed" -eq 0 ]
