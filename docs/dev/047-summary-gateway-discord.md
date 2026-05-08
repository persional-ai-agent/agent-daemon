# 047 总结：Discord 网关适配器

## 变更摘要

1. 新增 `DiscordAdapter` 实现 `PlatformAdapter` 接口
2. GatewayRunner 支持按平台区分的授权白名单
3. 通过 `agentd serve` 与 Telegram 适配器并行运行

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/gateway/platforms/discord.go` | 新建：`DiscordAdapter` 实现 `PlatformAdapter`（websocket + REST） |
| `internal/gateway/platforms/discord_test.go` | 新建：Discord 适配器测试 |
| `internal/gateway/runner.go` | `allowedUsers` 改为 `allowedFor func(platform) string` 支持按平台授权 |
| `internal/config/config.go` | 新增 `DiscordToken`/`DiscordAllowed` 及环境变量 |
| `cmd/agentd/main.go` | `buildGatewayAdapters` 注册 Discord；`NewRunner` 按平台返回白名单 |
| `go.mod` / `go.sum` | 新增 `discordgo v0.29.0` + `gorilla/websocket` + `golang.org/x/crypto` |

## 新增能力

### Discord 适配器

- 使用 `discordgo` 库的 Gateway WebSocket 连接
- 支持 DM 和服务器频道（`chat_type: dm/group`）
- 支持回复引用（`ReferencedMessage`）
- 支持线程（`Thread`）
- 消息内容使用 `ContentWithMentionsReplaced()` 解析提及
- 2000 字符截断（Discord 消息限制）
- 流式编辑和输入中状态

### 按平台授权

```go
runner := gateway.NewRunner(adapters, eng, func(platform string) string {
    switch platform {
    case "telegram":
        return cfg.TelegramAllowed
    case "discord":
        return cfg.DiscordAllowed
    }
    return ""
})
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `AGENT_DISCORD_BOT_TOKEN` | Discord Bot Token |
| `AGENT_DISCORD_ALLOWED_USERS` | 授权用户 ID 逗号分隔 |

### 启动示例

```bash
export AGENT_GATEWAY_ENABLED=true
export AGENT_TELEGRAM_BOT_TOKEN=123:abc
export AGENT_DISCORD_BOT_TOKEN=xxx.yyy.zzz
export AGENT_DISCORD_ALLOWED_USERS=123456789
agentd serve
```

## 测试结果

`go build ./...` ✅ | `go test ./...` 全部通过 ✅ | `go vet ./...` 无警告 ✅
