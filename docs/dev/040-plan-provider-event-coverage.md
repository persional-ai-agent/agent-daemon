# 040 计划：Provider 完整事件字典覆盖

## 目标

补齐各 provider 流式事件中可主动提供的关键字段，使下游消费者可依赖统一字段。

## 实施步骤

### 1. Codex `message_start` 补 `message_id`

修改 `internal/model/codex.go`：
- 在 `response.output_item.added` 事件中检查 `response.id`
- 或在流式开始时从首个事件提取 `response_id`
- 将 `message_id` 传入 `message_start` 事件

验证：Codex 流式 `message_start` 事件包含 `message_id`

### 2. Anthropic `message_done` 补 `incomplete_reason`

修改 `internal/model/anthropic.go`：
- 当 `stop_reason=max_tokens` 时，在 `message_done` 中设置 `incomplete_reason=length`

验证：Anthropic `max_tokens` stop_reason 时 `message_done` 包含 `incomplete_reason`

### 3. OpenAI `message_done` 补 `incomplete_reason`

修改 `internal/model/openai.go`：
- 当 `finish_reason=length` 时，在 `message_done` 中设置 `incomplete_reason=length`

验证：OpenAI `length` finish_reason 时 `message_done` 包含 `incomplete_reason`

### 4. 增加测试并回归

- 补充各 provider 的事件字段覆盖测试
- `go test ./...` 通过

## 模块影响

- `internal/model/anthropic.go`
- `internal/model/codex.go`
- `internal/model/openai.go`

## 取舍

- OpenAI 的 `message_id` 和 `stop_sequence` 属于上游 API 限制，本轮不补齐
- 仅补齐 provider 层可主动提供的字段
- 不修改 `normalizeStreamEvent` 的归一逻辑（已足够健壮）
