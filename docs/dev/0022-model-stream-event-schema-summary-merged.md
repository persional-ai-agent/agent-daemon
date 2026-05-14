# 0022 model-stream-event-schema summary merged

## 模块

- `model-stream-event-schema`

## 类型

- `summary`

## 合并来源

- `0030-model-stream-event-schema-summary-merged.md`

## 合并内容

### 来源：`0030-model-stream-event-schema-summary-merged.md`

# 0030 model-stream-event-schema summary merged

## 模块

- `model-stream-event-schema`

## 类型

- `summary`

## 合并来源

- `0019-model-stream-event-schema-v1.md`
- `0020-model-stream-event-schema-v2.md`
- `0021-model-stream-event-schema-v2-args-lifecycle.md`
- `0022-model-stream-event-schema-v2-usage.md`
- `0023-model-stream-event-schema-v2-finish-reason-and-id-aliases.md`
- `0024-model-stream-event-schema-v2-id-source-compat.md`
- `0025-model-stream-event-schema-v2-termination-metadata.md`
- `0026-model-stream-event-schema-v2-finish-incomplete-consistency.md`
- `0027-model-stream-event-schema-v2-usage-cache-tokens.md`
- `0028-model-stream-event-schema-v2-usage-reasoning-tokens.md`
- `0029-model-stream-event-schema-v2-usage-total-consistency.md`
- `0030-model-stream-event-schema-v2-usage-consistency-status.md`
- `0031-model-stream-event-schema-v2-usage-status-invalid.md`
- `0032-model-stream-event-schema-v2-usage-status-provider-coverage.md`
- `0033-model-stream-event-schema-v2-usage-status-source-only-coverage.md`
- `0034-model-stream-event-schema-v2-usage-status-e2e-provider-streaming.md`
- `0035-model-stream-event-schema-v2-usage-status-adjusted-e2e.md`
- `0036-model-stream-event-schema-v2-usage-status-adjusted-e2e-anthropic.md`
- `0037-model-stream-event-schema-v2-usage-status-table-driven.md`

## 合并内容

### 来源：`0019-model-stream-event-schema-v1.md`

# 020 总结：`model_stream_event` 标准字典（v1）

## 已完成

- 新增模型流式事件标准化逻辑，并在统一事件入口应用
- 标准化字段：
  - `text_delta` -> `event_data.text`
  - `tool_arguments_delta` -> `event_data.tool_name` + `event_data.arguments_delta`
- 支持历史别名兼容映射：
  - `delta/content` -> `text`
  - `name/function_name` -> `tool_name`
  - `delta/partial_json` -> `arguments_delta`
- 新增模型层标准化测试与 agent 层透传验证

## 验证

- `go test ./internal/model ./internal/agent ./...` 通过

## 当前边界

- 仅覆盖 v1 最小事件集，完整事件字典仍待后续扩展

### 来源：`0020-model-stream-event-schema-v2.md`

# 021 总结：`model_stream_event` 标准字典（v2 最小扩展）

## 已完成

- 在模型层事件标准化中新增 v2 生命周期事件映射：
  - `message_start` / `message_done`
  - `tool_call_start` / `tool_call_done`
- OpenAI / Anthropic / Codex 流式路径均补发上述事件
- 保持 Agent 透传结构不变，客户端仍按 `event_type` + `event_data` 消费
- 补测试：
  - 模型层 v2 标准化映射测试
  - Agent 层 v2 事件透传测试

## 验证

- `go test ./internal/model ./internal/agent ./...` 通过

## 当前边界

- 仍是最小事件集，完整 provider 原生事件字典未全部纳入

### 来源：`0021-model-stream-event-schema-v2-args-lifecycle.md`

# 022 总结：`model_stream_event` v2 参数生命周期补齐

## 已完成

- 事件标准化层新增参数生命周期：
  - `tool_args_start`
  - `tool_args_delta`
  - `tool_args_done`
- 兼容历史别名：
  - `tool_arguments_start/delta/done`
- 三 provider 流式路径均补发参数生命周期事件
- `message_done` 增加统一 `finish_reason`：
  - 有工具调用：`tool_calls`
  - 无工具调用：`stop`
- 更新模型层与 agent 层测试，覆盖别名映射与事件透传

## 验证

- `go test ./internal/model ./internal/agent ./...` 通过

## 当前边界

- 仍为统一字典最小集合，完整 provider 原生事件全集未纳入

### 来源：`0022-model-stream-event-schema-v2-usage.md`

# 023 总结：`model_stream_event` v2 用量事件补齐

## 已完成

- `normalizeStreamEvent` 新增 `usage` 标准化：
  - `input_tokens -> prompt_tokens`
  - `output_tokens -> completion_tokens`
  - 自动补 `total_tokens`
- 三 provider 流式路径补发 `usage` 事件：
  - OpenAI：流式 chunk 的 `usage`
  - Anthropic：`message_delta.usage`
  - Codex：completed envelope 的 `response.usage`
- 模型层测试新增 `usage` 事件覆盖
- agent 层透传测试新增 `usage` 断言

## 验证

- `go test ./internal/model ./internal/agent` 通过
- `go test ./...` 通过

## 当前边界

- 仅提供最小统一用量字段，未覆盖 provider 的全部计费细节

### 来源：`0023-model-stream-event-schema-v2-finish-reason-and-id-aliases.md`

# 024 总结：`model_stream_event` v2 结束原因与 ID 别名归一

## 已完成

- `message_done.finish_reason` 新增常见枚举归一：
  - `end_turn -> stop`
  - `tool_use -> tool_calls`
  - `max_tokens/max_output_tokens -> length`
- `tool_call_*` 与 `tool_args_*` 事件新增 `tool_use_id -> tool_call_id` 兼容映射。
- 模型层与 agent 层测试已补齐：
  - 结束原因归一
  - `tool_use_id` 别名
  - 透传链路中的 `reason=end_turn` 归一到 `finish_reason=stop`

## 验证

- `go test ./internal/model ./internal/agent` 通过
- `go test ./...` 通过

## 当前边界

- 仅收敛最小高频结束原因，未覆盖 provider 全量原生 stop reason

### 来源：`0024-model-stream-event-schema-v2-id-source-compat.md`

# 025 总结：`model_stream_event` v2 消息/工具 ID 来源兼容补齐

## 已完成

- 标准化层增强 `message_id` 来源兼容：
  - `id`
  - `response_id`
  - `message.id`
- 标准化层增强 `tool_call_id` 来源兼容：
  - `call_id`
  - `tool_use_id`
  - `item_id`
  - `output_item_id`
- Anthropic 流式路径新增：
  - 解析 `message_start.message.id` 并透传
  - 透传 `message_delta.stop_reason`
- Codex completed envelope 新增：
  - 透传 `response.id` 为 `response_id`
- 模型层测试新增别名来源覆盖，定向回归通过。

## 验证

- `go test ./internal/model ./internal/agent` 通过
- `go test ./...` 通过

## 当前边界

- 仍为最小兼容集合，未覆盖 provider 的全部元数据字段

### 来源：`0025-model-stream-event-schema-v2-termination-metadata.md`

# 026 总结：`model_stream_event` v2 终止元数据补齐

## 已完成

- `normalizeStreamEvent(message_done)` 新增终止元数据归一：
  - `stop -> stop_sequence`
  - `incomplete_details.reason` / `reason_detail -> incomplete_reason`
- Anthropic 流式路径补透传：
  - `message_delta.stop_sequence`
- Codex completed envelope 补透传：
  - `response.incomplete_details.reason`
- 测试补齐：
  - 标准化测试新增终止元字段断言
  - Anthropic/Codex 流式测试新增透传断言

## 验证

- `go test ./internal/model ./internal/agent` 通过
- `go test ./...` 通过

## 当前边界

- 仅覆盖最小终止元字段，未纳入 provider 全量中止诊断数据

### 来源：`0026-model-stream-event-schema-v2-finish-incomplete-consistency.md`

# 027 总结：`model_stream_event` v2 终止原因一致性补齐

## 已完成

- `message_done` 新增 `incomplete_reason` 归一规则：
  - `max_tokens/max_output_tokens -> length`
- 新增组合约束：
  - 当 `finish_reason=length` 且上游未给出 `incomplete_reason` 时，自动补齐为 `length`
- 测试新增：
  - `finish_reason=length` 自动补 `incomplete_reason`
  - `incomplete_reason` 别名归一

## 验证

- `go test ./internal/model ./internal/agent` 通过
- `go test ./...` 通过

## 当前边界

- 仍为最小一致性规则，未扩展 provider 全量终止诊断矩阵

### 来源：`0027-model-stream-event-schema-v2-usage-cache-tokens.md`

# 028 总结：`model_stream_event` v2 用量缓存 token 字段补齐

## 已完成

- `usage` 标准化新增缓存 token 字段：
  - `prompt_cache_write_tokens`
  - `prompt_cache_read_tokens`
- 兼容映射已补齐：
  - `cache_creation_input_tokens -> prompt_cache_write_tokens`
  - `cache_read_input_tokens -> prompt_cache_read_tokens`
  - `prompt_tokens_details.cached_tokens -> prompt_cache_read_tokens`
  - `input_tokens_details.cached_tokens -> prompt_cache_read_tokens`
- 新增标准化测试覆盖直接字段与嵌套字段。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 仅补最小缓存 token 字段，未扩展 provider 全量计费维度

### 来源：`0028-model-stream-event-schema-v2-usage-reasoning-tokens.md`

# 029 总结：`model_stream_event` v2 用量推理 token 字段补齐

## 已完成

- `usage` 标准化新增 `reasoning_tokens` 字段。
- 兼容映射已补齐：
  - `completion_tokens_details.reasoning_tokens -> reasoning_tokens`
  - `output_tokens_details.reasoning_tokens -> reasoning_tokens`
  - `reasoning_tokens_count -> reasoning_tokens`
- 新增测试覆盖 OpenAI/Codex 两类嵌套字段来源。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 仅补最小推理 token 统计，未扩展更细粒度推理阶段指标

### 来源：`0029-model-stream-event-schema-v2-usage-total-consistency.md`

# 030 总结：`model_stream_event` v2 用量总量一致性补齐

## 已完成

- `usage.total_tokens` 一致性兜底已补齐：
  - 缺失时自动按 `prompt_tokens + completion_tokens` 补齐
  - 当上游总量偏小时自动校正为推导总量
- 新增校正标记：
  - `total_tokens_adjusted=true`
- 新增测试覆盖：
  - 缺失总量补齐
  - 偏小总量校正

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 仅在统一层做最小一致性兜底，不改 provider 侧原始 usage 采集

### 来源：`0030-model-stream-event-schema-v2-usage-consistency-status.md`

# 031 总结：`model_stream_event` v2 用量一致性状态字段补齐

## 已完成

- `usage` 新增统一状态字段：
  - `usage_consistency_status`
- 状态已覆盖核心路径：
  - `derived`：总量自动补齐
  - `adjusted`：总量偏小已校正
  - `ok`：总量与推导一致
  - `source_only`：仅有上游总量
- 标准化测试已补齐状态断言。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 状态字段仅覆盖最小一致性集合，未扩展 provider 级诊断细分

### 来源：`0031-model-stream-event-schema-v2-usage-status-invalid.md`

# 032 总结：`model_stream_event` v2 用量异常状态补齐

## 已完成

- `usage_consistency_status` 新增状态：
  - `invalid`
- 标准化层新增 token 信号识别：
  - 在存在 token 字段但无法形成有效总量一致性路径时，标记为 `invalid`
- 新增测试覆盖：
  - 字符串数值输入可正常归一
  - 非法字符串输入标记 `invalid`

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 仅提供单一异常状态，未细分非法来源类型

### 来源：`0032-model-stream-event-schema-v2-usage-status-provider-coverage.md`

# 033 总结：`model_stream_event` v2 用量状态 provider 覆盖测试

## 已完成

- 新增 `usage_consistency_status` 的 provider 覆盖测试：
  - OpenAI 场景断言 `ok`
  - Anthropic 场景断言 `derived`
  - Codex 场景断言 `invalid`
- 测试集中在 `internal/model/streaming_test.go`，不改协议字段。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 本轮仅补覆盖，不新增字段或状态

### 来源：`0033-model-stream-event-schema-v2-usage-status-source-only-coverage.md`

# 034 总结：`model_stream_event` v2 用量 `source_only` 状态 provider 覆盖测试

## 已完成

- 新增 `source_only` 的 provider 覆盖测试：
  - OpenAI：仅 `total_tokens` 输入断言 `source_only`
  - Anthropic：仅 `total_tokens` 输入断言 `source_only`
  - Codex：仅 `total_tokens` 输入断言 `source_only`
- 测试仅扩展覆盖，不变更标准化逻辑与事件协议。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 本轮仅补覆盖，不新增状态或字段

### 来源：`0034-model-stream-event-schema-v2-usage-status-e2e-provider-streaming.md`

# 035 总结：`model_stream_event` v2 用量状态 provider 流式端到端覆盖

## 已完成

- 新增 provider 流式端到端测试（均通过 `CompleteWithEvents`）：
  - OpenAI：`usage_consistency_status=source_only`
  - Anthropic：`usage_consistency_status=source_only`
  - Codex：`usage_consistency_status=invalid`
- 补齐了“provider 原始流式事件 -> 标准化字典事件”的关键闭环覆盖。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 本轮仅补 E2E 覆盖，不变更事件字段与归一规则

### 来源：`0035-model-stream-event-schema-v2-usage-status-adjusted-e2e.md`

# 036 总结：`model_stream_event` v2 用量 `adjusted` 状态端到端覆盖

## 已完成

- 新增 `adjusted` 的 provider 流式端到端测试：
  - OpenAI：`usage_consistency_status=adjusted`
  - Codex：`usage_consistency_status=adjusted`
- 两条用例均额外断言：
  - `total_tokens_adjusted=true`
  - `total_tokens` 为校正值

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- Anthropic `adjusted` E2E 未补（可在下一轮补齐）

### 来源：`0036-model-stream-event-schema-v2-usage-status-adjusted-e2e-anthropic.md`

# 037 总结：`model_stream_event` v2 用量 `adjusted` 状态 Anthropic 端到端补齐

## 已完成

- 新增 Anthropic `adjusted` 流式端到端测试：
  - 上游偏小 `total_tokens` 经标准化后输出 `usage_consistency_status=adjusted`
  - 同时断言 `total_tokens_adjusted=true` 与校正后的 `total_tokens`
- 至此 `adjusted` 端到端覆盖已具备三 provider：
  - OpenAI
  - Anthropic
  - Codex

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 本轮仅补覆盖，不新增字段

### 来源：`0037-model-stream-event-schema-v2-usage-status-table-driven.md`

# 038 总结：`model_stream_event` v2 用量状态表驱动测试补齐

## 已完成

- 新增 `usage_consistency_status` 表驱动测试，统一覆盖核心状态：
  - `ok`
  - `derived`
  - `source_only`
  - `adjusted`
- 用例内同步断言关键副字段：
  - `total_tokens`
  - `total_tokens_adjusted`（按场景）

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 本轮仅优化测试组织，不新增字段或状态
