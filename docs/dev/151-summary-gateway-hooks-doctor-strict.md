# 151 - Summary: `hooks doctor --strict`

## Goal

Make hook diagnostics composable in CI/script flows by returning non-zero exit code when health is not `ok`.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks doctor` adds `-strict`.
  - When enabled, command exits with code `1` if computed status is not `ok`, while keeping JSON diagnosis output.

