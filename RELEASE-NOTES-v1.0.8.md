# RunnerKit v1.0.8 Release Notes

Date: 2026-05-11

## Highlights

- BYO/cloud cleanup reliability: service teardown order now uninstalls service metadata before runner deregistration (`config.sh remove`) to avoid partial cleanup checkpoints.
- Recovery parity: rereregister flows use the same run-as strategy as install/remove for scoped sudoers hosts.
- Structured diagnostics: CLI now emits JSON structured logs (optional) across command boundaries, remote SSH command execution, GitHub API calls, and Hetzner lifecycle operations.
- Debugging ergonomics: `remote_cleanup_pending` now carries bounded redacted detail to reduce ambiguity.

## Stopwatch / Live Smoke

- `make smoke-live` rerun after cleanup-order fixes: BYO and cloud paths completed cleanly.
- BYO duration (latest rerun): 126s
- Cloud duration (latest rerun): 190s

## Logging Controls

```bash
# Level: off|info|warn|error|debug
RUNNERKIT_LOG=debug

# Sink: stderr|stdout|file:/path/to/log.jsonl
RUNNERKIT_LOG_DEST=file:/tmp/runnerkit-debug.jsonl
```

## Notes

- See `docs/release-process.md` for the full tag-and-verify workflow.
