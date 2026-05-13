# Chat API 契约对齐（/v1/chat*）

本轮将非 UI 的 chat 系列接口也统一到结构化错误语义，并保持向后兼容字段。

## 覆盖范围

- `POST /v1/chat`
- `POST /v1/chat/stream`（握手前错误）
- `GET/POST /v1/chat/cancel`
- `GET/WS /v1/chat/ws`（WS 错误事件）

## 统一规则

- 错误响应统一为：
  - `ok: false`
  - `error.code`
  - `error.message`
  - `api_version`
  - `compat`
- Header 统一携带：
  - `X-Agent-UI-API-Version`
  - `X-Agent-UI-API-Compat`

## 向后兼容

- `/v1/chat` 成功响应新增 `result` 包装的同时，保留历史顶层字段：
  - `session_id/final_response/messages/turns_used/finished_naturally/summary`
- `/v1/chat/cancel` 保留历史顶层字段：
  - `session_id`
  - `cancelled`

## 测试

- 新增 `internal/api/chat_contract_test.go`
- 增强 `internal/api/ui_contract_test.go` 通用错误 envelope 断言
