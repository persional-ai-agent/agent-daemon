# 快速完善产品功能：纳入 Chat Stream 契约与回放

本轮将实时流式对话链路（`POST /v1/chat/stream`）纳入正式契约体系，避免只覆盖非流式接口。

## 主要改动

- OpenAPI 契约补齐
  - `docs/api/ui-chat-contract.openapi.yaml` 新增 `/v1/chat/stream`
  - 定义 `text/event-stream` 成功响应与 4XX 错误响应
- Replay 回放增强
  - `internal/api/testdata/replay/cases.json` 新增 `chat_stream_success`
  - `internal/api/contract_replay_test.go` 支持 SSE 用例：
    - 校验 `Content-Type: text/event-stream`
    - 校验关键事件标记（`event: session`、`event: result`）
- 覆盖率门禁升级
  - `scripts/contract_coverage/main.go` 将 `POST /v1/chat/stream` 纳入核心端点统计
  - 核心覆盖率继续要求 100%
- 规则文档更新
  - `docs/api/contract-versioning.md` 更新核心端点列表

## 验证

- `make contract-check`
- `go test ./...`
