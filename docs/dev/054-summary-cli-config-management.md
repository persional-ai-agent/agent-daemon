# 054 总结：CLI 配置管理最小对齐

## 变更摘要

新增 `agentd config list|get|set`，补齐 Hermes 风格配置管理面的最小可用入口。

## 新增能力

```bash
agentd config list
agentd config get api.model
agentd config set api.model gpt-4o-mini
agentd config set provider.fallback anthropic
```

默认读写 `config/config.ini`，也支持：

- `AGENT_CONFIG_FILE=/path/to/config.ini`
- 子命令 `-file /path/to/config.ini`

配置优先级保持不变：环境变量 > 配置文件 > 内置默认值。

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/config/config.go` | `Load()` 增加 `AGENT_CONFIG_FILE` 查找 |
| `internal/config/manage.go` | 新增 INI 配置读写、列表与密钥脱敏函数 |
| `internal/config/config_test.go` | 增加配置管理与 `AGENT_CONFIG_FILE` 测试 |
| `cmd/agentd/main.go` | 新增 `config list|get|set` 子命令 |
| `README.md` | 增加配置管理示例 |
| `docs/overview-product.md` | 增加 CLI 配置管理能力说明 |
| `docs/overview-product-dev.md` | 增加配置管理模块与设计说明 |
| `docs/dev/README.md` | 增加 054 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/config`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过（需要沙箱外运行；默认沙箱禁止 `httptest` 监听本地端口）
- 临时配置文件手动验证：`config set/get/list -file /tmp/.../config.ini` 通过
