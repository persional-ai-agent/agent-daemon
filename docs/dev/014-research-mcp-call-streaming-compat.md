# 014 调研：MCP `/call` 流式响应兼容

## 背景

013 已补齐 MCP HTTP OAuth（client_credentials），但 `/call` 仍默认按一次性 JSON 响应处理。  
部分 MCP 服务会通过 `text/event-stream` 分块返回结果，当前实现无法消费。

## 缺口

- MCP `/call` 缺少 SSE 解析逻辑
- 流式返回下无法得到可用的最终工具结果

## 本轮目标

在保持现有同步调用接口不变的前提下，补齐最小流式兼容：

- 当 `Content-Type` 为 `text/event-stream` 时，解析 SSE `data:` 事件
- 支持常见事件形态：
  - `{"result": {...}}`
  - `{"structuredContent": {...}}`
  - `{"error": {...}}`
  - `[DONE]`
- 聚合为最终 `map[string]any` 返回给现有工具链路

## 本轮边界

- 不做增量事件透传到 Agent EventSink
- 不做多路并发流内控制协议
