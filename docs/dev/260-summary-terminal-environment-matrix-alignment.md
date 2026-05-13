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

