# 279 总结：Web Model / Provider 管理面补齐

## 背景

延续“先做功能”的原则，在 Web Dashboard 已具备基础管理、Cron 管理后，补齐模型/Provider 管理入口，让用户可以不通过 CLI 直接查看与切换模型配置。

## 完成内容

- `internal/api/server.go`
  - 新增 `GET /v1/ui/model`：当前 provider/model/base_url 与模型运行配置。
  - 新增 `GET /v1/ui/model/providers`：可用 provider 列表，包含内置 provider 与 provider 插件。
  - 新增 `POST /v1/ui/model/set`：写入 provider/model/base_url。
- `cmd/agentd/main.go`
  - 在 `runServe` 注入模型管理回调，复用 CLI 已有的 provider 发现、校验与保存逻辑。
- `web/src/lib/api.ts`
  - 新增模型管理 API client。
- `web/src/App.tsx`
  - 新增 `models` 页面，支持当前模型查看、provider 列表展示与模型切换。
- 文档同步
  - 更新 Web README、Frontend/TUI 用户与开发文档、产品/开发总览。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./internal/api ./cmd/agentd`
- `npm run test`
- `npm run build`

## 边界

本次补的是模型选择与 provider 列表管理面；OAuth 登录、API key 安全录入、provider profile、账号用量与 provider routing 仍是后续功能。
