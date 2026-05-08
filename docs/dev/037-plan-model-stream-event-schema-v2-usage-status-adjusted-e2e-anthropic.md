# 037 计划：`model_stream_event` v2 用量 `adjusted` 状态 Anthropic 端到端补齐

## 目标

完成 `adjusted` 状态在 OpenAI/Anthropic/Codex 三 provider 的端到端测试闭环。

## 实施步骤

1. 在 `anthropic_stream_test.go` 新增 E2E 用例：
   - `message_delta.usage` 提供 `input/output/total` 且 `total` 偏小
   - 调用 `CompleteWithEvents`
   - 断言 `usage_consistency_status=adjusted`
   - 断言 `total_tokens_adjusted=true` 与校正后的 `total_tokens`
2. 执行 `go test ./internal/model` 与 `go test ./...`。
3. 更新 `docs/dev/README.md` 与 summary。

## 验证标准

- Anthropic `adjusted` E2E 通过
- 三 provider `adjusted` 端到端覆盖齐全
