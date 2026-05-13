# 261-summary-cli-tui-standalone-auto-mode

本轮针对“CLI/TUI 差异项 1”做了可执行收口：`agentd tui` 不再固定走轻量内置循环，而是默认优先使用独立 `ui-tui`（完整命令面），并保留兼容回退。

## 变更

- `cmd/agentd/main.go`
  - `agentd tui` 新增参数：`-mode auto|standalone|lite`
  - 默认 `-mode auto`：
    - 优先启动独立 `ui-tui` 可执行文件
    - 若不可用，自动回退内置 lite TUI（原 `internal/cli/tui.go` 路径）
  - `-mode standalone`：强制独立 `ui-tui`，不可用即报错
  - `-mode lite`：强制内置 lite TUI
  - 新增 `resolveUITUIBinary()`：
    - 优先 `AGENT_UI_TUI_BIN`
    - 其次 `PATH` 中 `ui-tui`
    - 再次当前仓库 `ui-tui/tui.run`

- 测试
  - 新增 `cmd/agentd/main_tui_test.go`
    - 覆盖 `AGENT_UI_TUI_BIN` 分支
    - 覆盖本地 `ui-tui/tui.run` 候选路径分支

- 文档
  - `docs/frontend-tui-user.md`：增加 `agentd tui -mode` 使用说明
  - `docs/frontend-tui-dev.md`：增加运行模式与回退策略说明

## 验证

- `go test ./cmd/agentd -count=1`
- `go test ./...`

