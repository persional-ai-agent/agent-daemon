# 0015 hermes summary merged

## 模块

- `hermes`

## 类型

- `summary`

## 合并来源

- `0022-hermes-summary-merged.md`

## 合并内容

### 来源：`0022-hermes-summary-merged.md`

# 0022 hermes summary merged

## 模块

- `hermes`

## 类型

- `summary`

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
- `0070-hermes-core-tool-stubs.md`
- `0000-hermes-agent-go-port.md`

## 合并内容

### 来源：`0000-hermes-agent-go-port.md`

# 001 总结：Hermes Agent Go 版实现结果

## 已完成

- 搭建 `core message + tool schema` 共享类型
- 实现 OpenAI 兼容模型客户端
- 实现多轮 Agent Loop 与重试机制
- 实现工具注册中心
- 实现内置工具：
  - `terminal`
  - `process_status`
  - `stop_process`
  - `read_file`
  - `write_file`
  - `search_files`
  - `todo`
  - `memory`
  - `session_search`
  - `web_fetch`
- 实现 `delegate_task` 子 Agent 委派、批量并发执行、并发度控制、结构化状态返回，以及超时/失败策略
- 实现 Agent 结构化事件流
- 完成事件协议文档沉淀
- 实现 `/v1/chat/stream` SSE 流式接口
- 实现 `/v1/chat/cancel` 会话取消接口
- 实现 `/v1/chat` 轻量 `summary`
- 实现 SQLite 会话持久化
- 实现 `MEMORY.md` / `USER.md` 记忆存储
- 实现 CLI 与 HTTP API 双入口
- 增加关键单元测试并通过 `go test ./...`
- 增加 Agent Loop 级委派事件测试
- 增加 SSE 级委派事件透传测试
- 增加 `tool_finished` 结构化事件测试
- 增加错误类事件结构化测试

## 与原计划的偏差

无重大偏差，整体按计划落地。

实现过程中将无法通过工作区删除机制清理的历史文件 `internal/agent/types.go` 安全改名为 `internal/agent/types.go.legacy`，避免其继续参与 Go 源码路径。

## 当前能力边界

当前版本实现的是 Hermes 的“完整核心功能”，但不是 1:1 全生态复刻。

已对齐的部分：

- Agent Loop
- 工具 schema / dispatch
- 文件与终端核心能力
- Session / Memory / Todo 状态分层
- CLI / HTTP 入口
- 非流式摘要与 SSE 事件协议

尚未覆盖的外围生态：

- 多平台网关
- MCP
- 技能系统
- Context Compression
- 多 provider API mode
- 审批系统与复杂安全护栏

## 后续建议

- 增加更细粒度的工具级中断控制
- 抽象 provider，补 OpenAI/Anthropic/Codex 多模式
- 为工具系统增加权限与审批
- 为 `search_files` / `read_file` 增加更强分页与 glob 能力
- 引入上下文压缩，支持长会话

### 来源：`0001-hermes-gap-closure.md`

# 002 总结：Hermes 核心闭环差异补齐

## 已完成

- 修复 system prompt 只在首轮注入的问题，`Run()` 现在每次都会装配运行时系统提示词
- 为运行时系统提示词增加持久记忆回灌，自动注入 `MEMORY.md` 与 `USER.md`
- 为运行时系统提示词增加工作区规则注入，自动读取最近的 `AGENTS.md`
- 为 `read_file`、`write_file`、`search_files` 增加 `Workdir` 边界约束
- 为 `terminal` 增加最基础的灾难性命令硬阻断
- 增加针对性测试，覆盖提示词装配、记忆快照、路径越界、防危险命令执行

## 关键决策

- 不把 system prompt 写回会话存储，而是在每次运行前动态重建
  - 理由：这样可确保记忆与工作区规则始终使用最新版本，也避免历史中堆积过期 system message
- 安全侧先做“硬阻断 + 工作区边界”，不在本次引入完整审批流
  - 理由：可以先补齐核心闭环与安全基线，同时避免审批系统显著扩大范围

## 与计划偏差

- 无重大偏差
- 原计划中的“工作区规则注入”最终采用“向上查找最近 `AGENTS.md`”实现，便于兼容多层目录工作区

## 当前能力边界

已经补齐的核心闭环：

- system prompt 跨请求持续生效
- 长期记忆可写、可持久化、可回灌
- 仓库级规则可进入提示词
- 文件工具具备工作区安全边界
- terminal 具备基础硬阻断护栏

仍未覆盖的 Hermes 外围生态：

- Context Compression
- Skills 系统
- MCP
- 完整审批系统
- 多 provider API mode
- 多平台 Gateway

## 验证

- `go test ./...`

## 后续建议

- 将 `AGENTS.md` 之外的上下文文件注入抽象成统一 prompt builder
- 为 terminal 增加“危险但可审批”的中间层，而不只有硬阻断
- 在长会话下继续补齐上下文压缩，避免提示词和工具输出持续膨胀

### 来源：`0052-hermes-feature-alignment.md`

# 053 总结：Hermes 功能对齐复核与文档补齐

## 变更摘要

完成 `/data/code/agent-daemon` 与 `/data/source/hermes-agent` 的功能对齐复核，并补齐文档边界说明。

核心结论：当前项目已对齐 Hermes 的核心 Agent daemon 主干，但不是 Hermes Agent 的完整 Go 版复刻。

## 修改文件

| 文件 | 变更 |
|------|------|
| `README.md` | 增加对齐边界说明与详细文档链接 |
| `docs/overview-product.md` | 增加“对齐状态”与“暂未覆盖能力”，澄清产品边界 |
| `docs/overview-product-dev.md` | 增加 Hermes 功能对齐矩阵与后续补齐建议 |
| `docs/dev/053-research-hermes-feature-alignment.md` | 记录调研结论 |
| `docs/dev/053-plan-hermes-feature-alignment.md` | 记录文档补齐计划 |
| `docs/dev/README.md` | 增加 053 文档索引 |

## 对齐结论

已基本对齐：

- Agent Loop、工具调用回灌、事件流。
- OpenAI / Anthropic / Codex 三模式模型调用。
- Provider fallback、race、circuit、cascade、成本感知与标准化流式事件。
- SQLite 会话、Markdown 记忆、todo、session search。
- MCP HTTP/stdio/OAuth/streaming。
- Skills 本地管理、预加载、过滤、同步、搜索。
- CLI + HTTP + SSE + WebSocket。
- Telegram / Discord / Slack 最小 Gateway。

未完整对齐：

- Hermes TUI、完整 CLI 命令体系、setup/doctor/update/model/tools 配置流。
- 18+ provider 插件生态。
- 68 个内置工具、52 个 toolsets 与多类平台工具。
- Docker/SSH/Modal/Daytona/Singularity/Vercel Sandbox 等执行环境。
- Gateway 的完整平台矩阵、DM pairing、slash command、队列/中断、delivery、hooks。
- 通用插件系统、ACP、cron、Web/TUI dashboard、RL/trajectory 数据链路。
- FTS5 + LLM 摘要式跨会话检索和 memory provider 插件。

## 验证

- 文档复核：已完成。
- 代码测试：未运行。此次未修改 Go 源码，验证重点为文档 diff。

## 后续建议

如果目标是继续逼近 Hermes 完整体验，优先级建议：

1. 补齐配置与 CLI 管理面，先让 provider、tool、gateway、skills 可被用户稳定配置。
2. 补齐 toolset/可用性过滤和插件加载边界，再扩展工具数量。
3. 补齐 Gateway 授权、slash command、中断/队列和 delivery，再扩更多平台。
4. 若需要 IDE 或自动化场景，再规划 ACP 与 cron。

### 来源：`0061-hermes-cron-alignment.md`

# 062 总结：Hermes Cron 最小对齐（interval/one-shot）

## 完成情况

已补齐 Hermes Cron 域的最小可用能力：

- SQLite 内新增 `cron_jobs` / `cron_runs` 存储。
- 新增 `internal/cron` 调度器（ticker 扫描 due job，支持并发度控制）。
- 新增内置工具 `cronjob`（action 压缩 schema）用于管理作业。
- `serve` / `chat` 模式支持通过配置开启 cron scheduler。
- 文档更新：对齐矩阵将 Cron 从“未覆盖”调整为“部分对齐”，并写明边界与后续工作。

## 使用方式（最小）

- 开启：
  - `AGENT_CRON_ENABLED=true`
  - `AGENT_CRON_TICK_SECONDS=5`
  - `AGENT_CRON_MAX_CONCURRENCY=1`
- 创建作业：通过 `cronjob` tool 调用（适用于 agent 内部自举）。

## 边界与待补齐

- cron 表达式目前仅识别并存储，不执行；创建时会提示不支持。
- 未实现 Hermes 的平台投递、prompt threat scanning、脚本型作业与链式上下文。

### 来源：`0062-hermes-toolsets-alignment.md`

# 063 总结：Hermes Toolsets 最小对齐

## 完成情况

- 新增最小 toolsets：`internal/tools/toolsets.go`，支持 `includes` 组合。
- 新增配置：`tools.enabled_toolsets`（env：`AGENT_ENABLED_TOOLSETS`）用于收缩 registry/tool schema 面。
- 新增 CLI：`agentd toolsets list`、`agentd toolsets resolve ...`。
- 文档对齐矩阵更新：toolsets 标记为“部分对齐”。

## 使用方式

- 推荐默认：不设置 `AGENT_ENABLED_TOOLSETS`，保持现有行为（完整内置工具 + MCP 工具）。
- 如需缩减：`AGENT_ENABLED_TOOLSETS=core`（或多个 toolset：`core,web`）。

## 边界

- 不做 Hermes 的 check_fn 可用性 gating；也不对 MCP tools 做独立分组过滤（仍会被 registry 过滤逻辑影响，取决于工具名是否在解析集合内）。
- `core` toolset 当前倾向对齐 Hermes 的 core 工具名集合，可能包含尚未能力对齐的 stub 工具与最小实现（例如 browser/vision/tts/execute_code）。

### 来源：`0063-hermes-send-message-alignment.md`

# 064 总结：Hermes send_message 最小对齐

## 完成情况

- 新增 `internal/platform`：adapter 接口 + 运行时 registry。
- Gateway runner 在 adapter connect/disconnect 时 register/unregister。
- 新增工具 `send_message`：
  - `list`：返回已连接平台列表
  - `send`：按 `platform + chat_id`（或 `target=platform:chat_id`）投递文本
- toolsets 增加 `messaging`，并纳入 `core` includes。

## 边界

- 无 channel directory / 目标名称解析。
- 无媒体附件路由、线程/话题、重试与错误脱敏对齐。

### 来源：`0064-hermes-patch-tool-alignment.md`

# 065 总结：Hermes patch 工具最小对齐

## 完成情况

- 新增内置工具 `patch`（字符串替换语义）。
- `file` toolset 纳入 `patch`。
- 补单测与文档索引。

## 边界

- 目前是最小 patch（old/new string），不支持 unified diff 与 fuzzy patch。

### 来源：`0065-hermes-web-tools-alignment.md`

# 066 总结：Hermes web tools 最小对齐

## 完成情况

- 新增 `web_search`（DDG HTML 抓取 + 最小解析）与 `web_extract`（最小 HTML->text 清洗 + 截断）。
- toolsets `web` 默认调整为 `web_search/web_extract`。
- 保留 `web_fetch` 作为兼容与调试用途。

## 边界

- `web_search` 依赖第三方 HTML 页面结构，可能随时间变化；必要时可通过 `base_url` 指向自托管/替代实现。

### 来源：`0066-hermes-clarify-tool-alignment.md`

# 067 总结：Hermes clarify 工具最小对齐

## 完成情况

- 新增内置工具 `clarify`（返回结构化 question/options）。
- toolsets 增加 `clarify` 并纳入 `core`。
- 文档与索引更新。

## 边界

- `clarify` 不直接收集用户答案；需要用户下一条消息回复选项 label 或自由文本。

### 来源：`0067-hermes-execute-code-alignment.md`

# 068 总结：Hermes execute_code 最小对齐

## 完成情况

- 新增 `execute_code`：python 子进程执行（workdir 限制 + timeout）。
- toolsets 新增 `code_execution`（未默认纳入 core）。
- toolsets 新增 `code_execution`（现已纳入 core，用于对齐 Hermes 默认 core tool list；能力仍为最小本地脚本执行）。
- 文档与索引更新，补单测。

## 边界

- 未实现 Hermes 的“脚本内部调用 tools”的编排能力。

### 来源：`0070-hermes-core-tool-stubs.md`

# 071 总结：Hermes core 工具缺口的 stub 对齐

## 背景

Hermes `_HERMES_CORE_TOOLS` 默认包含 browser/vision/image_gen/tts 等工具域。
agent-daemon 目前不实现这些能力，但为了减少与 Hermes toolsets/脚本的“名称不匹配”问题，可以先补齐接口级 stub。

## 本次变更

新增以下工具名的 stub（调用会返回 `not implemented in agent-daemon`）：

- `vision_analyze`
- `image_generate`
- `mixture_of_agents`
- `text_to_speech`
- `browser_*`（navigate/snapshot/click/type/scroll/back/press/get_images/vision/console/cdp/dialog）

并在 `toolsets` 中补齐 `vision/image_gen/browser/tts` 分组，且将它们纳入 `core` includes（接口对齐优先）。

## 边界

这只是接口对齐，不代表能力对齐；真正实现需要引入浏览器后端、视觉模型接入与音频管线等。

### 来源：`0000-hermes-agent-go-port.md`

# 001 事件协议：Hermes Agent Go 版

## 目标

定义 `AgentEvent` 在 SSE 与内部回调面的稳定字段，用于前端、SDK 或测试统一消费。

## 基础结构

所有事件都使用统一结构：

```json
{
  "type": "tool_finished",
  "session_id": "session-id",
  "turn": 1,
  "tool_name": "read_file",
  "content": "...",
  "data": {}
}
```

公共字段说明：

- `type`：事件类型
- `session_id`：当前会话或子任务会话 ID
- `turn`：当前回合序号
- `tool_name`：工具事件对应的工具名
- `content`：面向展示的原始文本内容
- `data`：结构化扩展字段

## 主要事件

### 会话事件

- `user_message`
- `turn_started`
- `assistant_message`
- `completed`
- `cancelled`
- `error`
- `max_iterations_reached`

### 工具事件

- `tool_started`
- `tool_finished`

### 委派事件

- `delegate_started`
- `delegate_finished`
- `delegate_failed`

## 结构化字段

### `assistant_message`

`data` 字段包含：

- `status`
- `message_role`
- `content_length`
- `tool_call_count`
- `has_tool_calls`

### `completed`

`data` 字段包含：

- `status`
- `message_role`
- `content_length`
- `tool_call_count`
- `has_tool_calls`
- `finished_naturally`

### `tool_started`

`data` 字段包含：

- `status`
- `tool_call_id`
- `tool_name`
- `arguments`

### `tool_finished`

`data` 字段包含：

- `status`
- `success`
- `tool_call_id`
- `tool_name`
- `arguments`
- `result`

`result` 在工具返回 JSON 时为对象；否则为原始字符串。

### `delegate_started`

`data` 字段包含：

- `parent_session_id`
- `goal`
- `status`

### `delegate_finished`

`data` 字段包含：

- `parent_session_id`
- `goal`
- `status`
- `success`
- `result`

### `delegate_failed`

`data` 字段包含：

- `parent_session_id`
- `goal`
- `status`
- `success`
- `error`

### `cancelled`

`data` 字段包含：

- `status`
- `turn`
- `error`

SSE 兜底取消事件至少包含：

- `session_id`
- `status`
- `reason`

### `error`

`data` 字段包含：

- `status`
- `turn`
- `error`

SSE 兜底错误事件至少包含：

- `session_id`
- `status`
- `error`

### `max_iterations_reached`

`data` 字段包含：

- `status`
- `max_iterations`
- `finished`

## 兼容性约定

- 新增字段优先追加到 `data` 中，避免破坏现有顶层结构。
- `content` 保留为可直接展示的文本，不要求客户端再反向解析。
- 客户端若需要稳定消费，应优先读取 `data`，将 `content` 作为展示兜底。
