# 022 总结：`model_stream_event` v2 参数生命周期补齐

## 已完成

- 事件标准化层新增参数生命周期：
  - `tool_args_start`
  - `tool_args_delta`
  - `tool_args_done`
- 兼容历史别名：
  - `tool_arguments_start/delta/done`
- 三 provider 流式路径均补发参数生命周期事件
- `message_done` 增加统一 `finish_reason`：
  - 有工具调用：`tool_calls`
  - 无工具调用：`stop`
- 更新模型层与 agent 层测试，覆盖别名映射与事件透传

## 验证

- `go test ./internal/model ./internal/agent ./...` 通过

## 当前边界

- 仍为统一字典最小集合，完整 provider 原生事件全集未纳入
