# 六项差异收口（基础版）一次性补齐

本轮按“能力先可用、再逐步增强”的方式，一次性补齐了 6 类差异的基础能力闭环。

## 1) 工具能力可观测（doctor）

- `cmd/agentd/main.go`
  - `agentd doctor` 新增 `tool_capabilities` 检查项：
    - `browser_cdp`（是否启用）
    - `openai_media`（vision/image/tts 实后端可用性）
    - `fal_image`（image_generate 兼容后端可用性）

## 2) 执行环境抽象（local + docker）

- `internal/tools/process.go`
  - 新增 `ForegroundBackendOptions`
  - 新增 `RunForegroundWithOptions(...)`
  - 新增 `buildForegroundCommand(...)`，支持：
    - `backend=local`（原行为）
    - `backend=docker`（`docker run --rm -i ... sh -lc`）
- `internal/tools/builtin.go`
  - `terminal` 新增参数：
    - `backend`：`local|docker`
    - `docker_image`：镜像名（默认 `alpine:3.20`）
  - `background=true` 时限制仅支持 `backend=local`（避免语义不一致）
  - 返回增加 `backend` 字段
- 测试：
  - `internal/tools/process_backend_test.go`
  - `internal/tools/builtin_test.go` 增加非 local 后台执行拒绝测试

## 3) 插件骨架（最小清单发现）

- 新增 `internal/plugins/loader.go`
  - 支持从：
    - `<workdir>/.agent-daemon/plugins`
    - `<workdir>/plugins`
  - 加载 JSON manifest（`name/type/version/entry/enabled`）
- 新增 CLI：
  - `agentd plugins list`
- 测试：
  - `internal/plugins/loader_test.go`

## 4) Session 检索可读摘要

- `internal/store/session_store.go`
  - `Search(...)` 结果新增 `summary` 字段（180 rune 截断摘要）
  - 保留原 `content`，向后兼容

## 5) Gateway 命令一致性约束

- 新增测试 `internal/gateway/platforms/commands_consistency_test.go`
  - 校验 Telegram/Discord 两端关键审批命令集一致：
    - `/approve` `/deny` `/pending` `/approvals` `/grant` `/revoke` `/status` `/help`

## 6) 文档与验收闭环

- 本文档记录六项基础收口结果。
- 通过全量测试与契约门禁，确保不回退。

## 验证

- `go test ./internal/tools ./internal/plugins ./internal/store ./internal/gateway/platforms ./cmd/agentd`
- `make contract-check`
- `go test ./...`
