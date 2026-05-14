# 0004 cli research merged

## 模块

- `cli`

## 类型

- `research`

## 合并来源

- `0005-cli-research-merged.md`
- `0015-doctor-research-merged.md`

## 合并内容

### 来源：`0005-cli-research-merged.md`

# 0005 cli research merged

## 模块

- `cli`

## 类型

- `research`

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

# 054 Research：CLI 配置管理最小对齐

## 背景

Hermes 提供 `hermes config set` 等配置管理入口，使用户可以不直接编辑配置文件完成模型、工具、网关等运行参数调整。当前 Go 项目已经支持从 `config/config.ini` 与环境变量加载配置，但缺少 CLI 读写入口。

## 目标

补齐最小 CLI 配置管理面：

- 查看当前配置文件中的键值。
- 读取单个 `section.key`。
- 写入单个 `section.key`。
- 保持环境变量优先级不变。

## 范围

本次只实现 INI 文件管理，不做完整 Hermes 配置系统：

- 不引入 YAML 配置。
- 不实现交互式 setup。
- 不做 provider/model 在线发现。
- 不实现工具启停配置。

## 方案

- 在 `internal/config` 增加小型 INI 管理函数：`ListConfigValues`、`ReadConfigValue`、`SaveConfigValue`。
- 增加 `AGENT_CONFIG_FILE` 作为配置文件路径覆盖入口；未设置时沿用 `config/config.ini` / `config.ini` 查找。
- 在 `cmd/agentd` 增加 `config list|get|set` 子命令。
- `config list` 默认隐藏包含 `api_key/token/secret/password` 的值，避免误打印凭据。

## 三角色审视

- 高级产品：解决用户配置入口缺失，不扩展到 setup wizard。
- 高级架构师：复用现有 INI 配置和 `ini.v1` 依赖，不改变运行时配置结构。
- 高级工程师：新增单元测试覆盖读写、列表排序、密钥脱敏与 `AGENT_CONFIG_FILE`。

### 来源：`0054-cli-model-management.md`

# 055 Research：CLI 模型管理最小对齐

## 背景

Hermes 提供 `hermes model` 入口用于查看和切换模型。当前 Go 项目已经具备 OpenAI、Anthropic、Codex 三类 provider 和 `agentd config set`，但缺少面向用户的模型切换命令。

## 目标

补齐最小模型管理面：

- 查看当前运行时 provider、model、base URL。
- 列出当前内置 provider。
- 写入 provider 与 provider 对应的 model 配置。

## 范围

- 只支持当前内置 provider：`openai`、`anthropic`、`codex`。
- 不做在线模型发现。
- 不处理 OAuth、凭据登录或 provider 插件。
- 不改变运行时模型调用逻辑。

## 推荐方案

- 在 `cmd/agentd` 增加 `model show|providers|set`。
- `model set openai gpt-4o-mini` 写入 `api.type=openai` 与 `api.model`。
- `model set anthropic claude-...` 写入 `api.type=anthropic` 与 `api.anthropic.model`。
- `model set codex gpt-5-codex` 写入 `api.type=codex` 与 `api.codex.model`。
- 可选 `-base-url` 写入对应 provider 的 `base_url`。

## 三角色审视

- 高级产品：模型切换是 Hermes CLI 体验中的核心高频操作，最小实现直接提升可用性。
- 高级架构师：复用已有 INI 管理能力，不扩展 provider 架构。
- 高级工程师：通过纯函数测试覆盖解析与配置键位，避免 CLI `os.Exit` 路径导致测试脆弱。

### 来源：`0055-cli-tools-inspection.md`

# 056 Research：CLI 工具查看最小对齐

## 背景

Hermes 提供 `hermes tools` 管理入口。当前 Go 项目已有 `agentd tools`，但只能列出工具名，无法查看模型实际接收的工具 schema。

## 目标

补齐最小工具查看能力：

- 保持 `agentd tools` 继续列出工具名。
- 新增 `agentd tools list` 显式列出工具名。
- 新增 `agentd tools show tool_name` 查看单个工具 schema。
- 新增 `agentd tools schemas` 输出完整工具 schema 列表。

## 范围

- 不实现工具启停配置。
- 不实现 toolset 解析。
- 不改变工具注册或 dispatch 行为。

## 推荐方案

在 `cmd/agentd` 中复用现有 `mustBuildEngine()` 与 `Registry.Schemas()`，只增加 CLI 输出层。`show/schemas` 使用 JSON pretty print，便于前端、SDK 或人工排查。

## 三角色审视

- 高级产品：解决工具可发现性不足，保留向后兼容。
- 高级架构师：不引入 toolset 或插件架构，避免超出当前阶段。
- 高级工程师：新增 helper 测试覆盖 schema 查找，运行时通过手动命令验证输出。

### 来源：`0056-cli-doctor.md`

# 057 Research：CLI 本地诊断最小对齐

## 背景

Hermes 提供 `hermes doctor` 用于诊断本地环境。当前 Go 项目已有配置、模型、工具查看命令，但缺少启动前诊断入口。

## 目标

实现最小 `agentd doctor`，检查本地可判定的问题：

- 配置文件路径与环境变量优先级提示。
- 工作目录存在性。
- 数据目录可创建且可写。
- 当前 provider/model 是否受支持。
- 当前 provider API key 是否为空。
- MCP transport 配置是否明显错误。
- Gateway 启用时是否至少配置一个平台 token。
- 内置工具是否成功注册。

## 范围

- 不发起网络请求。
- 不调用模型 API。
- 不启动 Gateway 或 MCP 进程。
- 不检查远端凭据是否有效。

## 推荐方案

在 `cmd/agentd` 中新增 `doctor` 子命令，输出 `ok/warn/error`。硬错误返回非零退出码；缺少 API key、Gateway 启用但没有 token 等情况作为 warning。

## 三角色审视

- 高级产品：诊断覆盖用户最常见的启动前问题，不把远端健康检查纳入本期。
- 高级架构师：只读检查，不改变配置或运行时状态；数据目录检查仅创建临时文件后删除。
- 高级工程师：通过 helper 测试覆盖缺 key、坏 workdir、Gateway token 缺失等分支。

### 来源：`0057-cli-gateway-management.md`

# 058 Research：CLI 网关管理最小对齐

## 背景

Hermes 提供 `hermes gateway` 入口管理消息网关。当前 Go 项目已有 Telegram、Discord、Slack 网关适配器和 `AGENT_GATEWAY_ENABLED` / `gateway.enabled` 配置，但缺少专用 CLI 管理入口。

## 目标

补齐最小网关管理面：

- 查看网关是否启用。
- 查看已配置 token 的平台。
- 列出支持的平台。
- 写入 `gateway.enabled=true/false`。

## 范围

- 不启动或停止运行中的进程。
- 不写入平台 token，避免专用命令处理 secret；token 继续通过 `agentd config set gateway.telegram.bot_token ...` 等方式配置。
- 不实现 Hermes 的 pairing、setup wizard、token lock 或平台级状态探测。

## 推荐方案

在 `cmd/agentd` 增加 `gateway status|platforms|enable|disable`。`status` 默认输出文本，支持 `-json`，并可通过 `-file` 指定配置文件。

## 三角色审视

- 高级产品：提供用户最需要的网关开关和状态查看，不伪装成完整 Gateway setup。
- 高级架构师：复用现有配置系统和 Gateway 支持平台，不启动外部连接。
- 高级工程师：helper 测试覆盖支持平台与已配置平台判断。

### 来源：`0059-cli-sessions.md`

# 060 Research：CLI 会话列表与检索

## 背景

Hermes 有跨会话检索能力，且通常提供直接的 CLI 使用面。当前 Go 项目已有 SQLite 会话存储和 `session_search` 工具，但缺少直接的命令行入口查看/检索历史。

## 目标

提供最小 CLI：

- 列出最近 session（按最新消息排序）。
- 按关键词搜索历史消息内容（优先使用 FTS5，缺失时回退 LIKE）。

## 范围

- 不做 LLM 摘要。
- 不做会话导出/删除。
- 仅本地 `sessions.db`。

## 方案

- `internal/store.SessionStore` 增加 `ListRecentSessions(limit)`。
- `cmd/agentd` 增加 `sessions list` / `sessions search` 子命令，默认输出 JSON。
- `sessions search` 支持 `-exclude session_id` 排除当前会话。

### 来源：`0060-cli-session-show-stats.md`

# 061 Research：CLI 会话详情查看与统计

## 背景

Hermes 的 session store/CLI 通常支持：

- 列出最近会话
- 搜索历史
- 查看某个 session 的消息（分页）
- 查看会话统计（消息数、时间范围、工具相关计数等）

Go 版 `agent-daemon` 目前已有 `sessions list/search`，且 SQLite `messages` 表里保存了足够信息来做最小 `show/stats`，但缺少 CLI 入口。

## 目标

补齐最小可用的 CLI：

- `agentd sessions show <session_id>`：分页查看消息。
- `agentd sessions stats <session_id>`：输出统计信息，便于排障与外部 UI 取数。

## 范围与非目标

- 不做会话删除/导出（Hermes 有 JSON snapshot 能力，这里先不做）。
- 不做基于 LLM 的摘要与跨会话语义检索（仍保持关键词检索）。
- 默认输出 JSON（与现有 `config/model/tools/gateway` 命令保持一致）。

## 方案

- 复用 `internal/store.SessionStore`：
  - `LoadMessagesPage(sessionID, offset, limit)`
  - `SessionStats(sessionID)`
- `cmd/agentd` 增加子命令：
  - `sessions show [-offset N] [-limit N] session_id`
  - `sessions stats session_id`
