# ui-tui：实时诊断能力对齐（对齐 Web）

本轮将 `ui-tui` 的流式会话可观测能力对齐到 Web 诊断面板能力，补齐诊断状态展示与诊断包导出。

## 主要改动

- `ui-tui/main.go`
  - 在 `appState` 新增诊断字段：
    - `activeTransport`、`lastTurnID`、`reconnectCount`
    - `lastErrorCode`、`lastErrorText`、`fallbackHint`、`diagUpdatedAt`
  - 增加诊断能力函数：
    - `diagnosticsSnapshot()`：输出实时诊断快照
    - `exportDiagnostics(path)`：导出诊断包（包含 `diagnostics/runtime_state/recent_events`）
  - `sendTurn` 增强：
    - 每轮初始化 `turn_id`、transport、重连计数
    - 发生重连时累计 `reconnectCount` 并记录 `fallbackHint`
    - 终止事件中补采集 `error_code/error`
  - 新增命令：
    - `/diag`：查看实时诊断
    - `/diag export <file>`：导出诊断包
  - 扩展 `/reconnect status` 输出，增加 `reconnect_count/fallback_hint/last_error_code`。
- `ui-tui/main_test.go`
  - 在 `TestSendTurnReconnect` 增加诊断字段断言（重连计数、fallback 提示、turn_id）。
  - 新增 `TestExportDiagnostics`，校验导出文件结构和关键字段。
- `ui-tui/README.md`
  - 补充 `/diag` 与 `/diag export` 用法与诊断字段说明。

## 验证

- `go test ./ui-tui`
- `go test ./...`
