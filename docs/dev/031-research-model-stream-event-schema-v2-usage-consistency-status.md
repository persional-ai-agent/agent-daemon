# 031 调研：`model_stream_event` v2 用量一致性状态字段补齐

## 背景

030 已补齐 `total_tokens` 缺失补齐与偏小校正，但客户端仍需通过多字段组合来判断“当前总量是原值、推导还是校正值”。

## 缺口

- 缺少统一状态字段表达 usage 总量一致性结果
- 客户端需要同时检查 `total_tokens`、`total_tokens_adjusted` 与上下文字段

## 本轮目标

新增统一状态字段：

- `usage_consistency_status`

最小状态集合：

- `ok`：上游总量与推导一致
- `derived`：由 `prompt + completion` 自动补齐
- `adjusted`：上游总量偏小，已校正
- `source_only`：仅有上游总量，无法推导校验

## 本轮边界

- 不改 provider 透传路径，仅在标准化层输出状态字段
