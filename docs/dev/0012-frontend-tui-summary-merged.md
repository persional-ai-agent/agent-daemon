# 0012 frontend-tui summary merged

## 模块

- `frontend-tui`

## 类型

- `summary`

## 合并来源

- `0019-frontend-tui-summary-merged.md`

## 合并内容

### 来源：`0019-frontend-tui-summary-merged.md`

# 0019 frontend-tui summary merged

## 模块

- `frontend-tui`

## 类型

- `summary`

## 合并来源

- `0207-frontend-tui-parity-phase1.md`
- `0208-frontend-tui-parity-phase2-ui-api.md`
- `0209-frontend-tui-parity-phase3-stream-timeline.md`
- `0210-frontend-tui-parity-phase4-detail-pages.md`
- `0211-frontend-tui-parity-phase5-config-gateway-actions.md`
- `0212-frontend-tui-parity-phase6-usability-and-docs.md`
- `0213-frontend-tui-parity-phase7-tui-entry.md`
- `0214-frontend-tui-parity-phase8-standalone-ui-tui.md`
- `0215-frontend-tui-parity-phase9-ui-tui-commands.md`

## 合并内容

### 来源：`0207-frontend-tui-parity-phase1.md`

# Frontend 与 TUI 对齐总结（Phase 1）

## 本阶段完成

1. 新增 `web/` 工程基座（Vite + React + TypeScript）：
   - `Chat / Sessions / Tools / Gateway / Config` 五页入口已建立。
   - Chat 页已打通 `/v1/chat` 与 `/v1/chat/cancel`。
2. 增强 CLI 交互层：
   - `internal/cli/chat.go` 增加 slash 命令：`/help`、`/session`、`/tools`、`/history`、`/reload`、`/clear`、`/tui`。
3. 新增 CLI 测试：
   - `internal/cli/chat_test.go` 覆盖 `/clear` 与 `/reload` 核心行为。
4. 更新产品/开发总览文档，新增 Frontend/TUI 迭代状态说明。

## 验证

- Go 侧执行 `go test ./...` 应通过（本阶段未引入 Go 依赖变更）。
- Web 侧按 `web/README.md` 可本地启动并手动验证 chat/cancel 链路。

## 后续批次

1. 补齐 `sessions/tools/gateway/config` 页面真实数据接口与操作能力。
2. 增加前端会话流式展示（SSE/WebSocket）与工具调用时间线视图。
3. 评估并引入完整 TUI 框架（或继续增强现有 CLI 交互层）。

### 来源：`0208-frontend-tui-parity-phase2-ui-api.md`

# Frontend 与 TUI 对齐总结（Phase 2：UI API 联通）

## 本阶段完成

1. 新增后端 UI API：
   - `GET /v1/ui/tools`
   - `GET /v1/ui/sessions?limit=N`
   - `GET /v1/ui/config`
   - `GET /v1/ui/gateway/status`
2. `runServe` 注入 dashboard 所需快照回调（配置快照、网关状态快照）。
3. `web` 四个骨架页完成数据联通：
   - `sessions` 展示最近会话
   - `tools` 展示可用工具清单
   - `gateway` 展示网关状态
   - `config` 展示运行配置快照
4. 新增 API 回归测试：`internal/api/server_test.go`。

## 验证

- `go test ./...` 通过。
- `npm --prefix web run build` 通过。

### 来源：`0209-frontend-tui-parity-phase3-stream-timeline.md`

# Frontend 与 TUI 对齐总结（Phase 3：流式会话与时间线）

## 本阶段完成

1. Chat 页面新增流式模式开关：
   - 支持 `POST /v1/chat/stream` 的 SSE 消费。
2. 新增事件时间线（Timeline）：
   - 展示 `session/user_message/turn_started/tool_started/tool_finished/completed/result` 等事件序列。
3. 在 `result` 事件自动提取 `final_response` 回填主输出区。

## 验证

- `npm --prefix web run build` 通过。
- `go test ./...` 通过（后端逻辑未回归）。

### 来源：`0210-frontend-tui-parity-phase4-detail-pages.md`

# Frontend 与 TUI 对齐总结（Phase 4：详情页与可点击交互）

## 本阶段完成

1. 新增后端详情接口：
   - `GET /v1/ui/sessions/{session_id}?offset&limit`
   - `GET /v1/ui/tools/{tool_name}/schema`
2. 前端 `sessions/tools` 页面从纯 JSON 列表升级为可点击详情页：
   - sessions：左侧会话列表，右侧消息与统计详情。
   - tools：左侧工具列表，右侧 schema 明细。
3. 补齐 API 回归测试，覆盖新增详情接口。

## 验证

- `go test ./...` 通过。
- `npm --prefix web run build` 通过。

### 来源：`0211-frontend-tui-parity-phase5-config-gateway-actions.md`

# Frontend 与 TUI 对齐总结（Phase 5：配置与网关动作）

## 本阶段完成

1. 新增后端可操作接口：
   - `POST /v1/ui/config/set`：写入指定 `section.key=value`
   - `POST /v1/ui/gateway/action`：网关启用/禁用动作（`enable|disable`）
2. 前端 `gateway/config` 页面升级为可操作页：
   - gateway 页支持“启用网关/禁用网关”按钮。
   - config 页支持键值写入并刷新快照。
3. 新增 API 回归测试覆盖动作接口。

## 验证

- `go test ./...` 通过。
- `npm --prefix web run build` 通过。

### 来源：`0212-frontend-tui-parity-phase6-usability-and-docs.md`

# Frontend 与 TUI 对齐总结（Phase 6：可用性增强与文档补齐）

## 本阶段完成

1. CLI 类 TUI 增强：
   - 新增 `/sessions`、`/stats`、`/show` 命令。
2. Web 可用性增强：
   - sessions 页支持分页切换与刷新。
   - tools 页支持筛选与当前选中标识。
   - gateway/config 页补充刷新按钮。
3. 文档补齐：
   - 新增使用文档：`docs/frontend-tui-user.md`
   - 新增开发文档：`docs/frontend-tui-dev.md`

## 验证

- `go test ./...` 通过。
- `npm --prefix web run build` 通过。

### 来源：`0213-frontend-tui-parity-phase7-tui-entry.md`

# Frontend 与 TUI 对齐总结（Phase 7：独立 TUI 入口与实时事件轨迹）

## 本阶段完成

1. 新增 `agentd tui` 命令入口（独立于 `agentd chat`）。
2. 新增 `internal/cli/tui.go`：
   - 通过 `Engine.EventSink` 输出实时事件（turn/tool/completed/error）。
   - 保持与现有 chat/slash 命令兼容。
3. 更新使用文档与开发文档，明确 `tui` 模式用途与实现位置。

## 验证

- `go test ./...` 通过。

### 来源：`0214-frontend-tui-parity-phase8-standalone-ui-tui.md`

# Frontend 与 TUI 对齐总结（Phase 8：独立 ui-tui 子工程）

## 本阶段完成

1. 新增独立 `ui-tui/` 子工程：
   - `ui-tui/main.go`：WebSocket 客户端循环，发送消息并显示事件流。
   - `ui-tui/README.md`：启动说明与环境变量说明。
2. 补齐用户/开发文档，对独立 TUI 入口给出明确路径。

## 验证

- Go 主仓：`go test ./...` 通过（无回归）。
- `ui-tui` 工程可本地 `go run ./ui-tui` 启动。

### 来源：`0215-frontend-tui-parity-phase9-ui-tui-commands.md`

# Frontend 与 TUI 对齐总结（Phase 9：ui-tui 命令体系增强）

## 本阶段完成

1. 增强 `ui-tui` 命令体系：
   - `/help`
   - `/session` / `/session <id>`
   - `/api` / `/api <ws-url>`
   - `/quit`
2. 支持运行时切换会话 ID 与 WebSocket 地址，减少重启成本。
3. 更新 `ui-tui/README.md` 的命令说明。

## 验证

- `go test ./...` 通过（无回归）。
