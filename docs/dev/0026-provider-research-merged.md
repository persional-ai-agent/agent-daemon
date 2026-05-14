# 0026 provider research merged

## 模块

- `provider`

## 类型

- `research`

## 合并来源

- `0035-provider-research-merged.md`

## 合并内容

### 来源：`0035-provider-research-merged.md`

# 0035 provider research merged

## 模块

- `provider`

## 类型

- `research`

## 合并来源

- `0004-provider-modes.md`
- `0014-provider-fallback-minimal.md`
- `0015-provider-streaming-openai-minimal.md`
- `0016-provider-streaming-anthropic-minimal.md`
- `0017-provider-streaming-codex-minimal.md`
- `0018-provider-stream-events-passthrough.md`
- `0038-provider-race-circuit.md`
- `0039-provider-event-coverage.md`

## 合并内容

### 来源：`0004-provider-modes.md`

# 005 调研：Provider 多模式补齐

## 背景

当前 Go 版仅支持 OpenAI 兼容 `chat/completions`，与 Hermes 的多 provider 能力存在差距。

在不引入过度复杂抽象的前提下，优先补齐第二种主流模式：Anthropic Messages API。

## 差异点

- 缺少 provider 选择机制
- 缺少 Anthropic 消息协议转换
- 缺少跨 provider 的 tool call 结构映射

## 方案

采用“统一 `model.Client` 接口 + provider 实现分文件”的方式：

- 保持 `Engine` 与 `Agent Loop` 不变
- 新增 `AnthropicClient` 实现同一 `ChatCompletion` 接口
- 启动时按 `AGENT_MODEL_PROVIDER` 选择 client

## 映射策略

- 内部仍统一使用 OpenAI 风格 `core.Message`
- 调 Anthropic 前做协议转换：
  - `system` 提取为独立 `system` 字段
  - `tool` 消息映射为 `tool_result` block
  - `assistant.tool_calls` 映射为 `tool_use` block
- 从 Anthropic 响应反解回 `core.Message` + `ToolCalls`

## 结论

该方案可在不改核心 loop 的前提下，把 provider 能力从单模式扩展到双模式，后续再继续补 Codex/Anthropic 高级特性与更多 provider。

### 来源：`0014-provider-fallback-minimal.md`

# 015 调研：Provider 故障切换最小补齐

## 背景

当前模型调用已支持 OpenAI / Anthropic / Codex 三种 provider，但一次运行仅绑定单 provider。  
当主 provider 出现限流或暂时故障时，会直接失败。

## 缺口

- 无主备 provider 自动切换
- 无可配置的故障降级路径

## 本轮目标

补齐最小故障切换能力：

- 新增 `FallbackClient` 包装器
- 主 provider 失败且命中可重试错误（如 429/5xx/timeout）时，自动回退备用 provider
- 通过配置启用：`AGENT_MODEL_FALLBACK_PROVIDER`

## 本轮边界

- 不做并行竞速请求
- 不做多级级联 fallback
- 不做 provider 健康探针与熔断器

### 来源：`0015-provider-streaming-openai-minimal.md`

# 016 调研：Provider 流式统一（OpenAI 最小落地）

## 背景

015 已补齐 provider 故障切换，但模型调用仍以非流式为主。  
当前系统层面已经有 SSE 输出能力，模型层需要先具备统一的流式聚合能力作为基础。

## 缺口

- OpenAI 客户端未使用 `stream=true`
- 模型层缺少“流式片段 -> 最终 `core.Message`”聚合逻辑

## 本轮目标

以最小改动先落地 OpenAI 流式聚合：

- 增加配置开关 `AGENT_MODEL_USE_STREAMING`
- OpenAI 在开关开启时走 `chat/completions` 流式模式
- 解析 SSE `data:` 事件并聚合：
  - assistant 文本增量
  - tool call 增量参数拼接
- 对外仍返回原有 `ChatCompletion` 的 `core.Message`

## 本轮边界

- Anthropic/Codex 暂未接入流式模式
- 暂不透传 provider 级增量事件到 Agent 事件流

### 来源：`0016-provider-streaming-anthropic-minimal.md`

# 017 调研：Provider 流式统一（Anthropic 最小落地）

## 背景

016 已完成 OpenAI 流式聚合，但 Anthropic 仍为非流式调用。  
要推进 provider 流式统一，需要继续补齐 Anthropic。

## 缺口

- Anthropic 客户端未启用 `stream=true`
- 缺少 Anthropic SSE 事件聚合逻辑

## 本轮目标

以最小改动补齐 Anthropic 流式聚合：

- 增加 `UseStreaming` 开关
- 开关开启时发送 `stream=true`
- 解析并聚合核心流式事件：
  - `content_block_start`
  - `content_block_delta`（`text_delta` / `input_json_delta`）
- 聚合为标准 `core.Message` 返回

## 本轮边界

- Codex 流式暂未补齐
- 暂不透传 provider 级增量事件到 Agent 事件流

### 来源：`0017-provider-streaming-codex-minimal.md`

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

### 来源：`0018-provider-stream-events-passthrough.md`

# 019 调研：Provider 增量事件透传（最小版）

## 背景

018 已补齐三种 provider 的流式聚合，但聚合过程中的增量事件仍停留在模型层，Agent 事件流无法感知。

## 缺口

- `Engine` 无法接收 provider 的流式增量事件
- SSE 客户端看不到模型生成中的中间进度

## 本轮目标

在不破坏现有 `model.Client` 基础接口的前提下，补齐最小透传：

- 增加可选模型事件接口（扩展接口，不替换原接口）
- OpenAI / Anthropic / Codex 在流式解析中上报增量事件
- `Engine` 统一转发为 `model_stream_event`

## 本轮边界

- 仅透传最小事件类型（文本增量、工具参数增量）
- 不做 provider-specific 的完整事件字典标准化

### 来源：`0038-provider-race-circuit.md`

# 039 调研：Provider 并行竞速与熔断

## 背景

当前 `FallbackClient` 采用串行降级策略：主 provider 失败后才尝试备用 provider。这在高可用场景下存在两个问题：

1. **延迟叠加**：主 provider 超时后才触发 fallback，总延迟 = 主超时 + 备用响应
2. **无状态感知**：连续失败后仍会尝试已故障的 provider，浪费请求配额

## 缺口分析

### 当前能力

- 串行 fallback：主失败 → 备用
- 基于错误码/超时判断是否 fallback
- 无 provider 健康状态记忆

### 缺失能力

- 并行竞速：同时向多个 provider 发请求，取最快响应
- 熔断器：连续失败后临时隔离 provider，避免无效请求
- 半开探测：熔断后自动试探性恢复
- 成本感知：优先使用低成本 provider，高成本仅作为竞速备选

## 方案对比

### 方案 A：纯并行竞速

**做法**：同时向 N 个 provider 发请求，`select` 取第一个成功响应，取消其余。

**优点**：
- 延迟最低（取最快者）
- 实现简单

**缺点**：
- 每次请求都消耗多个 provider 配额
- 成本高（N 倍 token 消耗）
- 无故障隔离，某个 provider 持续故障仍会被调用

**适用场景**：对延迟极度敏感、成本不敏感的场景

### 方案 B：熔断器 + 串行 fallback（当前方案的增强）

**做法**：在现有 `FallbackClient` 基础上增加熔断器状态机：

- **Closed**（正常）：正常调用，失败计数增加
- **Open**（熔断）：连续失败达到阈值后，跳过该 provider，直接 fallback
- **Half-Open**（半开）：熔断超时后，允许一次试探请求，成功则恢复 Closed，失败则继续 Open

**优点**：
- 成本可控（不并行发请求）
- 自动隔离故障 provider
- 实现复杂度适中

**缺点**：
- 延迟与当前 fallback 相同（串行）
- 需要维护状态（线程安全）

**适用场景**：成本敏感、可接受串行延迟的场景

### 方案 C：熔断器 + 可选并行竞速（推荐）

**做法**：融合方案 A 和 B：

- 默认使用熔断器 + 串行 fallback
- 可选开启并行竞速模式（通过配置 `AGENT_MODEL_RACE_ENABLED=true`）
- 并行竞速时，仅对未熔断的 provider 发请求
- 竞速失败会增加该 provider 的失败计数

**优点**：
- 兼顾成本与延迟（用户可选择）
- 故障隔离（熔断器保护）
- 灵活配置

**缺点**：
- 实现复杂度较高
- 需要管理并发状态

**适用场景**：通用场景，用户可根据需求选择模式

## 推荐方案

采用 **方案 C**，理由：

1. 向后兼容：默认行为与当前 fallback 一致
2. 渐进增强：用户可按需开启并行竞速
3. 故障隔离：熔断器保护所有模式
4. 可观测性：暴露 provider 健康状态，便于调试

## 核心设计

### 熔断器状态机

```
Closed ──(连续失败≥阈值)──> Open
  ↑                           │
  │                      (熔断超时)
  │                           ↓
  │                       Half-Open
  │                           │
  └──(试探成功)───────────────┘
  │
  └──(试探失败)──> Open
```

### 配置项

- `AGENT_MODEL_RACE_ENABLED`：是否开启并行竞速（默认 `false`）
- `AGENT_MODEL_CIRCUIT_FAILURE_THRESHOLD`：连续失败阈值（默认 `3`）
- `AGENT_MODEL_CIRCUIT_RECOVERY_TIMEOUT_SECONDS`：熔断恢复超时（默认 `60`）
- `AGENT_MODEL_CIRCUIT_HALF_OPEN_MAX_REQUESTS`：半开状态最大试探请求数（默认 `1`）

### 数据结构

```go
type CircuitState int

const (
    CircuitClosed CircuitState = iota
    CircuitOpen
    CircuitHalfOpen
)

type ProviderCircuit struct {
    mu                sync.RWMutex
    state             CircuitState
    failureCount      int
    successCount      int
    lastFailureTime   time.Time
    lastStateChange   time.Time
    threshold         int
    recoveryTimeout   time.Duration
    halfOpenMaxReqs   int
    halfOpenReqs      int
}

type RaceClient struct {
    Primary      Client
    PrimaryName  string
    Fallback     Client
    FallbackName string
    PrimaryCircuit *ProviderCircuit
    FallbackCircuit *ProviderCircuit
    RaceEnabled  bool
}
```

### 执行流程

**串行模式（默认）**：
1. 检查主 provider 熔断器状态
2. 若 Open，直接走 fallback
3. 若 Closed/Half-Open，调用主 provider
4. 成功 → 重置失败计数；失败 → 增加失败计数，可能触发熔断
5. 若主失败且可 fallback，重复 2-4 走备用 provider

**并行竞速模式**：
1. 检查所有 provider 熔断器状态
2. 过滤掉 Open 状态的 provider
3. 向剩余 provider 并发发请求
4. `select` 取第一个成功响应
5. 成功者重置失败计数，失败者增加失败计数
6. 取消其余进行中的请求

## 结论

该方案可在保持向后兼容的前提下，为 Provider 层增加故障隔离与延迟优化能力，属于 L3 级别需求（跨模块、影响关键链路）。

### 来源：`0039-provider-event-coverage.md`

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
