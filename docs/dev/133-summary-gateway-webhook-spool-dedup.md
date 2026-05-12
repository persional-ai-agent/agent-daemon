# 133 - Summary: Webhook spool dedup by `event_id`

## Goal

Reduce spool growth and avoid re-sending duplicate webhook events by deduplicating using the webhook `event_id`.

## What changed

- `internal/gateway/runner.go`:
  - When spooling is enabled, keeps an in-memory “seen event_id” set (loaded from the spool file on startup).
  - Skips appending to spool if `event_id` has already been seen (default on).
  - During replay, also drops duplicate `event_id` entries while rewriting the spool file.

## Configuration

- `AGENT_GATEWAY_HOOK_SPOOL_DEDUP` (default `true`; set to `false` to disable)
- `AGENT_GATEWAY_HOOK_SPOOL_DEDUP_MAX` (default `5000`; max in-memory ids)

