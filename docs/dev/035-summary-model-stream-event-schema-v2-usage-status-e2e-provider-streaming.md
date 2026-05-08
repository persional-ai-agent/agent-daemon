# 035 总结：`model_stream_event` v2 用量状态 provider 流式端到端覆盖

## 已完成

- 新增 provider 流式端到端测试（均通过 `CompleteWithEvents`）：
  - OpenAI：`usage_consistency_status=source_only`
  - Anthropic：`usage_consistency_status=source_only`
  - Codex：`usage_consistency_status=invalid`
- 补齐了“provider 原始流式事件 -> 标准化字典事件”的关键闭环覆盖。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 本轮仅补 E2E 覆盖，不变更事件字段与归一规则
