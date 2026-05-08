# 035 调研：`model_stream_event` v2 用量状态 provider 流式端到端覆盖

## 背景

034 已补 `source_only` 的 provider 维度标准化测试，但仍缺“provider 流式输出 -> `CompleteWithEvents` 标准化”的端到端覆盖。

## 缺口

- 现有 provider stream 测试多数直接断言原始事件
- 未显式覆盖 `CompleteWithEvents` 对 `usage_consistency_status` 的跨 provider 端到端归一结果

## 本轮目标

补齐 provider 流式端到端用例：

- OpenAI 流式 usage -> `source_only`
- Anthropic 流式 usage -> `source_only`
- Codex completed envelope usage -> `invalid`

## 本轮边界

- 仅补测试覆盖，不修改标准化规则
