# 0022 model-stream-event-schema plan merged

## 模块

- `model-stream-event-schema`

## 类型

- `plan`

## 合并来源

- `0030-model-stream-event-schema-plan-merged.md`

## 合并内容

### 来源：`0030-model-stream-event-schema-plan-merged.md`

# 0030 model-stream-event-schema plan merged

## 模块

- `model-stream-event-schema`

## 类型

- `plan`

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

# 020 计划：`model_stream_event` 标准字典（v1）

## 目标

在不破坏现有 `model_stream_event` 事件外层结构的前提下，统一 `event_data` 的最小字段集。

## 实施步骤

1. 在 `internal/model` 增加事件标准化函数。
2. 在模型事件入口统一应用标准化（而非各 provider 各自分散处理）。
3. 增加测试，覆盖别名字段到标准字段的映射。
4. 增加 `agent` 层事件测试，验证透传后的标准字段可用。
5. 全量回归 `go test ./...`。

## 验证标准

- `text_delta` 始终包含 `event_data.text`
- `tool_arguments_delta` 始终包含 `event_data.tool_name` 与 `event_data.arguments_delta`
- 全量测试通过

### 来源：`0020-model-stream-event-schema-v2.md`

# 021 计划：`model_stream_event` 标准字典（v2 最小扩展）

## 目标

补齐 `model_stream_event` 的最小生命周期事件，提升客户端跨 provider 的一致消费能力。

## 实施步骤

1. 在 `normalizeStreamEvent` 中扩展 v2 事件标准化映射。
2. 在 OpenAI/Anthropic/Codex 流式路径补发：
   - `message_start` / `message_done`
   - `tool_call_start` / `tool_call_done`
3. 保持 `Engine` 透传结构不变（`provider`、`event_type`、`event_data`）。
4. 补模型层标准化测试与 agent 层透传测试。
5. 执行 `go test ./...` 回归。

## 验证标准

- 三 provider 流式路径具备最小生命周期事件
- `event_data` 字段在 v2 事件上保持统一语义
- 全量测试通过

### 来源：`0021-model-stream-event-schema-v2-args-lifecycle.md`

# 022 计划：`model_stream_event` v2 参数生命周期补齐

## 目标

将工具参数流式事件从“仅增量”扩展为“开始-增量-结束”三段式，并统一 `message_done.finish_reason`。

## 实施步骤

1. 扩展 `normalizeStreamEvent`：
   - 别名映射 `tool_arguments_* -> tool_args_*`
   - 标准化 `tool_args_start/delta/done` 字段
   - `message_done` 默认补 `finish_reason=stop`
2. OpenAI/Anthropic/Codex 流式路径补发 `tool_args_start/delta/done`。
3. `message_done` 统一补 `finish_reason`（有工具调用时 `tool_calls`，否则 `stop`）。
4. 更新模型层与 agent 层测试。
5. 全量回归 `go test ./...`。

## 验证标准

- 三 provider 在流式场景均包含参数生命周期事件
- `message_done.finish_reason` 始终可用
- 全量测试通过

### 来源：`0022-model-stream-event-schema-v2-usage.md`

# 023 计划：`model_stream_event` v2 用量事件补齐

## 目标

为 `model_stream_event` 增加跨 provider 可统一消费的 `usage` 事件，并规范最小字段集。

## 实施步骤

1. 扩展 `normalizeStreamEvent`：
   - 新增 `usage` 字段标准化
   - 对齐 `input_tokens/output_tokens` 到 `prompt_tokens/completion_tokens`
   - 缺少 `total_tokens` 时用前两者自动补齐
2. OpenAI 流式路径解析并发出 `usage`。
3. Anthropic 流式路径解析 `message_delta.usage` 并发出 `usage`。
4. Codex 流式路径解析 completed envelope 的 `response.usage` 并发出 `usage`。
5. 更新模型层与 agent 层测试。
6. 全量回归 `go test ./...`。

## 验证标准

- `model_stream_event` 可收到 `event_type=usage`
- `event_data` 至少可稳定提供 `prompt_tokens/completion_tokens/total_tokens`
- 全量测试通过

### 来源：`0023-model-stream-event-schema-v2-finish-reason-and-id-aliases.md`

# 024 计划：`model_stream_event` v2 结束原因与 ID 别名归一

## 目标

继续增强 `model_stream_event` 字典一致性，减少客户端 provider 分支判断。

## 实施步骤

1. 在 `normalizeStreamEvent` 中新增 `finish_reason` 归一：
   - `end_turn -> stop`
   - `tool_use -> tool_calls`
   - `max_tokens/max_output_tokens -> length`
2. 在 `tool_call_*` 与 `tool_args_*` 标准化中补齐 `tool_use_id -> tool_call_id`。
3. 更新模型层测试，覆盖结束原因归一与 `tool_use_id` 别名。
4. 更新 agent 层透传测试，验证 `message_done.reason=end_turn` 时输出标准 `finish_reason=stop`。
5. 执行回归测试并同步文档。

## 验证标准

- `message_done.finish_reason` 可稳定落在最小标准集合
- `tool_use_id` 事件可被统一消费为 `tool_call_id`
- 测试通过

### 来源：`0024-model-stream-event-schema-v2-id-source-compat.md`

# 025 计划：`model_stream_event` v2 消息/工具 ID 来源兼容补齐

## 目标

提升 `message_id` 与 `tool_call_id` 的跨 provider 稳定性，进一步减少客户端分支判断。

## 实施步骤

1. 扩展 `normalizeStreamEvent`：
   - `message_id` 兼容 `response_id`、`message.id`
   - `tool_call_id` 兼容 `item_id`、`output_item_id`
2. Anthropic 流式路径补充：
   - 解析并透传 `message_start.message.id`
   - 透传 `message_delta.stop_reason` 供统一层归一
3. Codex completed envelope 补充：
   - 透传 `response.id` 到 `response_id`
4. 更新模型层测试覆盖新增别名来源。
5. 执行回归与文档同步。

## 验证标准

- `model_stream_event` 的 `message_*` 事件可稳定提取 `message_id`
- `tool_*` 与 `tool_args_*` 事件可稳定提取 `tool_call_id`
- 全量测试通过

### 来源：`0025-model-stream-event-schema-v2-termination-metadata.md`

# 026 计划：`model_stream_event` v2 终止元数据补齐

## 目标

统一 `message_done` 的终止元数据表达，降低前端/SDK 对 provider 差异的适配成本。

## 实施步骤

1. 扩展 `normalizeStreamEvent(message_done)`：
   - `stop -> stop_sequence`
   - `incomplete_details.reason` / `reason_detail -> incomplete_reason`
2. Anthropic 流式路径：
   - 透传 `message_delta.stop_sequence`
3. Codex completed envelope：
   - 透传 `response.incomplete_details.reason`
4. 更新模型层测试：
   - 标准化测试覆盖 `stop_sequence` 与 `incomplete_reason`
   - provider 流式测试覆盖透传字段
5. 回归 `go test ./...` 并同步文档。

## 验证标准

- `message_done` 可稳定输出 `stop_sequence`（若上游存在）
- `message_done` 可稳定输出 `incomplete_reason`（若上游存在）
- 回归测试通过

### 来源：`0026-model-stream-event-schema-v2-finish-incomplete-consistency.md`

# 027 计划：`model_stream_event` v2 终止原因一致性补齐

## 目标

增强 `message_done.finish_reason` 与 `message_done.incomplete_reason` 的一致性，减少客户端推断逻辑。

## 实施步骤

1. 扩展 `normalizeStreamEvent(message_done)`：
   - `incomplete_reason` 归一（`max_tokens/max_output_tokens -> length`）
   - 当 `finish_reason=length` 且 `incomplete_reason` 缺失时，自动补 `incomplete_reason=length`
2. 更新标准化测试：
   - 覆盖自动补全场景
   - 覆盖别名归一场景
3. 回归测试并同步文档。

## 验证标准

- `finish_reason=length` 时，`incomplete_reason` 必定存在且为 `length`
- 历史别名可统一收敛
- 测试通过

### 来源：`0027-model-stream-event-schema-v2-usage-cache-tokens.md`

# 028 计划：`model_stream_event` v2 用量缓存 token 字段补齐

## 目标

扩展 `usage` 统一字典，使缓存 token 可跨 provider 统一消费。

## 实施步骤

1. 扩展 `normalizeStreamEvent(usage)`：
   - `cache_creation_input_tokens -> prompt_cache_write_tokens`
   - `cache_read_input_tokens -> prompt_cache_read_tokens`
   - `prompt_tokens_details.cached_tokens -> prompt_cache_read_tokens`
   - `input_tokens_details.cached_tokens -> prompt_cache_read_tokens`
2. 增加标准化测试覆盖：
   - 直接字段映射
   - 嵌套字段映射
3. 回归测试并同步文档。

## 验证标准

- `usage` 事件可稳定输出 `prompt_cache_write_tokens/prompt_cache_read_tokens`
- 测试通过

### 来源：`0028-model-stream-event-schema-v2-usage-reasoning-tokens.md`

# 029 计划：`model_stream_event` v2 用量推理 token 字段补齐

## 目标

扩展 `usage` 字典，统一推理 token 的读取口径。

## 实施步骤

1. 扩展 `normalizeStreamEvent(usage)`：
   - `completion_tokens_details.reasoning_tokens -> reasoning_tokens`
   - `output_tokens_details.reasoning_tokens -> reasoning_tokens`
   - `reasoning_tokens_count -> reasoning_tokens`
2. 增加标准化测试覆盖：
   - OpenAI 风格嵌套字段
   - Codex 风格嵌套字段
3. 执行回归并同步文档。

## 验证标准

- `usage` 事件可稳定输出 `reasoning_tokens`
- 测试通过

### 来源：`0029-model-stream-event-schema-v2-usage-total-consistency.md`

# 030 计划：`model_stream_event` v2 用量总量一致性补齐

## 目标

增强 `usage.total_tokens` 的可用性与一致性，减少客户端重复兜底逻辑。

## 实施步骤

1. 扩展 `normalizeStreamEvent(usage)`：
   - 当 `total_tokens` 缺失且 `prompt_tokens/completion_tokens` 可用时自动补齐
   - 当 `total_tokens < prompt_tokens + completion_tokens` 时自动校正
   - 输出 `total_tokens_adjusted=true` 标记校正行为
2. 增加标准化测试：
   - 缺失总量自动补齐
   - 总量偏小自动校正
3. 回归 `go test ./...` 并同步文档。

## 验证标准

- `usage.total_tokens` 在主路径可稳定使用
- 校正场景可通过 `total_tokens_adjusted` 识别
- 测试通过

### 来源：`0030-model-stream-event-schema-v2-usage-consistency-status.md`

# 031 计划：`model_stream_event` v2 用量一致性状态字段补齐

## 目标

为 `usage` 增加统一一致性状态，降低客户端解析复杂度。

## 实施步骤

1. 扩展 `normalizeStreamEvent(usage)`：
   - 新增 `usage_consistency_status`
   - 状态规则：
     - `derived`
     - `adjusted`
     - `ok`
     - `source_only`
2. 更新标准化测试：
   - 补齐 `derived/adjusted/ok` 场景断言
3. 回归测试并同步文档。

## 验证标准

- `usage` 事件可稳定输出 `usage_consistency_status`
- 状态与 `total_tokens` 处理路径一致
- 测试通过

### 来源：`0031-model-stream-event-schema-v2-usage-status-invalid.md`

# 032 计划：`model_stream_event` v2 用量异常状态补齐

## 目标

为 `usage_consistency_status` 增加 `invalid`，使客户端可直接识别脏数据场景。

## 实施步骤

1. 扩展 `normalizeStreamEvent(usage)`：
   - 识别 token 信号字段是否存在
   - 在未命中 `ok/derived/adjusted/source_only` 时，输出 `invalid`
2. 增加标准化测试：
   - 字符串数值可解析场景（应正常归一）
   - 非法字符串场景（应标记 `invalid`）
3. 回归 `go test ./...` 并同步文档。

## 验证标准

- 异常 token 输入时，`usage_consistency_status=invalid`
- 正常字符串数值输入仍可按既有规则归一
- 测试通过

### 来源：`0032-model-stream-event-schema-v2-usage-status-provider-coverage.md`

# 033 计划：`model_stream_event` v2 用量状态 provider 覆盖测试

## 目标

增强 `usage_consistency_status` 的回归稳定性，确保不同 provider 的典型输入都可落入预期状态。

## 实施步骤

1. 在 `internal/model/streaming_test.go` 增加 provider 维度测试：
   - OpenAI 一致总量 -> `ok`
   - Anthropic 输入/输出推导 -> `derived`
   - Codex 非法 token -> `invalid`
2. 执行 `go test ./internal/model` 与全量回归。
3. 更新 dev 文档索引与总结。

## 验证标准

- provider 维度断言可稳定通过
- 全量测试通过

### 来源：`0033-model-stream-event-schema-v2-usage-status-source-only-coverage.md`

# 034 计划：`model_stream_event` v2 用量 `source_only` 状态 provider 覆盖测试

## 目标

增强 `usage_consistency_status` 的测试完整性，补齐 `source_only` 在三 provider 的最小覆盖。

## 实施步骤

1. 在 `internal/model/streaming_test.go` 新增三条测试：
   - OpenAI：仅 `total_tokens` -> `source_only`
   - Anthropic：仅 `total_tokens` -> `source_only`
   - Codex：仅 `total_tokens` -> `source_only`
2. 执行 `go test ./internal/model`。
3. 执行 `go test ./...` 全量回归。
4. 同步 `docs/dev/README.md` 与 summary 文档。

## 验证标准

- 三 provider 场景均稳定断言 `source_only`
- 全量测试通过

### 来源：`0034-model-stream-event-schema-v2-usage-status-e2e-provider-streaming.md`

# 035 计划：`model_stream_event` v2 用量状态 provider 流式端到端覆盖

## 目标

验证 `usage_consistency_status` 在真实 provider 流式路径下经过 `CompleteWithEvents` 后仍稳定可用。

## 实施步骤

1. 在 provider 测试中新增 E2E 用例：
   - `openai_stream_test.go`：断言 `source_only`
   - `anthropic_stream_test.go`：断言 `source_only`
   - `codex_stream_test.go`：断言 `invalid`
2. 用例统一调用 `CompleteWithEvents`，通过 sink 读取标准化事件。
3. 执行 `go test ./internal/model` 与 `go test ./...`。
4. 更新 `docs/dev/README.md` 与总结文档。

## 验证标准

- 三个 provider 的端到端断言均通过
- 全量测试通过

### 来源：`0035-model-stream-event-schema-v2-usage-status-adjusted-e2e.md`

# 036 计划：`model_stream_event` v2 用量 `adjusted` 状态端到端覆盖

## 目标

验证 `adjusted` 状态在 provider 流式路径下经过 `CompleteWithEvents` 后可稳定输出。

## 实施步骤

1. 在 `openai_stream_test.go` 新增 E2E 用例：
   - 输入 `prompt/completion/total` 且 `total` 偏小
   - 断言 `usage_consistency_status=adjusted`
   - 断言 `total_tokens_adjusted=true` 和校正后的 `total_tokens`
2. 在 `codex_stream_test.go` 新增同类 E2E 用例。
3. 执行 `go test ./internal/model` 与 `go test ./...`。
4. 更新 dev 文档索引与总结。

## 验证标准

- OpenAI/Codex 两条 E2E 均通过
- 校正字段断言稳定

### 来源：`0036-model-stream-event-schema-v2-usage-status-adjusted-e2e-anthropic.md`

# 037 计划：`model_stream_event` v2 用量 `adjusted` 状态 Anthropic 端到端补齐

## 目标

完成 `adjusted` 状态在 OpenAI/Anthropic/Codex 三 provider 的端到端测试闭环。

## 实施步骤

1. 在 `anthropic_stream_test.go` 新增 E2E 用例：
   - `message_delta.usage` 提供 `input/output/total` 且 `total` 偏小
   - 调用 `CompleteWithEvents`
   - 断言 `usage_consistency_status=adjusted`
   - 断言 `total_tokens_adjusted=true` 与校正后的 `total_tokens`
2. 执行 `go test ./internal/model` 与 `go test ./...`。
3. 更新 `docs/dev/README.md` 与 summary。

## 验证标准

- Anthropic `adjusted` E2E 通过
- 三 provider `adjusted` 端到端覆盖齐全

### 来源：`0037-model-stream-event-schema-v2-usage-status-table-driven.md`

# 038 计划：`model_stream_event` v2 用量状态表驱动测试补齐

## 目标

提升 `usage_consistency_status` 测试的可维护性与可读性。

## 实施步骤

1. 在 `internal/model/streaming_test.go` 增加表驱动测试：
   - 覆盖 `ok/derived/source_only/adjusted`
2. 在每个 case 中统一断言：
   - `usage_consistency_status`
   - `total_tokens`
   - `total_tokens_adjusted`（按需）
3. 执行 `go test ./internal/model` 与 `go test ./...`。
4. 更新 `docs/dev/README.md` 与 summary。

## 验证标准

- 表驱动用例覆盖核心状态集合
- 全量测试通过
