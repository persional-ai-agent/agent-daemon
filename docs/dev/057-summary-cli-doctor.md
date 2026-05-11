# 057 总结：CLI 本地诊断最小对齐

## 变更摘要

新增 `agentd doctor`，补齐 Hermes `doctor` 的最小本地诊断能力。

## 新增能力

```bash
agentd doctor
agentd doctor -json
```

检查项：

- 配置文件路径与环境变量优先级提示
- `workdir`
- `data_dir`
- provider/model 配置
- provider API key 是否为空
- MCP transport
- Gateway token 基础配置
- 内置工具注册数量

## 修改文件

| 文件 | 变更 |
|------|------|
| `cmd/agentd/main.go` | 新增 `doctor` 子命令与检查 helper |
| `cmd/agentd/main_test.go` | 增加 doctor 分支测试 |
| `README.md` | 增加诊断命令示例 |
| `docs/overview-product.md` | 更新 CLI 管理面能力说明并修正内置工具数量 |
| `docs/overview-product-dev.md` | 更新 CLI doctor 设计说明 |
| `docs/dev/README.md` | 增加 057 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 手动验证：`AGENT_DAEMON_HOME=/tmp/agentd-doctor-home go run ./cmd/agentd doctor -json` 输出 `ok/warn/error` JSON
- 手动验证：默认 `~/.agent-daemon` 在当前沙箱不可写时，`doctor` 返回 `data_dir` error 并以非零状态退出
