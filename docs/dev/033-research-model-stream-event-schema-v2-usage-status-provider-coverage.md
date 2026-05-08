# 033 调研：`model_stream_event` v2 用量状态 provider 覆盖测试

## 背景

031/032 已完成 `usage_consistency_status` 字段及 `invalid` 状态，但缺少按 provider 维度的覆盖测试。

## 缺口

- OpenAI/Anthropic/Codex 在使用统一字典时，状态断言主要覆盖通用路径
- 尚未显式验证各 provider 典型输入是否稳定落到预期状态

## 本轮目标

补齐 provider 维度的最小测试覆盖：

- OpenAI：`ok`
- Anthropic：`derived`
- Codex：`invalid`

## 本轮边界

- 仅补测试覆盖，不变更字段设计与事件协议
