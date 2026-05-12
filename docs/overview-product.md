# 产品设计总览

## 目标

在 `/data/code/agent-daemon` 中用 Go 实现一个可独立运行的 Agent 系统，参考 `/data/source/hermes-agent` 的核心 Agent 设计思路，提供：

- 统一多轮会话 Agent
- 工具调用与工具结果回灌
- 会话持久化与长期记忆
- CLI 与 HTTP 双入口
- 后续向多模型、多网关、多插件扩展的清晰边界

## 目标用户

- 需要在本地或服务端运行代码 Agent 的开发者
- 需要通过 API 调用 Agent 的平台系统
- 需要持续扩展工具、会话、记忆能力的研发团队

## 核心能力

- Agent Loop：模型响应若产生 `tool_calls`，则执行工具并将结果回灌到上下文，直到模型自然停止
- 执行事件流：对外暴露用户消息、回合开始、工具执行、子任务委派、完成/失败等结构化事件
- 工具系统：统一注册、统一 schema、统一 dispatch
- 执行工具：终端命令、文件读写、文本搜索、网页抓取
- 状态工具：todo、session_search、memory、delegate_task
- 持久化：SQLite 会话历史与 Markdown 记忆文件
- 双入口：交互式 CLI + HTTP API
- 配置管理：`agentd config list|get|set` 可读写 `config/config.ini`，`agentd model show|providers|set` 可查看和切换模型，`agentd tools list|show|schemas|disable|enable` 可查看和启停工具，`agentd doctor` 可做本地配置诊断，`agentd setup` 可一键写入最小模型/网关配置，`agentd gateway status|platforms|enable|disable|setup` 可管理网关开关与平台配置，环境变量仍保持最高优先级
- 消息网关：Telegram + Discord + Slack + Yuanbao 适配器（Yuanbao 目前为最小 inbound：TIMTextElem；其余能力按需扩展），`PlatformAdapter` 接口可按需扩展
- 非流式摘要：`/v1/chat` 返回轻量 `summary`
- 流式 API：基于 SSE 的 `/v1/chat/stream`
- 中断控制：支持按 `session_id` 取消活动中的 HTTP 会话
- 事件协议：已提供独立事件协议文档，便于前端或 SDK 对接
- 运行时提示词装配：每次运行都会重新注入 system prompt、持久记忆与工作区规则
- 基础安全护栏：文件工具限制在工作区内，terminal 会硬阻断灾难性命令
- 命令审批门禁：危险命令与 tirith 扫描告警会进入审批流程（hardline 命令仍直接阻断）
- WebSocket：`/v1/chat/ws` 端点支持全双工实时通信
- 上下文压缩：长会话超预算时自动压缩中段历史，保留头尾上下文

## 与 Hermes 的关系

本项目采用 Hermes 的核心设计思想，而不是逐文件翻译 Python 实现：

- 保留 Hermes 的“模型回合 -> 工具调用 -> 结果回灌 -> 再次推理”主循环
- 保留“工具注册中心 + schema 暴露 + handler 分发”模式
- 保留“会话历史 + 长期记忆 + Todo 状态”的状态分层
- 将完整 TUI、全量 CLI 命令、完整多平台 Gateway、ACP、Cron、复杂插件与研究/训练链路作为后续扩展层

## 对齐状态

当前项目已对齐 Hermes 的核心 Agent daemon 子集，而不是 Hermes Agent 的完整 Go 版复刻。

| 能力域 | 当前状态 | 说明 |
|--------|----------|------|
| Agent Loop | 已对齐 | 支持多轮推理、工具调用、工具结果回灌与最大迭代控制 |
| 工具注册与分发 | 部分对齐 | 统一 schema、registry、dispatch、JSON 结果；已补最小 toolsets 过滤（可缩减 schema 面） |
| 模型 API 模式 | 部分对齐 | 内置 OpenAI、Anthropic、Codex；未覆盖 Hermes 的 provider 插件生态 |
| Provider 韧性 | 已对齐核心能力 | 已有 fallback、熔断、竞速、cascade、成本感知与流式事件标准化 |
| 会话与记忆 | 部分对齐 | SQLite 会话、Markdown 记忆、session search；未实现 Hermes 的 FTS5 + LLM 摘要式检索和 memory provider 插件 |
| 上下文管理 | 已对齐核心能力 | system prompt 重建、工作区规则注入、长会话压缩 |
| MCP | 已对齐核心能力 | 支持 HTTP、stdio、OAuth client credentials、OAuth authorization code 与流式事件透传 |
| Skills | 已对齐核心能力 | 支持本地列表/查看/管理、条件过滤、预加载、同步与 GitHub 搜索 |
| API 服务 | 已对齐核心能力 | HTTP、SSE、WebSocket、取消接口 |
| Gateway | 部分对齐 | 支持 Telegram、Discord、Slack、Yuanbao；已补最小 `send_message` + Telegram/Discord/Slack 本地文件投递（`MEDIA:` / `media_path`）+ Yuanbao best-effort 媒体投递（COS 上传链路，依赖网络与凭证）+ 最小 pairing/slash command/队列中断/hooks 运维 + `gateway run/start/stop/restart/install/uninstall` 管理面 + 同 workdir 单实例锁 + 基于平台凭证指纹的跨工作区 `token lock` + 文本状态/审批命令 `/status`/`/pending`/`/approvals`/`/grant`/`/revoke`/`/approve`/`/deny` + Telegram 最小原生命令菜单/审批按钮/manifest 导出 + Discord 最小原生 slash 命令（含 `grant` / `revoke`）/审批按钮/命令清单导出 + Slack 最小原生审批按钮、slash 命令入口与 manifest 导出 + Yuanbao 最小审批快捷回复/manifest 导出；未覆盖 Hermes 的完整平台矩阵、更多平台原生 slash UI 和更完整 token lock |
| CLI/TUI | 部分对齐 | 已有交互式 chat、serve、tools list/show/schemas/enable/disable、config、model、doctor、`setup`/`setup wizard`、`bootstrap`、`version`、`update status/check/release/apply/install/uninstall`、gateway 与最小 `gateway setup/run/start/stop/restart/install/uninstall`；未实现 Hermes 全屏 TUI、完整安装器级 update 与更完整命令体系 |
| 工具全集 | 部分对齐 | 已对齐 Hermes 文档中的 68 个内置工具“工具名/Toolsets 名称”（含 `discord`、`yb_*`、`process` 动作面等）；其中 browser/vision/image_generate 等仍存在能力级差距，但 browser 已支持可选 CDP 后端（配置 `BROWSER_CDP_URL`）以执行 JS/DOM |
| 终端环境 | 最小覆盖 | 当前为本地 Linux 执行；未覆盖 Docker、SSH、Modal、Daytona、Singularity、Vercel Sandbox |
| 插件/ACP/Cron/训练 | 部分对齐 | 已有最小 cron scheduler + 作业存储（需显式开启）；暂无通用插件系统、ACP adapter、batch/RL/trajectory 链路 |

## 暂未覆盖能力

以下能力属于 Hermes 完整产品体验的一部分，但当前项目未实现或只保留最小骨架：

- 全屏 TUI、完整安装器级 update 流程和更完整的 CLI 管理面
- 18+ provider 与 provider 插件加载机制
- 52 个 Hermes toolsets 的完整动态行为（按平台/环境动态过滤、UI 交互管理）
- browser（真实浏览器/JS/DOM）、vision（模型推理）、tts（真实语音合成）、image_generate（真实 FAL 后端）等“能力级”实现
- Docker、SSH、Singularity、Modal、Daytona、Vercel Sandbox 等终端后端
- 多平台 Gateway 的原生 slash UI、更完整 token lock 策略和更多平台适配器（当前 Telegram 具备最小原生命令菜单/审批按钮/manifest 导出，Discord 具备最小原生 slash 命令含 `grant` / `revoke`、审批按钮与命令清单导出，Slack 具备最小原生审批按钮、通用 slash 命令入口与 manifest 导出，Yuanbao 具备最小审批快捷回复与 manifest 导出）
- 通用插件系统、ACP/IDE 集成、Cron 的平台投递/脚本/链式上下文等高级能力、Web/TUI dashboard、研究/训练数据链路

## 当前范围

- 已实现：核心闭环、系统提示词动态装配、记忆回灌、工作区规则注入、上下文压缩、Hermes 68 工具名对齐 + toolsets 名称兼容（另含若干额外辅助工具）、并发子 Agent 委派、结构化事件流、持久化（SQLite）、CLI + HTTP API（同步/SSE/WebSocket）、CLI 配置管理（`config list|get|set` + `model show|providers|set` + `tools list|show|schemas|enable|disable` + `doctor` + `setup` + `setup wizard` + `bootstrap` + `version` + `update status/check/release/apply/install/uninstall` + `gateway status|platforms|enable|disable|setup|run|start|stop|restart|install|uninstall|manifest`）、安全护栏（hardline 阻断 + 审批门禁 + 交互确认 + pattern 级授权 + tirith 可选预扫描）、MCP（http/stdio/OAuth CC/授权码/`/call` 流式 + 事件透传）、技能（索引注入 + 条件过滤 + sync 同步 + 预加载 + GitHub 搜索）、Provider（OpenAI/Anthropic/Codex 流式聚合 + 故障切换 + 熔断 + 并行竞速 + 多级级联 + 成本感知 + `model_stream_event` v2 完整字典）、多平台网关（Telegram + Discord + Slack + Yuanbao，含最小 pairing/queue/cancel/hooks 运维、最小进程/脚本安装管理、同 workdir 单实例锁、跨工作区 token lock、文本状态/审批命令 `/status`/`/pending`/`/approvals`/`/grant`/`/revoke`/`/approve`/`/deny`、Telegram 最小原生命令菜单/审批按钮/manifest 导出、Discord 最小原生 slash 命令含 `grant` / `revoke`、审批按钮与命令清单导出、Slack 最小原生审批按钮、通用 slash 命令入口与 manifest 导出、Yuanbao 最小审批快捷回复与 manifest 导出）
