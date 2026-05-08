# 016 总结：Provider 流式统一（OpenAI 最小落地）

## 已完成

- `OpenAIClient` 新增 `UseStreaming` 开关
- 开关开启时，`ChatCompletion` 使用 `stream=true` 请求
- 新增 SSE 聚合逻辑：
  - 文本增量合并为最终 `content`
  - 工具调用按 index 聚合并重组 `tool_calls`
- 新增配置项：`AGENT_MODEL_USE_STREAMING`
- 启动装配支持将流式配置传入 OpenAI 客户端
- 新增测试覆盖：
  - 流式文本聚合
  - 流式工具调用聚合
  - 装配层流式开关行为

## 验证

- `go test ./internal/model ./cmd/agentd ./...` 通过

## 当前边界

- Anthropic/Codex 仍未接入流式模式
- provider 级增量事件暂未透传到 Agent 事件流
