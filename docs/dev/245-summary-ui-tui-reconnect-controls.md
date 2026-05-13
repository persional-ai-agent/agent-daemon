# ui-tui 重连状态可视化与人工恢复控制

本轮补齐了 ui-tui 在实时会话中的“可感知、可控、可恢复”能力。

## 主要改动

- 新增重连状态机字段（`ui-tui/main.go`）
  - `reconnectEnabled`
  - `reconnectState`（`connecting/resumed/degraded/failed`）
  - `timeoutAction`（`wait/reconnect/cancel`）
- 新增命令
  - `/reconnect status`
  - `/reconnect on|off`
  - `/reconnect now`
  - `/reconnect timeout wait|reconnect|cancel`
- `/status` 输出增强
  - 追加展示重连启用状态、当前状态、最大重连次数与超时策略
- `sendTurn` 重连逻辑增强
  - 支持超时策略分支：
    - `wait`：继续等待
    - `reconnect`：强制断开并立即重连
    - `cancel`：触发 `/v1/chat/cancel` 并结束本轮
  - 重连期间按 payload 去重，避免重复 assistant 事件
- 帮助文档更新
  - `printHelp()` 增加 `/reconnect*` 命令说明
- 回归测试
  - `ui-tui/main_test.go` 的重连用例调整为验证去重效果（重复 assistant 仅一次，result 仅一次）

## 验证

- `go test ./ui-tui -count=1`
- `make contract-check`
- `go test ./...`
