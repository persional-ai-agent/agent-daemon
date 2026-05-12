# 149 - Summary: `hooks doctor` next actions

## Goal

Make diagnostics actionable by returning concrete remediation suggestions alongside detected issues.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks doctor` now outputs `next_actions` with suggested commands/config fixes for common warnings/errors.

