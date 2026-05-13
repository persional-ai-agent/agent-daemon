# CLI 语义对齐 UI 契约（Phase 19）

本轮将 `internal/cli`（`agentd chat/tui`）的命令输出语义与 `/v1/ui/*` 契约对齐，统一为结构化 `ok/error.code/error.message` 模式。

## 变更

- `internal/cli/chat.go` 新增统一输出助手 `printCLIEnvelope`：
  - 成功：`ok=true` + 业务字段
  - 失败：`ok=false` + `error.code` + `error.message`
  - 同步携带 `api_version` 与 `compat`
- 覆盖命令：
  - `/session`、`/tools`、`/sessions`、`/stats`、`/show`
  - `/reload`、`/clear`
  - 参数错误、能力不支持、未知命令等失败分支

## 结果

- API / ui-tui / CLI 三端错误语义一致，便于日志采集与自动化处理。

## 验证

- `go test ./internal/cli ./internal/api ./ui-tui`：通过
