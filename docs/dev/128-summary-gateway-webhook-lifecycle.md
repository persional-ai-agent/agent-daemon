# 128 - Summary: Gateway webhook lifecycle events (started/failed + optional tool events)

## Goal

Extend the gateway webhook hooks beyond completion-only to provide minimal lifecycle observability, closer to Hermes “hooks/delivery” expectations.

## What changed

- `internal/gateway/runner.go`:
  - Always emits:
    - `gateway.started`
    - `gateway.completed`
    - `gateway.failed` (when run returns error)
  - Optional verbose tool events:
    - env: `AGENT_GATEWAY_HOOK_VERBOSE=true`
    - emits `gateway.tool_started` / `gateway.tool_finished` / `gateway.error`
  - Payloads are truncated to avoid oversized webhook bodies.

## Configuration

- `AGENT_GATEWAY_HOOK_URL` (required)
- `AGENT_GATEWAY_HOOK_TIMEOUT_SECONDS` (optional; default 4)
- `AGENT_GATEWAY_HOOK_VERBOSE=true` (optional)

