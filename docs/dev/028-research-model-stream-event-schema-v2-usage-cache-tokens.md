# 028 调研：`model_stream_event` v2 用量缓存 token 字段补齐

## 背景

当前 `usage` 已统一 `prompt/completion/total_tokens`，但缓存命中相关 token 在不同 provider 命名差异较大。

## 缺口

- Anthropic 常见字段：
  - `cache_creation_input_tokens`
  - `cache_read_input_tokens`
- OpenAI/Codex 常见字段：
  - `prompt_tokens_details.cached_tokens`
  - `input_tokens_details.cached_tokens`
- 客户端难以跨 provider 统一展示 cache token 统计。

## 本轮目标

在 `usage` 事件中增加最小统一字段：

- `prompt_cache_write_tokens`
- `prompt_cache_read_tokens`

## 本轮边界

- 仅补最小缓存 token 字段，不扩展计费明细矩阵
