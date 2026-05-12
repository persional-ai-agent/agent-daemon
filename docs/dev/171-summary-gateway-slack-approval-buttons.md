# 171 总结：Gateway Slack 原生审批按钮最小闭环

## 1. 背景

在 Telegram 与 Discord 已补最小原生审批按钮后，Slack 侧仍只有文本审批命令：

- `/pending`
- `/approve`
- `/deny`

这会让 Slack 用户继续依赖手工输入命令，审批交互仍不够原生。

因此本轮补一个最小版本：仅增加 Slack block actions 审批按钮，不扩展为完整原生 slash command / modal 流。

## 2. 本轮实现

### 2.1 Slack 富文本发送支持审批按钮

在 `internal/gateway/platforms/slack.go`：

- 新增 `SendText(..., meta)` 实现
- 当 metadata 带有 `approval_id` 时，为消息附加一个 `ActionBlock`
- 其中包含两个按钮：
  - `Approve`
  - `Deny`

按钮 `action_id` 直接复用已有文本命令格式：

- `/approve <id>`
- `/deny <id>`

### 2.2 Socket Mode 交互事件回流统一命令处理

`SlackAdapter.Connect()` 现在除了 `events_api` 消息事件，还会处理 `interactive` 事件：

- 只接收 `block_actions`
- 取第一项 block action 的 `action_id`
- 若其内容是 slash 文本命令，则转成统一 `gateway.MessageEvent`
- 再复用既有 `sessionWorker.handleEvent()` slash command 分支

这样做保持了最小复杂度：

- 不新增审批状态机
- 不新增 Slack 专属确认协议
- 继续复用现有授权与 `approval` 工具链

### 2.3 交互确认

收到 `interactive` 事件后会先 `Ack`，避免 Slack 客户端报交互失败。

## 3. 结果

当前原生审批按钮覆盖变为：

- Telegram：最小原生命令菜单 + 原生审批按钮
- Discord：最小原生审批按钮
- Slack：最小原生审批按钮

审批原生化已经覆盖当前三大主流聊天平台适配器。

## 4. 验证

- `go test ./...`

## 5. 文档同步

已更新：

- `README.md`
- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `docs/dev/README.md`

## 6. 仍未覆盖

本轮没有补：

- Slack 原生命令注册 / 完整 slash command 清单
- Yuanbao 原生审批按钮
- 更完整原生 slash UI / modal / form 流
- 更完整 token lock / 分布式协调
