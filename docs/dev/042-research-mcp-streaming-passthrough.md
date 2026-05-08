# 042 调研：MCP 流式事件透传

## 背景

014 已补齐 MCP `/call` 的 SSE 兼容——当 MCP 服务返回 `text/event-stream` 时，`parseMCPCallSSE` 会聚合所有事件为最终结果一次性返回。但这意味着：

- 客户端无法实时感知 MCP 工具的执行进度
- 长时间运行的 MCP 工具（如代码分析、文件索引）在 SSE 模式下表现为"黑盒等待"
- Agent 事件总线已有 `model_stream_event` 透传模型层增量事件，但 MCP 工具层没有类似机制

## 当前架构分析

### MCP 调用链路

```
Agent Loop → Registry.Dispatch → mcpToolProxy.Call → HTTP POST /call → parseMCPCallSSE → 聚合返回
```

- `mcpToolProxy.Call` 返回 `(map[string]any, error)`，是同步接口
- `parseMCPCallSSE` 聚合所有 SSE 事件后才返回
- `Registry.Dispatch` 将结果序列化为 JSON string 返回给 Agent Loop
- Agent Loop 在 `tool_started` 和 `tool_finished` 之间无法感知中间进度

### Agent 事件系统

- `Engine.EventSink` 接收 `core.AgentEvent`
- `model_stream_event` 已支持模型层增量事件透传
- 工具层仅有 `tool_started` / `tool_finished`，无中间事件

### Tool 接口

```go
type Tool interface {
    Name() string
    Schema() core.ToolSchema
    Call(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error)
}
```

- `ToolContext` 当前无事件回调能力
- `Registry.Dispatch` 返回 `string`，无法传递中间状态

## 缺口

1. MCP SSE 中间事件被丢弃，客户端无法实时感知 MCP 工具执行进度
2. `ToolContext` 无事件回调，工具无法向 Agent 事件总线发送中间事件
3. Agent Loop 在工具执行期间无法发出增量事件

## 本轮目标

在保持现有同步调用接口不变的前提下，让 MCP SSE 中间事件透传到 Agent 事件总线：

1. 扩展 `ToolContext` 增加可选事件回调 `ToolEventSink`
2. MCP `/call` SSE 模式下，每解析到一个事件就通过回调发送
3. Agent Loop 在工具执行时注册回调，将 MCP 中间事件映射为 `mcp_stream_event`
4. SSE 客户端可直接消费 `mcp_stream_event`，实现实时进度感知

## 本轮边界

- 不改变 `Tool.Call` 的同步返回语义（最终结果仍由返回值提供）
- 不做 stdio MCP 的流式事件透传（stdio 使用 JSON-RPC 帧，无增量事件语义）
- 不做 MCP 会话绑定与多路并发控制
- 不引入新的 SSE 事件协议字段，复用现有 `AgentEvent` 结构

## 方案

### 1. 扩展 ToolContext

在 `ToolContext` 中新增可选字段：

```go
type ToolEventSink func(eventType string, data map[string]any)

type ToolContext struct {
    // ... existing fields
    ToolEventSink ToolEventSink // optional: emit intermediate tool events
}
```

### 2. MCP SSE 逐事件回调

改造 `mcpToolProxy.Call` 的 SSE 分支：

- 不再使用 `parseMCPCallSSE`（聚合模式）
- 新增 `parseMCPCallSSEWithCallback`，每解析到一个 SSE 事件就调用 `tc.ToolEventSink`
- 最终结果仍由函数返回值提供

### 3. Agent Loop 注册回调

在 `Engine.Run` 的工具执行循环中：

```go
tc := tools.ToolContext{
    // ... existing fields
    ToolEventSink: func(eventType string, data map[string]any) {
        e.emit(core.AgentEvent{
            Type:      "mcp_stream_event",
            SessionID: sessionID,
            Turn:      turn + 1,
            ToolName:  tc.Function.Name,
            Data: map[string]any{
                "tool_name":   tc.Function.Name,
                "event_type":  eventType,
                "event_data":  data,
            },
        })
    },
}
```

### 4. SSE 透传

`api/server.go` 的 SSE handler 已按 `event.Type` 透传所有 `AgentEvent`，新增的 `mcp_stream_event` 会自动透传到客户端，无需额外修改。

## 风险

- MCP SSE 事件格式无统一标准，不同 MCP 服务可能返回不同结构——透传时保留原始数据，不做标准化
- `ToolEventSink` 是可选的，不影响非 MCP 工具和现有测试
