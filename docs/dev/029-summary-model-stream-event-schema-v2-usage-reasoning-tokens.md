# 029 总结：`model_stream_event` v2 用量推理 token 字段补齐

## 已完成

- `usage` 标准化新增 `reasoning_tokens` 字段。
- 兼容映射已补齐：
  - `completion_tokens_details.reasoning_tokens -> reasoning_tokens`
  - `output_tokens_details.reasoning_tokens -> reasoning_tokens`
  - `reasoning_tokens_count -> reasoning_tokens`
- 新增测试覆盖 OpenAI/Codex 两类嵌套字段来源。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 仅补最小推理 token 统计，未扩展更细粒度推理阶段指标
