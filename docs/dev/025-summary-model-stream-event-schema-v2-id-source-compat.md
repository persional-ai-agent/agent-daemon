# 025 总结：`model_stream_event` v2 消息/工具 ID 来源兼容补齐

## 已完成

- 标准化层增强 `message_id` 来源兼容：
  - `id`
  - `response_id`
  - `message.id`
- 标准化层增强 `tool_call_id` 来源兼容：
  - `call_id`
  - `tool_use_id`
  - `item_id`
  - `output_item_id`
- Anthropic 流式路径新增：
  - 解析 `message_start.message.id` 并透传
  - 透传 `message_delta.stop_reason`
- Codex completed envelope 新增：
  - 透传 `response.id` 为 `response_id`
- 模型层测试新增别名来源覆盖，定向回归通过。

## 验证

- `go test ./internal/model ./internal/agent` 通过
- `go test ./...` 通过

## 当前边界

- 仍为最小兼容集合，未覆盖 provider 的全部元数据字段
