# 058 总结：CLI 网关管理最小对齐

## 变更摘要

新增 `agentd gateway status|platforms|enable|disable`，补齐 Hermes `gateway` 管理入口的最小配置能力。

## 新增能力

```bash
agentd gateway status
agentd gateway status -json
agentd gateway platforms
agentd gateway enable
agentd gateway disable
```

`enable/disable` 只写入 `gateway.enabled`；平台 token 继续通过 `agentd config set` 管理。

当前 `gateway platforms` 输出包含：Telegram、Discord、Slack、Yuanbao（Yuanbao 凭证来自 `YUANBAO_TOKEN` 或 `YUANBAO_APP_ID/YUANBAO_APP_SECRET`）。

## 修改文件

| 文件 | 变更 |
|------|------|
| `cmd/agentd/main.go` | 新增 gateway 子命令与状态 helper |
| `cmd/agentd/main_test.go` | 增加 gateway 状态测试 |
| `README.md` | 增加网关配置示例 |
| `docs/overview-product.md` | 更新 CLI 管理面能力说明 |
| `docs/overview-product-dev.md` | 更新 Gateway CLI 设计说明 |
| `docs/dev/README.md` | 增加 058 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 手动验证：临时配置文件执行 `gateway enable`、`gateway status`、`gateway platforms`、`gateway disable`、`gateway status -json` 均通过
- JSON 输出在没有已配置平台时保持 `configured_platforms: []`
