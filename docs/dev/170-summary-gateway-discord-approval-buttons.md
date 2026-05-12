# 170 总结：Gateway Discord 原生审批按钮最小闭环

## 1. 背景

上一轮已经补了 Telegram 原生命令菜单与审批按钮，但 Discord 侧仍只有文本命令：

- `/pending`
- `/approve`
- `/deny`

这意味着 Discord 用户仍需要手工复制或输入审批命令，原生交互体验仍弱于 Hermes 的按钮式处理流。

因此本轮补一个最小版本：只给 Discord 增加原生 Approve / Deny 按钮，不引入原生命令注册，不重写既有审批链路。

## 2. 本轮实现

### 2.1 Discord 富文本发送支持审批按钮

在 `internal/gateway/platforms/discord.go`：

- 新增 `SendText(..., meta)` 实现
- 当 metadata 带有 `approval_id` 时，为消息附加 Discord message components：
  - `Approve`
  - `Deny`

按钮的 `custom_id` 直接复用已有文本命令格式：

- `/approve <id>`
- `/deny <id>`

### 2.2 组件点击回流统一命令内核

新增 `InteractionCreate` handler：

- 仅处理 `InteractionMessageComponent`
- 提取 `custom_id`
- 若内容是 `/approve ...` 或 `/deny ...` 这类 slash 文本，则转成统一 `gateway.MessageEvent`
- 继续走现有 `sessionWorker.handleEvent()` 的 slash command 分支

这样做的结果是：

- 不新增审批状态机
- 不新增额外协议
- 仍复用现有授权校验与 `approval` 工具调用链

### 2.3 交互确认

点击按钮后会先向 Discord 回一个 `DeferredMessageUpdate`，避免客户端出现“interaction failed”。

## 3. 结果

Gateway 审批原生化现在分平台推进：

- Telegram：最小原生命令菜单 + 原生审批按钮
- Discord：最小原生审批按钮
- 其他平台：仍走文本命令保底

这使“原生审批按钮流”从 Telegram 单平台推进到 Telegram + Discord 两个平台。

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

- Discord 原生命令注册 / 原生 slash command 清单
- Slack / Yuanbao 原生审批按钮
- 更完整原生 slash UI / 表单流
- 更完整 token lock / 分布式协调
