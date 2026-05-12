# 165-summary-gateway-approval-manage-commands

## 背景

上一轮已经补齐了 `/approvals`、`/approve`、`/deny`，但文本审批闭环还差“直接在聊天里管理授权本身”。现有 `approval` 工具已经支持 `grant/revoke`，因此本轮继续把授权管理透出到 Gateway 命令层。

## 本次实现

- 新增 Gateway 文本命令：
  - `/grant [ttl_seconds]`
  - `/grant pattern <name> [ttl_seconds]`
  - `/revoke`
  - `/revoke pattern <name>`
- 命令直接复用 `approval` 工具：
  - `action=grant`
  - `action=revoke`
- `/help` 文案同步加入授权管理命令

## 设计取舍

### 1. 保持与现有 approval 工具语义一致

Gateway 不自己维护授权模型，而是把现有 session/pattern 两级授权语义原样透出，避免引入第二套权限状态机。

### 2. 先补文本管理，不做平台按钮/表单

本轮仍坚持最小可用范围，优先补文本命令闭环。按钮、卡片和原生 UI 交互仍属于后续平台特定实现。

## 验证

- `go test ./...`
- 代码路径验证：
  - slash 命令新增 `/grant` `/revoke`
  - `/help` 已包含授权管理入口

## 文档更新

- README 增加 `/grant` `/revoke` 说明
- 产品/开发总览更新为“文本审批闭环已包含查看 + 授权管理 + 批准/拒绝”

## 剩余差距

Gateway 主线仍未对齐的重点包括：

- 原生平台 slash UI / richer command UX
- 原生审批按钮流
- 更完整 token lock / 分布式协调
- 更多平台适配器
