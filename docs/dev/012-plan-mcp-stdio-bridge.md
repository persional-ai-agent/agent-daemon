# 012 计划：MCP `stdio` 最小桥接

## 目标

新增 `stdio` 传输能力，使 MCP 工具可通过子进程标准输入输出接入。

## 实施步骤

1. 扩展 `internal/tools/mcp.go`：
   - `MCPClient` 增加 `StdioCommand`
   - 新增 `NewMCPStdioClient`
   - `DiscoverTools` 和工具 `Call` 增加 `stdio` 分支
2. 新增最小 JSON-RPC framing：
   - 读写 `Content-Length` 帧
   - 请求/响应 `id` 匹配
3. 新增一次会话调用流程：
   - 启动子进程
   - `initialize` + `notifications/initialized`
   - `tools/list` 或 `tools/call`
4. 扩展配置与启动装配：
   - `AGENT_MCP_TRANSPORT`
   - `AGENT_MCP_STDIO_COMMAND`
5. 增加 `mcp_test` 的 `stdio` 子进程测试（helper process）。
6. 执行全量测试 `go test ./...`。

## 验证标准

- HTTP 模式行为不回退
- `stdio` 模式可发现并调用 MCP 工具
- 全量测试通过
