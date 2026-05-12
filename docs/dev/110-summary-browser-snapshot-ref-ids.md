# 110 Summary - browser_snapshot/ref 对齐增强（轻量实现）

## 变更

- `browser_snapshot`：输出增加 best-effort ref IDs（`@e1/@e2/...`），并返回 `pending_dialogs: []` 字段（Hermes 风格）。
- `browser_click`：新增 `ref` 参数，可直接点击 snapshot 中的 ref（链接）。
- `browser_type`：新增 `ref` 参数，可对 snapshot 中的 input ref 写入（用于 GET 表单提交的 best-effort）。
- `browser_snapshot` schema 增加 `full` 参数（兼容 Hermes；轻量实现中忽略）。

## 修改文件

- `internal/tools/browser_light.go`
- `internal/tools/builtin.go`

