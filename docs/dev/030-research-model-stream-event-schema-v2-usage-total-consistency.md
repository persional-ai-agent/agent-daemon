# 030 调研：`model_stream_event` v2 用量总量一致性补齐

## 背景

`usage` 事件已统一多个 token 字段，但 `total_tokens` 在部分 provider 场景可能缺失，或与 `prompt_tokens + completion_tokens` 不一致。

## 缺口

- `total_tokens` 缺失时，客户端仍需自行回推
- `total_tokens` 偏小时，客户端会出现展示和计费估算不一致

## 本轮目标

- 在统一层提供 `total_tokens` 兜底：
  - 缺失时自动按 `prompt_tokens + completion_tokens` 补齐
  - 当上游 `total_tokens` 小于 `prompt + completion` 时自动校正
- 增加校正标记字段：
  - `total_tokens_adjusted=true`

## 本轮边界

- 不修改 provider 原始 usage 透传逻辑，仅在标准化阶段做一致性兜底
