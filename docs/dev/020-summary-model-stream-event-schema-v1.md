# 020 总结：`model_stream_event` 标准字典（v1）

## 已完成

- 新增模型流式事件标准化逻辑，并在统一事件入口应用
- 标准化字段：
  - `text_delta` -> `event_data.text`
  - `tool_arguments_delta` -> `event_data.tool_name` + `event_data.arguments_delta`
- 支持历史别名兼容映射：
  - `delta/content` -> `text`
  - `name/function_name` -> `tool_name`
  - `delta/partial_json` -> `arguments_delta`
- 新增模型层标准化测试与 agent 层透传验证

## 验证

- `go test ./internal/model ./internal/agent ./...` 通过

## 当前边界

- 仅覆盖 v1 最小事件集，完整事件字典仍待后续扩展
