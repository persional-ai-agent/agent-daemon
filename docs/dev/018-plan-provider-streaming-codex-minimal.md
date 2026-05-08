# 018 计划：Provider 流式统一（Codex 最小落地）

## 目标

在保持现有 `model.Client` 接口不变的前提下，补齐 Codex 流式聚合能力。

## 实施步骤

1. 在 `CodexClient` 增加 `UseStreaming` 开关。
2. `ChatCompletion` 按开关切换流式分支。
3. 流式分支中：
   - `stream=true` 发起请求
   - 解析 SSE `data:` 事件
   - 聚合文本与函数调用参数增量
4. 在启动装配中传递 `AGENT_MODEL_USE_STREAMING` 到 Codex 客户端。
5. 新增 Codex 流式文本/工具调用测试。
6. 执行 `go test ./...` 回归。

## 验证标准

- Codex 流式文本可正确聚合
- Codex 流式函数调用参数可正确拼接
- 全量测试通过
