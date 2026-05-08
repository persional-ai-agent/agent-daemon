# 019 计划：Provider 增量事件透传（最小版）

## 目标

打通“模型流式增量 -> Agent 事件流 -> SSE”的最小链路。

## 实施步骤

1. 在 `internal/model` 新增可选事件扩展接口与通用调用 helper。
2. 在 OpenAI/Anthropic/Codex 流式解析中上报增量事件。
3. 在 `FallbackClient` 中透传事件，保证主备切换不丢事件。
4. 在 `Engine.callWithRetry` 中消费模型事件并发出 `model_stream_event`。
5. 增加 `agent` 层测试，验证事件透传。
6. 全量回归 `go test ./...`。

## 验证标准

- 启用流式时可看到 `model_stream_event`
- fallback 场景下事件链路不中断
- 不启用流式时行为不回退
