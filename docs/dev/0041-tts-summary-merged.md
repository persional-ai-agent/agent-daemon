# 0041 tts summary merged

## 模块

- `tts`

## 类型

- `summary`

## 合并来源

- `0051-tts-summary-merged.md`

## 合并内容

### 来源：`0051-tts-summary-merged.md`

# 0051 tts summary merged

## 模块

- `tts`

## 类型

- `summary`

## 合并来源

- `0112-tts-openai-backend-media-prefix.md`
- `0114-tts-deliver-to-gateway.md`

## 合并内容

### 来源：`0112-tts-openai-backend-media-prefix.md`

# 113 Summary - text_to_speech 支持 OpenAI 后端（可选）+ MEDIA: 前缀对齐

## 变更

- `text_to_speech`：
  - 当存在 `OPENAI_API_KEY` 时，best-effort 调用 `OPENAI_BASE_URL` 的 `POST /audio/speech` 生成音频文件（默认 `mp3`）。
  - 否则回退为本地 placeholder beep WAV。
  - 返回中增加 `media: "MEDIA: <path>"` 字段（Hermes 风格，便于后续 gateway 交付层识别）。

## 环境变量

- `OPENAI_API_KEY`（启用 OpenAI TTS）
- `OPENAI_BASE_URL`（默认 `https://api.openai.com/v1`）
- `OPENAI_TTS_MODEL`（默认 `gpt-4o-mini-tts`）
- `OPENAI_TTS_VOICE`（默认 `alloy`）

## 修改文件

- `internal/tools/media_tools.go`

### 来源：`0114-tts-deliver-to-gateway.md`

# 115 - Summary: `text_to_speech` optional gateway delivery (`deliver=true`)

## Goal

Reduce friction for Hermes-style “generate media then deliver to current chat” by letting `text_to_speech` optionally push the generated audio file through the active gateway adapter when running inside a gateway-triggered tool context.

## What changed

- `text_to_speech` schema adds:
  - `format` (mp3/wav/opus/aac; used by real backends)
  - `deliver` (boolean)
- When `deliver=true` and a gateway context is present (`ToolContext.GatewayPlatform` + `GatewayChatID`):
  - If the connected adapter implements `platform.MediaSender`, the tool sends the generated file immediately.
  - Tool output includes `delivered`, `delivery_platform`, `delivery_chat_id`, `delivery_message_id` (best-effort).
  - If delivery is not possible, tool still succeeds (audio generated) but sets `delivered=false` and `delivery_error`.

## Usage

- In a gateway-triggered conversation:
  - `text_to_speech(text="...", format="mp3", deliver=true)`

## Notes

- Delivery currently works on adapters that support `platform.MediaSender` (Telegram/Discord/Slack). Yuanbao delivery remains pending.
- Reply behavior uses the triggering message id as `reply_to` when available.
