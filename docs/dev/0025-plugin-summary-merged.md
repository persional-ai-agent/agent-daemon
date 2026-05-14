# 0025 plugin summary merged

## 模块

- `plugin`

## 类型

- `summary`

## 合并来源

- `0033-plugin-summary-merged.md`

## 合并内容

### 来源：`0033-plugin-summary-merged.md`

# 0033 plugin summary merged

## 模块

- `plugin`

## 类型

- `summary`

## 合并来源

- `0254-plugin-runtime-and-cli-management.md`
- `0276-plugin-ecosystem-closure.md`

## 合并内容

### 来源：`0254-plugin-runtime-and-cli-management.md`

# 254 总结：插件运行时与 CLI 管理闭环补齐

本次针对与 Hermes 的“插件生态”差异，完成 Go 版最小可用闭环：

- 插件契约：`internal/plugins` 新增 manifest 校验（`type=tool`、`tool.command`、`tool.schema` 必填）。
- 插件运行时：支持将 JSON manifest 注册为运行时工具（schema + command 执行）。
- 插件调用协议：插件进程通过 stdin 接收 JSON 参数，stdout 支持返回 JSON 结果或文本结果。
- 插件控制面：新增 `plugins.disabled` 配置，并在运行时过滤禁用插件。
- CLI 管理：`agentd plugins list/show/validate/enable/disable`。
- 健康检查：`agentd doctor` 新增 `plugins` 检查项（发现数量与启用数量）。

## 主要实现

- `internal/plugins/loader.go`
  - 扩展 manifest 字段（`description/tool/env`）
  - 增加 `ValidateManifest`、`IsEnabled`
- `internal/plugins/tool_plugin.go`
  - 新增 runtime tool plugin，执行外部命令并注册到 `tools.Registry`
- `cmd/agentd/main.go`
  - engine 构建阶段接入插件发现与工具注册
  - 新增 plugins CLI 子命令与禁用配置读写
  - `doctor` 增加插件检查项

## 测试

- `internal/plugins/loader_test.go`：manifest 校验与发现
- `internal/plugins/tool_plugin_test.go`：插件工具注册与调用
- `cmd/agentd/main_test.go`：禁用过滤与 doctor 插件检查

验证通过：

- `go test ./internal/plugins ./cmd/agentd -count=1`
- `go test ./...`

### 来源：`0276-plugin-ecosystem-closure.md`

# 276 总结：插件生态闭环补齐

## 背景

用户要求一次性完成 Hermes 差异清单中的第 7 项：插件 marketplace、签名/校验、沙箱、dashboard slot 和内置插件实能力闭环。目标是在不重写 Hermes 全量插件生态的前提下，让 Go 版具备可安装、可发现、可校验、可运行、可展示且默认隔离的插件闭环。

## 实现结果

- `internal/plugins` 支持 JSON 与 YAML manifest，并支持插件目录中的 `plugin.yaml` / `manifest.yaml` / `plugin.json` / `manifest.json`。
- manifest 支持 Hermes 风格元数据字段：`kind`、`author`、`provides_tools`、`hooks`、`platforms`、`pip_dependencies`。
- manifest 支持多能力声明：`tools`、`providers`、`commands`、`dashboard` / `dashboards`；运行时会把多能力 plugin 展开成 tool/provider 注册项。
- 新增本地插件安装/卸载能力，目录插件会按插件名安装到 `.agent-daemon/plugins`，卸载时限制在配置的插件目录内。
- 新增 marketplace index 支持，`plugins marketplace list/install -file index.json` 可从本地索引按名称安装插件，并支持 source sha256 校验。
- 新增 security manifest：支持 Ed25519 manifest 签名校验，以及插件目录内文件 sha256 校验；`plugins verify` 输出每个 manifest 的校验结果。
- 新增默认进程沙箱：插件 tool/provider/command 默认不继承完整宿主环境，命令默认限制在插件目录内执行；manifest 可显式配置 env passthrough、workdir 和逃逸开关。
- 新增 plugin command 执行能力：`agentd plugins exec <command> [args...]`，命令通过 stdin 接收 JSON payload，stdout 可返回 JSON 或纯文本。
- CLI 管理面新增：`plugins commands`、`plugins dashboards`、`plugins verify`、`plugins install`、`plugins uninstall`、`plugins exec`、`plugins marketplace`。
- API 管理面新增 `/v1/ui/plugins/dashboards`，为 Web/TUI 提供 dashboard slot 元数据。

## 边界

- 当前 marketplace 支持本地索引和本地 source，不做远程下载。
- 当前 dashboard slot 提供 API 元数据，前端真实挂载仍是后续 Web/TUI 工作。
- 当前未做依赖自动安装、版本兼容协商和 OS 级强隔离。

## 验证

```bash
GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./internal/plugins ./internal/api ./cmd/agentd
GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./...
```

结果：通过。
