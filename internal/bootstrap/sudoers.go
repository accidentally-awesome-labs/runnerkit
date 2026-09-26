package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

// SudoersFilePath is the canonical absolute path of the installer sudoers
// fragment that `install.sh` installs (cloud-init user-data writes the same
// fragment on RunnerKit-provisioned cloud hosts). The file is owned by
// root, mode 0440, and grants the SSH user passwordless sudo for the
// command set `runnerkit up` bootstrap uses. That command set is
// root-equivalent; see RenderSudoersEntry.
const SudoersFilePath = "/etc/sudoers.d/runnerkit-installer"

// RenderSudoersEntry renders the root-equivalent installer sudoers
// fragment (NOPASSWD) for the given SSH user. The output is byte-stable
// so the idempotency check in SudoersIsPrepared can compare against the
// on-disk content; TestRenderSudoersEntryGolden pins it byte for byte.
//
// Command set per gap docs + smoke regressions:
//   - apt-get / dnf / yum (package install for fix_dependencies)
//   - useradd (create_runner_user)
//   - install + curl + sha256sum + tar + chown + rm + su
//     (download_runner/configure_runner bootstrap flow)
//   - systemctl (service control)
//   - /opt/actions-runner/runnerkit-*/svc.sh (the runner service helper at its
//     real runtime path — see Bug 27 below)
//
// Bug 27 (Plan 06-11, 2026-05-06): the svc.sh entry was previously the
// literal `/opt/runnerkit-runner/svc.sh`, but RunnerKit installs the
// runner under `/opt/actions-runner/runnerkit-<owner>-<repo>-local/`
// (see install.go RenderInstallScript). The literal path never matched
// the actual runtime path, so `verify_service` (`cd $InstallPath &&
// sudo ./svc.sh status`) required password threading at runtime even on
// hosts where the fragment was installed — defeating the one-time host
// install.
//
// The fix uses a sudoers `*` wildcard. Sudoers `*` does NOT match `/`,
// so `runnerkit-*/svc.sh` is bounded to a single directory level under
// `/opt/actions-runner/` and cannot escape into other directories. The
// safety bounds match the original literal entry.
//
// Bug 32 (Plan 06-14, 2026-05-08): preflight probe was fixed to use an
// allowlisted command, but the installer sudoers fragment still omitted several
// commands used by the non-interactive bootstrap path (`sudo curl`,
// `sudo sha256sum -c -`, `sudo chown`, `sudo rm`, `sudo su -s /bin/bash -`).
// In tee/non-PTY smoke execution, preflight passed yet bootstrap failed with
// `sudo: a terminal is required ...` because those commands fell outside the
// fragment and required password prompting. The list below now includes the
// full root-runas command surface used by Apply/RenderInstallScript so a host
// prepared by install.sh works end-to-end in non-interactive runs.
//
// Bug 33 (smoke-discovery 2026-05-18): the GitHub-hosted runner image parity
// step (RenderImageSetupScript) and the ephemeral log-preservation step in
// script.go use `sudo ln`, `sudo chmod`, `sudo cp`, and `sudo cat` for
// post-fix_dependencies work — symlinking Go/chromedriver binaries into
// /usr/local/bin, marking geckodriver executable, preserving ephemeral runner
// _diag logs, and reading back the installed sudoers fragment for verification.
// None of these were in the allowlist, so bootstrap failed at setup_runner_image
// with the same "terminal is required" symptom Bug 32 fixed elsewhere. Adding
// them here closes the bug class for v1.3.x. Narrowing this list does not fix
// the privilege model; a dedicated root-owned helper is planned to replace it.
//
// Root-equivalent: su, tee, cp, apt-get, systemctl etc. with any arguments
// allow a root shell; see docs/security-posture.md.
//
// Caller MUST ensure user is the SSH user from a previously-validated
// remote.Target. No sanitization is done here.
func RenderSudoersEntry(user string) string {
	return fmt.Sprintf(`# /etc/sudoers.d/runnerkit-installer (managed by runnerkit install.sh)
%s ALL=(root) NOPASSWD: \
  /usr/bin/apt-get, /usr/bin/dnf, /usr/bin/yum, \
  /usr/sbin/useradd, \
  /usr/bin/install, \
  /usr/bin/curl, \
  /usr/bin/sha256sum, \
  /usr/bin/tee, /usr/bin/gpg, \
  /bin/mkdir, /usr/bin/mkdir, /usr/bin/unzip, \
  /usr/sbin/usermod, /usr/bin/dpkg, /usr/bin/add-apt-repository, \
  /bin/chown, /usr/bin/chown, \
  /bin/chmod, /usr/bin/chmod, \
  /bin/cp, /usr/bin/cp, \
  /bin/cat, /usr/bin/cat, \
  /bin/ln, /usr/bin/ln, \
  /bin/rm, /usr/bin/rm, \
  /bin/su, /usr/bin/su, \
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
// installer sudoers fragment and the optional CI sudoers drop-in.
// Nothing calls it at present, not even tests: the command that used it
// was removed in v1.0.8. docs/security-posture.md gives the equivalent
// manual `sudo rm -f` for users.
func RemoteSudoersRemoveScript() string {
	return `set -euo pipefail
sudo rm -f ` + SudoersFilePath + ` ` + RunnerCISudoersFilePath + `
`
}

// SudoersIsPrepared returns true when the remote sudoers file exists
// AND its content (trimmed) matches what RenderSudoersEntry(user)
// would produce. Only tests call it at present: the command that used it
// to short-circuit re-runs was removed in v1.0.8, and doctor's
// byo_host_prepared finding only checks that SudoersFilePath exists
// (`test -f`), not its content.
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
