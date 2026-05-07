# 001 事件协议：Hermes Agent Go 版

## 目标

定义 `AgentEvent` 在 SSE 与内部回调面的稳定字段，用于前端、SDK 或测试统一消费。

## 基础结构

所有事件都使用统一结构：

```json
{
  "type": "tool_finished",
  "session_id": "session-id",
  "turn": 1,
  "tool_name": "read_file",
  "content": "...",
  "data": {}
}
```

公共字段说明：

- `type`：事件类型
- `session_id`：当前会话或子任务会话 ID
- `turn`：当前回合序号
- `tool_name`：工具事件对应的工具名
- `content`：面向展示的原始文本内容
- `data`：结构化扩展字段

## 主要事件

### 会话事件

- `user_message`
- `turn_started`
- `assistant_message`
- `completed`
- `cancelled`
- `error`
- `max_iterations_reached`

### 工具事件

- `tool_started`
- `tool_finished`

### 委派事件

- `delegate_started`
- `delegate_finished`
- `delegate_failed`

## 结构化字段

### `assistant_message`

`data` 字段包含：

- `status`
- `message_role`
- `content_length`
- `tool_call_count`
- `has_tool_calls`

### `completed`

`data` 字段包含：

- `status`
- `message_role`
- `content_length`
- `tool_call_count`
- `has_tool_calls`
- `finished_naturally`

### `tool_started`

`data` 字段包含：

- `status`
- `tool_call_id`
- `tool_name`
- `arguments`

### `tool_finished`

`data` 字段包含：

- `status`
- `success`
- `tool_call_id`
- `tool_name`
- `arguments`
- `result`

`result` 在工具返回 JSON 时为对象；否则为原始字符串。

### `delegate_started`

`data` 字段包含：

- `parent_session_id`
- `goal`
- `status`

### `delegate_finished`

`data` 字段包含：

- `parent_session_id`
- `goal`
- `status`
- `success`
- `result`

### `delegate_failed`

`data` 字段包含：

- `parent_session_id`
- `goal`
- `status`
- `success`
- `error`

### `cancelled`

`data` 字段包含：

- `status`
- `turn`
- `error`

SSE 兜底取消事件至少包含：

- `session_id`
- `status`
- `reason`

### `error`

`data` 字段包含：

- `status`
- `turn`
- `error`

SSE 兜底错误事件至少包含：

- `session_id`
- `status`
- `error`

### `max_iterations_reached`

`data` 字段包含：

- `status`
- `max_iterations`
- `finished`

## 兼容性约定

- 新增字段优先追加到 `data` 中，避免破坏现有顶层结构。
- `content` 保留为可直接展示的文本，不要求客户端再反向解析。
- 客户端若需要稳定消费，应优先读取 `data`，将 `content` 作为展示兜底。
