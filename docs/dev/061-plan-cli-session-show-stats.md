# 061 Plan：CLI 会话详情查看与统计

## 任务

1. `cmd/agentd` 增加 `sessions show/stats` 子命令与 usage。
2. `internal/store` 增加单元测试覆盖 `LoadMessagesPage` 与 `SessionStats` 的基础行为。
3. 更新 README 示例与 `docs/dev/README.md` 索引。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 手动：
  - `go run ./cmd/agentd sessions stats <session_id>`
  - `go run ./cmd/agentd sessions show -offset 0 -limit 50 <session_id>`

