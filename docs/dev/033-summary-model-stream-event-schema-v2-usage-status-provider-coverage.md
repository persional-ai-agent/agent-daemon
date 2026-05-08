# 033 总结：`model_stream_event` v2 用量状态 provider 覆盖测试

## 已完成

- 新增 `usage_consistency_status` 的 provider 覆盖测试：
  - OpenAI 场景断言 `ok`
  - Anthropic 场景断言 `derived`
  - Codex 场景断言 `invalid`
- 测试集中在 `internal/model/streaming_test.go`，不改协议字段。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 本轮仅补覆盖，不新增字段或状态
