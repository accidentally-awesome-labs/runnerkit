#!/usr/bin/env bash
# runnerkit-install.sh — SEED-001 candidate: one-time privileged host setup
#
# Goal: a single command the maintainer runs on their host (one sudo prompt)
# that leaves the host ready for `runnerkit register` over SSH-as-non-root
# without further password prompts.
#
# This is a DRAFT for the discovery experiment, not the shipped installer.
# Differences from a real install.sh:
# - No checksum / signature verification (would add cosign)
# - No version pinning to a release tag
# - Embedded sudoers content rather than fetching versioned spec
# - No idempotency hardening (rerun would partially recreate things)
#
# Usage on the host:
#   curl -fsSL https://example.invalid/runnerkit-install.sh | sudo bash
#
# After it runs:
# - User `runnerkit-runner` exists with passwordless service account home
# - /etc/sudoers.d/runnerkit-installer has scoped allowlist for SSH-driven lifecycle
# - Baseline apt packages installed (build-essential, git, curl, etc.)
# - Cache directory at /opt/actions-runner/runnerkit-shared-bin/ ready for tarballs
# - SSH-as-non-root from maintainer's laptop can now call `sudo -n` on the allowlist

set -euo pipefail

# Require root (we expect to be piped through sudo).
if [ "${EUID:-$(id -u)}" -ne 0 ]; then
  echo "ERROR: runnerkit-install.sh must run as root (use 'curl ... | sudo bash')" >&2
  exit 2
fi

# Detect OS — this draft only supports Ubuntu/Debian. Fedora/RHEL would be a separate path.
. /etc/os-release
case "${ID:-}" in
  ubuntu|debian) ;;
  *) echo "ERROR: this draft only supports ubuntu/debian (saw ID=${ID:-unknown})" >&2; exit 3 ;;
esac

SERVICE_USER="${SERVICE_USER:-runnerkit-runner}"
SHARED_BIN_DIR="/opt/actions-runner/runnerkit-shared-bin"
SUDOERS_PATH="/etc/sudoers.d/runnerkit-installer"

echo "==> runnerkit-install.sh (draft) — host ${HOSTNAME:-?} (${ID} ${VERSION_ID:-?})"
echo "==> service user: ${SERVICE_USER}"

# ── service user ──────────────────────────────────────────────────────────
if ! id -u "${SERVICE_USER}" >/dev/null 2>&1; then
  echo "==> creating service user ${SERVICE_USER}"
  useradd --system --create-home --shell /usr/sbin/nologin "${SERVICE_USER}"
fi

# ── baseline packages (subset of bootstrap.BaselinePackages for demo) ─────
echo "==> installing baseline packages (minimal subset for demo)"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y --no-install-recommends \
  build-essential pkg-config git curl jq unzip ca-certificates \
  >/dev/null

# ── shared-bin cache ──────────────────────────────────────────────────────
mkdir -p "${SHARED_BIN_DIR}"
chown root:root "${SHARED_BIN_DIR}"
chmod 0755 "${SHARED_BIN_DIR}"

# ── scoped sudoers ────────────────────────────────────────────────────────
# Stage to a tempfile, validate with visudo, atomically move into place.
# Mirrors RemoteVisudoCheckScript() from internal/bootstrap/sudoers.go but
# runs locally so there is no SSH-non-TTY problem.
#
# IMPORTANT: this draft includes the v1.3.3 fix candidates (ln, chmod, cp, cat)
# alongside the existing allowlist so the post-install non-TTY lifecycle works
# for runnerkit up's setup_runner_image step.
SUDOERS_USER="$(id -un "${SUDO_USER:-root}")"

# If the user piping the curl was a real user (not root via su), prefer them.
if [ -n "${SUDO_USER:-}" ] && [ "${SUDO_USER}" != "root" ]; then
  SUDOERS_USER="${SUDO_USER}"
fi

TMP_SUDOERS=$(mktemp /tmp/runnerkit-installer.XXXXXX)
chmod 0440 "${TMP_SUDOERS}"
cat > "${TMP_SUDOERS}" <<EOF
# /etc/sudoers.d/runnerkit-installer (managed by runnerkit-install.sh)
${SUDOERS_USER} ALL=(root) NOPASSWD: \\
  /usr/bin/apt-get, /usr/bin/dnf, /usr/bin/yum, \\
  /usr/sbin/useradd, \\
  /usr/bin/install, \\
  /usr/bin/curl, \\
  /usr/bin/sha256sum, \\
  /usr/bin/tee, /usr/bin/gpg, \\
  /bin/mkdir, /usr/bin/mkdir, /usr/bin/unzip, \\
  /usr/sbin/usermod, /usr/bin/dpkg, /usr/bin/add-apt-repository, \\
  /bin/chown, /usr/bin/chown, \\
  /bin/rm, /usr/bin/rm, \\
  /bin/su, /usr/bin/su, \\
  /bin/tar, /usr/bin/tar, \\
  /bin/systemctl, /usr/bin/systemctl, \\
  /bin/ln, /usr/bin/ln, \\
  /bin/chmod, /usr/bin/chmod, \\
  /bin/cp, /usr/bin/cp, \\
  /bin/cat, /usr/bin/cat, \\
  /opt/actions-runner/runnerkit-*/svc.sh
EOF

if ! visudo -cf "${TMP_SUDOERS}"; then
  echo "ERROR: visudo validation failed — sudoers fragment NOT installed" >&2
  rm -f "${TMP_SUDOERS}"
  exit 21
fi

mv "${TMP_SUDOERS}" "${SUDOERS_PATH}"
chmod 0440 "${SUDOERS_PATH}"
chown root:root "${SUDOERS_PATH}"

echo "==> installed scoped sudoers fragment at ${SUDOERS_PATH}"
echo "==> done. Host is ready for: runnerkit register --host ${SUDOERS_USER}@${HOSTNAME:-?} --repo owner/name"
