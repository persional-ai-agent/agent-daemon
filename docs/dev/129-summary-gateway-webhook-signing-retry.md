# 129 - Summary: Gateway webhook signing + retries

## Goal

Harden gateway webhooks for production usage by adding:

- optional request signing (HMAC)
- best-effort retries with backoff

## What changed

- `internal/gateway/runner.go`:
  - Adds headers:
    - `X-Agent-Event`: event type
    - `X-Agent-Timestamp`: unix seconds
    - `X-Agent-Signature`: optional (when secret configured)
  - Optional signing:
    - env: `AGENT_GATEWAY_HOOK_SECRET`
    - signature: `hex(hmac_sha256(secret, ts + "." + body))`
  - Best-effort retries:
    - env: `AGENT_GATEWAY_HOOK_RETRIES` (default 2; total attempts = retries + 1)
    - env: `AGENT_GATEWAY_HOOK_BACKOFF_MS` (default 250; linear backoff per attempt)

## Notes

- Retries trigger on non-2xx responses or network errors.
- This remains fire-and-forget; hook failures do not affect gateway chat delivery.

