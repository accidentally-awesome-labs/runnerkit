# RunnerKit v1.3.3 Release Notes

Date: 2026-05-18

## Highlights

- **BYO bootstrap fix (Bug 33):** `runnerkit up --host …` failed at the `setup_runner_image` step on every Ubuntu/Debian BYO host using Path C (`runnerkit byo-prepare`) scoped sudoers, because the GitHub-hosted runner image-parity step uses `sudo ln`, `sudo chmod`, `sudo cp`, and `sudo cat` — none of which were in the scoped allowlist. Bootstrap reported only `sudo: a terminal is required to read the password` without naming the offending command. v1.3.3 adds `/bin/ln`, `/usr/bin/ln`, `/bin/chmod`, `/usr/bin/chmod`, `/bin/cp`, `/usr/bin/cp`, `/bin/cat`, `/usr/bin/cat` to `RenderSudoersEntry` so `runnerkit up` completes end-to-end on Path C hosts.
- **`doctor --json` error envelope contract (Bug 33-C):** Previously, `runnerkit doctor --json` against a repo with no saved state returned a stripped envelope (`{ok:false, error:{...}, redactions_applied:true}`) missing `schema_version`, `stage`, `next_actions[]`, and `host_incident_hints[]`. Agent/MCP/SDK consumers reading the JSON contract broke silently. v1.3.3 routes all doctor error paths through `doctorJSONError` so the four contract fields are always present (with `next_actions` and `host_incident_hints` as empty arrays, never null). `scripts/smoke/assert-doctor-json-contract.sh` now exercises the error path.

## Behavior changes

- **Scoped sudoers fragment is wider.** Hosts upgrading from v1.3.2 should re-run `runnerkit byo-prepare --host user@host` to refresh `/etc/sudoers.d/runnerkit-installer` with the four new commands. Hosts that never installed the fragment are unaffected; fresh installs get the wider allowlist automatically.
- **`doctor --json` shape on error paths is additive.** Existing consumers that read `ok`, `error`, and `redactions_applied` continue to work unchanged. New fields (`schema_version`, `stage`, `next_actions`, `host_incident_hints`, `command`) are added alongside.

## Security caveat

The newly allowlisted commands (`ln`, `chmod`, `cp`, `cat`) currently match any path argument. `sudo cp` and `sudo cat` of arbitrary paths are latent privilege-escalation vectors. v1.4.0 (planned next milestone) tightens these to path-scoped sudoers entries (e.g. `/usr/bin/cp /tmp/* /opt/actions-runner/*`) and replaces the SSH-from-laptop scoped-sudoers model with a one-time `runnerkit-install.sh` driven by `runnerkit init` — see `.planning/smoke-discovery-2026-05-18/RECOMMENDATION.md` for full design.

## Docs

- `internal/bootstrap/sudoers.go` — expanded allowlist with Bug 33 rationale comment
- `scripts/smoke/assert-doctor-json-contract.sh` — added error-path contract assertion (`state_not_found` envelope)

## Upgrade path

From [`docs/upgrade.md`](docs/upgrade.md): `runnerkit upgrade` for the CLI. After upgrading, **re-run `runnerkit byo-prepare --host user@host`** on each BYO host so the sudoers fragment includes the four new commands. Cloud (`--cloud hetzner`) hosts are unaffected — cloud-init provisions the latest sudoers fragment on every fresh VM.
