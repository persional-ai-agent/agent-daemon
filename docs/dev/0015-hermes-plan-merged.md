# 0015 hermes plan merged

## 模块

- `hermes`

## 类型

- `plan`

## 合并来源

- `0022-hermes-plan-merged.md`

## 合并内容

### 来源：`0022-hermes-plan-merged.md`

# 0022 hermes plan merged

## 模块

- `hermes`

## 类型

- `plan`

## 合并来源

- `0000-hermes-agent-go-port.md`
- `0001-hermes-gap-closure.md`
- `0052-hermes-feature-alignment.md`
- `0061-hermes-cron-alignment.md`
- `0062-hermes-toolsets-alignment.md`
- `0063-hermes-send-message-alignment.md`
- `0064-hermes-patch-tool-alignment.md`
- `0065-hermes-web-tools-alignment.md`
- `0066-hermes-clarify-tool-alignment.md`
- `0067-hermes-execute-code-alignment.md`

## 合并内容

### 来源：`0000-hermes-agent-go-port.md`

# 001 计划：Hermes Agent Go 版实施计划

## 目标

在 Go 中实现 Hermes 风格 Agent 的完整核心闭环，并同时提供 CLI 与 HTTP API。

## 实施步骤

1. 建立核心共享类型与模型客户端
验证：可向 OpenAI 兼容接口发送消息并解析响应

2. 建立工具注册中心与内置工具
验证：可输出 tool schema，并能按工具名 dispatch

3. 实现 Agent Loop
验证：模型返回 `tool_calls` 时，工具结果可回灌并继续多轮执行

4. 实现会话与记忆持久化
验证：可加载 session 历史，可执行 session_search，可写入 `MEMORY.md` / `USER.md`

5. 实现 CLI 与 HTTP API
验证：CLI 可交互调用；HTTP `/v1/chat` 可返回完整结果

6. 添加关键测试并跑通
验证：`go test ./...` 通过

7. 沉淀调研、设计、总结文档
验证：`docs/` 与 `docs/dev/README.md` 索引完整

### 来源：`0001-hermes-gap-closure.md`

# 002 计划：Hermes 核心闭环差异补齐

## 目标

补齐当前 Go 版与 Hermes 核心闭环之间的剩余关键差异，使 Agent 在跨请求、多轮运行和工具执行安全边界上达到“完整核心功能”状态。

## 实施步骤

1. 重构系统提示词装配
验证：无论是否存在历史消息，每次 `Run()` 都会携带 system message，且不会重复叠加多份 system message

2. 补齐持久记忆回灌
验证：`MEMORY.md` / `USER.md` 的内容会进入系统提示词，后续 session 可直接复用

3. 注入工作区规则
验证：工作目录存在 `AGENTS.md` 时，其内容会以受控方式进入系统提示词

4. 增加文件工具工作区路径约束
验证：`read_file` / `write_file` / `search_files` 只能访问 `Workdir` 内路径，越界访问返回明确错误

5. 增加 terminal 硬阻断护栏
验证：明显灾难性命令会被拒绝执行，正常命令保持兼容

6. 增加针对性测试并回归验证
验证：新增单元测试覆盖提示词装配、记忆回灌、路径约束、危险命令阻断，`go test ./...` 通过

## 模块影响

- `internal/agent`
- `internal/memory`
- `internal/tools`
- `cmd/agentd`
- `docs/`

## 取舍

- 先补“闭环缺口”，不在本次引入完整审批系统与上下文压缩，避免为了追求 1:1 复刻而显著扩大范围
- 安全侧优先实现硬阻断与工作区边界，后续再扩展到审批、URL 安全和更细粒度权限

### 来源：`0052-hermes-feature-alignment.md`

# 053 Plan：Hermes 功能对齐文档完善

## 目标

明确当前 Go 项目与 `/data/source/hermes-agent` 的功能对齐范围，并补齐总览文档中的差异说明。

## 变更范围

- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `README.md`
- `docs/dev/053-research-hermes-feature-alignment.md`
- `docs/dev/053-plan-hermes-feature-alignment.md`
- `docs/dev/053-summary-hermes-feature-alignment.md`
- `docs/dev/README.md`

不修改 Go 源码、不新增依赖、不改变运行行为。

## 执行步骤

1. 梳理 Hermes 和当前项目功能面。
   - 验证：Research 文档列出已对齐、最小覆盖、未覆盖能力。
2. 更新产品总览。
   - 验证：总览明确当前项目是 Hermes 核心 Agent daemon 子集，而非完整复刻。
3. 更新 README 入口说明。
   - 验证：仓库首页能直接看到对齐边界并链接到详细矩阵。
4. 更新开发总览。
   - 验证：开发文档包含模块级功能矩阵和后续补齐建议。
5. 更新需求索引和 Summary。
   - 验证：`docs/dev/README.md` 能追溯到 053 三份文档。

## 不做事项

- 不实现 Hermes 缺失功能。
- 不调整已有配置或源码。
- 不修改当前工作区中已有的非文档变更。

## 验证方式

- 查看 `git diff -- docs README.md`，确认只包含文档补齐。
- 人工复核对齐矩阵与本地源码、Hermes 文档一致。

## 三角色审视

- 高级产品：任务聚焦“分析与文档完善”，没有扩展到功能开发。
- 高级架构师：文档按产品/开发/需求沉淀分层，便于后续需求引用。
- 高级工程师：变更可回滚、无运行时风险，验证成本低。

### 来源：`0061-hermes-cron-alignment.md`

# 062 计划：Hermes Cron 最小对齐（interval/one-shot）

## 目标（可验证）

- `cronjob` 工具可用：`create/list/get/pause/resume/remove/trigger`。
- 开启 `AGENT_CRON_ENABLED=true` 后，调度器会周期性扫描 due job 并触发独立 session 的 agent run。
- cron job 与 run 结果可持久化在 SQLite（与 `sessions.db` 同库）。
- 文档对齐矩阵更新：Cron 从“未覆盖”更新为“部分对齐”，并写明边界。

## 实施步骤

1. **存储**
   - 新增 `cron_jobs`、`cron_runs` 表，复用现有 SQLite 连接。
2. **调度器**
   - ticker 扫描 due jobs；按并发度执行；对 interval/once 计算 next_run_at。
3. **工具**
   - 新增内置工具 `cronjob`，action 压缩 schema。
4. **集成**
   - `serve` 与 `chat` 模式按配置启动 scheduler。
5. **文档**
   - 更新 `README.md`、`docs/overview-product*.md` 与 `docs/dev/README.md` 索引。
6. **测试**
   - schedule 解析与 cron store CRUD 单测（无网络/无端口依赖）。

## 不在本次范围

- cron 表达式执行
- 平台投递与 origin 捕获
- prompt threat scanning
- `no_agent` 脚本作业、context_from 链式作业

### 来源：`0062-hermes-toolsets-alignment.md`

# 063 计划：Hermes Toolsets 最小对齐

## 目标（可验证）

- `tools.enabled_toolsets` / `AGENT_ENABLED_TOOLSETS` 可限制 registry 仅保留解析后的工具集合。
- `agentd toolsets list` 输出内置 toolsets。
- `agentd toolsets resolve core` 输出 core toolset 展开后的工具名列表。
- 文档对齐矩阵更新：toolsets 从“未覆盖”调整为“部分对齐”。

## 实施步骤

1. 新增 `internal/tools/toolsets.go`：toolset 定义 + includes 解析。
2. 新增配置项：`tools.enabled_toolsets`（env：`AGENT_ENABLED_TOOLSETS`）。
3. Engine 构建时应用 toolset 过滤（先 enabled，再 disabled）。
4. 新增 CLI：`agentd toolsets list|resolve`。
5. 更新文档与索引，补单测。

### 来源：`0063-hermes-send-message-alignment.md`

# 064 计划：Hermes send_message 最小对齐

## 目标（可验证）

- `send_message(action='list')` 返回当前进程已连接的 gateway adapters。
- `send_message(action='send', platform, chat_id, message)` 可投递文本消息。
- Gateway runner 会在 adapter connect/disconnect 时注册/注销 adapter。
- 文档对齐矩阵更新：Gateway/toolsets 标记调整，补 docs/dev 索引。

## 实施步骤

1. 解耦 adapter 接口到 `internal/platform`。
2. 新增运行时 adapter registry。
3. Gateway runner hook：connect 后 register，退出前 unregister。
4. 新增工具 `send_message` 并注册到 engine。
5. 更新 toolsets `messaging` + `core` includes。
6. 补单测与文档。

### 来源：`0064-hermes-patch-tool-alignment.md`

# 065 计划：Hermes patch 工具最小对齐

## 目标（可验证）

- 新增内置工具 `patch`，并纳入 `file` toolset。
- `patch` 受 `AGENT_WORKDIR` 限制，避免越权写文件。
- 单测覆盖单次替换与多匹配保护策略。

## 实施步骤

1. 内置工具注册 `patch`。
2. 实现替换逻辑（与 `skill_manage patch` 一致的最小语义）。
3. toolsets `file` 增加 `patch`。
4. 更新文档与索引。

### 来源：`0065-hermes-web-tools-alignment.md`

# 066 计划：Hermes web tools 最小对齐

## 目标（可验证）

- 新增内置工具：`web_search`、`web_extract`。
- `toolsets.web` 默认包含 `web_search/web_extract`（保留 `web_fetch` 兼容）。
- 单测覆盖：DDG 结果解析与 HTML->text 抽取基础行为。

## 实施步骤

1. 在 builtin tools 中注册并实现 `web_search/web_extract`。
2. 新增最小解析与清洗 helper。
3. 更新 toolsets/web。
4. 更新 docs 与 `docs/dev/README.md` 索引。

### 来源：`0066-hermes-clarify-tool-alignment.md`

# 067 计划：Hermes clarify 工具最小对齐

## 目标（可验证）

- 新增内置工具 `clarify`，并纳入 `toolsets.core`。
- `clarify` 对空 question 报错；对 options 做最小校验与清洗。
- 更新 docs/dev 索引与工具清单。

## 实施步骤

1. 在 builtin tools 中注册并实现 `clarify`。
2. 在 toolsets 中新增 `clarify` toolset，并让 core includes 它。
3. 更新文档与索引。

### 来源：`0067-hermes-execute-code-alignment.md`

# 068 计划：Hermes execute_code 最小对齐

## 目标（可验证）

- 新增工具 `execute_code`，可运行 python 片段并返回 stdout/stderr/exit code。
- 限制在 workdir 内执行，支持超时。
- 单测覆盖基础执行成功路径。

## 实施步骤

1. 新增 `internal/tools/execute_code.go` 并注册到 engine。
2. toolsets 增加 `code_execution`（默认不纳入 core）。
3. 更新 docs/dev 索引与工具清单。
