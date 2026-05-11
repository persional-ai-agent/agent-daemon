# 060 总结：CLI 会话列表与检索

## 变更摘要

新增 `agentd sessions list/search`，提供最小跨会话查看与检索入口。

## 新增能力

```bash
agentd sessions list
agentd sessions search hello
agentd sessions search -limit 50 -exclude session-id hello
```

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/store/session_store.go` | 增加 `ListRecentSessions` |
| `internal/store/session_store_test.go` | 增加最近会话列表测试 |
| `cmd/agentd/main.go` | 增加 `sessions list/search` CLI |
| `README.md` | 增加会话检索示例 |
| `docs/dev/README.md` | 增加 060 索引 |
| `docs/dev/060-*.md` | 新增 060 文档 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/store ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 手动验证：`sessions list -data-dir /tmp/agentd-sessions-home` 输出 JSON
- 手动验证：`sessions search -data-dir /tmp/agentd-sessions-home -limit 5 hello` 正常返回匹配消息
