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
