# Frontend 与 TUI 对齐总结（Phase 2：UI API 联通）

## 本阶段完成

1. 新增后端 UI API：
   - `GET /v1/ui/tools`
   - `GET /v1/ui/sessions?limit=N`
   - `GET /v1/ui/config`
   - `GET /v1/ui/gateway/status`
2. `runServe` 注入 dashboard 所需快照回调（配置快照、网关状态快照）。
3. `web` 四个骨架页完成数据联通：
   - `sessions` 展示最近会话
   - `tools` 展示可用工具清单
   - `gateway` 展示网关状态
   - `config` 展示运行配置快照
4. 新增 API 回归测试：`internal/api/server_test.go`。

## 验证

- `go test ./...` 通过。
- `npm --prefix web run build` 通过。
