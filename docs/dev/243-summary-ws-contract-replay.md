# WS 实时链路契约化（/v1/chat/ws）

本轮一次性补齐 WebSocket 实时链路的契约、回放、覆盖与 CI 产物。

## 主要改动

- 新增 WS 机器可读契约
  - `docs/api/ws-chat-events.schema.json`
  - 定义关键事件最小字段：`session/user_message/turn_started/assistant_message/completed/result/error`
- 新增 WS 回放用例
  - `internal/api/testdata/replay/ws_cases.json`
  - 覆盖 `chat_ws_success`，要求事件序列包含关键类型
- 新增 WS 真回放测试
  - `internal/api/contract_replay_test.go` 增加 `TestContractWSReplay`
  - 真实握手 `/v1/chat/ws`，发送请求并逐条校验事件字段
  - 输出报告：`contract-ws-replay.json`
- 覆盖率门禁升级为 HTTP + WS 双通道
  - `scripts/contract_coverage/main.go` 新增 WS 事件覆盖统计
  - 门禁要求核心 HTTP 与 WS 事件覆盖同时满足 100%
- Make/CI 接入
  - `Makefile` 新增 `contract-ws-replay`，并纳入 `contract-check`
  - `.github/workflows/contract-guard.yml` 新增 WS replay 执行与 artifact 上传
- 文档更新
  - `docs/api/contract-versioning.md` 增加 WS replay 产物与校验说明

## 验证

- `make contract-check`
- `go test ./...`
