# 020 调研：`model_stream_event` 标准字典（v1）

## 背景

019 已补齐 provider 增量事件透传，但 `event_data` 字段仍存在 provider 差异（`delta`/`partial_json`/`name` 等别名）。

## 缺口

- 前端消费 `model_stream_event` 需要写 provider 分支
- 事件字段缺少统一最小标准

## 本轮目标

定义并落地最小标准字典（v1）：

- `text_delta`：
  - `event_data.text`
- `tool_arguments_delta`：
  - `event_data.tool_name`
  - `event_data.arguments_delta`

并兼容历史别名输入。

## 本轮边界

- 仅覆盖最小事件类型
- 未引入完整的 provider 事件枚举体系
