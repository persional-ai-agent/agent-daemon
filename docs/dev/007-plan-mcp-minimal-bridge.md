# 007 计划：MCP 最小接入骨架

## 目标

实现 MCP HTTP 最小桥接能力，使外部 MCP 工具可被本地 Agent 直接使用。

## 实施步骤

1. 新增 MCP client
验证：可请求 `/tools` 并解析工具定义

2. 新增动态注册流程
验证：启动时发现到的 MCP 工具可进入 `Registry`

3. 新增调用转发
验证：调用 MCP 工具时可向 `/call` 发送 `name + arguments + context`

4. 增加配置项
验证：`AGENT_MCP_ENDPOINT` 配置后可自动启用 MCP 发现

5. 增加测试并回归
验证：MCP 发现与调用转发测试通过，`go test ./...` 通过
