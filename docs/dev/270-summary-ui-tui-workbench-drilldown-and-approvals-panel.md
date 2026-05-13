# 270-summary-ui-tui-workbench-drilldown-and-approvals-panel

本轮继续按“CLI/TUI 一次性完整实现”目标补齐工作台闭环，新增审批面板与条目钻取动作，减少命令跳转成本。

## 变更

- `ui-tui/main.go`
  - 全屏面板新增：`approvals`
  - `dashboard` 聚合面板新增审批摘要
  - 新增命令：`/open <index>`
    - 在 `sessions` 面板：切换到对应会话
    - 在 `tools` 面板：打开对应工具 schema
    - 在 `approvals` 面板：交互式 approve/deny 执行
  - `/panel` 帮助与提示同步更新（包含 approvals）

- `ui-tui/main_test.go`
  - 面板集合断言新增 `approvals`
  - 新增 panel 数据选择辅助函数测试（session/tool/approval）

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

