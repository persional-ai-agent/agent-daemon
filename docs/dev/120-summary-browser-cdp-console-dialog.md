# 120 - Summary: CDP browser console + dialog support

## Goal

Bring CDP mode closer to Hermes expectations for:

- `browser_console`: return console output / JS errors
- `browser_dialog`: accept/dismiss native JS dialogs and surface `pending_dialogs` via `browser_snapshot`

## What changed

- `internal/tools/browser_cdp_client.go`:
  - Parses CDP event envelopes (`method`/`params`) and forwards to an event callback.
- `internal/tools/browser_cdp_impl.go`:
  - Buffers logs from:
    - `Runtime.consoleAPICalled`
    - `Runtime.exceptionThrown`
    - `Log.entryAdded`
  - Tracks dialogs from:
    - `Page.javascriptDialogOpening` / `Page.javascriptDialogClosed`
  - `browser_snapshot` now populates `pending_dialogs` when a dialog is open.
  - `browser_console` returns and clears buffered logs.
  - `browser_dialog(action=accept|dismiss, prompt_text=...)` calls `Page.handleJavaScriptDialog`.
- `internal/tools/builtin.go`:
  - Updates `browser_dialog` schema to accept `action` and `prompt_text` (Hermes-compatible).

## Notes

- CDP mode must be enabled (`BROWSER_CDP_URL` set) for these behaviors; otherwise tools remain lightweight.
- Logs are buffered per session and capped (200 entries).

