# 076 总结：process 工具补齐 list 动作（对齐 Hermes 体验）

## 变更

`process` 工具新增 `action=list`，用于列出当前进程内跟踪的后台任务（terminal background）。

返回字段包含：

- `session_id`、`command`、`started_at`、`status`、`exit_code`、`output_file`、`error`

## 边界

只覆盖 agent-daemon 自己启动并跟踪的后台进程，不做系统级进程枚举。

