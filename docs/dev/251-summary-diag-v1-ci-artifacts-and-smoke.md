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
