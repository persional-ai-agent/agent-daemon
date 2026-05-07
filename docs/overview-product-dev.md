# 开发设计总览

## 架构分层

- `cmd/agentd`：程序入口，负责装配配置、存储、工具、模型客户端、CLI/API
- `internal/core`：统一消息、工具 schema、运行结果等共享类型
- `internal/agent`：Agent Loop，处理多轮推理、重试、tool call 执行与结果回灌
- `internal/model`：模型客户端，目前实现 OpenAI 兼容接口
- `internal/tools`：工具注册中心、工具上下文、内置工具、后台进程管理、Todo 状态
- `internal/store`：SQLite 会话存储与 session search
- `internal/memory`：`MEMORY.md` / `USER.md` 管理
- `internal/cli`：CLI 交互层
- `internal/api`：HTTP 服务层
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

### 2. 工具注册中心

`Registry` 负责：

- 注册工具
- 暴露工具 schema 给模型
- 根据工具名分发执行
- 保证统一错误返回结构

这与 Hermes 的 `registry + handle_function_call` 模式一致。

### 3. 状态分层

- 会话消息：SQLite，适合历史加载与 session_search
- 长期记忆：Markdown 文件，便于人工查看与维护
- Todo：进程内 session 级状态，适合当前多轮执行周期

### 4. 终端执行分层

- 前台命令：同步等待结果
- 后台命令：生成 `session_id`，通过状态轮询与停止接口管理

这对应 Hermes 中 terminal/process registry 的核心思路，但当前只实现本地 Linux 后端。

## 扩展点

后续可以继续增加：

- provider 抽象与 Anthropic/Codex 模式
- MCP 工具接入
- 技能系统
- 上下文压缩
- delegate_task 子 Agent 并发执行
- WebSocket / 流式输出
- 多平台网关
