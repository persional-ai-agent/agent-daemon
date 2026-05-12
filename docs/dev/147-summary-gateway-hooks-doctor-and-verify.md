# 147 - Summary: Gateway hooks `doctor` + spool `verify`

## Goal

Provide first-class diagnostics for webhook operations:

- `hooks doctor` for config/env sanity checks
- `spool verify` for spool file integrity checks

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks doctor` reports resolved settings, status (`ok/warn/error`), and issue list.
  - `agentd gateway hooks spool verify` checks line-level validity and reports invalid samples, with `-all` support.

