# 036 调研：`model_stream_event` v2 用量 `adjusted` 状态端到端覆盖

## 背景

035 已覆盖 `source_only/invalid` 的 provider 流式端到端路径，但 `adjusted` 仍缺同层级验证。

## 缺口

- 当上游 `total_tokens` 偏小且需要标准化层校正时，尚未在 provider 流式路径做 E2E 断言
- 客户端最依赖该场景来识别“校正后总量”

## 本轮目标

补齐 `adjusted` 端到端覆盖：

- OpenAI 流式 usage（`total_tokens` 偏小） -> `adjusted`
- Codex completed envelope usage（`total_tokens` 偏小） -> `adjusted`

并同时断言：

- `total_tokens_adjusted=true`
- `total_tokens` 为校正后的值

## 本轮边界

- 仅补测试，不改字段规则
