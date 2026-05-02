# Troubleshooting: Cleanup, State, and CLI Input

Stable codes for this component:

- `RKD-CLEAN-001`..`RKD-CLEAN-005` — `down` / `destroy` cleanup paths.
- `RKD-STATE-001`..`RKD-STATE-004` — local state JSON, backup, migration, schema.
- `RKD-CORE-001`..`RKD-CORE-002` — generic CLI input failures.

Anchors are stable across renames (D-15).

***

<a name="rkd-clean-001"></a>
## RKD-CLEAN-001: Cleanup checkpoints or notes are pending

**Severity:** warning
**Component:** cleanup

### Symptom

`runnerkit doctor --repo owner/repo` warns:

```
RKD-CLEAN-001: Cleanup checkpoints or notes are pending
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/cleanup.md#rkd-clean-001
```

### Diagnosis

A prior `runnerkit down` or `runnerkit destroy` did not finish all of its
steps and saved checkpoints describing what is still to do. RunnerKit
keeps these checkpoints across runs so cleanup is resumable.

### Fix

```bash
runnerkit down --repo owner/repo --dry-run         # BYO cleanup
runnerkit destroy --repo owner/repo --dry-run      # cloud cleanup
runnerkit down --repo owner/repo --yes
runnerkit destroy --repo owner/repo --yes
```

After cleanup completes, the warning clears.

***

<a name="rkd-clean-002"></a>
## RKD-CLEAN-002: Ephemeral cleanup checkpoints are pending

**Severity:** warning
**Component:** cleanup

### Symptom

`runnerkit doctor` warns the ephemeral runner's cleanup finalizer (log
preservation, registration removal, file cleanup) did not complete.

### Diagnosis

Phase 5 finalizers preserve `_diag` archives before deleting the runner.
A finalizer step failed — typically log archive permissions, GitHub
deregistration after TTL expiry, or a partial `down` against an
in-progress ephemeral runner.

### Fix

Re-run cleanup; checkpoints make this safe:

```bash
runnerkit down --repo owner/repo --yes        # BYO ephemeral
runnerkit destroy --repo owner/repo --yes     # cloud ephemeral
```

If logs are critical and the finalizer cannot preserve them, archive them
manually before forcing cleanup:

```bash
ssh user@host 'sudo tar czf /tmp/_diag.tar.gz /var/lib/runnerkit/ephemeral/<runner>/logs'
scp user@host:/tmp/_diag.tar.gz ./
```

***

<a name="rkd-clean-003"></a>
## RKD-CLEAN-003: down: file removal failed

**Severity:** error
**Component:** cleanup

### Symptom

`runnerkit down` fails with:

```
RKD-CLEAN-003: down: file removal failed
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/cleanup.md#rkd-clean-003
```

### Diagnosis

A managed path (`/opt/actions-runner/...`, `/var/lib/runnerkit/...`,
the systemd unit) could not be removed. Causes: permission denied,
file in use by the runner service, or the SSH user lost sudo capability.

### Fix

```bash
runnerkit down --repo owner/repo --yes
```

If the second attempt fails identically, remove the managed paths by
hand and re-run `down` to clear the local state:

```bash
ssh user@host '
  sudo systemctl stop runnerkit-runner || true
  sudo rm -rf /opt/actions-runner/runnerkit-* /var/lib/runnerkit
  sudo rm -f /etc/systemd/system/runnerkit-runner.service
  sudo systemctl daemon-reload
'
runnerkit down --repo owner/repo --yes
```

***

<a name="rkd-clean-004"></a>
## RKD-CLEAN-004: Ephemeral log preservation failed

**Severity:** warning
**Component:** cleanup

### Symptom

`runnerkit down` or `runnerkit destroy` warns the ephemeral log
preservation step did not produce the expected archive.

### Diagnosis

The runner user does not have write permission on
`/var/lib/runnerkit/ephemeral/<runner>/logs`, or the directory was
deleted out of band.

### Fix

```bash
ssh user@host '
  sudo install -d -o runnerkit-runner -g runnerkit-runner -m 0750 \
    /var/lib/runnerkit/ephemeral
'
```

Then re-run cleanup. If logs are needed for debugging and preservation
keeps failing, copy them manually before forcing cleanup (see
[RKD-CLEAN-002](#rkd-clean-002)).

***

<a name="rkd-clean-005"></a>
## RKD-CLEAN-005: destroy: partial cleanup, checkpoints retained

**Severity:** warning
**Component:** cleanup

### Symptom

`runnerkit destroy --repo owner/repo` ends with:

```
RKD-CLEAN-005: destroy: partial cleanup, checkpoints retained
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/cleanup.md#rkd-clean-005
```

### Diagnosis

Some destroy steps (provider, GitHub, remote, local state) succeeded but
at least one did not. RunnerKit keeps cleanup checkpoints until everything
verifies clean.

### Fix

Look at `runnerkit doctor` to see which step is still pending (likely
[RKD-PROV-006](provider.md#rkd-prov-006), [RKD-GH-006](github.md#rkd-gh-006),
or [RKD-CLEAN-003](#rkd-clean-003) above). Apply the relevant fix, then
re-run:

```bash
runnerkit destroy --repo owner/repo --yes
```

***

<a name="rkd-state-001"></a>
## RKD-STATE-001: state.json is not valid JSON

**Severity:** error
**Component:** state

### Symptom

Any `runnerkit` command that reads state fails with:

```
RKD-STATE-001: state.json is not valid JSON
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/cleanup.md#rkd-state-001
```

### Diagnosis

The state file (`$XDG_STATE_HOME/runnerkit/state.json` or
`$HOME/.local/state/runnerkit/state.json`) is not parseable JSON. This
usually means a manual edit went wrong, the file was truncated by a
crash, or another tool wrote into the path.

### Fix

If a sibling backup exists, restore it:

```bash
ls -lt $HOME/.local/state/runnerkit/state.json.backup-v*-Z 2>/dev/null
cp $HOME/.local/state/runnerkit/state.json.backup-v1-...Z \
   $HOME/.local/state/runnerkit/state.json
```

Otherwise, delete the corrupted file and re-run `runnerkit up` (you will
lose local metadata but the GitHub-side runner can be reattached):

```bash
rm $HOME/.local/state/runnerkit/state.json
runnerkit up --repo owner/repo --host user@host
```

***

<a name="rkd-state-002"></a>
## RKD-STATE-002: state backup write failed

**Severity:** error
**Component:** state

### Symptom

The state migration framework (`runnerkit` startup) fails to write the
side-by-side backup (`state.json.backup-v<N>-<RFC3339Z>`) before
mutating state.

### Diagnosis

Either the state directory is full, the user lost write permission on
it, or the directory was made read-only externally (chmod 0500).

### Fix

```bash
df -h $HOME/.local/state
ls -ld $HOME/.local/state/runnerkit
chmod 0700 $HOME/.local/state/runnerkit    # restore default perms
```

Then retry the failing command. The migration is forward-only and will
re-attempt the backup on next read.

***

<a name="rkd-state-003"></a>
## RKD-STATE-003: state migration failed

**Severity:** error
**Component:** state

### Symptom

A state load fails with:

```
RKD-STATE-003: state migration failed
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/cleanup.md#rkd-state-003
```

### Diagnosis

A forward state migration (e.g., v1 → v2) raised an error. RunnerKit
preserves the original state at `state.json.backup-v<N>-<RFC3339Z>` so
no data is lost.

### Fix

This indicates a bug in the migration. Please file a report with the
RunnerKit version, the exact error message, and (if you are willing)
the redacted backup file. As a workaround, downgrade to the prior
RunnerKit minor version that wrote your current schema and re-attempt
operations there.

***

<a name="rkd-state-004"></a>
## RKD-STATE-004: state schema_version newer than this CLI knows

**Severity:** error
**Component:** state

### Symptom

A state load fails with:

```
RKD-STATE-004: state schema_version newer than this CLI knows
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/cleanup.md#rkd-state-004
```

### Diagnosis

You downgraded RunnerKit (or are running an older binary against state
written by a newer one). RunnerKit refuses to mutate forward-incompatible
state — Phase 6 D-09 contract.

### Fix

Run `runnerkit upgrade` to install a CLI that understands the saved
state:

```bash
runnerkit upgrade
```

If you intentionally want the older binary's behavior, delete the state
and re-run setup, accepting the loss of local metadata:

```bash
rm $HOME/.local/state/runnerkit/state.json
runnerkit up --repo owner/repo --host user@host
```

***

<a name="rkd-core-001"></a>
## RKD-CORE-001: Input required for non-interactive flow

**Severity:** error
**Component:** core

### Symptom

`runnerkit ... --non-interactive` (or any TTY-less invocation) fails with:

```
RKD-CORE-001: Input required for non-interactive flow
See: https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/cleanup.md#rkd-core-001
```

### Diagnosis

A required input (e.g., `--repo`, `--yes`, a typed acknowledgment) was
missing for a non-interactive flow.

### Fix

Re-run with the missing flag. The error message names the specific input
needed; common pairings are:

```bash
runnerkit up --repo owner/repo --host user@host --yes
runnerkit destroy --repo owner/repo --yes
```

***

<a name="rkd-core-002"></a>
## RKD-CORE-002: Invalid CLI input

**Severity:** error
**Component:** core

### Symptom

A `runnerkit` command rejects a flag value (e.g., malformed `--repo`,
unknown `--mode`, bad duration).

### Diagnosis

The CLI validates inputs aggressively and refuses to act on
ambiguous values rather than guessing.

### Fix

```bash
runnerkit <command> --help
```

Re-run with a valid value. For `--repo`, the format is `owner/name`.
For `--mode`, the values are `persistent` or `ephemeral`. For
`--ephemeral-ttl`, the format is a Go duration (`1h`, `24h`, `30m`).
