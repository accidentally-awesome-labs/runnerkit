package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

// SudoersFilePath is the canonical absolute path of the scoped sudoers
// entry that `runnerkit byo-prepare` installs (Plan 06-06 Path C). The
// file is owned by root, mode 0440, and grants the SSH user passwordless
// sudo for the minimum command set required by `runnerkit up` bootstrap.
const SudoersFilePath = "/etc/sudoers.d/runnerkit-installer"

// RenderSudoersEntry renders the scoped NOPASSWD sudoers content for
// the given SSH user. The output is byte-stable so the idempotency
// check in SudoersIsPrepared can compare against the on-disk content.
//
// Command set per gap doc 06-GAP-byo-sudo-handling.md lines 194-202:
//   - apt-get / dnf / yum (package install for fix_dependencies)
//   - useradd (create_runner_user)
//   - install (install -d -o serviceUser for download_runner)
//   - tar (tar xzf for download_runner)
//   - systemctl (service control)
//   - /opt/actions-runner/runnerkit-*/svc.sh (the runner service helper at its
//     real runtime path — see Bug 27 below)
//
// Bug 27 (Plan 06-11, 2026-05-06): the svc.sh entry was previously the
// literal `/opt/runnerkit-runner/svc.sh`, but RunnerKit installs the
// runner under `/opt/actions-runner/runnerkit-<owner>-<repo>-local/`
// (see install.go RenderInstallScript). The literal path never matched
// the actual runtime path, so `verify_service` (`cd $InstallPath &&
// sudo ./svc.sh status`) required Path B password threading at runtime
// even on Path C-prepared hosts — defeating the "one-time prepare"
// promise.
//
// The fix uses a sudoers `*` wildcard. Sudoers `*` does NOT match `/`,
// so `runnerkit-*/svc.sh` is bounded to a single directory level under
// `/opt/actions-runner/` and cannot escape into other directories. The
// safety bounds match the original literal entry.
//
// Critical: this is NOT a blanket NOPASSWD ALL. The user retains
// password-protected sudo for everything else.
//
// Caller MUST ensure user is the SSH user from a previously-validated
// remote.Target. No sanitization is done here.
func RenderSudoersEntry(user string) string {
	return fmt.Sprintf(`# /etc/sudoers.d/runnerkit-installer (managed by runnerkit byo-prepare)
%s ALL=(root) NOPASSWD: \
  /usr/bin/apt-get, /usr/bin/dnf, /usr/bin/yum, \
  /usr/sbin/useradd, \
  /usr/bin/install, \
  /bin/tar, /usr/bin/tar, \
  /bin/systemctl, /usr/bin/systemctl, \
  /opt/actions-runner/runnerkit-*/svc.sh
`, user)
}

// RemoteVisudoCheckScript renders the remote shell script that
// (a) writes the proposed sudoers content from $RUNNERKIT_SUDOERS_CONTENT
// to a tempfile under /tmp (mode 0440), (b) validates with
// `sudo visudo -cf <tmp>`, (c) ATOMICALLY renames into SudoersFilePath
// on success, and (d) bails with `exit 21` on visudo failure WITHOUT
// touching SudoersFilePath.
//
// Critical: the visudo step MUST run BEFORE the mv. A malformed
// sudoers file persisted to /etc/sudoers.d/ can lock the user out of
// sudo entirely; the visudo gate is the only thing preventing that.
//
// Bug 5 (Plan 06-07 attempt-3, 2026-05-05) — the staging tempfile
// MUST be created via `sudo mktemp` so that the file is root-owned
// from the start. On Ubuntu 24.04 LTS (kernel hardening default
// fs.protected_regular=2) a tempfile created by an unprivileged user
// in /tmp cannot be O_CREAT-opened by root because /tmp is sticky and
// world-writable. The protection applies to root, NOT just the file
// owner; subsequent `sudo tee` then fails with EACCES.
func RemoteVisudoCheckScript() string {
	return `set -euo pipefail
TMP=$(sudo mktemp /tmp/runnerkit-installer.XXXXXX)
printf '%s' "$RUNNERKIT_SUDOERS_CONTENT" | sudo tee "$TMP" >/dev/null
sudo chmod 0440 "$TMP"
if ! sudo visudo -cf "$TMP"; then
  sudo rm -f "$TMP"
  echo "visudo validation failed; sudoers entry not installed" >&2
  exit 21
fi
sudo mv "$TMP" ` + SudoersFilePath + `
sudo chmod 0440 ` + SudoersFilePath + `
sudo chown root:root ` + SudoersFilePath + `
`
}

// RemoteSudoersReadScript reads the existing sudoers entry (if any)
// and emits its content on stdout. ExitCode 0 means the file exists
// and stdout is its content; ExitCode 1 means absent. Used by
// SudoersIsPrepared for the idempotency comparison.
func RemoteSudoersReadScript() string {
	return `set -euo pipefail
if [ -f ` + SudoersFilePath + ` ]; then
  sudo cat ` + SudoersFilePath + `
else
  exit 1
fi
`
}

// RemoteSudoersRemoveScript renders the script that removes the
// scoped sudoers entry. Used by `runnerkit byo-prepare --remove`.
func RemoteSudoersRemoveScript() string {
	return `set -euo pipefail
sudo rm -f ` + SudoersFilePath + `
`
}

// SudoersIsPrepared returns true when the remote sudoers file exists
// AND its content (trimmed) matches what RenderSudoersEntry(user)
// would produce. Used by:
//   - byo-prepare to short-circuit re-runs ("already prepared")
//   - doctor to emit the byo_host_prepared finding
//
// Missing file is NOT an error — returns (false, nil).
func SudoersIsPrepared(ctx context.Context, exec remote.Executor, target remote.Target, user string) (bool, error) {
	if exec == nil {
		return false, nil
	}
	result, err := exec.Run(ctx, target, remote.Command{
		ID:     "read_sudoers",
		Script: RemoteSudoersReadScript(),
	})
	if err != nil || result.ExitCode != 0 {
		return false, nil
	}
	return strings.TrimSpace(result.Stdout) == strings.TrimSpace(RenderSudoersEntry(user)), nil
}
