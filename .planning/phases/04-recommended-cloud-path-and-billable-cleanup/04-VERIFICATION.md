---
phase: 04-recommended-cloud-path-and-billable-cleanup
verified: 2026-05-01T17:59:31Z
status: passed
score: 5/5 must-haves verified
gaps: []
human_verification: []
requirements: [MACH-03, MACH-04, MACH-05, CLEAN-01, CLEAN-04, DOC-02]
source:
  - 04-01-SUMMARY.md
  - 04-02-SUMMARY.md
  - 04-03-SUMMARY.md
  - 04-04-SUMMARY.md
---

# Phase 04: Recommended Cloud Path and Billable Cleanup Verification Report

**Phase Goal:** Developers without a machine can provision one recommended low-cost cloud runner path, manage it through the same lifecycle as BYO, and destroy billable resources with confidence.
**Verified:** 2026-05-01T17:59:31Z
**Status:** passed

## Goal Achievement

### Observable Truths

| #   | Truth                                                                                                                                                                               | Status     | Evidence                                                                                                                                                                                                                                                                                                                                                                                                                 |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | Developer can provision one recommended low-cost cloud machine path from RunnerKit when they do not already have a machine.                                                         | ✓ VERIFIED | `internal/provider/profile.go` defines the Hetzner default profile; `internal/provider/hetzner/provision.go` creates the VM/SSH key/firewall/network resources; `internal/cli/up.go` gates non-interactive cloud setup on explicit `--cloud hetzner --yes`; `TestUpCloudPlanNoMutationBeforeYes`, `TestUpCloudYesProvisionsAndSavesPendingCheckpoint`, and Hetzner provider tests cover planning and provisioning seams. |
| 2   | Developer can install and manage the cloud runner through the same registration, status, logs, doctor, and cleanup lifecycle as BYO machines.                                       | ✓ VERIFIED | Cloud `up` reuses `bootstrap.Apply`, GitHub registration, and online verification after provider/SSH readiness; `internal/cli/status.go`, `internal/cli/logs.go`, and `internal/cli/doctor.go` expose cloud provider facts without mutation; `internal/cli/destroy.go` adds cloud cleanup while `down` remains BYO.                                                                                                      |
| 3   | RunnerKit state shows enough machine/provider identity to safely manage, reconcile, or remove the cloud runner later.                                                               | ✓ VERIFIED | `internal/state/schema.go` and cloud up paths persist `ProviderRef.Cloud`, resource IDs, runner ID, machine target, managed paths, cleanup IDs, and provider inventory; status/doctor/logs tests use saved provider facts and read-only warnings.                                                                                                                                                                        |
| 4   | Developer can run a cleanup/destroy flow that shows a plan, removes GitHub registration and RunnerKit-created cloud resources, and verifies those resources are no longer billable. | ✓ VERIFIED | `runnerkit destroy --dry-run` uses `ops.BuildCloudDestroyPlan`; `destroy --yes`/typed confirmation call remote cleanup, GitHub deletion, provider `Destroy`, then provider `VerifyDestroyed`; local state removal happens only after successful verification; partial cleanup stores pending checkpoints.                                                                                                                |
| 5   | Developer can follow a concise cloud quickstart for the supported provider path.                                                                                                    | ✓ VERIFIED | `README.md` links `docs/cloud-quickstart.md`; both include Hetzner auth, setup commands, labels, status/logs/doctor, destroy dry-run/apply, cost caveats, Phase 4 limitations, and workflow YAML non-editing guidance; `TestCloudQuickstartDocsContainRequiredCopy` enforces required copy.                                                                                                                              |

**Score:** 5/5 truths verified

## Required Artifacts

| Artifact                                                                     | Expected                                                                            | Status     | Details                                                                                                                                                            |
| ---------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `internal/provider/provider.go`                                              | Cloud lifecycle boundary includes provision, describe, destroy, and verify methods. | ✓ VERIFIED | `Provider` defines `Plan`, `Provision`, `WaitReady`, `Describe`, `Destroy`, and `VerifyDestroyed`; provider results expose billable-resource verification fields.  |
| `internal/provider/profile.go`                                               | Selected default low-cost profile and plan output.                                  | ✓ VERIFIED | Registered Hetzner profile includes region/server/image/SSH user/cost caveat; plan output includes resources, tags, labels, and future destroy command.            |
| `internal/provider/hetzner/provision.go`                                     | Hetzner VM/SSH key/firewall/readiness workflow.                                     | ✓ VERIFIED | Adapter creates and tracks server, SSH key, firewall, primary IP facts, and readiness state through hcloud-go.                                                     |
| `internal/cli/up.go`                                                         | Cloud setup CLI and BYO lifecycle reuse.                                            | ✓ VERIFIED | Cloud path requires explicit cloud intent, renders plan-before-mutation, persists pending checkpoints, then reuses BYO bootstrap/registration/online verification. |
| `internal/ops/status.go` / `internal/ops/doctor.go` / `internal/ops/logs.go` | Provider-aware read-only operations.                                                | ✓ VERIFIED | Operations include provider facts, cloud warnings, provider JSON sources, and drift remediation without mutating provider resources.                               |
| `internal/cli/destroy.go`                                                    | Cloud billable cleanup command.                                                     | ✓ VERIFIED | Contains `Use: "destroy"`, `--dry-run`, `--yes`, typed confirmation, partial cleanup checkpoints, provider verification, and safe state removal.                   |
| `internal/ops/cloud_destroy.go`                                              | Destroy plan and billing warning model.                                             | ✓ VERIFIED | `BuildCloudDestroyPlan` covers GitHub runner, remote runner, provider server/SSH key/firewall/primary IP, local state, and billing warning copy.                   |
| `internal/provider/hetzner/destroy.go`                                       | Hetzner resource deletion and verify-absent behavior.                               | ✓ VERIFIED | Implements server/SSH key/firewall/primary IP deletion, already-absent handling, and `VerifyDestroyed` billable-resource reporting.                                |
| `docs/cloud-quickstart.md` / `README.md`                                     | Supported-provider quickstart and limitations.                                      | ✓ VERIFIED | Docs include provider auth, setup, status/logs/doctor, labels, destroy, cost caveats, limitations, and smoke-test guidance.                                        |

**Artifacts:** 9/9 verified

## Key Link Verification

| From                                                                         | To                                               | Via                                       | Status  | Details                                                                                                                               |
| ---------------------------------------------------------------------------- | ------------------------------------------------ | ----------------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/cli/root.go`                                                       | `internal/cli/up.go` / `internal/cli/destroy.go` | Cobra command registration                | ✓ WIRED | Root wires cloud setup through `up` and cleanup through `newDestroyCommand`.                                                          |
| `internal/cli/up.go`                                                         | `internal/provider/provider.go`                  | Provider registry and cloud profile       | ✓ WIRED | Cloud setup looks up Hetzner provider, validates env credentials, renders provider plan, provisions, and waits for readiness.         |
| `internal/provider/hetzner/provision.go`                                     | `internal/state/schema.go`                       | Provider resource IDs and cloud inventory | ✓ WIRED | Provisioned server, SSH key, firewall, public IPs, region/image/server-type, and tags are persisted for later operations and cleanup. |
| `internal/cli/up.go`                                                         | `internal/bootstrap/install.go`                  | Shared BYO bootstrap path                 | ✓ WIRED | Cloud setup uses the same bootstrap/service/GitHub registration and online verification path as BYO after SSH readiness.              |
| `internal/cli/status.go` / `internal/cli/doctor.go` / `internal/cli/logs.go` | `internal/provider/provider.go`                  | `Describe` and saved provider metadata    | ✓ WIRED | Status/doctor read provider facts through `Describe`; logs expose saved provider metadata without provider mutation.                  |
| `internal/cli/destroy.go`                                                    | `internal/ops/cloud_destroy.go`                  | `BuildCloudDestroyPlan`                   | ✓ WIRED | `destroy --dry-run` and apply payloads share the plan model with billing warnings and artifact ordering.                              |
| `internal/cli/destroy.go`                                                    | `internal/provider/provider.go`                  | `Destroy` followed by `VerifyDestroyed`   | ✓ WIRED | Provider cleanup is applied and then verified before local state can be removed.                                                      |
| `internal/cli/destroy.go`                                                    | `internal/state/store.go`                        | `UpdateRepository` / `RemoveRepository`   | ✓ WIRED | Partial cleanup writes pending destroy checkpoints; complete cleanup removes local state only after GitHub/provider verification.     |
| `README.md`                                                                  | `docs/cloud-quickstart.md`                       | Quickstart link and command consistency   | ✓ WIRED | README links the detailed cloud quickstart and includes the same setup/destroy commands and limitations.                              |

**Wiring:** 9/9 connections verified

## Requirement Coverage

| Requirement | Status      | Evidence                                                                                                                                                                                    |
| ----------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| MACH-03     | ✓ SATISFIED | Hetzner default cloud profile, env-only credential discovery, explicit cloud setup flags, provisioning plan output, and provider adapter/tests enable one recommended low-cost cloud path.  |
| MACH-04     | ✓ SATISFIED | Cloud setup reuses BYO bootstrap and GitHub registration/online verification, then operations/status/logs/doctor work against the saved cloud state.                                        |
| MACH-05     | ✓ SATISFIED | State persists cloud provider kind/name/region, resource IDs, cloud inventory, runner ID, machine ref, cleanup IDs, managed paths, and provider facts for reconciliation/removal.           |
| CLEAN-01    | ✓ SATISFIED | BYO `down` and cloud `destroy --dry-run` both show plans before mutation; cloud plan includes billing warning and local-state removal condition.                                            |
| CLEAN-04    | ✓ SATISFIED | `runnerkit destroy` removes GitHub/remote/provider resources, verifies Hetzner resources absent or non-billable via `VerifyDestroyed`, and preserves partial cleanup checkpoints for retry. |
| DOC-02      | ✓ SATISFIED | `docs/cloud-quickstart.md` and README cloud section cover auth, setup, labels, operations, destroy verification, costs, limitations, and smoke-test guidance.                               |

**Coverage:** 6/6 requirements satisfied

## Automated Checks

- `go test ./...` — passed
- `go vet ./...` — passed
- `go run ./cmd/runnerkit --help` — passed
- Acceptance greps — passed for `Use: "destroy"`, `BuildCloudDestroyPlan`, `VerifyDestroyed`, `newDestroyCommand`, `RemoveRepository`, `provider_verification_pending`, cloud setup docs, and cloud destroy docs.
- BYO docs guard — passed: `docs/byo-quickstart.md` does not instruct BYO users to run `runnerkit destroy --repo owner/name`.

## Anti-Patterns Found

| File | Pattern | Severity | Impact                                                                                                                                               |
| ---- | ------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| None | -       | -        | No stubs, unimplemented cloud destroy methods, raw provider-token persistence, provider mutation in status/logs/doctor, or BYO `destroy` docs found. |

**Anti-patterns:** 0 found

## Human Verification

None required for phase acceptance. A controlled live Hetzner smoke remains recommended before public release because it creates billable resources and needs real credentials, but the Phase 4 verification contract is satisfied by automated adapter/CLI coverage and documented smoke guidance.

## Gaps

None.

## Result

**PASSED** — Phase 4 achieves the recommended cloud path and billable cleanup goal with 5/5 must-haves, 9/9 required artifacts, 9/9 key links, and 6/6 requirements verified.
