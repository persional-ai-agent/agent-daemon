# 021 调研：`model_stream_event` 标准字典（v2 最小扩展）

## 背景

020 已统一 v1 字段（`text_delta`、`tool_arguments_delta`），但客户端仍缺少“消息开始/结束、工具调用开始/结束”的稳定节点。

## 缺口

- 仅有增量片段，客户端难以做进度条、分段渲染、工具调用状态展示
- provider 事件语义无法在 Agent 层形成统一生命周期

## 本轮目标

在 v1 基础上扩展 v2 最小生命周期事件：

- `message_start`
- `message_done`
- `tool_call_start`
- `tool_call_done`

并保持现有 `event_type` + `event_data` 结构不变。

## 本轮边界

- 仅补最小生命周期事件，不覆盖所有 provider 原生事件
- `message_id` 允许为空（某些 provider 不稳定返回）
