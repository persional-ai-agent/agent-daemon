# agent-daemon

Go 版 Agent 守护进程与交互式 CLI，参考 `/data/source/hermes-agent` 的 Agent 架构实现。

## 对齐边界

本项目当前对齐的是 Hermes 的核心 Agent daemon 子集，不是 Hermes Agent 的完整 Go 版复刻。已覆盖核心 Agent Loop、工具调用、会话/记忆、MCP、Skills、Provider 韧性、HTTP/SSE/WebSocket 与 Telegram/Discord/Slack 最小网关；暂未覆盖 Hermes 的完整 TUI/CLI、provider 插件生态、68 个内置工具、52 个 toolsets、多终端后端、ACP、Cron、完整 Gateway、通用插件系统和研究/训练链路。

详细对齐矩阵见：

- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `docs/dev/053-summary-hermes-feature-alignment.md`

## 当前能力

- OpenAI `chat/completions`、Anthropic `messages`、Codex `responses` 三模式模型调用
- 支持通过 MCP HTTP/stdio 发现并注册外部工具（最小桥接）
- 多轮 Agent Loop，支持 `tool_calls`
- 结构化执行事件流，可输出回合、工具、子任务进展
- 内置工具注册与分发
- 内置工具：`terminal`、`process`（兼容 `process_status`/`stop_process`）、`read_file`、`write_file`、`patch`、`search_files`、`todo`、`memory`、`session_search`、`web_fetch`、`web_search`、`web_extract`、`clarify`、`delegate_task`、`execute_code`
- 补丁工具：`patch`（最小字符串替换 patch）
- 审批工具：`approval`（`status`/`grant`/`revoke`）
- 内置技能工具：`skill_list`、`skill_view`、`skill_manage`（`create/edit/patch/delete/write_file/remove_file`）
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
- 最小 Cron：`cronjob` 工具 + 后台调度器（需开启 `AGENT_CRON_ENABLED=true`）

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
export AGENT_MODEL_FALLBACK_PROVIDER=anthropic
export AGENT_MODEL_USE_STREAMING=true
```

也可以直接用统一 setup 入口写入配置文件：

```bash
go run ./cmd/agentd setup -provider openai -model gpt-4o-mini -api-key "$OPENAI_API_KEY"
go run ./cmd/agentd setup -provider anthropic -model claude-3-5-haiku-latest -api-key "$ANTHROPIC_API_KEY" -gateway-platform telegram -gateway-token "$AGENT_TELEGRAM_BOT_TOKEN"
printf 'openai\ngpt-4o-mini\n\n\nanthropic\nnone\n' | go run ./cmd/agentd setup wizard
go run ./cmd/agentd bootstrap init
go run ./cmd/agentd bootstrap status -json
go run ./cmd/agentd version
go run ./cmd/agentd version -check-update -json
go run ./cmd/agentd update changelog -json
go run ./cmd/agentd update doctor -json
go run ./cmd/agentd update status -json
go run ./cmd/agentd update -fetch
go run ./cmd/agentd update release -json
go run ./cmd/agentd update install
go run ./cmd/agentd update apply
go run ./cmd/agentd update uninstall
	go run ./cmd/agentd gateway status -json
	go run ./cmd/agentd gateway install
	go run ./cmd/agentd gateway manifest -platform slack -command /agent -json
	go run ./cmd/agentd gateway manifest -platform discord -json
	go run ./cmd/agentd gateway manifest -platform telegram -json
	go run ./cmd/agentd gateway manifest -platform yuanbao -json
	go run ./cmd/agentd gateway start
	go run ./cmd/agentd gateway stop
	go run ./cmd/agentd gateway uninstall
```

说明：Gateway 现在同时持有同一 `workdir` 单实例锁和基于平台凭证指纹的全局 `token lock`，避免不同工作区重复启动同一组网关消费者；聊天侧可用 `/status` 查看当前会话概况，用 `/pending` 查看最近待审批项，再用 `/approvals`、`/grant`、`/revoke`、`/approve`、`/deny` 处理审批闭环；Telegram 已补最小原生命令菜单、审批按钮与 manifest 导出，Discord 已补最小原生 slash 命令（含 `grant` / `revoke`）、审批按钮与命令清单导出，Slack 已补最小原生审批按钮、通用 slash 命令入口与 manifest 导出，Yuanbao 已补最小审批快捷回复与 manifest 导出（如“批准”/“拒绝”）。

说明：`AGENT_MODEL_USE_STREAMING=true` 当前可用于 `openai` / `anthropic` / `codex` 三种 provider 的流式聚合调用。

说明：`update install` 现在会在 `.agent-daemon/bin/` 下生成 `update-status.sh`、`update-check.sh`、`update-release.sh`、`update-apply.sh` 四个最小运维脚本。

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

MCP 最小桥接（HTTP）：

```bash
export AGENT_MCP_ENDPOINT=http://127.0.0.1:9000
export AGENT_MCP_TIMEOUT_SECONDS=30
export AGENT_APPROVAL_TTL_SECONDS=300
```

MCP HTTP + OAuth（client_credentials）：

```bash
export AGENT_MCP_TRANSPORT=http
export AGENT_MCP_ENDPOINT=http://127.0.0.1:9000
export AGENT_MCP_OAUTH_TOKEN_URL=https://auth.example.com/oauth/token
export AGENT_MCP_OAUTH_CLIENT_ID=your_client_id
export AGENT_MCP_OAUTH_CLIENT_SECRET=your_client_secret
export AGENT_MCP_OAUTH_SCOPES="mcp.read mcp.call"
```

MCP 最小桥接（stdio）：

```bash
export AGENT_MCP_TRANSPORT=stdio
export AGENT_MCP_STDIO_COMMAND='npx -y @modelcontextprotocol/server-filesystem /data/code/agent-daemon'
export AGENT_MCP_TIMEOUT_SECONDS=30
```

启动 HTTP 服务：

```bash
go run ./cmd/agentd serve
```

开启 Cron 调度器（interval/one-shot）：

```bash
export AGENT_CRON_ENABLED=true
export AGENT_CRON_TICK_SECONDS=5
export AGENT_CRON_MAX_CONCURRENCY=1
```

配置管理：

```bash
go run ./cmd/agentd config list
go run ./cmd/agentd config get api.model
go run ./cmd/agentd config set api.model gpt-4o-mini
go run ./cmd/agentd config set provider.fallback anthropic
```

Toolsets（缩减工具 schema 面，类似 Hermes toolsets）：

```bash
export AGENT_ENABLED_TOOLSETS=core
go run ./cmd/agentd toolsets list
go run ./cmd/agentd toolsets show core
go run ./cmd/agentd toolsets resolve core
```

Gateway + `send_message`（最小对齐 Hermes messaging tool）：

- 需要 `agentd serve` 启用 gateway（`AGENT_GATEWAY_ENABLED=true`）并配置平台 token。
- `send_message(action='list')` 会返回当前已连接的 platform adapter（如 `telegram`/`discord`/`slack`/`yuanbao`）。
- Yuanbao：设置 `YUANBAO_TOKEN`（或 `YUANBAO_APP_ID` + `YUANBAO_APP_SECRET`）后，可使用 `yb_send_dm` / `yb_send_sticker` / `yb_query_group_info` / `yb_query_group_members`。

默认读写 `config/config.ini`。也可以通过 `AGENT_CONFIG_FILE=/path/to/config.ini` 或 `-file /path/to/config.ini` 指定配置文件；环境变量仍优先于配置文件。

模型切换：

```bash
go run ./cmd/agentd model show
go run ./cmd/agentd model providers
go run ./cmd/agentd model set openai gpt-4o-mini
go run ./cmd/agentd model set anthropic:claude-3-5-haiku-latest
go run ./cmd/agentd model set -base-url https://api.openai.com/v1 codex gpt-5-codex
```

工具查看：

```bash
go run ./cmd/agentd tools
go run ./cmd/agentd tools list
go run ./cmd/agentd tools show terminal
go run ./cmd/agentd tools schemas
go run ./cmd/agentd tools disable terminal
go run ./cmd/agentd tools disabled
go run ./cmd/agentd tools enable terminal
```

诊断检查：

```bash
go run ./cmd/agentd doctor
go run ./cmd/agentd doctor -json
```

网关配置：

```bash
go run ./cmd/agentd gateway status
go run ./cmd/agentd gateway platforms
go run ./cmd/agentd gateway enable
go run ./cmd/agentd gateway disable
go run ./cmd/agentd gateway setup -platform telegram -token "$AGENT_TELEGRAM_BOT_TOKEN"
go run ./cmd/agentd gateway setup -platform slack -bot-token "$AGENT_SLACK_BOT_TOKEN" -app-token "$AGENT_SLACK_APP_TOKEN"
```

会话检索：

```bash
go run ./cmd/agentd sessions list
go run ./cmd/agentd sessions search hello
go run ./cmd/agentd sessions search -limit 50 -exclude session-id hello
go run ./cmd/agentd sessions show your-session-id
go run ./cmd/agentd sessions show -offset 200 -limit 200 your-session-id
go run ./cmd/agentd sessions stats your-session-id
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
- `model_stream_event`
- `delegate_started`
- `delegate_finished`
- `delegate_failed`
- `completed`
- `cancelled`
- `max_iterations_reached`
- `error`
- `result`

`model_stream_event` 最小标准字段（v2）：

- `event_type=text_delta` 时，`event_data.text`
- `event_type=tool_args_start` 时，`event_data.tool_call_id`、`event_data.tool_name`
- `event_type=tool_args_delta` 时，`event_data.tool_name`、`event_data.arguments_delta`
- `event_type=tool_args_done` 时，`event_data.tool_call_id`、`event_data.tool_name`、`event_data.arguments`
- `event_type=message_start` 时，`event_data.message_id`（可为空，统一兼容 `id/response_id/message.id`）
- `event_type=message_done` 时，`event_data.message_id`（可为空，统一兼容 `id/response_id/message.id`）、`event_data.finish_reason`（标准值：`stop`/`tool_calls`/`length`）、`event_data.stop_sequence`（可选）、`event_data.incomplete_reason`（可选；`finish_reason=length` 时归一为 `length`）
- `event_type=usage` 时，`event_data.prompt_tokens`、`event_data.completion_tokens`、`event_data.total_tokens`（缺失时自动补齐；异常偏小时自动校正，`event_data.total_tokens_adjusted=true`）、`event_data.usage_consistency_status`（`ok/derived/adjusted/source_only/invalid`）、`event_data.prompt_cache_write_tokens`（可选）、`event_data.prompt_cache_read_tokens`（可选）、`event_data.reasoning_tokens`（可选）
- `event_type=tool_call_start` 时，`event_data.tool_call_id`、`event_data.tool_name`（`tool_call_id` 统一兼容 `call_id/tool_use_id/item_id/output_item_id`）
- `event_type=tool_call_done` 时，`event_data.tool_call_id`、`event_data.tool_name`、`event_data.arguments`（`tool_call_id` 同上）

## 文档

- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `docs/dev/README.md`
