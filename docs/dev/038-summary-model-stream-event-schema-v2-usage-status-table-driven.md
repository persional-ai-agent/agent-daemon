# 038 总结：`model_stream_event` v2 用量状态表驱动测试补齐

## 已完成

- 新增 `usage_consistency_status` 表驱动测试，统一覆盖核心状态：
  - `ok`
  - `derived`
  - `source_only`
  - `adjusted`
- 用例内同步断言关键副字段：
  - `total_tokens`
  - `total_tokens_adjusted`（按场景）

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 本轮仅优化测试组织，不新增字段或状态
