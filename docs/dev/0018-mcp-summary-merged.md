# 0018 mcp summary merged

## 模块

- `mcp`

## 类型

- `summary`

## 合并来源

- `0026-mcp-summary-merged.md`

## 合并内容

### 来源：`0026-mcp-summary-merged.md`

# 0026 mcp summary merged

## 模块

- `mcp`

## 类型

- `summary`

## 合并来源

- `0006-mcp-minimal-bridge.md`
- `0011-mcp-stdio-bridge.md`
- `0012-mcp-oauth-client-credentials.md`
- `0013-mcp-call-streaming-compat.md`
- `0041-mcp-streaming-passthrough.md`
- `0042-mcp-oauth-auth-code.md`

## 合并内容

### 来源：`0006-mcp-minimal-bridge.md`

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

### 来源：`0011-mcp-stdio-bridge.md`

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

### 来源：`0012-mcp-oauth-client-credentials.md`

# 013 总结：MCP OAuth（Client Credentials）最小补齐结果

## 已完成

- MCP HTTP 客户端新增 OAuth 配置与注入能力
- 支持 `client_credentials` 获取 access token
- `/tools` 与 `/call` 自动注入 `Authorization: Bearer <token>`
- 新增 token 缓存逻辑，避免重复申请
- 新增配置项：
  - `AGENT_MCP_OAUTH_TOKEN_URL`
  - `AGENT_MCP_OAUTH_CLIENT_ID`
  - `AGENT_MCP_OAUTH_CLIENT_SECRET`
  - `AGENT_MCP_OAUTH_SCOPES`
- 启动装配支持按配置启用 MCP OAuth
- 新增测试覆盖 OAuth token 申请、认证头注入、token 缓存复用

## 验证

- `go test ./internal/tools -run MCP -v` 通过
- `go test ./...` 通过

## 当前边界

- 未覆盖授权码模式与 refresh_token
- 未覆盖 MCP streaming 增量消息

### 来源：`0013-mcp-call-streaming-compat.md`

# 014 总结：MCP `/call` 流式响应兼容结果

## 已完成

- MCP `/call` 增加 `text/event-stream` 分支处理
- 新增最小 SSE 解析器，支持：
  - `data:` 多行拼接
  - `[DONE]` 终止标记
  - `result` / `structuredContent` 结果提取
  - `error` 事件错误返回
- 保持原 JSON 响应处理逻辑不变
- 新增 SSE `/call` 测试覆盖

## 验证

- `go test ./internal/tools -run MCP -v` 通过
- `go test ./...` 通过

## 当前边界

- 未将流式中间事件透传到 Agent 事件总线
- 仍以“聚合后一次返回”方式对接现有工具调用接口

### 来源：`0041-mcp-streaming-passthrough.md`

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

### 来源：`0042-mcp-oauth-auth-code.md`

# 043-summary: MCP OAuth 授权码流程与刷新令牌

## 实现结果

已完成 MCP OAuth 授权码流程（authorization_code）与刷新令牌（refresh_token）的完整实现，使 MCP 客户端能够对接需要用户交互授权的 OAuth 2.0 服务。

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/tools/mcp.go` | 扩展 `MCPOAuthConfig`（新增 AuthURL/RedirectURL/GrantType）；新增 `OAuthTokenStore` 接口；新增 `ConfigureOAuthAuthCode`、`refreshOAuthToken`、`ExchangeAuthCode`、`BuildAuthURL`、`StartOAuthCallbackServer` 等方法；重构 `oauthAccessToken` 支持刷新令牌与持久化加载 |
| `internal/store/session_store.go` | 新增 `oauth_tokens` 表；实现 `SaveOAuthToken`/`LoadOAuthToken`/`DeleteOAuthToken` 方法 |
| `internal/config/config.go` | 新增 `MCPOAuthAuthURL`/`MCPOAuthRedirectURL`/`MCPOAuthGrantType`/`MCPOAuthCallbackPort` 配置项及环境变量 |
| `cmd/agentd/main.go` | 启动时根据 GrantType 分支装配授权码或客户端凭证模式；授权码模式下启动回调服务器并等待授权完成 |
| `internal/tools/mcp_test.go` | 新增 `TestMCPClientOAuthRefreshToken`、`TestMCPClientOAuthAuthCodeExchange`、`TestMCPClientBuildAuthURL`、`TestMCPClientOAuthTokenPersistence` |
| `internal/store/session_store_test.go` | 新增 `TestSessionStoreOAuthTokenSaveLoadAndDelete`、`TestSessionStoreOAuthTokenLoadMissing` |

## 新增能力

1. **授权码流程**：通过 `ConfigureOAuthAuthCode` 配置，`BuildAuthURL` 生成授权链接，`ExchangeAuthCode` 用授权码换取令牌
2. **刷新令牌**：令牌过期时自动使用 refresh_token 刷新，无需重新授权
3. **令牌持久化**：通过 `OAuthTokenStore` 接口将令牌存入 SQLite，进程重启后自动加载
4. **回调服务器**：`StartOAuthCallbackServer` 在本地启动 HTTP 服务器接收授权回调
5. **配置驱动**：通过环境变量 `AGENT_MCP_OAUTH_GRANT_TYPE=authorization_code` 切换模式

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `AGENT_MCP_OAUTH_GRANT_TYPE` | OAuth 授权类型：`client_credentials` 或 `authorization_code` | `client_credentials` |
| `AGENT_MCP_OAUTH_AUTH_URL` | 授权端点 URL | - |
| `AGENT_MCP_OAUTH_REDIRECT_URL` | 回调地址 | - |
| `AGENT_MCP_OAUTH_CALLBACK_PORT` | 本地回调服务器端口 | `9876` |

## 测试结果

全部测试通过（`go test ./... -count=1`），包括：
- 令牌持久化 CRUD
- 刷新令牌自动续期
- 授权码交换
- 授权 URL 构建
- 令牌持久化集成
