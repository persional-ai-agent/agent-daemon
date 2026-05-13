# Frontend 与 TUI 对齐总结（Phase 3：流式会话与时间线）

## 本阶段完成

1. Chat 页面新增流式模式开关：
   - 支持 `POST /v1/chat/stream` 的 SSE 消费。
2. 新增事件时间线（Timeline）：
   - 展示 `session/user_message/turn_started/tool_started/tool_finished/completed/result` 等事件序列。
3. 在 `result` 事件自动提取 `final_response` 回填主输出区。

## 验证

- `npm --prefix web run build` 通过。
- `go test ./...` 通过（后端逻辑未回归）。
