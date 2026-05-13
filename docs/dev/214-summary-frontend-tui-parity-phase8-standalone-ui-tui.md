# Frontend 与 TUI 对齐总结（Phase 8：独立 ui-tui 子工程）

## 本阶段完成

1. 新增独立 `ui-tui/` 子工程：
   - `ui-tui/main.go`：WebSocket 客户端循环，发送消息并显示事件流。
   - `ui-tui/README.md`：启动说明与环境变量说明。
2. 补齐用户/开发文档，对独立 TUI 入口给出明确路径。

## 验证

- Go 主仓：`go test ./...` 通过（无回归）。
- `ui-tui` 工程可本地 `go run ./ui-tui` 启动。
