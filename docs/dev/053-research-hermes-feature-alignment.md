# 053 Research：Hermes 功能对齐复核

## 背景

用户要求分析 `/data/code/agent-daemon` 是否与 `/data/source/hermes-agent` 功能对齐，并在需要时完善文档。

本次复核基于本地源码与文档，不联网：

- 当前项目：`README.md`、`docs/overview-product.md`、`docs/overview-product-dev.md`、`internal/*`
- Hermes 参考：`README.md`、`website/docs/developer-guide/architecture.md`、`website/docs/developer-guide/tools-runtime.md`、`website/docs/developer-guide/provider-runtime.md`、`website/docs/developer-guide/gateway-internals.md`、`website/docs/reference/tools-reference.md`

## 结论

当前项目与 Hermes 的“核心 Agent daemon 子集”对齐，但不等同于 Hermes Agent 的完整功能面。

已对齐的主干能力：

- 多轮 Agent Loop：模型响应、工具调用、结果回灌、继续推理。
- 工具注册与分发：统一 schema、统一 registry、统一 JSON 结果。
- 模型模式：OpenAI Chat Completions、Anthropic Messages、Codex Responses 三类主路径。
- Provider 韧性：主备 fallback、熔断、竞速、cascade、成本感知、流式事件标准化。
- 状态：SQLite 会话、`MEMORY.md` / `USER.md`、session search、todo。
- 上下文：运行时 system prompt 重建、工作区规则注入、长会话压缩。
- 安全：工作区路径收敛、危险命令检测、硬阻断、会话/模式级审批、交互确认。
- MCP：HTTP、stdio、OAuth client credentials、OAuth authorization code、`/call` 流式事件透传。
- Skills：本地列表/查看/管理、支撑文件、条件过滤、预加载、GitHub 搜索与同步。
- 服务入口：CLI、HTTP `/v1/chat`、SSE `/v1/chat/stream`、WebSocket `/v1/chat/ws`。
- Gateway 最小层：Telegram、Discord、Slack 适配器与流式编辑输出。

未对齐或仅最小覆盖的 Hermes 功能：

- TUI 与完整 CLI 命令体系：Hermes 有全屏 TUI、slash commands、模型/工具/配置/setup/doctor/update 等命令；当前项目只有 `chat`、`serve`、`tools`。
- Provider 生态：Hermes 通过插件覆盖 OpenRouter、Nous、Gemini、DeepSeek、Kimi、MiniMax、HuggingFace、Copilot、Bedrock、自定义 provider 等；当前项目只内置 OpenAI、Anthropic、Codex 三类。
- 工具全集：Hermes 文档列出 68 个内置工具和 52 个 toolsets；当前项目内置约 17 个工具，缺 browser、browser-cdp、code execution、cronjob、vision、tts、messaging、Home Assistant、Feishu、Spotify、Yuanbao、RL 等。
- 终端运行环境：Hermes 支持 local、Docker、SSH、Singularity、Modal、Daytona、Vercel Sandbox；当前项目只实现本地 Linux 执行。
- Gateway 完整能力：Hermes 有 20 个左右平台、DM pairing、slash command、运行中断/队列、delivery、hooks、token lock、跨平台发送等；当前项目是 Telegram/Discord/Slack 的最小消息入口。
- 插件系统：Hermes 支持工具、模型 provider、memory provider、context engine、dashboard 等插件；当前项目无通用插件加载框架。
- ACP / IDE 集成：Hermes 有 ACP adapter；当前项目无 ACP。
- Cron / 自动化：Hermes 有 cron scheduler 和平台投递；当前项目无 cron。
- 记忆与检索高级能力：Hermes 有 memory provider 插件、Honcho 等用户建模、FTS5 与 LLM 摘要式 session search；当前项目是 Markdown 记忆 + SQLite LIKE 搜索。
- Web/TUI dashboard：Hermes 有 Web 和 TUI 前端；当前项目只提供 HTTP API。
- 研究/训练链路：Hermes 有 batch runner、Atropos/RL environments、trajectory compression；当前项目无训练/数据生成模块。

## 文档缺口

现有 `docs/overview-product.md` 与 `docs/overview-product-dev.md` 对已实现能力记录较完整，但存在两个问题：

- “核心能力已对齐 Hermes”等表述容易被理解为完整功能对齐。
- 缺少一张稳定的功能对齐矩阵，后续继续补功能时难以判断优先级与边界。

## 推荐方案

本次只做文档完善，不做功能实现：

- 在产品总览中补充“对齐状态”和“暂未覆盖能力”，把当前项目定位为 Hermes 核心 daemon 子集。
- 在开发总览中补充功能矩阵和后续补齐建议，明确哪些是已实现、最小覆盖、未实现。
- 新增本次 Research / Plan / Summary，并更新 `docs/dev/README.md`。

## 三角色审视

- 高级产品：边界澄清能直接解决“是否对齐”的问题，避免用户误以为已完整复刻 Hermes。
- 高级架构师：不引入新依赖，不改变代码结构，只沉淀功能矩阵。
- 高级工程师：只改文档，风险低；验证以文件内容和 diff 为主。
