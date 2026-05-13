# Frontend 与 TUI 对齐总结（Phase 7：独立 TUI 入口与实时事件轨迹）

## 本阶段完成

1. 新增 `agentd tui` 命令入口（独立于 `agentd chat`）。
2. 新增 `internal/cli/tui.go`：
   - 通过 `Engine.EventSink` 输出实时事件（turn/tool/completed/error）。
   - 保持与现有 chat/slash 命令兼容。
3. 更新使用文档与开发文档，明确 `tui` 模式用途与实现位置。

## 验证

- `go test ./...` 通过。
