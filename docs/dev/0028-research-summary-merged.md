# 0028 research summary merged

## 模块

- `research`

## 类型

- `summary`

## 合并来源

- `0037-research-summary-merged.md`

## 合并内容

### 来源：`0037-research-summary-merged.md`

# 0037 research summary merged

## 模块

- `research`

## 类型

- `summary`

## 合并来源

- `001-research-hermes-agent-go-port.md`
- `002-research-hermes-gap-closure.md`
- `003-research-context-compression.md`
- `004-research-approval-guardrails.md`
- `005-research-provider-modes.md`
- `006-research-codex-responses-mode.md`
- `007-research-mcp-minimal-bridge.md`
- `008-research-skills-minimal-skeleton.md`
- `009-research-session-approval-state.md`
- `010-research-skill-manage-minimal.md`
- `011-research-skill-manage-support-files.md`
- `012-research-mcp-stdio-bridge.md`
- `013-research-mcp-oauth-client-credentials.md`
- `014-research-mcp-call-streaming-compat.md`
- `015-research-provider-fallback-minimal.md`
- `016-research-provider-streaming-openai-minimal.md`
- `017-research-provider-streaming-anthropic-minimal.md`
- `018-research-provider-streaming-codex-minimal.md`
- `019-research-provider-stream-events-passthrough.md`
- `020-research-model-stream-event-schema-v1.md`
- `021-research-model-stream-event-schema-v2.md`
- `022-research-model-stream-event-schema-v2-args-lifecycle.md`
- `023-research-model-stream-event-schema-v2-usage.md`
- `024-research-model-stream-event-schema-v2-finish-reason-and-id-aliases.md`
- `025-research-model-stream-event-schema-v2-id-source-compat.md`
- `0259-research-trajectory-runtime-minimal.md`
- `026-research-model-stream-event-schema-v2-termination-metadata.md`
- `027-research-model-stream-event-schema-v2-finish-incomplete-consistency.md`
- `028-research-model-stream-event-schema-v2-usage-cache-tokens.md`
- `029-research-model-stream-event-schema-v2-usage-reasoning-tokens.md`
- `030-research-model-stream-event-schema-v2-usage-total-consistency.md`
- `031-research-model-stream-event-schema-v2-usage-consistency-status.md`
- `032-research-model-stream-event-schema-v2-usage-status-invalid.md`
- `033-research-model-stream-event-schema-v2-usage-status-provider-coverage.md`
- `034-research-model-stream-event-schema-v2-usage-status-source-only-coverage.md`
- `035-research-model-stream-event-schema-v2-usage-status-e2e-provider-streaming.md`
- `036-research-model-stream-event-schema-v2-usage-status-adjusted-e2e.md`
- `037-research-model-stream-event-schema-v2-usage-status-adjusted-e2e-anthropic.md`
- `038-research-model-stream-event-schema-v2-usage-status-table-driven.md`
- `039-research-provider-race-circuit.md`
- `040-research-provider-event-coverage.md`
- `041-research-approval-persistence.md`
- `042-research-mcp-streaming-passthrough.md`
- `043-research-mcp-oauth-auth-code.md`
- `044-research-gateway-minimal.md`
- `045-research-skills-adv-trigger-sync.md`
- `053-research-hermes-feature-alignment.md`
- `054-research-cli-config-management.md`
- `055-research-cli-model-management.md`
- `056-research-cli-tools-inspection.md`
- `057-research-cli-doctor.md`
- `058-research-cli-gateway-management.md`
- `059-research-tool-disable-config.md`
- `060-research-cli-sessions.md`
- `061-research-cli-session-show-stats.md`
- `062-research-hermes-cron-alignment.md`
- `063-research-hermes-toolsets-alignment.md`
- `064-research-hermes-send-message-alignment.md`
- `065-research-hermes-patch-tool-alignment.md`
- `066-research-hermes-web-tools-alignment.md`
- `067-research-hermes-clarify-tool-alignment.md`
- `068-research-hermes-execute-code-alignment.md`
- `069-research-hermes-read-file-alignment.md`
- `207-research-frontend-tui-parity.md`

## 合并内容

### 来源：`001-research-hermes-agent-go-port.md`

# 001 调研：Hermes Agent 架构与 Go 实现映射

## 任务背景

目标是调研 `/data/source/hermes-agent` 项目中的 Agent 实现思路，形成完整文档，并在 `/data/code/agent-daemon/` 中以 Go 实现其完整核心功能。

由于 Hermes 是完整生态系统，覆盖：

- CLI / Gateway / ACP 多入口
- 多 provider / 多 API mode
- 工具注册中心与 60+ 工具
- 会话状态、长期记忆、技能、上下文压缩
- 多平台消息网关、插件、MCP、RL 环境

因此本次调研先抽取其“Agent 核心闭环”和“最小完整运行集合”。

## Hermes 核心结论

### 1. 核心对象是 `AIAgent`

Hermes 的核心 orchestrator 位于 `run_agent.py` 的 `AIAgent`。

其职责包括：

- 构建系统提示词
- 选择 provider / API mode
- 发起模型调用
- 解析 `tool_calls`
- 顺序或并发执行工具
- 将工具结果追加回消息历史
- 持久化 session
- 管理重试、回退与中断

### 2. 统一消息模型是主轴

Hermes 内部统一使用 OpenAI 风格消息：

- `system`
- `user`
- `assistant`
- `tool`

这是它能兼容不同 provider，同时又统一工具调用逻辑的关键。

### 3. 工具系统采用“注册中心 + 动态 schema”

Hermes 的 `tools/registry.py` 和 `model_tools.py` 负责：

- 发现工具
- 注册 schema
- 检查工具可用性
- 统一分发工具调用
- 统一错误封装

这是 Go 版本必须保留的骨架。

### 4. Agent Loop 的本质是 while 循环

Hermes 的运行过程可概括为：

1. 组装消息与工具 schema
2. 调用模型
3. 若模型返回文本且无工具调用，则结束
4. 若返回 `tool_calls`，则执行工具
5. 把工具结果作为 `tool` 消息追加进历史
6. 继续下一轮模型调用
7. 达到最大轮次则停止

### 5. 状态是分层保存的

Hermes 并不把所有状态都混在一起，而是拆成：

- 会话历史：结构化、可检索
- 长期记忆：跨 session 持久化
- todo / 当前工作状态：Agent 局部状态

### 6. terminal 工具是 Hermes 的关键能力

Hermes 中 terminal 能力不仅是执行 shell，还包含：

- 前台/后台任务
- 危险命令审批
- 多后端环境
- 进程状态追踪

Go 版本首期抽取其中最核心的：

- 本地 Linux 前台命令
- 本地 Linux 后台命令
- 进程状态轮询与停止

## Go 版设计映射

### 保留项

- 统一消息模型
- Tool registry
- OpenAI 兼容 `tool_calls`
- Session persistence
- Memory persistence
- CLI 与 HTTP 双入口
- 前台/后台 terminal

### 延后项

- 多 API mode（Codex Responses、Anthropic Messages）
- MCP
- Skills
- Context compression
- Gateway 多平台适配
- delegate_task 并发子 Agent
- 审批系统与复杂安全护栏

## 结论

Hermes 最值得复用的不是它的 Python 代码细节，而是它的架构骨架：

- OpenAI 风格消息内核
- 工具注册中心
- 多轮 tool-calling loop
- 状态分层持久化
- 入口层与核心层解耦

Go 版已经按这个骨架实现，可继续向 Hermes 的外围能力扩展。

### 来源：`002-research-hermes-gap-closure.md`

# 002 调研：Hermes 核心闭环差异补齐

## 背景

`/data/code/agent-daemon` 已实现 Hermes 风格的基础 Agent Loop、工具注册中心、会话持久化、CLI/API 双入口与结构化事件流。

但将当前实现与 `/data/source/hermes-agent` 的核心源码对照后，仍存在几处会影响“闭环完整性”的关键差异。

## 源码级差异

### 1. 系统提示词没有跨请求持续生效

当前 `internal/agent/loop.go` 仅在 `existing` 为空时追加 system message。

而会话历史从 `internal/store/session_store.go` 读取时并不包含 system message，这会导致：

- 第一轮请求后，后续请求丢失系统提示词
- CLI / HTTP 多次调用同一 session 时行为不稳定
- 与 Hermes 持续重建系统提示词的方式不一致

这属于核心闭环缺口，必须补齐。

### 2. 持久记忆可写不可读，未回灌到后续推理

当前 `internal/memory/store.go` 只实现 `Manage()` 写入 `MEMORY.md` / `USER.md`。

但 `internal/agent/loop.go` 没有在运行前加载这些内容，因此：

- `memory` 工具写入的信息不会影响后续 session
- “长期记忆”只有存储层，没有推理层闭环

相比 Hermes 的 memory manager / prompt builder，这也是明显缺口。

### 3. 工作区规则未进入系统提示词

Hermes 会把 `AGENTS.md`、上下文文件与环境提示拼进系统提示词。

当前 Go 版 `internal/agent/prompt.go` 只有固定两行默认提示，无法把项目级约束传递给模型。这会降低仓库内执行的一致性，也与项目自身的 `AGENTS.md` 工作流不匹配。

### 4. 工具侧缺少基础安全护栏

Hermes 在核心工具层有：

- 路径安全约束
- 危险命令识别/拦截
- URL / skill / tool guardrails

当前 Go 版的 `read_file`、`write_file`、`search_files`、`terminal` 直接对输入执行：

- 文件工具可越过工作区访问任意路径
- terminal 缺少最基础的灾难性命令阻断

完整审批系统仍属后续扩展，但工作区边界与硬阻断护栏属于当前应补齐的核心安全基线。

## 范围判断

本次“补齐为止”的目标定义为：补齐 Hermes 核心 Agent 闭环所必需的缺口，而不是一次性复刻其外围生态。

本次纳入范围：

- 系统提示词跨请求稳定注入
- 持久记忆回灌
- 工作区规则注入
- 文件路径安全约束
- 危险命令硬阻断

暂不纳入本次范围：

- Context Compression
- Skills 系统
- MCP
- 多 provider API mode
- 多平台 Gateway
- 完整审批系统

## 结论

当前项目离 Hermes 的“完整核心闭环”只差最后一层运行时装配与工具护栏。

优先补齐提示词装配、记忆回灌和基础安全边界后，可认为 Go 版已经真正闭合以下链路：

- system prompt / workspace rules
- memory persistence / memory reuse
- session history / multi-turn continuation
- tool execution / workspace-safe operations

### 来源：`003-research-context-compression.md`

# 003 调研：Context Compression 补齐

## 背景

在 002 阶段之后，Go 版已补齐 system prompt、memory 回灌、工作区规则与基础安全护栏。

剩余核心体验差异中，优先级最高的是长会话场景下的上下文膨胀问题：历史消息与工具输出持续增长，会带来请求失败风险和成本上升。

Hermes 在 `agent/context_compressor.py` 里通过中段压缩、头尾保护、摘要前缀和预算控制解决这一问题。

## 当前项目缺口

- 无上下文预算控制
- 无自动压缩触发
- 无“压缩后摘要消息”机制
- 无压缩可观测事件

这会导致长 session 下的 `messages` 不断变大。

## 方案选择

本次采用“最小可用压缩器”而非一次性复刻 Hermes 的辅助模型摘要器：

- 使用字符预算估算上下文体积（`max_context_chars`）
- 保留 system message 与最近 N 条消息（`compression_tail_messages`）
- 将中段历史压缩为一条 assistant 摘要消息
- 输出 `context_compacted` 结构化事件

不在本次引入：

- 额外 summarizer 模型调用
- 多轮迭代摘要合并策略
- 多模态精细 token 估算

## 结论

该方案能在不新增外部依赖和模型调用的前提下，立即补齐长会话核心能力，并与现有 Engine 架构自然集成。

### 来源：`004-research-approval-guardrails.md`

# 004 调研：审批护栏补齐

## 背景

在 002 阶段已补齐 hardline 灾难命令硬阻断，在 003 阶段补齐了上下文压缩。

剩余差异中，“完整审批系统”依然缺失：当前 `terminal` 工具除 hardline 外，对高风险命令缺少显式审批门禁。

## 差异点

对照 Hermes 的 `tools/approval.py`，其能力包含：

- 危险命令识别
- 审批状态管理
- 交互式审批回调
- allowlist 持久化

当前 Go 版尚不具备交互审批链路，但可以先补齐“危险命令 + requires_approval 门禁”这一核心护栏。

## 本次范围

纳入：

- 新增危险命令识别（可审批）
- `terminal` 新增 `requires_approval` 参数
- 危险命令默认拒绝，显式 `requires_approval=true` 时放行
- hardline 命令始终阻断，不可通过审批放行

不纳入：

- 交互式审批 UI/回调
- 会话级审批状态机
- 审批白名单持久化

## 结论

该补齐能在不引入复杂交互链路的前提下，完成“硬阻断 + 可审批门禁”的分层安全模型，是当前阶段最小且必要的落地方式。

### 来源：`005-research-provider-modes.md`

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

### 来源：`006-research-codex-responses-mode.md`

# 006 调研：Codex Responses 模式补齐

## 背景

在 005 阶段已补齐 OpenAI + Anthropic 双模式，但 Hermes 侧仍有 Responses/Codex 兼容能力。

为继续缩小核心差距，本阶段补齐 `provider=codex` 的最小可用模式。

## 差异点

- 当前无 `/responses` 接口 client
- 无 `function_call` 与 `function_call_output` 映射
- 无 codex provider 配置项

## 方案

保持 `model.Client` 不变，新增 `CodexClient`：

- 请求路径：`/responses`
- 输入映射：
  - user/assistant/system 消息映射为 `type=message`
  - assistant 工具调用映射为 `type=function_call`
  - tool 消息映射为 `type=function_call_output`
- 输出映射：
  - `message` 内容反解到 `assistant.content`
  - `function_call` 反解到 `assistant.tool_calls`

## 结论

该实现可在不改 Agent Loop 的前提下，把 provider 侧能力扩展到 OpenAI/Anthropic/Codex 三模式，为后续继续补齐 provider 级高级特性打下基础。

### 来源：`007-research-mcp-minimal-bridge.md`

# 007 调研：MCP 最小接入骨架

## 背景

当前项目工具系统已具备本地工具注册与分发能力，但尚未支持通过 MCP 生态接入外部工具。

在不引入复杂 stdio/OAuth/会话管理的前提下，本阶段先补齐可用骨架：HTTP 发现 + HTTP 调用转发。

## 本次范围

- 支持从 MCP 端点发现工具 schema（`GET /tools`）
- 将发现到的工具动态注册到本地 `Registry`
- 支持将工具调用转发到 MCP 端点（`POST /call`）
- 调用时传递最小上下文：`session_id`、`workdir`

不纳入：

- stdio transport
- OAuth / 鉴权协商
- 复杂会话绑定和流式调用

## 结论

该方案能以最小成本打通 “MCP 工具发现 -> 注册 -> 调用” 主链路，为后续接入完整 MCP 协议能力提供稳定起点。

### 来源：`008-research-skills-minimal-skeleton.md`

# 008 调研：Skills 最小骨架补齐

## 背景

当前项目已经具备工具注册中心、MCP 代理和多 provider，但仍缺少本地技能系统入口。

Hermes 的 Skills 体系较完整（发现、调用、管理、更新），本阶段先补齐最小可用能力：本地技能发现与查看。

## 差异点

- 当前无技能目录发现能力
- 当前无技能查看工具
- 当前无技能内容注入入口

## 本次范围

- 新增 `skill_list`：列出本地 `skills/*/SKILL.md`
- 新增 `skill_view`：按技能名读取 `SKILL.md`
- 做好路径安全限制，禁止路径穿越

不纳入：

- `skill_manage` 创建/编辑
- skill 执行脚本与依赖管理
- skills hub 同步

## 结论

该骨架能够打通“技能可见性”闭环，为后续补齐技能管理与自动技能调用提供基础。

### 来源：`009-research-session-approval-state.md`

# 009 调研：会话级审批状态补齐

## 背景

当前项目已有：

- hardline 永久阻断
- `requires_approval` 单次显式放行

但仍缺少会话级审批状态，导致同一会话中的多个危险命令需要反复携带审批参数，交互成本较高。

## 本次范围

- 新增会话级审批状态存储（内存态 + TTL）
- 新增 `approval` 工具：
  - `status`
  - `grant`
  - `revoke`
- terminal 在危险命令判定时读取会话审批状态

不纳入：

- 跨进程持久化审批状态
- 用户侧 UI 交互确认流
- 命令级细粒度审批策略

## 结论

该方案可在不改变现有工具协议和 API 结构的前提下，实现最低成本的交互式审批状态闭环。

### 来源：`010-research-skill-manage-minimal.md`

# 010 调研：`skill_manage` 最小补齐

## 背景

当前项目已具备：

- `skill_list`：列出本地技能
- `skill_view`：读取单个技能

但仍缺少 Hermes 中由 Agent 直接维护技能内容的关键入口，即 `skill_manage`。这会导致技能体系只能“读”，不能“写”。

## 与 Hermes 的差异

Hermes 的 `skill_manage` 支持较完整生命周期（create/edit/patch/delete/write_file/remove_file）以及更复杂的安全、审计与策略。

当前 Go 版差异集中在：

- 无技能创建能力
- 无技能修改能力
- 无技能删除能力
- 无“定点 patch”能力

## 本轮补齐目标（最小可用）

在保持现有工作区安全边界与简洁实现的前提下，先补齐最小管理闭环：

- 新增 `skill_manage` 工具
- 支持动作：`create` / `edit` / `patch` / `delete`
- 操作范围限制在 `<workdir>/skills`（或工具参数指定的受限子路径）
- 技能名校验，拒绝路径穿越与目录分隔符
- `patch` 默认要求唯一匹配，`replace_all=true` 才允许批量替换

## 明确暂不覆盖

为了避免一次性引入过高复杂度，本轮不做：

- `write_file` / `remove_file` 支撑文件管理
- Skills hub 同步与来源追踪
- 技能使用统计、pin/curator 联动
- 自动触发与策略评分

### 来源：`011-research-skill-manage-support-files.md`

# 011 调研：`skill_manage` 支撑文件能力补齐

## 背景

010 已补齐 `skill_manage` 的基础生命周期（`create/edit/patch/delete`），但仍有明显缺口：

- 无法为技能写入 `references/templates/scripts/assets` 等支撑文件
- 无法删除技能内无用支撑文件

这会限制技能从“单文件说明”演进为“可复用资产包”。

## 与 Hermes 差异

Hermes 的 `skill_manage` 已支持：

- `write_file`
- `remove_file`

并且对路径有子目录约束，防止越权写入。

## 本轮最小补齐目标

- 在 Go 版 `skill_manage` 增加：
  - `write_file`
  - `remove_file`
- 路径约束：
  - 必须为相对路径
  - 禁止路径穿越
  - 必须落在允许子目录：`references/`、`templates/`、`scripts/`、`assets/`
- 保持所有写删操作仍受工作区边界约束。

### 来源：`012-research-mcp-stdio-bridge.md`

# 012 调研：MCP `stdio` 最小桥接

## 背景

当前 Go 版 MCP 已支持 HTTP 桥接（`/tools` + `/call`），但 Hermes 侧还覆盖 `stdio` 场景。  
这导致本项目无法直接接入仅暴露标准输入输出通道的 MCP Server。

## 差异点

- 已有：HTTP MCP 发现与调用
- 缺失：`stdio` MCP 发现与调用

## 本轮目标

在不破坏现有 HTTP 桥接的前提下，补一个最小 `stdio` 实现：

- 配置可切换 MCP 传输模式（`http` / `stdio`）
- `stdio` 会话支持最小 JSON-RPC 流程：
  - `initialize`
  - `notifications/initialized`
  - `tools/list`
  - `tools/call`
- 保持“发现工具 -> 注册代理 -> 调用透传”的现有结构不变

## 本轮边界

- 不做 OAuth
- 不做 streaming 增量消息消费
- 不做跨调用持久连接池（采用一次调用一会话的简化策略）

### 来源：`013-research-mcp-oauth-client-credentials.md`

# 013 调研：MCP OAuth（Client Credentials）最小补齐

## 背景

012 已补齐 MCP `stdio`，但 MCP HTTP 仍缺少 OAuth 能力，无法接入要求 Bearer Token 的 MCP 服务。

## 缺口

- HTTP MCP 请求无 `Authorization` 注入
- 无 token 申请与缓存机制

## 本轮目标

补齐最小 OAuth 能力（`client_credentials`）：

- 通过 token endpoint 获取 access token
- 为 MCP `/tools` 与 `/call` 自动注入 `Authorization: Bearer ...`
- 在内存中缓存 token，避免每次请求都换 token

## 本轮边界

- 不做授权码模式
- 不做 refresh_token 流程
- 不做多租户/多会话 token 隔离（当前进程级单配置）

### 来源：`014-research-mcp-call-streaming-compat.md`

# 014 调研：MCP `/call` 流式响应兼容

## 背景

013 已补齐 MCP HTTP OAuth（client_credentials），但 `/call` 仍默认按一次性 JSON 响应处理。  
部分 MCP 服务会通过 `text/event-stream` 分块返回结果，当前实现无法消费。

## 缺口

- MCP `/call` 缺少 SSE 解析逻辑
- 流式返回下无法得到可用的最终工具结果

## 本轮目标

在保持现有同步调用接口不变的前提下，补齐最小流式兼容：

- 当 `Content-Type` 为 `text/event-stream` 时，解析 SSE `data:` 事件
- 支持常见事件形态：
  - `{"result": {...}}`
  - `{"structuredContent": {...}}`
  - `{"error": {...}}`
  - `[DONE]`
- 聚合为最终 `map[string]any` 返回给现有工具链路

## 本轮边界

- 不做增量事件透传到 Agent EventSink
- 不做多路并发流内控制协议

### 来源：`015-research-provider-fallback-minimal.md`

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

### 来源：`016-research-provider-streaming-openai-minimal.md`

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

### 来源：`017-research-provider-streaming-anthropic-minimal.md`

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

### 来源：`018-research-provider-streaming-codex-minimal.md`

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

### 来源：`019-research-provider-stream-events-passthrough.md`

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

### 来源：`020-research-model-stream-event-schema-v1.md`

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

### 来源：`021-research-model-stream-event-schema-v2.md`

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

### 来源：`022-research-model-stream-event-schema-v2-args-lifecycle.md`

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

### 来源：`023-research-model-stream-event-schema-v2-usage.md`

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

### 来源：`024-research-model-stream-event-schema-v2-finish-reason-and-id-aliases.md`

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

### 来源：`025-research-model-stream-event-schema-v2-id-source-compat.md`

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

### 来源：`0259-research-trajectory-runtime-minimal.md`

# 259 总结：Research/RL/Trajectory 最小运行时闭环

本次补齐了 Research/RL/Trajectory 的最小可用主链路，目标是“可批量执行任务并产出可压缩轨迹数据”。

## 新增能力

- `agentd research run`
  - 输入：`-tasks <jsonl>`
  - 每行任务结构：`{input, session_id?, id?, metadata?}`
  - 输出：trajectory `jsonl`（包含事件、结果、耗时、错误）
- `agentd research compress`
  - 输入 trajectory `jsonl`
  - 输出压缩后的 `compact jsonl.gz`（保留训练关键字段，裁剪长文本）
- `agentd research stats`
  - 统计 trajectory 文件的总量、成功数、失败数、平均耗时

## 实现

- `internal/research/batch.go`
  - `LoadTasks`
  - `RunBatch`
  - `CompressTrajectories`
  - `StatsTrajectories`
- `cmd/agentd/main.go`
  - 新增 `research` 子命令路由与 `run/compress/stats` 三个子命令

## 测试

- `internal/research/batch_test.go`
  - 任务加载
  - 轨迹压缩与统计

验证：

- `go test ./internal/research -count=1`
- `go test ./...`

### 来源：`026-research-model-stream-event-schema-v2-termination-metadata.md`

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

### 来源：`027-research-model-stream-event-schema-v2-finish-incomplete-consistency.md`

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

### 来源：`028-research-model-stream-event-schema-v2-usage-cache-tokens.md`

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

### 来源：`029-research-model-stream-event-schema-v2-usage-reasoning-tokens.md`

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

### 来源：`030-research-model-stream-event-schema-v2-usage-total-consistency.md`

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

### 来源：`031-research-model-stream-event-schema-v2-usage-consistency-status.md`

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

### 来源：`032-research-model-stream-event-schema-v2-usage-status-invalid.md`

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

### 来源：`033-research-model-stream-event-schema-v2-usage-status-provider-coverage.md`

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

### 来源：`034-research-model-stream-event-schema-v2-usage-status-source-only-coverage.md`

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

### 来源：`035-research-model-stream-event-schema-v2-usage-status-e2e-provider-streaming.md`

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

### 来源：`036-research-model-stream-event-schema-v2-usage-status-adjusted-e2e.md`

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

### 来源：`037-research-model-stream-event-schema-v2-usage-status-adjusted-e2e-anthropic.md`

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

### 来源：`038-research-model-stream-event-schema-v2-usage-status-table-driven.md`

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

### 来源：`039-research-provider-race-circuit.md`

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

### 来源：`040-research-provider-event-coverage.md`

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

### 来源：`041-research-approval-persistence.md`

# 041 调研：审批状态持久化与细粒度审批策略

## 背景

当前审批系统已有：
- hardline 永久阻断（不可放行）
- `requires_approval` 单次显式放行
- 会话级审批状态（内存态 + TTL）
- `approval` 工具（`status`/`grant`/`revoke`）

但存在两个核心缺口：
1. **审批状态仅内存态**：进程重启后所有审批授权丢失，需要重新 grant
2. **审批粒度仅到 session 级**：grant 后该 session 所有危险命令均可执行，无法按命令模式或工具类型细粒度控制

## 当前实现分析

### ApprovalStore（内存态）

```go
type ApprovalStore struct {
    mu         sync.Mutex
    items      map[string]time.Time  // sessionID -> expiresAt
    defaultTTL time.Duration
}
```

- 按 `sessionID` 授权，粒度为整个会话
- 无持久化，进程重启丢失
- TTL 过期自动失效

### terminal 审批判定

```go
if reason, dangerous := detectDangerousCommand(command); dangerous {
    approved := tc.ApprovalStore != nil && tc.ApprovalStore.IsApproved(tc.SessionID)
    if !requiresApproval && !approved {
        return nil, fmt.Errorf("dangerous command requires approval: %s ...", reason)
    }
}
```

- 仅检查 session 级授权，不区分危险命令类型
- `requires_approval=true` 可单次放行并自动 grant session 级授权

## 缺口清单

1. **审批持久化**：授权记录应持久化到 SQLite，进程重启后可恢复
2. **命令模式级审批**：可按危险命令类别（如 `recursive_delete`、`world_writable`、`remote_pipe_shell`）授权，而非全量放行
3. **审批记录审计**：授权/撤销/使用记录可追溯

## 方案

### 1. 审批持久化

在现有 `sessions.db` 中新增 `approvals` 表：

```sql
CREATE TABLE IF NOT EXISTS approvals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'session',
    pattern TEXT NOT NULL DEFAULT '',
    granted_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
```

- `scope`：`session`（会话级）或 `pattern`（命令模式级）
- `pattern`：当 `scope=pattern` 时，存储危险命令类别（如 `recursive_delete`）
- `expires_at`：过期时间，过期记录在查询时惰性清理

### 2. 命令模式级审批

扩展 `approval` 工具：
- `grant` 新增可选 `scope` 和 `pattern` 参数
- `scope=session`（默认）：行为与当前一致
- `scope=pattern`：仅授权指定类别的危险命令

扩展 `detectDangerousCommand` 返回值，使其返回类别标识符（如 `recursive_delete`、`world_writable`）。

terminal 审批判定逻辑：
1. 先检查 session 级授权
2. 若无 session 级授权，检查 pattern 级授权（匹配命令类别）
3. 若均无，拒绝执行

### 3. 审批记录审计

`approval` 工具 `status` 返回当前有效授权列表，包含 scope 和 pattern。

## 取舍

- 不做用户侧交互确认 UI（保持工具级控制）
- 不做跨进程审批策略同步（单进程 SQLite 足够）
- 不做审批策略热重载（重启加载即可）
- 保持向后兼容：`scope=session` 行为与当前完全一致

### 来源：`042-research-mcp-streaming-passthrough.md`

# 042 调研：MCP 流式事件透传

## 背景

014 已补齐 MCP `/call` 的 SSE 兼容——当 MCP 服务返回 `text/event-stream` 时，`parseMCPCallSSE` 会聚合所有事件为最终结果一次性返回。但这意味着：

- 客户端无法实时感知 MCP 工具的执行进度
- 长时间运行的 MCP 工具（如代码分析、文件索引）在 SSE 模式下表现为"黑盒等待"
- Agent 事件总线已有 `model_stream_event` 透传模型层增量事件，但 MCP 工具层没有类似机制

## 当前架构分析

### MCP 调用链路

```
Agent Loop → Registry.Dispatch → mcpToolProxy.Call → HTTP POST /call → parseMCPCallSSE → 聚合返回
```

- `mcpToolProxy.Call` 返回 `(map[string]any, error)`，是同步接口
- `parseMCPCallSSE` 聚合所有 SSE 事件后才返回
- `Registry.Dispatch` 将结果序列化为 JSON string 返回给 Agent Loop
- Agent Loop 在 `tool_started` 和 `tool_finished` 之间无法感知中间进度

### Agent 事件系统

- `Engine.EventSink` 接收 `core.AgentEvent`
- `model_stream_event` 已支持模型层增量事件透传
- 工具层仅有 `tool_started` / `tool_finished`，无中间事件

### Tool 接口

```go
type Tool interface {
    Name() string
    Schema() core.ToolSchema
    Call(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error)
}
```

- `ToolContext` 当前无事件回调能力
- `Registry.Dispatch` 返回 `string`，无法传递中间状态

## 缺口

1. MCP SSE 中间事件被丢弃，客户端无法实时感知 MCP 工具执行进度
2. `ToolContext` 无事件回调，工具无法向 Agent 事件总线发送中间事件
3. Agent Loop 在工具执行期间无法发出增量事件

## 本轮目标

在保持现有同步调用接口不变的前提下，让 MCP SSE 中间事件透传到 Agent 事件总线：

1. 扩展 `ToolContext` 增加可选事件回调 `ToolEventSink`
2. MCP `/call` SSE 模式下，每解析到一个事件就通过回调发送
3. Agent Loop 在工具执行时注册回调，将 MCP 中间事件映射为 `mcp_stream_event`
4. SSE 客户端可直接消费 `mcp_stream_event`，实现实时进度感知

## 本轮边界

- 不改变 `Tool.Call` 的同步返回语义（最终结果仍由返回值提供）
- 不做 stdio MCP 的流式事件透传（stdio 使用 JSON-RPC 帧，无增量事件语义）
- 不做 MCP 会话绑定与多路并发控制
- 不引入新的 SSE 事件协议字段，复用现有 `AgentEvent` 结构

## 方案

### 1. 扩展 ToolContext

在 `ToolContext` 中新增可选字段：

```go
type ToolEventSink func(eventType string, data map[string]any)

type ToolContext struct {
    // ... existing fields
    ToolEventSink ToolEventSink // optional: emit intermediate tool events
}
```

### 2. MCP SSE 逐事件回调

改造 `mcpToolProxy.Call` 的 SSE 分支：

- 不再使用 `parseMCPCallSSE`（聚合模式）
- 新增 `parseMCPCallSSEWithCallback`，每解析到一个 SSE 事件就调用 `tc.ToolEventSink`
- 最终结果仍由函数返回值提供

### 3. Agent Loop 注册回调

在 `Engine.Run` 的工具执行循环中：

```go
tc := tools.ToolContext{
    // ... existing fields
    ToolEventSink: func(eventType string, data map[string]any) {
        e.emit(core.AgentEvent{
            Type:      "mcp_stream_event",
            SessionID: sessionID,
            Turn:      turn + 1,
            ToolName:  tc.Function.Name,
            Data: map[string]any{
                "tool_name":   tc.Function.Name,
                "event_type":  eventType,
                "event_data":  data,
            },
        })
    },
}
```

### 4. SSE 透传

`api/server.go` 的 SSE handler 已按 `event.Type` 透传所有 `AgentEvent`，新增的 `mcp_stream_event` 会自动透传到客户端，无需额外修改。

## 风险

- MCP SSE 事件格式无统一标准，不同 MCP 服务可能返回不同结构——透传时保留原始数据，不做标准化
- `ToolEventSink` 是可选的，不影响非 MCP 工具和现有测试

### 来源：`043-research-mcp-oauth-auth-code.md`

# 043 调研：MCP OAuth 授权码模式与刷新令牌

## 背景

013 已补齐 MCP OAuth `client_credentials` 模式，支持服务间认证。但许多 MCP 服务（如 GitHub、Google Drive、Slack 等）要求用户交互授权，即 OAuth 2.0 授权码模式（Authorization Code Flow）。当前实现无法接入这类服务。

此外，`client_credentials` 模式下 token 过期后只能重新申请，没有 `refresh_token` 刷新机制，导致频繁的 token 申请请求。

## 当前实现分析

### MCPOAuthConfig

```go
type MCPOAuthConfig struct {
    TokenURL     string
    ClientID     string
    ClientSecret string
    Scopes       string
}
```

- 仅支持 `client_credentials`
- 无 `AuthorizationURL`（授权端点）
- 无 `RedirectURL`（回调地址）
- 无 `refresh_token` 存储

### oauthAccessToken

```go
func (c *MCPClient) oauthAccessToken(ctx context.Context) (string, error) {
    // 仅支持 grant_type=client_credentials
    form.Set("grant_type", "client_credentials")
    // ...
    var tokenResp struct {
        AccessToken string `json:"access_token"`
        ExpiresIn   int    `json:"expires_in"`
    }
    // 不解析 refresh_token
}
```

- Token 过期后重新走 `client_credentials` 流程
- 不支持 `refresh_token` 刷新
- 不支持授权码模式

### 配置

```go
MCPOAuthTokenURL     string  // 仅 token endpoint
MCPOAuthClientID     string
MCPOAuthClientSecret string
MCPOAuthScopes       string
```

- 缺少 `AGENT_MCP_OAUTH_AUTH_URL`
- 缺少 `AGENT_MCP_OAUTH_REDIRECT_URL`

## 缺口清单

1. **授权码模式**：无法接入需要用户交互授权的 MCP 服务
2. **刷新令牌**：token 过期后只能重新申请，不支持 `refresh_token` 刷新
3. **令牌持久化**：`refresh_token` 应持久化到 SQLite，避免进程重启后丢失授权
4. **回调服务器**：授权码模式需要本地 HTTP 回调服务器接收授权码

## 本轮目标

在保持现有 `client_credentials` 模式不变的前提下，补齐：

1. **授权码模式**：支持 `grant_type=authorization_code`
2. **刷新令牌**：支持 `grant_type=refresh_token`
3. **令牌持久化**：`refresh_token` 持久化到 SQLite
4. **回调服务器**：启动时可选启动本地回调服务器

## 方案

### 1. 扩展 MCPOAuthConfig

```go
type MCPOAuthConfig struct {
    TokenURL      string
    AuthURL       string   // 新增：授权端点
    RedirectURL   string   // 新增：回调地址
    ClientID      string
    ClientSecret  string
    Scopes        string
    GrantType     string   // 新增：grant_type，默认 "client_credentials"，可选 "authorization_code"
}
```

### 2. 授权码流程

1. 启动时检测 `GrantType=authorization_code`
2. 启动本地回调服务器（默认 `localhost:9876/callback`）
3. 构造授权 URL，输出到日志/事件
4. 用户在浏览器中授权
5. 回调服务器接收授权码
6. 用授权码换取 access_token + refresh_token
7. 持久化 refresh_token 到 SQLite

### 3. 刷新令牌流程

1. `oauthAccessToken` 检测 token 过期
2. 如果有 `refresh_token`，用 `grant_type=refresh_token` 刷新
3. 刷新成功后更新缓存的 access_token
4. 刷新失败（如 refresh_token 已失效）则重新走授权码流程

### 4. 令牌持久化

在 `session_store.go` 新增 `oauth_tokens` 表：

```sql
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
```

### 5. 配置扩展

```bash
AGENT_MCP_OAUTH_GRANT_TYPE=authorization_code  # 默认 client_credentials
AGENT_MCP_OAUTH_AUTH_URL=https://github.com/login/oauth/authorize
AGENT_MCP_OAUTH_REDIRECT_URL=http://localhost:9876/callback
AGENT_MCP_OAUTH_CALLBACK_PORT=9876
```

## 本轮边界

- 不做 PKCE（Authorization Code + PKCE 适合无 client_secret 的 SPA/移动端，当前场景有 client_secret）
- 不做多 MCP 端点的独立 OAuth 配置（当前进程级单配置）
- 不做 OAuth token 的自动轮换策略
- 不做 CLI 交互式授权引导（仅输出授权 URL 到日志/事件）

## 风险

- 授权码模式需要用户在浏览器中操作，headless 环境下可能无法完成
- `refresh_token` 持久化到 SQLite 需要考虑加密存储（本轮暂不加密，标记为后续改进）
- 回调服务器端口可能被占用，需要可配置

### 来源：`044-research-gateway-minimal.md`

# 044 调研：多平台消息网关最小可行架构

## 任务背景

产品设计明确标出「未完全覆盖：Hermes 的多平台网关」。当前 agent-daemon 仅支持 CLI 和 HTTP API 两种入口，需要新增多平台消息网关层，使同一 Agent Engine 可被 Telegram、Discord 等聊天平台访问。

## Hermes Gateway 核心结论

### 架构骨架

```
┌──────────────────────────────────────────┐
│            GatewayRunner                  │
│  ┌──────────┐  ┌──────────┐              │
│  │ Platform │  │ Platform │  ...         │
│  │ Adapter  │  │ Adapter  │              │
│  └────┬─────┘  └────┬─────┘              │
│       └──────┬──────┘                    │
│              ▼                           │
│       _handleMessage()                   │
│              │                           │
│     ┌────────┼────────┐                  │
│     ▼        ▼        ▼                  │
│  Command   AIAgent   Queue               │
│  dispatch  creation  sessions            │
└──────────────────────────────────────────┘
```

核心模式：**平台适配器 → 消息归一化 → Agent Engine 调用 → 响应回传平台**。

### 关键接口

1. **`BasePlatformAdapter`**：`connect()` / `disconnect()` / `send()` / `handle_message()` 为每个平台适配器的最小合约
2. **`MessageEvent`**：归一化后的入站消息（text, media_urls, reply_to_id 等）
3. **`SendResult`**：平台发送结果
4. **`GatewayRunner`**：管理适配器生命周期、消息路由、会话映射、授权检查

### 授权模型

多层逐级放行：全局允许 → 平台级允许列表 → DM 配对码 → 默认拒绝。

## 当前 agent-daemon 已有能力（可直接复用）

| 组件 | 状态 | 可复用性 |
|------|------|----------|
| `agent.Engine.Run()` | 已实现 | ✅ 网关直接调用 |
| `agent.Engine.EventSink` | 已实现 | ✅ 事件可推送至平台 |
| `store.SessionStore` | SQLite 已实现 | ✅ 会话持久化复用 |
| `internal/api/server.go` | SSE 流式 | ⚠️ 可参考事件消费模式 |
| `internal/config/config.go` | 环境变量 | ✅ 扩展平台配置 |
| `internal/core.AgentEvent` | 统一事件 | ✅ 映射到平台消息 |

## 推荐方案：最小可行网关（v1）

### 范围界定

**本期（v1）实现**：
- 统一适配器接口（`PlatformAdapter`）
- 网关核心（`GatewayRunner`）：适配器生命周期、消息路由、会话管理
- 1 个平台适配器：**Telegram**（Hermes 中最成熟、Go 生态良好）
- 基础授权：`AGENT_TELEGRAM_ALLOWED_USERS` 环境变量
- `agentd serve` 命令中可选启动网关（与 HTTP API 共存）
- 将 `model_stream_event` 增量映射到 Telegram 消息流式编辑

**后续扩展**：
- Discord、Slack、WhatsApp 等平台适配器
- 插件注册机制
- 频道目录
- 投递路由
- 生命周期钩子
- 代理模式

### 包结构

```
internal/gateway/
├── adapter.go          # PlatformAdapter 接口 + MessageEvent/SendResult 类型
├── runner.go           # GatewayRunner：生命周期、消息路由、会话管理
├── session.go          # 网关会话键构建、SessionSource
├── auth.go             # 授权检查
├── config.go           # 网关配置（从 config 扩展）
├── platforms/
│   └── telegram.go     # Telegram 适配器
└── events.go           # AgentEvent → 平台消息映射（增量流式编辑）
```

### 核心接口设计

```go
// PlatformAdapter 是所有平台适配器必须实现的合约
type PlatformAdapter interface {
    // Name 返回平台名称（telegram, discord, ...）
    Name() string
    // Connect 建立平台连接，失败返回 error
    Connect(ctx context.Context) error
    // Disconnect 断开连接，可传入超时 context
    Disconnect(ctx context.Context) error
    // Send 发送文本消息到指定聊天
    Send(ctx context.Context, chatID string, content string, replyTo string) (SendResult, error)
    // EditMessage 编辑已发送消息（流式场景）
    EditMessage(ctx context.Context, chatID string, messageID string, content string) error
    // SendTyping 发送输入中状态
    SendTyping(ctx context.Context, chatID string) error
    // OnMessage 注册消息回调（同一适配器单 goroutine 串行调用）
    OnMessage(ctx context.Context, handler MessageHandler)
}
```

### 会话键设计

复用 Hermes 模式：`agent:main:{platform}:{chat_type}:{chat_id}`

- DM: `agent:main:telegram:dm:123456`
- Group: `agent:main:telegram:group:-789012`
- 线程/子话题保留 `thread_id`，由 SessionSource 携带

### 与 Agent Engine 集成

```go
// GatewayRunner._handleMessage 的流程：
// 1. 构建 sessionKey
// 2. 授权检查
// 3. 加载历史消息
// 4. 构建 EventSink（映射事件到平台消息：流式编辑）
// 5. 调用 eng.Run(ctx, sessionKey, userMessage, systemPrompt, history)
// 6. 如果中断/取消，发送通知
```

### 流式事件映射

| AgentEvent.Type | Telegram 行为 |
|-----------------|---------------|
| `model_stream_event.text_delta` | 累积文本，`editMessageText`（限流 500ms 间隔） |
| `tool_started` | 可选：回复工具执行状态 |
| `tool_finished` | 可选：回复工具结果摘要 |
| `completed` | 最终化消息 |
| `error` / `cancelled` | 发送错误/取消通知 |
| `context_compacted` | 不映射（静默） |

### 配置扩展

```go
// 新增字段到 config.Config
type GatewayConfig struct {
    Enabled         bool
    TelegramToken   string
    TelegramAllowed string   // 用户 ID 逗号分隔，空 = 全部拒绝
}

// 环境变量：
// AGENT_GATEWAY_ENABLED=true
// AGENT_TELEGRAM_BOT_TOKEN=...
// AGENT_TELEGRAM_ALLOWED_USERS=123456,789012
```

### Go Telegram 库选择

推荐 `github.com/go-telegram-bot-api/telegram-bot-api/v5`：
- 社区活跃（5k+ stars）
- 纯 Go 实现
- 支持长轮询（`GetUpdatesChan`）
- 支持消息编辑（`EditMessageText`）
- 无额外 CGO 依赖

### 安全边界

1. Telegram 消息内容通过 `AgentEngine` 的安全管道（工具仍受 approval/workdir 约束）
2. 授权检查在每次消息处理前执行
3. 会话按用户+聊天隔离，用户只能访问自己的会话
4. 禁止跨平台会话访问

## 与 Hermes 的设计差异

| 方面 | Hermes | agent-daemon (本方案) |
|------|--------|----------------------|
| 异步模型 | asyncio | goroutines + channels |
| 适配器注册 | if/elif 链 + 动态 Plugin | 显式注册表 |
| 流式消费 | GatewayStreamConsumer | 复用 EventSink 回调（已在 HTTP 验证） |
| 进程管理 | 内置进程管理器 | 复用 `cmd/agentd` 统一入口 |
| 依赖注入 | 全局对象 | 构造函数注入 |

## 技术风险评估

| 风险 | 等级 | 缓解 |
|------|------|------|
| Telegram Bot API 限流 | 低 | 标准限流 ≤30 msg/s，远低于实际 |
| goroutine 泄漏 | 中 | 统一 context 取消树，关闭时等待 |
| 会话冲突（两个用户同一会话键） | 低 | 会话键天然隔离用户 |
| Go Telegram 库维护状态 | 低 | v5 为最新主要版本，活跃维护 |
| 网关崩溃影响 HTTP API | 中 | 网关与 HTTP 共享进程，需优雅 panic recovery |

## 结论

最小可行网关方案可行且风险可控。建议以 Telegram 为首个平台适配器，验证适配器接口 + 消息路由 + 流式编辑的核心路径，后续按需扩展其他平台。

**方案推荐**：
1. 创建 `internal/gateway/` 包，定义 `PlatformAdapter` 接口和 `GatewayRunner`
2. 实现 `internal/gateway/platforms/telegram.go` 作为首个适配器
3. 扩展 `internal/config/` 支持网关配置
4. 修改 `cmd/agentd/main.go` 的 `serve` 模式，可选启动网关
5. 不需要新增外部依赖之外的库

### 来源：`045-research-skills-adv-trigger-sync.md`

# 045 调研：技能高级能力（自动触发 + 同步）

## 任务背景

当前 agent-daemon 的 Skills 系统实现了基础骨架（`skill_list`/`skill_view`/`skill_manage`），但 LLM 无法自动发现技能目录中的技能，也无法从外部源同步技能。需要补齐两个核心缺口：

1. **自动触发**：运行时将技能目录索引注入 system prompt，使 LLM 自动发现并加载相关技能
2. **技能同步**：支持从外部源（如 GitHub 仓库）下载技能到本地目录

## Hermes 参考

### 自动触发机制

Hermes 使用三层渐进式揭露：

| 层 | 机制 | 触发条件 |
|----|------|---------|
| 1. Skill Index | `build_skills_system_prompt()` 每轮注入技能目录 | 自动，每次模型调用前 |
| 2. 条件过滤 | `required_tools`/`fallback_for_tools` 按工具可用性过滤 | 静态配置 |
| 3. 显式加载 | CLI `--skills`、Slash `/skill-name`、LLM 调用 `skill_view()` | 用户或 LLM 触发 |

第 1 层是核心：系统提示词中注入强指令 + 精简技能目录，LLM 自行判断何时调用 `skill_view()`。

### 技能同步机制

Hermes 的 Skills Hub 是联邦式多源适配器架构（~10 个 source adapter），对 agent-daemon 过于庞大。最小落地方案建议仅支持 **GitHub 仓库** 和 **直接 URL** 两种源。

## 推荐方案

### 范围（L3：跨模块，但不引入新基础设施）

**本期实现**：
1. **Skill Index 注入**：`buildRuntimeSystemPrompt` 中扫描 `<workdir>/skills/` 目录，生成精简技能目录注入 system prompt
2. **`skill_manage sync` 动作**：支持从 GitHub 仓库下载技能到本地

**不纳入**：
- 条件过滤（`required_tools`/`fallback_for_tools`）
- 多源适配器（HermesIndex、skills.sh、ClawHub 等）
- 技能安全扫描（skills_guard）
- 技能缓存清单（bundled_manifest）
- 技能预加载 CLI 参数

### 技能索引格式

```
## Available Skills
Before each task, check if any skill below matches. If relevant, load it with skill_view(name).
- sample-skill: A brief description of what this skill does
- git-workflow: Git branch, commit, PR workflow helper
- api-testing: REST API testing with curl and jq
```

### 技能同步：GitHub 仓库

```
skill_manage action=sync source=github repo=<owner/name> path=<subdir>
```

流程：
1. 请求 `https://api.github.com/repos/{owner}/{name}/contents/{path}`
2. 遍历目录，对每个包含 `SKILL.md` 的子目录下载文件
3. 写入本地 `<workdir>/skills/<skill-name>/`

### 技能同步：直接 URL

```
skill_manage action=sync source=url url=<skill-url> name=<name>
```

流程：
1. HTTP GET 获取原始 SKILL.md 内容
2. 写入 `<workdir>/skills/<name>/SKILL.md`

## 修改文件清单

| 文件 | 变更 |
|------|------|
| `internal/agent/system_prompt.go` | 新增 `buildSkillsIndexBlock()`，`buildRuntimeSystemPrompt` 中调用 |
| `internal/tools/builtin.go` | `skill_manage` 新增 `sync` 动作 |
| `internal/tools/builtin_test.go` | 新增技能索引 + 同步测试 |
| `internal/agent/system_prompt_test.go` | 新增系统提示词技能注入测试 |

## 技术风险

| 风险 | 等级 | 缓解 |
|------|------|------|
| 技能目录过大导致 prompt 膨胀 | 低 | 限制索引条目数（默认 50），仅包含 name + description |
| GitHub API 限流 | 低 | 单次操作，不频繁触发 |
| 网络超时 | 低 | 30s timeout，`context.Context` 传播 |

## 结论

技能索引注入是 Skill 闭环的关键缺失环节，实现后 LLM 真正拥有自动发现并加载技能的能力。同步功能提供最小可用的技能获取渠道。

### 来源：`053-research-hermes-feature-alignment.md`

# 053 Research：Hermes 功能对齐复核

## 背景

用户要求分析 `/data/code/agent-daemon` 是否与 `/data/source/hermes-agent` 功能对齐，并在需要时完善文档。

本次复核基于本地源码与文档，不联网：

- 当前项目：`README.md`、`docs/overview-product.md`、`docs/overview-product-dev.md`、`internal/*`
- Hermes 参考：`README.md`、`website/docs/developer-guide/architecture.md`、`website/docs/developer-guide/tools-runtime.md`、`website/docs/developer-guide/provider-runtime.md`、`website/docs/developer-guide/gateway-internals.md`、`website/docs/reference/tools-reference.md`

## 结论

当前项目与 Hermes 的“核心 Agent daemon 子集”对齐，但不等同于 Hermes Agent 的完整功能面。

已对齐的主干能力：

- 多轮 Agent Loop：模型响应、工具调用、结果回灌、继续推理。
- 工具注册与分发：统一 schema、统一 registry、统一 JSON 结果。
- 模型模式：OpenAI Chat Completions、Anthropic Messages、Codex Responses 三类主路径。
- Provider 韧性：主备 fallback、熔断、竞速、cascade、成本感知、流式事件标准化。
- 状态：SQLite 会话、`MEMORY.md` / `USER.md`、session search、todo。
- 上下文：运行时 system prompt 重建、工作区规则注入、长会话压缩。
- 安全：工作区路径收敛、危险命令检测、硬阻断、会话/模式级审批、交互确认。
- MCP：HTTP、stdio、OAuth client credentials、OAuth authorization code、`/call` 流式事件透传。
- Skills：本地列表/查看/管理、支撑文件、条件过滤、预加载、GitHub 搜索与同步。
- 服务入口：CLI、HTTP `/v1/chat`、SSE `/v1/chat/stream`、WebSocket `/v1/chat/ws`。
- Gateway 最小层：Telegram、Discord、Slack 适配器与流式编辑输出。

未对齐或仅最小覆盖的 Hermes 功能：

- TUI 与完整 CLI 命令体系：Hermes 有全屏 TUI、slash commands、模型/工具/配置/setup/doctor/update 等命令；当前项目只有 `chat`、`serve`、`tools`。
- Provider 生态：Hermes 通过插件覆盖 OpenRouter、Nous、Gemini、DeepSeek、Kimi、MiniMax、HuggingFace、Copilot、Bedrock、自定义 provider 等；当前项目只内置 OpenAI、Anthropic、Codex 三类。
- 工具全集：Hermes 文档列出 68 个内置工具和 52 个 toolsets；当前项目已对齐 68 工具名与 toolsets 名称（另包含额外工具），但 browser/vision/tts 等仍以轻量实现/占位为主，与 Hermes “能力级”实现仍有差距。
- 终端运行环境：Hermes 支持 local、Docker、SSH、Singularity、Modal、Daytona、Vercel Sandbox；当前项目只实现本地 Linux 执行。
- Gateway 完整能力：Hermes 有 20 个左右平台、DM pairing、slash command、运行中断/队列、delivery、hooks、token lock、跨平台发送等；当前项目是 Telegram/Discord/Slack 的最小消息入口。
- 插件系统：Hermes 支持工具、模型 provider、memory provider、context engine、dashboard 等插件；当前项目无通用插件加载框架。
- ACP / IDE 集成：Hermes 有 ACP adapter；当前项目无 ACP。
- Cron / 自动化：Hermes 有 cron scheduler 和平台投递；当前项目无 cron。
- 记忆与检索高级能力：Hermes 有 memory provider 插件、Honcho 等用户建模、FTS5 与 LLM 摘要式 session search；当前项目是 Markdown 记忆 + SQLite LIKE 搜索。
- Web/TUI dashboard：Hermes 有 Web 和 TUI 前端；当前项目只提供 HTTP API。
- 研究/训练链路：Hermes 有 batch runner、Atropos/RL environments、trajectory compression；当前项目无训练/数据生成模块。

## 文档缺口

现有 `docs/overview-product.md` 与 `docs/overview-product-dev.md` 对已实现能力记录较完整，但存在两个问题：

- “核心能力已对齐 Hermes”等表述容易被理解为完整功能对齐。
- 缺少一张稳定的功能对齐矩阵，后续继续补功能时难以判断优先级与边界。

## 推荐方案

本次只做文档完善，不做功能实现：

- 在产品总览中补充“对齐状态”和“暂未覆盖能力”，把当前项目定位为 Hermes 核心 daemon 子集。
- 在开发总览中补充功能矩阵和后续补齐建议，明确哪些是已实现、最小覆盖、未实现。
- 新增本次 Research / Plan / Summary，并更新 `docs/dev/README.md`。

## 三角色审视

- 高级产品：边界澄清能直接解决“是否对齐”的问题，避免用户误以为已完整复刻 Hermes。
- 高级架构师：不引入新依赖，不改变代码结构，只沉淀功能矩阵。
- 高级工程师：只改文档，风险低；验证以文件内容和 diff 为主。

### 来源：`054-research-cli-config-management.md`

# 054 Research：CLI 配置管理最小对齐

## 背景

Hermes 提供 `hermes config set` 等配置管理入口，使用户可以不直接编辑配置文件完成模型、工具、网关等运行参数调整。当前 Go 项目已经支持从 `config/config.ini` 与环境变量加载配置，但缺少 CLI 读写入口。

## 目标

补齐最小 CLI 配置管理面：

- 查看当前配置文件中的键值。
- 读取单个 `section.key`。
- 写入单个 `section.key`。
- 保持环境变量优先级不变。

## 范围

本次只实现 INI 文件管理，不做完整 Hermes 配置系统：

- 不引入 YAML 配置。
- 不实现交互式 setup。
- 不做 provider/model 在线发现。
- 不实现工具启停配置。

## 方案

- 在 `internal/config` 增加小型 INI 管理函数：`ListConfigValues`、`ReadConfigValue`、`SaveConfigValue`。
- 增加 `AGENT_CONFIG_FILE` 作为配置文件路径覆盖入口；未设置时沿用 `config/config.ini` / `config.ini` 查找。
- 在 `cmd/agentd` 增加 `config list|get|set` 子命令。
- `config list` 默认隐藏包含 `api_key/token/secret/password` 的值，避免误打印凭据。

## 三角色审视

- 高级产品：解决用户配置入口缺失，不扩展到 setup wizard。
- 高级架构师：复用现有 INI 配置和 `ini.v1` 依赖，不改变运行时配置结构。
- 高级工程师：新增单元测试覆盖读写、列表排序、密钥脱敏与 `AGENT_CONFIG_FILE`。

### 来源：`055-research-cli-model-management.md`

# 055 Research：CLI 模型管理最小对齐

## 背景

Hermes 提供 `hermes model` 入口用于查看和切换模型。当前 Go 项目已经具备 OpenAI、Anthropic、Codex 三类 provider 和 `agentd config set`，但缺少面向用户的模型切换命令。

## 目标

补齐最小模型管理面：

- 查看当前运行时 provider、model、base URL。
- 列出当前内置 provider。
- 写入 provider 与 provider 对应的 model 配置。

## 范围

- 只支持当前内置 provider：`openai`、`anthropic`、`codex`。
- 不做在线模型发现。
- 不处理 OAuth、凭据登录或 provider 插件。
- 不改变运行时模型调用逻辑。

## 推荐方案

- 在 `cmd/agentd` 增加 `model show|providers|set`。
- `model set openai gpt-4o-mini` 写入 `api.type=openai` 与 `api.model`。
- `model set anthropic claude-...` 写入 `api.type=anthropic` 与 `api.anthropic.model`。
- `model set codex gpt-5-codex` 写入 `api.type=codex` 与 `api.codex.model`。
- 可选 `-base-url` 写入对应 provider 的 `base_url`。

## 三角色审视

- 高级产品：模型切换是 Hermes CLI 体验中的核心高频操作，最小实现直接提升可用性。
- 高级架构师：复用已有 INI 管理能力，不扩展 provider 架构。
- 高级工程师：通过纯函数测试覆盖解析与配置键位，避免 CLI `os.Exit` 路径导致测试脆弱。

### 来源：`056-research-cli-tools-inspection.md`

# 056 Research：CLI 工具查看最小对齐

## 背景

Hermes 提供 `hermes tools` 管理入口。当前 Go 项目已有 `agentd tools`，但只能列出工具名，无法查看模型实际接收的工具 schema。

## 目标

补齐最小工具查看能力：

- 保持 `agentd tools` 继续列出工具名。
- 新增 `agentd tools list` 显式列出工具名。
- 新增 `agentd tools show tool_name` 查看单个工具 schema。
- 新增 `agentd tools schemas` 输出完整工具 schema 列表。

## 范围

- 不实现工具启停配置。
- 不实现 toolset 解析。
- 不改变工具注册或 dispatch 行为。

## 推荐方案

在 `cmd/agentd` 中复用现有 `mustBuildEngine()` 与 `Registry.Schemas()`，只增加 CLI 输出层。`show/schemas` 使用 JSON pretty print，便于前端、SDK 或人工排查。

## 三角色审视

- 高级产品：解决工具可发现性不足，保留向后兼容。
- 高级架构师：不引入 toolset 或插件架构，避免超出当前阶段。
- 高级工程师：新增 helper 测试覆盖 schema 查找，运行时通过手动命令验证输出。

### 来源：`057-research-cli-doctor.md`

# 057 Research：CLI 本地诊断最小对齐

## 背景

Hermes 提供 `hermes doctor` 用于诊断本地环境。当前 Go 项目已有配置、模型、工具查看命令，但缺少启动前诊断入口。

## 目标

实现最小 `agentd doctor`，检查本地可判定的问题：

- 配置文件路径与环境变量优先级提示。
- 工作目录存在性。
- 数据目录可创建且可写。
- 当前 provider/model 是否受支持。
- 当前 provider API key 是否为空。
- MCP transport 配置是否明显错误。
- Gateway 启用时是否至少配置一个平台 token。
- 内置工具是否成功注册。

## 范围

- 不发起网络请求。
- 不调用模型 API。
- 不启动 Gateway 或 MCP 进程。
- 不检查远端凭据是否有效。

## 推荐方案

在 `cmd/agentd` 中新增 `doctor` 子命令，输出 `ok/warn/error`。硬错误返回非零退出码；缺少 API key、Gateway 启用但没有 token 等情况作为 warning。

## 三角色审视

- 高级产品：诊断覆盖用户最常见的启动前问题，不把远端健康检查纳入本期。
- 高级架构师：只读检查，不改变配置或运行时状态；数据目录检查仅创建临时文件后删除。
- 高级工程师：通过 helper 测试覆盖缺 key、坏 workdir、Gateway token 缺失等分支。

### 来源：`058-research-cli-gateway-management.md`

# 058 Research：CLI 网关管理最小对齐

## 背景

Hermes 提供 `hermes gateway` 入口管理消息网关。当前 Go 项目已有 Telegram、Discord、Slack 网关适配器和 `AGENT_GATEWAY_ENABLED` / `gateway.enabled` 配置，但缺少专用 CLI 管理入口。

## 目标

补齐最小网关管理面：

- 查看网关是否启用。
- 查看已配置 token 的平台。
- 列出支持的平台。
- 写入 `gateway.enabled=true/false`。

## 范围

- 不启动或停止运行中的进程。
- 不写入平台 token，避免专用命令处理 secret；token 继续通过 `agentd config set gateway.telegram.bot_token ...` 等方式配置。
- 不实现 Hermes 的 pairing、setup wizard、token lock 或平台级状态探测。

## 推荐方案

在 `cmd/agentd` 增加 `gateway status|platforms|enable|disable`。`status` 默认输出文本，支持 `-json`，并可通过 `-file` 指定配置文件。

## 三角色审视

- 高级产品：提供用户最需要的网关开关和状态查看，不伪装成完整 Gateway setup。
- 高级架构师：复用现有配置系统和 Gateway 支持平台，不启动外部连接。
- 高级工程师：helper 测试覆盖支持平台与已配置平台判断。

### 来源：`059-research-tool-disable-config.md`

# 059 Research：工具禁用配置最小对齐

## 背景

Hermes 支持通过工具配置控制可用工具。当前 Go 项目已经能查看工具 schema，但所有注册工具都会暴露给模型，缺少最小工具启停能力。

## 目标

实现最小工具禁用配置：

- 支持环境变量 `AGENT_DISABLED_TOOLS`。
- 支持 INI `[tools] disabled = terminal,web_fetch`。
- 支持 CLI `agentd tools disable|enable|disabled`。
- 被禁用工具不出现在 registry names/schemas 中，也无法 dispatch。

## 范围

- 只实现禁用列表，不实现 toolset 分组。
- 不实现按平台/会话的工具集。
- 不实现 allowlist 模式。

## 推荐方案

- 在 `config.Config` 增加 `DisabledTools`。
- 在 `tools.Registry` 增加 `Disable`。
- `mustBuildEngine` 完成 builtins/MCP 注册后，统一删除禁用工具。
- CLI 通过 `[tools] disabled` 写入逗号分隔列表。

## 三角色审视

- 高级产品：用户可以关闭高风险或不需要的工具，直接提升可控性。
- 高级架构师：采用简单 denylist，不引入 Hermes 完整 toolset 架构。
- 高级工程师：测试覆盖列表解析、registry 过滤、配置加载。

### 来源：`060-research-cli-sessions.md`

# 060 Research：CLI 会话列表与检索

## 背景

Hermes 有跨会话检索能力，且通常提供直接的 CLI 使用面。当前 Go 项目已有 SQLite 会话存储和 `session_search` 工具，但缺少直接的命令行入口查看/检索历史。

## 目标

提供最小 CLI：

- 列出最近 session（按最新消息排序）。
- 按关键词搜索历史消息内容（优先使用 FTS5，缺失时回退 LIKE）。

## 范围

- 不做 LLM 摘要。
- 不做会话导出/删除。
- 仅本地 `sessions.db`。

## 方案

- `internal/store.SessionStore` 增加 `ListRecentSessions(limit)`。
- `cmd/agentd` 增加 `sessions list` / `sessions search` 子命令，默认输出 JSON。
- `sessions search` 支持 `-exclude session_id` 排除当前会话。

### 来源：`061-research-cli-session-show-stats.md`

# 061 Research：CLI 会话详情查看与统计

## 背景

Hermes 的 session store/CLI 通常支持：

- 列出最近会话
- 搜索历史
- 查看某个 session 的消息（分页）
- 查看会话统计（消息数、时间范围、工具相关计数等）

Go 版 `agent-daemon` 目前已有 `sessions list/search`，且 SQLite `messages` 表里保存了足够信息来做最小 `show/stats`，但缺少 CLI 入口。

## 目标

补齐最小可用的 CLI：

- `agentd sessions show <session_id>`：分页查看消息。
- `agentd sessions stats <session_id>`：输出统计信息，便于排障与外部 UI 取数。

## 范围与非目标

- 不做会话删除/导出（Hermes 有 JSON snapshot 能力，这里先不做）。
- 不做基于 LLM 的摘要与跨会话语义检索（仍保持关键词检索）。
- 默认输出 JSON（与现有 `config/model/tools/gateway` 命令保持一致）。

## 方案

- 复用 `internal/store.SessionStore`：
  - `LoadMessagesPage(sessionID, offset, limit)`
  - `SessionStats(sessionID)`
- `cmd/agentd` 增加子命令：
  - `sessions show [-offset N] [-limit N] session_id`
  - `sessions stats session_id`

### 来源：`062-research-hermes-cron-alignment.md`

# 062 调研：Hermes Cron 能力差异与最小对齐路径

## 背景

Hermes Agent 提供内置 Cron scheduler（`cron/` + `tools/cronjob_tools.py`），支持：

- job 存储（jobs.json）、输出归档、repeat/暂停/恢复/触发
- 多种 schedule：`every 30m`、一次性 duration、cron 表达式、时间戳
- 运行方式：触发时启动一个“新会话”的 agent run（或 `no_agent` 仅脚本）
- 可选投递：origin / local / 指定平台（Telegram/Discord/Slack…）与线程
- prompt 扫描与高危注入/外泄模式阻断（cron prompt 是高危入口）
- 可选链式上下文：把其他 job 的最近输出注入当前 job

本项目此前在对齐矩阵中标记 Cron 为“未覆盖”。

## 差异点（对齐目标）

对齐目标分两层：

1. **最小可用 Cron（本次优先）**
   - 作业存储 + interval/one-shot 调度
   - tool 入口：创建/列出/暂停/恢复/删除/触发
   - 运行：按 job prompt 启动独立 session 的 agent run，并保存 run 结果

2. **Hermes 完整 Cron（后续）**
   - cron 表达式计算
   - 平台投递（origin / 指定 chat/thread）
   - job 输出归档与链式上下文（context_from）
   - prompt threat scanning（注入/外泄/不可见字符）
   - `no_agent` 脚本型作业、toolset 限制与 workdir 继承

## 方案选择

考虑到本项目以 Go 实现、且已有 SQLite（`sessions.db`）：

- 先用 SQLite 承载 cron job / run 存储（避免另起 jobs.json 与一致性问题）。
- 先落地 interval/one-shot，cron expr 先存储但不执行（对齐路线明确可迭代）。
- 调度器以 goroutine + ticker 实现，支持并发度控制，避免与 Agent Loop 交织。

## 风险与约束

- 当前 sandbox 环境对网络/监听端口有限制，测试需要避开 `httptest` 监听。
- cron prompt 是“脱离用户实时监督”的入口，需要后续补 prompt scanning 与审批策略对齐。

### 来源：`063-research-hermes-toolsets-alignment.md`

# 063 调研：Hermes Toolsets 与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 用 `toolsets.py` 定义 toolset（工具集合）：

- toolset 可以直接列工具，也可以通过 `includes` 组合其它 toolset。
- 主要用途是缩减可用工具与 schema 面，减少 token 开销，并按场景启用能力（CLI、API server、cron、平台 bot 等）。

## 当前项目差异

本项目此前只有：

- 固定内置工具列表（register builtins + MCP discovery）。
- `tools.disabled` 禁用名单（从 registry 删除）。

缺少 Hermes 风格的 toolsets 组合与 “enabled_toolsets” 限制机制。

另一个差异点是工具命名：Hermes 的 terminal toolset 使用 `process`，而本项目早期是 `process_status/stop_process`；后续以 `process` wrapper 对齐并保持兼容。

## 最小对齐目标

- 提供内置最小 toolsets（覆盖现有 builtins + cronjob）。
- 支持组合（includes）。
- 支持通过配置将 registry 收缩到指定 toolset 解析结果，从而缩减 schema 面。
- 提供 CLI 入口用于查看与解析 toolsets（便于调试与文档化）。

## 不在本次范围

- Hermes 的 toolset “可用性检查”（check_fn / 环境变量 gating）。
- 动态 schema patch、toolset 分发到不同平台配置、插件发现等。

### 来源：`064-research-hermes-send-message-alignment.md`

# 064 调研：Hermes send_message 与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 的 `send_message`（`tools/send_message_tool.py`）是跨平台投递工具：

- 可 `action=list` 展示可投递目标（channel directory）。
- `action=send` 支持平台 target 解析、频道名解析、媒体附件抽取等。
- 依赖 gateway platform adapters（Telegram/Discord/Slack…）与配置。

## 当前项目差异

本项目已有最小 Gateway（Telegram/Discord/Slack）用于“接收消息 -> 运行 agent -> 回复”，但缺少：

- agent 主循环可调用的跨平台投递工具。
- gateway adapters 的运行时注册/查询机制（供工具调用）。

## 最小对齐目标（本次）

- 提供 `send_message` 工具：
  - `action=list` 返回当前已连接的 adapter 平台名列表。
  - `action=send` 通过运行时 adapter 直接发送文本到指定 `platform + chat_id`。
- 将 adapter 接口从 gateway 包中解耦，避免与 tools/agent 的 import cycle。

## 不在本次范围

- 频道目录（按名称解析 target）、媒体附件、线程/话题路由、重试策略与错误脱敏等。

### 来源：`065-research-hermes-patch-tool-alignment.md`

# 065 调研：Hermes patch 工具与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 内置 `patch` 工具用于对文件做局部修改，避免整文件重写带来的 token 与冲突风险。

## 当前项目差异

Go 版此前只有 `write_file`（整文件写入）与 `skill_manage patch`（仅技能文件），缺少通用 `patch`。

## 最小对齐目标（本次）

- 新增通用 `patch` 工具：
  - `path`：文件路径（限制在 `AGENT_WORKDIR` 内）
  - `old_string/new_string`：字符串替换
  - `replace_all`：控制是否允许多处匹配

## 不在本次范围

- unified diff / fuzzy patch / 多文件 patch。

### 来源：`066-research-hermes-web-tools-alignment.md`

# 066 调研：Hermes web_search/web_extract 与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 的核心 web 工具是：

- `web_search`：搜索网页结果
- `web_extract`：抓取并提取网页正文（可读文本）

用于 research 场景，通常比单纯 `web_fetch` 更省上下文。

## 当前项目差异

Go 版此前只有 `web_fetch`（返回原始内容），缺少搜索与正文抽取。

## 最小对齐目标（本次）

- 新增 `web_search`：使用 DuckDuckGo HTML 页面抓取并解析结果链接（可通过 `base_url` 覆盖，便于测试/自托管）。
- 新增 `web_extract`：抓取并做最小 HTML->text 清洗，返回可读文本并支持 `max_chars` 截断。
- toolsets/web 由 `web_fetch` 调整为 `web_search+web_extract`（保留 `web_fetch` 兼容）。

## 边界

- 解析策略是最小实现（regex/清洗），不保证对所有站点完美。
- 没有高级抓取（JS 渲染、反爬、阅读模式、结构化提取）。

### 来源：`067-research-hermes-clarify-tool-alignment.md`

# 067 调研：Hermes clarify 工具与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 提供 `clarify` 工具用于结构化澄清问题（尤其是消息平台/CLI UI 可以渲染选项时），让 agent 在关键决策点向用户确认。

## 当前项目差异

Go 版此前没有 `clarify` 工具，agent 只能直接用自然语言提问，缺少“选项/结构化”的标准出口。

## 最小对齐目标（本次）

- 新增 `clarify` 工具：
  - 输入：`question`、可选 `options[{label,description}]`、`allow_freeform`
  - 输出：结构化 payload，提示上层/UI/模型向用户提问并收集答案

## 边界

- 不做交互式 UI（仅返回结构化数据）；用户回答仍通过下一条 user message 回到 agent loop。

### 来源：`068-research-hermes-execute-code-alignment.md`

# 068 调研：Hermes execute_code 与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 的 `execute_code` 允许用脚本方式编排工具调用（减少多轮 LLM 往返）。

## 当前项目差异

Go 版此前没有 `execute_code`，复杂 pipeline 只能通过多轮 tool calls 完成。

## 最小对齐目标（本次）

- 新增 `execute_code`：执行短 Python 代码片段，返回 stdout/stderr/exit_code。
- 执行目录受 `AGENT_WORKDIR` 限制，并支持 `timeout_seconds`。

## 边界

- 当前 `execute_code` 仅做“本地脚本执行”，不具备 Hermes 那种脚本内部调用 tools 的 RPC 编排能力。

### 来源：`069-research-hermes-read-file-alignment.md`

# 069 调研：Hermes read_file 输出格式与 Go 版对齐

## Hermes 现状（参考）

Hermes 的 `read_file` 返回纯文本内容（并用 offset/limit 做分页），便于模型直接粘贴/分析，不需要再剥离行号前缀。

## 当前项目差异

Go 版此前默认在每行前加 `N→` 行号，这会：

- 增加 token 开销
- 影响模型进行精确字符串匹配/patch

## 对齐目标（本次）

- `read_file` 默认返回纯文本（不带行号）。
- 提供可选 `with_line_numbers=true` 保留旧行为（调试/定位场景）。

### 来源：`207-research-frontend-tui-parity.md`

# Frontend 与 TUI 对齐调研（Phase 1）

## 背景

当前 `agent-daemon` 已有核心 API、CLI、Gateway 与工具体系，但缺少独立 Web 前端工程与完整 TUI 产品面。  
对比 `/data/source/hermes-agent`，前端/TUI 是独立子系统，不是单文件补丁可对齐。

## 差距结论

1. 仓库内无 `web/` 与 `ui-tui/` 工程基座。
2. CLI 交互缺少统一 slash 命令面，无法承载 “类 TUI” 的日常操作流。
3. 使用文档与开发文档缺少前端/TUI 专章，无法指导后续迭代。

## Phase 1 目标

1. 新增 `web/` 最小可运行工程，先打通 chat/cancel 主链路。
2. 增强 CLI 交互，补齐核心 slash 命令（help/session/tools/history/reload/clear/tui）。
3. 补齐产品与开发文档，明确后续分批收敛路径。

## 边界

- 本阶段不引入完整 Hermes UI/TUI 功能全集（插件槽位、复杂主题系统、完整 slash 生态等）。
- 本阶段重点是“工程基座 + 可运行链路 + 文档对齐”，确保后续可持续推进。
