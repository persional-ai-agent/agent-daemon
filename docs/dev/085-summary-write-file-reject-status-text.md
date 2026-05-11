# 085 Summary - write_file 拒绝写入 read_file 内部状态文本

## 背景

Hermes 的 `write_file` 会拒绝将 `read_file` 的内部状态提示文本当作文件内容写入磁盘。

该类错误在模型行为中很常见：模型在 `read_file` 返回 dedup stub 后，误把提示信息当作“需要写入的内容”，导致文件被破坏。

## 变更

- `write_file` 在写入前增加校验：
  - 如果 `content` 等于 `read_file` 的 dedup 提示文本，或在短包装（长度不超过 2x 提示文本）下包含该提示文本，则拒绝写入并返回错误。

实现位置：

- `internal/tools/builtin.go`：新增 `isInternalReadFileStatusText`，并在 `write_file` 中调用。

## 验证

- `internal/tools/builtin_test.go`：新增用例覆盖拒绝写入 dedup 状态文本。

