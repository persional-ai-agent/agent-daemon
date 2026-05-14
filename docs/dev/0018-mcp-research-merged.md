# 0018 mcp research merged

## 模块

- `mcp`

## 类型

- `research`

## 合并来源

- `0026-mcp-research-merged.md`

## 合并内容

### 来源：`0026-mcp-research-merged.md`

# 0026 mcp research merged

## 模块

- `mcp`

## 类型

- `research`

## 合并来源

- `0006-mcp-minimal-bridge.md`
- `0011-mcp-stdio-bridge.md`
- `0012-mcp-oauth-client-credentials.md`
- `0013-mcp-call-streaming-compat.md`
- `0041-mcp-streaming-passthrough.md`
- `0042-mcp-oauth-auth-code.md`

## 合并内容

### 来源：`0006-mcp-minimal-bridge.md`

# 007 调研：MCP 最小接入骨架

## 背景

当前项目工具系统已具备本地工具注册与分发能力，但尚未支持通过 MCP 生态接入外部工具。

在不引入复杂 stdio/OAuth/会话管理的前提下，本阶段先补齐可用骨架：HTTP 发现 + HTTP 调用转发。

## 本次范围

- 支持从 MCP 端点发现工具 schema（`GET /tools`）
- 将发现到的工具动态注册到本地 `Registry`
- 支持将工具调用转发到 MCP 端点（`POST /call`）
- 调用时传递最小上下文：`session_id`、`workdir`

不纳入：

- stdio transport
- OAuth / 鉴权协商
- 复杂会话绑定和流式调用

## 结论

该方案能以最小成本打通 “MCP 工具发现 -> 注册 -> 调用” 主链路，为后续接入完整 MCP 协议能力提供稳定起点。

### 来源：`0011-mcp-stdio-bridge.md`

# 012 调研：MCP `stdio` 最小桥接

## 背景

当前 Go 版 MCP 已支持 HTTP 桥接（`/tools` + `/call`），但 Hermes 侧还覆盖 `stdio` 场景。  
这导致本项目无法直接接入仅暴露标准输入输出通道的 MCP Server。

## 差异点

- 已有：HTTP MCP 发现与调用
- 缺失：`stdio` MCP 发现与调用

## 本轮目标

在不破坏现有 HTTP 桥接的前提下，补一个最小 `stdio` 实现：

- 配置可切换 MCP 传输模式（`http` / `stdio`）
- `stdio` 会话支持最小 JSON-RPC 流程：
  - `initialize`
  - `notifications/initialized`
  - `tools/list`
  - `tools/call`
- 保持“发现工具 -> 注册代理 -> 调用透传”的现有结构不变

## 本轮边界

- 不做 OAuth
- 不做 streaming 增量消息消费
- 不做跨调用持久连接池（采用一次调用一会话的简化策略）

### 来源：`0012-mcp-oauth-client-credentials.md`

# 013 调研：MCP OAuth（Client Credentials）最小补齐

## 背景

012 已补齐 MCP `stdio`，但 MCP HTTP 仍缺少 OAuth 能力，无法接入要求 Bearer Token 的 MCP 服务。

## 缺口

- HTTP MCP 请求无 `Authorization` 注入
- 无 token 申请与缓存机制

## 本轮目标

补齐最小 OAuth 能力（`client_credentials`）：

- 通过 token endpoint 获取 access token
- 为 MCP `/tools` 与 `/call` 自动注入 `Authorization: Bearer ...`
- 在内存中缓存 token，避免每次请求都换 token

## 本轮边界

- 不做授权码模式
- 不做 refresh_token 流程
- 不做多租户/多会话 token 隔离（当前进程级单配置）

### 来源：`0013-mcp-call-streaming-compat.md`

# 014 调研：MCP `/call` 流式响应兼容

## 背景

013 已补齐 MCP HTTP OAuth（client_credentials），但 `/call` 仍默认按一次性 JSON 响应处理。  
部分 MCP 服务会通过 `text/event-stream` 分块返回结果，当前实现无法消费。

## 缺口

- MCP `/call` 缺少 SSE 解析逻辑
- 流式返回下无法得到可用的最终工具结果

## 本轮目标

在保持现有同步调用接口不变的前提下，补齐最小流式兼容：

- 当 `Content-Type` 为 `text/event-stream` 时，解析 SSE `data:` 事件
- 支持常见事件形态：
  - `{"result": {...}}`
  - `{"structuredContent": {...}}`
  - `{"error": {...}}`
  - `[DONE]`
- 聚合为最终 `map[string]any` 返回给现有工具链路

## 本轮边界

- 不做增量事件透传到 Agent EventSink
- 不做多路并发流内控制协议

### 来源：`0041-mcp-streaming-passthrough.md`

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

### 来源：`0042-mcp-oauth-auth-code.md`

# 043 调研：MCP OAuth 授权码模式与刷新令牌

## 背景

013 已补齐 MCP OAuth `client_credentials` 模式，支持服务间认证。但许多 MCP 服务（如 GitHub、Google Drive、Slack 等）要求用户交互授权，即 OAuth 2.0 授权码模式（Authorization Code Flow）。当前实现无法接入这类服务。

此外，`client_credentials` 模式下 token 过期后只能重新申请，没有 `refresh_token` 刷新机制，导致频繁的 token 申请请求。

## 当前实现分析

### MCPOAuthConfig

```go
type MCPOAuthConfig struct {
    TokenURL     string
    ClientID     string
    ClientSecret string
    Scopes       string
}
```

- 仅支持 `client_credentials`
- 无 `AuthorizationURL`（授权端点）
- 无 `RedirectURL`（回调地址）
- 无 `refresh_token` 存储

### oauthAccessToken

```go
func (c *MCPClient) oauthAccessToken(ctx context.Context) (string, error) {
    // 仅支持 grant_type=client_credentials
    form.Set("grant_type", "client_credentials")
    // ...
    var tokenResp struct {
        AccessToken string `json:"access_token"`
        ExpiresIn   int    `json:"expires_in"`
    }
    // 不解析 refresh_token
}
```

- Token 过期后重新走 `client_credentials` 流程
- 不支持 `refresh_token` 刷新
- 不支持授权码模式

### 配置

```go
MCPOAuthTokenURL     string  // 仅 token endpoint
MCPOAuthClientID     string
MCPOAuthClientSecret string
MCPOAuthScopes       string
```

- 缺少 `AGENT_MCP_OAUTH_AUTH_URL`
- 缺少 `AGENT_MCP_OAUTH_REDIRECT_URL`

## 缺口清单

1. **授权码模式**：无法接入需要用户交互授权的 MCP 服务
2. **刷新令牌**：token 过期后只能重新申请，不支持 `refresh_token` 刷新
3. **令牌持久化**：`refresh_token` 应持久化到 SQLite，避免进程重启后丢失授权
4. **回调服务器**：授权码模式需要本地 HTTP 回调服务器接收授权码

## 本轮目标

在保持现有 `client_credentials` 模式不变的前提下，补齐：

1. **授权码模式**：支持 `grant_type=authorization_code`
2. **刷新令牌**：支持 `grant_type=refresh_token`
3. **令牌持久化**：`refresh_token` 持久化到 SQLite
4. **回调服务器**：启动时可选启动本地回调服务器

## 方案

### 1. 扩展 MCPOAuthConfig

```go
type MCPOAuthConfig struct {
    TokenURL      string
    AuthURL       string   // 新增：授权端点
    RedirectURL   string   // 新增：回调地址
    ClientID      string
    ClientSecret  string
    Scopes        string
    GrantType     string   // 新增：grant_type，默认 "client_credentials"，可选 "authorization_code"
}
```

### 2. 授权码流程

1. 启动时检测 `GrantType=authorization_code`
2. 启动本地回调服务器（默认 `localhost:9876/callback`）
3. 构造授权 URL，输出到日志/事件
4. 用户在浏览器中授权
5. 回调服务器接收授权码
6. 用授权码换取 access_token + refresh_token
7. 持久化 refresh_token 到 SQLite

### 3. 刷新令牌流程

1. `oauthAccessToken` 检测 token 过期
2. 如果有 `refresh_token`，用 `grant_type=refresh_token` 刷新
3. 刷新成功后更新缓存的 access_token
4. 刷新失败（如 refresh_token 已失效）则重新走授权码流程

### 4. 令牌持久化

在 `session_store.go` 新增 `oauth_tokens` 表：

```sql
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
```

### 5. 配置扩展

```bash
AGENT_MCP_OAUTH_GRANT_TYPE=authorization_code  # 默认 client_credentials
AGENT_MCP_OAUTH_AUTH_URL=https://github.com/login/oauth/authorize
AGENT_MCP_OAUTH_REDIRECT_URL=http://localhost:9876/callback
AGENT_MCP_OAUTH_CALLBACK_PORT=9876
```

## 本轮边界

- 不做 PKCE（Authorization Code + PKCE 适合无 client_secret 的 SPA/移动端，当前场景有 client_secret）
- 不做多 MCP 端点的独立 OAuth 配置（当前进程级单配置）
- 不做 OAuth token 的自动轮换策略
- 不做 CLI 交互式授权引导（仅输出授权 URL 到日志/事件）

## 风险

- 授权码模式需要用户在浏览器中操作，headless 环境下可能无法完成
- `refresh_token` 持久化到 SQLite 需要考虑加密存储（本轮暂不加密，标记为后续改进）
- 回调服务器端口可能被占用，需要可配置
