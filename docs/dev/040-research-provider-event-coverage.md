# 040 调研：Provider 完整事件字典覆盖

## 背景

v2 标准事件字典已定义 9 种事件类型，三个 provider 均已实现。但各 provider 在关键字段的覆盖上存在差异，导致下游消费者（SSE 客户端、事件处理器）无法依赖统一字段。

## 当前覆盖矩阵

### 事件类型覆盖

| 事件 | OpenAI | Anthropic | Codex |
|------|--------|-----------|-------|
| `message_start` | ✅ | ✅ | ✅ |
| `message_done` | ✅ | ✅ | ✅ |
| `text_delta` | ✅ | ✅ | ✅ |
| `tool_call_start` | ✅ | ✅ | ✅ |
| `tool_call_done` | ✅ | ✅ | ✅ |
| `tool_args_start` | ✅ | ✅ | ✅ |
| `tool_args_delta` | ✅ | ✅ | ✅ |
| `tool_args_done` | ✅ | ✅ | ✅ |
| `usage` | ✅ | ✅ | ✅ |

### 关键字段覆盖差异

#### `message_start`

| 字段 | OpenAI | Anthropic | Codex |
|------|--------|-----------|-------|
| `message_id` | ❌ 缺失 | ✅ 从 `message.id` 提取 | ❌ 缺失 |

OpenAI 的 `chat/completions` 流式响应不提供顶层消息 ID。但 OpenAI 的非流式响应在 `choices[0].message` 中可能有 `id` 字段（部分兼容 API 提供）。

Codex 的 `response.output_item.added` 事件中 `item.id` 是输出项 ID，不是消息 ID。`response.created` 事件可能携带 `response.id`。

#### `message_done`

| 字段 | OpenAI | Anthropic | Codex |
|------|--------|-----------|-------|
| `message_id` | ❌ 缺失 | ✅ | ✅ (`response_id`) |
| `finish_reason` | ✅ | ✅ | ✅ |
| `stop_sequence` | ❌ 缺失 | ✅ | ❌ 缺失 |
| `incomplete_reason` | ❌ 缺失 | ❌ 缺失 | ✅ |

## 缺口清单

1. **OpenAI `message_start` 缺 `message_id`**：OpenAI 流式不提供消息 ID，但 normalizeStreamEvent 已能从 `id`/`response_id` 别名补齐。问题在于 OpenAI 流式根本不发这些字段。
2. **OpenAI `message_done` 缺 `message_id`**：同上。
3. **OpenAI `message_done` 缺 `stop_sequence`**：OpenAI 不提供 stop_sequence，无法从原始数据提取。
4. **OpenAI `message_done` 缺 `incomplete_reason`**：OpenAI 的 `length` finish_reason 对应的详细信息不在流式响应中。
5. **Codex `message_start` 缺 `message_id`**：Codex 流式有 `response.id`，但当前未在 `message_start` 中提取。
6. **Anthropic `message_done` 缺 `incomplete_reason`**：Anthropic 的 `max_tokens` stop_reason 可映射为 `incomplete_reason`。

## 方案

### 可补齐的缺口

1. **Codex `message_start` 补 `message_id`**：从 `response.created` 或首个 `response.output_item.added` 事件的 `response.id` 提取。
2. **Anthropic `message_done` 补 `incomplete_reason`**：当 `stop_reason=max_tokens` 时，设置 `incomplete_reason=length`。
3. **OpenAI/Codex `message_done` 补 `incomplete_reason`**：当 `finish_reason=length` 时，设置 `incomplete_reason=length`（normalizeStreamEvent 已处理此逻辑，但 provider 层应主动提供）。

### 不可补齐的缺口（上游 API 限制）

1. **OpenAI `message_start/message_done` 的 `message_id`**：OpenAI `chat/completions` 流式响应不提供消息 ID。normalizeStreamEvent 的别名归一逻辑已覆盖，但原始数据缺失。后续如果 OpenAI API 增加此字段可自动适配。
2. **OpenAI `stop_sequence`**：OpenAI 不支持自定义 stop sequence 的流式返回。

## 结论

本轮补齐可从 provider 层主动提供的字段入手，确保 `normalizeStreamEvent` 的归一逻辑有原始数据可用。不可补齐的缺口属于上游 API 限制，保持当前 normalizeStreamEvent 的兼容处理即可。
