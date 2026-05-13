# 256 总结：终端多环境补齐（SSH 后端）

本次针对“多执行环境矩阵”差异，补齐了 `terminal` 的 `ssh` 后端，使当前执行后端从 `local/docker` 扩展到 `local/docker/ssh`。

## 主要改动

- `internal/tools/process.go`
  - `ForegroundBackendOptions` 新增 SSH 选项：
    - `SSHHost` / `SSHUser` / `SSHPort`
    - `SSHKeyPath`
    - `SSHStrictHostKey`
    - `SSHKnownHostsFile`
  - `buildForegroundCommand` 新增 `backend=ssh` 分支，构造 `ssh ... sh -lc <cmd>` 调用。
- `internal/tools/builtin.go`
  - `terminal` 支持读取 SSH 参数并传入执行层。
  - `terminal` 返回中新增 `ssh_host/ssh_user/ssh_port`（当 backend=ssh）。
  - `terminal` schema 新增 SSH 参数字段，并将 backend 枚举扩展为 `local/docker/ssh`。

## 测试

- `internal/tools/process_backend_test.go`
  - 新增 SSH 命令构造测试。
  - 保留 unsupported backend 负例。
- `internal/tools/builtin_test.go`
  - 新增 `backend=ssh` 且缺少 `ssh_host` 的失败行为测试。

验证：

- `go test ./internal/tools -count=1`
- `go test ./...`
