# 042 总结：MCP 流式事件透传

## 变更摘要

1. MCP `/call` SSE 中间事件可透传到 Agent 事件总线
2. 新增 `ToolEventSink` 回调机制，工具可向 Agent 发送中间事件
3. Agent Loop 自动将工具中间事件映射为 `mcp_stream_event`
4. SSE 客户端可直接消费 `mcp_stream_event`，实时感知 MCP 工具执行进度

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/tools/registry.go` | 新增 `ToolEventSink` 类型与 `ToolContext.ToolEventSink` 字段 |
| `internal/tools/mcp.go` | 新增 `parseMCPCallSSEWithCallback`；`mcpToolProxy.Call` SSE 分支根据 `ToolEventSink` 选择回调或聚合模式 |
| `internal/agent/loop.go` | 工具执行循环注册 `ToolEventSink`，将中间事件映射为 `mcp_stream_event` |
| `internal/tools/mcp_test.go` | 新增 2 个测试：SSE 回调透传、无回调回退聚合 |
| `internal/agent/loop_test.go` | 新增 1 个测试：Agent Loop 发出 `mcp_stream_event` |

## 新增能力

### ToolEventSink

```go
type ToolEventSink func(eventType string, data map[string]any)
```

- 可选字段，不影响现有工具
- MCP SSE 模式下，每解析到一个事件就调用 `ToolEventSink`
- 非 MCP 工具和 stdio MCP 不受影响

### mcp_stream_event

Agent Loop 在工具执行期间发出的新事件类型：

```json
{
  "type": "mcp_stream_event",
  "session_id": "...",
  "turn": 1,
  "tool_name": "mcp_tool_name",
  "data": {
    "tool_name": "mcp_tool_name",
    "event_type": "progress",
    "event_data": {"percent": 50}
  }
}
```

- `event_type`：来自 MCP SSE 事件的 `type` 字段，无 `type` 字段时默认为 `mcp_event`
- `event_data`：MCP SSE 事件的原始 JSON 数据

### SSE 客户端消费

`api/server.go` 的 SSE handler 已按 `event.Type` 透传所有 `AgentEvent`，`mcp_stream_event` 自动透传到客户端，无需额外修改。

## 向后兼容

- `ToolEventSink` 为可选字段，现有工具和测试无需修改
- 无 `ToolEventSink` 时，MCP SSE 仍使用原聚合模式 `parseMCPCallSSE`
- `mcp_stream_event` 是新增事件类型，不影响现有事件消费

## 测试结果

`go test ./...` 全部通过，新增 3 个测试用例覆盖本次变更。
