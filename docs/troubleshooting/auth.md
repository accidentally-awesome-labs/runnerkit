# Troubleshooting: GitHub Authentication and Safety

Stable codes for this component: `RKD-AUTH-001`..`RKD-AUTH-004`. Anchors are
stable across renames (D-15).

***

<a name="rkd-auth-001"></a>
## RKD-AUTH-001: Persistent runner on public repository is blocked

**Severity:** error
**Component:** auth

### Symptom

`runnerkit up --repo owner/public-repo --mode persistent` fails with:

```
RKD-AUTH-001: Persistent runner on public repository is blocked
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-001
```

### Diagnosis

Persistent self-hosted runners on public repositories let any pull-request
contributor execute code on the runner host. RunnerKit blocks this by default
(safety policy from Phase 5; see [docs/safety.md](../safety.md)).

### Fix

Use ephemeral cloud (recommended for untrusted workloads):

```bash
runnerkit up --repo owner/public-repo --mode ephemeral --cloud hetzner
```

Or accept the risk explicitly (NOT recommended):

```bash
runnerkit up --repo owner/public-repo --allow-public-repo-risk --yes
```

Read [docs/safety.md](../safety.md) before allowing.

***

<a name="rkd-auth-002"></a>
## RKD-AUTH-002: Cannot reach github.com / api.github.com

**Severity:** error
**Component:** auth

### Symptom

`runnerkit up`, `runnerkit doctor`, or `runnerkit status` fails with:

```
RKD-AUTH-002: Cannot reach github.com / api.github.com
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-002
```

### Diagnosis

Either the host or the runner machine cannot HTTPS-connect to GitHub. RunnerKit
needs egress to `https://github.com` and `https://api.github.com` (and the
runner host needs the same).

### Fix

```bash
# From the host:
curl -fsSL https://api.github.com/zen
curl -fsSL https://github.com

# From the runner machine (replace user@host):
ssh user@host 'curl -fsSL https://api.github.com/zen'
```

If a corporate proxy is required, set `HTTPS_PROXY` and re-run. If a firewall
is blocking egress, allow HTTPS to `github.com` and `api.github.com`.

***

<a name="rkd-auth-003"></a>
## RKD-AUTH-003: Ephemeral BYO on public/fork repo requires acknowledgment

**Severity:** error
**Component:** auth

### Symptom

`runnerkit up --repo owner/public-repo --mode ephemeral` (BYO target) fails
with:

```
RKD-AUTH-003: Ephemeral BYO on public/fork repo requires acknowledgment
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-003
```

### Diagnosis

Ephemeral mode is the recommended path for public/fork workloads, but BYO
ephemeral on a shared host still carries some risk (the host's local
filesystem is touched between job runs even if the runner is one-shot).
Phase 5 requires explicit acknowledgment.

### Fix

Either typed acknowledgment in the interactive prompt, or:

```bash
runnerkit up --repo owner/public-repo --mode ephemeral --allow-ephemeral-byo-risk --yes
```

Or switch to ephemeral cloud for stronger isolation:

```bash
runnerkit up --repo owner/public-repo --mode ephemeral --cloud hetzner
```

***

<a name="rkd-auth-004"></a>
## RKD-AUTH-004: Token lacks runner-management permission

**Severity:** error
**Component:** auth

### Symptom

`runnerkit up` fails during the auth step with:

```
RKD-AUTH-004: Token lacks runner-management permission
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-004
```

### Diagnosis

The token used for auth (gh CLI cached credential or fine-grained PAT) does
not have permission to create runner registration tokens for this repository.

### Fix

If using `gh` CLI:

```bash
gh auth refresh -h github.com -s repo,workflow
```

If using a fine-grained PAT, regenerate at
<https://github.com/settings/tokens?type=beta> with:

- Repository access: `owner/repo` (the target repo)
- Repository permissions: `Administration: Read and write` (required for
  self-hosted runner management)

Then re-run `runnerkit up`.
