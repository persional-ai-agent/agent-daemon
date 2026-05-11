# 072 总结：read_file 增加 max_chars 防护（Hermes 风格）

## 变更

`read_file` 新增：

- `max_chars`（默认 100000，上限 200000）
- `reject_on_truncate`（默认 true；超限时返回 error 而非截断内容）
- `truncated` 标记（仅在 `reject_on_truncate=false` 的兼容模式下返回截断内容时使用）

用于防止一次性读取超大文件导致上下文爆炸，并与 Hermes 的 read-size guard 思路对齐。
