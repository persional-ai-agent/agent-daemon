# Frontend / TUI 开发文档

## 1. Web 架构

- 目录：`web/`
- 技术栈：Vite + React + TypeScript
- API 封装：`web/src/lib/api.ts`
- 页面入口：`web/src/App.tsx`

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

## 4. 测试策略

- API 回归：`internal/api/server_test.go`
- CLI 命令回归：`internal/cli/chat_test.go`
- Web 构建自测：`npm --prefix web run build`
