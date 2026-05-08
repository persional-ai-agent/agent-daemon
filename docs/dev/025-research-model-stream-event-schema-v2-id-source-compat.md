# 025 调研：`model_stream_event` v2 消息/工具 ID 来源兼容补齐

## 背景

024 已统一 `finish_reason` 与 `tool_use_id`，但在跨 provider 的事件消费中，`message_id` 与 `tool_call_id` 仍可能来自不同字段来源。

## 缺口

- `message_id` 可能来自 `id`、`response_id` 或 `message.id`
- `tool_call_id` 可能来自 `call_id`、`tool_use_id`、`item_id`、`output_item_id`
- provider 的 completed envelope 中可能携带 `response.id`，当前统一层应可自动兜底

## 本轮目标

- 扩展标准化映射，统一提取 `message_id` 与 `tool_call_id`
- 在 provider 流式路径补齐可用来源字段透传（如 Codex `response_id`、Anthropic `message_start.message.id`）
- 保持现有外层协议不变

## 本轮边界

- 不引入新事件类型
- 仅做字段来源兼容与最小测试增强
