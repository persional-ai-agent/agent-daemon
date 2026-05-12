# 116 - Summary: `image_generate` optional gateway delivery (`deliver=true`)

## Goal

Make `image_generate` behave more like Hermes “artifact → chat” flows by allowing direct delivery of the generated image to the current gateway chat when available.

## What changed

- `image_generate` schema adds:
  - `deliver` (boolean)
  - `caption` (string; used when delivering)
- Tool output now includes `media: "MEDIA: <path>"` and, when `deliver=true`, the same `delivered` / `delivery_*` fields as `text_to_speech` (best-effort).

## Notes

- Actual image generation backend is still not wired in `agent-daemon` (gated by `FAL_KEY`); delivery works for the produced local file.

