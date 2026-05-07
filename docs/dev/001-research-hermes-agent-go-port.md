# 001 调研：Hermes Agent 架构与 Go 实现映射

## 任务背景

目标是调研 `/data/source/hermes-agent` 项目中的 Agent 实现思路，形成完整文档，并在 `/data/code/agent-daemon/` 中以 Go 实现其完整核心功能。

由于 Hermes 是完整生态系统，覆盖：

- CLI / Gateway / ACP 多入口
- 多 provider / 多 API mode
- 工具注册中心与 60+ 工具
- 会话状态、长期记忆、技能、上下文压缩
- 多平台消息网关、插件、MCP、RL 环境

因此本次调研先抽取其“Agent 核心闭环”和“最小完整运行集合”。

## Hermes 核心结论

### 1. 核心对象是 `AIAgent`

Hermes 的核心 orchestrator 位于 `run_agent.py` 的 `AIAgent`。

其职责包括：

- 构建系统提示词
- 选择 provider / API mode
- 发起模型调用
- 解析 `tool_calls`
- 顺序或并发执行工具
- 将工具结果追加回消息历史
- 持久化 session
- 管理重试、回退与中断

### 2. 统一消息模型是主轴

Hermes 内部统一使用 OpenAI 风格消息：

- `system`
- `user`
- `assistant`
- `tool`

这是它能兼容不同 provider，同时又统一工具调用逻辑的关键。

### 3. 工具系统采用“注册中心 + 动态 schema”

Hermes 的 `tools/registry.py` 和 `model_tools.py` 负责：

- 发现工具
- 注册 schema
- 检查工具可用性
- 统一分发工具调用
- 统一错误封装

这是 Go 版本必须保留的骨架。

### 4. Agent Loop 的本质是 while 循环

Hermes 的运行过程可概括为：

1. 组装消息与工具 schema
2. 调用模型
3. 若模型返回文本且无工具调用，则结束
4. 若返回 `tool_calls`，则执行工具
5. 把工具结果作为 `tool` 消息追加进历史
6. 继续下一轮模型调用
7. 达到最大轮次则停止

### 5. 状态是分层保存的

Hermes 并不把所有状态都混在一起，而是拆成：

- 会话历史：结构化、可检索
- 长期记忆：跨 session 持久化
- todo / 当前工作状态：Agent 局部状态

### 6. terminal 工具是 Hermes 的关键能力

Hermes 中 terminal 能力不仅是执行 shell，还包含：

- 前台/后台任务
- 危险命令审批
- 多后端环境
- 进程状态追踪

Go 版本首期抽取其中最核心的：

- 本地 Linux 前台命令
- 本地 Linux 后台命令
- 进程状态轮询与停止

## Go 版设计映射

### 保留项

- 统一消息模型
- Tool registry
- OpenAI 兼容 `tool_calls`
- Session persistence
- Memory persistence
- CLI 与 HTTP 双入口
- 前台/后台 terminal

### 延后项

- 多 API mode（Codex Responses、Anthropic Messages）
- MCP
- Skills
- Context compression
- Gateway 多平台适配
- delegate_task 并发子 Agent
- 审批系统与复杂安全护栏

## 结论

Hermes 最值得复用的不是它的 Python 代码细节，而是它的架构骨架：

- OpenAI 风格消息内核
- 工具注册中心
- 多轮 tool-calling loop
- 状态分层持久化
- 入口层与核心层解耦

Go 版已经按这个骨架实现，可继续向 Hermes 的外围能力扩展。
