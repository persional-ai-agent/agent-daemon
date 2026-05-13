# Web Chat：WS 主通道 + SSE 降级

本轮将 Web Chat 流式通道升级为 WS 主通道，并保留 SSE 降级，统一重连状态与错误语义。

## 主要改动

- `web/src/lib/api.ts`
  - `streamChat` 支持 `transport=ws|sse` 与 `fallbackToSSE`
  - 新增 WS 实现：
    - 发送 `session_id/message/turn_id/resume`
    - 状态回调：`connecting/resumed/degraded/failed`
    - 超时策略：`wait/reconnect/cancel`
  - 保留并复用 SSE 流式逻辑（降级）
  - 新增错误归一化 `normalizeAPIError`
  - 新增 `streamEventDedupeKey`
- `web/src/App.tsx`
  - Chat 页新增传输模式切换（WS/SSE）
  - 连接状态条显示 `streamStatus + transport`
  - 事件去重渲染，避免重连重复展示
  - 错误展示统一读取归一化结果
- `web/src/styles.css`
  - 优化控制区与状态条样式，适配多控件布局
- `web/src/lib/api.test.ts`
  - 新增/更新回归测试（错误归一化、去重键稳定性）
- `web/package.json`
  - 保持 `vitest` 测试入口可用

## 验证

- `npm --prefix web run test`
- `npm --prefix web run build`
- `make contract-check`
- `go test ./...`
