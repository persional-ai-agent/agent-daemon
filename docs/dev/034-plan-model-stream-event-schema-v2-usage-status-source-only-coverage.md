# 034 计划：`model_stream_event` v2 用量 `source_only` 状态 provider 覆盖测试

## 目标

增强 `usage_consistency_status` 的测试完整性，补齐 `source_only` 在三 provider 的最小覆盖。

## 实施步骤

1. 在 `internal/model/streaming_test.go` 新增三条测试：
   - OpenAI：仅 `total_tokens` -> `source_only`
   - Anthropic：仅 `total_tokens` -> `source_only`
   - Codex：仅 `total_tokens` -> `source_only`
2. 执行 `go test ./internal/model`。
3. 执行 `go test ./...` 全量回归。
4. 同步 `docs/dev/README.md` 与 summary 文档。

## 验证标准

- 三 provider 场景均稳定断言 `source_only`
- 全量测试通过
