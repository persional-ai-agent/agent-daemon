# 036 总结：`model_stream_event` v2 用量 `adjusted` 状态端到端覆盖

## 已完成

- 新增 `adjusted` 的 provider 流式端到端测试：
  - OpenAI：`usage_consistency_status=adjusted`
  - Codex：`usage_consistency_status=adjusted`
- 两条用例均额外断言：
  - `total_tokens_adjusted=true`
  - `total_tokens` 为校正值

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- Anthropic `adjusted` E2E 未补（可在下一轮补齐）
