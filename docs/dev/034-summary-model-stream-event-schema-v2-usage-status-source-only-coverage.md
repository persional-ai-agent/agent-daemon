# 034 总结：`model_stream_event` v2 用量 `source_only` 状态 provider 覆盖测试

## 已完成

- 新增 `source_only` 的 provider 覆盖测试：
  - OpenAI：仅 `total_tokens` 输入断言 `source_only`
  - Anthropic：仅 `total_tokens` 输入断言 `source_only`
  - Codex：仅 `total_tokens` 输入断言 `source_only`
- 测试仅扩展覆盖，不变更标准化逻辑与事件协议。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 本轮仅补覆盖，不新增状态或字段
