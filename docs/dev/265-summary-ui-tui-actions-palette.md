# 265-summary-ui-tui-actions-palette

本轮继续推进 CLI/TUI 差异项 1，新增 `ui-tui` 快捷操作面板，提升高频运维与诊断动作的交互效率。

## 变更

- `ui-tui/main.go`
  - 新增命令：`/actions`
  - 打开后展示编号动作列表，支持选择后直接执行：
    - `/tools`
    - `/sessions 20`
    - `/show`
    - `/gateway status`
    - `/config get`
    - `/doctor`
    - `/diag`
    - `/reconnect status`
    - `/pending 5`
    - `/fullscreen on|off`（根据当前状态动态切换）
    - `/help`
  - 新增 `actionMenuItems`、`actionCommandByIndex` 便于复用与测试。

- 测试
  - `ui-tui/main_test.go` 增加动作面板索引映射测试（含 fullscreen 动态分支）。

- 文档
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

