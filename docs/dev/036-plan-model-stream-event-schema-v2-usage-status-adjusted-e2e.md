# 036 计划：`model_stream_event` v2 用量 `adjusted` 状态端到端覆盖

## 目标

验证 `adjusted` 状态在 provider 流式路径下经过 `CompleteWithEvents` 后可稳定输出。

## 实施步骤

1. 在 `openai_stream_test.go` 新增 E2E 用例：
   - 输入 `prompt/completion/total` 且 `total` 偏小
   - 断言 `usage_consistency_status=adjusted`
   - 断言 `total_tokens_adjusted=true` 和校正后的 `total_tokens`
2. 在 `codex_stream_test.go` 新增同类 E2E 用例。
3. 执行 `go test ./internal/model` 与 `go test ./...`。
4. 更新 dev 文档索引与总结。

## 验证标准

- OpenAI/Codex 两条 E2E 均通过
- 校正字段断言稳定
