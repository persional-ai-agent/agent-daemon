# 144 - Summary: Webhook spool `verify` command

## Goal

Add a quick integrity check for spool files so operators can detect malformed or incomplete lines before replay/export/prune operations.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool verify`
  - Supports `-all` to verify rotated files too.
  - Reports:
    - total/valid/invalid line counts
    - per-file invalid samples (up to 5)

