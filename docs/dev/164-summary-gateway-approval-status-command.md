# 164-summary-gateway-approval-status-command

## 背景

上一轮已经补齐了 `/approve` 和 `/deny`，但聊天平台用户仍缺少一个“先查看当前授权状态”的入口。现有 `approval` 工具已经支持 `status`，因此本轮把它直接透出到 Gateway 文本命令层。

## 本次实现

- 新增 Gateway 文本命令：
  - `/approvals`
  - `/approval`
- 命令会直接调用现有 `approval(action=status)` 工具
- 返回内容包括：
  - 当前是否存在 session 级授权
  - 当前 pattern 级授权列表
  - 到期时间（若有）
- `/help` 文案同步补上 `/approvals`

## 设计取舍

### 1. 直接复用现有审批状态工具

不新增单独的 Gateway 状态存储，也不写新的审批查询协议，而是把现有 `approval status` 结果翻译成面向聊天用户的文本输出。

### 2. 先做文本状态，不做平台卡片 UI

与 `/approve` `/deny` 一样，本轮先补文本闭环，避免为不同平台分别实现卡片、按钮或原生 action callback。

## 验证

- `go test ./...`
- 代码路径验证：
  - slash 命令新增 `/approvals` `/approval`
  - `/help` 已包含审批状态入口

## 文档更新

- README 增加 `/approvals` 说明
- 产品/开发总览更新为“文本审批闭环包含查看 + 批准/拒绝”

## 剩余差距

Gateway 主线仍未对齐的重点包括：

- 原生平台 slash UI / richer command UX
- 原生审批按钮流
- 更完整 token lock / 分布式协调
- 更多平台适配器
