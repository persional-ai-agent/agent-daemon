# 0042 ui-tui summary merged

## 模块

- `ui-tui`

## 类型

- `summary`

## 合并来源

- `0052-ui-summary-merged.md`
- `0053-ui-tui-summary-merged.md`

## 合并内容

### 来源：`0052-ui-summary-merged.md`

# 0052 ui summary merged

## 模块

- `ui`

## 类型

- `summary`

## 合并来源

- `0232-ui-api-contract-freeze.md`

## 合并内容

### 来源：`0232-ui-api-contract-freeze.md`

# UI API 契约冻结（/v1/ui/*）

本轮将 `/v1/ui/*` 管理接口统一为稳定契约，并补齐版本兼容与错误码规范。

## 契约统一

- 所有 `/v1/ui/*` 成功响应统一包含：
  - `ok: true`
  - `api_version: "v1"`
  - `compat: "2026-05-13"`
- 所有 `/v1/ui/*` 错误响应统一包含：
  - `ok: false`
  - `error.code`
  - `error.message`
  - `api_version` / `compat`
- 所有 `/v1/ui/*` 响应统一携带 Header：
  - `X-Agent-UI-API-Version: v1`
  - `X-Agent-UI-API-Compat: 2026-05-13`

## 错误码规范

- `method_not_allowed`
- `invalid_json`
- `invalid_argument`
- `not_found`
- `not_supported`
- `engine_unavailable`
- `session_store_unavailable`
- `internal_error`
- `tool_error`

## 兼容策略

- `api_version` 用于主版本识别（当前 `v1`）。
- `compat` 用于契约冻结日期标识（当前 `2026-05-13`）。
- 新增字段只允许向后兼容扩展，不破坏既有字段含义。

## 契约测试

- 新增 `internal/api/ui_contract_test.go`：
  - 验证成功 envelope + headers
  - 验证错误 envelope（`ok=false` + `error.code/message`）

### 来源：`0053-ui-tui-summary-merged.md`

# 0053 ui-tui summary merged

## 模块

- `ui-tui`

## 类型

- `summary`

## 合并来源

- `0216-ui-tui-migrate-to-go.md`
- `0217-ui-tui-parity-management-commands.md`
- `0218-ui-tui-ux-output-controls.md`
- `0219-ui-tui-pagination-shortcuts.md`
- `0220-ui-tui-quick-session-pick.md`
- `0221-ui-tui-command-aliases.md`
- `0222-ui-tui-command-status.md`
- `0223-ui-tui-runtime-ops-and-persistence.md`
- `0224-ui-tui-reliability-observability-e2e.md`
- `0225-ui-tui-recovery-approval-alignment.md`
- `0226-ui-tui-config-ini-unification.md`
- `0227-ui-tui-usability-finalization.md`
- `0228-ui-tui-final-audit-and-baseline.md`
- `0230-ui-tui-doctor-command.md`
- `0231-ui-tui-efficiency-observability-bundle.md`
- `0233-ui-tui-adapt-ui-api-contract.md`
- `0245-ui-tui-reconnect-controls.md`
- `0249-ui-tui-realtime-diagnostics-parity.md`
- `0263-ui-tui-fullscreen-dashboard-mode.md`
- `0264-ui-tui-fullscreen-timeline-and-runtime-toggle.md`
- `0265-ui-tui-actions-palette.md`
- `0266-ui-tui-fullscreen-quiet-and-timeline-command.md`
- `0267-ui-tui-fullscreen-multi-panel.md`
- `0268-ui-tui-workbench-completion-bundle.md`
- `0269-ui-tui-workbench-auto-refresh-and-persistence.md`
- `0270-ui-tui-workbench-drilldown-and-approvals-panel.md`
- `0271-ui-tui-workbench-open-action-closure.md`
- `0272-ui-tui-workbench-profiles-bundle.md`
- `0273-ui-tui-workflow-orchestration-bundle.md`

## 合并内容

### 来源：`0216-ui-tui-migrate-to-go.md`

# ui-tui 迁移总结（Go 实现）

## 变更

1. `ui-tui` 从 Node.js 实现迁移到 Go 实现。
2. 删除：
   - `ui-tui/package.json`
   - `ui-tui/src/index.mjs`
3. 新增：
   - `ui-tui/main.go`（WebSocket 交互式客户端）
4. 文档同步改为 `go run ./ui-tui` 启动方式。

## 验证

- `go test ./...` 通过。
- `go run ./ui-tui` 可启动（需后端服务可用）。

### 来源：`0217-ui-tui-parity-management-commands.md`

# ui-tui 对齐总结（管理命令补齐）

## 本阶段完成

在 Go 版 `ui-tui` 中补齐管理面命令，使其可覆盖 Web 管理页的核心操作：

1. 工具管理：`/tools`、`/tool <name>`
2. 会话管理：`/sessions [n]`、`/show [sid] [offset] [limit]`、`/stats [sid]`
3. 网关管理：`/gateway status|enable|disable`
4. 配置管理：`/config get`、`/config set <section.key> <value>`
5. 连接管理：`/http`、`/http <http-url>`（配合已有 `/api`）

## 结果

`ui-tui` 不依赖前端界面即可完成对话与运维管理的闭环操作。

## 验证

- `go test ./...` 通过。

### 来源：`0218-ui-tui-ux-output-controls.md`

# ui-tui 体验增强总结（输出控制）

## 本阶段完成

1. 新增输出控制命令：
   - `/pretty on|off`：切换 JSON 输出格式
   - `/last`：回看最近一次 JSON 响应
   - `/save <file>`：保存最近一次 JSON 响应
2. 管理命令输出统一走可切换的格式化逻辑。

## 结果

Go 版 `ui-tui` 在纯终端下具备更稳定的“查看-复用-落盘”闭环体验。

## 验证

- `go test ./...` 通过。

### 来源：`0219-ui-tui-pagination-shortcuts.md`

# ui-tui 体验增强总结（分页快捷翻页）

## 本阶段完成

1. 在 Go 版 `ui-tui` 中新增会话分页快捷命令：
   - `/next`
   - `/prev`
2. 基于最近一次 `/show [sid] [offset] [limit]` 记住会话与分页上下文，支持连续翻页。

## 结果

会话浏览从“每次手输 offset/limit”升级为“show 一次后快捷翻页”，交互效率提升。

## 验证

- `go test ./...` 通过。

### 来源：`0220-ui-tui-quick-session-pick.md`

# ui-tui 体验增强总结（快速会话选择）

## 本阶段完成

1. 新增 `/pick <index>` 命令：
   - 基于最近一次 `/sessions [n]` 返回的会话列表按序号快速切换会话。
2. `ui-tui` 内部维护最近会话缓存，减少手动复制 `session_id`。

## 结果

会话切换从“复制粘贴 session_id”优化为“列表 + 序号选择”。

## 验证

- `go test ./...` 通过。

### 来源：`0221-ui-tui-command-aliases.md`

# ui-tui 体验增强总结（命令别名与容错输入）

## 本阶段完成

1. 新增命令别名与无斜杠输入容错：
   - `:q` / `quit` -> `/quit`
   - `ls` -> `/tools`
   - `show ...` -> `/show ...`
   - `gw` / `gw ...` -> `/gateway status` / `/gateway ...`
   - `cfg` / `cfg ...` -> `/config get` / `/config ...`
2. `help` 输出同步显示别名提示。

## 结果

在纯终端环境下减少输入负担，提升高频操作效率。

## 验证

- `go test ./...` 通过。

### 来源：`0222-ui-tui-command-status.md`

# ui-tui 体验增强总结（命令状态提示）

## 本阶段完成

1. 新增 `/status` 命令，显示最近一次命令执行状态与详情。
2. 提示符升级为 `tui[ok]>` / `tui[err]>`，即时反映最近一次命令结果。
3. 对关键命令补齐状态更新（成功/失败）。

## 验证

- `go test ./...` 通过。

### 来源：`0223-ui-tui-runtime-ops-and-persistence.md`

# ui-tui 运行控制与持久化能力补齐（Phase 10）

本阶段在纯 Go 的 `ui-tui` 上继续补齐运行控制、可追溯性和会话快捷管理能力，目标是让终端端产品面在不依赖前端页面的前提下，覆盖常见操作闭环。

## 新增能力

- 运行控制：
  - `/health`：查看后端健康状态
  - `/cancel`：取消当前会话中的运行任务
- 可追溯能力：
  - `/history [n]`：查看本地命令历史
  - `/rerun <index>`：按历史序号重放命令
  - `/events [n]`：查看最近运行事件
  - `/events save <file>`：导出运行事件日志
- 会话快捷管理：
  - `/bookmark add <name> [sid]`
  - `/bookmark list`
  - `/bookmark use <name>`
- 交互反馈增强：
  - 提示符显示最近命令状态（`tui[ok]` / `tui[err]`）
  - `/status` 输出最近一次命令状态与摘要

## 持久化行为

- 历史命令写入 `~/.agent-daemon/ui-tui-history.log`
- 书签写入 `~/.agent-daemon/ui-tui-bookmarks.json`
- 事件日志默认驻留内存，可按需落盘

## 验证

- 已执行：`go test ./...`
- 结果：通过

### 来源：`0224-ui-tui-reliability-observability-e2e.md`

# ui-tui 稳定性与可观测性一次性补齐（Phase 11）

本阶段一次性补齐了 ui-tui 在长连接稳定性、诊断可观测性、长会话容量治理与回归自测上的缺口，全部基于 Go 实现，不依赖前端页面。

## 补齐内容

- 事件流健壮性：
  - WebSocket 断线自动重连（最多 2 次）
  - 重连保持同 `session_id`，并携带 `turn_id` 与 `resume` 标志
  - 45s 读超时提示（等待中），8m 单轮超时中断
- 长会话性能：
  - 本地历史命令文件滚动上限 2000 行
  - 内存事件日志滚动上限 2000 条
  - `/history`、`/events` 的读取请求自动受上限裁剪
- 错误可诊断性：
  - 统一错误分类：`network/timeout/auth/request/server/unknown`
  - 提示符显示 `状态/错误码`，`/status` 输出 `status/code/detail`
- 端到端回归：
  - 新增 `ui-tui/e2e_smoke.sh`，覆盖命令面烟测与可选后端健康联通路径
- 文档补齐：
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./...`：通过
- `./ui-tui/e2e_smoke.sh`：通过（后端不可达时自动跳过 health 联通子场景）

### 来源：`0225-ui-tui-recovery-approval-alignment.md`

# ui-tui 收口对齐（会话恢复 + 审批闭环 + 导出过滤 + 回归）

本次完成 ui-tui 收口对齐，聚焦四个目标：会话恢复、审批闭环、事件导出标准化、可靠性回归扩展。

## 功能补齐

- 会话恢复：
  - 新增 `~/.agent-daemon/ui-tui-state.json`
  - 启动自动恢复最近 `session_id`、`ws_base`、`http_base`
  - 会话/endpoint 切换后自动持久化
- 审批闭环：
  - `ui-tui` 新增 `/pending`、`/approve <id>`、`/deny <id>`
  - `/pending` 从会话消息中提取最近 `pending_approval` 与 `approval_id`
  - 新增后端接口 `POST /v1/ui/approval/confirm`，直连 approval 工具确认，避免依赖模型回合
- 导出标准化：
  - `/events save` 支持 `json|ndjson`
  - 支持 `since=<RFC3339>`、`until=<RFC3339>` 时间范围过滤
- 回归扩展：
  - 新增 `ui-tui/main_test.go`：
    - 断线重连恢复（`resume`）回归
    - pending approval 提取回归
    - 导出参数/时间过滤回归
  - `ui-tui/e2e_smoke.sh` 增加上述回归测试执行

## 验证

- `go test ./...`：通过
- `./ui-tui/e2e_smoke.sh`：通过

### 来源：`0226-ui-tui-config-ini-unification.md`

# ui-tui 配置统一到 config.ini（Phase 12）

本次将 ui-tui 关键运行参数统一接入 `config/config.ini`，并新增 `[ui-tui]` 配置段，确保终端端参数可集中管理。

## 变更内容

- 配置文件新增 `[ui-tui]` 示例项：
  - `ws_base`
  - `http_base`
  - `ws_read_timeout_seconds`
  - `ws_turn_timeout_seconds`
  - `ws_reconnect_max`
  - `history_max_lines`
  - `event_max_items`
- `ui-tui/main.go` 改为从 `internal/config.Load()` 读取上述配置（环境变量仍可覆盖）。
- 配置加载路径增强：新增 `../config/config.ini` 兜底，适配在 `ui-tui/` 子目录执行 `go run .` 的场景。

## 测试

- `internal/config/config_test.go` 新增：
  - `[ui-tui]` 配置读取验证
  - `../config/config.ini` 路径发现验证
- 全量验证：
  - `go test ./...`
  - `./ui-tui/e2e_smoke.sh`

### 来源：`0227-ui-tui-usability-finalization.md`

# ui-tui 易用性收口（Phase 13）

本轮一次性完成 ui-tui 易用性收口，覆盖审批体验、配置热更新、状态文件自修复与运维文档。

## 功能补齐

- 审批体验：
  - `/pending [n]`：支持查看最近 N 条待审批项
  - `/approve [id]` / `/deny [id]`：不传 id 时默认处理最近一条
- 配置热更新：
  - 新增 `/reload-config`，运行时重载 `config/config.ini` 的 `[ui-tui]` 参数
- 状态文件自修复：
  - `ui-tui-state.json` 解析失败时自动备份为 `ui-tui-state.json.corrupt.<timestamp>` 并重建
- 运维文档：
  - 新增 `docs/ui-tui-ops.md`，覆盖网络/超时/鉴权/审批/状态文件等排障流程

## 测试

- `go test ./...`：通过
- `./ui-tui/e2e_smoke.sh`：通过（已纳入 reload-config 与状态修复回归）

### 来源：`0228-ui-tui-final-audit-and-baseline.md`

# ui-tui 最终审计与基线标记（Phase 14）

本轮完成了 ui-tui 的最终收尾动作：文档命令面审计、真实环境回归执行、发布基线标记。

## 执行结果

- 命令面与文档审计：
  - 已核对 `ui-tui/main.go` 的 `/help` 命令列表与 `ui-tui/README.md`、`docs/ui-tui-ops.md`、`docs/frontend-tui-user.md`。
  - 对齐结果：一致；补充了兼容性风险说明。
- 真实环境回归：
  - 执行命令链：`/reload-config`、`/health`、`/status`、`/pending 3`、`/approve`、`/deny`、`/events save ...`。
  - 结果：
    - 基础链路（reload/health/status/events）通过。
    - 发现两项环境兼容风险：
      1. `/pending` 在该环境返回 500（`converting NULL to int64`）。
      2. `/approve`/`/deny` 返回 404（后端未启用 `POST /v1/ui/approval/confirm`）。
  - 已将处理建议写入 `docs/ui-tui-ops.md`。
- 发布基线：
  - 创建里程碑标签：`ui-tui-parity-v1`

## 结论

ui-tui 当前代码基线已完成能力收口；剩余问题主要是“运行环境后端版本/历史数据兼容性”而非 ui-tui 端功能缺失。

### 来源：`0230-ui-tui-doctor-command.md`

# ui-tui 增加后端能力预检命令（Phase 16）

新增 `ui-tui` 命令 `/doctor`，用于在交互期快速识别“后端接口版本不匹配/连通性异常”问题。

## 覆盖检查项

- `health`：`GET /health`
- `sessions_detail`：`GET /v1/ui/sessions/{session_id}`
- `approval_confirm`：`POST /v1/ui/approval/confirm`
  - `404` 明确提示后端版本过旧
  - `200/400` 视为接口已存在
- `config_effective`：展示当前生效配置（ws/http/重连/超时/上限）
- `ws_reachable`：WebSocket 握手检查

## 回归

- 新增 `TestRunDoctor` 覆盖 `/doctor` 主链路。
- `ui-tui/e2e_smoke.sh` 已纳入 `/doctor` 执行。

## 验证

- `go test ./...`：通过
- `./ui-tui/e2e_smoke.sh`：通过

### 来源：`0231-ui-tui-efficiency-observability-bundle.md`

# ui-tui 一次性优化补全（Phase 17）

本轮一次性完成了交互效率、可读性、恢复建议、安全审计、测试矩阵、配置治理与发布体验补全。

## 本轮补全

- 交互效率：
  - `/sessions` 支持列出后立即交互选择切换会话
  - `/pending [n]` 支持交互选择并直接 approve/deny
  - `/show` 支持消息索引交互提示
- 输出可读性：
  - 新增 `/view human|json` 视图模式切换
- 错误恢复：
  - `/approve`、`/deny`、`/events save`、`/save`、`/cancel` 失败时输出 retry suggestion
- 安全与审计：
  - 关键操作审计日志 `~/.agent-daemon/ui-tui-audit.log`
  - 覆盖 `approve`/`deny`/`cancel`/`config set`
- 可测试性：
  - 扩展 mock 后端测试矩阵（含缺失 approval endpoint 的 doctor 场景）
- 配置治理：
  - 新增 `/config tui` 显示 ui-tui 生效配置与来源（env/config）
  - `[ui-tui]` 增加 `view_mode`、`auto_doctor`
- 发布体验：
  - 新增 `ui-tui/release.sh` 单文件构建脚本
  - 新增 `/version` 输出构建元信息

## 验证

- `go test ./...`：通过
- `./ui-tui/e2e_smoke.sh`：通过

### 来源：`0233-ui-tui-adapt-ui-api-contract.md`

# ui-tui 适配冻结后的 /v1/ui/* 契约（Phase 18）

本轮将 ui-tui 客户端解析逻辑对齐到已冻结的 UI API 契约，确保后端升级后前后端行为一致。

## 适配点

- `httpJSON`：
  - 支持解析标准错误 envelope（`ok=false` + `error.code/message`）
  - 优先输出结构化错误信息
- 新增统一解包辅助：
  - `uiPayload(...)` 支持优先读取新契约字段（如 `result/snapshot/status/schema/tools/stats`）
  - 同时兼容旧契约顶层字段
- 命令面适配：
  - `/health`、`/cancel`、`/tools`、`/tool`、`/sessions`、`/stats`
  - `/gateway status|enable|disable`
  - `/config get|set`
  - `/approve`、`/deny`、`/pending` 交互路径

## 测试

- 新增 `TestHTTPJSONParsesUIErrorEnvelope`，验证客户端错误 envelope 解析。
- 全量 `go test ./...` 通过。

### 来源：`0245-ui-tui-reconnect-controls.md`

# ui-tui 重连状态可视化与人工恢复控制

本轮补齐了 ui-tui 在实时会话中的“可感知、可控、可恢复”能力。

## 主要改动

- 新增重连状态机字段（`ui-tui/main.go`）
  - `reconnectEnabled`
  - `reconnectState`（`connecting/resumed/degraded/failed`）
  - `timeoutAction`（`wait/reconnect/cancel`）
- 新增命令
  - `/reconnect status`
  - `/reconnect on|off`
  - `/reconnect now`
  - `/reconnect timeout wait|reconnect|cancel`
- `/status` 输出增强
  - 追加展示重连启用状态、当前状态、最大重连次数与超时策略
- `sendTurn` 重连逻辑增强
  - 支持超时策略分支：
    - `wait`：继续等待
    - `reconnect`：强制断开并立即重连
    - `cancel`：触发 `/v1/chat/cancel` 并结束本轮
  - 重连期间按 payload 去重，避免重复 assistant 事件
- 帮助文档更新
  - `printHelp()` 增加 `/reconnect*` 命令说明
- 回归测试
  - `ui-tui/main_test.go` 的重连用例调整为验证去重效果（重复 assistant 仅一次，result 仅一次）

## 验证

- `go test ./ui-tui -count=1`
- `make contract-check`
- `go test ./...`

### 来源：`0249-ui-tui-realtime-diagnostics-parity.md`

# ui-tui：实时诊断能力对齐（对齐 Web）

本轮将 `ui-tui` 的流式会话可观测能力对齐到 Web 诊断面板能力，补齐诊断状态展示与诊断包导出。

## 主要改动

- `ui-tui/main.go`
  - 在 `appState` 新增诊断字段：
    - `activeTransport`、`lastTurnID`、`reconnectCount`
    - `lastErrorCode`、`lastErrorText`、`fallbackHint`、`diagUpdatedAt`
  - 增加诊断能力函数：
    - `diagnosticsSnapshot()`：输出实时诊断快照
    - `exportDiagnostics(path)`：导出诊断包（包含 `diagnostics/runtime_state/recent_events`）
  - `sendTurn` 增强：
    - 每轮初始化 `turn_id`、transport、重连计数
    - 发生重连时累计 `reconnectCount` 并记录 `fallbackHint`
    - 终止事件中补采集 `error_code/error`
  - 新增命令：
    - `/diag`：查看实时诊断
    - `/diag export <file>`：导出诊断包
  - 扩展 `/reconnect status` 输出，增加 `reconnect_count/fallback_hint/last_error_code`。
- `ui-tui/main_test.go`
  - 在 `TestSendTurnReconnect` 增加诊断字段断言（重连计数、fallback 提示、turn_id）。
  - 新增 `TestExportDiagnostics`，校验导出文件结构和关键字段。
- `ui-tui/README.md`
  - 补充 `/diag` 与 `/diag export` 用法与诊断字段说明。

## 验证

- `go test ./ui-tui`
- `go test ./...`

### 来源：`0263-ui-tui-fullscreen-dashboard-mode.md`

# 263-summary-ui-tui-fullscreen-dashboard-mode

本轮对“CLI/TUI 差异项 1”继续收口，新增 `ui-tui` 全屏看板模式，并让 `agentd tui` 可直接开启该模式。

## 变更

- `ui-tui/main.go`
  - 新增启动参数解析：
    - `--fullscreen`
    - 环境变量 `AGENT_UI_TUI_FULLSCREEN=1|true`
  - 新增全屏渲染帧：
    - 清屏重绘
    - 展示 session、ws/http、状态码、传输/重连信息、最近事件与操作提示
  - 保持原命令面与输入循环不变（兼容现有脚本与操作习惯）。

- `cmd/agentd/main.go`
  - `agentd tui` 新增 `-fullscreen` 参数。
  - 在 standalone/auto 独立 `ui-tui` 路径下透传为 `AGENT_UI_TUI_FULLSCREEN=1`。

- 测试
  - `ui-tui/main_test.go` 新增 `parseStartupFlags` 测试。

- 文档
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`
  - `ui-tui/README.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./cmd/agentd -count=1`
- `go test ./...`

### 来源：`0264-ui-tui-fullscreen-timeline-and-runtime-toggle.md`

# 264-summary-ui-tui-fullscreen-timeline-and-runtime-toggle

本轮继续完善 CLI/TUI 差异项 1，在现有全屏看板基础上补齐“时间线可视化 + 运行时切换”。

## 变更

- `ui-tui/main.go`
  - 全屏模式新增 `timeline` 面板，展示最近对话轨迹（user/assistant/tool/result/error 摘要）。
  - 新增运行时命令：
    - `/fullscreen`：查看当前状态
    - `/fullscreen on|off`：运行时开关全屏看板
  - 启动首条消息、普通输入消息、发送失败场景均会记录到时间线。
  - 增加时间线容量控制（默认上限 2000，超限滚动清理）。

- `ui-tui/main_test.go`
  - 新增 `addChatLine` 截断与滚动上限测试。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

### 来源：`0274-ui-tui-command-routing-casefold-bundle.md`

# 274-summary-ui-tui-command-routing-casefold-bundle

本轮针对 `ui-tui` 命令路由做了一次性收口，目标是消除大小写与缩写输入导致的分派漂移，减少 CLI/TUI 命令面歧义。

## 变更

- `ui-tui/main.go`
  - `canonicalInput` 增强：
    - slash 命令名统一小写归一化（保留参数原样）。
    - 原有 alias（`show/sessions/tool/gw/cfg`）改为大小写不敏感。
    - 新增 slash 缩写 alias：
      - `/gw -> /gateway`
      - `/cfg -> /config`
      - `/sess -> /sessions`
      - `/wb -> /workbench`
      - `/wf -> /workflow`
      - `/bm -> /bookmark`
      - `/q -> /quit`
      - `/h -> /help`
  - `parseEventSaveArgs` 增强为大小写无关（`save/json/ndjson/since/until`）。

- `ui-tui/command_logic.go`
  - 子命令大小写收口：
    - `/reconnect status|on|off|now|timeout ...`
    - `/diag export ...`
    - `/events save ...`
    - `/pretty on|off`
    - `/fullscreen on|off`
  - 统一保持“命令词大小写无关、业务参数保持原样”的语义。

- `ui-tui/command_logic_test.go`
  - 新增/增强回归覆盖：
    - canonical alias 大小写与 slash 缩写。
    - reconnect / diag export / events save 关键路径大小写混用。
    - pretty/fullscreen 参数大小写混用。

## 验证

- `go test ./ui-tui`
- `go test ./...`

### 来源：`0275-gateway-command-dispatch-canonicalization-bundle.md`

# 275-summary-gateway-command-dispatch-canonicalization-bundle

本轮继续收口 `TODO-002`（命令语义统一），将 Gateway 侧命令入口统一到可预测的 canonical 形式，减少平台输入差异带来的分派漂移。

## 变更

- `internal/gateway/runner.go`
  - 增强 `normalizeGatewayCommand(platformName, text)`：
    - 对所有平台：若输入为 slash 命令，统一命令词小写（参数原样保留）。
    - 对所有平台：支持非 slash 英文内建命令自动 canonical（如 `approve id` -> `/approve id`）。
    - 对 Yuanbao：中文别名继续支持，且新增参数透传（如 `批准 <id>`、`拒绝 <id>`）。
  - 新增 `withTail(parts []string)` 辅助函数，统一参数拼接逻辑。

- `internal/gateway/runner_command_test.go`
  - 扩展命令归一化测试矩阵：
    - slash 大小写归一化（`/STATUS` -> `/status`）。
    - 英文非 slash 内建命令 canonical（`APPROVE ap-4` -> `/approve ap-4`）。
    - Yuanbao 中文别名参数透传（`批准 ap-1` -> `/approve ap-1`）。

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0276-gateway-builtin-command-single-source-bundle.md`

# 276-summary-gateway-builtin-command-single-source-bundle

本轮继续收口 `TODO-002`，将 Gateway 内建命令集合收敛为单一真源，避免 `runner/slack/一致性测试` 各自维护列表导致漂移。

## 变更

- 新增 `internal/gateway/commands.go`
  - `IsBuiltInGatewayCommand(name string) bool`
  - `BuiltInGatewaySlashCommands() []string`
  - 内建命令集合集中定义（`pair/unpair/cancel/queue/status/pending/approvals/grant/revoke/approve/deny/help`）。

- `internal/gateway/runner.go`
  - `normalizeGatewayCommand` 改为调用 `IsBuiltInGatewayCommand`，不再手写命令列表。

- `internal/gateway/platforms/slack.go`
  - `isBuiltInGatewaySlashCommand` 改为委托 `gateway.IsBuiltInGatewayCommand`。

- `internal/gateway/platforms/commands_consistency_test.go`
  - 跨平台一致性校验改为使用 `gateway.BuiltInGatewaySlashCommands()` 作为 shared source。

- 新增测试 `internal/gateway/commands_test.go`
  - 覆盖内建命令识别（大小写、空白、unknown）。
  - 覆盖导出的 slash 命令全集。

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0277-gateway-command-alias-resolver-bundle.md`

# 277-summary-gateway-command-alias-resolver-bundle

本轮继续收口 `TODO-002`，在 Gateway 入口层新增命令别名解析器，并将 slash/非 slash 输入统一 canonical 到同一命令语义。

## 变更

- `internal/gateway/commands.go`
  - 新增 `ResolveGatewayCommand(name string) (canonical string, ok bool)`：
    - 先识别内建命令；
    - 再解析 alias：
      - `approval -> approvals`
      - `pendings -> pending`
      - `abort/stop -> cancel`
      - `q -> queue`
      - `s -> status`
      - `h -> help`

- `internal/gateway/runner.go`
  - `normalizeGatewayCommand` 改为统一调用 `ResolveGatewayCommand`：
    - slash 命令：命令词 canonical，参数保留。
    - 非 slash 英文命令：识别后自动转为 canonical slash。

- `internal/gateway/commands_test.go`
  - 新增 alias 解析覆盖。

- `internal/gateway/runner_command_test.go`
  - 扩展 normalize 覆盖：
    - `/APPROVAL`、`/PENDINGS`、`/STOP`、`/Q`
    - `approval`、`pendings`、`abort`

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0278-gateway-unified-command-parser-bundle.md`

# 278-summary-gateway-unified-command-parser-bundle

本轮继续推进 `TODO-002`，将 Gateway 命令处理主链改为“先统一解析，再分发”，减少 `handleEvent` 内重复 `strings.Fields` 与多处二次解析造成的行为漂移。

## 变更

- `internal/gateway/runner.go`
  - 新增 `gatewayCommand` 结构与 `parseGatewayCommand(platformName, text)`：
    - 输入先走 `normalizeGatewayCommand`；
    - 输出统一的 `raw/head/args/isSlash`。
  - `handleEvent` slash 分支改为基于解析结果分发，不再在各 case 重复拆词。
  - `resolveApprovalID` 改签名为 `resolveApprovalID(args []string)`，统一从解析后的参数读取审批 ID，缺失时回退 last pending ID。

- `internal/gateway/runner_command_test.go`
  - 新增 `TestParseGatewayCommand` 覆盖：
    - 英文非 slash canonical（`APPROVE ap-1`）
    - Yuanbao 中文命令 canonical（`批准 ap-2`）
    - 普通文本保持非 slash 语义

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0279-gateway-help-from-command-catalog-bundle.md`

# 279-summary-gateway-help-from-command-catalog-bundle

本轮继续推进 `TODO-002`，将 Gateway `/help` 文本改为由命令目录自动生成，避免命令集变化后 help 文案与实际分发能力不一致。

## 变更

- `internal/gateway/commands.go`
  - 新增 `GatewayHelpText(yuanbao bool) string`，基于集中命令目录生成帮助文本。
  - 新增 `gatewayHelpCommandOrder` 与 `gatewayHelpCommandEntry`，统一命令展示顺序与参数提示（如 `/pair <code>`、`/approve <id>`）。
  - Yuanbao 模式自动附加中文快捷别名提示。

- `internal/gateway/runner.go`
  - `/help` 分支改为调用 `GatewayHelpText(...)`，移除硬编码命令串。

- `internal/gateway/commands_test.go`
  - 新增帮助文本测试：
    - 覆盖所有内建命令均出现在 help。
    - 覆盖 Yuanbao 帮助包含 quick reply aliases。

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0280-gateway-structured-command-catalog-bundle.md`

# 280-summary-gateway-structured-command-catalog-bundle

本轮继续推进 `TODO-002`，将 Gateway 命令相关元数据（内建命令、alias、help 参数模板）收口到结构化 catalog，消除多张表并行维护的漂移风险。

## 变更

- `internal/gateway/commands.go`
  - 引入 `gatewayCommandSpec` 与 `gatewayCommandCatalog`，统一声明：
    - canonical 命令名
    - 参数模板（如 `<code>`, `[ttl]`, `<id>`）
    - alias（如 `stop -> cancel`, `approval -> approvals`）
  - 通过 `init()` 从 catalog 派生：
    - `builtInGatewayCommandSet`
    - `gatewayAliasToCanonical`
    - `gatewayCommandSpecByName`
    - `gatewayHelpCommandOrder`
  - 现有 `IsBuiltInGatewayCommand` / `ResolveGatewayCommand` / `BuiltInGatewaySlashCommands` / `GatewayHelpText` 全部基于同一 catalog 数据源。

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0281-gateway-command-usage-template-catalog-bundle.md`

# 281-summary-gateway-command-usage-template-catalog-bundle

本轮继续推进 `TODO-002`，将 Gateway help 从“参数片段拼接”升级为“完整 usage 模板”，确保复杂命令（如 grant/revoke pattern）在帮助文案中稳定呈现。

## 变更

- `internal/gateway/commands.go`
  - `gatewayCommandSpec` 增加 `HelpUsage` 字段，用于声明完整帮助文案模板。
  - 为复杂命令补齐完整 usage：
    - `/grant [ttl], /grant pattern <name> [ttl]`
    - `/revoke, /revoke pattern <name>`
  - `gatewayHelpCommandEntry` 优先使用 `HelpUsage`，无模板时回退 `/<name>`。

- `internal/gateway/commands_test.go`
  - 增加断言：help 文案必须包含 grant/revoke pattern 用法，防止回归为简化版提示。

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0282-gateway-catalog-consistency-guardrail-bundle.md`

# 282-summary-gateway-catalog-consistency-guardrail-bundle

本轮继续推进 `TODO-002`，补齐 Gateway 命令目录的“防漂移护栏”：catalog 内部完整性检查 + 平台命令集精确一致性校验。

## 变更

- `internal/gateway/commands.go`
  - 新增导出接口：
    - `BuiltInGatewayCommandNames() []string`
    - `GatewayCommandAliases() map[string]string`
  - 为测试与一致性校验提供稳定数据来源。

- `internal/gateway/commands_test.go`
  - 新增 `TestGatewayCommandAliasesIntegrity`：
    - alias 不能与 canonical 同名
    - alias 不能覆盖内建命令名
    - alias 必须指向有效内建命令

- `internal/gateway/platforms/commands_consistency_test.go`
  - 将“跨平台命令一致性”从“包含关系”升级为“精确集合一致”：
    - Telegram / Discord 命令集大小与 catalog 一致
    - 不允许平台端存在 catalog 之外命令
  - 新增 `TestGatewayCatalogAndSlackBuiltinSetMatch`：
    - Slack builtin 判定集合与 catalog 命令名逐项一致

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0283-gateway-platform-command-spec-unification-bundle.md`

# 283-summary-gateway-platform-command-spec-unification-bundle

本轮继续推进 `TODO-002`，将 Telegram/Discord 的命令注册从各自硬编码改为共享 command spec，避免平台侧描述和参数定义分叉。

## 变更

- 新增 `internal/gateway/platforms/command_specs.go`
  - 定义 `gatewayPlatformCommandSpec`：
    - `Name`
    - `Description`
    - `Telegram/Discord` 开关
    - `DiscordOpts`（slash 参数）
  - 提供构建函数：
    - `telegramCommandsFromSpecs(...)`
    - `discordCommandsFromSpecs(...)`

- `internal/gateway/platforms/telegram.go`
  - `TelegramCommands()` 改为通过 `gatewayPlatformCommandSpecs()` 生成。

- `internal/gateway/platforms/discord.go`
  - `DiscordApplicationCommands()` 改为通过同一 spec 生成。

- `internal/gateway/platforms/commands_consistency_test.go`
  - 新增 `TestGatewayPlatformCommandSpecsCoverCatalog`：
    - 平台 spec 必须覆盖 gateway catalog 全部命令
    - 每条 spec 必须有描述，且同时启用 Telegram/Discord

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0284-gateway-platform-command-spec-contract-bundle.md`

# 284-summary-gateway-platform-command-spec-contract-bundle

本轮继续推进 `TODO-002`，将平台注册命令的一致性校验升级为“结构化契约测试”，不仅校验命令存在，还校验参数形态与定义一致。

## 变更

- `internal/gateway/platforms/command_specs.go`
  - 导出 `GatewayPlatformCommandSpecs()`，作为平台注册命令的权威数据源。

- 新增 `internal/gateway/platforms/command_specs_test.go`
  - `TestGatewayPlatformCommandSpecsShape`：
    - command name 唯一
    - description 非空
    - Telegram/Discord 双端启用
  - `TestGatewayPlatformCommandSpecsDiscordOptionsContract`：
    - 校验 `pair/grant/revoke/approve/deny` 的 Discord 参数结构（名称、类型、required）。

- `internal/gateway/platforms/commands_consistency_test.go`
  - 新增严格一致性校验：
    - `TelegramCommands()` 与 `telegramCommandsFromSpecs(...)` 逐项一致
    - `DiscordApplicationCommands()` 与 `discordCommandsFromSpecs(...)` 逐项一致（含 options）

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0285-gateway-discord-slash-rendering-unification-bundle.md`

# 285-summary-gateway-discord-slash-rendering-unification-bundle

本轮继续推进 `TODO-002`，将 Discord slash 渲染从硬编码命令分支改为共享命令解析器驱动，减少平台交互层的语义分叉。

## 变更

- `internal/gateway/platforms/discord.go`
  - `renderDiscordSlashCommand` 改为先调用 `gateway.ResolveGatewayCommand(...)` 获取 canonical 命令。
  - 新增通用 option 读取函数：
    - `discordOptionString(...)`
    - `discordOptionIntString(...)`
  - `pair/approve/deny/grant/revoke` 的参数拼装统一走通用提取逻辑，未知命令返回空字符串。

- `internal/gateway/platforms/discord_test.go`
  - 新增 `TestRenderDiscordSlashCommand`，覆盖：
    - `pair` 带 code
    - `approve` 带 id
    - `grant pattern + ttl`
    - `revoke pattern`
    - 简单内建命令（`status`）
    - 未知命令

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0286-gateway-slack-slash-rendering-unification-bundle.md`

# 286-summary-gateway-slack-slash-rendering-unification-bundle

本轮继续推进 `TODO-002`，将 Slack slash 渲染链路对齐到共享命令解析器，确保 Slack/Discord 在命令 canonical 规则上保持一致。

## 变更

- `internal/gateway/platforms/slack.go`
  - `renderSlackSlashCommand` 改为统一走 `gateway.ResolveGatewayCommand`：
    - 内建命令入口（`/approve` 等）自动 canonical（含大小写）
    - `/agent` 入口下文本命令支持 alias canonical（如 `abort -> /cancel`）
    - 已带 `/` 的文本命令先做 canonical 化再返回
  - 新增 `canonicalizeSlashText` 辅助函数，统一 slash 文本首词归一规则。

- `internal/gateway/platforms/slack_test.go`
  - 增强回归覆盖：
    - `/STATUS` -> `/status`
    - `/approval` -> `/approvals`
    - `abort` -> `/cancel`
    - `/APPROVE` 命令词大小写归一

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0287-gateway-approval-command-catalog-derivation-bundle.md`

# 287-summary-gateway-approval-command-catalog-derivation-bundle

本轮继续推进 `TODO-002`，将审批相关命令校验与 usage 文案统一改为由 Gateway command catalog 派生，减少 runner 与测试中的残余硬编码。

## 变更

- `internal/gateway/commands.go`
  - 新增：
    - `GatewayCommandUsage(name string) string`
    - `GatewayApprovalSlashCommands() []string`
  - 审批命令集合（`approve/deny/pending/approvals/grant/revoke/status/help`）集中由 catalog 导出。

- `internal/gateway/runner.go`
  - `/approve` `/deny` 缺参提示改为复用 `GatewayCommandUsage(...)` 输出。
  - grant/revoke 的通用 usage 错误提示改为由 catalog usage 拼接，避免文案漂移。

- `internal/gateway/platforms/commands_consistency_test.go`
  - `TestTelegramDiscordApprovalCommandsConsistency` 改为使用 `gateway.GatewayApprovalSlashCommands()`，不再手写审批命令列表。

- `internal/gateway/commands_test.go`
  - 新增 `GatewayCommandUsage` 与 `GatewayApprovalSlashCommands` 回归测试。

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0288-gateway-yuanbao-quick-reply-catalog-bundle.md`

# 288-summary-gateway-yuanbao-quick-reply-catalog-bundle

本轮继续推进 `TODO-002`，将 Yuanbao 中文快捷命令映射从 runner 硬编码迁移到 command catalog 侧统一维护。

## 变更

- `internal/gateway/commands.go`
  - 新增：
    - `ResolveYuanbaoQuickReplyCommand(head string) (slash string, ok bool)`
    - `YuanbaoQuickReplyAliasesText() string`
  - `GatewayHelpText(yuanbao=true)` 改为复用 `YuanbaoQuickReplyAliasesText()`，不再写死别名串。

- `internal/gateway/runner.go`
  - `normalizeGatewayCommand` 的 Yuanbao 分支改为调用 `ResolveYuanbaoQuickReplyCommand`，移除本地 `switch` 硬编码。

- `internal/gateway/commands_test.go`
  - 新增 quick-reply 映射覆盖：
    - 批准/同意/通过 -> `/approve`
    - 拒绝/驳回 -> `/deny`
    - 状态/待审批/审批/帮助 -> `/status|/pending|/approvals|/help`
  - 新增 quick-reply 文本函数断言。

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0289-gateway-grant-revoke-usage-catalog-bundle.md`

# 289-summary-gateway-grant-revoke-usage-catalog-bundle

本轮继续推进 `TODO-002`，将 grant/revoke 相关 usage 字符串从 runner 逻辑中剥离，统一由 command catalog 导出函数提供，消除最后一段审批管理文案硬编码。

## 变更

- `internal/gateway/commands.go`
  - 新增 usage 导出函数：
    - `GatewayGrantPatternUsage()`
    - `GatewayRevokePatternUsage()`
    - `GatewayGrantRevokeCombinedUsage()`

- `internal/gateway/runner.go`
  - `parseApprovalManageCommand` 的错误提示改为调用上述导出函数，不再内嵌 `/grant ...` `/revoke ...` 字符串。

- `internal/gateway/commands_test.go`
  - 新增 `TestGrantRevokeUsageHelpers`，锁定 usage 函数输出，防止回归硬编码或文案漂移。

## 验证

- `go test ./internal/gateway ./internal/gateway/platforms`
- `go test ./...`

### 来源：`0265-ui-tui-actions-palette.md`

# 265-summary-ui-tui-actions-palette

本轮继续推进 CLI/TUI 差异项 1，新增 `ui-tui` 快捷操作面板，提升高频运维与诊断动作的交互效率。

## 变更

- `ui-tui/main.go`
  - 新增命令：`/actions`
  - 打开后展示编号动作列表，支持选择后直接执行：
    - `/tools`
    - `/sessions 20`
    - `/show`
    - `/gateway status`
    - `/config get`
    - `/doctor`
    - `/diag`
    - `/reconnect status`
    - `/pending 5`
    - `/fullscreen on|off`（根据当前状态动态切换）
    - `/help`
  - 新增 `actionMenuItems`、`actionCommandByIndex` 便于复用与测试。

- 测试
  - `ui-tui/main_test.go` 增加动作面板索引映射测试（含 fullscreen 动态分支）。

- 文档
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

### 来源：`0266-ui-tui-fullscreen-quiet-and-timeline-command.md`

# 266-summary-ui-tui-fullscreen-quiet-and-timeline-command

本轮继续完善 CLI/TUI 差异项 1，重点提升全屏模式可读性与时间线可用性。

## 变更

- `ui-tui/main.go`
  - 全屏模式下事件输出改为“静默控制台打印 + 写入时间线”，避免刷屏破坏看板布局。
  - 新增 `/timeline [n]` 命令，在非全屏场景可查看最近对话时间线摘要。
  - 新增 `timelineSlice` 统一时间线裁剪逻辑。
  - `printEvent` 增加 `emit` 控制参数，支持“只记录不打印”。

- `ui-tui/main_test.go`
  - 新增 `timelineSlice` 行为测试。

- 文档
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

### 来源：`0267-ui-tui-fullscreen-multi-panel.md`

# 267-summary-ui-tui-fullscreen-multi-panel

本轮按“一次性完成 CLI/TUI 任务”的要求，集中补齐了全屏模式的多面板能力与统一刷新入口，减少在命令流中频繁切换上下文的成本。

## 变更

- `ui-tui/main.go`
  - 新增全屏面板状态：
    - `overview`
    - `sessions`
    - `tools`
    - `gateway`
    - `diag`
  - 新增命令：
    - `/panel`（查看当前面板）
    - `/panel <name>`（切换到指定面板）
    - `/panel next|prev`（循环切换）
    - `/refresh`（刷新当前面板数据）
  - 新增面板数据刷新逻辑：
    - sessions -> `/v1/ui/sessions`
    - tools -> `/v1/ui/tools`
    - gateway -> `/v1/ui/gateway/status`
    - diag/overview -> 本地诊断快照
  - `/actions` 快捷动作增加 `/panel next`。

- `ui-tui/main_test.go`
  - 新增面板循环逻辑测试（`nextPanel` / `prevPanel`）。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

### 来源：`0268-ui-tui-workbench-completion-bundle.md`

# 268-summary-ui-tui-workbench-completion-bundle

本轮按“CLI/TUI 一次性完整实现”目标，集中补齐 `ui-tui` 全屏工作台的核心闭环能力，避免继续碎片化迭代。

## 变更

- 全屏工作台能力增强（`ui-tui/main.go`）
  - 面板体系扩展为：
    - `overview`
    - `dashboard`（聚合 sessions/tools/gateway/diag）
    - `sessions`
    - `tools`
    - `gateway`
    - `diag`
  - 新增面板管理能力：
    - `/panel list`
    - `/panel <name>`
    - `/panel next|prev`
    - `/refresh`
  - 面板与全屏状态持久化到 `ui-tui-state.json`：
    - `fullscreen`
    - `fullscreen_panel`
  - `actions` 面板增加 `panel` 快捷动作，支持工作台内快速切换。

- 测试补齐（`ui-tui/main_test.go`）
  - `parseStartupFlags` 兼容新返回值校验。
  - 面板循环逻辑校验（含 `dashboard`）。
  - 新增 runtime state 持久化回归（fullscreen + panel）。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

### 来源：`0269-ui-tui-workbench-auto-refresh-and-persistence.md`

# 269-summary-ui-tui-workbench-auto-refresh-and-persistence

本轮继续按“CLI/TUI 一次性完整实现”收口，补齐全屏工作台的自动刷新策略与偏好持久化能力。

## 变更

- `ui-tui/main.go`
  - 全屏工作台新增面板自动刷新机制：
    - `panel_auto_refresh`（默认开启）
    - `panel_refresh_interval_seconds`（默认 8 秒）
    - 循环输入前按间隔自动刷新当前面板
  - 面板命令补齐：
    - `/panel status`
    - `/panel auto on|off`
    - `/panel interval <sec>`（1..300）
  - 状态持久化增强（`ui-tui-state.json`）：
    - `fullscreen`
    - `fullscreen_panel`
    - `panel_auto`
    - `panel_interval_seconds`
  - `dashboard` 聚合面板纳入统一刷新链路。

- `ui-tui/main_test.go`
  - 更新 action 索引断言（面板命令增强后的新序号）。
  - 新增/增强 runtime state 持久化回归：覆盖 fullscreen/panel/auto/interval。

- 文档
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

### 来源：`0270-ui-tui-workbench-drilldown-and-approvals-panel.md`

# 270-summary-ui-tui-workbench-drilldown-and-approvals-panel

本轮继续按“CLI/TUI 一次性完整实现”目标补齐工作台闭环，新增审批面板与条目钻取动作，减少命令跳转成本。

## 变更

- `ui-tui/main.go`
  - 全屏面板新增：`approvals`
  - `dashboard` 聚合面板新增审批摘要
  - 新增命令：`/open <index>`
    - 在 `sessions` 面板：切换到对应会话
    - 在 `tools` 面板：打开对应工具 schema
    - 在 `approvals` 面板：交互式 approve/deny 执行
  - `/panel` 帮助与提示同步更新（包含 approvals）

- `ui-tui/main_test.go`
  - 面板集合断言新增 `approvals`
  - 新增 panel 数据选择辅助函数测试（session/tool/approval）

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

### 来源：`0271-ui-tui-workbench-open-action-closure.md`

# 271-summary-ui-tui-workbench-open-action-closure

本轮继续按“CLI/TUI 一次性完整实现”推进，补齐工作台“面板查看 -> 条目操作”的最后一段闭环。

## 变更

- `ui-tui/main.go`
  - 全屏面板新增 `approvals`。
  - `dashboard` 聚合面板新增审批摘要数据。
  - 新增统一钻取命令：`/open <index>`
    - 在 `sessions` 面板：按索引切换到指定会话。
    - 在 `tools` 面板：按索引打开工具 schema。
    - 在 `approvals` 面板：按索引执行 approve/deny 交互动作。
  - `/panel` 帮助与面板枚举同步更新（包含 approvals）。

- `ui-tui/main_test.go`
  - 新增 panel 选择辅助函数测试（session/tool/approval）。
  - 面板集合断言包含 `approvals`。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

### 来源：`0272-ui-tui-workbench-profiles-bundle.md`

# 272-summary-ui-tui-workbench-profiles-bundle

本轮按“CLI/TUI 一次性完整实现”要求，补齐工作台配置方案能力，解决“状态靠手工重复配置”的问题。

## 变更

- `ui-tui/main.go`
  - 新增 `workbench profile` 模型与持久化文件：
    - `~/.agent-daemon/ui-tui-workbenches.json`
  - 新增命令：
    - `/workbench save <name>`
    - `/workbench list`
    - `/workbench load <name>`
    - `/workbench delete <name>`
  - profile 覆盖字段：
    - `session_id`
    - `ws/http endpoint`
    - `fullscreen/fullscreen_panel`
    - `panel_auto_refresh/panel_refresh_sec`
    - `view_mode`
  - 与工作台行为联动：
    - load 后自动落盘 runtime state 并触发当前面板 refresh
    - actions 菜单新增 `workbench list` 快捷项

- `ui-tui/main_test.go`
  - 新增 workbench profile 的 save/load/delete 回归测试。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

### 来源：`0273-ui-tui-workflow-orchestration-bundle.md`

# 273-summary-ui-tui-workflow-orchestration-bundle

本轮按“CLI/TUI 一次性完整实现”要求，补齐工作台的命令编排能力，使其从“可操作”升级为“可复用流程执行”。

## 变更

- `ui-tui/main.go`
  - 新增 workflow 持久化文件：
    - `~/.agent-daemon/ui-tui-workflows.json`
  - 新增 workflow 命令：
    - `/workflow save <name> <cmd1;cmd2;...>`
    - `/workflow list`
    - `/workflow run <name> [dry]`
    - `/workflow delete <name>`
  - 支持命令队列执行：
    - run 时将命令序列入队，交互循环自动逐条消费执行
    - `dry` 模式仅输出将执行的命令清单
  - 与工作台能力联动：
    - `actions` 增加 `workbench list` 快捷项
    - workflow 关键动作写审计日志

- `ui-tui/main_test.go`
  - 新增 workflow 解析测试（`;` 分隔、自动补 `/`）。
  - 新增 workflow save/get/delete 回归测试。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`
