# 122 - Summary: `image_generate` optional OpenAI backend

## Goal

Close the placeholder gap for `image_generate` by enabling a best-effort real image generation backend when `OPENAI_API_KEY` is configured, while preserving the existing fallback behavior and `deliver` support.

## What changed

- `internal/tools/builtin.go`:
  - `image_generate` schema adds optional `size` and `model`.
- `internal/tools/media_tools.go`:
  - Tool is considered configured when either `OPENAI_API_KEY` or `FAL_KEY` is present.
  - If `OPENAI_API_KEY` is set, calls `POST /images/generations` with `response_format=b64_json`, writes bytes to `output_path`, and returns `media: "MEDIA: <path>"`.
  - Env knobs:
    - `OPENAI_BASE_URL` (default `https://api.openai.com/v1`)
    - `OPENAI_IMAGE_MODEL` (default `gpt-image-1`)
  - If the OpenAI call fails (network/HTTP/shape), falls back to deterministic placeholder PNG.

## Notes

- Network access is required for OpenAI generation; in restricted environments the tool will fall back to placeholder output.
- `deliver=true` continues to use the gateway `MediaSender` implementations (Telegram/Discord/Slack/Yuanbao).

