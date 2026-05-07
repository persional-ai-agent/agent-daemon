# 005 计划：Provider 多模式补齐

## 目标

在当前 OpenAI 模式基础上，新增 Anthropic 模式并实现运行时可配置切换。

## 实施步骤

1. 新增 Anthropic client
验证：可调用 `/messages` 并解析文本与 `tool_use`

2. 实现协议转换
验证：`core.Message` 可稳定映射到 Anthropic 请求格式并反解回来

3. 增加 provider 选择配置
验证：`AGENT_MODEL_PROVIDER=anthropic` 时启动 Anthropic client

4. 增加测试并回归
验证：新增 `internal/model` 单元测试通过，`go test ./...` 通过
