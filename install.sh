#!/usr/bin/env bash
# RunnerKit one-time BYO host install: scoped sudoers for the SSH user so
# `runnerkit register` / `runnerkit up` can run non-interactively over SSH.
# Run on the runner host: curl -fsSL <url> | sudo bash
# Or: sudo RUNNERKIT_INSTALL_USER=alice bash install.sh
set -euo pipefail

RK_USER="${RUNNERKIT_INSTALL_USER:-}"
if [[ -z "${RK_USER}" ]]; then
	if [[ -n "${SUDO_USER:-}" ]]; then
		RK_USER="${SUDO_USER}"
	else
		echo "runnerkit install: set RUNNERKIT_INSTALL_USER to the SSH login user (e.g. export RUNNERKIT_INSTALL_USER=alice) or invoke via sudo from that user's shell so \$SUDO_USER is set." >&2
		exit 2
	fi
fi

if [[ "$(id -u)" -ne 0 ]]; then
	echo "runnerkit install: must run as root (use: curl ... | sudo bash)" >&2
	exit 1
fi

if [[ "${RK_USER}" == "root" ]]; then
	echo "runnerkit install: RUNNERKIT_INSTALL_USER must not be root; use the non-root SSH user." >&2
	exit 2
fi

SUDOERS_PATH="/etc/sudoers.d/runnerkit-installer"

render_sudoers() {
	local u="$1"
	cat <<EOF
# /etc/sudoers.d/runnerkit-installer (managed by runnerkit install.sh)
${u} ALL=(root) NOPASSWD: \\
  /usr/bin/apt-get, /usr/bin/dnf, /usr/bin/yum, \\
  /usr/sbin/useradd, \\
  /usr/bin/install, \\
  /usr/bin/curl, \\
  /usr/bin/sha256sum, \\
  /bin/chown, /usr/bin/chown, \\
  /bin/rm, /usr/bin/rm, \\
  /bin/su, /usr/bin/su, \\
  /bin/tar, /usr/bin/tar, \\
  /bin/systemctl, /usr/bin/systemctl, \\
  /opt/actions-runner/runnerkit-*/svc.sh
EOF
}

TMP="$(mktemp "${TMPDIR:-/tmp}/runnerkit-installer.XXXXXX")"
cleanup() { rm -f "${TMP}"; }
trap cleanup EXIT

render_sudoers "${RK_USER}" >"${TMP}"
chmod 0440 "${TMP}"
if ! visudo -cf "${TMP}"; then
	echo "runnerkit install: visudo rejected sudoers content" >&2
	exit 21
fi
install -m 0440 -o root -g root "${TMP}" "${SUDOERS_PATH}"
trap - EXIT
rm -f "${TMP}"

echo "runnerkit install: wrote ${SUDOERS_PATH} for user ${RK_USER}. You can now run runnerkit register or runnerkit up from your workstation."

# Optional: NOPASSWD for package managers as the Actions runner service user (RKD-GH-008).
#   sudo RUNNERKIT_GRANT_CI_SUDO=1 RUNNERKIT_SERVICE_USER=runnerkit-runner bash install.sh
if [[ "${RUNNERKIT_GRANT_CI_SUDO:-}" == "1" ]]; then
	RK_SVC="${RUNNERKIT_SERVICE_USER:-runnerkit-runner}"
	RK_CI_PATH="/etc/sudoers.d/runnerkit-runner-ci"
	render_ci_sudoers() {
		local u="$1"
		cat <<EOF
# ${RK_CI_PATH} (managed by runnerkit install.sh --grant-ci-sudo)
${u} ALL=(root) NOPASSWD: \\
  /usr/bin/apt-get, /usr/bin/apt, /bin/apt, \\
  /usr/bin/dnf, /usr/bin/yum, /usr/bin/microdnf, \\
  /usr/bin/zypper, /usr/sbin/zypper, \\
  /usr/bin/pacman, \\
  /sbin/apk, /usr/bin/apk
EOF
	}
	TMP_CI="$(mktemp "${TMPDIR:-/tmp}/runnerkit-runner-ci.XXXXXX")"
	render_ci_sudoers "${RK_SVC}" >"${TMP_CI}"
	chmod 0440 "${TMP_CI}"
	if ! visudo -cf "${TMP_CI}"; then
		echo "runnerkit install: visudo rejected CI sudoers content" >&2
		rm -f "${TMP_CI}"
		exit 22
	fi
	install -m 0440 -o root -g root "${TMP_CI}" "${RK_CI_PATH}"
	rm -f "${TMP_CI}"
	echo "runnerkit install: wrote ${RK_CI_PATH} for service user ${RK_SVC}."
fi
