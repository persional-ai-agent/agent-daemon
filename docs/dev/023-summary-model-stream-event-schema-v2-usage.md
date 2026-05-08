# 023 总结：`model_stream_event` v2 用量事件补齐

## 已完成

- `normalizeStreamEvent` 新增 `usage` 标准化：
  - `input_tokens -> prompt_tokens`
  - `output_tokens -> completion_tokens`
  - 自动补 `total_tokens`
- 三 provider 流式路径补发 `usage` 事件：
  - OpenAI：流式 chunk 的 `usage`
  - Anthropic：`message_delta.usage`
  - Codex：completed envelope 的 `response.usage`
- 模型层测试新增 `usage` 事件覆盖
- agent 层透传测试新增 `usage` 断言

## 验证

- `go test ./internal/model ./internal/agent` 通过
- `go test ./...` 通过

## 当前边界

- 仅提供最小统一用量字段，未覆盖 provider 的全部计费细节
