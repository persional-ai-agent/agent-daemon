# 125 - Summary: Gateway pairing management (`/unpair` + CLI revoke/list)

## Goal

Make the minimal pairing flow operable:

- allow users to unpair themselves from chat
- allow operators to list and revoke pairings via CLI without editing JSON manually

## What changed

- `internal/gateway/runner.go`:
  - Adds `/unpair` slash command to remove the current user from the pairing store.
  - Pairing data remains in `<workdir>/.agent-daemon/gateway_pairs.json`.
- `cmd/agentd/main.go`:
  - Adds `agentd gateway pairs list` to show current pairings.
  - Adds `agentd gateway pairs revoke -platform <p> -user <id>` to revoke a specific user id.

## Usage

- In chat:
  - `/pair <code>`
  - `/unpair`
- CLI:
  - `agentd gateway pairs list`
  - `agentd gateway pairs revoke -platform telegram -user <id>`

