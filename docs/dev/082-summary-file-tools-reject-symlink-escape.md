# 082 Summary - 文件工具拒绝 symlink 逃逸 workdir

## 背景

仅依赖“路径字符串是否在 workdir 内”无法阻止通过符号链接（symlink）实现的路径逃逸，例如：

- 在 workdir 内创建 `out -> /tmp/outside` 的目录 symlink；
- 再调用 `write_file path=out/a.txt`，实际会写入 workdir 外部路径。

Hermes 的文件工具通常会对 symlink/realpath 做更严格的约束，避免该类逃逸。

## 变更

- 增加 `rejectSymlinkEscape(workdir, resolvedPath)`：
  - 从 workdir 开始逐级 `lstat` 检查已存在的路径组件；
  - 发现任一组件为 symlink 即拒绝，防止读/写/补丁操作穿透到 workdir 外。
- `read_file` / `patch` / `write_file` 在执行前接入该校验：
  - `write_file` 在 `MkdirAll` 前校验，避免通过目录 symlink 在 workdir 外创建文件。

实现位置：

- `internal/tools/safety.go`：新增 `rejectSymlinkEscape`
- `internal/tools/builtin.go`：接入 `read_file` / `patch` / `write_file`

## 验证

- `internal/tools/read_file_guardrails_test.go`：新增用例，`read_file` 读取指向 workdir 外的 symlink 文件应拒绝。
- `internal/tools/builtin_test.go`：新增用例，`write_file` 写入 symlink 目录路径应拒绝，且不会写到 workdir 外。

## 边界与后续

- 当前策略是“路径组件中出现 symlink 即拒绝”，更严格但更安全；如需要支持 workdir 内部 symlink（且仍不逃逸），可后续演进为 realpath 归一化 + 根前缀校验策略。

