# 141 - Summary: Webhook spool `export` and `prune` commands

## Goal

Close the operational loop for webhook spool triage and cleanup by adding:

- targeted event export for incident analysis
- targeted event prune for controlled backlog cleanup

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool export`:
    - required: `-out <file>`
    - optional filters: `-type`, `-id`, `-before` (RFC3339/RFC3339Nano), `-all` (include rotated files)
  - Adds `agentd gateway hooks spool prune`:
    - optional filters: `-type`, `-id`, `-before`, `-all`
    - removes only matching entries; non-matching entries stay in spool

## Usage

- `agentd gateway hooks spool export -out /tmp/hook-export.jsonl -type gateway.delivery.media -all`
- `agentd gateway hooks spool prune -type gateway.delivery.media -before 2026-05-01T00:00:00Z -all`

