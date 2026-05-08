# 018 总结：Provider 流式统一（Codex 最小落地）

## 已完成

- `CodexClient` 新增 `UseStreaming` 开关
- 开关开启时使用 `stream=true` 调用
- 新增 Codex SSE 聚合逻辑，支持：
  - `response.output_item.added`
  - `response.output_text.delta`
  - `response.function_call_arguments.delta`
  - `response.output` 完整结果包兼容
- 聚合结果统一转换为标准 `core.Message`
- 启动装配支持将 `AGENT_MODEL_USE_STREAMING` 传入 Codex 客户端
- 新增测试：
  - 流式文本聚合
  - 流式函数调用聚合
  - 装配层 Codex 流式开关

## 验证

- `go test ./internal/model ./cmd/agentd ./...` 通过

## 当前边界

- provider 增量事件仍未透传到 Agent 事件流
- 未实现并行竞速与熔断
