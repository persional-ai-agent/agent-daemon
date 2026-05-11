# 079 总结：session_search 支持排除/包含会话参数（对齐 Hermes 体验）

## 变更

`session_search` 新增可选参数：

- `exclude_session_id`：排除指定会话
- `include_current_session`：是否包含当前会话（默认 false，保持原行为：排除当前会话）

返回中会回显实际使用的 `exclude_session_id`，便于调试。

