# 024 计划：`model_stream_event` v2 结束原因与 ID 别名归一

## 目标

继续增强 `model_stream_event` 字典一致性，减少客户端 provider 分支判断。

## 实施步骤

1. 在 `normalizeStreamEvent` 中新增 `finish_reason` 归一：
   - `end_turn -> stop`
   - `tool_use -> tool_calls`
   - `max_tokens/max_output_tokens -> length`
2. 在 `tool_call_*` 与 `tool_args_*` 标准化中补齐 `tool_use_id -> tool_call_id`。
3. 更新模型层测试，覆盖结束原因归一与 `tool_use_id` 别名。
4. 更新 agent 层透传测试，验证 `message_done.reason=end_turn` 时输出标准 `finish_reason=stop`。
5. 执行回归测试并同步文档。

## 验证标准

- `message_done.finish_reason` 可稳定落在最小标准集合
- `tool_use_id` 事件可被统一消费为 `tool_call_id`
- 测试通过
