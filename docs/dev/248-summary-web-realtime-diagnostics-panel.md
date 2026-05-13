# Web Chat：实时诊断面板与诊断包导出

本轮在 Web Chat 页补齐实时诊断可观测能力，覆盖传输状态、降级提示、重连统计与诊断包导出。

## 主要改动

- `web/src/lib/api.ts`
  - `StreamChatOptions` 新增 `onTransport` 回调，暴露当前实际传输通道（`ws|sse`）。
  - 新增 `createTransportFallbackEvent`，统一构造 `transport_fallback` 事件（`from/to/reason/at`）。
  - `streamChat` 在 WS 失败并降级 SSE 时：
    - 回调 `onTransport("sse")`
    - 发出 `transport_fallback` 事件，供上层 UI 诊断展示。
- `web/src/App.tsx`
  - 新增实时诊断状态：`activeTransport`、`lastTurnID`、`reconnectCount`、`lastErrorCode`、`fallbackHint`。
  - Chat 流式发送中接入 `onTransport` 与 `transport_fallback` 事件，展示降级原因与时间。
  - 新增“实时诊断”面板，实时展示核心运行字段。
  - 新增“导出诊断包”按钮，导出当前诊断上下文 JSON。
- `web/src/styles.css`
  - 新增降级提示条 `fallback-note` 样式。
  - 优化诊断面板 `pre` 容器可读性样式。
- `web/src/lib/api.test.ts`
  - 新增 `createTransportFallbackEvent` 契约测试，固定字段形状与枚举语义。

## 验证

- `npm --prefix web run test`
- `npm --prefix web run build`
- `make contract-check`
- `go test ./...`
