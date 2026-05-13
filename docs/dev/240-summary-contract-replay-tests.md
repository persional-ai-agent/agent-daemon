# 契约回放测试（Replay）与 CI 报告

本轮新增运行时契约回放，覆盖“真实请求 -> 实际响应”的回归验证，避免仅靠静态文档和字段 diff 造成漏检。

## 主要改动

- 新增回放用例清单：`internal/api/testdata/replay/cases.json`
  - 覆盖 `/v1/ui/tools`、`/v1/chat`、`/v1/chat/cancel`
- 新增回放测试：`internal/api/contract_replay_test.go`
  - 读取回放请求样例并逐条执行 HTTP 调用
  - 校验 OpenAPI（路径/方法/响应码桶/required 顶层字段）
  - 校验既有快照（与 `internal/api/testdata/contracts/*.json` 对齐）
  - 输出回放报告 JSON
- Makefile 接入
  - 新增 `contract-replay`
  - `contract-check` 增加 replay 执行
- CI 接入
  - `contract-guard` 新增 Contract Replay 步骤
  - 上传 artifact：`contract-replay-report`（`artifacts/contract-replay.json`）
- 文档更新
  - `docs/api/contract-versioning.md` 增加 replay 报告与校验项说明

## 验证

- `make contract-check`
- `go test ./...`
