package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

// RunnerCISudoersFilePath is the scoped sudoers drop-in that grants the GitHub
// Actions runner service user passwordless sudo for common package managers only.
// See docs/troubleshooting/github.md (RKD-GH-008).
const RunnerCISudoersFilePath = "/etc/sudoers.d/runnerkit-runner-ci"

// RenderRunnerCISudoersEntry renders NOPASSWD sudoers for the runner service
// user so CI workflows can run `sudo apt-get`, `sudo dnf`, etc. without a TTY.
//
// Paths are listed explicitly — sudoers requires absolute paths. Unused paths
// on a given distro are harmless (the binaries simply do not exist). Covers
// Debian/Ubuntu, RHEL/Fedora, openSUSE, Arch, Alpine (apk paths vary).
//
// This is NOT a blanket NOPASSWD ALL.
func RenderRunnerCISudoersEntry(serviceUser string) string {
	return fmt.Sprintf(`# %s (managed by runnerkit byo-prepare --grant-ci-sudo)
%s ALL=(root) NOPASSWD: \
  /usr/bin/apt-get, /usr/bin/apt, /bin/apt, \
  /usr/bin/dnf, /usr/bin/yum, /usr/bin/microdnf, \
  /usr/bin/zypper, /usr/sbin/zypper, \
  /usr/bin/pacman, \
  /sbin/apk, /usr/bin/apk
`, RunnerCISudoersFilePath, serviceUser)
}

// RemoteKernelNameScript prints `uname -s` on one line (Linux vs Darwin vs …).
func RemoteKernelNameScript() string {
	return `set -euo pipefail
uname -s | tr -d '\r'
`
}

// RemoteRunnerCIVisudoCheckScript installs RunnerCISudoersFilePath using the
// same visudo-before-mv safety pattern as RemoteVisudoCheckScript.
func RemoteRunnerCIVisudoCheckScript() string {
	return `set -euo pipefail
TMP=$(sudo mktemp /tmp/runnerkit-runner-ci.XXXXXX)
printf '%s' "$RUNNERKIT_CI_SUDOERS_CONTENT" | sudo tee "$TMP" >/dev/null
sudo chmod 0440 "$TMP"
if ! sudo visudo -cf "$TMP"; then
  sudo rm -f "$TMP"
  echo "visudo validation failed; CI sudoers entry not installed" >&2
  exit 21
fi
sudo mv "$TMP" ` + RunnerCISudoersFilePath + `
sudo chmod 0440 ` + RunnerCISudoersFilePath + `
sudo chown root:root ` + RunnerCISudoersFilePath + `
`
}

func remoteRunnerCISudoersReadScript() string {
	return `set -euo pipefail
if [ -f ` + RunnerCISudoersFilePath + ` ]; then
  sudo cat ` + RunnerCISudoersFilePath + `
else
  exit 1
fi
`
}

// RunnerCISudoersIsPrepared reports whether the CI sudoers file matches the
// rendered entry for serviceUser.
func RunnerCISudoersIsPrepared(ctx context.Context, exec remote.Executor, target remote.Target, serviceUser string) (bool, error) {
	if exec == nil {
		return false, nil
	}
	result, err := exec.Run(ctx, target, remote.Command{
		ID:     "read_ci_sudoers",
		Script: remoteRunnerCISudoersReadScript(),
	})
	if err != nil || result.ExitCode != 0 {
		return false, nil
	}
	return strings.TrimSpace(result.Stdout) == strings.TrimSpace(RenderRunnerCISudoersEntry(serviceUser)), nil
}
