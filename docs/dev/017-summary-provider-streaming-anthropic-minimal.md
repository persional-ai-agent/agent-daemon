# 017 总结：Provider 流式统一（Anthropic 最小落地）

## 已完成

- `AnthropicClient` 新增 `UseStreaming` 开关
- 开关开启时使用 `stream=true` 调用
- 新增 Anthropic SSE 聚合逻辑：
  - `content_block_start`
  - `content_block_delta.text_delta`
  - `content_block_delta.input_json_delta`
- 聚合结果统一转换为标准 `core.Message`
- 启动装配支持将 `AGENT_MODEL_USE_STREAMING` 传入 Anthropic 客户端
- 新增测试：
  - 流式文本聚合
  - 流式工具调用聚合
  - 装配层 Anthropic 流式开关

## 验证

- `go test ./internal/model ./cmd/agentd ./...` 通过

## 当前边界

- Codex 流式仍未补齐
- provider 增量事件仍未透传到 Agent 事件流
