# 027 调研：`model_stream_event` v2 终止原因一致性补齐

## 背景

026 已引入 `stop_sequence` 与 `incomplete_reason`，但 `finish_reason` 与 `incomplete_reason` 之间仍可能出现语义不一致。

## 缺口

- `finish_reason=length` 时，部分 provider 不会显式给出 `incomplete_reason`
- `incomplete_reason` 存在别名值（如 `max_tokens`、`max_output_tokens`），客户端仍需自行归一

## 本轮目标

- 统一 `incomplete_reason` 常见枚举到最小集合
- 当 `finish_reason=length` 且缺少 `incomplete_reason` 时，自动补 `incomplete_reason=length`

## 本轮边界

- 仅处理最小高频枚举，不引入 provider 全量终止诊断信息
