# 146 - Summary: Webhook spool `import` command

## Goal

Add a safe import workflow for webhook spool events so offline-exported JSONL events can be merged back for replay.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool import -in <file>`
  - Supports:
    - `-append=true|false` (default true)
    - `-path` to set target spool file
  - Deduplicates by `event_id` (`hookSpoolEntry.ID`) when appending.
  - Validates minimal shape (`type` and `body`) and skips malformed lines.

## Usage

- `agentd gateway hooks spool import -in /tmp/events.jsonl`
- `agentd gateway hooks spool import -in /tmp/events.jsonl -append=false`

