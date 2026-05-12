# 150 - Summary: `spool import` filter options

## Goal

Allow targeted import for spool recovery/migration instead of always importing all events.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks spool import` adds `-type` / `-id` / `-before`.
  - Import path now reuses existing spool filter semantics to select events by type, event id, and cutoff time.
  - Import result JSON adds `filter` payload for traceability.

