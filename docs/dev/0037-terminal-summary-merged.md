# 0037 terminal summary merged

## 模块

- `terminal`

## 类型

- `summary`

## 合并来源

- `0047-terminal-summary-merged.md`

## 合并内容

### 来源：`0047-terminal-summary-merged.md`

# 0047 terminal summary merged

## 模块

- `terminal`

## 类型

- `summary`

## 合并来源

- `0105-terminal-notify-on-complete.md`
- `0256-terminal-ssh-backend-alignment.md`
- `0260-terminal-environment-matrix-alignment.md`

## 合并内容

### 来源：`0105-terminal-notify-on-complete.md`

# 106 Summary - terminal(background) 补齐 notify_on_complete（Hermes 体验对齐）

## 变更

- `terminal` 工具新增参数 `notify_on_complete`：
  - 当 `background=true` 且 `notify_on_complete=true` 时，进程结束会 best-effort 发出一个工具事件 `process_complete`（通过现有 ToolEventSink 通道）。

## 修改文件

- `internal/tools/builtin.go`

### 来源：`0256-terminal-ssh-backend-alignment.md`

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

### 来源：`0260-terminal-environment-matrix-alignment.md`

# 260-summary-terminal-environment-matrix-alignment

本次完成“执行环境矩阵”第 1 项补齐：将 `terminal` 执行后端从 `local/docker/ssh` 扩展为 `local/docker/podman/singularity/ssh/daytona/vercel/modal`，并同步完成 schema、实现分支、测试与文档对齐。

## 变更内容

- 执行实现（`internal/tools/process.go`）
  - 新增后端分支：
    - `podman`
    - `singularity`
    - `daytona`
    - `vercel`
    - `modal`
  - 新增对应参数字段透传结构：
    - `podman_image`
    - `singularity_image`
    - `daytona_workspace`
    - `vercel_sandbox_id`
    - `modal_ref`

- tool 入参与 schema（`internal/tools/builtin.go`）
  - `terminal` 参数解析与 `ForegroundBackendOptions` 透传补齐以上字段。
  - `terminalParams().backend.enum` 扩展为：
    - `local/docker/podman/singularity/ssh/daytona/vercel/modal`
  - 新增 schema 字段说明：
    - `podman_image`
    - `singularity_image`
    - `daytona_workspace`
    - `vercel_sandbox_id`
    - `modal_ref`
  - 返回元数据按后端补齐，便于调用侧做诊断。

- 测试补齐
  - `internal/tools/process_backend_test.go`
    - 新增 `podman` 命令构造测试。
    - 新增 `singularity/daytona/vercel/modal` 缺参失败测试。
  - `internal/tools/builtin_test.go`
    - 新增 `terminal` 在扩展后端缺失必填参数时返回失败的行为测试。
  - `internal/tools/schema_alignment_test.go`
    - 新增 `terminal.backend` 枚举与实现分支一致性断言。

- 文档更新
  - `docs/overview-product.md`
  - `docs/overview-product-dev.md`
  - 对齐状态更新为“终端环境核心能力已对齐”，并明确 `background=true` 仍仅支持 `local`。

## 验证

- `go test ./internal/tools -count=1`
- `go test ./...`
