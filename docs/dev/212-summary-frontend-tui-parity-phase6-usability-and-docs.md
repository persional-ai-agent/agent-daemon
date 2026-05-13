# Frontend 与 TUI 对齐总结（Phase 6：可用性增强与文档补齐）

## 本阶段完成

1. CLI 类 TUI 增强：
   - 新增 `/sessions`、`/stats`、`/show` 命令。
2. Web 可用性增强：
   - sessions 页支持分页切换与刷新。
   - tools 页支持筛选与当前选中标识。
   - gateway/config 页补充刷新按钮。
3. 文档补齐：
   - 新增使用文档：`docs/frontend-tui-user.md`
   - 新增开发文档：`docs/frontend-tui-dev.md`

## 验证

- `go test ./...` 通过。
- `npm --prefix web run build` 通过。
