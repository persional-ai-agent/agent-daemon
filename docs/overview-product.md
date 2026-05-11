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
- 配置管理：`agentd config list|get|set` 可读写 `config/config.ini`，`agentd model show|providers|set` 可查看和切换模型，环境变量仍保持最高优先级
- 消息网关：Telegram + Discord + Slack 适配器，`PlatformAdapter` 接口可按需扩展
- 非流式摘要：`/v1/chat` 返回轻量 `summary`
- 流式 API：基于 SSE 的 `/v1/chat/stream`
- 中断控制：支持按 `session_id` 取消活动中的 HTTP 会话
- 事件协议：已提供独立事件协议文档，便于前端或 SDK 对接
- 运行时提示词装配：每次运行都会重新注入 system prompt、持久记忆与工作区规则
- 基础安全护栏：文件工具限制在工作区内，terminal 会硬阻断灾难性命令
- 命令审批门禁：危险命令需显式 `requires_approval=true` 才可执行
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
| 工具注册与分发 | 已对齐 | 统一 schema、registry、dispatch、JSON 结果 |
| 模型 API 模式 | 部分对齐 | 内置 OpenAI、Anthropic、Codex；未覆盖 Hermes 的 provider 插件生态 |
| Provider 韧性 | 已对齐核心能力 | 已有 fallback、熔断、竞速、cascade、成本感知与流式事件标准化 |
| 会话与记忆 | 部分对齐 | SQLite 会话、Markdown 记忆、session search；未实现 Hermes 的 FTS5 + LLM 摘要式检索和 memory provider 插件 |
| 上下文管理 | 已对齐核心能力 | system prompt 重建、工作区规则注入、长会话压缩 |
| MCP | 已对齐核心能力 | 支持 HTTP、stdio、OAuth client credentials、OAuth authorization code 与流式事件透传 |
| Skills | 已对齐核心能力 | 支持本地列表/查看/管理、条件过滤、预加载、同步与 GitHub 搜索 |
| API 服务 | 已对齐核心能力 | HTTP、SSE、WebSocket、取消接口 |
| Gateway | 最小对齐 | 支持 Telegram、Discord、Slack；未覆盖 Hermes 的完整平台矩阵、配对、slash command、队列/中断、delivery、hooks |
| CLI/TUI | 最小覆盖 | 已有交互式 chat、serve、tools、config、model；未实现 Hermes 全屏 TUI 与完整命令体系 |
| 工具全集 | 最小覆盖 | 当前内置核心工具；未覆盖 browser、code execution、cron、vision、tts、messaging、Home Assistant、Feishu、Spotify、RL 等 Hermes 工具集 |
| 终端环境 | 最小覆盖 | 当前为本地 Linux 执行；未覆盖 Docker、SSH、Modal、Daytona、Singularity、Vercel Sandbox |
| 插件/ACP/Cron/训练 | 未覆盖 | 暂无通用插件系统、ACP adapter、cron scheduler、batch/RL/trajectory 链路 |

## 暂未覆盖能力

以下能力属于 Hermes 完整产品体验的一部分，但当前项目未实现或只保留最小骨架：

- 全屏 TUI、slash commands、工具/setup/doctor/update 等 CLI 管理面
- 18+ provider 与 provider 插件加载机制
- 68 个 Hermes 内置工具、52 个 toolsets 与按平台/环境动态过滤
- browser、browser-cdp、code execution、cronjob、vision、tts、messaging、Home Assistant、Feishu、Spotify、Yuanbao、RL 等工具域
- Docker、SSH、Singularity、Modal、Daytona、Vercel Sandbox 等终端后端
- 多平台 Gateway 的 DM pairing、运行中断/队列、跨平台 delivery、hooks、token lock 和更多平台适配器
- 通用插件系统、ACP/IDE 集成、Cron 自动化、Web/TUI dashboard、研究/训练数据链路

## 当前范围

- 已实现：核心闭环、系统提示词动态装配、记忆回灌、工作区规则注入、上下文压缩、17 个内置工具、并发子 Agent 委派、结构化事件流、持久化（SQLite）、CLI + HTTP API（同步/SSE/WebSocket）、CLI 配置管理（`config list|get|set` + `model show|providers|set`）、安全护栏（hardline 阻断 + 审批门禁 + 交互确认 + pattern 级授权）、MCP（http/stdio/OAuth CC/授权码/`/call` 流式 + 事件透传）、技能（索引注入 + 条件过滤 + sync 同步 + 预加载 + GitHub 搜索）、Provider（OpenAI/Anthropic/Codex 流式聚合 + 故障切换 + 熔断 + 并行竞速 + 多级级联 + 成本感知 + `model_stream_event` v2 完整字典）、多平台网关（Telegram + Discord + Slack）
