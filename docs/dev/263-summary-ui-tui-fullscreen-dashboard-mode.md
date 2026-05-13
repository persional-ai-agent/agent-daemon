# 263-summary-ui-tui-fullscreen-dashboard-mode

本轮对“CLI/TUI 差异项 1”继续收口，新增 `ui-tui` 全屏看板模式，并让 `agentd tui` 可直接开启该模式。

## 变更

- `ui-tui/main.go`
  - 新增启动参数解析：
    - `--fullscreen`
    - 环境变量 `AGENT_UI_TUI_FULLSCREEN=1|true`
  - 新增全屏渲染帧：
    - 清屏重绘
    - 展示 session、ws/http、状态码、传输/重连信息、最近事件与操作提示
  - 保持原命令面与输入循环不变（兼容现有脚本与操作习惯）。

- `cmd/agentd/main.go`
  - `agentd tui` 新增 `-fullscreen` 参数。
  - 在 standalone/auto 独立 `ui-tui` 路径下透传为 `AGENT_UI_TUI_FULLSCREEN=1`。

- 测试
  - `ui-tui/main_test.go` 新增 `parseStartupFlags` 测试。

- 文档
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`
  - `ui-tui/README.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./cmd/agentd -count=1`
- `go test ./...`

