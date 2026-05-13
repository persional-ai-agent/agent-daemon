# 273-summary-ui-tui-workflow-orchestration-bundle

本轮按“CLI/TUI 一次性完整实现”要求，补齐工作台的命令编排能力，使其从“可操作”升级为“可复用流程执行”。

## 变更

- `ui-tui/main.go`
  - 新增 workflow 持久化文件：
    - `~/.agent-daemon/ui-tui-workflows.json`
  - 新增 workflow 命令：
    - `/workflow save <name> <cmd1;cmd2;...>`
    - `/workflow list`
    - `/workflow run <name> [dry]`
    - `/workflow delete <name>`
  - 支持命令队列执行：
    - run 时将命令序列入队，交互循环自动逐条消费执行
    - `dry` 模式仅输出将执行的命令清单
  - 与工作台能力联动：
    - `actions` 增加 `workbench list` 快捷项
    - workflow 关键动作写审计日志

- `ui-tui/main_test.go`
  - 新增 workflow 解析测试（`;` 分隔、自动补 `/`）。
  - 新增 workflow save/get/delete 回归测试。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

