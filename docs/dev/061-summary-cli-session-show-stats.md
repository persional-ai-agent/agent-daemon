# 061 总结：CLI 会话详情查看与统计

## 变更摘要

新增 `agentd sessions show/stats`，补齐最小会话查看与统计能力，便于排障与外部系统读取会话元数据。

## 新增能力

```bash
go run ./cmd/agentd sessions show your-session-id
go run ./cmd/agentd sessions show -offset 200 -limit 200 your-session-id
go run ./cmd/agentd sessions stats your-session-id
```

## 修改文件

| 文件 | 变更 |
|------|------|
| `cmd/agentd/main.go` | 增加 `sessions show/stats` |
| `internal/store/session_store_test.go` | 增加 `LoadMessagesPage` / `SessionStats` 测试 |
| `README.md` | 增加示例 |
| `docs/dev/README.md` | 增加 061 索引 |
| `docs/dev/061-*.md` | 新增 061 文档 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过（本地）

