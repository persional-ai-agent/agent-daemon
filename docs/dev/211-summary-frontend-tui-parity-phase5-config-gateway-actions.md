# Frontend 与 TUI 对齐总结（Phase 5：配置与网关动作）

## 本阶段完成

1. 新增后端可操作接口：
   - `POST /v1/ui/config/set`：写入指定 `section.key=value`
   - `POST /v1/ui/gateway/action`：网关启用/禁用动作（`enable|disable`）
2. 前端 `gateway/config` 页面升级为可操作页：
   - gateway 页支持“启用网关/禁用网关”按钮。
   - config 页支持键值写入并刷新快照。
3. 新增 API 回归测试覆盖动作接口。

## 验证

- `go test ./...` 通过。
- `npm --prefix web run build` 通过。
