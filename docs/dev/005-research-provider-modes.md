# 005 调研：Provider 多模式补齐

## 背景

当前 Go 版仅支持 OpenAI 兼容 `chat/completions`，与 Hermes 的多 provider 能力存在差距。

在不引入过度复杂抽象的前提下，优先补齐第二种主流模式：Anthropic Messages API。

## 差异点

- 缺少 provider 选择机制
- 缺少 Anthropic 消息协议转换
- 缺少跨 provider 的 tool call 结构映射

## 方案

采用“统一 `model.Client` 接口 + provider 实现分文件”的方式：

- 保持 `Engine` 与 `Agent Loop` 不变
- 新增 `AnthropicClient` 实现同一 `ChatCompletion` 接口
- 启动时按 `AGENT_MODEL_PROVIDER` 选择 client

## 映射策略

- 内部仍统一使用 OpenAI 风格 `core.Message`
- 调 Anthropic 前做协议转换：
  - `system` 提取为独立 `system` 字段
  - `tool` 消息映射为 `tool_result` block
  - `assistant.tool_calls` 映射为 `tool_use` block
- 从 Anthropic 响应反解回 `core.Message` + `ToolCalls`

## 结论

该方案可在不改核心 loop 的前提下，把 provider 能力从单模式扩展到双模式，后续再继续补 Codex/Anthropic 高级特性与更多 provider。
