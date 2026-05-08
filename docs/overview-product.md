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
- 非流式摘要：`/v1/chat` 返回轻量 `summary`
- 流式 API：基于 SSE 的 `/v1/chat/stream`
- 中断控制：支持按 `session_id` 取消活动中的 HTTP 会话
- 事件协议：已提供独立事件协议文档，便于前端或 SDK 对接
- 运行时提示词装配：每次运行都会重新注入 system prompt、持久记忆与工作区规则
- 基础安全护栏：文件工具限制在工作区内，terminal 会硬阻断灾难性命令
- 命令审批门禁：危险命令需显式 `requires_approval=true` 才可执行
- 上下文压缩：长会话超预算时自动压缩中段历史，保留头尾上下文

## 与 Hermes 的关系

本项目采用 Hermes 的核心设计思想，而不是逐文件翻译 Python 实现：

- 保留 Hermes 的“模型回合 -> 工具调用 -> 结果回灌 -> 再次推理”主循环
- 保留“工具注册中心 + schema 暴露 + handler 分发”模式
- 保留“会话历史 + 长期记忆 + Todo 状态”的状态分层
- 将多平台 Gateway、ACP、复杂插件作为后续扩展层

## 本期范围

- 已实现：核心闭环、system prompt 跨请求装配、记忆回灌、工作区规则注入、上下文压缩、常用内置工具、并发子 Agent 委派、结构化事件流、事件协议文档、持久化、CLI、HTTP API、`/v1/chat` 摘要、SSE、关键测试、基础安全护栏、危险命令审批门禁、MCP（http/stdio/OAuth client_credentials/`/call` 流式兼容与事件透传）、Provider（主备故障切换、熔断器、并行竞速、OpenAI/Anthropic/Codex 流式聚合、增量事件最小透传、`model_stream_event` v2+ 最小标准字典：`tool_args_start/delta/done`、`message_done.finish_reason`（`stop/tool_calls/length`）+ `stop_sequence/incomplete_reason`（`length` 场景自动补齐）、`usage.prompt_tokens/completion_tokens/total_tokens`（含缺失补齐与偏小校正标记）+ `usage_consistency_status`（`ok/derived/adjusted/source_only/invalid`）+ `prompt_cache_write_tokens/prompt_cache_read_tokens/reasoning_tokens`，并兼容多来源 `message_id/tool_call_id`）
- 未完全覆盖：Hermes 的多平台网关、MCP 高级能力（OAuth 授权码/刷新、流式事件透传已实现）、技能高级能力（同步/自动触发）、Provider 高级能力（并行竞速与熔断已实现；完整事件字典覆盖已补齐 `message_done.message_id`（Codex/Anthropic）、`message_done.incomplete_reason`（OpenAI/Anthropic/Codex）；OpenAI 的 `message_id` 和 `stop_sequence` 属于上游 API 限制暂不补齐）、审批状态持久化与细粒度审批策略（已实现：SQLite 持久化 + pattern 级细粒度授权；未实现：用户侧交互确认 UI）
