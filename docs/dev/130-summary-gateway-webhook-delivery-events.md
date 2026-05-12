# 130 - Summary: Gateway webhook delivery events (send/edit/media)

## Goal

Extend gateway hooks to cover “delivery” events so external systems can observe what was actually sent/edited, including media delivery.

## What changed

- `internal/gateway/runner.go`:
  - Adds delivery-level events when `AGENT_GATEWAY_HOOK_DELIVERY=true`:
    - `gateway.delivery.send` (text send)
    - `gateway.delivery.edit` (message edit)
    - `gateway.delivery.media` (media send via `platform.MediaSender`)
  - Delivery events include platform/chat/message id, success/error, and small metadata tags (`phase`, `turn`, `slash`).

## Configuration

- Requires `AGENT_GATEWAY_HOOK_URL`
- Enable delivery events: `AGENT_GATEWAY_HOOK_DELIVERY=true`

