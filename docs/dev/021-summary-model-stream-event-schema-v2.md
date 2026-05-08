# 021 总结：`model_stream_event` 标准字典（v2 最小扩展）

## 已完成

- 在模型层事件标准化中新增 v2 生命周期事件映射：
  - `message_start` / `message_done`
  - `tool_call_start` / `tool_call_done`
- OpenAI / Anthropic / Codex 流式路径均补发上述事件
- 保持 Agent 透传结构不变，客户端仍按 `event_type` + `event_data` 消费
- 补测试：
  - 模型层 v2 标准化映射测试
  - Agent 层 v2 事件透传测试

## 验证

- `go test ./internal/model ./internal/agent ./...` 通过

## 当前边界

- 仍是最小事件集，完整 provider 原生事件字典未全部纳入
