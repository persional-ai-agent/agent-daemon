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
- `internal/api`：HTTP 服务层，提供同步与 SSE 流式接口
- `internal/config`：环境变量配置

## Hermes 到 Go 的映射

- `run_agent.py / AIAgent` -> `internal/agent/loop.go`
- `model_tools.py` -> `internal/tools/registry.go` + `internal/tools/builtin.go`
- `tools/terminal_tool.py` -> `internal/tools/process.go` + `terminal`/`process_status`/`stop_process`
- `hermes_state.py / session storage` -> `internal/store/session_store.go`
- `memory_tool.py` -> `internal/memory/store.go`
- CLI / API 入口 -> `internal/cli/chat.go`、`internal/api/server.go`、`cmd/agentd/main.go`

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

交互式审批流程与审批状态持久化仍作为后续扩展项保留。

### 8. Context Compression

`internal/agent/compressor.go` 提供最小可用上下文压缩能力：

- 在每轮模型调用前估算消息体积
- 超过预算时保留 system + 最近 N 条消息
- 将中段历史压缩成一条 assistant 摘要消息
- 发出 `context_compacted` 事件，包含压缩前后体积和裁剪数量

相关配置：

- `AGENT_MAX_CONTEXT_CHARS`（默认 `120000`）
- `AGENT_COMPRESSION_TAIL_MESSAGES`（默认 `14`）

## 扩展点

后续可以继续增加：

- provider 级高级能力（重试策略、完整事件字典覆盖、并行竞速与熔断）
- MCP 高级能力（OAuth 授权码/刷新、流式事件透传与会话绑定）
- 技能高级能力（同步与自动触发）
- 上下文压缩
- WebSocket
- 多平台网关
