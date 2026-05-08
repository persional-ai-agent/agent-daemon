# 019 调研：Provider 增量事件透传（最小版）

## 背景

018 已补齐三种 provider 的流式聚合，但聚合过程中的增量事件仍停留在模型层，Agent 事件流无法感知。

## 缺口

- `Engine` 无法接收 provider 的流式增量事件
- SSE 客户端看不到模型生成中的中间进度

## 本轮目标

在不破坏现有 `model.Client` 基础接口的前提下，补齐最小透传：

- 增加可选模型事件接口（扩展接口，不替换原接口）
- OpenAI / Anthropic / Codex 在流式解析中上报增量事件
- `Engine` 统一转发为 `model_stream_event`

## 本轮边界

- 仅透传最小事件类型（文本增量、工具参数增量）
- 不做 provider-specific 的完整事件字典标准化
