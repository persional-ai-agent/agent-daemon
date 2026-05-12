# 126 - Summary: Gateway slow-response hint message

## Goal

Align Hermes gateway UX by sending a one-time “still working” message when the agent hasn’t produced visible output for a while (useful for long tool runs or slow models).

## What changed

- `internal/gateway/runner.go`:
  - Adds a per-run slow-response notifier:
    - if no stream edits/messages were emitted for `N` seconds, sends a waiting message once and stops.
  - Env knobs:
    - `AGENT_GATEWAY_SLOW_RESPONSE_TIMEOUT_SECONDS` (default `120`; set `0` to disable)
    - `AGENT_GATEWAY_SLOW_RESPONSE_MESSAGE` (default: `任务有点复杂，正在努力处理中，请耐心等待...`)

## Notes

- This is best-effort and does not replace proper progress events; it only improves user feedback during quiet periods.

