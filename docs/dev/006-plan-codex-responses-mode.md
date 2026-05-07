# 006 计划：Codex Responses 模式补齐

## 目标

新增 `provider=codex` 模式，支持 Responses API 下的 tool-calling 闭环。

## 实施步骤

1. 新增 Codex client
验证：可调用 `/responses` 并解析 assistant 文本

2. 增加工具调用映射
验证：可解析 `function_call` 并生成 `core.ToolCall`

3. 增加工具结果映射
验证：tool 消息可映射到 `function_call_output` 输入项

4. 接入配置切换
验证：`AGENT_MODEL_PROVIDER=codex` 时可正常创建 Codex client

5. 增加测试并回归
验证：模型层新增测试通过，`go test ./...` 通过
