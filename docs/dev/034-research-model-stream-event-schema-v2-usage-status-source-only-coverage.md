# 034 调研：`model_stream_event` v2 用量 `source_only` 状态 provider 覆盖测试

## 背景

033 已覆盖 `ok/derived/invalid` 的 provider 维度测试，但 `source_only` 仍缺同层级覆盖。

## 缺口

- OpenAI/Anthropic/Codex 在仅提供 `total_tokens` 时，尚未显式验证统一输出 `usage_consistency_status=source_only`

## 本轮目标

补齐 `source_only` 的 provider 覆盖测试：

- OpenAI
- Anthropic
- Codex

## 本轮边界

- 仅补测试，不修改协议与标准化逻辑
