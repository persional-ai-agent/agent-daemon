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
