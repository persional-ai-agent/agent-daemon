# 031 总结：`model_stream_event` v2 用量一致性状态字段补齐

## 已完成

- `usage` 新增统一状态字段：
  - `usage_consistency_status`
- 状态已覆盖核心路径：
  - `derived`：总量自动补齐
  - `adjusted`：总量偏小已校正
  - `ok`：总量与推导一致
  - `source_only`：仅有上游总量
- 标准化测试已补齐状态断言。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 状态字段仅覆盖最小一致性集合，未扩展 provider 级诊断细分
