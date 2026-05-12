# 143 - Summary: Webhook spool `stats` command

## Goal

Add an explicit stats command for spool backlog inspection, complementary to `status`.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool stats`
  - Supports `-all` for aggregated stats across rotated files + base file.
  - Returns:
    - per-file count/size/type distribution/oldest timestamp
    - global totals and oldest event age

## Usage

- `agentd gateway hooks spool stats`
- `agentd gateway hooks spool stats -all`

