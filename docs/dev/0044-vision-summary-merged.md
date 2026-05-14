# 0044 vision summary merged

## 模块

- `vision`

## 类型

- `summary`

## 合并来源

- `0055-vision-summary-merged.md`

## 合并内容

### 来源：`0055-vision-summary-merged.md`

# 0055 vision summary merged

## 模块

- `vision`

## 类型

- `summary`

## 合并来源

- `0120-vision-analyze-openai-backend.md`

## 合并内容

### 来源：`0120-vision-analyze-openai-backend.md`

# 121 - Summary: `vision_analyze` optional OpenAI vision backend

## Goal

Reduce the “metadata-only” gap for `vision_analyze` by enabling a best-effort real vision backend when `OPENAI_API_KEY` is configured, while keeping the lightweight fallback when not configured or when requests fail.

## What changed

- `internal/tools/builtin.go`:
  - `vision_analyze` schema now includes optional `question` (string).
- `internal/tools/media_tools.go`:
  - Reads the local image bytes (<= 10MB) and detects width/height via `image.DecodeConfig`.
  - If `OPENAI_API_KEY` is set, calls `POST /chat/completions` with a text+image payload (`image_url` as data URL).
  - Env knobs:
    - `OPENAI_BASE_URL` (default `https://api.openai.com/v1`)
    - `OPENAI_VISION_MODEL` (default `gpt-4o-mini`)
  - On success returns `analysis` text plus basic metadata; otherwise falls back to metadata-only output with a hint.

## Notes

- Network access is required for the OpenAI backend; in restricted environments the tool will fall back to metadata-only.
- The fallback behavior keeps the tool usable without credentials.
