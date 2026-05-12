# 131 - Summary: Gateway webhook spool (dead-letter + replay)

## Goal

Reduce webhook event loss on transient failures or restarts by adding an optional local spool:

- when hook delivery fails after retries, append the event to a JSONL spool file
- a background loop periodically retries sending spooled events and rewrites the file with remaining failures

## What changed

- `internal/gateway/runner.go`:
  - Env: `AGENT_GATEWAY_HOOK_SPOOL=true` enables spooling and replay loop.
  - Env: `AGENT_GATEWAY_HOOK_SPOOL_PATH` overrides spool path (default `<workdir>/.agent-daemon/gateway_hooks_spool.jsonl`)
  - Env: `AGENT_GATEWAY_HOOK_SPOOL_REPLAY_SECONDS` (default 10)
  - Env: `AGENT_GATEWAY_HOOK_SPOOL_MAX_LINES` (default 2000)

## Notes / limitations

- Best-effort: spool is local file, not an at-least-once durable queue with dedup/ordering guarantees.
- Replay is bounded per tick (caps work to avoid long blocks).

