# 134 - Summary: CLI for webhook spool inspection/clear

## Goal

Improve operability of gateway webhook spooling by adding CLI commands to inspect and clear the spool without editing files manually.

## What changed

- `cmd/agentd/main.go`:
  - Adds:
    - `agentd gateway hooks spool status [-workdir dir] [-path file]` (JSON output with size/count/mtime)
    - `agentd gateway hooks spool clear  [-workdir dir] [-path file]`

## Notes

- Replay is handled by the running gateway process when `AGENT_GATEWAY_HOOK_SPOOL=true`; the CLI currently focuses on inspection/clear.

