# 058 Plan：CLI 网关管理最小实现

## 任务

1. 增加 `agentd gateway status [-file path] [-json]`。
   - 验证：输出 enabled、configured_platforms、supported_platforms。
2. 增加 `agentd gateway platforms`。
   - 验证：输出 telegram、discord、slack。
3. 增加 `agentd gateway enable|disable [-file path]`。
   - 验证：写入 `gateway.enabled=true/false`。
4. 更新 README、总览文档与需求索引。
   - 验证：文档说明 token 仍通过 config 管理。

## 边界

- 不启动 Gateway。
- 不检查平台 token 真实性。
- 不实现 pairing 或 setup wizard。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 手动验证 `gateway status/platforms/enable/disable`
