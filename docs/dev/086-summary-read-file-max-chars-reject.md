# 086 Summary - read_file 超过 max_chars 默认拒绝（Hermes 行为）

## 背景

Hermes 的 `read_file` 对单次读取返回的字符数有安全上限；当返回内容超过 `max_chars` 时，会直接返回错误并提示使用 `offset/limit` 进行更精确的分页读取，以避免上下文被大文件占满。

Go 版 `agent-daemon` 之前在超过 `max_chars` 时会返回截断内容（`truncated=true`），与 Hermes 默认行为存在差异。

## 变更

- `read_file` 新增参数 `reject_on_truncate`（默认 `true`）：
  - 当读取将超过 `max_chars` 时：
    - `reject_on_truncate=true`：返回 `success=false` + `error`，并附带 `file_size/total_lines` 等元信息（Hermes 风格）。
    - `reject_on_truncate=false`：保持旧行为，返回截断内容 + `truncated=true`（兼容模式）。

实现位置：

- `internal/tools/builtin.go`：`read_file` 增加 oversize 拒绝逻辑与参数 schema。

## 验证

- `internal/tools/read_file_guardrails_test.go`：
  - 覆盖默认拒绝
  - 覆盖兼容模式下允许截断

