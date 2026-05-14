# 0016 image summary merged

## 模块

- `image`

## 类型

- `summary`

## 合并来源

- `0024-image-summary-merged.md`

## 合并内容

### 来源：`0024-image-summary-merged.md`

# 0024 image summary merged

## 模块

- `image`

## 类型

- `summary`

## 合并来源

- `0108-image-generate-gate.md`
- `0115-image-generate-deliver-to-gateway.md`
- `0121-image-generate-openai-backend.md`

## 合并内容

### 来源：`0108-image-generate-gate.md`

# 109 Summary - image_generate 增加 FAL_KEY gate（Hermes 可用性对齐）

## 变更

- `image_generate`：当缺少 `FAL_KEY` 时返回 `available=false`（避免“看似可用但实际不是远端生成”的误导）。
- 仍保留 placeholder 输出实现（在 key 存在时返回本地生成的确定性 PNG），后续可按需接入真实 FAL backend。

## 修改文件

- `internal/tools/media_tools.go`

### 来源：`0115-image-generate-deliver-to-gateway.md`

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

### 来源：`0121-image-generate-openai-backend.md`

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
