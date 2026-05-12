# 137 - Summary: Webhook spool rotation (size-based)

## Goal

Control disk usage of the webhook spool by rotating the JSONL spool file when it grows beyond a configured size.

## What changed

- `internal/gateway/runner.go`:
  - Adds rotation before appending and before replay:
    - `AGENT_GATEWAY_HOOK_SPOOL_MAX_BYTES` (default 5MB; set `0` to disable)
    - `AGENT_GATEWAY_HOOK_SPOOL_ROTATE_KEEP` (default 3; number of rotated files to keep)
  - Rotation renames:
    - `<spool>.YYYYMMDD_HHMMSS`
  - Best-effort deletes older rotated files beyond keep.

## Notes

- Rotation is a safety valve; it can drop events if spool grows too fast and the hook endpoint is consistently failing.

