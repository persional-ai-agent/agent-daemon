# 033 计划：`model_stream_event` v2 用量状态 provider 覆盖测试

## 目标

增强 `usage_consistency_status` 的回归稳定性，确保不同 provider 的典型输入都可落入预期状态。

## 实施步骤

1. 在 `internal/model/streaming_test.go` 增加 provider 维度测试：
   - OpenAI 一致总量 -> `ok`
   - Anthropic 输入/输出推导 -> `derived`
   - Codex 非法 token -> `invalid`
2. 执行 `go test ./internal/model` 与全量回归。
3. 更新 dev 文档索引与总结。

## 验证标准

- provider 维度断言可稳定通过
- 全量测试通过
