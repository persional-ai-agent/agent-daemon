# 002 调研：Hermes 核心闭环差异补齐

## 背景

`/data/code/agent-daemon` 已实现 Hermes 风格的基础 Agent Loop、工具注册中心、会话持久化、CLI/API 双入口与结构化事件流。

但将当前实现与 `/data/source/hermes-agent` 的核心源码对照后，仍存在几处会影响“闭环完整性”的关键差异。

## 源码级差异

### 1. 系统提示词没有跨请求持续生效

当前 `internal/agent/loop.go` 仅在 `existing` 为空时追加 system message。

而会话历史从 `internal/store/session_store.go` 读取时并不包含 system message，这会导致：

- 第一轮请求后，后续请求丢失系统提示词
- CLI / HTTP 多次调用同一 session 时行为不稳定
- 与 Hermes 持续重建系统提示词的方式不一致

这属于核心闭环缺口，必须补齐。

### 2. 持久记忆可写不可读，未回灌到后续推理

当前 `internal/memory/store.go` 只实现 `Manage()` 写入 `MEMORY.md` / `USER.md`。

但 `internal/agent/loop.go` 没有在运行前加载这些内容，因此：

- `memory` 工具写入的信息不会影响后续 session
- “长期记忆”只有存储层，没有推理层闭环

相比 Hermes 的 memory manager / prompt builder，这也是明显缺口。

### 3. 工作区规则未进入系统提示词

Hermes 会把 `AGENTS.md`、上下文文件与环境提示拼进系统提示词。

当前 Go 版 `internal/agent/prompt.go` 只有固定两行默认提示，无法把项目级约束传递给模型。这会降低仓库内执行的一致性，也与项目自身的 `AGENTS.md` 工作流不匹配。

### 4. 工具侧缺少基础安全护栏

Hermes 在核心工具层有：

- 路径安全约束
- 危险命令识别/拦截
- URL / skill / tool guardrails

当前 Go 版的 `read_file`、`write_file`、`search_files`、`terminal` 直接对输入执行：

- 文件工具可越过工作区访问任意路径
- terminal 缺少最基础的灾难性命令阻断

完整审批系统仍属后续扩展，但工作区边界与硬阻断护栏属于当前应补齐的核心安全基线。

## 范围判断

本次“补齐为止”的目标定义为：补齐 Hermes 核心 Agent 闭环所必需的缺口，而不是一次性复刻其外围生态。

本次纳入范围：

- 系统提示词跨请求稳定注入
- 持久记忆回灌
- 工作区规则注入
- 文件路径安全约束
- 危险命令硬阻断

暂不纳入本次范围：

- Context Compression
- Skills 系统
- MCP
- 多 provider API mode
- 多平台 Gateway
- 完整审批系统

## 结论

当前项目离 Hermes 的“完整核心闭环”只差最后一层运行时装配与工具护栏。

优先补齐提示词装配、记忆回灌和基础安全边界后，可认为 Go 版已经真正闭合以下链路：

- system prompt / workspace rules
- memory persistence / memory reuse
- session history / multi-turn continuation
- tool execution / workspace-safe operations
