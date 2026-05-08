# 024 总结：`model_stream_event` v2 结束原因与 ID 别名归一

## 已完成

- `message_done.finish_reason` 新增常见枚举归一：
  - `end_turn -> stop`
  - `tool_use -> tool_calls`
  - `max_tokens/max_output_tokens -> length`
- `tool_call_*` 与 `tool_args_*` 事件新增 `tool_use_id -> tool_call_id` 兼容映射。
- 模型层与 agent 层测试已补齐：
  - 结束原因归一
  - `tool_use_id` 别名
  - 透传链路中的 `reason=end_turn` 归一到 `finish_reason=stop`

## 验证

- `go test ./internal/model ./internal/agent` 通过
- `go test ./...` 通过

## 当前边界

- 仅收敛最小高频结束原因，未覆盖 provider 全量原生 stop reason
