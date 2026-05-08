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
- 将多平台 Gateway、ACP、复杂插件作为后续扩展层

## 本期范围

- 已实现：核心闭环、系统提示词动态装配、记忆回灌、工作区规则注入、上下文压缩、16 个内置工具、并发子 Agent 委派、结构化事件流、持久化（SQLite）、CLI + HTTP API（同步/SSE/WebSocket）、安全护栏（hardline 阻断 + 审批门禁 + 交互确认 + pattern 级授权）、MCP（http/stdio/OAuth CC/授权码/`/call` 流式 + 事件透传）、技能（索引注入 + 条件过滤 + sync 同步 + 预加载 + GitHub 搜索）、Provider（OpenAI/Anthropic/Codex 流式聚合 + 故障切换 + 熔断 + 并行竞速 + 多级级联 + 成本感知 + `model_stream_event` v2 完整字典）、多平台网关（Telegram + Discord + Slack）
