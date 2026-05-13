# 契约治理自动化（快照测试/CI/OpenAPI/版本策略）

本轮将接口契约对齐从“人工约定”升级为“自动化防回退”。

## 交付项

- 契约快照测试（`internal/api`）
  - 新增 `internal/api/contract_snapshot_test.go`
  - 固化 4 组快照 fixture（`internal/api/testdata/contracts/*.json`）：
    - `ui_tools_success`
    - `ui_tools_method_not_allowed`
    - `chat_success`
    - `chat_cancel_success`
- 契约文档一致性测试
  - 新增 `internal/api/contract_spec_test.go`
  - 校验 OpenAPI 文档中的 `api_version` / `compat` 与服务端常量保持同步
- CI 契约门禁
  - 新增 `.github/workflows/contract-guard.yml`
  - PR / main push 自动执行：`go test ./internal/api ./internal/cli ./ui-tui`
- 机器可读契约文档
  - 新增 `docs/api/ui-chat-contract.openapi.yaml`
  - 覆盖 `/v1/ui/*`、`/v1/chat`、`/v1/chat/cancel` 的核心 envelope 与兼容字段
- 版本兼容策略文档
  - 新增 `docs/api/contract-versioning.md`
  - 明确 `api_version` / `compat` 升级规则、错误码策略与门禁要求

## 结果

- 契约升级路径具备“文档 -> 测试 -> CI”闭环。
- 未来修改接口时，若破坏已冻结字段或文档不同步，将在本地测试或 CI 直接失败。
