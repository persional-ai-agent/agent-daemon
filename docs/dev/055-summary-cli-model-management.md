# 055 总结：CLI 模型管理最小对齐

## 变更摘要

新增 `agentd model show|providers|set`，补齐 Hermes `hermes model` 的最小本地模型切换体验。

## 新增能力

```bash
agentd model show
agentd model providers
agentd model set openai gpt-4o-mini
agentd model set anthropic:claude-3-5-haiku-latest
agentd model set -base-url https://api.openai.com/v1 codex gpt-5-codex
```

`model set` 默认写入 `config/config.ini`，也支持 `AGENT_CONFIG_FILE` 和 `-file`。环境变量仍优先于配置文件。

## 修改文件

| 文件 | 变更 |
|------|------|
| `cmd/agentd/main.go` | 新增 `model show|providers|set` 子命令与解析/写入 helper |
| `cmd/agentd/main_test.go` | 增加模型参数解析与 provider 专属配置键写入测试 |
| `internal/config/config.go` | 新增 `LoadFile`，支持 `model show -file` 读取指定配置 |
| `internal/config/config_test.go` | 增加 `LoadFile` 覆盖 |
| `README.md` | 增加模型切换示例 |
| `docs/overview-product.md` | 更新 CLI 管理面能力说明 |
| `docs/overview-product-dev.md` | 更新 CLI 配置/模型管理设计说明 |
| `docs/dev/README.md` | 增加 055 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/config ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 临时配置文件手动验证：`model set -file ... -base-url ... anthropic:claude-test`、`model show -file ...`、`config list -file ...` 均通过
