# Frontend / TUI 开发文档

## 1. Web 架构

- 目录：`web/`
- 技术栈：Vite + React + TypeScript
- API 封装：`web/src/lib/api.ts`
- 页面入口：`web/src/App.tsx`
- 流式通道：默认 WS（`/v1/chat/ws`），失败可降级 SSE（`/v1/chat/stream`）
- 重连控制：Chat 页内置连接状态条与重连策略控制（wait/reconnect/cancel）

### 当前 API 契约

- `POST /v1/chat`
- `POST /v1/chat/stream`
- `POST /v1/chat/cancel`
- `GET /v1/ui/tools`
- `GET /v1/ui/tools/{name}/schema`
- `GET /v1/ui/sessions`
- `GET /v1/ui/sessions/{session_id}`
- `GET /v1/ui/config`
- `POST /v1/ui/config/set`
- `GET /v1/ui/gateway/status`
- `POST /v1/ui/gateway/action`
- `POST /v1/ui/approval/confirm`

`/v1/ui/*` 响应契约（冻结）：

- Header：`X-Agent-UI-API-Version`、`X-Agent-UI-API-Compat`
- Success：`{ ok: true, api_version, compat, ...payload }`
- Error：`{ ok: false, error: { code, message }, api_version, compat }`
- 机器可读契约：`docs/api/ui-chat-contract.openapi.yaml`
- 版本兼容策略：`docs/api/contract-versioning.md`

## 2. 后端注入点

`internal/api.Server` 通过回调注入管理面能力：

- `ConfigSnapshotFn`
- `GatewayStatusFn`
- `ConfigUpdateFn`
- `GatewayActionFn`

这些回调在 `cmd/agentd/main.go` 的 `runServe` 内绑定，避免 API 包直接依赖 CLI 逻辑。

## 3. CLI 类 TUI 架构

- 文件：`internal/cli/chat.go`
- 模式：单循环读取输入 + slash 命令分发 + `Engine.Run` 对话执行
- slash 命令扩展：集中在 `handleSlashCommand`
- 增强入口：`agentd tui` -> `internal/cli/tui.go`，通过 `EventSink` 输出实时事件轨迹（turn/tool/completed/error）
- `agentd tui` 增加模式参数：`-mode auto|standalone|lite`
  - `auto`（默认）：优先拉起独立 `ui-tui` 进程；不可用时回退到内置 lite
  - `standalone`：仅独立 `ui-tui`
  - `lite`：强制内置 `internal/cli/tui.go`
  - 启动优先级：`AGENT_UI_TUI_BIN` > `PATH(ui-tui)` > 仓库源码回退 `go run ./ui-tui`（仅仓库根目录可用）
  - `-fullscreen`：透传独立 `ui-tui` 全屏看板模式（等价设置 `AGENT_UI_TUI_FULLSCREEN=1`）

## 3.1 独立 TUI 子工程

- 目录：`ui-tui/`
- 运行方式：Go（`go run ./ui-tui`），连接 `/v1/chat/ws`
- 入口：`ui-tui/main.go`
- 目标：提供独立于 `agentd` 主进程交互循环的 TUI 客户端基座
- 稳定性机制：
  - WS 读取超时提示 + 单轮超时中断
  - 断线自动重连（同 session + `turn_id` + `resume`）
  - 审批确认直连 API（不依赖模型回合）：`/v1/ui/approval/confirm`
  - 错误分类码（`network/timeout/auth/request/server/unknown`）
  - 本地历史与事件日志滚动上限，避免无界增长
  - 启动自动 doctor 预检（可通过 `--no-doctor` 或 `[ui-tui] auto_doctor=false` 关闭）
  - 关键操作审计日志（approve/deny/cancel/config set）
  - 全屏看板模式（`--fullscreen` / `AGENT_UI_TUI_FULLSCREEN=1`）支持实时状态 + 最近事件 + 时间线
  - 快捷操作面板（`/actions`）支持编号选择高频管理动作，降低命令输入成本
  - 全屏多面板：`overview/dashboard/sessions/tools/approvals/gateway/diag`，支持 `/panel` 切换与 `/refresh` 拉取数据
  - 面板条目钻取：`/open <index>`，在 sessions/tools/approvals 面板执行上下文动作
  - 工作台配置方案：`/workbench save|list|load|delete`，用于保存/恢复完整工作台状态
  - 面板自动刷新策略：`/panel auto on|off` + `/panel interval <sec>`，并持久化到 `ui-tui-state.json`

## 4. 测试策略

- API 回归：`internal/api/server_test.go`
- CLI 命令回归：`internal/cli/chat_test.go`
- Web 单测：`npm --prefix web run test`
- Web 构建自测：`npm --prefix web run build`
- ui-tui 烟测：`./ui-tui/e2e_smoke.sh`
- ui-tui 发布脚本：`./ui-tui/release.sh <version>`

## 5. 运维文档

- `docs/ui-tui-ops.md`：ui-tui 专项故障排查与运维手册
