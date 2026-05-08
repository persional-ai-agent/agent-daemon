# 038 计划：`model_stream_event` v2 用量状态表驱动测试补齐

## 目标

提升 `usage_consistency_status` 测试的可维护性与可读性。

## 实施步骤

1. 在 `internal/model/streaming_test.go` 增加表驱动测试：
   - 覆盖 `ok/derived/source_only/adjusted`
2. 在每个 case 中统一断言：
   - `usage_consistency_status`
   - `total_tokens`
   - `total_tokens_adjusted`（按需）
3. 执行 `go test ./internal/model` 与 `go test ./...`。
4. 更新 `docs/dev/README.md` 与 summary。

## 验证标准

- 表驱动用例覆盖核心状态集合
- 全量测试通过
