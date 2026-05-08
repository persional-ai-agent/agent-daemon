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
