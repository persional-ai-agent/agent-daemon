# 050 总结：Slack 网关适配器

## 变更摘要

新增 Slack Socket Mode 适配器，实现 `PlatformAdapter` 接口。

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/gateway/platforms/slack.go` | 新建：`SlackAdapter` 实现 `PlatformAdapter`（Socket Mode WebSocket） |
| `internal/gateway/platforms/slack_test.go` | 新建：Slack 适配器测试 |
| `internal/config/config.go` | 新增 `SlackBotToken`/`SlackAppToken`/`SlackAllowed` |
| `cmd/agentd/main.go` | `buildGatewayAdapters` 注册 Slack；`allowedFor` 追加 Slack |
| `go.mod` / `go.sum` | 新增 `slack-go v0.23.0`；升级 `gorilla/websocket v1.5.3` |

## 环境变量

| 变量 | 说明 |
|------|------|
| `AGENT_SLACK_BOT_TOKEN` | Slack Bot Token (`xoxb-...`) |
| `AGENT_SLACK_APP_TOKEN` | Slack App Token (`xapp-...`) |
| `AGENT_SLACK_ALLOWED_USERS` | 授权用户 ID 逗号分隔 |

## 启动

```bash
export AGENT_GATEWAY_ENABLED=true
export AGENT_SLACK_BOT_TOKEN=xoxb-...
export AGENT_SLACK_APP_TOKEN=xapp-...
export AGENT_SLACK_ALLOWED_USERS=U123456
agentd serve  # Telegram + Discord + Slack 并行
```

## 测试结果

`go build ./...` ✅ | `go test ./...` 全部通过 ✅ | `go vet ./...` 无警告 ✅
