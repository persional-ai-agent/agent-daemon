# 044 总结：多平台消息网关最小落地

## 变更摘要

1. 新增 `internal/gateway/` 包，定义 `PlatformAdapter` 接口和 `GatewayRunner`
2. 实现 Telegram 首个平台适配器（`internal/gateway/platforms/telegram.go`）
3. 通过 `agentd serve --gateway` 在 HTTP API 进程内启动网关
4. 流式增量事件映射到 Telegram 消息编辑，实现实时输出

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/config/config.go` | 新增 `GatewayEnabled`/`TelegramToken`/`TelegramAllowed` 配置字段和环境变量 |
| `internal/gateway/adapter.go` | 新建：`PlatformAdapter` 接口、`MessageEvent`、`SendResult`、`MessageHandler` 类型 |
| `internal/gateway/runner.go` | 新建：`GatewayRunner`（适配器生命周期、消息路由、事件映射、流式编辑） |
| `internal/gateway/session.go` | 新建：`BuildSessionKey`、`SessionSource` |
| `internal/gateway/auth.go` | 新建：`CheckAuthorization`（白名单白名单，默认拒绝） |
| `internal/gateway/events.go` | 新建：`StreamCollector`（增量文本收集，500ms 限流编辑） |
| `internal/gateway/platforms/telegram.go` | 新建：`TelegramAdapter` 实现 `PlatformAdapter` |
| `cmd/agentd/main.go` | 修改：`runServe` 中集成网关启动，gateway 开启时强制 streaming |
| `internal/gateway/auth_test.go` | 新建：7 个授权检查测试 |
| `internal/gateway/session_test.go` | 新建：3 个会话键构建测试 |
| `internal/gateway/events_test.go` | 新建：3 个 StreamCollector 测试 |
| `internal/gateway/platforms/telegram_test.go` | 新建：3 个 Telegram 适配器测试 |
| `go.mod` / `go.sum` | 新增依赖 `go-telegram-bot-api/v5 v5.5.1` |

## 新增能力

### PlatformAdapter 接口

```go
type PlatformAdapter interface {
    Name() string
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
    Send(ctx context.Context, chatID, content, replyTo string) (SendResult, error)
    EditMessage(ctx context.Context, chatID, messageID, content string) error
    SendTyping(ctx context.Context, chatID string) error
    OnMessage(ctx context.Context, handler MessageHandler)
}
```

新增平台只需实现此接口并注册到 `GatewayRunner`。

### GatewayRunner 消息路由

```
入站消息 → 授权检查 → sessionKey 构建 → 加载历史
  → goroutine 中运行 engine.Run()
  → EventSink 捕获 model_stream_event.text_delta
  → StreamCollector 累积 + 500ms 限流
  → Telegram editMessageText 流式编辑
  → 最终 Send 完整响应
```

### Telegram 适配器

- 使用 `go-telegram-bot-api/v5` 的长轮询（`GetUpdatesChan`）
- 支持 DM、群组、supergroup 的 `chat_type` 识别
- 支持 `ReplyToMessage` 回复关联
- `MarkdownV2` 解析模式 + 内容自动转义
- `/command` 文本直接透传，由 Agent 自然处理

### 会话隔离

会话键格式：`agent:main:{platform}:{chat_type}:{chat_id}`

- 同一用户在不同聊天中的会话互不影响
- 同一聊天中的不同用户共享同一会话（群组模式）
- 与 HTTP API 的 session_id 命名空间兼容

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `AGENT_GATEWAY_ENABLED` | 启用网关 | `false` |
| `AGENT_TELEGRAM_BOT_TOKEN` | Telegram Bot Token | - |
| `AGENT_TELEGRAM_ALLOWED_USERS` | 授权用户 ID 逗号分隔 | - |

### 安全边界

- 空 `AGENT_TELEGRAM_ALLOWED_USERS` = 拒绝所有消息
- 未授权用户收到 `Access denied.` 回复
- Agent 执行仍受 approval/workdir 安全约束
- 会话键天然隔离不同用户的会话

## 与计划的偏差

无偏差。按计划全部实现。

## 测试结果

```
go test ./... -count=1
ok  cmd/agentd
ok  internal/agent
ok  internal/api
ok  internal/gateway
ok  internal/gateway/platforms
ok  internal/memory
ok  internal/model
ok  internal/store
ok  internal/tools
```

`go vet ./...` 无警告。

## 后续扩展建议

1. **Discord 适配器**：实现 `PlatformAdapter` 接口，使用 `discordgo`
2. **Slack 适配器**：实现 `PlatformAdapter` 接口，使用 Socket Mode
3. **插件注册表**：类似 Hermes `platform_registry`，支持第三方适配器注册
4. **交互式审批 UI**：Telegram 内联键盘按钮实现 `/approve`/`/deny` 交互
5. **频道目录**：缓存可访问的聊天列表，供 `send_message` 工具使用
6. **WebSocket**：双向实时通信替代 SSE

## 风险与已知限制

- Telegram `Message.MessageThreadId` 字段在 v5.5.1 中不可用，话题消息的 thread_id 暂不采集
- 消息内容使用简化版 Markdown 转义，复杂格式可能被破坏
- 网关与 HTTP API 共享进程，一个崩溃会影响另一个（已通过 goroutine 隔离缓解）
- 未实现 Hermes 的 `set_busy_queue` 忙线排队机制（当前直接并发处理）
