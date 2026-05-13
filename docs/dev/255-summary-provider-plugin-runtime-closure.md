# 255 总结：Provider 插件运行时闭环补齐

本次完成“Provider 插件生态”最小完整闭环：

- 插件契约新增 `type=provider`：
  - `provider.command` 必填
  - 可配置 `provider.args`、`provider.timeout_seconds`、`provider.model`
- 运行时接入：
  - `buildProviderClient` 在非内置 provider 时，从插件清单加载 provider 插件并构造模型客户端
  - 支持插件返回两种响应形态：
    - `{"message": {...}}`
    - OpenAI 兼容 `{"choices":[{"message": {...}}]}`
- 配置与 CLI：
  - `model providers` 现在输出“内置 provider + provider 插件”
  - `model set` / `setup` / `setup wizard` 支持插件 provider 名称
  - `doctor` 对插件 provider 的凭证检查改为“由插件运行时自行管理”

## 测试补齐

- `internal/plugins/provider_client_test.go`
- `internal/plugins/loader_test.go`（provider manifest 校验）
- `cmd/agentd/main_test.go`（provider 插件发现与加载调用）

验证：

- `go test ./internal/plugins ./cmd/agentd -count=1`
- `go test ./...`
