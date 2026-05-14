# 0032 session summary merged

## 模块

- `session`

## 类型

- `summary`

## 合并来源

- `0041-session-summary-merged.md`

## 合并内容

### 来源：`0041-session-summary-merged.md`

# 0041 session summary merged

## 模块

- `session`

## 类型

- `summary`

## 合并来源

- `0008-session-approval-state.md`
- `0078-session-search-exclude.md`

## 合并内容

### 来源：`0008-session-approval-state.md`

# 009 总结：会话级审批状态补齐结果

## 已完成

- 新增 `ApprovalStore`（内存态 + TTL）
- `ToolContext` 增加 `ApprovalStore` 传递
- `Engine` 与启动装配接入审批状态存储
- 新增 `approval` 工具，支持：
  - `status`
  - `grant`
  - `revoke`
- terminal 危险命令判定支持会话级审批状态复用
- hardline 命令继续保持不可放行

## 新增配置

- `AGENT_APPROVAL_TTL_SECONDS`：会话授权默认 TTL（秒）

## 验证

- `go test ./...` 通过

## 当前边界

本次为最小会话级状态补齐，仍未覆盖：

- 跨进程持久化
- 用户侧交互确认 UI
- 命令级细粒度审批策略

### 来源：`0078-session-search-exclude.md`

# 079 总结：session_search 支持排除/包含会话参数（对齐 Hermes 体验）

## 变更

`session_search` 新增可选参数：

- `exclude_session_id`：排除指定会话
- `include_current_session`：是否包含当前会话（默认 false，保持原行为：排除当前会话）

返回中会回显实际使用的 `exclude_session_id`，便于调试。
