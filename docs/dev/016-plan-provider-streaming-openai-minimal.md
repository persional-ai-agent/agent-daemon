# 016 计划：Provider 流式统一（OpenAI 最小落地）

## 目标

在不改动 Agent Loop 协议的前提下，为 OpenAI 客户端补齐流式聚合能力。

## 实施步骤

1. 在 `OpenAIClient` 增加 `UseStreaming` 开关。
2. 在 `ChatCompletion` 中按开关选择：
   - 非流式：保持原逻辑
   - 流式：`stream=true` 调用并解析 SSE
3. 实现流式聚合：
   - 文本 `delta.content` 追加
   - `delta.tool_calls` 按 index 聚合函数名与参数
4. 新增配置项：`AGENT_MODEL_USE_STREAMING`
5. 在启动装配中把配置传入 OpenAI 客户端。
6. 补测试并全量回归。

## 验证标准

- OpenAI 流式文本与工具调用可正确聚合
- 默认行为不变（不开开关仍走非流式）
- `go test ./...` 通过
