# Troubleshooting: GitHub Runner

Stable codes for this component: `RKD-GH-001`..`RKD-GH-007`. Anchors are
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
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-001
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
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-002
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
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-003
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
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-004
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
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-005
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
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-006
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
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/github.md#rkd-gh-007
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
