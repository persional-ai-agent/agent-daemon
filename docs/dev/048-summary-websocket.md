# 048 总结：WebSocket 端点 `/v1/chat/ws`

## 变更摘要

新增 `/v1/chat/ws` WebSocket 端点，使用 `gorilla/websocket` 协议实现双向实时通信，与现有 SSE 端点互补。

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/api/server.go` | 新增 `handleChatWS` 方法 + `/v1/chat/ws` 路由 + `wsUpgrader`；拆分细粒度事件事件 |

## 协议

### 客户端 → 服务端（首条消息）

```json
{"session_id": "uuid-or-empty", "message": "your message"}
```

### 服务端 → 客户端（事件流）

```json
{"type": "session", "session_id": "xxx"}
{"type": "user_message", "session_id": "xxx", "content": "hello"}
{"type": "turn_started", "session_id": "xxx", "turn": 1}
{"type": "model_stream_event", "session_id": "xxx", ...}
{"type": "assistant_message", "session_id": "xxx", "content": "..."}
{"type": "tool_started", "session_id": "xxx", "tool_name": "read_file", ...}
{"type": "tool_finished", "session_id": "xxx", "tool_name": "read_file", ...}
{"type": "completed", "session_id": "xxx", "content": "final response"}
{"type": "result", "session_id": "xxx", "final_response": "...", "summary": {...}}
```

### 事件类型

| 事件 | 说明 |
|------|------|
| `session` | 会话建立，返回 session_id |
| `user_message` | 用户消息已记录 |
| `turn_started` | 回合开始 |
| `model_stream_event` | 模型流式事件（text_delta, tool_call_start, etc.） |
| `assistant_message` | 助手响应完成 |
| `tool_started` | 工具开始执行 |
| `tool_finished` | 工具执行完成 |
| `completed` | Agent 运行完成 |
| `result` | 最终结果（含 summary） |
| `error` | 错误 |
| `cancelled` | 已取消 |

### 取消

通过 HTTP `/v1/chat/cancel` (POST `{"session_id":"xxx"}`) 取消活动中的 WebSocket 会话。

## 与 SSE 的对比

| 方面 | SSE (`/v1/chat/stream`) | WebSocket (`/v1/chat/ws`) |
|------|------------------------|--------------------------|
| 协议 | HTTP SSE | WebSocket |
| 双向 | 服务端 → 客户端 | 未来可扩展双向 |
| 浏览器 | 原生 EventSource | 需要 WebSocket 库 |
| 事件格式 | `event: type\ndata: json` | 纯 JSON 对象 |

## 测试结果

`go build ./...` ✅ | `go test ./...` 全部通过 ✅ | `go vet ./...` 无警告 ✅
