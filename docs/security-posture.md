# Security posture

This page lists every known security weakness in RunnerKit, what it means in
plain terms, and when it is planned to be fixed. These issues are **known
and disclosed**; see [SECURITY.md](../SECURITY.md) for reporting new ones.

Short version:

- **RunnerKit needs root on the runner host.** The sudoers fragment it
  installs for your SSH user is root-equivalent, whatever older docs called
  it.
- **Every job can become root on the runner host** through the `docker`
  group.
- **Ephemeral BYO mode is not isolation.** The host is reused between jobs.
- **Public or untrusted code belongs on GitHub-hosted runners**, which are
  free and unlimited for public repositories. Do not run it on a RunnerKit
  runner.

## Known issues and fix stages

"Stage 2" is the privilege-model rework planned for 2027 only if the project
passes its 2026-12-21 go/kill decision. Until then, treat the issues marked
Stage 2 as permanent.

| ID | Plain statement | Fix stage |
| --- | --- | --- |
| SEC-1 | The sudoers fragment for the SSH user (`/etc/sudoers.d/runnerkit-installer`) is root-equivalent: `sudo su -`, `tee`, `cp`, `chmod`, `apt-get` hooks and `systemctl` with any arguments each give a root shell. It has been root-equivalent since v1.0.8. In v1.3.4, `install.sh` writes the same list as RunnerKit's template (`RenderSudoersEntry`); that adds lines to `install.sh` but no capability. | Stage 2 (replace the list with a root-owned helper) |
| SEC-2 | The runner install directory is owned by the runner user, so a job can rewrite `svc.sh`, which RunnerKit later runs as root (`sudo ./svc.sh`) during setup, recovery and cleanup. | Stage 2 |
| SEC-3 | `/var/lib/runnerkit` is owned by the runner user while root writes into it. | Stage 2 |
| SEC-4 | The SSH host-key pin is advisory: it is not bound to the SSH session that runs commands, and the recorded fingerprint is not in the standard OpenSSH format. `logs` skips the check. | Stage 2 (strict `known_hosts`) |
| SEC-5 | The runner service user is in the `docker` group, which is root-equivalent for every job that runs on the host. Before v1.3.4 this silently did not take effect on fresh hosts (the user was created after Docker setup); from v1.3.4 it does. | Disclosed now; opt-in in Stage 2 |
| SEC-6 | Runner registration tokens appear in process arguments and sudo logs on the host while the runner is configured. They are short-lived (about one hour). | Stage 2 |
| SEC-7 | All repositories registered on one host share one Unix user (`runnerkit-runner`). A job from one repository can read or change another repository's runner files and work directories. | Stage 2 (per-repository users) |
| SEC-8 | The public/fork safety gate is checked only at `up` and `register`. It is not re-checked by `status` or `doctor` when a repository later becomes public. | Stage 2 |
| SEC-9 | When the GitHub CLI is installed and logged in, RunnerKit uses the `gh` token even if `RUNNERKIT_GITHUB_TOKEN` is set, so a narrow fine-grained token can be silently bypassed by a broader one. (Older docs also told you to add the `workflow` scope; it is not needed and that advice is removed.) | Stage 2; docs fixed now |
| SEC-10 | Redaction is split across code paths, file logs (`RUNNERKIT_LOG_DEST=file:…`) are created world-readable (0644), and token detection relies on a narrow pattern. | Stage 2 |
| SEC-11 | Cloud only: the Hetzner firewall allows SSH from `0.0.0.0/0` unless you pass `--ssh-allowed-cidr`, and the cloud admin user (`runnerkit-admin`) has `NOPASSWD:ALL` from cloud-init. | Cloud is frozen behind `--experimental`; fixed only if cloud is revived |
| SEC-12 | The optional CI sudo grant (`install.sh` with `RUNNERKIT_GRANT_CI_SUDO=1`) is root-equivalent: `sudo apt-get -o APT::Update::Pre-Invoke::=<cmd> update` runs any command as root. Any workflow on the host can use it. | Disclosed now; explicit profiles in Stage 2 |
| SEC-13 | The release workflow had no test gate and no repository guard, and uses mutable action tags (`@v4`, not commit SHAs). | Test gate and repository guard in v1.3.4; SHA pinning planned next |

Practical consequences:

- Anyone who can push a workflow to a repository with a RunnerKit runner can
  get root on that runner host (SEC-2, SEC-5, SEC-12).
- Give RunnerKit a host you would give root to that repository's
  contributors. Do not share the host with other workloads or secrets.
- Register repositories on the same host only if they have the same trust
  level (SEC-7).

## Ephemeral mode is not isolation

BYO ephemeral mode (`--mode ephemeral --experimental` with `--host`) gives
each job a one-job GitHub registration on the **same** machine. Files,
packages, processes and credentials left by one job are visible to the next.
It also has known defects (the finalizer runs unprivileged and the TTL is
ignored) and is untested in this release.

Ephemeral cloud runners are disabled in v1.3.4: the VM was never destroyed
after its job and kept billing.

For public, fork-based or otherwise untrusted code, use GitHub-hosted
runners.

## If you already installed RunnerKit

These steps remove what RunnerKit granted on a host. Run them on the runner
host as a user with sudo. They do not need the RunnerKit CLI except for
step 4.

1. **Remove the sudoers fragments** and check that sudo still parses:

   ```bash
   sudo rm -f /etc/sudoers.d/runnerkit-installer /etc/sudoers.d/runnerkit-runner-ci
   sudo visudo -c
   ```

   After this, `runnerkit up`, `register`, `down` and `recover` cannot run
   non-interactively on that host until you run `install.sh` again.

2. **Stop root from executing runner-writable files.** The runner install
   directory is owned by `runnerkit-runner`, so a job can rename `svc.sh` or
   `bin/` and put its own copy in place; the next `sudo ./svc.sh …` then
   runs it as root. Make the directory itself root-owned first (not
   recursive), then the files root runs or reads:

   ```bash
   sudo chown root:root /opt/actions-runner/runnerkit-*/
   sudo chown root:root /opt/actions-runner/runnerkit-*/svc.sh
   sudo chown -R -H root:root /opt/actions-runner/runnerkit-*/bin
   # After a runner self-update, bin is a symlink to bin.<version>; -H
   # follows it. Also cover the versioned copies and the service template
   # inputs, if present:
   sudo chown -R root:root /opt/actions-runner/runnerkit-*/bin.* /opt/actions-runner/runnerkit-*/externals* 2>/dev/null || true
   sudo chown root:root /opt/actions-runner/runnerkit-*/.service 2>/dev/null || true
   ```

   A root-owned install directory stops the runner from updating itself
   (the update writes `bin.<version>` and `externals.<version>` into it).
   Any later `runnerkit up` or `register` on this host runs
   `chown -R runnerkit-runner` on the install directory and undoes this
   step; repeat it afterwards.

   If you do not want to maintain that by hand, remove the runner instead.
   On a host that ran untrusted workflows, do **not** use `runnerkit down`,
   `up`, `recover` or `destroy`, or the manual `sudo ./svc.sh` lines in
   [troubleshooting/cleanup.md](troubleshooting/cleanup.md), before you
   have checked the install directory: they all run `svc.sh` from the
   runner-owned directory as root. Compare `svc.sh` and
   `bin/actions.runner.service.template` against the release tarball
   first, or tear the runner down without running `svc.sh`:

   ```bash
   sudo systemctl stop 'actions.runner.*'
   # systemctl disable takes no pattern, so name the unit files:
   sudo systemctl disable $(systemctl list-unit-files --plain --no-legend 'actions.runner.*' | awk '{print $1}')
   sudo rm /etc/systemd/system/actions.runner.*.service
   sudo systemctl daemon-reload
   # Remove the runner in the repository's Settings -> Actions -> Runners,
   # then delete its directory:
   sudo rm -rf /opt/actions-runner/runnerkit-<runner>
   ```

3. **Review docker-group membership:**

   ```bash
   id -nG runnerkit-runner
   # If you do not want jobs to have root through Docker:
   sudo gpasswd -d runnerkit-runner docker
   sudo systemctl restart 'actions.runner.*'
   ```

   Jobs that use `services:`, `container:` or `docker` then fail on this
   host.

   The next `runnerkit up` that runs image setup adds the membership back
   without asking again. That includes the first v1.3.4 `up` on a host
   whose `/var/lib/runnerkit/image-setup.json` marker is from an older
   release or missing, and any future image-setup version bump. Repeat
   the removal after each such run. `runnerkit up --dry-run` lists
   `setup_runner_image` on Ubuntu/Debian hosts; the step skips itself when
   the marker already matches the current version.

4. **Delete RunnerKit-created Hetzner resources** (cloud only). From the
   workstation:

   ```bash
   runnerkit destroy --repo owner/name --dry-run
   runnerkit destroy --repo owner/name
   ```

   Or delete them in the Hetzner Cloud Console: servers, firewalls, SSH keys
   and primary IPs with the label `runnerkit=true`. Do **not** use
   `runnerkit down` for cloud runners; v1.3.3 and earlier removed the local
   record and left the server billing.

Also rotate any GitHub token or Hetzner API token you used with RunnerKit if
the workstation or runner host may have been exposed.

## Workstation side

- RunnerKit keeps state in `~/.local/state/runnerkit/` (or
  `$RUNNERKIT_STATE_DIR` / `$XDG_STATE_HOME/runnerkit`). It stores hostnames,
  SSH users, runner names and resource IDs, not tokens.
- v1.1.0 through v1.3.3 wrote `config.json` (from `doctor --ignore`) and a
  `sessions/` directory (from BYO `up`/`register`) into the **current
  working directory**. `sessions/` contains `user@host`. If you ran those
  commands inside a git repository, delete those files and make sure they
  were not committed.
