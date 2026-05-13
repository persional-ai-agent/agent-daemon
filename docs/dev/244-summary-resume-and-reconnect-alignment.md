# SSE/WS 恢复与重连一致性补全

本轮面向产品稳定性，补齐实时链路在重连、取消、错误下的行为一致性，并加上回放与客户端去重保护。

## 主要改动

- 后端事件语义对齐（`internal/api/server.go`）
  - 新增统一错误/取消载荷结构：
    - `status`
    - `error_code`
    - `error` / `error_detail`
    - `reason`（cancelled）
  - SSE 与 WS 都支持 `resume` 显式事件：
    - `type: resumed`
    - `turn_id`
    - `transport`（`sse`/`ws`）
  - `timeout/cancelled/internal_error` 通过统一映射输出
- 回放测试增强
  - SSE replay 增加恢复场景：`chat_stream_resume_success`
  - WS replay 增加：
    - 成功恢复场景（含 `resumed`）
    - 无效 JSON 错误场景（覆盖 `error` 事件）
  - WS 事件契约更新：`docs/api/ws-chat-events.schema.json`
- ui-tui 去重保护（`ui-tui/main.go`）
  - 重连/恢复期间按事件 payload 去重，避免重复输出和重复处理
  - 对应回归测试更新（`ui-tui/main_test.go`）
- 契约文档同步
  - `/v1/chat/stream` 请求补充 `turn_id/resume` 字段
  - `docs/api/contract-versioning.md` 增加恢复与错误字段对齐说明

## 验证

- `make contract-check` 通过
- `go test ./...` 通过
- 覆盖率门禁：
  - HTTP 核心 100%
  - WS 事件 100%
