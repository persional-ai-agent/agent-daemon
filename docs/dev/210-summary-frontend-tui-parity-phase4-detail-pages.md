# Frontend 与 TUI 对齐总结（Phase 4：详情页与可点击交互）

## 本阶段完成

1. 新增后端详情接口：
   - `GET /v1/ui/sessions/{session_id}?offset&limit`
   - `GET /v1/ui/tools/{tool_name}/schema`
2. 前端 `sessions/tools` 页面从纯 JSON 列表升级为可点击详情页：
   - sessions：左侧会话列表，右侧消息与统计详情。
   - tools：左侧工具列表，右侧 schema 明细。
3. 补齐 API 回归测试，覆盖新增详情接口。

## 验证

- `go test ./...` 通过。
- `npm --prefix web run build` 通过。
