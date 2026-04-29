# GEMINI.md

<!-- GSD:project-start source:PROJECT.md -->

## Project

RunnerKit is a CLI-first tool for solo developers to create and manage self-hosted GitHub Actions runners without manual runner setup headaches. The v1 goal is a reliable, cost-effective runner online and usable in about 10 minutes.

Key project docs:

- `.planning/PROJECT.md` - product context and decisions
- `.planning/REQUIREMENTS.md` - v1 requirements and traceability
- `.planning/ROADMAP.md` - phased execution roadmap
- `.planning/STATE.md` - current GSD state
<!-- GSD:project-end -->

<!-- GSD:stack-start source:STACK.md -->

## Technology Stack

Planned stack from research:

- Go CLI, likely with Cobra
- GitHub REST API / `google/go-github` for runner management
- SSH + Linux/systemd for BYO runner bootstrap
- One low-cost cloud provider path, with Hetzner as the leading candidate pending implementation validation
- Local versioned state/config with strict secret redaction

See `.planning/research/STACK.md` and `.planning/research/SUMMARY.md` before implementation.

<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

Conventions not yet established. Establish patterns during Phase 1 and document them as they emerge.

<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

Planned architecture is a local-first CLI orchestrator with isolated adapters for GitHub, machine providers, remote execution, state/config, secrets/redaction, and health reconciliation.

See `.planning/research/ARCHITECTURE.md` for the proposed component boundaries and data flows.

<!-- GSD:architecture-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:

- `/gsd-discuss-phase 1` to gather implementation context for Phase 1
- `/gsd-plan-phase 1` to create Phase 1 execution plans
- `/gsd-execute-phase 1` for planned phase work after planning
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.

<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by GSD profile tooling - do not edit manually.

<!-- GSD:profile-end -->
