# Host resources, memory, and OOM during CI

Self-hosted runners share the host’s RAM with every compiler, linker, and test process GitHub Actions starts. **RunnerKit does not run your workflows** — it helps you **see** when the machine is undersized or when logs look like an OOM / hard kill.

## Preflight and doctor signals

- **Low `MemAvailable`:** preflight warns when available memory is below the default threshold (4 GiB). Override with `RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES`. See [RKD-BOOT-016](bootstrap.md#rkd-boot-016).
- **No swap under 8 GiB RAM:** preflight warns that peak workloads may OOM without swap. See [RKD-BOOT-017](bootstrap.md#rkd-boot-017).
- **Journal heuristics:** when the runner is offline or the service failed, `runnerkit doctor` may scan bounded journals for OOM / SIGKILL patterns. See [RKD-BOOT-018](bootstrap.md#rkd-boot-018). Use `--deep` to collect hints even when the runner looks healthy; use `--with-log-snippets` to print short matching lines (redacted).

## Typical pattern: release build passes, tests kill the runner

`cargo build --release` often uses one optimized link. **`cargo test`** builds many **dev-profile** test binaries and may link **large native stacks** (for example database and UI crates) **in parallel**. Each link can use multiple gigabytes with debug info. The kernel may kill `ld` (signal 9) and then the **GitHub runner agent**, which shows up as the runner receiving a **shutdown signal** or going **offline**.

## Mitigations (workflow and host)

- Reduce parallel links: `CARGO_BUILD_JOBS=1` or `cargo test -j 1`.
- Use a lighter `[profile.test]` or run the heaviest suite in `--release` in a dedicated job.
- Split integration tests across jobs or machines.
- **More RAM** or **swap** on the runner host (swap trades latency for survival under spikes).

## Commands

```bash
runnerkit doctor --repo owner/repo
runnerkit doctor --repo owner/repo --deep --with-log-snippets
runnerkit logs --repo owner/repo --since 48h
```

For stable codes, see [bootstrap.md — RKD-BOOT-016..018](bootstrap.md#rkd-boot-016).
