# ui-tui 迁移总结（Go 实现）

## 变更

1. `ui-tui` 从 Node.js 实现迁移到 Go 实现。
2. 删除：
   - `ui-tui/package.json`
   - `ui-tui/src/index.mjs`
3. 新增：
   - `ui-tui/main.go`（WebSocket 交互式客户端）
4. 文档同步改为 `go run ./ui-tui` 启动方式。

## 验证

- `go test ./...` 通过。
- `go run ./ui-tui` 可启动（需后端服务可用）。
