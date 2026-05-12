# 119 - Summary: Optional real CDP backend for `browser_*` tools

## Goal

Upgrade the existing lightweight HTTP-only browser implementation to an optional “real browser” mode that can execute JS and interact with the live DOM, closer to Hermes `browser-cdp` toolset expectations.

## What changed

- Added a minimal Chrome DevTools Protocol client:
  - `internal/tools/browser_cdp_client.go`
  - Supports connecting via:
    - direct `ws://...` target URL
    - HTTP base like `http://127.0.0.1:9222` (resolves `/json/version` or `/json/new`)
- Added CDP-backed implementations for key browser tools:
  - `internal/tools/browser_cdp_impl.go`
  - `browser_navigate` / `browser_snapshot` / `browser_click` / `browser_type` / `browser_press`
  - `browser_cdp` returns live DOM metadata and optional HTML (`include_html=true`)
- Wired tool selection:
  - If `BROWSER_CDP_URL` is set (and `BROWSER_CDP_ENABLED` is not `false`), the standard `browser_*` tools use CDP.
  - Otherwise they continue to use the lightweight HTTP mode.

## Configuration

- Enable CDP mode:
  - `BROWSER_CDP_URL=ws://...` (recommended) or `BROWSER_CDP_URL=http://127.0.0.1:9222`
- Disable explicitly:
  - `BROWSER_CDP_ENABLED=false`

## Notes / limitations

- Console log collection and JS dialogs are still not implemented in CDP mode (tools return empty logs / nil dialog with a note).
- Ref IDs (`@e1`...) in CDP mode are derived from a JS snapshot list; they can become stale if the page changes.

