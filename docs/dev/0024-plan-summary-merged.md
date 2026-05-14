# 0024 plan summary merged

## 模块

- `plan`

## 类型

- `summary`

## 合并来源

- `0032-plan-summary-merged.md`

## 合并内容

### 来源：`0032-plan-summary-merged.md`

# 0032 plan summary merged

## 模块

- `plan`

## 类型

- `summary`

## 合并来源

- `001-plan-hermes-agent-go-port.md`
- `002-plan-hermes-gap-closure.md`
- `003-plan-context-compression.md`
- `004-plan-approval-guardrails.md`
- `005-plan-provider-modes.md`
- `006-plan-codex-responses-mode.md`
- `007-plan-mcp-minimal-bridge.md`
- `008-plan-skills-minimal-skeleton.md`
- `009-plan-session-approval-state.md`
- `010-plan-skill-manage-minimal.md`
- `011-plan-skill-manage-support-files.md`
- `012-plan-mcp-stdio-bridge.md`
- `013-plan-mcp-oauth-client-credentials.md`
- `014-plan-mcp-call-streaming-compat.md`
- `015-plan-provider-fallback-minimal.md`
- `016-plan-provider-streaming-openai-minimal.md`
- `017-plan-provider-streaming-anthropic-minimal.md`
- `018-plan-provider-streaming-codex-minimal.md`
- `019-plan-provider-stream-events-passthrough.md`
- `020-plan-model-stream-event-schema-v1.md`
- `021-plan-model-stream-event-schema-v2.md`
- `022-plan-model-stream-event-schema-v2-args-lifecycle.md`
- `023-plan-model-stream-event-schema-v2-usage.md`
- `024-plan-model-stream-event-schema-v2-finish-reason-and-id-aliases.md`
- `025-plan-model-stream-event-schema-v2-id-source-compat.md`
- `026-plan-model-stream-event-schema-v2-termination-metadata.md`
- `027-plan-model-stream-event-schema-v2-finish-incomplete-consistency.md`
- `028-plan-model-stream-event-schema-v2-usage-cache-tokens.md`
- `029-plan-model-stream-event-schema-v2-usage-reasoning-tokens.md`
- `030-plan-model-stream-event-schema-v2-usage-total-consistency.md`
- `031-plan-model-stream-event-schema-v2-usage-consistency-status.md`
- `032-plan-model-stream-event-schema-v2-usage-status-invalid.md`
- `033-plan-model-stream-event-schema-v2-usage-status-provider-coverage.md`
- `034-plan-model-stream-event-schema-v2-usage-status-source-only-coverage.md`
- `035-plan-model-stream-event-schema-v2-usage-status-e2e-provider-streaming.md`
- `036-plan-model-stream-event-schema-v2-usage-status-adjusted-e2e.md`
- `037-plan-model-stream-event-schema-v2-usage-status-adjusted-e2e-anthropic.md`
- `038-plan-model-stream-event-schema-v2-usage-status-table-driven.md`
- `039-plan-provider-race-circuit.md`
- `040-plan-provider-event-coverage.md`
- `041-plan-approval-persistence.md`
- `042-plan-mcp-streaming-passthrough.md`
- `043-plan-mcp-oauth-auth-code.md`
- `044-plan-gateway-minimal.md`
- `045-plan-skills-adv-trigger-sync.md`
- `053-plan-hermes-feature-alignment.md`
- `054-plan-cli-config-management.md`
- `055-plan-cli-model-management.md`
- `056-plan-cli-tools-inspection.md`
- `057-plan-cli-doctor.md`
- `058-plan-cli-gateway-management.md`
- `059-plan-tool-disable-config.md`
- `060-plan-cli-sessions.md`
- `061-plan-cli-session-show-stats.md`
- `062-plan-hermes-cron-alignment.md`
- `063-plan-hermes-toolsets-alignment.md`
- `064-plan-hermes-send-message-alignment.md`
- `065-plan-hermes-patch-tool-alignment.md`
- `066-plan-hermes-web-tools-alignment.md`
- `067-plan-hermes-clarify-tool-alignment.md`
- `068-plan-hermes-execute-code-alignment.md`
- `207-plan-frontend-tui-parity-phase1.md`

## 合并内容

### 来源：`001-plan-hermes-agent-go-port.md`

# 001 计划：Hermes Agent Go 版实施计划

## 目标

在 Go 中实现 Hermes 风格 Agent 的完整核心闭环，并同时提供 CLI 与 HTTP API。

## 实施步骤

1. 建立核心共享类型与模型客户端
验证：可向 OpenAI 兼容接口发送消息并解析响应

2. 建立工具注册中心与内置工具
验证：可输出 tool schema，并能按工具名 dispatch

3. 实现 Agent Loop
验证：模型返回 `tool_calls` 时，工具结果可回灌并继续多轮执行

4. 实现会话与记忆持久化
验证：可加载 session 历史，可执行 session_search，可写入 `MEMORY.md` / `USER.md`

5. 实现 CLI 与 HTTP API
验证：CLI 可交互调用；HTTP `/v1/chat` 可返回完整结果

6. 添加关键测试并跑通
验证：`go test ./...` 通过

7. 沉淀调研、设计、总结文档
验证：`docs/` 与 `docs/dev/README.md` 索引完整

### 来源：`002-plan-hermes-gap-closure.md`

# 002 计划：Hermes 核心闭环差异补齐

## 目标

补齐当前 Go 版与 Hermes 核心闭环之间的剩余关键差异，使 Agent 在跨请求、多轮运行和工具执行安全边界上达到“完整核心功能”状态。

## 实施步骤

1. 重构系统提示词装配
验证：无论是否存在历史消息，每次 `Run()` 都会携带 system message，且不会重复叠加多份 system message

2. 补齐持久记忆回灌
验证：`MEMORY.md` / `USER.md` 的内容会进入系统提示词，后续 session 可直接复用

3. 注入工作区规则
验证：工作目录存在 `AGENTS.md` 时，其内容会以受控方式进入系统提示词

4. 增加文件工具工作区路径约束
验证：`read_file` / `write_file` / `search_files` 只能访问 `Workdir` 内路径，越界访问返回明确错误

5. 增加 terminal 硬阻断护栏
验证：明显灾难性命令会被拒绝执行，正常命令保持兼容

6. 增加针对性测试并回归验证
验证：新增单元测试覆盖提示词装配、记忆回灌、路径约束、危险命令阻断，`go test ./...` 通过

## 模块影响

- `internal/agent`
- `internal/memory`
- `internal/tools`
- `cmd/agentd`
- `docs/`

## 取舍

- 先补“闭环缺口”，不在本次引入完整审批系统与上下文压缩，避免为了追求 1:1 复刻而显著扩大范围
- 安全侧优先实现硬阻断与工作区边界，后续再扩展到审批、URL 安全和更细粒度权限

### 来源：`003-plan-context-compression.md`

# 003 计划：Context Compression 补齐

## 目标

为 `Engine.Run()` 增加最小可用上下文压缩能力，避免长会话消息无限膨胀。

## 实施步骤

1. 实现确定性压缩器
验证：给定超预算消息，能稳定输出“头部 + 摘要 + 尾部”结构

2. 接入 Agent Loop
验证：每轮模型调用前会执行预算检查，超预算则自动压缩

3. 增加可观测事件
验证：压缩发生时会发出 `context_compacted`，携带前后体积与裁剪数量

4. 增加配置项
验证：支持通过环境变量调整预算和尾部保留条数

5. 增加测试并回归
验证：新增压缩器和 loop 级测试通过，`go test ./...` 通过

## 配置约定

- `AGENT_MAX_CONTEXT_CHARS`：上下文字符预算，默认 `120000`
- `AGENT_COMPRESSION_TAIL_MESSAGES`：尾部保留消息数，默认 `14`

### 来源：`004-plan-approval-guardrails.md`

# 004 计划：审批护栏补齐

## 目标

在 terminal 工具中补齐“危险命令审批门禁”，形成 hardline 与危险命令分层控制。

## 实施步骤

1. 新增危险命令模式库
验证：可识别递归删除、高风险权限修改、远程脚本直管道执行等命令

2. 接入 terminal 门禁
验证：危险命令未设置 `requires_approval=true` 时拒绝执行

3. 保持 hardline 优先级
验证：hardline 命令即使设置 `requires_approval=true` 仍被阻断

4. 补充 schema 与返回字段
验证：`terminal` schema 包含 `requires_approval`，返回中保留该字段

5. 增加测试并回归
验证：新增审批门禁测试通过，`go test ./...` 通过

### 来源：`005-plan-provider-modes.md`

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

### 来源：`006-plan-codex-responses-mode.md`

# 006 计划：Codex Responses 模式补齐

## 目标

新增 `provider=codex` 模式，支持 Responses API 下的 tool-calling 闭环。

## 实施步骤

1. 新增 Codex client
验证：可调用 `/responses` 并解析 assistant 文本

2. 增加工具调用映射
验证：可解析 `function_call` 并生成 `core.ToolCall`

3. 增加工具结果映射
验证：tool 消息可映射到 `function_call_output` 输入项

4. 接入配置切换
验证：`AGENT_MODEL_PROVIDER=codex` 时可正常创建 Codex client

5. 增加测试并回归
验证：模型层新增测试通过，`go test ./...` 通过

### 来源：`007-plan-mcp-minimal-bridge.md`

# 007 计划：MCP 最小接入骨架

## 目标

实现 MCP HTTP 最小桥接能力，使外部 MCP 工具可被本地 Agent 直接使用。

## 实施步骤

1. 新增 MCP client
验证：可请求 `/tools` 并解析工具定义

2. 新增动态注册流程
验证：启动时发现到的 MCP 工具可进入 `Registry`

3. 新增调用转发
验证：调用 MCP 工具时可向 `/call` 发送 `name + arguments + context`

4. 增加配置项
验证：`AGENT_MCP_ENDPOINT` 配置后可自动启用 MCP 发现

5. 增加测试并回归
验证：MCP 发现与调用转发测试通过，`go test ./...` 通过

### 来源：`008-plan-skills-minimal-skeleton.md`

# 008 计划：Skills 最小骨架补齐

## 目标

补齐本地技能最小入口能力，让 Agent 可发现并读取技能说明。

## 实施步骤

1. 新增技能工具注册
验证：工具列表包含 `skill_list` 与 `skill_view`

2. 实现技能目录扫描
验证：可列出 `skills/<name>/SKILL.md`，并返回名称与简介

3. 实现技能查看
验证：按技能名读取技能全文；非法名称会被拒绝

4. 增加测试并回归
验证：技能工具测试通过，`go test ./...` 通过

### 来源：`009-plan-session-approval-state.md`

# 009 计划：会话级审批状态补齐

## 目标

补齐会话级危险命令审批状态，实现 `grant -> 多命令复用 -> revoke/过期` 的最小闭环。

## 实施步骤

1. 新增审批状态存储
验证：支持按 `session_id` 授权、撤销、过期判断

2. 新增 `approval` 工具
验证：支持 `status` / `grant` / `revoke`

3. 接入 terminal 审批判定
验证：危险命令在会话已授权时可执行；hardline 仍不可放行

4. 增加配置项
验证：支持默认授权 TTL 配置

5. 增加测试并回归
验证：审批状态与 terminal 联动测试通过，`go test ./...` 通过

### 来源：`010-plan-skill-manage-minimal.md`

# 010 计划：`skill_manage` 最小补齐

## 目标

补齐 Go 版技能管理最小闭环，使 Agent 在工作区内可安全地创建、编辑、定点修改和删除技能。

## 实施步骤

1. 在 `internal/tools/builtin.go` 注册 `skill_manage` 工具及参数 schema。
2. 实现 `skill_manage` handler，支持：
   - `create`：创建 `<skills>/<name>/SKILL.md`
   - `edit`：覆盖 `SKILL.md`
   - `patch`：按 `old_string/new_string` 做唯一或全量替换
   - `delete`：删除技能目录
3. 复用工作区路径约束，确保技能操作不越界。
4. 增加技能名校验规则（仅允许安全字符集）。
5. 在 `internal/tools/builtin_test.go` 补充功能测试与拒绝路径测试。
6. 执行 `go test ./...` 完整回归。
7. 更新文档索引与总览文档。

## 验证标准

- `skill_manage` 四类动作可用且返回结构化结果。
- 非法技能名会被拒绝。
- `patch` 在多重匹配且未开启 `replace_all` 时拒绝执行。
- 全量测试 `go test ./...` 通过。

### 来源：`011-plan-skill-manage-support-files.md`

# 011 计划：`skill_manage` 支撑文件能力补齐

## 目标

让 `skill_manage` 支持技能目录内支撑文件的写入与删除，保持最小可用和安全约束。

## 实施步骤

1. 在 `skill_manage` 中新增动作分支：
   - `write_file`
   - `remove_file`
2. 扩展 `skill_manage` 参数 schema：
   - `file_path`
   - `file_content`
3. 新增路径校验逻辑：
   - 仅允许相对路径
   - 拒绝 `..` 穿越
   - 强制首段在 `references/templates/scripts/assets`
4. 新增测试：
   - 写入并删除支撑文件成功
   - 非法路径被拒绝（穿越、非法子目录）
5. 执行 `go test ./...` 回归。

## 验证标准

- `write_file/remove_file` 可用且只在允许子目录生效
- 非法路径有明确拒绝错误
- 全量测试通过

### 来源：`012-plan-mcp-stdio-bridge.md`

# 012 计划：MCP `stdio` 最小桥接

## 目标

新增 `stdio` 传输能力，使 MCP 工具可通过子进程标准输入输出接入。

## 实施步骤

1. 扩展 `internal/tools/mcp.go`：
   - `MCPClient` 增加 `StdioCommand`
   - 新增 `NewMCPStdioClient`
   - `DiscoverTools` 和工具 `Call` 增加 `stdio` 分支
2. 新增最小 JSON-RPC framing：
   - 读写 `Content-Length` 帧
   - 请求/响应 `id` 匹配
3. 新增一次会话调用流程：
   - 启动子进程
   - `initialize` + `notifications/initialized`
   - `tools/list` 或 `tools/call`
4. 扩展配置与启动装配：
   - `AGENT_MCP_TRANSPORT`
   - `AGENT_MCP_STDIO_COMMAND`
5. 增加 `mcp_test` 的 `stdio` 子进程测试（helper process）。
6. 执行全量测试 `go test ./...`。

## 验证标准

- HTTP 模式行为不回退
- `stdio` 模式可发现并调用 MCP 工具
- 全量测试通过

### 来源：`013-plan-mcp-oauth-client-credentials.md`

# 013 计划：MCP OAuth（Client Credentials）最小补齐

## 目标

让 MCP HTTP 桥接支持 OAuth `client_credentials`，以接入需要 Bearer Token 的 MCP 服务。

## 实施步骤

1. 扩展 `MCPClient` OAuth 配置结构。
2. 增加 token 获取逻辑：
   - `grant_type=client_credentials`
   - Basic Auth 传 `client_id/client_secret`
3. 增加 token 缓存与到期前复用。
4. 在 HTTP `/tools` 与 `/call` 请求注入 Bearer Token。
5. 扩展配置项并在启动装配阶段注入 OAuth 配置。
6. 增加 MCP OAuth 测试并执行全量回归。

## 验证标准

- OAuth MCP 发现与调用均带认证头
- token 在有效期内被复用（非每次重复申请）
- `go test ./...` 通过

### 来源：`014-plan-mcp-call-streaming-compat.md`

# 014 计划：MCP `/call` 流式响应兼容

## 目标

让 MCP `/call` 在 `text/event-stream` 响应下也能返回稳定结果，并与现有非流式调用保持兼容。

## 实施步骤

1. 在 `mcpToolProxy.Call` 中识别 `Content-Type: text/event-stream`。
2. 新增 SSE 解析函数：
   - 按事件空行分隔
   - 聚合 `data:` 多行
   - 解析 JSON 事件并提取 `result/structuredContent/error`
3. 保留原非流式 JSON 处理逻辑，避免回归。
4. 新增测试覆盖 SSE `/call` 成功链路。
5. 运行 `go test ./...` 全量回归。

## 验证标准

- SSE MCP `/call` 能返回正确结果
- 原 HTTP JSON 模式无回归
- 全量测试通过

### 来源：`015-plan-provider-fallback-minimal.md`

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

### 来源：`016-plan-provider-streaming-openai-minimal.md`

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

### 来源：`017-plan-provider-streaming-anthropic-minimal.md`

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

### 来源：`018-plan-provider-streaming-codex-minimal.md`

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

### 来源：`019-plan-provider-stream-events-passthrough.md`

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

### 来源：`020-plan-model-stream-event-schema-v1.md`

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

### 来源：`021-plan-model-stream-event-schema-v2.md`

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

### 来源：`022-plan-model-stream-event-schema-v2-args-lifecycle.md`

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

### 来源：`023-plan-model-stream-event-schema-v2-usage.md`

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

### 来源：`024-plan-model-stream-event-schema-v2-finish-reason-and-id-aliases.md`

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

### 来源：`025-plan-model-stream-event-schema-v2-id-source-compat.md`

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

### 来源：`026-plan-model-stream-event-schema-v2-termination-metadata.md`

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

### 来源：`027-plan-model-stream-event-schema-v2-finish-incomplete-consistency.md`

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

### 来源：`028-plan-model-stream-event-schema-v2-usage-cache-tokens.md`

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

### 来源：`029-plan-model-stream-event-schema-v2-usage-reasoning-tokens.md`

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

### 来源：`030-plan-model-stream-event-schema-v2-usage-total-consistency.md`

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

### 来源：`031-plan-model-stream-event-schema-v2-usage-consistency-status.md`

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

### 来源：`032-plan-model-stream-event-schema-v2-usage-status-invalid.md`

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

### 来源：`033-plan-model-stream-event-schema-v2-usage-status-provider-coverage.md`

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

### 来源：`034-plan-model-stream-event-schema-v2-usage-status-source-only-coverage.md`

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

### 来源：`035-plan-model-stream-event-schema-v2-usage-status-e2e-provider-streaming.md`

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

### 来源：`036-plan-model-stream-event-schema-v2-usage-status-adjusted-e2e.md`

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

### 来源：`037-plan-model-stream-event-schema-v2-usage-status-adjusted-e2e-anthropic.md`

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

### 来源：`038-plan-model-stream-event-schema-v2-usage-status-table-driven.md`

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

### 来源：`039-plan-provider-race-circuit.md`

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

### 来源：`040-plan-provider-event-coverage.md`

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

### 来源：`041-plan-approval-persistence.md`

# 041 计划：审批状态持久化与细粒度审批策略

## 目标

1. 审批授权持久化到 SQLite，进程重启后可恢复
2. 支持按危险命令类别细粒度授权
3. 保持向后兼容

## 实施步骤

### 1. 扩展 SessionStore 新增 approvals 表

修改 `internal/store/session_store.go`：
- `init()` 新增 `approvals` 表创建
- 新增 `GrantApproval(sessionID, scope, pattern string, expiresAt time.Time) error`
- 新增 `RevokeApproval(sessionID string, scope string, pattern string) error`
- 新增 `ListApprovals(sessionID string) ([]ApprovalRecord, error)`
- 新增 `IsApproved(sessionID string, scope string, pattern string) (bool, error)`
- 新增 `CleanupExpiredApprovals() error`

验证：approval 表可正确创建和操作

### 2. 定义 ApprovalRecord 类型

在 `internal/store/session_store.go` 新增：
```go
type ApprovalRecord struct {
    ID        int64
    SessionID string
    Scope     string    // "session" or "pattern"
    Pattern   string    // 危险命令类别标识符
    GrantedAt time.Time
    ExpiresAt time.Time
}
```

### 3. 扩展 detectDangerousCommand 返回类别标识符

修改 `internal/tools/safety.go`：
- `detectDangerousCommand` 返回 `(category string, description string, dangerous bool)`
- `category` 为机器可读标识符（如 `recursive_delete`、`world_writable`、`root_ownership`、`remote_pipe_shell`、`service_lifecycle`）

验证：每个危险命令模式返回唯一类别标识符

### 4. 扩展 ApprovalStore 支持持久化

修改 `internal/tools/approval_store.go`：
- 新增 `PersistentApprovalStore`，组合内存缓存 + SQLite 持久化
- `Grant` 同时写内存和 SQLite
- `IsApproved` 先查内存，miss 时查 SQLite
- `Revoke` 同时删内存和 SQLite
- 新增 `LoadFromStore(sessionID string)` 从 SQLite 恢复到内存

验证：持久化 store 的 grant/revoke/status 行为与内存版一致

### 5. 扩展 approval 工具

修改 `internal/tools/builtin.go`：
- `grant` 新增可选 `scope`（默认 `session`）和 `pattern` 参数
- `status` 返回当前有效授权列表，包含 scope 和 pattern
- `revoke` 新增可选 `scope` 和 `pattern` 参数（不指定则撤销全部）

验证：`grant scope=pattern pattern=recursive_delete` 仅授权递归删除类命令

### 6. 修改 terminal 审批判定逻辑

修改 `internal/tools/builtin.go`：
- 危险命令判定时，先检查 session 级授权，再检查 pattern 级授权
- 使用 `detectDangerousCommand` 返回的 category 匹配 pattern

验证：pattern 级授权仅放行匹配类别的危险命令

### 7. 接入启动装配

修改 `cmd/agentd/main.go`：
- 使用 `PersistentApprovalStore` 替代内存版
- 启动时传入 SessionStore 用于持久化

验证：重启后审批授权仍有效

### 8. 增加测试并回归

- 持久化 store 测试
- 细粒度授权测试
- 重启恢复测试
- `go test ./...` 通过

## 模块影响

- `internal/store/session_store.go`
- `internal/tools/approval_store.go`
- `internal/tools/safety.go`
- `internal/tools/builtin.go`
- `internal/agent/loop.go`
- `cmd/agentd/main.go`

## 向后兼容

- `scope=session` 行为与当前完全一致
- 不指定 `scope`/`pattern` 时默认为 session 级授权
- 现有 `requires_approval` 单次放行逻辑不变

### 来源：`042-plan-mcp-streaming-passthrough.md`

# 042 计划：MCP 流式事件透传

## 目标

让 MCP `/call` SSE 中间事件透传到 Agent 事件总线，客户端可通过 SSE 实时感知 MCP 工具执行进度。

## 实施步骤

### 步骤 1：扩展 ToolContext 增加 ToolEventSink

文件：`internal/tools/registry.go`

- 新增 `ToolEventSink` 类型：`func(eventType string, data map[string]any)`
- 在 `ToolContext` 中新增 `ToolEventSink` 可选字段

### 步骤 2：新增 parseMCPCallSSEWithCallback

文件：`internal/tools/mcp.go`

- 新增 `parseMCPCallSSEWithCallback(body io.Reader, sink ToolEventSink) (map[string]any, error)`
- 每解析到一个 SSE data 事件就调用 `sink`
- 最终聚合结果仍由返回值提供
- 保留原 `parseMCPCallSSE` 不变（无回调场景使用）

### 步骤 3：改造 mcpToolProxy.Call 使用回调

文件：`internal/tools/mcp.go`

- SSE 分支判断 `tc.ToolEventSink != nil` 时使用 `parseMCPCallSSEWithCallback`
- 否则使用原 `parseMCPCallSSE`

### 步骤 4：Agent Loop 注册回调

文件：`internal/agent/loop.go`

- 在工具执行循环中，构造 `ToolContext` 时注册 `ToolEventSink`
- 回调内发出 `mcp_stream_event` 类型的 `AgentEvent`

### 步骤 5：增加测试

文件：`internal/tools/mcp_test.go`、`internal/agent/loop_test.go`

- MCP SSE 回调测试：验证中间事件被回调、最终结果正确
- Agent Loop 测试：验证 `mcp_stream_event` 被发出
- 原有聚合模式测试不回归

### 步骤 6：全量回归

`go test ./...`

## 验证标准

- MCP SSE 中间事件通过 `mcp_stream_event` 透传到 SSE 客户端
- 最终结果仍由 `tool_finished` 事件提供
- 非 MCP 工具和原有聚合模式不受影响
- 全量测试通过

### 来源：`043-plan-mcp-oauth-auth-code.md`

# 043 计划：MCP OAuth 授权码模式与刷新令牌

## 目标

在保持现有 `client_credentials` 模式不变的前提下，补齐 MCP OAuth 授权码模式与刷新令牌能力。

## 实施步骤

### 步骤 1：扩展 MCPOAuthConfig 与 MCPClient

修改 `internal/tools/mcp.go`：
- `MCPOAuthConfig` 新增 `AuthURL`、`RedirectURL`、`GrantType` 字段
- `MCPClient` 新增 `cachedRefreshToken` 字段
- 新增 `ConfigureOAuthAuthCode` 方法

验证：编译通过，现有测试不受影响

### 步骤 2：令牌持久化

修改 `internal/store/session_store.go`：
- 新增 `oauth_tokens` 表
- 新增 `SaveOAuthToken`、`LoadOAuthToken`、`DeleteOAuthToken` 方法

验证：持久化存取测试通过

### 步骤 3：刷新令牌逻辑

修改 `internal/tools/mcp.go` 的 `oauthAccessToken`：
- token 过期时优先使用 `refresh_token` 刷新
- 刷新成功后更新 `cachedAccessToken` 和 `cachedRefreshToken`
- 刷新失败时根据 `GrantType` 决定是否重新走授权码流程
- 解析 token 响应中的 `refresh_token` 字段

验证：刷新令牌测试通过

### 步骤 4：授权码流程

修改 `internal/tools/mcp.go`：
- 新增 `StartOAuthCallbackServer` 方法，启动本地 HTTP 回调服务器
- 新增 `ExchangeAuthCode` 方法，用授权码换取 token
- `oauthAccessToken` 在无缓存且 `GrantType=authorization_code` 时，从 SQLite 加载持久化 token

修改 `cmd/agentd/main.go`：
- 启动时检测 `GrantType=authorization_code`，启动回调服务器并输出授权 URL

验证：授权码流程端到端测试通过

### 步骤 5：配置扩展

修改 `internal/config/config.go`：
- 新增 `MCPOAuthGrantType`、`MCPOAuthAuthURL`、`MCPOAuthRedirectURL`、`MCPOAuthCallbackPort`

修改 `cmd/agentd/main.go`：
- 根据 `GrantType` 选择 `ConfigureOAuthClientCredentials` 或 `ConfigureOAuthAuthCode`

验证：配置加载正确

### 步骤 6：增加测试并回归

- 授权码模式端到端测试（httptest 模拟 OAuth 服务器）
- 刷新令牌测试
- 令牌持久化测试
- `go test ./...` 全量回归

## 模块影响

- `internal/tools/mcp.go`
- `internal/store/session_store.go`
- `internal/config/config.go`
- `cmd/agentd/main.go`

## 向后兼容

- `GrantType` 默认为 `client_credentials`，行为与之前完全一致
- 不配置新环境变量时，现有 MCP OAuth 行为不变
- `refresh_token` 对 `client_credentials` 模式同样生效（如果 OAuth 服务器返回了 refresh_token）

### 来源：`044-plan-gateway-minimal.md`

# 044 实施计划：多平台消息网关最小落地

## 实现步骤

### 1. 扩展配置 → 验证：`go build ./...` 通过
- 在 `Config` 中新增 `GatewayEnabled`、`TelegramToken`、`TelegramAllowed`
- 环境变量：`AGENT_GATEWAY_ENABLED`、`AGENT_TELEGRAM_BOT_TOKEN`、`AGENT_TELEGRAM_ALLOWED_USERS`

### 2. 创建 `internal/gateway/adapter.go` → 验证：编译通过
- 定义 `PlatformAdapter` 接口：`Name()`/`Connect()`/`Disconnect()`/`Send()`/`EditMessage()`/`SendTyping()`/`OnMessage()`
- 定义 `MessageEvent` 归一化入站消息类型
- 定义 `SendResult` 发送结果类型
- 定义 `MessageHandler` 回调类型

### 3. 创建 `internal/gateway/session.go` → 验证：编译通过
- `BuildSessionKey(platform, chatType, chatID)` 函数
- `SessionSource` 类型（platform, chat_id, chat_type, user_id, user_name）

### 4. 创建 `internal/gateway/auth.go` → 验证：编译通过
- `CheckAuthorization(allowedUsers, userID)` 函数
- 默认拒绝原则

### 5. 创建 `internal/gateway/events.go` → 验证：编译通过
- `StreamCollector` 结构：收集 `text_delta` 增量、合并最终内容
- `FinalContent()` 返回完整文本
- 为 Telegram 消息编辑提供进度缓冲

### 6. 创建 `internal/gateway/platforms/telegram.go` → 验证：`go build ./...` 通过
- `TelegramAdapter` 实现 `PlatformAdapter`
- 使用 `go-telegram-bot-api/v5` 的 `GetUpdatesChan`（长轮询）
- `Send`/`EditMessage`/`SendTyping` 实现
- `OnMessage` 注册回调、内部消息循环（单 goroutine 串行）

### 7. 创建 `internal/gateway/runner.go` → 验证：编译通过
- `GatewayRunner` 结构：适配器列表、store、engine、事件映射
- `Start()`/`Stop()` 生命周期管理
- `handleMessage()` 消息路由：sessionKey 构建 → 授权 → 加载历史 → 创建 EventSink → `engine.Run()` → 响应回传
- 流式事件映射：`text_delta` → 累积 + 限流编辑 → `completed` 最终化

### 8. 修改 `cmd/agentd/main.go` → 验证：`go build ./...` 通过
- `runServe` 中：gateway enabled 时创建 `GatewayRunner`，`Start()` + `Stop()` 通过 `context.WithCancel`
- `mustBuildEngine` 中：gateway 需要 `ModelUseStreaming = true`（流式编辑需要增量事件）
- 优雅关闭：SIGINT/SIGTERM 时 `Stop()` 所有适配器

### 9. 写入测试 → 验证：`go test ./...`
- `internal/gateway/platforms/telegram_test.go`：Mock Telegram Update/Message、验证 sessionKey 构建和消息路由
- `internal/gateway/events_test.go`：StreamCollector 增量累积正确性

### 10. 完整流程验证
- 构建 `go build ./...`
- 测试 `go test ./...` 全部通过
- `go vet ./...` 无警告

## 文件变更清单

| 文件 | 动作 | 内容 |
|------|------|------|
| `internal/config/config.go` | 修改 | 新增 GatewayEnabled/TelegramToken/TelegramAllowed |
| `internal/gateway/adapter.go` | 新建 | PlatformAdapter 接口 + MessageEvent/SendResult 类型 |
| `internal/gateway/session.go` | 新建 | BuildSessionKey + SessionSource |
| `internal/gateway/auth.go` | 新建 | CheckAuthorization |
| `internal/gateway/events.go` | 新建 | StreamCollector（增量事件累积） |
| `internal/gateway/runner.go` | 新建 | GatewayRunner 生命周期、消息路由、事件映射 |
| `internal/gateway/platforms/telegram.go` | 新建 | TelegramBotAdapter |
| `internal/gateway/platforms/telegram_test.go` | 新建 | Mock Telegram 测试 |
| `internal/gateway/events_test.go` | 新建 | StreamCollector 单元测试 |
| `cmd/agentd/main.go` | 修改 | serve 模式集成网关启动 |
| `go.mod` | 修改 | 新增 `go-telegram-bot-api/v5` 依赖 |

## 关键设计决策

1. **适配器单 goroutine 串行**：每个适配器的消息回调在单 goroutine 中串行调用，天然避免并发问题
2. **Agent 运行在独立 goroutine**：与 HTTP API 模式一致，Agent 运行在独立 goroutine，event 通过 channel 收集
3. **流式编辑限流 500ms**：避免 Telegram API 编辑频率过高触发限流
4. **默认拒绝授权**：空 `allowedUsers` = 拒绝所有用户，需要显式配置
5. **不引入新的外部持久化**：网关会话复用 `store.SessionStore`，不新增存储表

### 来源：`045-plan-skills-adv-trigger-sync.md`

# 045 实施计划：技能高级能力（自动触发 + 同步）

## 实现步骤

### 1. 技能索引注入 system prompt → 验证：编译 + 测试
- 新增 `buildSkillsIndexBlock(workdir string) string` 函数
- 扫描 `<workdir>/skills/*/SKILL.md`，提取 name + description
- 限制最大 50 个条目
- 在 `buildRuntimeSystemPrompt()` 中追加注入
- 新增 `TestBuildSkillsIndexBlock` 测试（空目录、有技能、超限截断）

### 2. `skill_manage` 新增 `sync` 动作 → 验证：编译 + 测试
- 支持 `source=github`：`repo` + `path` 参数
  - 调用 `https://api.github.com/repos/{repo}/contents/{path}`
  - 遍历含 SKILL.md 的子目录，下载 SKILL.md
  - 写入本地 `skills/<name>/SKILL.md`，同时下载 support files
- 支持 `source=url`：`url` + `name` 参数
  - HTTP GET 获取原始内容
  - 写入 `skills/<name>/SKILL.md`
- 新增 `TestSkillManageSyncURL`、`TestSkillManageSyncGitHub` Mock 测试

### 3. 完整验证
- `go build ./...`
- `go test ./...` 全部通过
- `go vet ./...` 无警告

## 文件变更清单

| 文件 | 动作 | 内容 |
|------|------|------|
| `internal/agent/system_prompt.go` | 修改 | 新增 `buildSkillsIndexBlock()` |
| `internal/tools/builtin.go` | 修改 | `skill_manage` 新增 `sync` 动作 |
| `internal/tools/builtin_test.go` | 修改 | 新增 sync 测试 |
| `internal/agent/loop_test.go` | 修改 | 新增 system prompt 技能注入测试 |

## 关键设计决策

1. **技能索引注入位置**：在 `buildRuntimeSystemPrompt` 中追加，与 memory 和 workspace rules 同级
2. **索引格式**：`## Available Skills` + 强指令 + `name: description` 列表
3. **GitHub sync**：使用 GitHub Contents API（无需认证，限流 60req/h），30s 超时
4. **URL sync**：直接 HTTP GET，支持任何可公开访问的原始 SKILL.md URL

### 来源：`053-plan-hermes-feature-alignment.md`

# 053 Plan：Hermes 功能对齐文档完善

## 目标

明确当前 Go 项目与 `/data/source/hermes-agent` 的功能对齐范围，并补齐总览文档中的差异说明。

## 变更范围

- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `README.md`
- `docs/dev/053-research-hermes-feature-alignment.md`
- `docs/dev/053-plan-hermes-feature-alignment.md`
- `docs/dev/053-summary-hermes-feature-alignment.md`
- `docs/dev/README.md`

不修改 Go 源码、不新增依赖、不改变运行行为。

## 执行步骤

1. 梳理 Hermes 和当前项目功能面。
   - 验证：Research 文档列出已对齐、最小覆盖、未覆盖能力。
2. 更新产品总览。
   - 验证：总览明确当前项目是 Hermes 核心 Agent daemon 子集，而非完整复刻。
3. 更新 README 入口说明。
   - 验证：仓库首页能直接看到对齐边界并链接到详细矩阵。
4. 更新开发总览。
   - 验证：开发文档包含模块级功能矩阵和后续补齐建议。
5. 更新需求索引和 Summary。
   - 验证：`docs/dev/README.md` 能追溯到 053 三份文档。

## 不做事项

- 不实现 Hermes 缺失功能。
- 不调整已有配置或源码。
- 不修改当前工作区中已有的非文档变更。

## 验证方式

- 查看 `git diff -- docs README.md`，确认只包含文档补齐。
- 人工复核对齐矩阵与本地源码、Hermes 文档一致。

## 三角色审视

- 高级产品：任务聚焦“分析与文档完善”，没有扩展到功能开发。
- 高级架构师：文档按产品/开发/需求沉淀分层，便于后续需求引用。
- 高级工程师：变更可回滚、无运行时风险，验证成本低。

### 来源：`054-plan-cli-config-management.md`

# 054 Plan：CLI 配置管理最小实现

## 任务

1. `internal/config` 增加配置文件管理函数。
   - 验证：单元测试可写入、读取、列出配置。
2. `config.Load()` 支持 `AGENT_CONFIG_FILE`。
   - 验证：测试中设置环境变量后读取指定文件。
3. `cmd/agentd` 增加 `config list|get|set`。
   - 验证：构建通过，命令语义可通过测试或手动命令验证。
4. 更新 README 与需求文档。
   - 验证：文档包含命令示例和配置优先级说明。

## 边界

- 不改变已有环境变量优先级：环境变量 > 配置文件 > 内置默认值。
- 不迁移配置格式。
- 不新增运行时依赖。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/config`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`（如沙箱允许监听端口）

### 来源：`055-plan-cli-model-management.md`

# 055 Plan：CLI 模型管理最小实现

## 任务

1. 增加 `agentd model show`。
   - 验证：根据 `config.Config` 输出当前 provider/model/base URL。
2. 增加 `agentd model providers`。
   - 验证：输出 `openai`、`anthropic`、`codex`。
3. 增加 `agentd model set`。
   - 验证：支持 `provider model` 与 `provider:model` 两种输入；写入正确 INI 键位。
4. 更新 README 与总览文档。
   - 验证：文档包含示例与边界说明。

## 边界

- 不做模型目录拉取。
- 不新增 provider。
- 不改变环境变量优先级。
- 不改变 `buildModelClient`。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 临时配置文件手动验证 `model set/show`。

### 来源：`056-plan-cli-tools-inspection.md`

# 056 Plan：CLI 工具查看最小实现

## 任务

1. 将 `agentd tools` 路由到 `runTools`。
   - 验证：无参数时仍列出工具名。
2. 增加 `tools list`。
   - 验证：输出与无参数保持一致。
3. 增加 `tools show tool_name`。
   - 验证：输出单个 `core.ToolSchema` JSON。
4. 增加 `tools schemas`。
   - 验证：输出完整 schema JSON 列表。
5. 更新 README、总览文档和需求索引。
   - 验证：文档列出新命令和边界。

## 边界

- 不新增工具。
- 不增加工具启停或 toolset 配置。
- 不改变 MCP discovery 行为。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 手动验证 `tools list/show/schemas`

### 来源：`057-plan-cli-doctor.md`

# 057 Plan：CLI 本地诊断最小实现

## 任务

1. 增加 `agentd doctor`。
   - 验证：输出文本检查结果，含 `ok/warn/error`。
2. 增加 `agentd doctor -json`。
   - 验证：输出结构化 JSON。
3. 增加诊断 helper。
   - 验证：测试覆盖缺 API key warning、坏 workdir error、Gateway 无 token warning。
4. 更新 README、总览文档和需求索引。
   - 验证：文档列出命令和本期边界。

## 边界

- 不做网络探测。
- 不调用 provider API。
- 不启动 MCP/Gateway。
- 不修改用户配置。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 手动验证 `doctor` 与 `doctor -json`

### 来源：`058-plan-cli-gateway-management.md`

# 058 Plan：CLI 网关管理最小实现

## 任务

1. 增加 `agentd gateway status [-file path] [-json]`。
   - 验证：输出 enabled、configured_platforms、supported_platforms。
2. 增加 `agentd gateway platforms`。
   - 验证：输出 telegram、discord、slack。
3. 增加 `agentd gateway enable|disable [-file path]`。
   - 验证：写入 `gateway.enabled=true/false`。
4. 更新 README、总览文档与需求索引。
   - 验证：文档说明 token 仍通过 config 管理。

## 边界

- 不启动 Gateway。
- 不检查平台 token 真实性。
- 不实现 pairing 或 setup wizard。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 手动验证 `gateway status/platforms/enable/disable`

### 来源：`059-plan-tool-disable-config.md`

# 059 Plan：工具禁用配置最小实现

## 任务

1. 增加配置项。
   - 验证：`LoadFile` 能读取 `[tools] disabled`。
2. 增加 registry 禁用能力。
   - 验证：禁用后 schema 中不再出现对应工具。
3. 增加 CLI。
   - 验证：`tools disable` 写入列表，`tools enable` 移除列表，`tools disabled` 可查看。
4. 更新文档。
   - 验证：README、overview、dev index 均说明新能力。

## 边界

- 不做 toolset。
- 不做平台级工具配置。
- 不改变已有工具实现。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/config ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 手动验证 `tools disable/list/enable`

### 来源：`060-plan-cli-sessions.md`

# 060 Plan：CLI 会话列表与检索

## 任务

1. 在 `internal/store` 增加最近会话列表查询。
2. 在 `cmd/agentd` 增加 `sessions list/search`。
3. 补单元测试覆盖列表排序。
4. 更新 README 与 docs 索引。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/store ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`

### 来源：`061-plan-cli-session-show-stats.md`

# 061 Plan：CLI 会话详情查看与统计

## 任务

1. `cmd/agentd` 增加 `sessions show/stats` 子命令与 usage。
2. `internal/store` 增加单元测试覆盖 `LoadMessagesPage` 与 `SessionStats` 的基础行为。
3. 更新 README 示例与 `docs/dev/README.md` 索引。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 手动：
  - `go run ./cmd/agentd sessions stats <session_id>`
  - `go run ./cmd/agentd sessions show -offset 0 -limit 50 <session_id>`

### 来源：`062-plan-hermes-cron-alignment.md`

# 062 计划：Hermes Cron 最小对齐（interval/one-shot）

## 目标（可验证）

- `cronjob` 工具可用：`create/list/get/pause/resume/remove/trigger`。
- 开启 `AGENT_CRON_ENABLED=true` 后，调度器会周期性扫描 due job 并触发独立 session 的 agent run。
- cron job 与 run 结果可持久化在 SQLite（与 `sessions.db` 同库）。
- 文档对齐矩阵更新：Cron 从“未覆盖”更新为“部分对齐”，并写明边界。

## 实施步骤

1. **存储**
   - 新增 `cron_jobs`、`cron_runs` 表，复用现有 SQLite 连接。
2. **调度器**
   - ticker 扫描 due jobs；按并发度执行；对 interval/once 计算 next_run_at。
3. **工具**
   - 新增内置工具 `cronjob`，action 压缩 schema。
4. **集成**
   - `serve` 与 `chat` 模式按配置启动 scheduler。
5. **文档**
   - 更新 `README.md`、`docs/overview-product*.md` 与 `docs/dev/README.md` 索引。
6. **测试**
   - schedule 解析与 cron store CRUD 单测（无网络/无端口依赖）。

## 不在本次范围

- cron 表达式执行
- 平台投递与 origin 捕获
- prompt threat scanning
- `no_agent` 脚本作业、context_from 链式作业

### 来源：`063-plan-hermes-toolsets-alignment.md`

# 063 计划：Hermes Toolsets 最小对齐

## 目标（可验证）

- `tools.enabled_toolsets` / `AGENT_ENABLED_TOOLSETS` 可限制 registry 仅保留解析后的工具集合。
- `agentd toolsets list` 输出内置 toolsets。
- `agentd toolsets resolve core` 输出 core toolset 展开后的工具名列表。
- 文档对齐矩阵更新：toolsets 从“未覆盖”调整为“部分对齐”。

## 实施步骤

1. 新增 `internal/tools/toolsets.go`：toolset 定义 + includes 解析。
2. 新增配置项：`tools.enabled_toolsets`（env：`AGENT_ENABLED_TOOLSETS`）。
3. Engine 构建时应用 toolset 过滤（先 enabled，再 disabled）。
4. 新增 CLI：`agentd toolsets list|resolve`。
5. 更新文档与索引，补单测。

### 来源：`064-plan-hermes-send-message-alignment.md`

# 064 计划：Hermes send_message 最小对齐

## 目标（可验证）

- `send_message(action='list')` 返回当前进程已连接的 gateway adapters。
- `send_message(action='send', platform, chat_id, message)` 可投递文本消息。
- Gateway runner 会在 adapter connect/disconnect 时注册/注销 adapter。
- 文档对齐矩阵更新：Gateway/toolsets 标记调整，补 docs/dev 索引。

## 实施步骤

1. 解耦 adapter 接口到 `internal/platform`。
2. 新增运行时 adapter registry。
3. Gateway runner hook：connect 后 register，退出前 unregister。
4. 新增工具 `send_message` 并注册到 engine。
5. 更新 toolsets `messaging` + `core` includes。
6. 补单测与文档。

### 来源：`065-plan-hermes-patch-tool-alignment.md`

# 065 计划：Hermes patch 工具最小对齐

## 目标（可验证）

- 新增内置工具 `patch`，并纳入 `file` toolset。
- `patch` 受 `AGENT_WORKDIR` 限制，避免越权写文件。
- 单测覆盖单次替换与多匹配保护策略。

## 实施步骤

1. 内置工具注册 `patch`。
2. 实现替换逻辑（与 `skill_manage patch` 一致的最小语义）。
3. toolsets `file` 增加 `patch`。
4. 更新文档与索引。

### 来源：`066-plan-hermes-web-tools-alignment.md`

# 066 计划：Hermes web tools 最小对齐

## 目标（可验证）

- 新增内置工具：`web_search`、`web_extract`。
- `toolsets.web` 默认包含 `web_search/web_extract`（保留 `web_fetch` 兼容）。
- 单测覆盖：DDG 结果解析与 HTML->text 抽取基础行为。

## 实施步骤

1. 在 builtin tools 中注册并实现 `web_search/web_extract`。
2. 新增最小解析与清洗 helper。
3. 更新 toolsets/web。
4. 更新 docs 与 `docs/dev/README.md` 索引。

### 来源：`067-plan-hermes-clarify-tool-alignment.md`

# 067 计划：Hermes clarify 工具最小对齐

## 目标（可验证）

- 新增内置工具 `clarify`，并纳入 `toolsets.core`。
- `clarify` 对空 question 报错；对 options 做最小校验与清洗。
- 更新 docs/dev 索引与工具清单。

## 实施步骤

1. 在 builtin tools 中注册并实现 `clarify`。
2. 在 toolsets 中新增 `clarify` toolset，并让 core includes 它。
3. 更新文档与索引。

### 来源：`068-plan-hermes-execute-code-alignment.md`

# 068 计划：Hermes execute_code 最小对齐

## 目标（可验证）

- 新增工具 `execute_code`，可运行 python 片段并返回 stdout/stderr/exit code。
- 限制在 workdir 内执行，支持超时。
- 单测覆盖基础执行成功路径。

## 实施步骤

1. 新增 `internal/tools/execute_code.go` 并注册到 engine。
2. toolsets 增加 `code_execution`（默认不纳入 core）。
3. 更新 docs/dev 索引与工具清单。

### 来源：`207-plan-frontend-tui-parity-phase1.md`

# Frontend 与 TUI 对齐计划（Phase 1）

## 任务拆解

1. 增强 CLI 交互命令面  
验证：`internal/cli` 单测通过，`/help` 等命令具备稳定输出。

2. 新建 `web/` Vite + React 工程骨架  
验证：目录结构完整，`npm run dev` 可启动，Chat 页可请求 `/v1/chat`。

3. 同步产品/开发文档  
验证：`docs/overview-product.md`、`docs/overview-product-dev.md` 有前端/TUI 章节并与实现一致。

4. 更新 `docs/dev/README.md` 索引并形成阶段总结  
验证：索引可检索到 207 系列文档。

## 风险与应对

1. 仓库当前无 Node 构建流水线  
应对：本阶段只落地工程与运行说明，不强行将 Node 构建纳入 Go 测试流水线。

2. “一次性彻底对齐”范围过大  
应对：采用多批次迭代，每批可运行、可验证、可提交。
