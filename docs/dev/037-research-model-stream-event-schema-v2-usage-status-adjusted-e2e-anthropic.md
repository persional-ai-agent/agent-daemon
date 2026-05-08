# 037 调研：`model_stream_event` v2 用量 `adjusted` 状态 Anthropic 端到端补齐

## 背景

036 已补 OpenAI/Codex 的 `adjusted` 端到端覆盖，Anthropic 仍是缺口。

## 缺口

- 三 provider 中仅 Anthropic 未验证：
  - 流式 usage 上游给出偏小 `total_tokens`
  - 经 `CompleteWithEvents` 归一后输出 `usage_consistency_status=adjusted`

## 本轮目标

补齐 Anthropic 的 `adjusted` 端到端测试，并断言：

- `usage_consistency_status=adjusted`
- `total_tokens_adjusted=true`
- `total_tokens` 为校正值

## 本轮边界

- 仅补测试覆盖，不修改标准化规则
