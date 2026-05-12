# 118 - Summary: Gateway auto-delivery for final `MEDIA:` responses

## Goal

Align Hermes “artifact-first” UX: when the agent’s final response is a media artifact pointer (`MEDIA: <path>`), the gateway should deliver the file directly to the chat without requiring an explicit `send_message(media_path=...)` call.

## What changed

- `internal/gateway/runner.go` adds best-effort auto delivery:
  - If the final assistant content begins with `MEDIA:`, the runner extracts the path and tries to send it via the current adapter’s `platform.MediaSender`.
  - Safety checks:
    - Path must resolve under `engine.Workdir` or `/tmp`
    - File must exist and be a regular file
  - If delivery succeeds, the runner stops and does not send the literal `MEDIA:` text.
  - If delivery fails (no MediaSender / validation failed / send error), the runner falls back to sending the original final text.

## Notes

- This only triggers for the *final assistant content* (not streaming partials).
- Caption is currently empty; if a tool wants a caption it can still use `send_message(media_path=..., message=...)`.

