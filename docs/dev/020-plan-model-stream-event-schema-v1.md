# 020 计划：`model_stream_event` 标准字典（v1）

## 目标

在不破坏现有 `model_stream_event` 事件外层结构的前提下，统一 `event_data` 的最小字段集。

## 实施步骤

1. 在 `internal/model` 增加事件标准化函数。
2. 在模型事件入口统一应用标准化（而非各 provider 各自分散处理）。
3. 增加测试，覆盖别名字段到标准字段的映射。
4. 增加 `agent` 层事件测试，验证透传后的标准字段可用。
5. 全量回归 `go test ./...`。

## 验证标准

- `text_delta` 始终包含 `event_data.text`
- `tool_arguments_delta` 始终包含 `event_data.tool_name` 与 `event_data.arguments_delta`
- 全量测试通过
