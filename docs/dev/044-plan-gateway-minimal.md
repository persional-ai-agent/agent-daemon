# 044 实施计划：多平台消息网关最小落地

## 实现步骤

### 1. 扩展配置 → 验证：`go build ./...` 通过
- 在 `Config` 中新增 `GatewayEnabled`、`TelegramToken`、`TelegramAllowed`
- 环境变量：`AGENT_GATEWAY_ENABLED`、`AGENT_TELEGRAM_BOT_TOKEN`、`AGENT_TELEGRAM_ALLOWED_USERS`

### 2. 创建 `internal/gateway/adapter.go` → 验证：编译通过
- 定义 `PlatformAdapter` 接口：`Name()`/`Connect()`/`Disconnect()`/`Send()`/`EditMessage()`/`SendTyping()`/`OnMessage()`
- 定义 `MessageEvent` 归一化入站消息类型
- 定义 `SendResult` 发送结果类型
- 定义 `MessageHandler` 回调类型

### 3. 创建 `internal/gateway/session.go` → 验证：编译通过
- `BuildSessionKey(platform, chatType, chatID)` 函数
- `SessionSource` 类型（platform, chat_id, chat_type, user_id, user_name）

### 4. 创建 `internal/gateway/auth.go` → 验证：编译通过
- `CheckAuthorization(allowedUsers, userID)` 函数
- 默认拒绝原则

### 5. 创建 `internal/gateway/events.go` → 验证：编译通过
- `StreamCollector` 结构：收集 `text_delta` 增量、合并最终内容
- `FinalContent()` 返回完整文本
- 为 Telegram 消息编辑提供进度缓冲

### 6. 创建 `internal/gateway/platforms/telegram.go` → 验证：`go build ./...` 通过
- `TelegramAdapter` 实现 `PlatformAdapter`
- 使用 `go-telegram-bot-api/v5` 的 `GetUpdatesChan`（长轮询）
- `Send`/`EditMessage`/`SendTyping` 实现
- `OnMessage` 注册回调、内部消息循环（单 goroutine 串行）

### 7. 创建 `internal/gateway/runner.go` → 验证：编译通过
- `GatewayRunner` 结构：适配器列表、store、engine、事件映射
- `Start()`/`Stop()` 生命周期管理
- `handleMessage()` 消息路由：sessionKey 构建 → 授权 → 加载历史 → 创建 EventSink → `engine.Run()` → 响应回传
- 流式事件映射：`text_delta` → 累积 + 限流编辑 → `completed` 最终化

### 8. 修改 `cmd/agentd/main.go` → 验证：`go build ./...` 通过
- `runServe` 中：gateway enabled 时创建 `GatewayRunner`，`Start()` + `Stop()` 通过 `context.WithCancel`
- `mustBuildEngine` 中：gateway 需要 `ModelUseStreaming = true`（流式编辑需要增量事件）
- 优雅关闭：SIGINT/SIGTERM 时 `Stop()` 所有适配器

### 9. 写入测试 → 验证：`go test ./...`
- `internal/gateway/platforms/telegram_test.go`：Mock Telegram Update/Message、验证 sessionKey 构建和消息路由
- `internal/gateway/events_test.go`：StreamCollector 增量累积正确性

### 10. 完整流程验证
- 构建 `go build ./...`
- 测试 `go test ./...` 全部通过
- `go vet ./...` 无警告

## 文件变更清单

| 文件 | 动作 | 内容 |
|------|------|------|
| `internal/config/config.go` | 修改 | 新增 GatewayEnabled/TelegramToken/TelegramAllowed |
| `internal/gateway/adapter.go` | 新建 | PlatformAdapter 接口 + MessageEvent/SendResult 类型 |
| `internal/gateway/session.go` | 新建 | BuildSessionKey + SessionSource |
| `internal/gateway/auth.go` | 新建 | CheckAuthorization |
| `internal/gateway/events.go` | 新建 | StreamCollector（增量事件累积） |
| `internal/gateway/runner.go` | 新建 | GatewayRunner 生命周期、消息路由、事件映射 |
| `internal/gateway/platforms/telegram.go` | 新建 | TelegramBotAdapter |
| `internal/gateway/platforms/telegram_test.go` | 新建 | Mock Telegram 测试 |
| `internal/gateway/events_test.go` | 新建 | StreamCollector 单元测试 |
| `cmd/agentd/main.go` | 修改 | serve 模式集成网关启动 |
| `go.mod` | 修改 | 新增 `go-telegram-bot-api/v5` 依赖 |

## 关键设计决策

1. **适配器单 goroutine 串行**：每个适配器的消息回调在单 goroutine 中串行调用，天然避免并发问题
2. **Agent 运行在独立 goroutine**：与 HTTP API 模式一致，Agent 运行在独立 goroutine，event 通过 channel 收集
3. **流式编辑限流 500ms**：避免 Telegram API 编辑频率过高触发限流
4. **默认拒绝授权**：空 `allowedUsers` = 拒绝所有用户，需要显式配置
5. **不引入新的外部持久化**：网关会话复用 `store.SessionStore`，不新增存储表
