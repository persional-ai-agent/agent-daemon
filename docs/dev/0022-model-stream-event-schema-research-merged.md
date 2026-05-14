# 0022 model-stream-event-schema research merged

## 模块

- `model-stream-event-schema`

## 类型

- `research`

## 合并来源

- `0030-model-stream-event-schema-research-merged.md`

## 合并内容

### 来源：`0030-model-stream-event-schema-research-merged.md`

# 0030 model-stream-event-schema research merged

## 模块

- `model-stream-event-schema`

## 类型

- `research`

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

# 020 调研：`model_stream_event` 标准字典（v1）

## 背景

019 已补齐 provider 增量事件透传，但 `event_data` 字段仍存在 provider 差异（`delta`/`partial_json`/`name` 等别名）。

## 缺口

- 前端消费 `model_stream_event` 需要写 provider 分支
- 事件字段缺少统一最小标准

## 本轮目标

定义并落地最小标准字典（v1）：

- `text_delta`：
  - `event_data.text`
- `tool_arguments_delta`：
  - `event_data.tool_name`
  - `event_data.arguments_delta`

并兼容历史别名输入。

## 本轮边界

- 仅覆盖最小事件类型
- 未引入完整的 provider 事件枚举体系

### 来源：`0020-model-stream-event-schema-v2.md`

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

### 来源：`0021-model-stream-event-schema-v2-args-lifecycle.md`

# 022 调研：`model_stream_event` v2 参数生命周期补齐

## 背景

021 已补 `message_*` 与 `tool_call_*` 生命周期，但工具参数仍只有增量事件，缺少参数生命周期起止。

## 缺口

- 客户端难以判定“参数开始接收”和“参数拼装完成”
- `message_done` 缺少统一 `finish_reason`，终止原因不够稳定

## 本轮目标

补齐最小参数生命周期与结束原因字段：

- `tool_args_start`
- `tool_args_delta`
- `tool_args_done`
- `message_done.finish_reason`

并兼容历史别名：

- `tool_arguments_start/delta/done`

## 本轮边界

- 不引入 provider 全量原生事件
- 仍以 Agent 统一事件字典为主

### 来源：`0022-model-stream-event-schema-v2-usage.md`

# 023 调研：`model_stream_event` v2 用量事件补齐

## 背景

022 已完成参数生命周期与 `message_done.finish_reason` 的统一，但客户端仍无法稳定获取跨 provider 的 token 用量信息。

## 缺口

- OpenAI / Anthropic / Codex 都可能返回 usage，但字段命名不一致
- 当前 `model_stream_event` 未定义统一 `usage` 事件，前端/SDK 需要自行做 provider 分支

## 本轮目标

补齐最小可用的统一用量事件：

- `event_type=usage`
- 标准字段：
  - `prompt_tokens`
  - `completion_tokens`
  - `total_tokens`

并兼容常见别名：

- `input_tokens -> prompt_tokens`
- `output_tokens -> completion_tokens`

## 本轮边界

- 仅补“最小统一字段”，不引入 provider 原生完整计费明细
- 不改变 `model_stream_event` 外层结构（仍是 `provider` + `event_type` + `event_data`）

### 来源：`0023-model-stream-event-schema-v2-finish-reason-and-id-aliases.md`

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

### 来源：`0024-model-stream-event-schema-v2-id-source-compat.md`

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

### 来源：`0025-model-stream-event-schema-v2-termination-metadata.md`

# 026 调研：`model_stream_event` v2 终止元数据补齐

## 背景

当前 `message_done` 已统一 `finish_reason`，但客户端在处理“为何中止/由何序列中止”时仍缺统一字段。

## 缺口

- 不同 provider 对终止信息命名不一致：
  - `stop_sequence` / `stop`
  - `incomplete_details.reason` / `reason_detail`
- 现有统一层未稳定提供这两个终止元字段。

## 本轮目标

在不改变事件外层结构前提下，为 `message_done` 增加最小终止元数据：

- `stop_sequence`
- `incomplete_reason`

并在 provider 流式路径尽量透传上游来源字段。

## 本轮边界

- 不扩展新的事件类型
- 仅处理高频终止元字段

### 来源：`0026-model-stream-event-schema-v2-finish-incomplete-consistency.md`

# 027 调研：`model_stream_event` v2 终止原因一致性补齐

## 背景

026 已引入 `stop_sequence` 与 `incomplete_reason`，但 `finish_reason` 与 `incomplete_reason` 之间仍可能出现语义不一致。

## 缺口

- `finish_reason=length` 时，部分 provider 不会显式给出 `incomplete_reason`
- `incomplete_reason` 存在别名值（如 `max_tokens`、`max_output_tokens`），客户端仍需自行归一

## 本轮目标

- 统一 `incomplete_reason` 常见枚举到最小集合
- 当 `finish_reason=length` 且缺少 `incomplete_reason` 时，自动补 `incomplete_reason=length`

## 本轮边界

- 仅处理最小高频枚举，不引入 provider 全量终止诊断信息

### 来源：`0027-model-stream-event-schema-v2-usage-cache-tokens.md`

# 028 调研：`model_stream_event` v2 用量缓存 token 字段补齐

## 背景

当前 `usage` 已统一 `prompt/completion/total_tokens`，但缓存命中相关 token 在不同 provider 命名差异较大。

## 缺口

- Anthropic 常见字段：
  - `cache_creation_input_tokens`
  - `cache_read_input_tokens`
- OpenAI/Codex 常见字段：
  - `prompt_tokens_details.cached_tokens`
  - `input_tokens_details.cached_tokens`
- 客户端难以跨 provider 统一展示 cache token 统计。

## 本轮目标

在 `usage` 事件中增加最小统一字段：

- `prompt_cache_write_tokens`
- `prompt_cache_read_tokens`

## 本轮边界

- 仅补最小缓存 token 字段，不扩展计费明细矩阵

### 来源：`0028-model-stream-event-schema-v2-usage-reasoning-tokens.md`

# 029 调研：`model_stream_event` v2 用量推理 token 字段补齐

## 背景

028 已补齐缓存 token 字段，但推理类 token 统计在不同 provider 的 usage 字段中仍不统一。

## 缺口

- 常见来源字段差异：
  - `completion_tokens_details.reasoning_tokens`
  - `output_tokens_details.reasoning_tokens`
  - 以及部分实现中的平铺别名字段
- 客户端难以跨 provider 一致展示“推理 token”开销。

## 本轮目标

在 `usage` 事件增加最小统一字段：

- `reasoning_tokens`

## 本轮边界

- 仅补推理 token 统一字段，不扩展更细粒度思考链路计费维度

### 来源：`0029-model-stream-event-schema-v2-usage-total-consistency.md`

# 030 调研：`model_stream_event` v2 用量总量一致性补齐

## 背景

`usage` 事件已统一多个 token 字段，但 `total_tokens` 在部分 provider 场景可能缺失，或与 `prompt_tokens + completion_tokens` 不一致。

## 缺口

- `total_tokens` 缺失时，客户端仍需自行回推
- `total_tokens` 偏小时，客户端会出现展示和计费估算不一致

## 本轮目标

- 在统一层提供 `total_tokens` 兜底：
  - 缺失时自动按 `prompt_tokens + completion_tokens` 补齐
  - 当上游 `total_tokens` 小于 `prompt + completion` 时自动校正
- 增加校正标记字段：
  - `total_tokens_adjusted=true`

## 本轮边界

- 不修改 provider 原始 usage 透传逻辑，仅在标准化阶段做一致性兜底

### 来源：`0030-model-stream-event-schema-v2-usage-consistency-status.md`

# 031 调研：`model_stream_event` v2 用量一致性状态字段补齐

## 背景

030 已补齐 `total_tokens` 缺失补齐与偏小校正，但客户端仍需通过多字段组合来判断“当前总量是原值、推导还是校正值”。

## 缺口

- 缺少统一状态字段表达 usage 总量一致性结果
- 客户端需要同时检查 `total_tokens`、`total_tokens_adjusted` 与上下文字段

## 本轮目标

新增统一状态字段：

- `usage_consistency_status`

最小状态集合：

- `ok`：上游总量与推导一致
- `derived`：由 `prompt + completion` 自动补齐
- `adjusted`：上游总量偏小，已校正
- `source_only`：仅有上游总量，无法推导校验

## 本轮边界

- 不改 provider 透传路径，仅在标准化层输出状态字段

### 来源：`0031-model-stream-event-schema-v2-usage-status-invalid.md`

# 032 调研：`model_stream_event` v2 用量异常状态补齐

## 背景

031 已新增 `usage_consistency_status`，但对于上游返回的非数值 token（如字符串脏值）仍缺明确状态表达。

## 缺口

- `usage` 出现 token 字段但无法解析时，客户端无法区分“无数据”与“脏数据”
- 现有状态集合缺少异常输入标识

## 本轮目标

新增并补齐异常状态：

- `usage_consistency_status=invalid`

触发条件：

- 存在 token 相关字段，但无法形成可用一致性路径（无法推导/校正/判定 source_only）

## 本轮边界

- 仅新增最小异常状态，不扩展错误码体系

### 来源：`0032-model-stream-event-schema-v2-usage-status-provider-coverage.md`

# 033 调研：`model_stream_event` v2 用量状态 provider 覆盖测试

## 背景

031/032 已完成 `usage_consistency_status` 字段及 `invalid` 状态，但缺少按 provider 维度的覆盖测试。

## 缺口

- OpenAI/Anthropic/Codex 在使用统一字典时，状态断言主要覆盖通用路径
- 尚未显式验证各 provider 典型输入是否稳定落到预期状态

## 本轮目标

补齐 provider 维度的最小测试覆盖：

- OpenAI：`ok`
- Anthropic：`derived`
- Codex：`invalid`

## 本轮边界

- 仅补测试覆盖，不变更字段设计与事件协议

### 来源：`0033-model-stream-event-schema-v2-usage-status-source-only-coverage.md`

# 034 调研：`model_stream_event` v2 用量 `source_only` 状态 provider 覆盖测试

## 背景

033 已覆盖 `ok/derived/invalid` 的 provider 维度测试，但 `source_only` 仍缺同层级覆盖。

## 缺口

- OpenAI/Anthropic/Codex 在仅提供 `total_tokens` 时，尚未显式验证统一输出 `usage_consistency_status=source_only`

## 本轮目标

补齐 `source_only` 的 provider 覆盖测试：

- OpenAI
- Anthropic
- Codex

## 本轮边界

- 仅补测试，不修改协议与标准化逻辑

### 来源：`0034-model-stream-event-schema-v2-usage-status-e2e-provider-streaming.md`

# 035 调研：`model_stream_event` v2 用量状态 provider 流式端到端覆盖

## 背景

034 已补 `source_only` 的 provider 维度标准化测试，但仍缺“provider 流式输出 -> `CompleteWithEvents` 标准化”的端到端覆盖。

## 缺口

- 现有 provider stream 测试多数直接断言原始事件
- 未显式覆盖 `CompleteWithEvents` 对 `usage_consistency_status` 的跨 provider 端到端归一结果

## 本轮目标

补齐 provider 流式端到端用例：

- OpenAI 流式 usage -> `source_only`
- Anthropic 流式 usage -> `source_only`
- Codex completed envelope usage -> `invalid`

## 本轮边界

- 仅补测试覆盖，不修改标准化规则

### 来源：`0035-model-stream-event-schema-v2-usage-status-adjusted-e2e.md`

# 036 调研：`model_stream_event` v2 用量 `adjusted` 状态端到端覆盖

## 背景

035 已覆盖 `source_only/invalid` 的 provider 流式端到端路径，但 `adjusted` 仍缺同层级验证。

## 缺口

- 当上游 `total_tokens` 偏小且需要标准化层校正时，尚未在 provider 流式路径做 E2E 断言
- 客户端最依赖该场景来识别“校正后总量”

## 本轮目标

补齐 `adjusted` 端到端覆盖：

- OpenAI 流式 usage（`total_tokens` 偏小） -> `adjusted`
- Codex completed envelope usage（`total_tokens` 偏小） -> `adjusted`

并同时断言：

- `total_tokens_adjusted=true`
- `total_tokens` 为校正后的值

## 本轮边界

- 仅补测试，不改字段规则

### 来源：`0036-model-stream-event-schema-v2-usage-status-adjusted-e2e-anthropic.md`

# 037 调研：`model_stream_event` v2 用量 `adjusted` 状态 Anthropic 端到端补齐

## 背景

036 已补 OpenAI/Codex 的 `adjusted` 端到端覆盖，Anthropic 仍是缺口。

## 缺口

- 三 provider 中仅 Anthropic 未验证：
  - 流式 usage 上游给出偏小 `total_tokens`
  - 经 `CompleteWithEvents` 归一后输出 `usage_consistency_status=adjusted`

## 本轮目标

补齐 Anthropic 的 `adjusted` 端到端测试，并断言：

- `usage_consistency_status=adjusted`
- `total_tokens_adjusted=true`
- `total_tokens` 为校正值

## 本轮边界

- 仅补测试覆盖，不修改标准化规则

### 来源：`0037-model-stream-event-schema-v2-usage-status-table-driven.md`

# 038 调研：`model_stream_event` v2 用量状态表驱动测试补齐

## 背景

当前 `usage_consistency_status` 的断言已较多，分散在多个独立测试中，新增状态或规则时维护成本上升。

## 缺口

- 缺少统一表驱动用例，难以一眼覆盖核心状态矩阵
- 新增状态时容易漏改多个测试函数

## 本轮目标

新增一个表驱动测试，统一覆盖核心状态：

- `ok`
- `derived`
- `source_only`
- `adjusted`

并同时校验关键副字段（如 `total_tokens_adjusted`、`total_tokens`）。

## 本轮边界

- 仅补测试组织方式，不改业务规则
