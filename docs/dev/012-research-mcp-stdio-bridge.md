# 012 调研：MCP `stdio` 最小桥接

## 背景

当前 Go 版 MCP 已支持 HTTP 桥接（`/tools` + `/call`），但 Hermes 侧还覆盖 `stdio` 场景。  
这导致本项目无法直接接入仅暴露标准输入输出通道的 MCP Server。

## 差异点

- 已有：HTTP MCP 发现与调用
- 缺失：`stdio` MCP 发现与调用

## 本轮目标

在不破坏现有 HTTP 桥接的前提下，补一个最小 `stdio` 实现：

- 配置可切换 MCP 传输模式（`http` / `stdio`）
- `stdio` 会话支持最小 JSON-RPC 流程：
  - `initialize`
  - `notifications/initialized`
  - `tools/list`
  - `tools/call`
- 保持“发现工具 -> 注册代理 -> 调用透传”的现有结构不变

## 本轮边界

- 不做 OAuth
- 不做 streaming 增量消息消费
- 不做跨调用持久连接池（采用一次调用一会话的简化策略）
