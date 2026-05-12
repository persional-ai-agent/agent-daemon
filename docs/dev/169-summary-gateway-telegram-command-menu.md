# 169 总结：Gateway Telegram 原生命令菜单最小闭环

## 1. 背景

上一轮已经补了 Telegram 原生审批按钮，但 Gateway 仍缺一块更基础的原生 slash UI：用户在 Telegram 客户端里看不到可选命令菜单，只能手工输入 `/status`、`/pending`、`/approve` 等命令。

对齐 Hermes 时，这会导致：

- 文本 slash command 已有，但平台原生发现性不足
- `/help` 可用，但入口仍偏“记忆型”

因此本轮先补 Telegram 的最小原生命令菜单，不引入新的命令协议，也不改动统一 Gateway 命令处理主链路。

## 2. 本轮实现

### 2.1 连接时自动注册 Telegram commands

在 `internal/gateway/platforms/telegram.go`：

- `Connect()` 成功后会调用 `registerCommands()`
- 通过 Telegram `setMyCommands` 注册最小命令集

当前同步的命令包括：

- `pair`
- `unpair`
- `cancel`
- `queue`
- `status`
- `pending`
- `approvals`
- `grant`
- `revoke`
- `approve`
- `deny`
- `help`

### 2.2 失败不阻断网关启动

命令菜单注册采用 best-effort：

- 注册失败只记日志
- 不影响 Telegram adapter 继续接收/发送消息

这样可以避免因为平台侧权限或 API 异常导致整个 Gateway 启动失败。

## 3. 结果

Telegram 侧现在具备两层原生交互增强：

- 原生命令菜单：帮助用户发现和选择 slash command
- 原生审批按钮：帮助用户直接处理待审批项

这让“原生平台 slash UI”从完全缺失收敛为“Telegram 最小落地，其他平台后补”。

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

- Discord / Slack / Yuanbao 原生命令菜单
- 更丰富的 Telegram command scopes / locale 分层
- 更完整原生 slash UI / 原生表单流
- 更完整 token lock / 分布式协调
