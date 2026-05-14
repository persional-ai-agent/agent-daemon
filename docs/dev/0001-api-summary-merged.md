# 0001 api summary merged

## 模块

- `api`

## 类型

- `summary`

## 合并来源

- `0004-chat-summary-merged.md`
- `0057-websocket-summary-merged.md`

## 合并内容

### 来源：`0004-chat-summary-merged.md`

# 0004 chat summary merged

## 模块

- `chat`

## 类型

- `summary`

## 合并来源

- `0235-chat-api-contract-alignment.md`
- `0242-chat-stream-contract-replay.md`

## 合并内容

### 来源：`0235-chat-api-contract-alignment.md`

# Chat API 契约对齐（/v1/chat*）

本轮将非 UI 的 chat 系列接口也统一到结构化错误语义，并保持向后兼容字段。

## 覆盖范围

- `POST /v1/chat`
- `POST /v1/chat/stream`（握手前错误）
- `GET/POST /v1/chat/cancel`
- `GET/WS /v1/chat/ws`（WS 错误事件）

## 统一规则

- 错误响应统一为：
  - `ok: false`
  - `error.code`
  - `error.message`
  - `api_version`
  - `compat`
- Header 统一携带：
  - `X-Agent-UI-API-Version`
  - `X-Agent-UI-API-Compat`

## 向后兼容

- `/v1/chat` 成功响应新增 `result` 包装的同时，保留历史顶层字段：
  - `session_id/final_response/messages/turns_used/finished_naturally/summary`
- `/v1/chat/cancel` 保留历史顶层字段：
  - `session_id`
  - `cancelled`

## 测试

- 新增 `internal/api/chat_contract_test.go`
- 增强 `internal/api/ui_contract_test.go` 通用错误 envelope 断言

### 来源：`0242-chat-stream-contract-replay.md`

# 快速完善产品功能：纳入 Chat Stream 契约与回放

本轮将实时流式对话链路（`POST /v1/chat/stream`）纳入正式契约体系，避免只覆盖非流式接口。

## 主要改动

- OpenAPI 契约补齐
  - `docs/api/ui-chat-contract.openapi.yaml` 新增 `/v1/chat/stream`
  - 定义 `text/event-stream` 成功响应与 4XX 错误响应
- Replay 回放增强
  - `internal/api/testdata/replay/cases.json` 新增 `chat_stream_success`
  - `internal/api/contract_replay_test.go` 支持 SSE 用例：
    - 校验 `Content-Type: text/event-stream`
    - 校验关键事件标记（`event: session`、`event: result`）
- 覆盖率门禁升级
  - `scripts/contract_coverage/main.go` 将 `POST /v1/chat/stream` 纳入核心端点统计
  - 核心覆盖率继续要求 100%
- 规则文档更新
  - `docs/api/contract-versioning.md` 更新核心端点列表

## 验证

- `make contract-check`
- `go test ./...`

### 来源：`0057-websocket-summary-merged.md`

# 0057 websocket summary merged

## 模块

- `websocket`

## 类型

- `summary`

## 合并来源

- `0047-websocket.md`

## 合并内容

### 来源：`0047-websocket.md`

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
