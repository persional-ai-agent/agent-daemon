# 003 调研：Context Compression 补齐

## 背景

在 002 阶段之后，Go 版已补齐 system prompt、memory 回灌、工作区规则与基础安全护栏。

剩余核心体验差异中，优先级最高的是长会话场景下的上下文膨胀问题：历史消息与工具输出持续增长，会带来请求失败风险和成本上升。

Hermes 在 `agent/context_compressor.py` 里通过中段压缩、头尾保护、摘要前缀和预算控制解决这一问题。

## 当前项目缺口

- 无上下文预算控制
- 无自动压缩触发
- 无“压缩后摘要消息”机制
- 无压缩可观测事件

这会导致长 session 下的 `messages` 不断变大。

## 方案选择

本次采用“最小可用压缩器”而非一次性复刻 Hermes 的辅助模型摘要器：

- 使用字符预算估算上下文体积（`max_context_chars`）
- 保留 system message 与最近 N 条消息（`compression_tail_messages`）
- 将中段历史压缩为一条 assistant 摘要消息
- 输出 `context_compacted` 结构化事件

不在本次引入：

- 额外 summarizer 模型调用
- 多轮迭代摘要合并策略
- 多模态精细 token 估算

## 结论

该方案能在不新增外部依赖和模型调用的前提下，立即补齐长会话核心能力，并与现有 Engine 架构自然集成。
