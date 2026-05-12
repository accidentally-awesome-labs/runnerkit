# Troubleshooting: Bootstrap and Service

Stable codes for this component: `RKD-BOOT-002`..`RKD-BOOT-018`.
`RKD-BOOT-001` is reserved for future use; numbering is stable across
renames (D-15).

***

<a name="rkd-boot-002"></a>
## RKD-BOOT-002: Bundled runner pin is newer than installed runner

**Severity:** warning
**Component:** bootstrap

### Symptom

`runnerkit doctor --repo owner/repo` warns:

```
RKD-BOOT-002: Bundled runner pin is newer than installed runner
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/bootstrap.md#rkd-boot-002
```

with evidence like `installed runner version 2.330.0 is older than bundled pin 2.334.0`.

### Diagnosis

This RunnerKit release pins a known-good GitHub Actions runner version
(`bootstrap.RunnerVersion`). The runner installed on your host is older than
that pin. GitHub eventually deprecates older runner versions; running stale
risks jobs failing with "the runner version is no longer supported".

### Fix

Roll the runner forward to the bundled pin without re-running full setup:

```bash
runnerkit upgrade-runner --repo owner/repo --yes
```

For ephemeral runners that are currently `waiting` or `busy`, see
[upgrade.md](../upgrade.md#waiting) — `upgrade-runner` refuses without
`--force` to avoid interrupting an in-flight job.

***

<a name="rkd-boot-003"></a>
## RKD-BOOT-003: systemd service failed

**Severity:** error
**Component:** bootstrap

### Symptom

`runnerkit doctor` reports the runner systemd service is in `ActiveState=failed`.

### Diagnosis

The systemd unit RunnerKit installed has crashed or repeatedly failed to
start. Causes include missing binaries, stale registration tokens, or
underlying host issues (disk full, network failure).

### Fix

```bash
ssh user@host 'systemctl status runnerkit-runner'
runnerkit logs --repo owner/repo --since 30m
runnerkit recover --repo owner/repo --reinstall-service --dry-run
runnerkit recover --repo owner/repo --reinstall-service --yes
```

***

<a name="rkd-boot-004"></a>
## RKD-BOOT-004: systemd service missing

**Severity:** error
**Component:** bootstrap

### Symptom

`runnerkit doctor` reports the saved systemd unit is no longer present on
the host (`LoadState=not-found`).

### Diagnosis

The unit file was removed manually, the host was reimaged, or the install
path was deleted out from under RunnerKit.

### Fix

```bash
runnerkit recover --repo owner/repo --reinstall-service --dry-run
runnerkit recover --repo owner/repo --reinstall-service --yes
```

***

<a name="rkd-boot-005"></a>
## RKD-BOOT-005: Runner install directory missing on host

**Severity:** error
**Component:** bootstrap

### Symptom

`runnerkit doctor` reports `/opt/actions-runner/runnerkit-...` (or the saved
install path) is missing.

### Diagnosis

The runner install directory was deleted, the host was reimaged, or the
disk was wiped. RunnerKit cannot reuse the saved install state.

### Fix

```bash
runnerkit recover --repo owner/repo --reregister --dry-run
runnerkit recover --repo owner/repo --reregister --yes
```

This re-runs registration and reinstalls into the saved path.

***

<a name="rkd-boot-006"></a>
## RKD-BOOT-006: Runner work directory missing on host

**Severity:** warning
**Component:** bootstrap

### Symptom

`runnerkit doctor` reports `/var/lib/runnerkit/work/...` (or the saved work
dir) is missing.

### Diagnosis

The runner work directory was deleted out from under the service. The
runner can still register, but jobs that need the work dir will fail.

### Fix

```bash
runnerkit recover --repo owner/repo --reinstall-service --yes
```

This recreates the work dir with the right ownership and permissions.

***

<a name="rkd-boot-007"></a>
## RKD-BOOT-007: Disk space low under /opt or /var/lib

**Severity:** warning
**Component:** bootstrap

### Symptom

Preflight or `runnerkit doctor` warns disk space is low under the runner
install or work directories.

### Diagnosis

The host is approaching disk-full. Runner downloads, container caches, and
job artifacts will start failing.

### Fix

```bash
df -h /opt /var/lib
sudo du -sh /opt/actions-runner/* /var/lib/runnerkit/*
```

Free space, then restart the runner service:

```bash
runnerkit recover --repo owner/repo --restart-service --yes
```

***

<a name="rkd-boot-008"></a>
## RKD-BOOT-008: Required CLI tools missing on host

**Severity:** warning
**Component:** bootstrap

### Symptom

Preflight reports missing tools (e.g., `curl`, `tar`, `systemctl`).

### Diagnosis

The runner host does not have the tools RunnerKit needs to install or
operate the runner. Common on minimal Debian/Ubuntu cloud images.

### Fix

On Debian/Ubuntu:

```bash
ssh user@host 'sudo apt-get update && sudo apt-get install -y curl tar ca-certificates'
```

On other distros, install the equivalent packages, then re-run
`runnerkit up` (or `runnerkit doctor` to re-check preflight).

***

<a name="rkd-boot-009"></a>
## RKD-BOOT-009: Host clock not synchronized (NTP)

**Severity:** warning
**Component:** bootstrap

### Symptom

Preflight or `runnerkit doctor` warns the host clock is unsynchronized.

### Diagnosis

If the runner host's clock drifts more than a few minutes from real time,
TLS handshakes to GitHub fail and registration tokens appear pre-expired.

### Fix

```bash
ssh user@host 'sudo timedatectl set-ntp true && timedatectl status'
```

Re-run `runnerkit doctor --repo owner/repo` to confirm.

***

<a name="rkd-boot-010"></a>
## RKD-BOOT-010: Linux distribution not in supported matrix

**Severity:** warning
**Component:** bootstrap

### Symptom

Preflight refuses to install on a distro RunnerKit has not validated.

### Diagnosis

RunnerKit's Phase 2 supported matrix covers Debian-family systemd Linux
hosts. Other distros may work, but install scripts are best-effort.

### Fix

If you accept the risk and the host has systemd:

```bash
runnerkit up --repo owner/repo --host user@host --allow-unknown-linux --yes
```

Please report the distro you are running on so it can join the supported
matrix.

***

<a name="rkd-boot-011"></a>
## RKD-BOOT-011: Preflight check failed

**Severity:** error
**Component:** bootstrap

### Symptom

`runnerkit up` aborts with a generic preflight failure during the BYO/Cloud
preflight step.

### Diagnosis

A preflight probe (disk, tools, network, time, distro detection) returned
an error result that did not map to a more specific code. The error message
includes the failing probe name.

### Fix

```bash
runnerkit doctor --repo owner/repo
```

`doctor` re-runs every preflight probe with stable IDs so the most-specific
finding (RKD-BOOT-007 / RKD-BOOT-008 / RKD-BOOT-009 / RKD-BOOT-010 / RKD-BOOT-016 / RKD-BOOT-017) tells
you exactly what to fix. When the runner is unhealthy, journal heuristics may add **RKD-BOOT-018** — see [host-resources.md](host-resources.md).

***

<a name="rkd-boot-012"></a>
## RKD-BOOT-012: runnerkit-runner user creation failed

**Severity:** error
**Component:** bootstrap

### Symptom

`runnerkit up` fails on the bootstrap step with:

```
RKD-BOOT-012: runnerkit-runner user creation failed
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/bootstrap.md#rkd-boot-012
```

### Diagnosis

The bootstrap script could not create the dedicated `runnerkit-runner`
service user. Causes: the SSH user does not have sudo without password, or
a conflicting user already exists with a different shell/home.

### Fix

Pre-create the user with the matching shape and re-run:

```bash
ssh user@host '
  sudo useradd --system --create-home --home-dir /var/lib/runnerkit-runner \
    --shell /bin/bash runnerkit-runner
'
runnerkit up --repo owner/repo --host user@host --yes
```

Or arrange passwordless sudo for the SSH user; see your distro's
`/etc/sudoers.d/` documentation.

***

<a name="rkd-boot-013"></a>
## RKD-BOOT-013: Runner tarball install failed

**Severity:** error
**Component:** bootstrap

### Symptom

`runnerkit up` fails downloading or extracting the GitHub Actions runner
tarball.

### Diagnosis

Network egress to `github.com/actions/runner` is blocked, the SHA256 did
not match (corrupted download), or disk filled mid-extract.

### Fix

```bash
ssh user@host 'curl -fsSL https://github.com/actions/runner/releases'
ssh user@host 'df -h /opt /tmp'
```

Once egress and disk are healthy, re-run `runnerkit up`. The bootstrap
script verifies the SHA256 against `bootstrap.RunnerVersion`; a verify
failure means the download was tampered with — try again from a clean shell.

***

<a name="rkd-boot-014"></a>
## RKD-BOOT-014: Runner did not report online before timeout

**Severity:** error
**Component:** bootstrap

### Symptom

`runnerkit up` completes the install but the online-verification step times
out:

```
RKD-BOOT-014: Runner did not report online before timeout
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/bootstrap.md#rkd-boot-014
```

### Diagnosis

Causes: the systemd service crashed shortly after start; egress to GitHub
flapping; clock drift causing the registration token to be rejected; slow
disk preventing service startup.

### Fix

```bash
runnerkit logs --repo owner/repo --since 30m --lines 200
ssh user@host 'systemctl status runnerkit-runner'
runnerkit doctor --repo owner/repo
```

If `doctor` reports a more specific code (RKD-BOOT-003, RKD-BOOT-009,
RKD-AUTH-002), follow that fix first, then re-run `runnerkit up`.

For slower cloud regions/images where cloud-init convergence is delayed, you
can increase the readiness budget:

```bash
export RUNNERKIT_CLOUD_INIT_TIMEOUT=15m
runnerkit up --repo owner/repo --cloud hetzner
```

***

<a name="rkd-boot-015"></a>
## RKD-BOOT-015: Remote sudo requires password

**Severity:** warning
**Component:** bootstrap

### Symptom

`runnerkit up --host user@host` warns during preflight:

```
[warning] host.privilege.password_required: sudo requires a password — run the one-time host install.
```

Or the bootstrap step fails with `bootstrap_failed` and the surfaced
remote stderr contains `sudo: a password is required` /
`sudo: a terminal is required`.

### Diagnosis

The SSH user can run `sudo` but only after entering their password.
RunnerKit's bootstrap commands run over a non-interactive SSH channel
and cannot answer a sudo prompt, so the very first sudo-prefixed
command fails.

### Fix

**Recommended — one-time `install.sh` on the runner host**

From your workstation, print the copy-paste line:

```bash
runnerkit init --print-install-command
```

SSH to the host and run that `curl … install.sh | sudo bash` command (or download `install.sh` from the release linked in your RunnerKit version and verify checksums). This writes `/etc/sudoers.d/runnerkit-installer` with NOPASSWD scoped to RunnerKit bootstrap commands only; `visudo -c` gates the write.

Then re-run `runnerkit up` / `runnerkit register` / `runnerkit down` from the workstation — no sudo password over SSH.

To revert on the host:

```bash
sudo rm -f /etc/sudoers.d/runnerkit-installer
```

See [BYO quickstart — Sudo setup](../byo-quickstart.md#sudo-setup-one-time-on-the-host).

***

<a name="rkd-boot-016"></a>
## RKD-BOOT-016: Low MemAvailable on runner host

**Severity:** warning
**Component:** bootstrap

### Symptom

`runnerkit up` preflight or `runnerkit doctor` shows `host_mem_low` / RKD-BOOT-016 with a message like “Low available memory … below recommended … for heavy native CI.”

### Diagnosis

`/proc/meminfo` `MemAvailable` is below the default warning threshold (4 GiB, or `RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES`). Heavy CI workloads (for example parallel Rust `cargo test` links) can spike RAM and trigger the OOM killer on small VMs.

### Fix

Use a larger machine, add swap, reduce compiler parallelism (for example `CARGO_BUILD_JOBS=1`), split CI jobs, or follow [Host resources and OOM](host-resources.md).

***

<a name="rkd-boot-017"></a>
## RKD-BOOT-017: No swap with constrained RAM

**Severity:** warning
**Component:** bootstrap

### Symptom

Preflight or doctor shows `host_swap_constrained` / RKD-BOOT-017: no swap and `MemAvailable` under 8 GiB.

### Diagnosis

Hosts without swap die abruptly when a single job exceeds RAM; the kernel OOM killer may reap `ld`, `rustc`, or the GitHub runner listener.

### Fix

Add swap or RAM, or reduce peak memory from CI (lighter test profiles, fewer parallel links). See [Host resources and OOM](host-resources.md).

***

<a name="rkd-boot-018"></a>
## RKD-BOOT-018: Likely OOM or hard kill from journal heuristics

**Severity:** warning
**Component:** bootstrap

### Symptom

`runnerkit doctor` prints `host_incident_hints` and/or **Host incident hints** with IDs like `likely_kernel_oom` or `likely_linker_sigkill`, often after the runner went offline or `systemd` shows `failed`.

### Diagnosis

RunnerKit scanned bounded `journalctl` output for common patterns (OOM killer lines, `signal 9` / `SIGKILL`, linker `collect2` killed). This is a **heuristic**, not a guaranteed root cause.

### Fix

Collect full context with `runnerkit logs --repo owner/repo --since 48h`. On the host, check `dmesg` / `journalctl -k` if permitted. Resize the VM, tune CI parallelism, and read [Host resources and OOM](host-resources.md).
