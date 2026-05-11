# 054 Plan：CLI 配置管理最小实现

## 任务

1. `internal/config` 增加配置文件管理函数。
   - 验证：单元测试可写入、读取、列出配置。
2. `config.Load()` 支持 `AGENT_CONFIG_FILE`。
   - 验证：测试中设置环境变量后读取指定文件。
3. `cmd/agentd` 增加 `config list|get|set`。
   - 验证：构建通过，命令语义可通过测试或手动命令验证。
4. 更新 README 与需求文档。
   - 验证：文档包含命令示例和配置优先级说明。

## 边界

- 不改变已有环境变量优先级：环境变量 > 配置文件 > 内置默认值。
- 不迁移配置格式。
- 不新增运行时依赖。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/config`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`（如沙箱允许监听端口）
