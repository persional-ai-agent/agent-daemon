# 173 总结：Gateway Discord 原生 slash 命令最小闭环

## 1. 背景

此前 Discord 已有最小原生审批按钮，但用户仍需要手工输入大多数管理命令：

- `/status`
- `/pending`
- `/approvals`
- `/queue`
- `/cancel`

这意味着 Discord 侧虽然已有原生审批交互，但“原生 slash UI”仍然缺位。

本轮目标是补一个最小可用的 Discord slash command 闭环，不追求完整参数面和所有管理命令，只覆盖最常用入口。

## 2. 本轮实现

### 2.1 启动时自动注册 Discord application commands

在 `internal/gateway/platforms/discord.go`：

- `Connect()` 成功后会调用 `registerCommands()`
- 使用 `ApplicationCommandBulkOverwrite` 注册全局命令

当前注册的最小命令集包括：

- `pair`
- `unpair`
- `cancel`
- `queue`
- `status`
- `pending`
- `approvals`
- `approve`
- `deny`
- `help`

### 2.2 slash command 事件回流统一命令内核

`InteractionCreate` 处理现在同时覆盖：

- `InteractionMessageComponent`
- `InteractionApplicationCommand`

其中 slash command 会被渲染成现有文本命令格式，例如：

- `pair(code=xxx)` -> `/pair xxx`
- `approve(id=...)` -> `/approve ...`
- `status` -> `/status`

这样可以继续复用：

- 既有授权检查
- 既有 queue/cancel/status/approval 逻辑
- 既有审批工具链

### 2.3 交互确认

slash command 收到后会先回复一条 ephemeral 确认消息：

- `Accepted. Check the next bot reply.`

随后实际结果仍由现有 Gateway 回复链路输出。

## 3. 结果

Discord 侧现在具备：

- 原生审批按钮
- 原生 slash 命令菜单

这让当前项目的“原生 slash UI”从 Telegram 单平台推进到 Telegram + Discord 两个平台。

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

- Discord `grant` / `revoke` 的完整原生参数面
- Slack / Yuanbao 原生命令菜单
- 更完整原生表单 / modal 流
- 更完整 token lock / 分布式协调
