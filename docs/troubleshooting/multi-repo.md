# Multi-repo on one BYO host (SEED-002)

RunnerKit supports **multiple GitHub repositories** on the same Linux machine:
each repo gets its own install directory under `/opt/actions-runner/runnerkit-*`,
its own systemd unit, and shared **`runnerkit-runner`** user plus a **versioned
tarball cache** under `/opt/actions-runner/runnerkit-shared-bin/<version>/`.

## Commands

- **`runnerkit init`** — one-time host install (interactive sudo on the machine).
- **`runnerkit register`** — add another repo after the host is prepared. Fails
  with `lifecycle_foundation_missing` if the shared runner user is missing.
- **`runnerkit list`** / **`runnerkit list --host user@host[:port]`** — inventory
  from local state (`--json` for automation).
- **`runnerkit unregister`** — alias of **`runnerkit down`** (same behavior).

## PAT scope

Easiest: one token with **`repo`** on every repository you register. Narrower
per-repo tokens work if you prefer not to reuse a broad credential.

## Isolation caveats

Self-hosted runners share the host OS and disk. A job in repository A can
read leftover files from repository B if something writes outside the job
workspace. Treat the machine as **trusted across repos** you register there.

## Parallelism

Each persistent runner is a separate service; two repos can run two jobs at
once subject to CPU and RAM (roughly **~150 MiB idle footprint per runner** in
addition to build load).

## Optional maintainer smoke (automation)

BYO live smoke ([`scripts/smoke/byo-permission.sh`](../../scripts/smoke/byo-permission.sh)) supports a **second registration** on the same host when:

- **`RUNNERKIT_SMOKE_MULTI_REPO=1`**
- **`RUNNERKIT_SMOKE_REPO2=owner/other`** — another **trusted private** repo (must differ from the primary `RUNNERKIT_SMOKE_REPO`)

With the gate on, the script runs **`runnerkit register`** for the second repo, asserts **`list --json`** shows **two** repos on that host ([`scripts/smoke/assert-list-host-repo-count.sh`](../../scripts/smoke/assert-list-host-repo-count.sh)), runs the **doctor JSON** contract for the second repo, then **`down`** the second repo before **`down`** on the primary.

`make smoke-live-byo` validates that **`RUNNERKIT_SMOKE_REPO2`** is set whenever **`RUNNERKIT_SMOKE_MULTI_REPO=1`**. See [`.env.example`](../../.env.example).
