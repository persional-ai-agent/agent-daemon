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
