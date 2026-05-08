# 022 调研：`model_stream_event` v2 参数生命周期补齐

## 背景

021 已补 `message_*` 与 `tool_call_*` 生命周期，但工具参数仍只有增量事件，缺少参数生命周期起止。

## 缺口

- 客户端难以判定“参数开始接收”和“参数拼装完成”
- `message_done` 缺少统一 `finish_reason`，终止原因不够稳定

## 本轮目标

补齐最小参数生命周期与结束原因字段：

- `tool_args_start`
- `tool_args_delta`
- `tool_args_done`
- `message_done.finish_reason`

并兼容历史别名：

- `tool_arguments_start/delta/done`

## 本轮边界

- 不引入 provider 全量原生事件
- 仍以 Agent 统一事件字典为主
