# UI/Chat 契约版本与兼容策略

适用范围：
- `/v1/ui/*`
- `/v1/chat`
- `/v1/chat/cancel`
- `/v1/chat/ws`（事件内含 `api_version` / `compat`）

当前冻结值：
- `api_version = v1`
- `compat = 2026-05-13`

## 变更规则（冻结）

1. 向后兼容新增（允许）  
   - 只新增字段，不删除、不重命名、不改变现有字段语义。
   - 新字段默认可选；消费者不得依赖字段顺序。

2. 兼容修复（允许）  
   - 修正文案、补充错误 message、补充缺失 header。
   - 不改变 `error.code` 既有语义。

3. 破坏性变更（禁止直接改 v1）  
   - 删除/重命名字段、修改字段类型、改变状态码语义，必须升级 `api_version`（例如 `v2`）。
   - 升级后必须保留旧版本或提供明确迁移窗口。

## `compat` 更新策略

- `compat` 表示“向后兼容承诺生效日期”。
- 仅当新增兼容能力、且不会破坏旧客户端时，才可推进日期。
- 推进 `compat` 时必须同步更新：
  - `internal/api/server.go` 常量
  - `docs/api/ui-chat-contract.openapi.yaml`
  - 契约快照测试 fixture

## 错误码策略

- 错误 envelope：`ok=false` + `error.code` + `error.message`。
- `error.code` 应稳定、可机读；`message` 可面向人类。
- 新增错误码允许；复用旧错误码时语义必须一致。

## 质量门禁

- 必须通过：
  - `go test ./internal/api ./internal/cli ./ui-tui`
  - `TestContractSnapshot*` 契约快照测试
  - `TestContractSpecVersionSync` 文档版本同步测试
