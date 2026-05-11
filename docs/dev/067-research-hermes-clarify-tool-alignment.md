# 067 调研：Hermes clarify 工具与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 提供 `clarify` 工具用于结构化澄清问题（尤其是消息平台/CLI UI 可以渲染选项时），让 agent 在关键决策点向用户确认。

## 当前项目差异

Go 版此前没有 `clarify` 工具，agent 只能直接用自然语言提问，缺少“选项/结构化”的标准出口。

## 最小对齐目标（本次）

- 新增 `clarify` 工具：
  - 输入：`question`、可选 `options[{label,description}]`、`allow_freeform`
  - 输出：结构化 payload，提示上层/UI/模型向用户提问并收集答案

## 边界

- 不做交互式 UI（仅返回结构化数据）；用户回答仍通过下一条 user message 回到 agent loop。

