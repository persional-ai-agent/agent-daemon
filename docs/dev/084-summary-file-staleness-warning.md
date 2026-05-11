# 084 Summary - write_file/patch 增加“文件已变更”告警（staleness warning）

## 背景

Hermes 的文件工具会在 `read_file` 后记录文件 mtime，并在后续 `write_file`/`patch` 时检查该文件是否在“上次读取之后被外部修改”，以提示模型：

- 你基于的内容可能已经过期；
- 需要考虑先重新读取确认再写入，避免覆盖他人修改或并发 agent 的写入。

Go 版 `agent-daemon` 之前缺少该类告警。

## 变更

- `read_file`：在同一 `session_id` 内记录该文件的 mtime（`ModTime().UnixNano()`）。
- `write_file` / `patch`：
  - 如果目标文件存在且记录过 read mtime，并且当前 mtime 与记录不一致，则在返回中加入 `_warning` 字段（不阻塞写入）。
  - 写入成功后刷新记录的 mtime，避免同一会话的连续写入误报。

实现位置：

- `internal/tools/builtin.go`：新增 `readStamp`（上限 1000，超限清空），并在 `read_file`/`write_file`/`patch` 接入。

## 验证

- `internal/tools/builtin_test.go`：新增 `write_file` staleness warning 用例。
- `internal/tools/patch_test.go`：新增 `patch` staleness warning 用例。

