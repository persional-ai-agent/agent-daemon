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
