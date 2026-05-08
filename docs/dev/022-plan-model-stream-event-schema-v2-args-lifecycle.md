# 022 计划：`model_stream_event` v2 参数生命周期补齐

## 目标

将工具参数流式事件从“仅增量”扩展为“开始-增量-结束”三段式，并统一 `message_done.finish_reason`。

## 实施步骤

1. 扩展 `normalizeStreamEvent`：
   - 别名映射 `tool_arguments_* -> tool_args_*`
   - 标准化 `tool_args_start/delta/done` 字段
   - `message_done` 默认补 `finish_reason=stop`
2. OpenAI/Anthropic/Codex 流式路径补发 `tool_args_start/delta/done`。
3. `message_done` 统一补 `finish_reason`（有工具调用时 `tool_calls`，否则 `stop`）。
4. 更新模型层与 agent 层测试。
5. 全量回归 `go test ./...`。

## 验证标准

- 三 provider 在流式场景均包含参数生命周期事件
- `message_done.finish_reason` 始终可用
- 全量测试通过
