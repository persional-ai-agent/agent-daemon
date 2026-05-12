# 123 - Summary: Gateway per-session queue + `/cancel` slash command

## Goal

Improve gateway UX toward Hermes behavior by:

- preventing overlapping agent runs in the same chat/session
- allowing users to interrupt a running task with a minimal slash command

## What changed

- `internal/gateway/runner.go` now uses a per-session worker queue:
  - each `(platform, chat_type, chat_id)` session has a buffered queue (size 32)
  - events are processed sequentially to avoid concurrent runs in the same chat
  - when queue is full, the oldest event is dropped (best-effort backpressure)
- Minimal slash commands (only in gateway mode):
  - `/cancel` or `/stop`: cancels the currently running agent context for that session (if any)
  - `/queue`: reports current queue length
  - `/help`: prints supported commands

## Notes / limitations

- This is a minimal alignment point; Hermes has richer command routing and queue policies.
- Cancellation is cooperative (context cancel); tool calls already respect context where possible.

