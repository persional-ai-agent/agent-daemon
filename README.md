# agent-daemon

Go 版 Agent 守护进程与交互式 CLI，参考 `/data/source/hermes-agent` 的 Agent 架构实现。

## 当前能力

- OpenAI `chat/completions`、Anthropic `messages`、Codex `responses` 三模式模型调用
- 支持通过 MCP HTTP 端点发现并注册外部工具（最小桥接）
- 多轮 Agent Loop，支持 `tool_calls`
- 结构化执行事件流，可输出回合、工具、子任务进展
- 内置工具注册与分发
- 内置工具：`terminal`、`process_status`、`stop_process`、`read_file`、`write_file`、`search_files`、`todo`、`memory`、`session_search`、`web_fetch`、`delegate_task`
- 审批工具：`approval`（`status`/`grant`/`revoke`）
- 内置技能工具：`skill_list`、`skill_view`、`skill_manage`
- `delegate_task` 批量模式支持并发子任务执行
- `delegate_task` 支持通过 `max_concurrency` 控制批量委派并发度
- `delegate_task` 支持通过 `timeout_seconds` 和 `fail_fast` 控制超时与失败策略
- `delegate_task` 返回结构化状态字段，如 `status`、`success` 和批量汇总计数
- `delegate_started` / `delegate_finished` 事件会携带结构化状态与子任务结果
- `tool_finished` 事件会携带结构化 `status`、`success` 和 `result`
- `tool_started` / `tool_finished` 会共享 `tool_call_id`、`tool_name`、`arguments` 等统一字段
- `assistant_message` / `completed` 也会携带结构化元数据，便于统一渲染事件流
- SQLite 会话持久化
- `MEMORY.md` / `USER.md` 持久记忆
- 每次运行动态重建 system prompt，并注入持久记忆与最近的 `AGENTS.md`
- 长会话自动上下文压缩（中段摘要 + 头尾保留）
- CLI 交互模式
- HTTP API 服务模式
- `/v1/chat/stream` SSE 流式接口
- `/v1/chat/cancel` 活动会话取消接口
- `/v1/chat` 会返回轻量 `summary`，概览消息数、工具调用数、工具名和委派次数
- 文件工具限制在 `AGENT_WORKDIR` 内，terminal 会硬阻断灾难性命令
- terminal 对危险命令要求 `requires_approval=true` 才可执行（hardline 始终阻断）

## 快速开始

```bash
go run ./cmd/agentd chat
```

设置模型：

```bash
export OPENAI_API_KEY=your_key
export OPENAI_MODEL=gpt-4o-mini
export OPENAI_BASE_URL=https://api.openai.com/v1
export AGENT_MODEL_PROVIDER=openai
```

Anthropic 模式：

```bash
export AGENT_MODEL_PROVIDER=anthropic
export ANTHROPIC_API_KEY=your_key
export ANTHROPIC_MODEL=claude-3-5-haiku-latest
export ANTHROPIC_BASE_URL=https://api.anthropic.com/v1
export AGENT_MAX_CONTEXT_CHARS=120000
export AGENT_COMPRESSION_TAIL_MESSAGES=14
```

Codex 模式：

```bash
export AGENT_MODEL_PROVIDER=codex
export CODEX_API_KEY=your_key
export CODEX_MODEL=gpt-5-codex
export CODEX_BASE_URL=https://api.openai.com/v1
```

MCP 最小桥接：

```bash
export AGENT_MCP_ENDPOINT=http://127.0.0.1:9000
export AGENT_MCP_TIMEOUT_SECONDS=30
export AGENT_APPROVAL_TTL_SECONDS=300
```

启动 HTTP 服务：

```bash
go run ./cmd/agentd serve
```

请求示例：

```bash
curl -s http://127.0.0.1:8080/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"请读取 README.md 并总结"}'
```

SSE 流式请求示例：

```bash
curl -N http://127.0.0.1:8080/v1/chat/stream \
  -H 'Content-Type: application/json' \
  -d '{"message":"请检查当前目录并说明你会先做什么"}'
```

取消活动会话示例：

```bash
curl -s http://127.0.0.1:8080/v1/chat/cancel \
  -H 'Content-Type: application/json' \
  -d '{"session_id":"your-session-id"}'
```

流式事件包括：

- `session`
- `user_message`
- `turn_started`
- `context_compacted`
- `assistant_message`
- `tool_started`
- `tool_finished`
- `delegate_started`
- `delegate_finished`
- `delegate_failed`
- `completed`
- `cancelled`
- `max_iterations_reached`
- `error`
- `result`

## 文档

- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `docs/dev/README.md`
