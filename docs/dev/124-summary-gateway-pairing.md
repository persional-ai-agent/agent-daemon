# 124 - Summary: Minimal gateway pairing via `/pair`

## Goal

Add a minimal “pairing” flow for gateway access so operators can bootstrap authorization without editing config files:

- users can send `/pair <code>` in chat to gain access
- paired user IDs persist across restarts (best-effort) in a local file

## What changed

- `internal/gateway/runner.go`:
  - Adds pairing store:
    - env: `AGENT_GATEWAY_PAIR_CODE` (shared pairing code)
    - file: `<workdir>/.agent-daemon/gateway_pairs.json`
  - Authorization logic:
    - normal allowlist (`gateway.<platform>.allowed_users`) still applies
    - if user is paired for that platform, access is granted even when allowlist is empty
  - Slash command:
    - `/pair <code>`: pairs the current user ID for the current platform when code matches
    - `/help` updated to include `/pair`

## Notes / limitations

- Pairing is platform-scoped (Telegram/Discord/Slack/Yuanbao user IDs are stored separately).
- This is a minimal alignment point; Hermes supports richer pairing/management flows.

