# 契约工具链与 Breaking 门禁

本轮把契约治理从“有测试”推进到“可发布、可比较、可门禁”。

## 交付项

- Make 入口统一
  - `Makefile` 新增：
    - `contract-test`
    - `contract-diff`
    - `contract-check`
    - `contract-release`
- 契约 diff 工具
  - 新增 `scripts/contract_diff.go`
  - 对比 base/target OpenAPI，输出 breaking/non-breaking 报告
  - breaking 检测规则（首版）：
    - operation 删除
    - response code 删除
    - request 新增 required 字段
    - 200 响应 required 字段删除
  - breaking 时要求声明文件 `-ack`
- 契约版本基线目录
  - 新增：
    - `docs/api/versions/v1/ui-chat-contract.openapi.yaml`
    - `docs/api/versions/v1/contract-versioning.md`
- CI Breaking 门禁
  - 更新 `.github/workflows/contract-guard.yml`
  - PR：对比 base 分支 OpenAPI
  - push：对比 v1 基线
  - breaking 变更需 `.contract/breaking-change-ack.md`
- 规范文档
  - 新增 `.contract/README.md`
  - 更新 `docs/api/contract-versioning.md`（增加工具链入口与声明规则）

## 验证

- `make contract-check` 通过
- `go test ./...` 通过
