# 012 总结：MCP `stdio` 最小桥接结果

## 已完成

- MCP 客户端新增 `stdio` 支持：
  - `NewMCPStdioClient`
  - `MCPClient.StdioCommand`
- MCP `stdio` 会话实现最小 JSON-RPC 流程：
  - `initialize`
  - `notifications/initialized`
  - `tools/list`
  - `tools/call`
- 工具发现与调用已支持 `http`/`stdio` 双通道
- 新增配置项：
  - `AGENT_MCP_TRANSPORT`（默认 `http`）
  - `AGENT_MCP_STDIO_COMMAND`
- 启动装配支持根据传输模式注册 MCP 工具
- 新增 `stdio` 子进程测试，验证发现与调用链路

## 验证

- `go test ./internal/tools -run MCP -v` 通过
- `go test ./...` 通过

## 当前边界

- 未覆盖 OAuth
- 未覆盖 MCP streaming 增量消息
- `stdio` 使用一次调用一会话，暂未做长连接复用
