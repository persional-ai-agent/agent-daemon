# 026 调研：`model_stream_event` v2 终止元数据补齐

## 背景

当前 `message_done` 已统一 `finish_reason`，但客户端在处理“为何中止/由何序列中止”时仍缺统一字段。

## 缺口

- 不同 provider 对终止信息命名不一致：
  - `stop_sequence` / `stop`
  - `incomplete_details.reason` / `reason_detail`
- 现有统一层未稳定提供这两个终止元字段。

## 本轮目标

在不改变事件外层结构前提下，为 `message_done` 增加最小终止元数据：

- `stop_sequence`
- `incomplete_reason`

并在 provider 流式路径尽量透传上游来源字段。

## 本轮边界

- 不扩展新的事件类型
- 仅处理高频终止元字段
