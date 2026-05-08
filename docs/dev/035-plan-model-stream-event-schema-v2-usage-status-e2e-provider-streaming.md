# 035 计划：`model_stream_event` v2 用量状态 provider 流式端到端覆盖

## 目标

验证 `usage_consistency_status` 在真实 provider 流式路径下经过 `CompleteWithEvents` 后仍稳定可用。

## 实施步骤

1. 在 provider 测试中新增 E2E 用例：
   - `openai_stream_test.go`：断言 `source_only`
   - `anthropic_stream_test.go`：断言 `source_only`
   - `codex_stream_test.go`：断言 `invalid`
2. 用例统一调用 `CompleteWithEvents`，通过 sink 读取标准化事件。
3. 执行 `go test ./internal/model` 与 `go test ./...`。
4. 更新 `docs/dev/README.md` 与总结文档。

## 验证标准

- 三个 provider 的端到端断言均通过
- 全量测试通过
