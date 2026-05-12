# 148 - Summary: `spool import -all` bulk import

## Goal

Support bulk import of multiple JSONL event files into target spool to simplify offline recovery workflows.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks spool import` adds `-all`.
  - `-all` scans candidate JSONL files (spool/hook-related naming) from input path context and imports in sorted order.
  - Keeps existing `event_id` dedup behavior when appending.

