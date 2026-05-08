# 025 计划：`model_stream_event` v2 消息/工具 ID 来源兼容补齐

## 目标

提升 `message_id` 与 `tool_call_id` 的跨 provider 稳定性，进一步减少客户端分支判断。

## 实施步骤

1. 扩展 `normalizeStreamEvent`：
   - `message_id` 兼容 `response_id`、`message.id`
   - `tool_call_id` 兼容 `item_id`、`output_item_id`
2. Anthropic 流式路径补充：
   - 解析并透传 `message_start.message.id`
   - 透传 `message_delta.stop_reason` 供统一层归一
3. Codex completed envelope 补充：
   - 透传 `response.id` 到 `response_id`
4. 更新模型层测试覆盖新增别名来源。
5. 执行回归与文档同步。

## 验证标准

- `model_stream_event` 的 `message_*` 事件可稳定提取 `message_id`
- `tool_*` 与 `tool_args_*` 事件可稳定提取 `tool_call_id`
- 全量测试通过
