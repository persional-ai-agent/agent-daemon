# 145 - Summary: Gateway hooks `doctor` command

## Goal

Provide a single diagnostic command to check webhook-related configuration consistency and common risk conditions.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks doctor`
  - Reports:
    - resolved hook/spool settings
    - status (`ok`/`warn`/`error`)
    - issue list (invalid numeric settings, missing URL, oversized spool warning, etc.)

