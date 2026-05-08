# 023 计划：`model_stream_event` v2 用量事件补齐

## 目标

为 `model_stream_event` 增加跨 provider 可统一消费的 `usage` 事件，并规范最小字段集。

## 实施步骤

1. 扩展 `normalizeStreamEvent`：
   - 新增 `usage` 字段标准化
   - 对齐 `input_tokens/output_tokens` 到 `prompt_tokens/completion_tokens`
   - 缺少 `total_tokens` 时用前两者自动补齐
2. OpenAI 流式路径解析并发出 `usage`。
3. Anthropic 流式路径解析 `message_delta.usage` 并发出 `usage`。
4. Codex 流式路径解析 completed envelope 的 `response.usage` 并发出 `usage`。
5. 更新模型层与 agent 层测试。
6. 全量回归 `go test ./...`。

## 验证标准

- `model_stream_event` 可收到 `event_type=usage`
- `event_data` 至少可稳定提供 `prompt_tokens/completion_tokens/total_tokens`
- 全量测试通过
