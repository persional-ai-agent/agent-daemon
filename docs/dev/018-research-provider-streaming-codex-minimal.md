# 018 调研：Provider 流式统一（Codex 最小落地）

## 背景

016/017 已补齐 OpenAI 与 Anthropic 的流式聚合，Codex 仍是流式能力缺口。  
要完成当前阶段的 provider 流式统一，需要补齐 Codex。

## 缺口

- Codex 客户端未启用 `stream=true`
- 缺少 Codex SSE 事件聚合逻辑

## 本轮目标

最小补齐 Codex 流式聚合：

- 增加 `UseStreaming` 开关
- 开关开启时发送 `stream=true`
- 解析并聚合核心事件：
  - `response.output_item.added`
  - `response.output_text.delta`
  - `response.function_call_arguments.delta`
  - 兼容 `response.output` 完整包
- 输出统一转换为标准 `core.Message`

## 本轮边界

- 仍未透传 provider 增量事件到 Agent 事件流
- 未实现并行竞速与熔断
