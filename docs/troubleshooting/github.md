# Troubleshooting: GitHub Runner

Stable codes for this component: `RKD-GH-001`..`RKD-GH-008`. Anchors are
stable across renames (D-15).

***

<a name="rkd-gh-001"></a>
## RKD-GH-001: GitHub reports runner offline

**Severity:** warning
**Component:** github

### Symptom

`runnerkit doctor --repo owner/repo` warns:

```
RKD-GH-001: GitHub reports runner offline
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-001
```

### Diagnosis

GitHub reports the saved runner's status as `offline`. The host may be
powered off, the systemd service may be stopped, or network egress to
GitHub is blocked.

### Fix

```bash
runnerkit logs --repo owner/repo --since 30m
runnerkit recover --repo owner/repo --restart-service --yes
```

If the runner stays offline after a restart, run `runnerkit doctor` again —
a more specific code (RKD-BOOT-003 service failed, RKD-AUTH-002 network
unreachable) tells you the underlying cause.

***

<a name="rkd-gh-002"></a>
## RKD-GH-002: Multiple RunnerKit runner candidates found in GitHub

**Severity:** error
**Component:** github

### Symptom

`runnerkit doctor` errors with:

```
RKD-GH-002: Multiple RunnerKit runner candidates found in GitHub
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-002
```

### Diagnosis

More than one runner in the repository has the `runnerkit-owner-repo`
label. This usually means a prior `runnerkit up` partially failed and left
a stale registration behind.

### Fix

```bash
runnerkit down --repo owner/repo --dry-run
runnerkit down --repo owner/repo --yes
runnerkit up --repo owner/repo --host user@host
```

If `down` cannot resolve the duplicates, list and delete the stale
registrations from the GitHub UI:
`Settings → Actions → Runners → … → Remove`.

***

<a name="rkd-gh-003"></a>
## RKD-GH-003: Saved labels drift from GitHub-reported labels

**Severity:** warning
**Component:** github

### Symptom

`runnerkit doctor` warns:

```
RKD-GH-003: Saved labels drift from GitHub-reported labels
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-003
```

### Diagnosis

Someone edited the runner's labels directly in the GitHub UI, or a second
RunnerKit invocation re-registered the same name with a different label
set.

### Fix

```bash
runnerkit recover --repo owner/repo --reregister --dry-run
runnerkit recover --repo owner/repo --reregister --yes
```

This re-registers the runner with the canonical RunnerKit labels.

***

<a name="rkd-gh-004"></a>
## RKD-GH-004: Failed to create runner registration token

**Severity:** error
**Component:** github

### Symptom

`runnerkit up` fails with:

```
RKD-GH-004: Failed to create runner registration token
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-004
```

### Diagnosis

GitHub refused to mint a registration token. Common causes: stale gh CLI
auth, fine-grained PAT lacking Administration: Read and write, or the
token has been revoked.

### Fix

Refresh `gh` auth:

```bash
gh auth refresh -h github.com -s repo,workflow
```

If you use a PAT directly, see [RKD-AUTH-004](auth.md#rkd-auth-004) for
the required permissions and regenerate.

***

<a name="rkd-gh-005"></a>
## RKD-GH-005: Runner registration failed

**Severity:** error
**Component:** github

### Symptom

`runnerkit up` fails after preflight with:

```
RKD-GH-005: Runner registration failed
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-005
```

### Diagnosis

The bootstrap script reached the host but the runner-side `config.sh`
registration call failed. Causes: stale token (took too long to use),
network blip mid-registration, or pre-existing registration with the same
name.

### Fix

```bash
runnerkit down --repo owner/repo --yes    # clear partial state
runnerkit up --repo owner/repo --host user@host --yes
```

If a same-named runner already exists in GitHub (from a prior partial
run), `runnerkit down` clears it as part of its cleanup contract.

***

<a name="rkd-gh-006"></a>
## RKD-GH-006: Stale GitHub runner deregistration failed

**Severity:** warning
**Component:** github

### Symptom

`runnerkit down` or `runnerkit destroy` warns:

```
RKD-GH-006: Stale GitHub runner deregistration failed
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-006
```

### Diagnosis

The cleanup flow could not delete a stale runner from GitHub. The token
may lack permission, or the runner was removed concurrently.

### Fix

Delete it manually in `Settings → Actions → Runners → … → Remove`, or
re-run cleanup with explicit targeting:

```bash
runnerkit down --repo owner/repo --github-runner-id <id> --yes
```

***

<a name="rkd-gh-007"></a>
## RKD-GH-007: recover --reregister failed

**Severity:** error
**Component:** github

### Symptom

`runnerkit recover --repo owner/repo --reregister` fails with:

```
RKD-GH-007: recover --reregister failed
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-007
```

### Diagnosis

`recover --reregister` combines a deregistration step with a fresh
registration. Either side can fail; the underlying cause is one of
RKD-AUTH-004 (token), RKD-GH-004 (token mint), RKD-GH-005 (registration),
or RKD-GH-006 (deregistration).

### Fix

Walk the underlying codes:

```bash
runnerkit doctor --repo owner/repo
```

Apply the more-specific fix first, then retry `recover --reregister`.

***

<a name="rkd-gh-008"></a>
## RKD-GH-008: Self-hosted runner workflow fails — `sudo` needs a password / TTY

**Severity:** info  
**Component:** github

### Symptom

A GitHub Actions workflow running on your **self-hosted** RunnerKit machine
fails on the first step that runs `sudo`, often `sudo apt-get install …`:

```
sudo: a terminal is required to read the password
sudo: a password is required
```

The same workflow succeeds on **`ubuntu-latest`** (hosted).

### Diagnosis

Jobs execute as the runner service user — RunnerKit defaults this to
**`runnerkit-runner`** (`internal/bootstrap/script.go`, `DefaultServiceUser`).
GitHub-hosted runners ship with passwordless `sudo` for their job user; a
plain Linux box usually does **not**. CI has **no TTY**, so `sudo` cannot
prompt.

This is separate from **`install.sh`** for the SSH user: that drop-in covers
the bootstrap/compute surface for the login user, not package-manager sudo for
**`runnerkit-runner`**.

### Fix

Pick one:

**A — Scoped sudoers for package managers (recommended, any Linux distro)**  

On the runner host (root), run `install.sh` with CI grants — same artifact as bootstrap uses for the SSH user:

```bash
sudo RUNNERKIT_GRANT_CI_SUDO=1 RUNNERKIT_SERVICE_USER=runnerkit-runner bash -s < install.sh
```

(or download `install.sh` from the RunnerKit release and pass `RUNNERKIT_GRANT_CI_SUDO=1`). That writes `/etc/sudoers.d/runnerkit-runner-ci` with NOPASSWD only for common package-manager binaries (see `internal/bootstrap/ci_sudoers.go`).

**Manual alternative:** `sudoers` only accepts **absolute paths** to executables. Those paths differ
by distribution (e.g. Alpine’s `apk` is often `/sbin/apk`, not
`/usr/bin/apk`). There is no single file that means “all package managers on
all distros” — you must list each binary you want to allow, using the paths
**on that host**.

1. On the self-hosted runner machine, as a user with root (or in a root shell),
   discover which package tools exist and print a comma-separated list:

   ```bash
   u=runnerkit-runner
   for c in apt-get apt dnf yum microdnf zypper pacman apk; do
     p=$(command -v -- "$c" 2>/dev/null) || continue
     case "$p" in /*) printf '%s\n' "$p";; esac
   done | sort -u | paste -sd, -
   ```

2. Create `/etc/sudoers.d/runnerkit-runner-ci` (mode **0440**, owned by
   **root**). One line, using the comma list from step 1 (example shape only —
   **your** paths must come from the command above):

   ```
   runnerkit-runner ALL=(root) NOPASSWD: /usr/bin/apt-get,/usr/bin/apt,/usr/bin/dnf
   ```

3. Validate and fix permissions:

   ```bash
   sudo chown root:root /etc/sudoers.d/runnerkit-runner-ci
   sudo chmod 0440 /etc/sudoers.d/runnerkit-runner-ci
   sudo visudo -cf /etc/sudoers.d/runnerkit-runner-ci
   ```

4. If workflows also call **`sudo` + something else** (for example
   `systemctl`, `snap`, `rpm`, `zypper` under a path not caught above), add
   those **exact** paths too (`command -v systemctl`, etc.), or use fix **B**
   / **C** below.

**Coverage notes**

| Family | Typical tools (discover with `command -v`) |
| ------ | ------------------------------------------- |
| Debian / Ubuntu | `apt-get`, `apt` |
| RHEL / Fedora / derivatives | `dnf`, `yum`, sometimes `microdnf` |
| openSUSE / SLES | `zypper` (path may be `/usr/sbin/zypper`) |
| Arch | `pacman` |
| Alpine | `apk` (often `/sbin/apk`) |

Unusual or immutable systems (NixOS, Guix, minimal read-only images): prefer
**B** — install dependencies outside the workflow or run jobs in a container
image that already includes tools so CI never needs `sudo`.

**B — Avoid `sudo` in CI**  
Pre-install packages on the host (or use a container job whose image already
contains dependencies) so workflow steps do not invoke `sudo`.

**C — Match hosted runners (broad)**  
Some teams use `runnerkit-runner ALL=(ALL) NOPASSWD: ALL`. Easiest for
workflows that call many privileged commands; weakest isolation — prefer A
when possible.

***
