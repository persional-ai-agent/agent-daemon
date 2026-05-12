# 175 总结：Gateway Slack slash 命令入口最小闭环

## 1. 背景

Slack 此前已经补了最小原生审批按钮，但原生 slash UI 仍缺入口：

- 用户只能发普通消息或点审批按钮
- 无法通过 Slack slash command 直接进入已有 Gateway 命令链

与 Discord 不同，Slack slash command 通常需要在 Slack app 后台显式配置，不能像 Telegram/Discord 那样在运行时自动完整注册。因此本轮目标是补“命令入口支持”，而不是补完整的自动注册体系。

## 2. 本轮实现

### 2.1 Socket Mode 处理 slash command 事件

在 `internal/gateway/platforms/slack.go`：

- 新增 `socketmode.EventTypeSlashCommand` 处理分支
- 解析 `slack.SlashCommand`
- 先 `Ack`
- 再将事件转成统一 `gateway.MessageEvent`

### 2.2 复用既有命令内核

新增 `renderSlackSlashCommand()`：

- 如果 slash command 本身就是 `/status`、`/pending` 这类命令且 `Text` 为空，则直接返回命令名
- 如果 slash command 带参数，则拼成 `/<command> <text>`
- 如果 `Text` 本身已经是 slash 命令，则直接透传

这样 Slack app 只要配置一个或多个 slash command 入口，就能复用现有：

- `/status`
- `/pending`
- `/approvals`
- `/grant`
- `/revoke`
- `/approve`
- `/deny`

等文本命令逻辑。

## 3. 结果

Slack 侧现在具备：

- 原生审批按钮
- 原生 slash command 入口（需 Slack app 预配置）

这让 Slack 从“只有按钮 + 普通文本消息”推进到“有最小原生命令入口”。

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

- Slack slash command 的自动注册 / app manifest 管理
- 更完整原生 modal / form 流
- Yuanbao 原生命令菜单
- 更完整 token lock / 分布式协调
