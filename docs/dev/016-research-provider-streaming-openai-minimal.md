# 016 调研：Provider 流式统一（OpenAI 最小落地）

## 背景

015 已补齐 provider 故障切换，但模型调用仍以非流式为主。  
当前系统层面已经有 SSE 输出能力，模型层需要先具备统一的流式聚合能力作为基础。

## 缺口

- OpenAI 客户端未使用 `stream=true`
- 模型层缺少“流式片段 -> 最终 `core.Message`”聚合逻辑

## 本轮目标

以最小改动先落地 OpenAI 流式聚合：

- 增加配置开关 `AGENT_MODEL_USE_STREAMING`
- OpenAI 在开关开启时走 `chat/completions` 流式模式
- 解析 SSE `data:` 事件并聚合：
  - assistant 文本增量
  - tool call 增量参数拼接
- 对外仍返回原有 `ChatCompletion` 的 `core.Message`

## 本轮边界

- Anthropic/Codex 暂未接入流式模式
- 暂不透传 provider 级增量事件到 Agent 事件流
