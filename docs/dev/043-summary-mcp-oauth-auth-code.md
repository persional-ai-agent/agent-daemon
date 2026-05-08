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
