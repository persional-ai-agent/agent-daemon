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
