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
