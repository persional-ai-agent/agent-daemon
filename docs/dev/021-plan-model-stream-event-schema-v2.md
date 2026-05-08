# 021 计划：`model_stream_event` 标准字典（v2 最小扩展）

## 目标

补齐 `model_stream_event` 的最小生命周期事件，提升客户端跨 provider 的一致消费能力。

## 实施步骤

1. 在 `normalizeStreamEvent` 中扩展 v2 事件标准化映射。
2. 在 OpenAI/Anthropic/Codex 流式路径补发：
   - `message_start` / `message_done`
   - `tool_call_start` / `tool_call_done`
3. 保持 `Engine` 透传结构不变（`provider`、`event_type`、`event_data`）。
4. 补模型层标准化测试与 agent 层透传测试。
5. 执行 `go test ./...` 回归。

## 验证标准

- 三 provider 流式路径具备最小生命周期事件
- `event_data` 字段在 v2 事件上保持统一语义
- 全量测试通过
