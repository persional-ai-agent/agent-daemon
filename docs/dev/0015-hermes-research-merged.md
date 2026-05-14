# 0015 hermes research merged

## 模块

- `hermes`

## 类型

- `research`

## 合并来源

- `0022-hermes-research-merged.md`

## 合并内容

### 来源：`0022-hermes-research-merged.md`

# 0022 hermes research merged

## 模块

- `hermes`

## 类型

- `research`

## 合并来源

- `0000-hermes-agent-go-port.md`
- `0001-hermes-gap-closure.md`
- `0052-hermes-feature-alignment.md`
- `0061-hermes-cron-alignment.md`
- `0062-hermes-toolsets-alignment.md`
- `0063-hermes-send-message-alignment.md`
- `0064-hermes-patch-tool-alignment.md`
- `0065-hermes-web-tools-alignment.md`
- `0066-hermes-clarify-tool-alignment.md`
- `0067-hermes-execute-code-alignment.md`
- `0068-hermes-read-file-alignment.md`

## 合并内容

### 来源：`0000-hermes-agent-go-port.md`

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

### 来源：`0001-hermes-gap-closure.md`

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

### 来源：`0052-hermes-feature-alignment.md`

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

### 来源：`0061-hermes-cron-alignment.md`

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

### 来源：`0062-hermes-toolsets-alignment.md`

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

### 来源：`0063-hermes-send-message-alignment.md`

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

### 来源：`0064-hermes-patch-tool-alignment.md`

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

### 来源：`0065-hermes-web-tools-alignment.md`

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

### 来源：`0066-hermes-clarify-tool-alignment.md`

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

### 来源：`0067-hermes-execute-code-alignment.md`

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

### 来源：`0068-hermes-read-file-alignment.md`

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
