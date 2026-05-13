# 契约 Diff 升级为字段级检测并输出报告

本轮将契约门禁从粗粒度升级到字段级，补齐可审计报告产物。

## 主要改动

- `scripts/contract_diff.go`
  - 新增字段级检测：
    - request/response 字段 `type` 变化（breaking）
    - enum 收缩（breaking）
    - path/query 参数 `required` 变化（可选->必填为 breaking）
    - 参数 `type` 与 enum 变化
  - 新增 `allOf` 与 `$ref` 解析，避免 OpenAPI 组合 schema 漏检
  - 新增 `-report` 输出 JSON 报告（默认 `artifacts/contract-diff.json`）
- `scripts/contract_diff_test.go`
  - 新增覆盖用例：类型变化、enum 收缩、参数 required 变化、allOf/$ref 解析
- `Makefile`
  - `contract-diff` / `contract-check` 默认生成 `artifacts/contract-diff.json`
- `.github/workflows/contract-guard.yml`
  - PR 与 push 都生成 `artifacts/contract-diff.json`
  - 上传 artifact：`contract-diff-report`
- `docs/api/contract-versioning.md`
  - 补充检测范围与报告产物说明

## 验证

- `go test ./scripts ./internal/api ./internal/cli ./ui-tui`
- `make contract-check`
- `go test ./...`
