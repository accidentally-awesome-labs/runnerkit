# Troubleshooting: SSH

Stable codes for this component: `RKD-SSH-001`..`RKD-SSH-004`. Anchors are
stable across renames (D-15).

***

<a name="rkd-ssh-001"></a>
## RKD-SSH-001: SSH host key fingerprint mismatch

**Severity:** error
**Component:** ssh

### Symptom

`runnerkit up`, `runnerkit doctor`, `runnerkit recover`, or any SSH-touching
command fails with:

```
RKD-SSH-001: SSH host key fingerprint mismatch
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/ssh.md#rkd-ssh-001
```

### Diagnosis

The saved host-key fingerprint in `state.json` does not match the fingerprint
the host now presents. RunnerKit fails closed (Phase 2 host-key trust contract)
because a fingerprint change can mean either:

- The host was reinstalled (legitimate; you must reaccept).
- The host was replaced (legitimate, but verify with the operator).
- A man-in-the-middle is intercepting your SSH connection (not legitimate).

### Fix

If the change is intentional and you have verified the new fingerprint out of
band:

```bash
runnerkit recover --repo owner/repo --reaccept-host-key --yes
```

If the change is unexpected: stop. Do not reaccept. Investigate compromise
before continuing — verify the fingerprint with the host operator over a
trusted channel (phone, in-person, signed message).

***

<a name="rkd-ssh-002"></a>
## RKD-SSH-002: SSH host unreachable

**Severity:** error
**Component:** ssh

### Symptom

`runnerkit up`, `runnerkit doctor`, or `runnerkit recover` fails with:

```
RKD-SSH-002: SSH host unreachable
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/ssh.md#rkd-ssh-002
```

### Diagnosis

The CLI cannot establish an SSH session to the saved host. Common causes:
network partition, host powered off, firewall blocking egress, sshd not
running, sshd configured on a non-standard port.

### Fix

Reproduce with verbose SSH:

```bash
ssh -v user@host
```

Walk through the output: connection refused → sshd is down or firewall is
blocking; permission denied → key/credential issue (see RKD-SSH-003); timeout
→ network or DNS.

Then re-run `runnerkit doctor --repo owner/repo` to confirm the fix.

***

<a name="rkd-ssh-003"></a>
## RKD-SSH-003: SSH private key file not found

**Severity:** error
**Component:** ssh

### Symptom

`runnerkit up --ssh-key path/to/key` fails with:

```
RKD-SSH-003: SSH private key file not found
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/ssh.md#rkd-ssh-003
```

### Diagnosis

The path passed to `--ssh-key` (or saved in state) does not exist or is not
readable by the current user.

### Fix

```bash
ls -l <key-path>           # confirm the file exists
chmod 600 <key-path>       # SSH refuses to use overly-readable private keys
```

Pass an absolute path (relative paths are resolved against the cwd of the
RunnerKit invocation):

```bash
runnerkit up --repo owner/repo --host user@host --ssh-key "$HOME/.ssh/runnerkit_ed25519"
```

***

<a name="rkd-ssh-004"></a>
## RKD-SSH-004: SSH port unreachable

**Severity:** error
**Component:** ssh

### Symptom

`runnerkit up --host user@host --ssh-port 2222` fails with:

```
RKD-SSH-004: SSH port unreachable
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/ssh.md#rkd-ssh-004
```

### Diagnosis

Nothing is listening on the configured SSH port, or a firewall is dropping
connections to it. The default port is 22; if your sshd listens elsewhere,
pass `--ssh-port`.

### Fix

```bash
nc -zv <host> 22         # default
nc -zv <host> 2222       # custom port
```

If sshd listens on a non-standard port, pass it explicitly:

```bash
runnerkit up --repo owner/repo --host user@host --ssh-port 2222
```

If a firewall sits between you and the host, allow your source IP on the SSH
port (cloud security groups, on-prem firewall rules, etc.).
