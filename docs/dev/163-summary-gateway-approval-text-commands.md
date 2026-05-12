# 163-summary-gateway-approval-text-commands

## 背景

目前 Gateway 已经具备 `pending_approval` 的底层能力，但用户仍需要模型继续走 `approval confirm` 工具链，缺少一个直接面向聊天平台用户的最小交互入口。Hermes 的理想形态是原生按钮流，本轮先补文本命令，把审批闭环推进到“用户可直接在聊天中处理”。

## 本次实现

- 新增 Gateway 文本命令：
  - `/approve <approval_id>`
  - `/deny <approval_id>`
- 当工具结果出现 `pending_approval` 时，Gateway 会主动发送提示：
  - `Pending approval: /approve <id> or /deny <id>`
- `sessionWorker` 会记住最近一次 `approval_id`
  - 因此 `/approve` 或 `/deny` 在省略参数时可复用最近待审批项
- `/help` 文案同步加入审批命令

## 设计取舍

### 1. 先补文本命令，不做按钮 UI

本轮目标是把审批能力真正落到 Gateway 用户可操作层，不依赖平台原生交互组件。按钮、卡片、交互回调属于下一层平台特定能力，暂不引入。

### 2. 直接复用已有 `approval confirm`

Gateway 命令本身不重新实现审批逻辑，而是直接调用现有 `approval` 工具：

- `action=confirm`
- `approval_id=<id>`
- `approve=true|false`

因此 CLI/HTTP/Gateway 三条路径复用同一套审批状态与执行语义。

## 验证

- `go test ./...`
- 代码路径验证：
  - Gateway slash 命令新增 `/approve` `/deny`
  - `pending_approval` 检测后会记录并提示 `approval_id`

说明：当前轮次没有真实平台在线会话，因此以编译和代码路径验证为主。

## 文档更新

- README 增加 Gateway 可直接处理 `pending_approval` 的说明
- 产品/开发总览从“缺审批按钮流”收口为“缺原生审批按钮流，但已有文本审批命令”

## 剩余差距

Gateway 主线仍未对齐的重点包括：

- 原生平台 slash UI / richer command UX
- 原生审批按钮流
- 更完整 token lock / 分布式协调
- 更多平台适配器
