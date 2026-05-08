# 042 计划：MCP 流式事件透传

## 目标

让 MCP `/call` SSE 中间事件透传到 Agent 事件总线，客户端可通过 SSE 实时感知 MCP 工具执行进度。

## 实施步骤

### 步骤 1：扩展 ToolContext 增加 ToolEventSink

文件：`internal/tools/registry.go`

- 新增 `ToolEventSink` 类型：`func(eventType string, data map[string]any)`
- 在 `ToolContext` 中新增 `ToolEventSink` 可选字段

### 步骤 2：新增 parseMCPCallSSEWithCallback

文件：`internal/tools/mcp.go`

- 新增 `parseMCPCallSSEWithCallback(body io.Reader, sink ToolEventSink) (map[string]any, error)`
- 每解析到一个 SSE data 事件就调用 `sink`
- 最终聚合结果仍由返回值提供
- 保留原 `parseMCPCallSSE` 不变（无回调场景使用）

### 步骤 3：改造 mcpToolProxy.Call 使用回调

文件：`internal/tools/mcp.go`

- SSE 分支判断 `tc.ToolEventSink != nil` 时使用 `parseMCPCallSSEWithCallback`
- 否则使用原 `parseMCPCallSSE`

### 步骤 4：Agent Loop 注册回调

文件：`internal/agent/loop.go`

- 在工具执行循环中，构造 `ToolContext` 时注册 `ToolEventSink`
- 回调内发出 `mcp_stream_event` 类型的 `AgentEvent`

### 步骤 5：增加测试

文件：`internal/tools/mcp_test.go`、`internal/agent/loop_test.go`

- MCP SSE 回调测试：验证中间事件被回调、最终结果正确
- Agent Loop 测试：验证 `mcp_stream_event` 被发出
- 原有聚合模式测试不回归

### 步骤 6：全量回归

`go test ./...`

## 验证标准

- MCP SSE 中间事件通过 `mcp_stream_event` 透传到 SSE 客户端
- 最终结果仍由 `tool_finished` 事件提供
- 非 MCP 工具和原有聚合模式不受影响
- 全量测试通过
