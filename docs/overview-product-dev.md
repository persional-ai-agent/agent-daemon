# Agent Daemon 开发概览

> 文档元数据
> - 文档版本：v1.0.0
> - 最后更新：2026-05-16
> - 更新来源：[docs/dev/1-task-docs-baseline.md](dev/1-task-docs-baseline.md)
> - 关联产品文档：docs/overview-product.md

## 1. 技术栈

| 类别 | 技术/版本 | 用途 | 备注 |
|------|-----------|------|------|
| 语言 | Go | daemon/cli/api/gateway/cron/plugin 核心实现 | 主工程位于 `cmd/` 与 `internal/` |
| 构建系统 | Make + Go Modules | 构建 `agent-daemon` 与 `agent-cli` | 根目录 `Makefile` |
| 数据存储 | SQLite（modernc.org/sqlite） | 会话、审批、摘要、cron 持久化 | 默认落盘到 `~/.agent-daemon` |
| Web 前端 | React + TypeScript + Vite | Dashboard 管理界面 | 子工程 `web/` |
| TUI 前端 | Go + Bubble Tea | 终端交互管理台 | 子工程 `ui-tui/` |
| 关键依赖 | gorilla/websocket、uuid、ini.v1、yaml.v3 | WS 通信、ID、配置、插件 manifest | 由 Go 模块统一管理 |

## 2. 架构边界

- 模块划分：
  - `cmd/agentd`：命令路由与流程编排（serve/chat/tui/tools/config/model/gateway/sessions/plugins/research/update/bootstrap/setup 等）。
  - `internal/agent`：Agent 执行循环、prompt/system prompt、上下文压缩。
  - `internal/api`：HTTP/WS 与 UI/API 路由层。
  - `internal/gateway`：多平台 adapter 接入、会话映射、命令规范化、事件分发。
  - `internal/tools`：内置工具、审批、会话工具、网关工具、流程工具。
  - `internal/plugins`：插件加载、manifest 校验、安装与执行。
  - `internal/store`：SQLite 持久化层（session/cron）。
  - `internal/model`：模型 provider、级联回退、流式、熔断/竞速逻辑。
  - `internal/cronrunner`：定时调度与执行编排。
- 进程边界：`agent-daemon` 后台服务；`agent-cli` 与 `ui-tui` 为客户端入口。
- 数据流：输入入口 -> API/Gateway -> Agent Engine -> Model/Tools -> Store -> 响应出口。
- 控制流：CLI 子命令或 HTTP 路由触发，统一落到引擎与持久化/插件/网关模块。
- 外部依赖：模型提供方 API、各网关平台 webhook/API、插件外部命令。

## 3. 关键接口

| 接口/协议/ABI | 调用方 | 提供方 | 兼容约束 | 说明 |
|---------------|--------|--------|----------|------|
| `/v1/chat` `/v1/chat/stream` `/v1/chat/ws` | CLI/Web/TUI | `internal/api` + `internal/agent` | 请求/响应字段需保持向后兼容 | 会话对话执行主接口 |
| `/v1/ui/*` | Web/TUI | `internal/api` | 页面依赖 contract snapshot 测试 | 会话、工具、模型、网关、Cron、审批、技能、语音管理 |
| `/v1/gateway/*/webhook` | 外部平台 | `internal/api` + `internal/gateway/platforms` | 各平台签名/字段约束不可破坏 | 平台消息入站 |
| 插件 manifest（json/yaml） | 插件作者/CLI | `internal/plugins` | schema 与安全字段兼容 | 扩展工具/provider/command/dashboard |
| SQLite schema（messages/cron_* 等） | 存储层 | `internal/store` | 迁移只增不破坏 | 会话与定时任务持久化 |

## 4. 数据与配置

- 核心数据结构：`core.Message`、`store.CronJob`、`store.CronRun`、`plugins.Manifest`。
- 配置文件/参数：
  - 支持 `config/config.ini`、`config.ini`、`AGENT_CONFIG_FILE`。
  - 环境变量优先于 ini（模型、网关、cron、ui-tui 等配置均可覆盖）。
- 持久化数据：
  - `sessions.db`：消息、审批、oauth token、会话摘要。
  - `cron_runs.db`（或同库表）：cron job/run 记录。
  - 运行状态与辅助文件：`~/.agent-daemon` 下 state/spool/audit 等。
- 迁移/兼容规则：`agentd setup migrate` 支持从历史目录导入，并可创建 checkpoint 回滚。
- 敏感信息处理：API key/token 以配置/环境变量注入；插件可声明 sandbox 与签名校验。

## 5. 高风险区域

| 风险区域 | 关注点 | 验证方式 | 关联文档 |
|----------|--------|----------|----------|
| 并发/锁 | gateway session worker、cron 并发槽与 cancel 控制 | 对应单测 + 关键流程回归 | [docs/dev/1-task-docs-baseline.md](dev/1-task-docs-baseline.md) |
| API/协议 | `/v1/ui/*` 与 gateway webhook contract 兼容 | contract snapshot/replay 测试 | [docs/dev/1-task-docs-baseline.md](dev/1-task-docs-baseline.md) |
| 数据一致性 | SQLite schema 迁移与 session/cron 写入时序 | store 层测试 + 迁移路径测试 | [docs/dev/1-task-docs-baseline.md](dev/1-task-docs-baseline.md) |
| 外部执行 | 插件命令、script cron、工具命令风险 | manifest 验证 + sandbox 策略 + 审批流 | [docs/dev/1-task-docs-baseline.md](dev/1-task-docs-baseline.md) |
| 升级/回滚 | update bundle apply/rollback、setup migrate checkpoint | bundle 命令链路验证 | [docs/dev/1-task-docs-baseline.md](dev/1-task-docs-baseline.md) |

## 6. 构建与验证

- 构建命令：`make`（根目录）；`go build ./cmd/agentd`。
- 单元测试：`go test ./...`（主仓）；`go test ./...`（`ui-tui/` 子工程）。
- 前端验证：`cd web && npm run build`（或 `npm run dev` + smoke）。
- 集成验证：CLI 子命令 smoke（chat/sessions/tools/gateway/cron/plugins）。
- 高风险验证：涉及 gateway/cron/store/model 时优先跑对应模块测试。
- 最小人工验证步骤：启动 `agentd serve`，通过 Web/TUI 完成一轮会话、会话查看、取消与网关状态查询。

## 7. 发布与回滚

- 产物：`agent-daemon`、`agent-cli` 二进制；`web/dist` 静态前端；`ui-tui` 可独立发布。
- 安装/部署方式：本地二进制运行；配置由 ini/env 提供。
- 配置变更：优先走 `config set`/`model set`/`setup` 命令。
- 升级步骤：`agentd update check/apply` 或 bundle 子命令流程。
- 回滚步骤：`agentd update bundle rollback`；迁移场景可用 `setup migrate rollback`。
- 止损条件：健康检查失败、关键 API 合约不兼容、会话或网关核心链路不可用。

## 8. 观测与排障

- 关键日志：daemon 标准日志、gateway 连接日志、cron 调度与执行日志。
- 指标/告警：当前以日志与诊断接口为主，未内置独立 metrics 后端。
- 常见故障：模型凭证缺失、网关签名不匹配、插件 manifest 校验失败、Cron 参数非法。
- 排障入口：`agentd doctor`、`/health`、`/v1/ui/gateway/diagnostics`、Web/TUI `diag.v1` 导出。

## 9. 文档索引

- 需求与任务索引：docs/dev/README.md
- 产品概览：docs/overview-product.md
- 按需片段模板：.dj-agent/fragments/
- 关键任务文档：
  - [1-task-docs-baseline.md](dev/1-task-docs-baseline.md)：文档基线补齐。

## 10. 变更记录

| 日期 | 变更 | 影响 | 关联文档 |
|------|------|------|----------|
| 2026-05-16 | 首次补齐开发概览，沉淀模块边界、关键接口与运维链路 | 为后续任务提供稳定架构基线 | [docs/dev/1-task-docs-baseline.md](dev/1-task-docs-baseline.md) |
