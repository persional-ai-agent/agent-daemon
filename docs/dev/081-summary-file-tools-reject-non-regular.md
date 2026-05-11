# 081 Summary - 文件工具拒绝非普通文件（FIFO/Socket/Device）

## 背景

Hermes 的文件相关工具对“非普通文件”（如命名管道/FIFO、socket、设备文件等）通常会做防护，以避免：

- 读取 FIFO 导致阻塞（等待写端）；
- 误读/误写设备文件带来安全风险；
- 工具行为不可预期（例如读 socket、读特殊设备）。

Go 版 `agent-daemon` 之前仅做了 “路径必须在 workdir 内” 的约束，但未显式拒绝特殊文件类型。

## 变更

- `read_file`：在打开文件前拒绝非普通文件（FIFO/socket/device/目录）。
- `patch`：在读取与写入前拒绝非普通文件（避免 `os.ReadFile` 在 FIFO 上阻塞）。
- `write_file`：如果目标路径已存在，则拒绝非普通文件（不存在时正常创建）。

实现位置：

- `internal/tools/safety.go`：新增 `rejectNonRegularFile(path)`。
- `internal/tools/builtin.go`：在上述工具中接入校验逻辑。

## 验证

- 新增用例：`internal/tools/read_file_guardrails_test.go` 覆盖 FIFO 场景，确保 `read_file` 立即返回错误而不是阻塞。

## 边界与后续

- 本次仅对“非普通文件类型”做拒绝；未处理“符号链接指向 workdir 外”的额外隔离（如 Hermes 进一步要求，可继续对齐）。

