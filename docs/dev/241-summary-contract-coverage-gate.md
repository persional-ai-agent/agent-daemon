# 契约覆盖率度量与缺口门禁

本轮新增“OpenAPI × replay”覆盖率度量，并将核心端点覆盖率提升为 CI 强制门禁。

## 主要改动

- replay 用例扩展为核心全覆盖：
  - 更新 `internal/api/testdata/replay/cases.json`
  - 补齐 `/v1/ui/*`、`POST /v1/chat`、`POST /v1/chat/cancel`
- 回放执行器增强：
  - `internal/api/contract_replay_test.go` 支持 `contract_path`（模板路径）
  - 为管理接口注入完整测试依赖（config/gateway/approval）
- 补齐新增快照：
  - `internal/api/testdata/contracts/ui_*_success*.json` 多个文件
- 新增覆盖率工具：
  - `scripts/contract_coverage/main.go`
  - `scripts/contract_coverage/main_test.go`
  - 产出 `artifacts/contract-coverage.json`
  - `-enforce-core=true` 时，核心端点未覆盖直接失败
- Makefile / CI 接入：
  - `make contract-coverage`
  - `make contract-check` 纳入 coverage gate
  - `.github/workflows/contract-guard.yml` 新增 coverage 步骤与 artifact 上传
- 文档规则更新：
  - `docs/api/contract-versioning.md` 增加覆盖率门禁与“新增接口必须补 replay case”要求

## 验证

- `make contract-check`
- `go test ./...`
