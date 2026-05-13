# 264-summary-ui-tui-fullscreen-timeline-and-runtime-toggle

本轮继续完善 CLI/TUI 差异项 1，在现有全屏看板基础上补齐“时间线可视化 + 运行时切换”。

## 变更

- `ui-tui/main.go`
  - 全屏模式新增 `timeline` 面板，展示最近对话轨迹（user/assistant/tool/result/error 摘要）。
  - 新增运行时命令：
    - `/fullscreen`：查看当前状态
    - `/fullscreen on|off`：运行时开关全屏看板
  - 启动首条消息、普通输入消息、发送失败场景均会记录到时间线。
  - 增加时间线容量控制（默认上限 2000，超限滚动清理）。

- `ui-tui/main_test.go`
  - 新增 `addChatLine` 截断与滚动上限测试。

- 文档更新
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./ui-tui -count=1`
- `go test ./...`

