# 060 Plan：CLI 会话列表与检索

## 任务

1. 在 `internal/store` 增加最近会话列表查询。
2. 在 `cmd/agentd` 增加 `sessions list/search`。
3. 补单元测试覆盖列表排序。
4. 更新 README 与 docs 索引。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/store ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
