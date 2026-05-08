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
