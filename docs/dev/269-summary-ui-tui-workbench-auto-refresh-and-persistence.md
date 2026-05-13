# 269-summary-ui-tui-workbench-auto-refresh-and-persistence

本轮继续按“CLI/TUI 一次性完整实现”收口，补齐全屏工作台的自动刷新策略与偏好持久化能力。

## 变更

- `ui-tui/main.go`
  - 全屏工作台新增面板自动刷新机制：
    - `panel_auto_refresh`（默认开启）
    - `panel_refresh_interval_seconds`（默认 8 秒）
    - 循环输入前按间隔自动刷新当前面板
  - 面板命令补齐：
    - `/panel status`
    - `/panel auto on|off`
    - `/panel interval <sec>`（1..300）
  - 状态持久化增强（`ui-tui-state.json`）：
    - `fullscreen`
    - `fullscreen_panel`
    - `panel_auto`
    - `panel_interval_seconds`
  - `dashboard` 聚合面板纳入统一刷新链路。

- `ui-tui/main_test.go`
  - 更新 action 索引断言（面板命令增强后的新序号）。
  - 新增/增强 runtime state 持久化回归：覆盖 fullscreen/panel/auto/interval。

- 文档
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

