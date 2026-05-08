# Troubleshooting: Cloud Provider (Hetzner)

Stable codes for this component: `RKD-PROV-001`..`RKD-PROV-007`. Anchors
are stable across renames (D-15).

***

<a name="rkd-prov-001"></a>
## RKD-PROV-001: Hetzner provider returned error during status

**Severity:** warning
**Component:** provider

### Symptom

`runnerkit status --repo owner/repo` or `runnerkit doctor` warns:

```
RKD-PROV-001: Hetzner provider returned error during status
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/provider.md#rkd-prov-001
```

### Diagnosis

The Hetzner Cloud API returned an error during a read-only describe call.
Causes: transient API hiccup, expired or revoked HCLOUD_TOKEN, region-wide
incident.

### Fix

Check Hetzner status and retry:

- <https://status.hetzner.com>

```bash
unset HCLOUD_TOKEN; export HCLOUD_TOKEN=...    # rotate the token if needed
runnerkit doctor --repo owner/repo
```

***

<a name="rkd-prov-002"></a>
## RKD-PROV-002: Hetzner resource missing for saved IDs

**Severity:** warning
**Component:** provider

### Symptom

`runnerkit doctor` warns the Hetzner server / volume / SSH key with the
saved ID is no longer present.

### Diagnosis

The cloud resource was deleted out of band — for example, manually in the
Hetzner Console, by another tool, or by a prior partial `runnerkit destroy`.
Saved state still references it.

### Fix

```bash
runnerkit destroy --repo owner/repo --yes        # clean up local state + any other resources
runnerkit up --repo owner/repo --cloud hetzner   # provision afresh
```

***

<a name="rkd-prov-003"></a>
## RKD-PROV-003: Hetzner inventory drift from saved state

**Severity:** warning
**Component:** provider

### Symptom

`runnerkit doctor` warns saved Hetzner inventory (server type, region, SSH
key fingerprint) differs from what the Hetzner API reports.

### Diagnosis

Someone manually changed resource attributes in the Hetzner Console, or
the resource was rebuilt with a different shape.

### Fix

```bash
runnerkit destroy --repo owner/repo --dry-run
runnerkit destroy --repo owner/repo --yes
runnerkit up --repo owner/repo --cloud hetzner
```

***

<a name="rkd-prov-004"></a>
## RKD-PROV-004: HCLOUD_TOKEN environment variable not set

**Severity:** error
**Component:** provider

### Symptom

`runnerkit up --cloud hetzner` fails with:

```
RKD-PROV-004: HCLOUD_TOKEN environment variable not set
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/provider.md#rkd-prov-004
```

### Diagnosis

RunnerKit reads the Hetzner Cloud API token from `HCLOUD_TOKEN` (preferred)
or `HETZNER_CLOUD_TOKEN` (alias). No config-file or state-file discovery
path is used for provider credentials in v1.

### Fix

Create a project-scoped token at:

- <https://console.hetzner.cloud/projects> → select project → Security → API Tokens

Then export it for the current shell:

```bash
export HCLOUD_TOKEN=...
# or: export HETZNER_CLOUD_TOKEN=...
runnerkit up --repo owner/repo --cloud hetzner
```

The token must have **Read & Write** scope; read-only is not enough for
provisioning.

***

<a name="rkd-prov-005"></a>
## RKD-PROV-005: Hetzner project quota exceeded

**Severity:** error
**Component:** provider

### Symptom

`runnerkit up --cloud hetzner` fails partway through provisioning with:

```
RKD-PROV-005: Hetzner project quota exceeded
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/provider.md#rkd-prov-005
```

### Diagnosis

The Hetzner project has hit its server / volume / IP quota. Hetzner
projects start with conservative quotas that grow on request.

### Fix

Either free up resources in the Hetzner Console (delete unused servers /
volumes), or request a quota increase at:

- <https://console.hetzner.cloud/projects> → Support → Request quota increase

Or pick a different region with available capacity:

```bash
runnerkit up --repo owner/repo --cloud hetzner --cloud-region nbg1
```

***

<a name="rkd-prov-006"></a>
## RKD-PROV-006: Hetzner partial destroy — resources remain

**Severity:** warning
**Component:** provider

### Symptom

`runnerkit destroy --repo owner/repo` finishes with:

```
RKD-PROV-006: Hetzner partial destroy — resources remain
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/provider.md#rkd-prov-006
```

### Diagnosis

Some Hetzner resources were deleted but at least one (server, volume, SSH
key) still exists when the verify-destroyed loop polled.

### Fix

```bash
runnerkit destroy --repo owner/repo --yes
```

If the second attempt also leaves resources behind, inspect the Hetzner
Console for orphans matching the `runnerkit-*` naming pattern and delete
them by hand. RunnerKit retains its cleanup checkpoints until verify
succeeds, so re-running `destroy` is safe.

***

<a name="rkd-prov-007"></a>
## RKD-PROV-007: Hetzner resource still billable after destroy

**Severity:** error
**Component:** provider

### Symptom

`runnerkit destroy --repo owner/repo` fails the deferred destroy
verification (D-12 gate 2) with:

```
RKD-PROV-007: Hetzner resource still billable after destroy
See: https://github.com/accidentally-awesome-labs/runnerkit/blob/main/docs/troubleshooting/provider.md#rkd-prov-007
```

after a resource ID that should be 404 still returns 200 from the Hetzner
API past the timeout window.

### Diagnosis

This is the live-smoke gate-2 trip from `06-04` — the deferred destroy
verifier polled until timeout and a `runnerkit-*` resource is still alive
and billing.

### Fix

**Critical:** manually delete the lingering resource in the Hetzner
Console immediately:

- <https://console.hetzner.cloud/projects> → select project → Servers / Volumes / Networks / SSH Keys

Filter for `runnerkit-*` names and delete any that remain. This is a bug —
please file a report with the runner ID, region, and the smoke output so
the destroy contract can be tightened.
