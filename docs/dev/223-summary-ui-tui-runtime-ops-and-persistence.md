# ui-tui 运行控制与持久化能力补齐（Phase 10）

本阶段在纯 Go 的 `ui-tui` 上继续补齐运行控制、可追溯性和会话快捷管理能力，目标是让终端端产品面在不依赖前端页面的前提下，覆盖常见操作闭环。

## 新增能力

- 运行控制：
  - `/health`：查看后端健康状态
  - `/cancel`：取消当前会话中的运行任务
- 可追溯能力：
  - `/history [n]`：查看本地命令历史
  - `/rerun <index>`：按历史序号重放命令
  - `/events [n]`：查看最近运行事件
  - `/events save <file>`：导出运行事件日志
- 会话快捷管理：
  - `/bookmark add <name> [sid]`
  - `/bookmark list`
  - `/bookmark use <name>`
- 交互反馈增强：
  - 提示符显示最近命令状态（`tui[ok]` / `tui[err]`）
  - `/status` 输出最近一次命令状态与摘要

## 持久化行为

- 历史命令写入 `~/.agent-daemon/ui-tui-history.log`
- 书签写入 `~/.agent-daemon/ui-tui-bookmarks.json`
- 事件日志默认驻留内存，可按需落盘

## 验证

- 已执行：`go test ./...`
- 结果：通过
