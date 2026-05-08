# 030 总结：`model_stream_event` v2 用量总量一致性补齐

## 已完成

- `usage.total_tokens` 一致性兜底已补齐：
  - 缺失时自动按 `prompt_tokens + completion_tokens` 补齐
  - 当上游总量偏小时自动校正为推导总量
- 新增校正标记：
  - `total_tokens_adjusted=true`
- 新增测试覆盖：
  - 缺失总量补齐
  - 偏小总量校正

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 仅在统一层做最小一致性兜底，不改 provider 侧原始 usage 采集
