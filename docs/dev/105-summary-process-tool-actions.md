# 105 Summary - process 工具补齐 poll/log/wait/kill/write 动作（Hermes 体验对齐）

## 背景

Hermes 的 `process` 工具支持对后台进程进行更丰富的管理（poll/log/wait/kill/write 等）。Go 版此前仅支持 `list/status/stop`，导致模型在长任务场景下无法按 Hermes 习惯读取增量日志或等待完成。

## 变更

- `process` 新增动作：
  - `poll`：返回自上次 poll 以来的新输出（按 session 记忆 offset）
  - `log`：按 byte offset 分页读取输出文件
  - `wait`：阻塞等待进程结束或超时，并返回一次最终 poll
  - `kill`：`stop` 别名
  - `write`：返回 not supported（Go 版未跟踪 stdin/pty）
- `process` schema 更新，暴露 `offset/max_chars/timeout_seconds`

## 修改文件

- `internal/tools/builtin.go`

