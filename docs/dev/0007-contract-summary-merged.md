# 0007 contract summary merged

## 模块

- `contract`

## 类型

- `summary`

## 合并来源

- `0008-contract-summary-merged.md`

## 合并内容

### 来源：`0008-contract-summary-merged.md`

# 0008 contract summary merged

## 模块

- `contract`

## 类型

- `summary`

## 合并来源

- `0236-contract-consumer-regression.md`
- `0237-contract-governance-automation.md`
- `0238-contract-toolchain-and-breaking-gate.md`
- `0239-contract-diff-field-level-and-report.md`
- `0240-contract-replay-tests.md`
- `0241-contract-coverage-gate.md`

## 合并内容

### 来源：`0236-contract-consumer-regression.md`

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

### 来源：`0237-contract-governance-automation.md`

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

### 来源：`0238-contract-toolchain-and-breaking-gate.md`

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

### 来源：`0239-contract-diff-field-level-and-report.md`

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

### 来源：`0240-contract-replay-tests.md`

# 契约回放测试（Replay）与 CI 报告

本轮新增运行时契约回放，覆盖“真实请求 -> 实际响应”的回归验证，避免仅靠静态文档和字段 diff 造成漏检。

## 主要改动

- 新增回放用例清单：`internal/api/testdata/replay/cases.json`
  - 覆盖 `/v1/ui/tools`、`/v1/chat`、`/v1/chat/cancel`
- 新增回放测试：`internal/api/contract_replay_test.go`
  - 读取回放请求样例并逐条执行 HTTP 调用
  - 校验 OpenAPI（路径/方法/响应码桶/required 顶层字段）
  - 校验既有快照（与 `internal/api/testdata/contracts/*.json` 对齐）
  - 输出回放报告 JSON
- Makefile 接入
  - 新增 `contract-replay`
  - `contract-check` 增加 replay 执行
- CI 接入
  - `contract-guard` 新增 Contract Replay 步骤
  - 上传 artifact：`contract-replay-report`（`artifacts/contract-replay.json`）
- 文档更新
  - `docs/api/contract-versioning.md` 增加 replay 报告与校验项说明

## 验证

- `make contract-check`
- `go test ./...`

### 来源：`0241-contract-coverage-gate.md`

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
