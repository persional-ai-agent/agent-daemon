# 272-summary-ui-tui-workbench-profiles-bundle

本轮按“CLI/TUI 一次性完整实现”要求，补齐工作台配置方案能力，解决“状态靠手工重复配置”的问题。

## 变更

- `ui-tui/main.go`
  - 新增 `workbench profile` 模型与持久化文件：
    - `~/.agent-daemon/ui-tui-workbenches.json`
  - 新增命令：
    - `/workbench save <name>`
    - `/workbench list`
    - `/workbench load <name>`
    - `/workbench delete <name>`
  - profile 覆盖字段：
    - `session_id`
    - `ws/http endpoint`
    - `fullscreen/fullscreen_panel`
    - `panel_auto_refresh/panel_refresh_sec`
    - `view_mode`
  - 与工作台行为联动：
    - load 后自动落盘 runtime state 并触发当前面板 refresh
    - actions 菜单新增 `workbench list` 快捷项

- `ui-tui/main_test.go`
  - 新增 workbench profile 的 save/load/delete 回归测试。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

