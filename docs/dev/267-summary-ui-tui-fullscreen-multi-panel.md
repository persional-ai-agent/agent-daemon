# 267-summary-ui-tui-fullscreen-multi-panel

本轮按“一次性完成 CLI/TUI 任务”的要求，集中补齐了全屏模式的多面板能力与统一刷新入口，减少在命令流中频繁切换上下文的成本。

## 变更

- `ui-tui/main.go`
  - 新增全屏面板状态：
    - `overview`
    - `sessions`
    - `tools`
    - `gateway`
    - `diag`
  - 新增命令：
    - `/panel`（查看当前面板）
    - `/panel <name>`（切换到指定面板）
    - `/panel next|prev`（循环切换）
    - `/refresh`（刷新当前面板数据）
  - 新增面板数据刷新逻辑：
    - sessions -> `/v1/ui/sessions`
    - tools -> `/v1/ui/tools`
    - gateway -> `/v1/ui/gateway/status`
    - diag/overview -> 本地诊断快照
  - `/actions` 快捷动作增加 `/panel next`。

- `ui-tui/main_test.go`
  - 新增面板循环逻辑测试（`nextPanel` / `prevPanel`）。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

