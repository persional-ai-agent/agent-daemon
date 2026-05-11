# 057 Plan：CLI 本地诊断最小实现

## 任务

1. 增加 `agentd doctor`。
   - 验证：输出文本检查结果，含 `ok/warn/error`。
2. 增加 `agentd doctor -json`。
   - 验证：输出结构化 JSON。
3. 增加诊断 helper。
   - 验证：测试覆盖缺 API key warning、坏 workdir error、Gateway 无 token warning。
4. 更新 README、总览文档和需求索引。
   - 验证：文档列出命令和本期边界。

## 边界

- 不做网络探测。
- 不调用 provider API。
- 不启动 MCP/Gateway。
- 不修改用户配置。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 手动验证 `doctor` 与 `doctor -json`
