# 契约消费者回归补齐（UI/Chat/CLI）

本轮在既有 `/v1/ui/*` 与 `/v1/chat*` 契约冻结基础上，补齐消费者侧与兼容字段回归，避免后续迭代出现“接口已对齐、调用方回退失效”。

## 目标

- 强化 `/v1/chat` 与 `/v1/chat/cancel` 的成功契约测试：
  - 标准字段：`ok/api_version/compat/result`
  - 兼容字段：`session_id/final_response/cancelled` 等顶层字段
- 强化 `ui-tui` 对新旧响应结构的读取回归：
  - 优先读取 `result.*`
  - 兼容读取 legacy 顶层字段
- 强化 `internal/cli` 结构化输出稳定性：
  - 成功与失败都输出统一 envelope（`ok/error.code/error.message`）

## 主要变更

- `internal/api/chat_contract_test.go`
  - 新增 `/v1/chat` 成功 envelope + 兼容字段断言
  - 新增 `/v1/chat/cancel` 成功 envelope + 兼容字段断言
  - 补充响应 Header 断言（`X-Agent-UI-API-Version`、`X-Agent-UI-API-Compat`）
- `internal/api/ui_contract_test.go`
  - 抽取通用断言工具：`assertUIContractHeaders`、`decodeJSONMap`
  - 统一 UI 契约测试与 Chat 契约测试复用
- `ui-tui/main_test.go`
  - 新增 `uiPayload` 优先取 `result` 与 fallback legacy 字段回归用例
- `internal/cli/chat_test.go`
  - 新增 `printCLIEnvelope` 成功/失败输出回归用例（含 `api_version/compat/error`）

## 验证

- `go test ./...`
- 预期：全量通过；新增测试覆盖契约字段、兼容字段和消费者解析路径。
