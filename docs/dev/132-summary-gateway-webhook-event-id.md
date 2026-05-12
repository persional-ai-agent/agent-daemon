# 132 - Summary: Gateway webhook `event_id` for deduplication

## Goal

Make webhook consumers able to deduplicate and trace events reliably by adding a stable event identifier to every webhook envelope, including spooled events.

## What changed

- `internal/gateway/runner.go`:
  - Webhook envelope now includes:
    - `id` (UUID)
    - `type`
    - `at` (RFC3339Nano)
    - `data`
  - Adds header: `X-Agent-Event-Id`
  - Spool entries store the event id and replay preserves it.

## Notes

- Receivers should use `id` (or `X-Agent-Event-Id`) as the primary dedup key.

