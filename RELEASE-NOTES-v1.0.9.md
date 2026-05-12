# RunnerKit v1.0.9 Release Notes

Date: 2026-05-12

## Highlights

- **Host capacity (Phase 7):** Preflight and `runnerkit doctor` surface RAM/swap headroom warnings (RKD-BOOT-016 / RKD-BOOT-017) with optional `RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES` override.
- **OOM / hard-kill hints:** When the runner is unhealthy or you pass **`doctor --deep`**, RunnerKit collects bounded journal excerpts and emits structured **`host_incident_hints`** (RKD-BOOT-018) with conservative “likely” wording.
- **Troubleshooting:** New **`docs/troubleshooting/host-resources.md`** (indexed from bootstrap / GitHub troubleshooting pages).
- **Live smoke:** BYO and Hetzner smokes run **`scripts/smoke/assert-doctor-json-contract.sh`** after interactive `doctor` to lock in the **`doctor --json`** / **`doctor --deep --json`** contract (requires **`python3`**).
- **JSON contract fix:** **`doctor --json`** always encodes **`host_incident_hints`** and **`next_actions`** as JSON **arrays** (including empty `[]`), never **`null`**, for stable machine parsing.

## Stopwatch / Live Smoke

- `make smoke-live` (maintainer run, 2026-05-12): BYO + cloud + destroy-verify completed cleanly; doctor JSON assert baseline + deep on both legs.
- BYO duration: **152s**
- Cloud duration: **291s**

## Notes

- See `docs/release-process.md` for the full tag-and-verify workflow.
- Skip deep journal collection in smoke only if needed: **`RUNNERKIT_SMOKE_SKIP_DOCTOR_DEEP=1`**.
