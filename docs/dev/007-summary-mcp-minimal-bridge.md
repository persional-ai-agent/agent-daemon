# 007 总结：MCP 最小接入骨架

## 已完成

- 新增 `internal/tools/mcp.go`：
  - `MCPClient`
  - `RegisterMCPTools`
  - `mcpToolProxy`
- 启动阶段支持按 `AGENT_MCP_ENDPOINT` 自动发现并注册 MCP 工具
- MCP 工具调用会转发到 `/call`，并携带 `session_id` 与 `workdir` 上下文
- 新增 `internal/tools/mcp_test.go`，覆盖发现注册、调用转发、发现失败场景

## 接口约定（当前版本）

- 发现：`GET {AGENT_MCP_ENDPOINT}/tools`
  - 返回：`{"tools":[{"name","description","parameters"}]}`
- 调用：`POST {AGENT_MCP_ENDPOINT}/call`
  - 请求：`{"name","arguments","context":{"session_id","workdir"}}`

## 验证

- `go test ./...` 通过

## 当前边界

当前仅支持 HTTP 最小桥接，不含：

- stdio transport
- MCP OAuth
- 流式工具调用
