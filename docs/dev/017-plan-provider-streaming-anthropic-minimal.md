# 017 计划：Provider 流式统一（Anthropic 最小落地）

## 目标

在不改动 Agent Loop 与 `model.Client` 接口的前提下，补齐 Anthropic 流式聚合能力。

## 实施步骤

1. 在 `AnthropicClient` 增加 `UseStreaming` 开关。
2. 在 `ChatCompletion` 中按开关切换到流式分支。
3. 在流式分支中：
   - `stream=true` 发起请求
   - 解析 `text/event-stream` 的 `data:` 事件
   - 聚合文本和 tool_use 参数增量
4. 在启动装配中复用 `AGENT_MODEL_USE_STREAMING` 开关。
5. 新增 Anthropic 流式文本与工具调用测试。
6. 执行 `go test ./...` 回归。

## 验证标准

- Anthropic 流式文本可正确聚合
- Anthropic 流式 tool_use 参数可正确拼接
- 全量测试通过
