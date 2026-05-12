# 139 - Summary: Aggregated spool status (`status -all`)

## Goal

Improve webhook spool observability by making `status` report backlog health across rotated files, not just the base spool file.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks spool status` adds `-all`.
  - `-all` aggregates over rotated files + base file and returns:
    - per-file stats (`exists`, `size_bytes`, `count`, `types`, `oldest_at`)
    - global totals (`total_count`, `total_size_bytes`, `types`, `oldest_at`, `oldest_age_seconds`)

## Usage

- `agentd gateway hooks spool status`
- `agentd gateway hooks spool status -all`

