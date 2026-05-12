# 153 - Summary: `setup` unified bootstrap entrypoint

## Goal

Close the remaining CLI bootstrap gap by providing one non-interactive command that writes minimal provider and optional gateway config.

## What changed

- `cmd/agentd/main.go`:
  - Adds top-level `agentd setup`.
  - Supports writing provider/model/base-url/api-key/fallback-provider config in one step.
  - Optionally chains a gateway platform setup for `telegram` / `discord` / `slack` / `yuanbao`.
  - Reuses the same gateway config writer as `agentd gateway setup` to avoid config drift.
- Documentation:
  - Reframes the remaining gap from “no setup” to “no interactive setup wizard/update flow”.
