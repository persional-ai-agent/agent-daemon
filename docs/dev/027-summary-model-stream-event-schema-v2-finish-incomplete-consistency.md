# 027 总结：`model_stream_event` v2 终止原因一致性补齐

## 已完成

- `message_done` 新增 `incomplete_reason` 归一规则：
  - `max_tokens/max_output_tokens -> length`
- 新增组合约束：
  - 当 `finish_reason=length` 且上游未给出 `incomplete_reason` 时，自动补齐为 `length`
- 测试新增：
  - `finish_reason=length` 自动补 `incomplete_reason`
  - `incomplete_reason` 别名归一

## 验证

- `go test ./internal/model ./internal/agent` 通过
- `go test ./...` 通过

## 当前边界

- 仍为最小一致性规则，未扩展 provider 全量终止诊断矩阵
