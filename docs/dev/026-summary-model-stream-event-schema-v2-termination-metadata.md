# 026 总结：`model_stream_event` v2 终止元数据补齐

## 已完成

- `normalizeStreamEvent(message_done)` 新增终止元数据归一：
  - `stop -> stop_sequence`
  - `incomplete_details.reason` / `reason_detail -> incomplete_reason`
- Anthropic 流式路径补透传：
  - `message_delta.stop_sequence`
- Codex completed envelope 补透传：
  - `response.incomplete_details.reason`
- 测试补齐：
  - 标准化测试新增终止元字段断言
  - Anthropic/Codex 流式测试新增透传断言

## 验证

- `go test ./internal/model ./internal/agent` 通过
- `go test ./...` 通过

## 当前边界

- 仅覆盖最小终止元字段，未纳入 provider 全量中止诊断数据
