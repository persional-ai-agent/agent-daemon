# 023 调研：`model_stream_event` v2 用量事件补齐

## 背景

022 已完成参数生命周期与 `message_done.finish_reason` 的统一，但客户端仍无法稳定获取跨 provider 的 token 用量信息。

## 缺口

- OpenAI / Anthropic / Codex 都可能返回 usage，但字段命名不一致
- 当前 `model_stream_event` 未定义统一 `usage` 事件，前端/SDK 需要自行做 provider 分支

## 本轮目标

补齐最小可用的统一用量事件：

- `event_type=usage`
- 标准字段：
  - `prompt_tokens`
  - `completion_tokens`
  - `total_tokens`

并兼容常见别名：

- `input_tokens -> prompt_tokens`
- `output_tokens -> completion_tokens`

## 本轮边界

- 仅补“最小统一字段”，不引入 provider 原生完整计费明细
- 不改变 `model_stream_event` 外层结构（仍是 `provider` + `event_type` + `event_data`）
