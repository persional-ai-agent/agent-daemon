# 135 - Summary: CLI `spool replay` for gateway webhooks

## Goal

Provide an operational “manual replay” for webhook spools so operators can force a retry without waiting for the gateway background replay loop.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool replay`:
    - Reads the JSONL spool file
    - Attempts to POST up to `-limit` events to the hook URL
    - Rewrites the spool file with remaining failures
  - Flags:
    - `-url` override (defaults to `AGENT_GATEWAY_HOOK_URL`)
    - `-secret` override (defaults to `AGENT_GATEWAY_HOOK_SECRET`)
    - `-limit` (default 200)
    - `-timeout` per-request seconds (default 4)

## Notes

- This is best-effort and does not replace a durable queue.

