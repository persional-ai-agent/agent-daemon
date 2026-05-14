# 0026 provider plan merged

## 模块

- `provider`

## 类型

- `plan`

## 合并来源

- `0035-provider-plan-merged.md`

## 合并内容

### 来源：`0035-provider-plan-merged.md`

# 0035 provider plan merged

## 模块

- `provider`

## 类型

- `plan`

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

# 005 计划：Provider 多模式补齐

## 目标

在当前 OpenAI 模式基础上，新增 Anthropic 模式并实现运行时可配置切换。

## 实施步骤

1. 新增 Anthropic client
验证：可调用 `/messages` 并解析文本与 `tool_use`

2. 实现协议转换
验证：`core.Message` 可稳定映射到 Anthropic 请求格式并反解回来

3. 增加 provider 选择配置
验证：`AGENT_MODEL_PROVIDER=anthropic` 时启动 Anthropic client

4. 增加测试并回归
验证：新增 `internal/model` 单元测试通过，`go test ./...` 通过

### 来源：`0014-provider-fallback-minimal.md`

# 015 计划：Provider 故障切换最小补齐

## 目标

在不改动 Agent Loop 的前提下，新增主备 provider 自动切换能力，提升可用性。

## 实施步骤

1. 在 `internal/model` 新增 `FallbackClient`：
   - 主调用成功直接返回
   - 主调用失败且为可重试错误时调用备用 provider
2. 新增回退判定规则：
   - 状态码：`408/429/500/502/503/504`
   - 网络超时与常见连接错误
3. 新增配置项：`AGENT_MODEL_FALLBACK_PROVIDER`
4. 在 `cmd/agentd` 的模型装配中启用 fallback 包装。
5. 新增模型层与装配层测试。
6. 运行 `go test ./...` 回归。

## 验证标准

- 主 provider 成功时不触发 fallback
- 可重试错误时 fallback 生效
- 非可重试错误时直接返回主错误
- 全量测试通过

### 来源：`0015-provider-streaming-openai-minimal.md`

# 016 计划：Provider 流式统一（OpenAI 最小落地）

## 目标

在不改动 Agent Loop 协议的前提下，为 OpenAI 客户端补齐流式聚合能力。

## 实施步骤

1. 在 `OpenAIClient` 增加 `UseStreaming` 开关。
2. 在 `ChatCompletion` 中按开关选择：
   - 非流式：保持原逻辑
   - 流式：`stream=true` 调用并解析 SSE
3. 实现流式聚合：
   - 文本 `delta.content` 追加
   - `delta.tool_calls` 按 index 聚合函数名与参数
4. 新增配置项：`AGENT_MODEL_USE_STREAMING`
5. 在启动装配中把配置传入 OpenAI 客户端。
6. 补测试并全量回归。

## 验证标准

- OpenAI 流式文本与工具调用可正确聚合
- 默认行为不变（不开开关仍走非流式）
- `go test ./...` 通过

### 来源：`0016-provider-streaming-anthropic-minimal.md`

# 017 计划：Provider 流式统一（Anthropic 最小落地）

## 目标

在不改动 Agent Loop 与 `model.Client` 接口的前提下，补齐 Anthropic 流式聚合能力。

## 实施步骤

1. 在 `AnthropicClient` 增加 `UseStreaming` 开关。
2. 在 `ChatCompletion` 中按开关切换到流式分支。
3. 在流式分支中：
   - `stream=true` 发起请求
   - 解析 `text/event-stream` 的 `data:` 事件
   - 聚合文本和 tool_use 参数增量
4. 在启动装配中复用 `AGENT_MODEL_USE_STREAMING` 开关。
5. 新增 Anthropic 流式文本与工具调用测试。
6. 执行 `go test ./...` 回归。

## 验证标准

- Anthropic 流式文本可正确聚合
- Anthropic 流式 tool_use 参数可正确拼接
- 全量测试通过

### 来源：`0017-provider-streaming-codex-minimal.md`

# 018 计划：Provider 流式统一（Codex 最小落地）

## 目标

在保持现有 `model.Client` 接口不变的前提下，补齐 Codex 流式聚合能力。

## 实施步骤

1. 在 `CodexClient` 增加 `UseStreaming` 开关。
2. `ChatCompletion` 按开关切换流式分支。
3. 流式分支中：
   - `stream=true` 发起请求
   - 解析 SSE `data:` 事件
   - 聚合文本与函数调用参数增量
4. 在启动装配中传递 `AGENT_MODEL_USE_STREAMING` 到 Codex 客户端。
5. 新增 Codex 流式文本/工具调用测试。
6. 执行 `go test ./...` 回归。

## 验证标准

- Codex 流式文本可正确聚合
- Codex 流式函数调用参数可正确拼接
- 全量测试通过

### 来源：`0018-provider-stream-events-passthrough.md`

# 019 计划：Provider 增量事件透传（最小版）

## 目标

打通“模型流式增量 -> Agent 事件流 -> SSE”的最小链路。

## 实施步骤

1. 在 `internal/model` 新增可选事件扩展接口与通用调用 helper。
2. 在 OpenAI/Anthropic/Codex 流式解析中上报增量事件。
3. 在 `FallbackClient` 中透传事件，保证主备切换不丢事件。
4. 在 `Engine.callWithRetry` 中消费模型事件并发出 `model_stream_event`。
5. 增加 `agent` 层测试，验证事件透传。
6. 全量回归 `go test ./...`。

## 验证标准

- 启用流式时可看到 `model_stream_event`
- fallback 场景下事件链路不中断
- 不启用流式时行为不回退

### 来源：`0038-provider-race-circuit.md`

# 039 计划：Provider 并行竞速与熔断

## 目标

在现有 `FallbackClient` 基础上增加熔断器与可选并行竞速能力，提升 Provider 层的故障隔离与延迟优化。

## 实施步骤

### 1. 实现熔断器核心状态机

新增 `internal/model/circuit.go`：

- `CircuitState` 枚举（Closed / Open / Half-Open）
- `ProviderCircuit` 结构体（失败计数、状态转换、时间窗口）
- `AllowRequest()` 判断是否允许发请求
- `RecordSuccess()` / `RecordFailure()` 更新状态
- `State()` 返回当前状态

验证：单元测试覆盖状态转换逻辑（Closed→Open→Half-Open→Closed/Open）

### 2. 改造 FallbackClient 集成熔断器

修改 `internal/model/fallback.go`：

- `FallbackClient` 增加 `PrimaryCircuit` / `FallbackCircuit` 字段
- `ChatCompletionWithEvents` 调用前检查熔断器状态
- 成功/失败后调用 `RecordSuccess()` / `RecordFailure()`
- 保持向后兼容：无熔断器配置时行为不变

验证：测试熔断器触发时跳过故障 provider

### 3. 实现并行竞速客户端

新增 `internal/model/race.go`：

- `RaceClient` 结构体（主备 provider + 熔断器 + 竞速开关）
- `ChatCompletionWithEvents` 实现并行发请求 + `select` 取最快
- 仅对未熔断的 provider 发请求
- 成功者重置状态，失败者记录失败
- 使用 `context.WithCancel` 取消慢请求

验证：测试竞速模式取最快响应、取消慢请求

### 4. 扩展配置项

修改 `internal/config/config.go`：

- `AGENT_MODEL_RACE_ENABLED`（默认 `false`）
- `AGENT_MODEL_CIRCUIT_FAILURE_THRESHOLD`（默认 `3`）
- `AGENT_MODEL_CIRCUIT_RECOVERY_TIMEOUT_SECONDS`（默认 `60`）
- `AGENT_MODEL_CIRCUIT_HALF_OPEN_MAX_REQUESTS`（默认 `1`）

验证：配置可正确加载并传入客户端

### 5. 更新启动装配

修改 `cmd/agentd/main.go`：

- 按 `AGENT_MODEL_RACE_ENABLED` 选择 `RaceClient` 或 `FallbackClient`
- 传入熔断器配置

验证：启动日志显示当前模式（串行 fallback / 并行 race）

### 6. 增加测试并回归

- `circuit_test.go`：熔断器状态机全覆盖
- `race_test.go`：竞速模式 + 熔断器联动
- `fallback_test.go`：补充熔断器集成测试

验证：`go test ./...` 通过

## 模块影响

- `internal/model`：新增 `circuit.go`、`race.go`，修改 `fallback.go`
- `internal/config`：新增配置项
- `cmd/agentd`：启动装配更新

## 取舍

- 先实现双 provider 竞速，不扩展到多级级联
- 熔断器为进程内状态，不做跨进程持久化
- 并行竞速默认关闭，避免成本激增
- 不引入外部依赖，纯标准库实现

### 来源：`0039-provider-event-coverage.md`

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
