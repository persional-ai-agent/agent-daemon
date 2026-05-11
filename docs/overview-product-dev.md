# 开发设计总览

## 架构分层

- `cmd/agentd`：程序入口，负责装配配置、存储、工具、模型客户端、CLI/API
- `internal/core`：统一消息、工具 schema、运行结果等共享类型
- `internal/agent`：Agent Loop，处理多轮推理、重试、tool call 执行、事件发射与结果回灌
- `internal/model`：模型客户端，当前实现 OpenAI / Anthropic / Codex 三模式，并支持主备 provider 故障切换、三 provider 流式聚合、增量事件最小透传与 `model_stream_event` v2+ 最小标准字典（含 `tool_args_start/delta/done`、`message_done.finish_reason` 归一到 `stop/tool_calls/length` 与 `stop_sequence/incomplete_reason`（`length` 场景自动补齐）、`usage.prompt_tokens/completion_tokens/total_tokens`（含缺失补齐与偏小校正标记）+ `usage_consistency_status`（`ok/derived/adjusted/source_only/invalid`）+ `prompt_cache_write_tokens/prompt_cache_read_tokens/reasoning_tokens`、`message_id/tool_call_id` 多来源别名归一）
- `internal/tools`：工具注册中心、工具上下文、内置工具、审批状态工具、技能发现/管理工具、MCP HTTP/stdio 代理工具（含 OAuth client_credentials 与 `/call` 流式兼容）、后台进程管理、Todo 状态
- `internal/store`：SQLite 会话存储与 session search
- `internal/memory`：`MEMORY.md` / `USER.md` 管理
- `internal/cli`：CLI 交互层
- `internal/api`：HTTP 服务层，提供同步、SSE 流式与 WebSocket 接口
- `internal/gateway`：多平台消息网关层，`PlatformAdapter` 接口 + `GatewayRunner`；含 Telegram、Discord、Slack、Yuanbao 适配器（Yuanbao 当前仅最小 inbound：TIMTextElem push）
- `internal/config`：环境变量与 INI 配置加载；提供 `config list|get|set` 所需的配置文件读写能力

## Hermes 到 Go 的映射

- `run_agent.py / AIAgent` -> `internal/agent/loop.go`
- `model_tools.py` -> `internal/tools/registry.go` + `internal/tools/builtin.go`
- `tools/terminal_tool.py` -> `internal/tools/process.go` + `terminal`/`process_status`/`stop_process`
- `gateway/run.py` + `gateway/platforms/` -> `internal/gateway/runner.go` + `internal/gateway/platforms/`
- `hermes_state.py / session storage` -> `internal/store/session_store.go`
- `memory_tool.py` -> `internal/memory/store.go`
- CLI / API 入口 -> `internal/cli/chat.go`、`internal/api/server.go`、`cmd/agentd/main.go`
- `hermes_cli/config.py` 的最小配置管理面 -> `internal/config/manage.go` + `agentd config list|get|set`

## 关键设计

### 1. 核心消息格式统一

内部统一使用 OpenAI 风格消息：

- `system`
- `user`
- `assistant`
- `tool`

这样可以直接复用 OpenAI 兼容接口的 `tool_calls` 机制，并为后续兼容更多 provider 降低改造成本。

### 1.1 运行时系统提示词重建

每次 `Engine.Run()` 都会动态重建并注入 system prompt，而不是依赖首轮历史：

- 基础 system prompt
- `MEMORY.md` / `USER.md` 持久记忆
- 工作目录向上查找到的最近 `AGENTS.md`

这样即使同一 session 跨多次 CLI/API 请求继续运行，也不会丢失系统提示词，并且能始终读取最新记忆和仓库规则。

### 2. 工具注册中心

`Registry` 负责：

- 注册工具
- 暴露工具 schema 给模型
- 根据工具名分发执行
- 保证统一错误返回结构

这与 Hermes 的 `registry + handle_function_call` 模式一致。

### 3. 事件回调面

`Engine` 通过 `EventSink` 发出统一的 `AgentEvent`，覆盖：

- 用户输入进入会话
- 回合开始
- assistant 输出
- 工具开始与结束
- `delegate_task` 子 Agent 开始、完成、失败
- 批量 `delegate_task` 按输入顺序收集结果，但内部并发执行，并支持 `max_concurrency` 限流
- 每个子任务支持 `timeout_seconds`，批量模式支持 `fail_fast` 在首个失败后取消剩余子任务
- `delegate_task` 的工具返回包含结构化状态字段，便于 CLI、API 和后续前端直接展示
- `delegate_started` / `delegate_finished` 事件的 `Data` 中会携带结构化状态与子任务结果，SSE 可直接透传
- `tool_finished` 事件的 `Data` 中会携带结构化工具结果，避免客户端重复解析 JSON 字符串
- `tool_started` / `tool_finished` 共享统一元数据字段，如 `tool_call_id`、`tool_name`、`arguments`、`status`
- `assistant_message` / `completed` 也会携带统一元数据，如 `message_role`、`content_length`、`tool_call_count`
- 正常完成、达到最大迭代、异常失败

这让 HTTP 层可以直接把内部执行过程映射为 SSE 事件，而不需要侵入工具实现。

事件字段与兼容性约定见 `docs/dev/001-events-hermes-agent-go-port.md`。

### 4. HTTP 会话取消

`internal/api` 维护活动会话表，将 `session_id` 绑定到请求级 `context.CancelFunc`：

- `/v1/chat` 和 `/v1/chat/stream` 在开始运行时注册活动会话
- `/v1/chat/cancel` 按 `session_id` 查找并触发取消
- SSE 在取消场景下输出 `cancelled` 事件，而不是泛化成普通 `error`

同时，`/v1/chat` 非流式响应会在保留原始 `RunResult` 顶层字段的前提下，补充一个轻量 `summary`，用于概览：

- 消息总数
- assistant 消息数
- 工具调用数
- 工具名列表
- `delegate_task` 调用次数

### 5. 状态分层

- 会话消息：SQLite，适合历史加载与 session_search
- 长期记忆：Markdown 文件，便于人工查看与维护，并在运行时回灌到 system prompt
- Todo：进程内 session 级状态，适合当前多轮执行周期

### 6. 终端执行分层

- 前台命令：同步等待结果
- 后台命令：生成 `session_id`，通过状态轮询与停止接口管理

这对应 Hermes 中 terminal/process registry 的核心思路，但当前只实现本地 Linux 后端。

### 7. 工具安全基线

对齐 Hermes 的基础护栏思路，当前 Go 版增加了两条最小安全边界：

- 文件工具通过 `Workdir` 做路径收敛，阻止越界访问
- terminal 对灾难性命令做硬阻断，如根目录递归删除、磁盘格式化、原始块设备写入、整机重启等
- terminal 对危险但可恢复命令增加审批门禁：需显式 `requires_approval=true` 才执行

交互式审批流程：已实现 `pending_approval` + `approval confirm` 交互确认闭环（危险命令未被预授权时返回待审批状态，用户可通过 `approval confirm` 批准并自动重新执行）。

### 8. Context Compression

`internal/agent/compressor.go` 提供最小可用上下文压缩能力：

- 在每轮模型调用前估算消息体积
- 超过预算时保留 system + 最近 N 条消息
- 将中段历史压缩成一条 assistant 摘要消息
- 发出 `context_compacted` 事件，包含压缩前后体积和裁剪数量

相关配置：

- `AGENT_MAX_CONTEXT_CHARS`（默认 `120000`）
- `AGENT_COMPRESSION_TAIL_MESSAGES`（默认 `14`）

### 9. CLI 配置管理

`cmd/agentd` 提供最小配置管理子命令：

- `agentd config list`：列出配置文件中的键值，默认隐藏 `api_key/token/secret/password` 等敏感值。
- `agentd config get section.key`：读取单个配置项。
- `agentd config set section.key value`：写入单个配置项。
- `agentd model show`：展示当前运行时 provider、model 与 base URL。
- `agentd model providers`：列出内置 provider：OpenAI、Anthropic、Codex。
- `agentd model set provider model` / `agentd model set provider:model`：写入 provider 选择与对应 provider 的 model 配置。
- `agentd tools list` / `agentd tools`：列出当前注册工具名。
- `agentd tools show tool_name`：输出单个工具的 function schema。
- `agentd tools schemas`：输出当前注册工具的完整 schema 列表。
- `agentd tools disable tool_name` / `enable tool_name` / `disabled`：写入并查看 `[tools] disabled`，运行时会从 registry 移除被禁用工具。
- `agentd doctor`：检查配置文件优先级、工作目录、数据目录、模型/provider、MCP、Gateway 与内置工具注册情况；支持 `-json`。
- `agentd gateway status` / `platforms` / `enable` / `disable`：查看网关状态、支持平台，并写入 `gateway.enabled` 开关；平台 token 继续通过 `config set` 管理。

默认读写 `config/config.ini`，也可通过 `AGENT_CONFIG_FILE` 或 `-file` 指定路径。运行时优先级仍为：环境变量 > 配置文件 > 内置默认值。

## 扩展点

核心 Agent daemon 能力已对齐 Hermes 的主干设计。后续可选扩展：

- 配置与 CLI 管理面：已具备最小 `config list|get|set`、`model show|providers|set`、`tools list|show|schemas|enable|disable`、`doctor`、`gateway status|platforms|enable|disable`；后续补齐 Gateway setup、setup wizard 等命令入口。
- Toolset 与插件系统：从固定内置工具列表演进为 toolset 解析、可用性检查、插件发现与动态 schema 过滤。
- Gateway 完整体验：补齐 DM pairing、slash command、运行中断/队列、delivery、hooks、token lock，再扩展更多平台。
- 执行环境：在 `internal/tools/process.go` 之外抽象本地、Docker、SSH、Modal、Daytona、Singularity、Vercel Sandbox 等后端。
- ACP 与自动化：按需新增 ACP adapter、cron scheduler、平台投递和任务状态存储。

## Hermes 功能对齐矩阵

| Hermes 能力域 | Hermes 实现参考 | Go 当前实现 | 对齐状态 | 后续补齐建议 |
|----------------|-----------------|-------------|----------|--------------|
| Agent Loop | `run_agent.py`、`agent/prompt_builder.py` | `internal/agent` | 已对齐核心 | 保持事件协议稳定，避免把 UI 逻辑塞入 loop |
| Provider runtime | `hermes_cli/runtime_provider.py`、`plugins/model-providers/` | `internal/model`、`internal/config` | 部分对齐 | 先抽象 provider profile，再增加更多 provider |
| Tool registry | `tools/registry.py`、`model_tools.py`、`toolsets.py` | `internal/tools/registry.go`、`builtin.go`、`toolsets.go` | 部分对齐 | 已补最小 toolsets 解析与 registry 过滤；后续补 availability check、动态 schema patch 与插件发现 |
| Built-in tools | `tools/*`，Hermes 文档列出 68 个工具 | terminal、process、file、todo、memory、session_search、web_fetch/web_search/web_extract、delegate、approval、skills、MCP、cronjob、send_message | 最小覆盖 | 按场景优先补 browser/code/vision/tts 等 |
| Terminal environments | `tools/environments/*` | `internal/tools/process.go` | 最小覆盖 | 抽象 Environment 接口后接 Docker/SSH 等后端 |
| Session storage | `hermes_state.py`、`gateway/session.py` | `internal/store/session_store.go` | 部分对齐 | 如需高质量检索，补 FTS5 与摘要层 |
| Memory | `agent/memory_manager.py`、`plugins/memory/*` | `internal/memory/store.go` | 最小覆盖 | 先定义 memory provider 接口，再接外部插件 |
| Context compression | `agent/context_compressor.py`、context engine plugins | `internal/agent/compressor.go` | 核心对齐 | 后续可加可替换 context engine |
| MCP | `tools/mcp_tool.py` | `internal/tools/mcp.go` | 核心对齐 | 继续补更完整的服务器能力与错误分类 |
| Skills | `agent/skill_*`、`tools/skills_*`、Skills Hub | `skill_list`（含别名 `skills_list`）、`skill_view`、`skill_manage`、`skill_search` | 核心对齐 | 补多源 Hub API、版本/来源元数据、冲突策略 |
| CLI/TUI | `cli.py`、`hermes_cli/*`、`ui-tui/` | `internal/cli/chat.go`、`cmd/agentd`、`internal/config/manage.go` | 最小覆盖 | 已补最小 config/model/tools 查看与启停/doctor/gateway 开关；后续补 gateway setup、setup wizard，再评估 TUI |
| HTTP/WebSocket | `gateway/platforms/api_server.py`、`web/` | `internal/api` | API 核心对齐 | 若需要管理后台，再单独设计 Web UI |
| Gateway | `gateway/run.py`、`gateway/platforms/*`、`tools/send_message_tool.py` | `internal/gateway` + Telegram/Discord/Slack + `send_message` | 部分对齐 | 已补最小 send_message（基于运行时 adapter registry）；后续补 delivery、配对、slash command、中断/队列 |
| Plugin system | `hermes_cli/plugins.py`、`plugins/*` | 无通用插件框架 | 未覆盖 | 明确插件边界后再引入，避免过早复杂化 |
| ACP/IDE | `acp_adapter/` | 无 | 未覆盖 | 仅在 IDE 场景明确时补齐 |
| Cron | `cron/`、`tools/cronjob_tools.py` | `internal/cron`、`cronjob` tool | 部分对齐 | 当前先覆盖 interval/one-shot 作业存储与调度；后续补 cron expr 计算、平台投递与链式上下文 |
| Research/RL/trajectory | `batch_runner.py`、`environments/`、`trajectory_compressor.py` | 无 | 未覆盖 | 与 daemon 主路径解耦，作为独立扩展 |
