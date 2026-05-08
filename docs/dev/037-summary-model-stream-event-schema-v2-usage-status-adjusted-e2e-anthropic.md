# 037 总结：`model_stream_event` v2 用量 `adjusted` 状态 Anthropic 端到端补齐

## 已完成

- 新增 Anthropic `adjusted` 流式端到端测试：
  - 上游偏小 `total_tokens` 经标准化后输出 `usage_consistency_status=adjusted`
  - 同时断言 `total_tokens_adjusted=true` 与校正后的 `total_tokens`
- 至此 `adjusted` 端到端覆盖已具备三 provider：
  - OpenAI
  - Anthropic
  - Codex

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 本轮仅补覆盖，不新增字段
