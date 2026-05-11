# 055 Plan：CLI 模型管理最小实现

## 任务

1. 增加 `agentd model show`。
   - 验证：根据 `config.Config` 输出当前 provider/model/base URL。
2. 增加 `agentd model providers`。
   - 验证：输出 `openai`、`anthropic`、`codex`。
3. 增加 `agentd model set`。
   - 验证：支持 `provider model` 与 `provider:model` 两种输入；写入正确 INI 键位。
4. 更新 README 与总览文档。
   - 验证：文档包含示例与边界说明。

## 边界

- 不做模型目录拉取。
- 不新增 provider。
- 不改变环境变量优先级。
- 不改变 `buildModelClient`。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 临时配置文件手动验证 `model set/show`。
