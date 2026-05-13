# 271-summary-ui-tui-workbench-open-action-closure

本轮继续按“CLI/TUI 一次性完整实现”推进，补齐工作台“面板查看 -> 条目操作”的最后一段闭环。

## 变更

- `ui-tui/main.go`
  - 全屏面板新增 `approvals`。
  - `dashboard` 聚合面板新增审批摘要数据。
  - 新增统一钻取命令：`/open <index>`
    - 在 `sessions` 面板：按索引切换到指定会话。
    - 在 `tools` 面板：按索引打开工具 schema。
    - 在 `approvals` 面板：按索引执行 approve/deny 交互动作。
  - `/panel` 帮助与面板枚举同步更新（包含 approvals）。

- `ui-tui/main_test.go`
  - 新增 panel 选择辅助函数测试（session/tool/approval）。
  - 面板集合断言包含 `approvals`。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

