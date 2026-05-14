# 0026 provider summary merged

## 模块

- `provider`

## 类型

- `summary`

## 合并来源

- `0035-provider-summary-merged.md`

## 合并内容

### 来源：`0035-provider-summary-merged.md`

# 0035 provider summary merged

## 模块

- `provider`

## 类型

- `summary`

## 合并来源

- `0004-provider-modes.md`
- `0014-provider-fallback-minimal.md`
- `0015-provider-streaming-openai-minimal.md`
- `0016-provider-streaming-anthropic-minimal.md`
- `0017-provider-streaming-codex-minimal.md`
- `0018-provider-stream-events-passthrough.md`
- `0038-provider-race-circuit.md`
- `0039-provider-event-coverage.md`
- `0048-provider-cascade.md`
- `0255-provider-plugin-runtime-closure.md`

## 合并内容

### 来源：`0004-provider-modes.md`

# 005 总结：Provider 多模式补齐结果

## 已完成

- 新增 `internal/model/anthropic.go`，实现 Anthropic Messages client
- 保留统一 `model.Client` 接口，`Engine` 无需改动
- 新增 provider 切换：
  - `AGENT_MODEL_PROVIDER=openai|anthropic`
- 新增 Anthropic 配置：
  - `ANTHROPIC_BASE_URL`
  - `ANTHROPIC_API_KEY`
  - `ANTHROPIC_MODEL`
- 新增 `internal/model/anthropic_test.go` 覆盖协议映射与解析

## 实现要点

- `system` 消息提取为 Anthropic `system` 字段
- `assistant.tool_calls` 映射为 `tool_use` block
- `tool` 消息映射为 `tool_result` block
- 响应中的 `tool_use` 反解为 `core.ToolCall`

## 验证

- `go test ./...` 通过

## 当前边界

已完成 OpenAI + Anthropic 双模式，但仍未覆盖：

- Codex Responses 模式
- provider 级高级重试/故障切换
- 各 provider 的流式差异抽象

### 来源：`0014-provider-fallback-minimal.md`

# 015 总结：Provider 故障切换最小补齐结果

## 已完成

- 新增 `model.FallbackClient`，支持主备 provider 调用
- 回退触发条件：
  - 错误码：`408/429/500/502/503/504`
  - `timeout`、`context deadline exceeded`、连接拒绝/重置
- 新增配置项：`AGENT_MODEL_FALLBACK_PROVIDER`
- 启动装配支持自动包裹 fallback 客户端
- 新增测试：
  - `internal/model/fallback_test.go`
  - `cmd/agentd/main_test.go`

## 验证

- `go test ./cmd/agentd ./internal/model ./...` 通过

## 当前边界

- 未覆盖并行竞速与熔断策略
- 未做多级 fallback 链路

### 来源：`0015-provider-streaming-openai-minimal.md`

# 016 总结：Provider 流式统一（OpenAI 最小落地）

## 已完成

- `OpenAIClient` 新增 `UseStreaming` 开关
- 开关开启时，`ChatCompletion` 使用 `stream=true` 请求
- 新增 SSE 聚合逻辑：
  - 文本增量合并为最终 `content`
  - 工具调用按 index 聚合并重组 `tool_calls`
- 新增配置项：`AGENT_MODEL_USE_STREAMING`
- 启动装配支持将流式配置传入 OpenAI 客户端
- 新增测试覆盖：
  - 流式文本聚合
  - 流式工具调用聚合
  - 装配层流式开关行为

## 验证

- `go test ./internal/model ./cmd/agentd ./...` 通过

## 当前边界

- Anthropic/Codex 仍未接入流式模式
- provider 级增量事件暂未透传到 Agent 事件流

### 来源：`0016-provider-streaming-anthropic-minimal.md`

# 017 总结：Provider 流式统一（Anthropic 最小落地）

## 已完成

- `AnthropicClient` 新增 `UseStreaming` 开关
- 开关开启时使用 `stream=true` 调用
- 新增 Anthropic SSE 聚合逻辑：
  - `content_block_start`
  - `content_block_delta.text_delta`
  - `content_block_delta.input_json_delta`
- 聚合结果统一转换为标准 `core.Message`
- 启动装配支持将 `AGENT_MODEL_USE_STREAMING` 传入 Anthropic 客户端
- 新增测试：
  - 流式文本聚合
  - 流式工具调用聚合
  - 装配层 Anthropic 流式开关

## 验证

- `go test ./internal/model ./cmd/agentd ./...` 通过

## 当前边界

- Codex 流式仍未补齐
- provider 增量事件仍未透传到 Agent 事件流

### 来源：`0017-provider-streaming-codex-minimal.md`

# 018 总结：Provider 流式统一（Codex 最小落地）

## 已完成

- `CodexClient` 新增 `UseStreaming` 开关
- 开关开启时使用 `stream=true` 调用
- 新增 Codex SSE 聚合逻辑，支持：
  - `response.output_item.added`
  - `response.output_text.delta`
  - `response.function_call_arguments.delta`
  - `response.output` 完整结果包兼容
- 聚合结果统一转换为标准 `core.Message`
- 启动装配支持将 `AGENT_MODEL_USE_STREAMING` 传入 Codex 客户端
- 新增测试：
  - 流式文本聚合
  - 流式函数调用聚合
  - 装配层 Codex 流式开关

## 验证

- `go test ./internal/model ./cmd/agentd ./...` 通过

## 当前边界

- provider 增量事件仍未透传到 Agent 事件流
- 未实现并行竞速与熔断

### 来源：`0018-provider-stream-events-passthrough.md`

# 019 总结：Provider 增量事件透传（最小版）

## 已完成

- 新增模型层可选事件扩展接口：
  - `EventClient`
  - `CompleteWithEvents`
  - `StreamEvent` / `StreamEventSink`
- OpenAI / Anthropic / Codex 流式分支新增增量事件上报
- `FallbackClient` 支持事件透传
- `Engine` 在模型调用中发出统一 `model_stream_event`
- 新增 `agent` 层测试覆盖 `model_stream_event` 透传

## 验证

- `go test ./internal/model ./internal/agent ./...` 通过

## 当前边界

- 事件类型仍是最小集合（`text_delta`、`tool_arguments_delta`）
- 暂未建立完整 provider 事件标准化字典

### 来源：`0038-provider-race-circuit.md`

# 039 总结：Provider 并行竞速与熔断补齐结果

## 已完成

- 新增 `internal/model/circuit.go`：
  - `ProviderCircuit` 熔断器状态机（Closed / Open / Half-Open）
  - `AllowRequest()` 自动处理 Open → Half-Open 转换
  - `RecordSuccess()` / `RecordFailure()` 状态转换与计数
  - `State()` 返回当前状态（含超时推断 Half-Open）
  - `IncrementHalfOpenRequests()` 半开试探计数

- 改造 `internal/model/fallback.go`：
  - `FallbackClient` 增加 `PrimaryCircuit` / `FallbackCircuit` 字段
  - 新增 `NewFallbackClientWithCircuit()` 构造函数
  - `ChatCompletionWithEvents` 调用前检查熔断器状态
  - 熔断器 Open 时自动跳过故障 provider
  - 成功/失败后自动更新熔断器状态

- 新增 `internal/model/race.go`：
  - `RaceClient` 并行竞速客户端
  - 同时向未熔断的 provider 发请求
  - `select` 取最快成功响应
  - 自动取消慢请求
  - 成功者重置状态，失败者记录失败

- 扩展 `internal/config/config.go`：
  - `AGENT_MODEL_RACE_ENABLED`：是否开启并行竞速（默认 `false`）
  - `AGENT_MODEL_CIRCUIT_FAILURE_THRESHOLD`：连续失败阈值（默认 `3`）
  - `AGENT_MODEL_CIRCUIT_RECOVERY_TIMEOUT_SECONDS`：熔断恢复超时（默认 `60`）
  - `AGENT_MODEL_CIRCUIT_HALF_OPEN_MAX_REQUESTS`：半开最大试探请求数（默认 `1`）

- 更新 `cmd/agentd/main.go`：
  - 按 `AGENT_MODEL_RACE_ENABLED` 选择 `RaceClient` 或 `FallbackClientWithCircuit`
  - 启动日志显示当前模式

- 新增测试：
  - `circuit_test.go`：熔断器状态机全覆盖（7 个用例）
  - `race_test.go`：竞速模式 + 熔断器联动（4 个用例）

## 新增配置

```bash
export AGENT_MODEL_RACE_ENABLED=true
export AGENT_MODEL_CIRCUIT_FAILURE_THRESHOLD=3
export AGENT_MODEL_CIRCUIT_RECOVERY_TIMEOUT_SECONDS=60
export AGENT_MODEL_CIRCUIT_HALF_OPEN_MAX_REQUESTS=1
```

## 验证

- `go test ./...` 通过
- 熔断器状态转换测试通过
- 竞速模式取最快响应测试通过
- 熔断器联动 fallback 测试通过

## 当前边界

- 熔断器为进程内状态，不做跨进程持久化
- 并行竞速默认关闭，避免成本激增
- 仅支持双 provider 竞速，不扩展到多级级联
- 未实现 provider 健康探针（依赖请求失败触发）

### 来源：`0039-provider-event-coverage.md`

# 040 总结：Provider 完整事件字典覆盖

## 变更摘要

补齐各 provider 流式事件中可主动提供的关键字段，使下游消费者可依赖统一字段。

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/model/codex.go` | 从 `response.created` 事件提取 `responseID`，在 `message_done` 中添加 `message_id` |
| `internal/model/anthropic.go` | 当 `stop_reason=max_tokens` 时，在 `message_done` 中添加 `incomplete_reason=length` |
| `internal/model/openai.go` | 从流式 chunk 提取 `finish_reason`，当 `finish_reason=length` 时添加 `incomplete_reason=length` |
| `internal/model/anthropic_stream_test.go` | 新增 `TestAnthropicStreamingMaxTokensIncompleteReason` |
| `internal/model/codex_stream_test.go` | 新增 `TestCodexStreamingMessageDoneWithResponseID` |
| `internal/model/openai_stream_test.go` | 新增 `TestOpenAIStreamingLengthIncompleteReason` |

## 补齐后覆盖矩阵

### `message_done` 字段覆盖

| 字段 | OpenAI | Anthropic | Codex |
|------|--------|-----------|-------|
| `message_id` | ❌ 上游限制 | ✅ | ✅ 新增 |
| `finish_reason` | ✅ (含流式提取) | ✅ | ✅ |
| `stop_sequence` | ❌ 上游限制 | ✅ | ❌ 上游限制 |
| `incomplete_reason` | ✅ 新增 | ✅ 新增 | ✅ (已有) |

## 不可补齐的缺口

- OpenAI `message_id`：`chat/completions` 流式响应不提供消息 ID
- OpenAI `stop_sequence`：OpenAI 不支持自定义 stop sequence 的流式返回
- Codex `stop_sequence`：Codex API 不提供 stop sequence

## 测试结果

`go test ./...` 全部通过，新增 3 个测试用例覆盖本次变更。

### 来源：`0048-provider-cascade.md`

# 049 总结：Provider 多级级联与成本感知

## 变更摘要

1. 新增 `CascadeClient` 支持 N 级 provider 级联回退
2. 支持 `cost_aware` 模式按成本权重排序后依次尝试
3. 每级独立熔断器，跳过已开路的 provider
4. 通过 `AGENT_MODEL_CASCADE` 环境变量配置

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/model/cascade.go` | 新建：`CascadeClient` + `ProviderEntry` + `ParseCascadeProviders` |
| `internal/model/cascade_test.go` | 新建：6 个测试用例 |
| `internal/config/config.go` | 新增 `ModelCascade` + `ModelCostAware` |
| `cmd/agentd/main.go` | `buildModelClient` 优先检查 cascade 模式；新增 `buildCascadeClient` |

## 新增能力

### CascadeClient

```go
c := NewCascadeClientWithCircuit([]ProviderEntry{
    {Client: openai, Name: "openai", Cost: 1.0},
    {Client: anthropic, Name: "anthropic", Cost: 2.5},
    {Client: codex, Name: "codex", Cost: 3.0},
}, true, 3, 60*time.Second, 1)
```

- 顺序（cost_aware=false）：按列表顺序依次尝试
- 成本感知（cost_aware=true）：按 Cost 升序排列后依次尝试
- 每级独立熔断器，短路失败快速回退
- 跳过开路 provider，emit `circuit_state` 事件

### 环境变量

```bash
# 级联列表：provider:cost,provider:cost,...
AGENT_MODEL_CASCADE=openai:1.0,anthropic:2.5,codex:3.0

# 启用成本感知排序
AGENT_MODEL_COST_AWARE=true
```

### 与 FallbackClient 的关系

- `AGENT_MODEL_CASCADE` 未设置 → 沿用现有 fallback/race 模式
- `AGENT_MODEL_CASCADE` 已设置 → 优先使用级联模式
- `AGENT_MODEL_COST_AWARE` 控制排序

## 测试结果

`go build ./...` ✅ | `go test ./...` 全部通过 ✅ | `go vet ./...` 无警告 ✅

### 来源：`0255-provider-plugin-runtime-closure.md`

# 255 总结：Provider 插件运行时闭环补齐

本次完成“Provider 插件生态”最小完整闭环：

- 插件契约新增 `type=provider`：
  - `provider.command` 必填
  - 可配置 `provider.args`、`provider.timeout_seconds`、`provider.model`
- 运行时接入：
  - `buildProviderClient` 在非内置 provider 时，从插件清单加载 provider 插件并构造模型客户端
  - 支持插件返回两种响应形态：
    - `{"message": {...}}`
    - OpenAI 兼容 `{"choices":[{"message": {...}}]}`
- 配置与 CLI：
  - `model providers` 现在输出“内置 provider + provider 插件”
  - `model set` / `setup` / `setup wizard` 支持插件 provider 名称
  - `doctor` 对插件 provider 的凭证检查改为“由插件运行时自行管理”

## 测试补齐

- `internal/plugins/provider_client_test.go`
- `internal/plugins/loader_test.go`（provider manifest 校验）
- `cmd/agentd/main_test.go`（provider 插件发现与加载调用）

验证：

- `go test ./internal/plugins ./cmd/agentd -count=1`
- `go test ./...`
