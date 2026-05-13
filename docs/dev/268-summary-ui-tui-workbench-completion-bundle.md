# 268-summary-ui-tui-workbench-completion-bundle

本轮按“CLI/TUI 一次性完整实现”目标，集中补齐 `ui-tui` 全屏工作台的核心闭环能力，避免继续碎片化迭代。

## 变更

- 全屏工作台能力增强（`ui-tui/main.go`）
  - 面板体系扩展为：
    - `overview`
    - `dashboard`（聚合 sessions/tools/gateway/diag）
    - `sessions`
    - `tools`
    - `gateway`
    - `diag`
  - 新增面板管理能力：
    - `/panel list`
    - `/panel <name>`
    - `/panel next|prev`
    - `/refresh`
  - 面板与全屏状态持久化到 `ui-tui-state.json`：
    - `fullscreen`
    - `fullscreen_panel`
  - `actions` 面板增加 `panel` 快捷动作，支持工作台内快速切换。

- 测试补齐（`ui-tui/main_test.go`）
  - `parseStartupFlags` 兼容新返回值校验。
  - 面板循环逻辑校验（含 `dashboard`）。
  - 新增 runtime state 持久化回归（fullscreen + panel）。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

