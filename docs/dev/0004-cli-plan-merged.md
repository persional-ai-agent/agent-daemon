# 0004 cli plan merged

## 模块

- `cli`

## 类型

- `plan`

## 合并来源

- `0005-cli-plan-merged.md`
- `0015-doctor-plan-merged.md`

## 合并内容

### 来源：`0005-cli-plan-merged.md`

# 0005 cli plan merged

## 模块

- `cli`

## 类型

- `plan`

## 合并来源

- `0053-cli-config-management.md`
- `0054-cli-model-management.md`
- `0055-cli-tools-inspection.md`
- `0056-cli-doctor.md`
- `0057-cli-gateway-management.md`
- `0059-cli-sessions.md`
- `0060-cli-session-show-stats.md`

## 合并内容

### 来源：`0053-cli-config-management.md`

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

### 来源：`0054-cli-model-management.md`

# 055 Plan：CLI 模型管理最小实现

## 任务

1. 增加 `agentd model show`。
   - 验证：根据 `config.Config` 输出当前 provider/model/base URL。
2. 增加 `agentd model providers`。
   - 验证：输出 `openai`、`anthropic`、`codex`。
3. 增加 `agentd model set`。
   - 验证：支持 `provider model` 与 `provider:model` 两种输入；写入正确 INI 键位。
4. 更新 README 与总览文档。
   - 验证：文档包含示例与边界说明。

## 边界

- 不做模型目录拉取。
- 不新增 provider。
- 不改变环境变量优先级。
- 不改变 `buildModelClient`。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 临时配置文件手动验证 `model set/show`。

### 来源：`0055-cli-tools-inspection.md`

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

### 来源：`0056-cli-doctor.md`

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

### 来源：`0057-cli-gateway-management.md`

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

### 来源：`0059-cli-sessions.md`

# 060 Plan：CLI 会话列表与检索

## 任务

1. 在 `internal/store` 增加最近会话列表查询。
2. 在 `cmd/agentd` 增加 `sessions list/search`。
3. 补单元测试覆盖列表排序。
4. 更新 README 与 docs 索引。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/store ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`

### 来源：`0060-cli-session-show-stats.md`

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
