# 017 调研：Provider 流式统一（Anthropic 最小落地）

## 背景

016 已完成 OpenAI 流式聚合，但 Anthropic 仍为非流式调用。  
要推进 provider 流式统一，需要继续补齐 Anthropic。

## 缺口

- Anthropic 客户端未启用 `stream=true`
- 缺少 Anthropic SSE 事件聚合逻辑

## 本轮目标

以最小改动补齐 Anthropic 流式聚合：

- 增加 `UseStreaming` 开关
- 开关开启时发送 `stream=true`
- 解析并聚合核心流式事件：
  - `content_block_start`
  - `content_block_delta`（`text_delta` / `input_json_delta`）
- 聚合为标准 `core.Message` 返回

## 本轮边界

- Codex 流式暂未补齐
- 暂不透传 provider 级增量事件到 Agent 事件流
