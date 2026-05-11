# 056 总结：CLI 工具查看最小对齐

## 变更摘要

新增 `agentd tools list|show|schemas`，保留 `agentd tools` 原有列名行为，补齐 Hermes 工具管理入口中的最小工具查看能力。

## 新增能力

```bash
agentd tools
agentd tools list
agentd tools show terminal
agentd tools schemas
```

## 修改文件

| 文件 | 变更 |
|------|------|
| `cmd/agentd/main.go` | 新增 `runTools`、schema 查找与 JSON 输出 |
| `cmd/agentd/main_test.go` | 增加 `findToolSchema` 测试 |
| `README.md` | 增加工具查看示例 |
| `docs/overview-product.md` | 更新 CLI 管理面能力说明 |
| `docs/overview-product-dev.md` | 更新 CLI 工具查看设计说明 |
| `docs/dev/README.md` | 增加 056 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 手动验证：`tools list` 输出工具名，`tools show terminal` 输出单个 JSON schema，`tools schemas` 输出完整 schema 列表
