# 127 - Summary: Gateway webhook hooks (best-effort)

## Goal

Provide a minimal “hooks” mechanism for gateway observability / integrations (Hermes has hooks/delivery concepts). This lets external systems receive notifications when a gateway session completes a run.

## What changed

- `internal/gateway/runner.go`:
  - Adds optional webhook POST on completion:
    - env: `AGENT_GATEWAY_HOOK_URL`
    - env: `AGENT_GATEWAY_HOOK_TIMEOUT_SECONDS` (default 4)
  - Event emitted: `gateway.completed`
  - Payload is JSON:
    - `{ "type": "gateway.completed", "data": { ... } }`
    - `data` includes platform/chat/user/message/session_key/final/at

## Notes / limitations

- Best-effort fire-and-forget; hook failures do not affect chat delivery.
- Only completion events are emitted currently (can be extended to started/error/tool events later).

