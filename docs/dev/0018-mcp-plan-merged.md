# 0018 mcp plan merged

## 模块

- `mcp`

## 类型

- `plan`

## 合并来源

- `0026-mcp-plan-merged.md`

## 合并内容

### 来源：`0026-mcp-plan-merged.md`

# 0026 mcp plan merged

## 模块

- `mcp`

## 类型

- `plan`

## 合并来源

- `0006-mcp-minimal-bridge.md`
- `0011-mcp-stdio-bridge.md`
- `0012-mcp-oauth-client-credentials.md`
- `0013-mcp-call-streaming-compat.md`
- `0041-mcp-streaming-passthrough.md`
- `0042-mcp-oauth-auth-code.md`

## 合并内容

### 来源：`0006-mcp-minimal-bridge.md`

# 007 计划：MCP 最小接入骨架

## 目标

实现 MCP HTTP 最小桥接能力，使外部 MCP 工具可被本地 Agent 直接使用。

## 实施步骤

1. 新增 MCP client
验证：可请求 `/tools` 并解析工具定义

2. 新增动态注册流程
验证：启动时发现到的 MCP 工具可进入 `Registry`

3. 新增调用转发
验证：调用 MCP 工具时可向 `/call` 发送 `name + arguments + context`

4. 增加配置项
验证：`AGENT_MCP_ENDPOINT` 配置后可自动启用 MCP 发现

5. 增加测试并回归
验证：MCP 发现与调用转发测试通过，`go test ./...` 通过

### 来源：`0011-mcp-stdio-bridge.md`

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

### 来源：`0012-mcp-oauth-client-credentials.md`

# 013 计划：MCP OAuth（Client Credentials）最小补齐

## 目标

让 MCP HTTP 桥接支持 OAuth `client_credentials`，以接入需要 Bearer Token 的 MCP 服务。

## 实施步骤

1. 扩展 `MCPClient` OAuth 配置结构。
2. 增加 token 获取逻辑：
   - `grant_type=client_credentials`
   - Basic Auth 传 `client_id/client_secret`
3. 增加 token 缓存与到期前复用。
4. 在 HTTP `/tools` 与 `/call` 请求注入 Bearer Token。
5. 扩展配置项并在启动装配阶段注入 OAuth 配置。
6. 增加 MCP OAuth 测试并执行全量回归。

## 验证标准

- OAuth MCP 发现与调用均带认证头
- token 在有效期内被复用（非每次重复申请）
- `go test ./...` 通过

### 来源：`0013-mcp-call-streaming-compat.md`

# 014 计划：MCP `/call` 流式响应兼容

## 目标

让 MCP `/call` 在 `text/event-stream` 响应下也能返回稳定结果，并与现有非流式调用保持兼容。

## 实施步骤

1. 在 `mcpToolProxy.Call` 中识别 `Content-Type: text/event-stream`。
2. 新增 SSE 解析函数：
   - 按事件空行分隔
   - 聚合 `data:` 多行
   - 解析 JSON 事件并提取 `result/structuredContent/error`
3. 保留原非流式 JSON 处理逻辑，避免回归。
4. 新增测试覆盖 SSE `/call` 成功链路。
5. 运行 `go test ./...` 全量回归。

## 验证标准

- SSE MCP `/call` 能返回正确结果
- 原 HTTP JSON 模式无回归
- 全量测试通过

### 来源：`0041-mcp-streaming-passthrough.md`

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

### 来源：`0042-mcp-oauth-auth-code.md`

# 043 计划：MCP OAuth 授权码模式与刷新令牌

## 目标

在保持现有 `client_credentials` 模式不变的前提下，补齐 MCP OAuth 授权码模式与刷新令牌能力。

## 实施步骤

### 步骤 1：扩展 MCPOAuthConfig 与 MCPClient

修改 `internal/tools/mcp.go`：
- `MCPOAuthConfig` 新增 `AuthURL`、`RedirectURL`、`GrantType` 字段
- `MCPClient` 新增 `cachedRefreshToken` 字段
- 新增 `ConfigureOAuthAuthCode` 方法

验证：编译通过，现有测试不受影响

### 步骤 2：令牌持久化

修改 `internal/store/session_store.go`：
- 新增 `oauth_tokens` 表
- 新增 `SaveOAuthToken`、`LoadOAuthToken`、`DeleteOAuthToken` 方法

验证：持久化存取测试通过

### 步骤 3：刷新令牌逻辑

修改 `internal/tools/mcp.go` 的 `oauthAccessToken`：
- token 过期时优先使用 `refresh_token` 刷新
- 刷新成功后更新 `cachedAccessToken` 和 `cachedRefreshToken`
- 刷新失败时根据 `GrantType` 决定是否重新走授权码流程
- 解析 token 响应中的 `refresh_token` 字段

验证：刷新令牌测试通过

### 步骤 4：授权码流程

修改 `internal/tools/mcp.go`：
- 新增 `StartOAuthCallbackServer` 方法，启动本地 HTTP 回调服务器
- 新增 `ExchangeAuthCode` 方法，用授权码换取 token
- `oauthAccessToken` 在无缓存且 `GrantType=authorization_code` 时，从 SQLite 加载持久化 token

修改 `cmd/agentd/main.go`：
- 启动时检测 `GrantType=authorization_code`，启动回调服务器并输出授权 URL

验证：授权码流程端到端测试通过

### 步骤 5：配置扩展

修改 `internal/config/config.go`：
- 新增 `MCPOAuthGrantType`、`MCPOAuthAuthURL`、`MCPOAuthRedirectURL`、`MCPOAuthCallbackPort`

修改 `cmd/agentd/main.go`：
- 根据 `GrantType` 选择 `ConfigureOAuthClientCredentials` 或 `ConfigureOAuthAuthCode`

验证：配置加载正确

### 步骤 6：增加测试并回归

- 授权码模式端到端测试（httptest 模拟 OAuth 服务器）
- 刷新令牌测试
- 令牌持久化测试
- `go test ./...` 全量回归

## 模块影响

- `internal/tools/mcp.go`
- `internal/store/session_store.go`
- `internal/config/config.go`
- `cmd/agentd/main.go`

## 向后兼容

- `GrantType` 默认为 `client_credentials`，行为与之前完全一致
- 不配置新环境变量时，现有 MCP OAuth 行为不变
- `refresh_token` 对 `client_credentials` 模式同样生效（如果 OAuth 服务器返回了 refresh_token）
