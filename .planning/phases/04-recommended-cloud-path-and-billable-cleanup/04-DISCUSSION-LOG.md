# Phase 4: Recommended Cloud Path and Billable Cleanup - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md - this log preserves the alternatives considered.

**Date:** 2026-04-30
**Phase:** 4-Recommended Cloud Path and Billable Cleanup
**Areas discussed:** Recommended provider/profile, Cloud setup UX, Cloud quickstart expectations, Cloud resource identity and readiness, Billable destroy semantics

---

## Initial gray-area selection

| Option                                | Description                                                                                                                                                                                                 | Selected |
| ------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Recommended provider/profile          | Which one low-cost cloud path should be the default, what size/region posture should it have, and how much cost estimate detail is needed? Research points to Hetzner; no provider adapter exists yet.      | ✓        |
| Cloud setup UX                        | How should users choose cloud vs BYO in `runnerkit up`, what flags/prompts should exist, and what should the provisioning plan show? Carries forward existing wizard + plan-before-mutation pattern.        | Later    |
| Cloud resource identity and readiness | What provider IDs, tags, SSH key/firewall/network facts, and readiness checks must be saved or shown so status/doctor can reconcile later? State already has ProviderRef and cleanup provider resource IDs. | Later    |
| Billable destroy semantics            | What command/confirmation/verification behavior should remove cloud resources without surprise bills? Phase 3 reserved `destroy` for cloud and built safe BYO cleanup patterns.                             | Later    |

**User's choice:** Recommended provider/profile
**Notes:** User then repeatedly chose to explore more gray areas until all surfaced areas were covered.

---

## Recommended provider/profile

### Provider choice

| Option                    | Description                                                                                                                                                           | Selected |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Hetzner Cloud             | Recommended: strongest low-cost VPS story for solo developers, simple VM model, and current research already points here; validate regions/quota/API during planning. |          |
| DigitalOcean              | More familiar and globally available for many developers, but typically less cost-aggressive than Hetzner.                                                            |          |
| You decide after research | Let downstream research/planning lock the provider based on cost, quota friction, SSH readiness, and API simplicity.                                                  | ✓        |

**User's choice:** You decide after research
**Notes:** Provider is intentionally left to downstream research/planning rather than locked during discussion.

### Default profile optimization

| Option                       | Description                                                                                                                  | Selected |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | -------- |
| Lowest credible monthly cost | Recommended: pick the cheapest VM/profile that can reliably run typical solo CI jobs, and show clear caveats if it is small. |          |
| Smoothest setup/reliability  | Prefer the provider/image/region/size with fewer quota, SSH, and networking surprises even if it costs a bit more.           | ✓        |
| Balanced default             | Pick a modest baseline that is cheap but not tiny; avoid both bare-minimum instances and overpowered defaults.               |          |

**User's choice:** Smoothest setup/reliability
**Notes:** Cost still matters, but reliability and first-run smoothness outweigh absolute lowest price.

### Cost display

| Option                                      | Description                                                                                                                                | Selected |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| Show hourly + monthly estimate with caveats | Recommended: give a visible approximate hourly/monthly cost, note region/provider pricing may change, and remind how to destroy resources. | ✓        |
| Show simple low-cost label only             | Keep the wizard concise; mention it is the recommended low-cost path without detailed numbers.                                             |          |
| Require explicit cost acknowledgment        | Make the user confirm the displayed estimate before creating billable resources; safer but adds friction.                                  |          |

**User's choice:** Show hourly + monthly estimate with caveats
**Notes:** Estimates should be visible before provisioning, but not overpromise exact billing.

### Credential handling

| Option                            | Description                                                                                                                                   | Selected |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Reuse provider CLI/env auth first | Recommended: detect existing provider CLI credentials or environment variables, never store tokens, and print exact setup steps when missing. | ✓        |
| Guided token prompt               | Ask the user for an API token in the wizard; still store only a reference, but this is more sensitive and harder to keep out of logs.         |          |
| Config-file reference             | Require users to configure a credential reference before running cloud provisioning; safer but less one-command.                              |          |

**User's choice:** Reuse provider CLI/env auth first
**Notes:** RunnerKit should not become a cloud secret vault.

---

## Cloud setup UX

### Entry point

| Option                                       | Description                                                                                                              | Selected |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | -------- |
| `runnerkit up` asks when no host is provided | Recommended: preserve one primary setup command; if no `--host` is provided, the wizard offers BYO vs recommended cloud. | ✓        |
| Explicit `runnerkit up --cloud`              | Require an explicit cloud flag so no one accidentally provisions billable resources from a missing host argument.        |          |
| Separate `runnerkit cloud up` command        | Clear separation, but weaker continuity with the existing BYO lifecycle and more command surface.                        |          |

**User's choice:** `runnerkit up` asks when no host is provided
**Notes:** The interactive path remains wizard-first and one-command-oriented.

### Automation/non-interactive setup

| Option                                      | Description                                                                                                                                             | Selected |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Require explicit cloud flags for automation | Recommended: interactive mode may ask, but non-interactive mode must pass flags like `--cloud`/provider/profile/region and `--yes` before provisioning. | ✓        |
| Allow defaults with only `--yes`            | Fastest path, but a missing `--host` plus `--yes` could create billable infrastructure too easily.                                                      |          |
| No non-interactive cloud provisioning yet   | Keep Phase 4 human-guided only; simpler but less useful for scripted setup and tests.                                                                   |          |

**User's choice:** Require explicit cloud flags for automation
**Notes:** Avoid accidental billable infrastructure from a missing host argument.

### Provisioning plan content

| Option                                        | Description                                                                                                                                               | Selected |
| --------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Resources + cost + identity + destroy command | Recommended: VM/profile/region, SSH key/firewall/network, tags/names, estimated cost, labels/snippet preview, and exact `runnerkit destroy` cleanup path. | ✓        |
| Short summary only                            | Keep it concise: provider/profile/region/cost and confirmation; leave details to verbose/JSON output.                                                     |          |
| Full step-by-step dry run                     | Show every planned provider/GitHub/SSH/bootstrap action before confirmation; safest but potentially noisy.                                                |          |

**User's choice:** Resources + cost + identity + destroy command
**Notes:** The setup plan should teach cleanup before paid resources are created.

### Prerequisite failure behavior

| Option                                             | Description                                                                                                                                          | Selected |
| -------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Fail before any mutation with exact fix steps      | Recommended: no partial GitHub/SSH/bootstrap work until provider prerequisites pass; show the provider setup command/docs and ask the user to rerun. | ✓        |
| Offer fallback to BYO in the same flow             | Helpful if the user has a machine after all, but can make the flow branchy.                                                                          |          |
| Keep retrying/ask for another region interactively | More adaptive, but may turn setup into provider troubleshooting rather than a fast RunnerKit path.                                                   |          |

**User's choice:** Fail before any mutation with exact fix steps
**Notes:** Credential/quota/region readiness should be checked before mutations.

---

## Cloud quickstart expectations

### Documentation priority

| Option                             | Description                                                                                                                      | Selected |
| ---------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Fast first successful cloud runner | Recommended: concise steps from auth → `runnerkit up` → workflow labels → `runnerkit destroy`, with details linked or collapsed. | ✓        |
| Safety-first walkthrough           | Lead with billable-resource warnings, credentials, cleanup, and troubleshooting before the happy path.                           |          |
| Reference-style completeness       | Document flags, provider concepts, state fields, and failure modes in one comprehensive guide.                                   |          |

**User's choice:** Fast first successful cloud runner
**Notes:** Concise happy path first; deeper details should not obscure first success.

### Mandatory quickstart contents

| Option                                  | Description                                                                                                                                                 | Selected |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Auth, cost, up, labels, status, destroy | Recommended: provider auth setup, estimated cost explanation, setup command, `runs-on` snippet, status/logs/doctor reuse, and cleanup/destroy verification. | ✓        |
| Only setup and labels                   | Shortest possible guide, but cleanup may be too easy to miss for billable resources.                                                                        |          |
| Full troubleshooting matrix too         | Include common provider credential/quota/SSH/firewall failures in the quickstart itself.                                                                    |          |

**User's choice:** Auth, cost, up, labels, status, destroy
**Notes:** Cleanup/destroy is mandatory in quickstart because Phase 4 introduces paid resources.

### Cost caveats in docs

| Option                                   | Description                                                                                                                   | Selected |
| ---------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | -------- |
| Clearly approximate and user-responsible | Recommended: state estimates are approximate, pricing varies by region/time, and the user must run `destroy` to stop billing. | ✓        |
| Minimal caveat                           | Keep docs upbeat; mention costs are provider-dependent but avoid heavy warnings.                                              |          |
| Strong warning before setup              | Emphasize billable resources and require users to read cleanup guidance before provisioning.                                  |          |

**User's choice:** Clearly approximate and user-responsible
**Notes:** The docs should be honest without making setup feel hostile.

### Limitations to call out

| Option                                               | Description                                                                                                                                                       | Selected |
| ---------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Single provider + persistent trusted/private default | Recommended: say Phase 4 supports one provider path, persistent mode is for trusted/private repos, ephemeral mode comes later, and workflows are not auto-edited. | ✓        |
| Provider limitation only                             | Just note that v1 supports one recommended provider path, with more providers deferred.                                                                           |          |
| Keep limitations out of quickstart                   | Keep the guide streamlined and put constraints in separate docs.                                                                                                  |          |

**User's choice:** Single provider + persistent trusted/private default
**Notes:** The quickstart should keep scope expectations clear.

---

## Cloud resource identity and readiness

### State inventory

| Option                              | Description                                                                                                                                       | Selected |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Full managed resource inventory     | Recommended: save VM ID/name, region, image, size/profile, SSH key ID/name, firewall/network IDs, public IP/host, tags, and cleanup resource IDs. | ✓        |
| Minimum VM identity only            | Save just provider kind, VM ID, region, and public host; simpler but weaker for destroy/orphan checks.                                            |          |
| Provider decides per implementation | Let each provider adapter decide which IDs matter; less product consistency but flexible.                                                         |          |

**User's choice:** Full managed resource inventory
**Notes:** State should support later status/doctor/destroy reconciliation.

### Resource naming/tagging

| Option                                       | Description                                                                                                                                     | Selected |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Predictable RunnerKit tags on every resource | Recommended: include `runnerkit`, repo slug, runner name/state ID, mode, created-at, and managed=true so destroy/status can identify ownership. | ✓        |
| Human-readable names only                    | Use obvious resource names but avoid a heavier tag schema.                                                                                      |          |
| Provider defaults                            | Let provider-generated names/tags stand; least work but weakest cleanup confidence.                                                             |          |

**User's choice:** Predictable RunnerKit tags on every resource
**Notes:** Ownership metadata should apply to every created resource.

### Readiness gate

| Option                                                 | Description                                                                                                                                                                        | Selected |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Provider running + SSH verified + BYO preflight passes | Recommended: wait for provider running state, SSH reachability/host key, cloud-init or boot readiness when available, and then reuse BYO preflight before any runner registration. | ✓        |
| SSH reachable is enough                                | Faster, but cloud-init/package/network readiness failures may surface later in bootstrap.                                                                                          |          |
| Provider running is enough                             | Simplest, but risks brittle sleep/retry behavior and poor setup errors.                                                                                                            |          |

**User's choice:** Provider running + SSH verified + BYO preflight passes
**Notes:** Do not request/use GitHub registration token before the readiness gate.

### Provider facts after setup

| Option                                          | Description                                                                                                                                                    | Selected |
| ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Include provider facts in status/doctor/destroy | Recommended: show provider instance state, region/profile, public IP, billable-resource summary, and drift/orphan warnings alongside GitHub/SSH/service facts. | ✓        |
| Only show provider facts in destroy             | Keep daily status focused on runner health; show provider details only when cleaning up.                                                                       |          |
| JSON only for provider facts                    | Human output stays simple; automation can inspect provider details in JSON.                                                                                    |          |

**User's choice:** Include provider facts in status/doctor/destroy
**Notes:** Cloud runners add a provider-side source of truth to reconciliation.

---

## Billable destroy semantics

### Command name

| Option                                  | Description                                                                                                                       | Selected |
| --------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | -------- |
| `runnerkit destroy` for cloud resources | Recommended: matches Phase 3 decision to reserve destroy language for billable cloud resources, while `down` remains BYO cleanup. | ✓        |
| `runnerkit down --destroy-cloud`        | Keeps one cleanup command but makes a dangerous billable action a flag.                                                           |          |
| `runnerkit cloud destroy`               | Clear cloud namespace, but adds more command taxonomy than the current top-level lifecycle.                                       |          |

**User's choice:** `runnerkit destroy` for cloud resources
**Notes:** Preserves the prior `down` vs `destroy` distinction.

### Confirmation style

| Option                                                        | Description                                                                                                                                              | Selected |
| ------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Plan first, then explicit confirmation for billable resources | Recommended: show GitHub/remote/provider/local-state artifacts, estimated billing impact, and require confirmation; `--yes` applies a safe default plan. | ✓        |
| Artifact-by-artifact prompts only                             | Maximum manual control, but can be tedious and awkward for automation.                                                                                   |          |
| One confirmation for everything                               | Fastest, but weaker for confidence when several billable resources are involved.                                                                         |          |

**User's choice:** Plan first, then explicit confirmation for billable resources
**Notes:** `destroy --yes` should exist for automation but apply a safe default plan.

### Success verification

| Option                                               | Description                                                                                                                                                                    | Selected |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| GitHub gone + provider resources non-billable/absent | Recommended: verify runner deregistration/removal plus provider VM/firewall/SSH key/network resources are deleted or demonstrably non-billable; record pending cleanup if not. | ✓        |
| Provider deletion API returned success               | Simpler and faster, but may miss delayed deletions, attached disks, or orphaned billable resources.                                                                            |          |
| Best-effort with warning                             | Do what it can and warn users to check the provider console manually.                                                                                                          |          |

**User's choice:** GitHub gone + provider resources non-billable/absent
**Notes:** A provider delete API response alone is insufficient for success.

### Partial failure state behavior

| Option                                      | Description                                                                                                                                | Selected |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| Keep state with pending cleanup checkpoints | Recommended: preserve enough provider/GitHub/remote identity to resume destroy, show pending billable resources, and do not hide failures. | ✓        |
| Remove state after provider delete attempt  | Cleaner local state, but risks losing the IDs needed to clean up remaining resources.                                                      |          |
| Ask user whether to keep or remove state    | Flexible, but adds one more decision during an already stressful cleanup flow.                                                             |          |

**User's choice:** Keep state with pending cleanup checkpoints
**Notes:** State should not be removed while billable resources may remain.

---

## the agent's Discretion

- Exact provider and default profile after targeted Phase 4 research.
- Exact flag names, JSON field names, provider adapter API shape, tag key names, and cost-estimation source.
- Exact output formatting for provisioning plans, provider facts, and destroy verification.

## Deferred Ideas

- No new deferred ideas were introduced. Existing roadmap deferrals remain: additional cloud providers later, ephemeral mode in Phase 5, and richer cost controls/idle shutdown/orphan detection beyond Phase 4's required destroy verification.
