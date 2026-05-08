# 029 调研：`model_stream_event` v2 用量推理 token 字段补齐

## 背景

028 已补齐缓存 token 字段，但推理类 token 统计在不同 provider 的 usage 字段中仍不统一。

## 缺口

- 常见来源字段差异：
  - `completion_tokens_details.reasoning_tokens`
  - `output_tokens_details.reasoning_tokens`
  - 以及部分实现中的平铺别名字段
- 客户端难以跨 provider 一致展示“推理 token”开销。

## 本轮目标

在 `usage` 事件增加最小统一字段：

- `reasoning_tokens`

## 本轮边界

- 仅补推理 token 统一字段，不扩展更细粒度思考链路计费维度
