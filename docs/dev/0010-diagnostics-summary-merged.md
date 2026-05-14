# 0010 diagnostics summary merged

## 模块

- `diagnostics`

## 类型

- `summary`

## 合并来源

- `0011-diag-summary-merged.md`
- `0012-diagnostics-summary-merged.md`

## 合并内容

### 来源：`0011-diag-summary-merged.md`

# 0011 diag summary merged

## 模块

- `diag`

## 类型

- `summary`

## 合并来源

- `0251-diag-v1-ci-artifacts-and-smoke.md`

## 合并内容

### 来源：`0251-diag-v1-ci-artifacts-and-smoke.md`

# diag.v1 接入 CI 产物链路（Web + ui-tui）

本轮将统一诊断包 `diag.v1` 纳入 CI 烟测与 artifact 产出，形成“样本生成 -> 校验回放 -> 报告上传”闭环。

## 主要改动

- `ui-tui/e2e_smoke.sh`
  - 新增 `/diag` 与 `/diag export <path>` 烟测路径
  - 内置调用 `go run ./scripts/diag_bundle -file ...` 校验导出样本
  - 支持 `ARTIFACTS_DIR`，输出 `diag-ui-tui.sample.json`
- `web/e2e_smoke.sh`（新增）
  - 执行 web test/build
  - 生成 `diag-web.sample.json`
  - 调用 `diag_bundle` 做校验
- `web/scripts/gen_diag_sample.mjs`（新增）
  - 生成 CI 用 `diag.v1` Web 样本
- `.github/workflows/contract-guard.yml`
  - 增加 Node 环境与 `npm --prefix web ci`
  - 增加 `ui-tui` 与 `web` smoke 步骤产出诊断样本
  - 增加 `diag_bundle` replay 报告生成
  - 上传 diagnostics sample/replay artifact
- 文档更新
  - `ui-tui/README.md` 与 `web/README.md` 增加 smoke + 诊断样本说明

## 验证

- `ARTIFACTS_DIR=/tmp/diag-artifacts ./ui-tui/e2e_smoke.sh`
- `ARTIFACTS_DIR=/tmp/diag-artifacts ./web/e2e_smoke.sh`
- `go run ./scripts/diag_bundle -file /tmp/diag-artifacts/diag-ui-tui.sample.json`
- `go run ./scripts/diag_bundle -file /tmp/diag-artifacts/diag-web.sample.json`
- `go test ./scripts/diag_bundle ./ui-tui`
- `make contract-check`

### 来源：`0012-diagnostics-summary-merged.md`

# 0012 diagnostics summary merged

## 模块

- `diagnostics`

## 类型

- `summary`

## 合并来源

- `0250-diagnostics-bundle-schema-and-replay.md`

## 合并内容

### 来源：`0250-diagnostics-bundle-schema-and-replay.md`

# 诊断闭环自动化：统一 diag.v1 契约 + validate/replay

本轮完成 Web 与 ui-tui 诊断包契约统一，并补齐本地自动校验与回放能力。

## 主要改动

- 统一诊断包 schema（`diag.v1`）
  - 新增 `docs/api/diagnostics.bundle.schema.json`
  - 统一关键字段：`source/schema_version/transport/reconnect/timeout/error/events`
- Web 导出对齐
  - 新增 `web/src/lib/diagnostics.ts`（`buildDiagnosticsBundle`）
  - `web/src/App.tsx` 的导出改为 `diag.v1` 统一结构，事件字段统一为 `events`
  - `web/src/lib/api.test.ts` 新增 bundle 结构测试
- ui-tui 导出对齐
  - `ui-tui/main.go` 的 `/diag` 与 `/diag export` 改为同一 `diag.v1` 输出结构
  - `ui-tui/main_test.go` 调整断言，校验 `schema_version/source/reconnect_count`
- 诊断 validate/replay 工具
  - 新增 `scripts/diag_bundle/main.go`
    - `-file` 读取诊断包
    - 校验核心字段类型与枚举
    - 回放事件序列并识别终止事件（`result/error/cancelled`）
    - 可选 `-report` 输出 JSON 报告
  - 新增 `scripts/diag_bundle/main_test.go`
- 质量门禁
  - `Makefile` 新增 `diag-check`
  - `contract-check` 纳入 `diag-check`
- 文档更新
  - `docs/api/contract-versioning.md` 增加诊断包契约说明与工具入口
  - `ui-tui/README.md` 增加 diag.v1 对齐说明

## 验证

- `npm --prefix web run test`
- `npm --prefix web run build`
- `go test ./scripts/diag_bundle`
- `go test ./ui-tui`
- `make contract-check`
- `go test ./...`
