# 024 调研：`model_stream_event` v2 结束原因与 ID 别名归一

## 背景

023 已补齐 `usage` 统一事件，但 `message_done.finish_reason` 在不同 provider 仍存在枚举差异，同时工具调用 ID 字段也有别名分歧。

## 缺口

- 结束原因存在 provider 差异值（如 `end_turn`、`tool_use`、`max_tokens`）
- 工具调用 ID 可能出现在 `tool_use_id`，当前客户端仍需写兼容分支

## 本轮目标

- 统一 `message_done.finish_reason` 常见枚举到最小集合：
  - `stop`
  - `tool_calls`
  - `length`
- 为工具调用事件补齐 `tool_use_id -> tool_call_id` 兼容映射。

## 本轮边界

- 仅处理最小高频枚举与别名，不扩展 provider 全量原生结束原因
- 不改变外层事件结构
