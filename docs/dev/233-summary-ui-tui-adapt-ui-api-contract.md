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
