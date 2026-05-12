# 178 总结：CLI 新增 Discord gateway 命令清单导出命令

## 1. 背景

Slack 侧上一轮已经补了 `gateway manifest` 导出，但 Discord 侧仍缺少一个对外可见的配置清单出口：

- 代码里已经有原生 slash commands
- 运行时会自动 bulk overwrite 注册
- 但部署者看不到命令面、权限面和安装提示的结构化输出

因此本轮补一个最小的 Discord 导出能力，方便运维和部署前检查。

## 2. 本轮实现

### 2.1 `gateway manifest` 支持 `discord`

在 `cmd/agentd/main.go`：

- `agentd gateway manifest -platform discord -json`

现在 `manifest` 子命令支持：

- `slack`
- `discord`

### 2.2 复用现有命令定义

在 `internal/gateway/platforms/discord.go`：

- 将命令注册清单提升为导出函数 `DiscordApplicationCommands()`

CLI 导出直接复用这份定义，避免命令面出现两套来源。

### 2.3 导出内容

Discord 导出结果包含：

- `commands`：slash commands 与 option 描述
- `permissions`：建议 scopes
- `bot_permissions`：建议 bot 权限
- `install_url_hint`：OAuth2 安装 URL 模板
- `next_actions`：部署后续步骤

## 3. 结果

Gateway CLI 现在不只可导出 Slack manifest，也可以导出 Discord 命令清单，便于统一运维和部署检查。

## 4. 验证

- `go test ./...`
- `go run ./cmd/agentd gateway manifest -platform discord -json`

## 5. 文档同步

已更新：

- `README.md`
- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `docs/dev/README.md`

## 6. 仍未覆盖

本轮没有补：

- Discord 安装 URL 的自动生成（仍需外部填充 `APP_ID`）
- 更完整平台安装器 / 发布流
- Yuanbao 原生命令菜单
- 更完整 token lock / 分布式协调
