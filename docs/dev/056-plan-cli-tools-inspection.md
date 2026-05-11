# 056 Plan：CLI 工具查看最小实现

## 任务

1. 将 `agentd tools` 路由到 `runTools`。
   - 验证：无参数时仍列出工具名。
2. 增加 `tools list`。
   - 验证：输出与无参数保持一致。
3. 增加 `tools show tool_name`。
   - 验证：输出单个 `core.ToolSchema` JSON。
4. 增加 `tools schemas`。
   - 验证：输出完整 schema JSON 列表。
5. 更新 README、总览文档和需求索引。
   - 验证：文档列出新命令和边界。

## 边界

- 不新增工具。
- 不增加工具启停或 toolset 配置。
- 不改变 MCP discovery 行为。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 手动验证 `tools list/show/schemas`
