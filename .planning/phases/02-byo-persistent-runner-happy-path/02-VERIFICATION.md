---
phase: 02-byo-persistent-runner-happy-path
status: passed
verified: 2026-04-29
requirements:
  [
    CLI-03,
    CLI-04,
    GH-02,
    GH-04,
    GH-05,
    MACH-01,
    MACH-02,
    RUN-01,
    RUN-03,
    DOC-01,
  ]
---

# Phase 02 Verification: BYO Persistent Runner Happy Path

## Result

**Status: passed**

Phase 2 achieved the planned BYO persistent runner happy path at the code and automated-validation level. The CLI now supports SSH target intake, host-key trust, Linux/systemd preflight, non-root bootstrap, GitHub repository runner registration, online verification, persistent state, copy-paste label guidance, safety warnings, and quickstart documentation.

## Requirement Coverage

| Requirement | Status | Evidence                                                                                                                          |
| ----------- | ------ | --------------------------------------------------------------------------------------------------------------------------------- |
| CLI-03      | Passed | `TestUpBYOHappyPathSmoke`, `runnerkit up --repo owner/repo --host alice@example.com --yes` fake path, system SSH executor default |
| CLI-04      | Passed | Human/JSON completion tests include runner name, labels, machine target, service name, next step                                  |
| GH-02       | Passed | Just-in-time `CreateRegistrationToken`, bootstrap env passing, fake online runner verification                                    |
| GH-04       | Passed | Stable `labels.Build` output and exact `runs-on` snippet tests                                                                    |
| GH-05       | Passed | Workflow file unchanged test and docs state RunnerKit does not edit workflow YAML                                                 |
| MACH-01     | Passed | `internal/remote`, host-key tests, `internal/preflight` full check aggregation, system SSH executor default                       |
| MACH-02     | Passed | `internal/bootstrap` package metadata, scripts, non-root service install, command-order tests                                     |
| RUN-01      | Passed | Saved state asserts `mode: persistent`; completion path verifies service and GitHub online runner                                 |
| RUN-03      | Passed | Safety copy and tests block public repo before SSH/preflight/token/bootstrap side effects                                         |
| DOC-01      | Passed | `README.md`, `docs/byo-quickstart.md`, and docs grep test                                                                         |

## Automated Checks

- `go test ./...` — passed
- `go vet ./...` — passed
- `go test ./internal/preflight -run Test` — passed
- Acceptance greps for plan-critical strings — passed

## Must-Have Verification

- SSH target parsing accepts `user@host`, `user@host:port`, and `ssh://user@host:port`; invalid forms are rejected.
- Host-key fingerprints are stored in `MachineRef` and mismatches fail closed before preflight commands.
- Preflight covers SSH connectivity, host key, OS release, arch, systemd, privilege, disk, tools, GitHub network, time, and runner conflicts.
- Bootstrap uses pinned Actions runner `2.334.0`, verifies SHA-256, and installs the service as `runnerkit-runner`.
- Registration tokens are passed through command environment/redaction args and not persisted in state.
- Duplicate GitHub runner names block before fresh registration tokens are requested.
- Completion output includes `BYO runner ready`, runner metadata, labels, state path, and exact workflow guidance.
- Public/fork risk copy is explicit and ordered before remote side effects.
- BYO quickstart docs contain prerequisites, command, safety warning, snippet, completion summary, and troubleshooting.

## Notes

A controlled live GitHub + disposable Linux systemd host smoke remains recommended before public release, but it is not a Phase 2 blocker because the phase validation contract scoped automated coverage to fake adapters plus a production system-SSH executor seam.

## Issues

None.
