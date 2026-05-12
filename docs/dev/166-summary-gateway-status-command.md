# 166-summary-gateway-status-command

## 背景

聊天侧已经有 `/queue`、`/approvals`、`/grant`、`/revoke`、`/approve`、`/deny`，但还缺少一个统一入口来快速查看当前 Gateway 会话概况。为减少用户在多条命令之间切换，本轮补一个最小 `/status` 文本命令。

## 本次实现

- 新增 Gateway 文本命令：
  - `/status`
- 返回的最小状态摘要包括：
  - `platform`
  - `session`
  - `queue`
  - `paired`
  - `running`
  - `last_approval_id`（若存在）
- `/help` 文案同步加入 `/status`

## 设计取舍

### 1. 先做会话级摘要

本轮不直接暴露完整 daemon/runtime 结构体，而是优先给聊天用户最常用的会话信息摘要。这样能保持命令输出简洁，避免把 CLI 级细节原样倾倒到聊天窗口。

### 2. 与 `/queue`、`/approvals` 互补

`/status` 负责总览；更细的审批明细仍由 `/approvals` 负责。避免单条消息过长，也便于后续升级成平台卡片 UI。

## 验证

- `go test ./...`
- 代码路径验证：
  - slash 命令新增 `/status`
  - `/help` 已包含 `/status`

## 文档更新

- README 增加 `/status` 说明
- 产品/开发总览更新为“Gateway 文本命令已具备最小状态总览 + 审批闭环”

## 剩余差距

Gateway 主线仍未对齐的重点包括：

- 原生平台 slash UI / richer command UX
- 原生审批按钮流
- 更完整 token lock / 分布式协调
- 更多平台适配器
