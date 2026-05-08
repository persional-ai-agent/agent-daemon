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
