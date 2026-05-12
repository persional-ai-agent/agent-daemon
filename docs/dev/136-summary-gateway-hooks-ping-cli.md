# 136 - Summary: CLI `gateway hooks ping` healthcheck

## Goal

Improve gateway webhook operability by providing a simple healthcheck command to validate:

- hook URL connectivity (HTTP POST)
- optional HMAC signing header generation (when secret configured)

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks ping`:
    - sends a `gateway.ping` event envelope to the configured hook URL
    - includes `X-Agent-Event`, `X-Agent-Event-Id`, `X-Agent-Timestamp`, and optional `X-Agent-Signature`
    - returns JSON with `success` and HTTP status code

## Usage

- `agentd gateway hooks ping` (uses env `AGENT_GATEWAY_HOOK_URL`)
- `agentd gateway hooks ping -url http://localhost:9000/hook -secret ...`

