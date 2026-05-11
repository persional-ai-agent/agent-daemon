# 059 总结：工具禁用配置最小对齐

## 变更摘要

新增工具禁用配置，补齐 Hermes 工具管理中的最小启停能力。

## 新增能力

```bash
agentd tools disable terminal
agentd tools disabled
agentd tools enable terminal
```

配置来源：

- `AGENT_DISABLED_TOOLS=terminal,web_fetch`
- `[tools] disabled = terminal,web_fetch`

禁用工具会从 registry 中移除，因此不会出现在 `tools list` / `tools schemas` 中，dispatch 时也会成为 unknown tool。

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/config/config.go` | 新增 `DisabledTools` 配置 |
| `internal/config/config_test.go` | 增加 disabled tools 配置读取测试 |
| `internal/tools/registry.go` | 新增 `Disable` |
| `cmd/agentd/main.go` | 增加 `tools disabled|disable|enable` 与运行时过滤 |
| `cmd/agentd/main_test.go` | 增加列表解析和 registry 过滤测试 |
| `README.md` | 增加工具启停示例 |
| `docs/overview-product.md` | 更新 CLI 工具管理说明 |
| `docs/overview-product-dev.md` | 更新工具禁用设计说明 |
| `docs/dev/README.md` | 增加 059 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/config ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 手动验证：临时配置文件执行 `tools disable -file ... terminal` 后，`tools disabled -file ...` 输出 `terminal`
- 手动验证：使用 `AGENT_CONFIG_FILE=... tools list` 时，`terminal` 从工具列表消失，`read_file` 仍保留
- 手动验证：`tools enable -file ... terminal` 后禁用列表清空
