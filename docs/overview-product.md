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
- 消息网关：Telegram + Discord + Slack + WhatsApp + Yuanbao 适配器（Yuanbao 目前为最小 inbound：TIMTextElem；WhatsApp 支持最小 inbound webhook（challenge + 签名校验 + 文本/媒体 URL 入站）；其余能力按需扩展），`PlatformAdapter` 接口可按需扩展
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
| 模型 API 模式 | 部分对齐 | 内置 OpenAI、Anthropic、Codex；并支持最小 provider 插件（`type=provider` command 运行时接入）；仍未覆盖 Hermes 的完整 provider 插件生态 |
| Provider 韧性 | 已对齐核心能力 | 已有 fallback、熔断、竞速、cascade、成本感知与流式事件标准化 |
| 会话与记忆 | 已对齐核心能力 | SQLite 会话、FTS5/LIKE 检索、按会话聚合摘要、关键词/事实/高亮召回、Markdown 记忆与 `memory.extract` 去重沉淀；仍未覆盖 Hermes 的外部 memory provider 插件和 LLM 级摘要质量 |
| 上下文管理 | 已对齐核心能力 | system prompt 重建、工作区规则注入、长会话压缩 |
| MCP | 已对齐核心能力 | 支持 HTTP、stdio、OAuth client credentials、OAuth authorization code 与流式事件透传 |
| Skills | 已对齐核心能力 | 支持本地列表/查看/管理、条件过滤、预加载、同步与 GitHub 搜索 |
| API 服务 | 已对齐核心能力 | HTTP、SSE、WebSocket、取消接口 |
| Gateway | 部分对齐 | 支持 Telegram、Discord、Slack、WhatsApp、Yuanbao；已补最小 `send_message` + Telegram/Discord/Slack 本地文件投递（`MEDIA:` / `media_path`）+ Yuanbao best-effort 媒体投递（COS 上传链路，依赖网络与凭证）+ WhatsApp 最小 inbound webhook（`/v1/gateway/whatsapp/webhook` challenge、`X-Hub-Signature-256` 签名校验、文本/媒体 URL 入站）+ 最小 pairing/slash command/队列中断/hooks 运维 + `gateway run/start/stop/restart/install/uninstall` 管理面 + 同 workdir 单实例锁 + 基于平台凭证指纹的跨工作区 `token lock` + 文本状态/审批命令 `/status`/`/pending`/`/approvals`/`/grant`/`/revoke`/`/approve`/`/deny` + Telegram 最小原生命令菜单/审批按钮/manifest 导出 + Discord 最小原生 slash 命令（含 `grant` / `revoke`）/审批按钮/命令清单导出 + Slack 最小原生审批按钮、slash 命令入口与 manifest 导出 + Yuanbao 最小审批快捷回复/manifest 导出；未覆盖 Hermes 的完整平台矩阵、更多平台原生 slash UI 和更完整 token lock |
| CLI/TUI | 部分对齐 | 已有交互式 chat、lite TUI 事件轨迹、会话切换/重试/撤销/压缩/导出、工具与 toolsets 查看、todo/memory/model 状态查看、serve、tools list/show/schemas/enable/disable、config、model、doctor、`setup`/`setup wizard`、`bootstrap`、`version`、`update bundle(build/inspect/manifest/plan/verify/unpack/apply/backups/status/doctor/prune/snapshot/snapshots/snapshots-prune/snapshots-doctor/snapshots-status/snapshots-restore-plan/snapshots-restore/snapshots-delete/rollback-plan/rollback)/changelog/doctor/status/check/release/apply/install/uninstall` 与最小 update 脚本安装面、gateway 与最小 `gateway setup/run/start/stop/restart/install/uninstall`；未完全覆盖 Hermes 的高级 TUI 交互、完整安装器级 update 与全部命令体系 |
| 工具全集 | 部分对齐 | 已对齐 Hermes 文档中的 68 个内置工具“工具名/Toolsets 名称”（含 `discord`、`yb_*`、`process` 动作面等）；其中 browser/vision/image_generate 等仍存在能力级差距，但 browser 已支持可选 CDP 后端（配置 `BROWSER_CDP_URL`）以执行 JS/DOM |
| 终端环境 | 已对齐核心能力 | 当前支持 `local/docker/podman/singularity/ssh/daytona/vercel/modal` 执行后端；后台模式仍限定 `local` |
| 插件/ACP/Cron/训练 | 部分对齐 | 已有 cron scheduler + 作业存储（需显式开启），支持 interval、one-shot、RFC3339 时间戳、5/6 字段 cron 表达式、运行结果投递到 Gateway 目标与可选链式上下文；插件系统已支持 JSON/YAML manifest、Hermes 风格 `plugin.yaml` 元数据、多能力 tool/provider/command/dashboard 声明、本地与 marketplace 安装/卸载、启停、校验、Ed25519 manifest 签名、文件 sha256 校验、默认进程沙箱、tool/provider 运行时注册、command 执行与 dashboard slot API；已补最小 ACP API 适配层；已补最小 research trajectory 链路（`agentd research run/compress/stats`） |

## 暂未覆盖能力

以下能力属于 Hermes 完整产品体验的一部分，但当前项目未实现或只保留最小骨架：

- Hermes 高级 TUI 交互、完整安装器级 update 流程和全部 CLI 命令体系
- 更完整的多 provider 生态能力（当前已支持最小 provider 插件加载，但尚未覆盖 Hermes 规模与流式/隔离能力）
- 52 个 Hermes toolsets 的完整动态行为（按平台/环境动态过滤、UI 交互管理）
- browser（真实浏览器/JS/DOM）、vision（模型推理）、tts（真实语音合成）、image_generate（真实 FAL 后端）等“能力级”实现
- 终端后端的后台进程模式（`background=true`）目前仅支持 `local`
- 多平台 Gateway 的原生 slash UI、更完整 token lock 策略和更多平台适配器（当前 Telegram 具备最小原生命令菜单/审批按钮/manifest 导出，Discord 具备最小原生 slash 命令含 `grant` / `revoke`、审批按钮与命令清单导出，Slack 具备最小原生审批按钮、通用 slash 命令入口与 manifest 导出，WhatsApp 具备最小 webhook 入站与签名校验，Yuanbao 具备最小审批快捷回复与 manifest 导出）
- ACP 完整协议能力、Cron 的脚本动作等高级能力、Web/TUI dashboard、完整研究/训练数据链路（当前仅最小 trajectory runtime）

## 当前范围

- 已实现：核心闭环、系统提示词动态装配、记忆回灌、工作区规则注入、上下文压缩、Hermes 68 工具名对齐 + toolsets 名称兼容（另含若干额外辅助工具）、并发子 Agent 委派、结构化事件流、持久化（SQLite）、CLI + HTTP API（同步/SSE/WebSocket）、CLI 配置管理（`config list|get|set` + `model show|providers|set` + `tools list|show|schemas|enable|disable` + `doctor` + `setup` + `setup wizard` + `bootstrap` + `version` + `update bundle(build/inspect/manifest/plan/verify/unpack/apply/backups/status/doctor/prune/snapshot/snapshots/snapshots-prune/snapshots-doctor/snapshots-status/snapshots-restore-plan/snapshots-restore/snapshots-delete/rollback-plan/rollback)/changelog/doctor/status/check/release/apply/install/uninstall` + `gateway status|platforms|enable|disable|setup|run|start|stop|restart|install|uninstall|manifest`）、安全护栏（hardline 阻断 + 审批门禁 + 交互确认 + pattern 级授权 + tirith 可选预扫描）、MCP（http/stdio/OAuth CC/授权码/`/call` 流式 + 事件透传）、技能（索引注入 + 条件过滤 + sync 同步 + 预加载 + GitHub 搜索）、Provider（OpenAI/Anthropic/Codex 流式聚合 + 故障切换 + 熔断 + 并行竞速 + 多级级联 + 成本感知 + `model_stream_event` v2 完整字典）、多平台网关（Telegram + Discord + Slack + WhatsApp + Yuanbao，含最小 pairing/queue/cancel/hooks 运维、最小进程/脚本安装管理、同 workdir 单实例锁、跨工作区 token lock、文本状态/审批命令 `/status`/`/pending`/`/approvals`/`/grant`/`/revoke`/`/approve`/`/deny`、Telegram 最小原生命令菜单/审批按钮/manifest 导出、Discord 最小原生 slash 命令含 `grant` / `revoke`、审批按钮与命令清单导出、Slack 最小原生审批按钮、通用 slash 命令入口与 manifest 导出、WhatsApp 最小 webhook 入站（challenge + 签名 + 文本/媒体 URL 入站）、Yuanbao 最小审批快捷回复与 manifest 导出）

## Frontend / TUI 迭代状态

- Web（Phase 1）：已新增独立 `web/` 工程骨架（Vite + React），包含 `Chat / Sessions / Tools / Skills / Agents / Cron / Models / Plugins / Gateway / Voice / Config` 十一页入口；Chat 已打通同步、WS/SSE 流式、取消、重连诊断，管理页已接入会话、工具、技能、子代理、Cron 任务、模型切换、插件 dashboard slot、网关诊断、语音状态与配置接口。
- CLI/TUI（Phase 1）：交互式 chat 已具备状态化 slash 命令面，覆盖 `/new`、`/resume`、`/retry`、`/undo`、`/compress`、`/save`、`/tools show`、`/toolsets`、`/todo`、`/memory`、`/model` 等常用会话与管理动作；lite TUI 会输出 user/turn/model/tool/delegate/context/completed 等实时事件轨迹。
- 后续：继续分批补齐 web 数据页与完整 TUI 交互，目标是逐步对齐 Hermes 的前端与 TUI 体验。
- 使用文档：`docs/frontend-tui-user.md`
- 开发文档：`docs/frontend-tui-dev.md`
