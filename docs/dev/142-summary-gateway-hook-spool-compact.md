# 142 - Summary: Webhook spool `compact` command

## Goal

Provide a maintenance command to reduce spool size/noise by deduplicating and trimming entries while preserving the newest useful events.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool compact`
  - Behavior:
    - drops malformed lines
    - deduplicates by `event_id` (`hookSpoolEntry.ID`)
    - sorts by `created_at` (oldest -> newest)
    - keeps only the newest `-max-lines` entries per file
  - Supports `-all` to compact rotated spool files too.

## Usage

- `agentd gateway hooks spool compact`
- `agentd gateway hooks spool compact -all -max-lines 500`

