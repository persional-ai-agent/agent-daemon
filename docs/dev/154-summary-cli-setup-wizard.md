# 154 - Summary: `setup wizard` interactive bootstrap

## Goal

Close the remaining human-facing bootstrap gap by adding an interactive terminal setup flow on top of the existing non-interactive `setup` command.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd setup wizard`.
  - Prompts for provider/model/base URL/API key/fallback provider and optional gateway platform credentials.
  - Reuses the same config application path as non-interactive `setup` and `gateway setup`.
- Documentation:
  - Updates parity docs to reflect that the remaining CLI gap is no longer “missing setup wizard”, but broader update/bootstrap completeness.
