# 097 Summary - 额外文件系统工具（mkdir/list_dir/delete_file/move_file）

## 背景

Hermes 提供更丰富的 file operations（delete/move 等）。Go 版 `agent-daemon` 之前仅有 `read_file/write_file/patch/search_files`。

## 变更

新增可选文件系统管理工具（全部限制在 workdir 内，并拒绝 symlink 组件与非普通文件）：

- `mkdir path`
- `list_dir path?`
- `delete_file path recursive?`
- `move_file src dst`

实现位置：

- `internal/tools/fs_admin.go`
- `internal/tools/builtin.go`：注册 + schema
- `internal/tools/toolsets.go`：新增 toolset `fs_admin`

