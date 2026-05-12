# 156 - Summary: `version` command

## Goal

Close the remaining basic CLI maintenance gap by exposing a dedicated version command that can also report update status on git checkouts.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd version`.
  - Reports app version, release date, build commit, and Go runtime version.
  - Supports `-check-update` to reuse the minimal git-based update status check and `-json` for automation.
- Documentation:
  - Updates CLI parity docs so `version` and `update` are treated as a paired maintenance surface.
