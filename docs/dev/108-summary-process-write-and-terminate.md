# 108 Summary - process write/terminate 行为补齐（Hermes 体验对齐）

## 变更

- `process(action="write")`：支持向后台进程 stdin 写入内容（best-effort；非 PTY）。
- `process(action="stop")`：从硬 kill 调整为优先 `SIGTERM`，超时后 `SIGKILL`（保留 `kill` 显式硬 kill）。

说明：

- `stop_process`（独立工具）仍保持历史行为（硬 kill），避免破坏既有调用习惯；推荐新代码使用 `process(action="stop")`。

## 修改文件

- `internal/tools/process.go`
- `internal/tools/builtin.go`

