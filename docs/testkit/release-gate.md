# Real-job release gate (A-20) and revocation drill

This is step 4 of the [pre-tag checklist](../release-process.md#pre-tag-checklist)
as a runbook. No release may say BYO works unless, for that commit, a real
GitHub job ran on a RunnerKit runner on a fresh password-sudo Ubuntu 24.04
x86_64 host prepared **only** by that commit's `install.sh`. The same host
then checks that the revocation steps in
[security-posture.md](../security-posture.md#if-you-already-installed-runnerkit)
work as written.

Read the [kit overview](README.md) first for what you need and the safety
rules.

## Part 1: prepare

### 1. Build the candidate

Use a clean checkout of the exact commit you intend to tag.

```bash
git fetch origin
git checkout --detach origin/main
git status --porcelain            # must print nothing
make generate-check               # install.sh's sudoers block matches the Go template
mkdir -p ~/runnerkit-testkit
go build -o ~/runnerkit-testkit/runnerkit ./cmd/runnerkit
git rev-parse HEAD                # the candidate commit; note it
```

Then, in every shell you use for this runbook:

```bash
export RUNNERKIT_BIN=~/runnerkit-testkit/runnerkit
export RUNNERKIT_STATE_DIR=~/runnerkit-testkit/state   # keeps your real RunnerKit state out of it
export RUNNERKIT_NO_UPDATE_NOTIFIER=1
```

The candidate reports its version as `dev`. `gate.sh` records the commit
and whether the checkout had local changes; a run from a changed checkout
does not count for a tag.

Also check that `RunnerVersion` in `internal/bootstrap/package.go` is
still the latest [actions/runner release](https://github.com/actions/runner/releases).
If a newer one is out, the runner updates itself before its first job and
G9 fails. Bump the pin on `main` first, then build again:

- in `internal/bootstrap/package.go`: `RunnerVersion`, and for both
  packages the `Filename`, the `URL` and the `SHA256` from the release page;
- the expected file names and hashes in
  `internal/bootstrap/package_test.go`;
- the runner-pin entry in the CHANGELOG, and every doc that names the old
  version: `git grep -n '<old version>' -- README.md CHANGELOG.md docs`.

### 2. Create the test repository

Create a **private** repository under your personal account, for example
`you/rk-gate`, with a README so it has a default branch. Then add the kit's
workflows, except the optional V-1 log (it has a schedule; add it only
when V-1 starts):

```bash
git clone https://github.com/you/rk-gate.git ~/runnerkit-testkit/rk-gate
mkdir -p ~/runnerkit-testkit/rk-gate/.github/workflows
cp scripts/testkit/workflows/rk-gate.yml scripts/testkit/workflows/rk-probe.yml \
   scripts/testkit/workflows/rk-fallback.yml scripts/testkit/workflows/rk-token-probe.yml \
   ~/runnerkit-testkit/rk-gate/.github/workflows/
cd ~/runnerkit-testkit/rk-gate
git add .github && git commit -m "RunnerKit test kit workflows" && git push
cd -
```

Actions is on by default for a new private repository. Each GitHub-hosted
job in the kit uses about a minute of your included minutes.

### 3. Create the token

In GitHub, go to **Settings > Developer settings > Fine-grained tokens** and
create a token:

- Resource owner: you. Repository access: only `you/rk-gate`.
- Repository permissions: **Administration**, **Actions** and **Variables**,
  each Read and write. (Metadata: Read is added automatically.)
- Expiration: 30 days.

```bash
read -rs GH_TOKEN && export GH_TOKEN      # paste the token; it is not echoed or kept in history
export RUNNERKIT_GITHUB_TOKEN="$GH_TOKEN"
```

When the GitHub CLI is installed and logged in, RunnerKit uses the token
`gh auth token` returns instead of `RUNNERKIT_GITHUB_TOKEN` (SEC-9);
`gh auth token` returns `GH_TOKEN` when it is set, so RunnerKit and the kit
scripts use the same token either way.

### 4. Get a fresh host

The host must be new, run Ubuntu 24.04 on x86_64, and have an SSH user
whose sudo **asks for a password**. Do not use a user with
`NOPASSWD: ALL`, such as the default `ubuntu` user that cloud-init creates
on many cloud images: it would hide missing `install.sh` entries, and
`gate.sh` fails check G7 for it.

**Option A: cloud VM (works from macOS and Linux).** Create the smallest VM
with 2 vCPU, 4 GB RAM (8 GB is better) and a 20 GB disk from your
provider's Ubuntu 24.04 x86_64 image. Then create the SSH user:

```bash
ssh root@<ip>                 # as root; prefix each command with sudo if you log in as the image's default user
adduser rkadmin               # set a password
usermod -aG sudo rkadmin
install -d -m 700 -o rkadmin -g rkadmin /home/rkadmin/.ssh
install -m 600 -o rkadmin -g rkadmin ~/.ssh/authorized_keys /home/rkadmin/.ssh/authorized_keys
exit
ssh rkadmin@<ip> 'sudo -n true'   # must fail with "a password is required"
```

Accepting the host key here puts it in your `~/.ssh/known_hosts`, which the
scripts need. Compare the fingerprint with the provider's console first.

**Option B: local container (Linux x86_64 workstation with Docker).**

```bash
scripts/testkit/host-container.sh up      # prints the SSH line and the sudo password
ssh -p 2222 rkadmin@127.0.0.1 'sudo -n true'   # must fail; accepts the host key
```

The host is then `rkadmin@127.0.0.1:2222`. The container is privileged
(Docker inside it needs that), so a job on its runner is root on your
workstation's kernel.

### 5. Prepare the host with the candidate install.sh

Copy `install.sh` from the candidate checkout. Do **not** use
`runnerkit init --print-install-command` for this: until v1.3.4 is tagged
it points at the latest release's (v1.3.3's) `install.sh`, which lacks 16
command paths.

```bash
scp install.sh rkadmin@<ip>:/tmp/runnerkit-install.sh      # scp -P 2222 ... for option B
sha256sum install.sh                                       # shasum -a 256 on macOS
ssh -t rkadmin@<ip> 'sha256sum /tmp/runnerkit-install.sh && sudo bash /tmp/runnerkit-install.sh'
```

The two hashes must match. sudo asks for the password once; `install.sh`
writes `/etc/sudoers.d/runnerkit-installer`, which is root-equivalent
(SEC-1).

## Part 2: run the gate

### 6. Set up the runner

Run RunnerKit from an empty directory, so it does not pick up another
checkout's workflows as extra packages:

```bash
mkdir -p ~/runnerkit-testkit/empty ~/runnerkit-testkit/you-rk-gate
cd ~/runnerkit-testkit/empty
time "$RUNNERKIT_BIN" up --repo you/rk-gate --host rkadmin@<ip> 2>&1 | tee ~/runnerkit-testkit/you-rk-gate/up.log
cd -
```

Check the `SHA256:` host-key fingerprint RunnerKit shows against the
host's (`ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` on the host)
before you accept it. Setup takes several minutes on a small VM: the runner
image downloads about 1.4 GB. Note the duration for `EVIDENCE.md`.

### 7. Run gate.sh

From the RunnerKit checkout:

```bash
scripts/testkit/gate.sh --repo you/rk-gate --host rkadmin@<ip>
```

It reads the runner's name and labels from `runnerkit status --json`,
dispatches `rk-gate.yml` on those labels, waits for the job and checks:

| # | Check | Pass when |
| --- | --- | --- |
| G1 | Runner online on GitHub with the RunnerKit labels | The GitHub runner record is `online` and has every label |
| G2 | The job succeeds on the RunnerKit runner | The job's conclusion is `success` and GitHub names this runner |
| G3 | gcc works | The job compiled and ran `hello.c` |
| G4 | Docker works | The job ran `docker run hello-world` as `runnerkit-runner` |
| G5 | Docker group (SEC-5) | `id -nG runnerkit-runner` on the host contains `docker` |
| G6 | Platform | The host is Ubuntu 24.04 x86_64 |
| G7 | Password-sudo host prepared only by install.sh | `/etc/sudoers.d/runnerkit-installer` is byte-identical to what the candidate `install.sh` writes, and `sudo -l` shows no `NOPASSWD: ALL` |
| G9 | The runner took the job at RunnerKit's pin | GitHub and the job log report `RunnerVersion` from `internal/bootstrap/package.go`, `bin` is still the installed directory and `_diag` has no `SelfUpdate` log |

It also runs `runnerkit status`, `doctor` and `list` with `--json`,
records host facts (install directory owners, the image-setup marker, disk
use) and the tool inventory the job printed, and writes everything with
`EVIDENCE.md` to `~/runnerkit-testkit/you-rk-gate/<time>-gate-gate/`.

### 8. Record the result

**Every row PASS:** fill in the manual checks at the end of `EVIDENCE.md`
(host provider and creation time, `sudo -n true` failing before
`install.sh`, the `install.sh` hash) and paste its CHANGELOG line into the
v1.3.4 section of [CHANGELOG.md](../../CHANGELOG.md). Keep the host for the
validations and the drill.

**Any row FAIL:** start with `"$RUNNERKIT_BIN" doctor --repo you/rk-gate --deep`
and `"$RUNNERKIT_BIN" logs --repo you/rk-gate --since 30m`. Fix the cause on
`main`, then repeat from step 1 on a **new** host: a host that already ran
an earlier candidate is not fresh. If it cannot pass by 2026-10-10, take
the fallback in the [pre-tag checklist](../release-process.md#pre-tag-checklist):
BYO `up`/`register` refuse without `--accept-known-issues`, and the README
and CHANGELOG say BYO is not supported in this release.

Next, run V-2, V-4 and V-3 from [validations.md](validations.md) while the
runner is online and `install.sh`'s sudo is still in place.

## Part 3: revocation drill

Run this after V-2, V-4 and V-3, on the same host. It follows
[If you already installed RunnerKit](../security-posture.md#if-you-already-installed-runnerkit)
exactly as published, so it tests the page.

### 9. Apply steps 1 to 3 on the host

```bash
ssh -t rkadmin@<ip>
```

Then, on the host, run steps 1, 2 and 3 from the page as written (step 4 is
cloud-only). sudo asks for your password for anything outside the
fragment, and for everything once step 1 has removed it.

After step 2, check as the runner user that `svc.sh` and `bin` can no
longer be replaced. Both commands must fail with `Permission denied`:

```bash
d=$(ls -d /opt/actions-runner/runnerkit-*-local)
sudo -u runnerkit-runner mv "$d/svc.sh" "$d/svc.sh.orig"
sudo -u runnerkit-runner mv "$d/bin" "$d/bin.orig"
```

After step 3 the runner service restarts; wait until the repository's
**Settings > Actions > Runners** page shows it as Idle again.

### 10. Check from GitHub

```bash
scripts/testkit/gate.sh --repo you/rk-gate --host rkadmin@<ip> --after-revocation
```

This dispatches `rk-gate.yml` in `revoked` mode. The job must still pass
(G1 to G3 and G6 as before) while:

| # | Check | Pass when |
| --- | --- | --- |
| G4 | Docker refused (step 3) | `docker run hello-world` fails inside the job |
| G5 | Docker group removed (step 3) | `id -nG runnerkit-runner` lacks `docker` |
| G7 | No passwordless sudo (step 1) | `sudo -n -l` fails for the SSH user |
| G8 | Install directory locked (step 2) | The job cannot rename `svc.sh` or `bin` or create a file in the install directory, and all three are `root:root` |

If `runnerkit status` cannot read the runner after the drill, pass
`--runner-name` and `--runs-on` with the values from the first
`EVIDENCE.md`.

Fill in the drill's manual checks in its `EVIDENCE.md`. The pre-tag
checklist's revocation gate is met when every row passes; if a step did not
work as written, fix [security-posture.md](../security-posture.md) before the
tag.

Then start V-1 ([validations.md](validations.md#v-1-offline-runner)), which
stops this runner for good.

## Part 4: tag

With the gate and the drill green, finish the rest of the
[pre-tag checklist](../release-process.md#pre-tag-checklist) (tap token,
govulncheck, CHANGELOG) and push the tag from the upstream repository as
described there.
