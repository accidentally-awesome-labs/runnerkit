#!/usr/bin/env bash
# Runs the SEED-001 curl|bash candidate against a fresh Ubuntu 24.04 Docker host.
#
# What this tests:
# 1. install.sh runs to completion on a clean Ubuntu host
# 2. Post-install, SSH-as-maintainer + `sudo -n` works for the allowlist
# 3. Specifically: the 4 NEW commands (ln, chmod, cp, cat) work non-interactively
# 4. UX comparison: how many manual steps did the maintainer perform?

set -euo pipefail

cd "$(dirname "$0")"

CONTAINER_NAME="${CONTAINER_NAME:-runnerkit-byo-experiment}"
HOST_PORT="${HOST_PORT:-22822}"

cleanup() {
  echo "===> cleanup"
  docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# 1. Generate ephemeral keypair for this experiment.
KEYDIR=$(mktemp -d -t runnerkit-byo-exp-XXXXXX)
ssh-keygen -t ed25519 -f "${KEYDIR}/id_ed25519" -N '' -q
PUBKEY=$(cat "${KEYDIR}/id_ed25519.pub")
echo "===> generated ephemeral ssh keypair at ${KEYDIR}"

# 2. Build the container image (idempotent rebuild on script change).
echo "===> building runnerkit-byo-host image"
docker build \
  --quiet \
  --build-arg "SSH_PUBKEY=${PUBKEY}" \
  -t runnerkit-byo-host:ubuntu-24.04 \
  -f Dockerfile \
  . | tail -1

# 3. Start the container.
docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
docker run -d --name "${CONTAINER_NAME}" \
  -p "${HOST_PORT}:22" \
  runnerkit-byo-host:ubuntu-24.04 >/dev/null
echo "===> container started: ${CONTAINER_NAME} (host port ${HOST_PORT})"

# 4. Wait for sshd.
SSH_OPTS=(-i "${KEYDIR}/id_ed25519" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=2 -p "${HOST_PORT}")
for i in $(seq 1 30); do
  if ssh "${SSH_OPTS[@]}" maintainer@127.0.0.1 'echo ok' 2>/dev/null | grep -q ok; then
    echo "===> sshd reachable (after ${i}s)"
    break
  fi
  sleep 1
done

# 5. Pre-install state — confirm sudo requires password.
echo
echo "===> PRE-INSTALL: maintainer sudo state"
ssh "${SSH_OPTS[@]}" maintainer@127.0.0.1 'sudo -n true 2>&1' || echo "  (expected: password required)"

# 6. Run runnerkit-install.sh via SSH using the SUDO PASSWORD method
#    (mimics what `curl | sudo bash` on the host's interactive shell would do).
#    Use sudo -S with a stdin pipe to feed the password.
echo
echo "===> RUNNING runnerkit-install.sh on host via 'curl | sudo bash' simulation"
START=$(date +%s)
# Stream the install script to the container then run it with sudo (password piped).
docker cp ./runnerkit-install.sh "${CONTAINER_NAME}:/tmp/runnerkit-install.sh"
ssh "${SSH_OPTS[@]}" maintainer@127.0.0.1 \
  'echo "dev-sudo-pwd" | sudo -S bash /tmp/runnerkit-install.sh 2>&1' | tee install-output.log
END=$(date +%s)
echo "===> install duration: $((END - START))s"

# 7. Post-install: SSH back in as maintainer and validate `sudo -n` works for all allowlist commands.
echo
echo "===> POST-INSTALL: validate non-TTY sudo for allowlist commands"
ssh "${SSH_OPTS[@]}" maintainer@127.0.0.1 'set -e
echo "--- baseline (already worked pre-fix) ---"
sudo -n apt-get -h >/dev/null && echo "  apt-get: OK" || echo "  apt-get: FAIL"
sudo -n install --version >/dev/null && echo "  install: OK" || echo "  install: FAIL"
sudo -n tar --help >/dev/null && echo "  tar: OK" || echo "  tar: FAIL"

echo "--- v1.3.3 fix candidates (new in allowlist) ---"
sudo -n ln -sf /tmp/exp-a /tmp/exp-b 2>/dev/null && echo "  ln: OK" || echo "  ln: FAIL"
sudo -n rm -f /tmp/exp-b 2>/dev/null
touch /tmp/exp-chmod && sudo -n chmod 0644 /tmp/exp-chmod 2>/dev/null && echo "  chmod: OK" || echo "  chmod: FAIL"
rm -f /tmp/exp-chmod
echo hello > /tmp/exp-src && sudo -n cp /tmp/exp-src /tmp/exp-dst 2>/dev/null && echo "  cp: OK" || echo "  cp: FAIL"
sudo -n rm -f /tmp/exp-src /tmp/exp-dst 2>/dev/null
sudo -n cat /etc/sudoers.d/runnerkit-installer >/dev/null 2>&1 && echo "  cat: OK" || echo "  cat: FAIL"

echo "--- NOT allowlisted (control — should fail) ---"
sudo -n whoami 2>/dev/null && echo "  whoami: ESCAPED (allowlist too wide)" || echo "  whoami: blocked (expected)"
sudo -n ls /root 2>/dev/null && echo "  ls: ESCAPED (allowlist too wide)" || echo "  ls: blocked (expected)"
' | tee post-install-check.log

# 8. Capture install-time facts.
echo
echo "===> INSTALL FACTS"
echo "Install duration: $((END - START))s"
echo "Container OS: $(ssh "${SSH_OPTS[@]}" maintainer@127.0.0.1 'cat /etc/os-release | head -2')"
echo "Service user present?  $(ssh "${SSH_OPTS[@]}" maintainer@127.0.0.1 'id runnerkit-runner >/dev/null 2>&1 && echo yes || echo NO')"
echo "Sudoers file present?  $(ssh "${SSH_OPTS[@]}" maintainer@127.0.0.1 'test -f /etc/sudoers.d/runnerkit-installer && echo yes || echo NO')"
echo "Shared-bin dir?        $(ssh "${SSH_OPTS[@]}" maintainer@127.0.0.1 'test -d /opt/actions-runner/runnerkit-shared-bin && echo yes || echo NO')"
echo "Baseline pkgs present? $(ssh "${SSH_OPTS[@]}" maintainer@127.0.0.1 'command -v gcc && command -v jq && command -v unzip' | head -1)"

# 9. Clean up keys.
rm -rf "${KEYDIR}"
echo
echo "===> experiment complete — see install-output.log + post-install-check.log"
