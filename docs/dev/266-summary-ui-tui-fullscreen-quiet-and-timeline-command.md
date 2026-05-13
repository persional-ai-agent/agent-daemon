# 266-summary-ui-tui-fullscreen-quiet-and-timeline-command

本轮继续完善 CLI/TUI 差异项 1，重点提升全屏模式可读性与时间线可用性。

## 变更

- `ui-tui/main.go`
  - 全屏模式下事件输出改为“静默控制台打印 + 写入时间线”，避免刷屏破坏看板布局。
  - 新增 `/timeline [n]` 命令，在非全屏场景可查看最近对话时间线摘要。
  - 新增 `timelineSlice` 统一时间线裁剪逻辑。
  - `printEvent` 增加 `emit` 控制参数，支持“只记录不打印”。

- `ui-tui/main_test.go`
  - 新增 `timelineSlice` 行为测试。

- 文档
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

