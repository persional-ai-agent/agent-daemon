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

