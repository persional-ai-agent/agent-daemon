# 091 Summary - mixture_of_agents 工具实现（基于 delegate_task 子代理聚合）

## 背景

Hermes core tools 中包含 `mixture_of_agents`（MoA），用于通过多个参考模型/代理并行产出，再由聚合器综合得到更可靠的最终答案。

Go 版 `agent-daemon` 之前仅提供 stub。

## 变更

- 新增 `mixture_of_agents` 最小实现：
  - 并行运行多个 reference subagents（复用现有 `delegate_task` 的子代理执行能力）
  - 再运行一个 aggregator subagent，对 reference 输出做综合与纠错
  - 返回 `references` + `aggregated`

实现位置：

- `internal/tools/moa.go`
- `internal/tools/builtin.go`：注册 + schema

## 边界

- 本实现使用同一后端模型的子代理来模拟“多模型” MoA；与 Hermes 的 OpenRouter 多模型 MoA 不同，但能实现“多视角 + 聚合”的核心效果。

