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
