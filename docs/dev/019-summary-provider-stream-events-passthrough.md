# 019 总结：Provider 增量事件透传（最小版）

## 已完成

- 新增模型层可选事件扩展接口：
  - `EventClient`
  - `CompleteWithEvents`
  - `StreamEvent` / `StreamEventSink`
- OpenAI / Anthropic / Codex 流式分支新增增量事件上报
- `FallbackClient` 支持事件透传
- `Engine` 在模型调用中发出统一 `model_stream_event`
- 新增 `agent` 层测试覆盖 `model_stream_event` 透传

## 验证

- `go test ./internal/model ./internal/agent ./...` 通过

## 当前边界

- 事件类型仍是最小集合（`text_delta`、`tool_arguments_delta`）
- 暂未建立完整 provider 事件标准化字典
