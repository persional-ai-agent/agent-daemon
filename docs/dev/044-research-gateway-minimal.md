# 044 调研：多平台消息网关最小可行架构

## 任务背景

产品设计明确标出「未完全覆盖：Hermes 的多平台网关」。当前 agent-daemon 仅支持 CLI 和 HTTP API 两种入口，需要新增多平台消息网关层，使同一 Agent Engine 可被 Telegram、Discord 等聊天平台访问。

## Hermes Gateway 核心结论

### 架构骨架

```
┌──────────────────────────────────────────┐
│            GatewayRunner                  │
│  ┌──────────┐  ┌──────────┐              │
│  │ Platform │  │ Platform │  ...         │
│  │ Adapter  │  │ Adapter  │              │
│  └────┬─────┘  └────┬─────┘              │
│       └──────┬──────┘                    │
│              ▼                           │
│       _handleMessage()                   │
│              │                           │
│     ┌────────┼────────┐                  │
│     ▼        ▼        ▼                  │
│  Command   AIAgent   Queue               │
│  dispatch  creation  sessions            │
└──────────────────────────────────────────┘
```

核心模式：**平台适配器 → 消息归一化 → Agent Engine 调用 → 响应回传平台**。

### 关键接口

1. **`BasePlatformAdapter`**：`connect()` / `disconnect()` / `send()` / `handle_message()` 为每个平台适配器的最小合约
2. **`MessageEvent`**：归一化后的入站消息（text, media_urls, reply_to_id 等）
3. **`SendResult`**：平台发送结果
4. **`GatewayRunner`**：管理适配器生命周期、消息路由、会话映射、授权检查

### 授权模型

多层逐级放行：全局允许 → 平台级允许列表 → DM 配对码 → 默认拒绝。

## 当前 agent-daemon 已有能力（可直接复用）

| 组件 | 状态 | 可复用性 |
|------|------|----------|
| `agent.Engine.Run()` | 已实现 | ✅ 网关直接调用 |
| `agent.Engine.EventSink` | 已实现 | ✅ 事件可推送至平台 |
| `store.SessionStore` | SQLite 已实现 | ✅ 会话持久化复用 |
| `internal/api/server.go` | SSE 流式 | ⚠️ 可参考事件消费模式 |
| `internal/config/config.go` | 环境变量 | ✅ 扩展平台配置 |
| `internal/core.AgentEvent` | 统一事件 | ✅ 映射到平台消息 |

## 推荐方案：最小可行网关（v1）

### 范围界定

**本期（v1）实现**：
- 统一适配器接口（`PlatformAdapter`）
- 网关核心（`GatewayRunner`）：适配器生命周期、消息路由、会话管理
- 1 个平台适配器：**Telegram**（Hermes 中最成熟、Go 生态良好）
- 基础授权：`AGENT_TELEGRAM_ALLOWED_USERS` 环境变量
- `agentd serve` 命令中可选启动网关（与 HTTP API 共存）
- 将 `model_stream_event` 增量映射到 Telegram 消息流式编辑

**后续扩展**：
- Discord、Slack、WhatsApp 等平台适配器
- 插件注册机制
- 频道目录
- 投递路由
- 生命周期钩子
- 代理模式

### 包结构

```
internal/gateway/
├── adapter.go          # PlatformAdapter 接口 + MessageEvent/SendResult 类型
├── runner.go           # GatewayRunner：生命周期、消息路由、会话管理
├── session.go          # 网关会话键构建、SessionSource
├── auth.go             # 授权检查
├── config.go           # 网关配置（从 config 扩展）
├── platforms/
│   └── telegram.go     # Telegram 适配器
└── events.go           # AgentEvent → 平台消息映射（增量流式编辑）
```

### 核心接口设计

```go
// PlatformAdapter 是所有平台适配器必须实现的合约
type PlatformAdapter interface {
    // Name 返回平台名称（telegram, discord, ...）
    Name() string
    // Connect 建立平台连接，失败返回 error
    Connect(ctx context.Context) error
    // Disconnect 断开连接，可传入超时 context
    Disconnect(ctx context.Context) error
    // Send 发送文本消息到指定聊天
    Send(ctx context.Context, chatID string, content string, replyTo string) (SendResult, error)
    // EditMessage 编辑已发送消息（流式场景）
    EditMessage(ctx context.Context, chatID string, messageID string, content string) error
    // SendTyping 发送输入中状态
    SendTyping(ctx context.Context, chatID string) error
    // OnMessage 注册消息回调（同一适配器单 goroutine 串行调用）
    OnMessage(ctx context.Context, handler MessageHandler)
}
```

### 会话键设计

复用 Hermes 模式：`agent:main:{platform}:{chat_type}:{chat_id}`

- DM: `agent:main:telegram:dm:123456`
- Group: `agent:main:telegram:group:-789012`
- 线程/子话题保留 `thread_id`，由 SessionSource 携带

### 与 Agent Engine 集成

```go
// GatewayRunner._handleMessage 的流程：
// 1. 构建 sessionKey
// 2. 授权检查
// 3. 加载历史消息
// 4. 构建 EventSink（映射事件到平台消息：流式编辑）
// 5. 调用 eng.Run(ctx, sessionKey, userMessage, systemPrompt, history)
// 6. 如果中断/取消，发送通知
```

### 流式事件映射

| AgentEvent.Type | Telegram 行为 |
|-----------------|---------------|
| `model_stream_event.text_delta` | 累积文本，`editMessageText`（限流 500ms 间隔） |
| `tool_started` | 可选：回复工具执行状态 |
| `tool_finished` | 可选：回复工具结果摘要 |
| `completed` | 最终化消息 |
| `error` / `cancelled` | 发送错误/取消通知 |
| `context_compacted` | 不映射（静默） |

### 配置扩展

```go
// 新增字段到 config.Config
type GatewayConfig struct {
    Enabled         bool
    TelegramToken   string
    TelegramAllowed string   // 用户 ID 逗号分隔，空 = 全部拒绝
}

// 环境变量：
// AGENT_GATEWAY_ENABLED=true
// AGENT_TELEGRAM_BOT_TOKEN=...
// AGENT_TELEGRAM_ALLOWED_USERS=123456,789012
```

### Go Telegram 库选择

推荐 `github.com/go-telegram-bot-api/telegram-bot-api/v5`：
- 社区活跃（5k+ stars）
- 纯 Go 实现
- 支持长轮询（`GetUpdatesChan`）
- 支持消息编辑（`EditMessageText`）
- 无额外 CGO 依赖

### 安全边界

1. Telegram 消息内容通过 `AgentEngine` 的安全管道（工具仍受 approval/workdir 约束）
2. 授权检查在每次消息处理前执行
3. 会话按用户+聊天隔离，用户只能访问自己的会话
4. 禁止跨平台会话访问

## 与 Hermes 的设计差异

| 方面 | Hermes | agent-daemon (本方案) |
|------|--------|----------------------|
| 异步模型 | asyncio | goroutines + channels |
| 适配器注册 | if/elif 链 + 动态 Plugin | 显式注册表 |
| 流式消费 | GatewayStreamConsumer | 复用 EventSink 回调（已在 HTTP 验证） |
| 进程管理 | 内置进程管理器 | 复用 `cmd/agentd` 统一入口 |
| 依赖注入 | 全局对象 | 构造函数注入 |

## 技术风险评估

| 风险 | 等级 | 缓解 |
|------|------|------|
| Telegram Bot API 限流 | 低 | 标准限流 ≤30 msg/s，远低于实际 |
| goroutine 泄漏 | 中 | 统一 context 取消树，关闭时等待 |
| 会话冲突（两个用户同一会话键） | 低 | 会话键天然隔离用户 |
| Go Telegram 库维护状态 | 低 | v5 为最新主要版本，活跃维护 |
| 网关崩溃影响 HTTP API | 中 | 网关与 HTTP 共享进程，需优雅 panic recovery |

## 结论

最小可行网关方案可行且风险可控。建议以 Telegram 为首个平台适配器，验证适配器接口 + 消息路由 + 流式编辑的核心路径，后续按需扩展其他平台。

**方案推荐**：
1. 创建 `internal/gateway/` 包，定义 `PlatformAdapter` 接口和 `GatewayRunner`
2. 实现 `internal/gateway/platforms/telegram.go` 作为首个适配器
3. 扩展 `internal/config/` 支持网关配置
4. 修改 `cmd/agentd/main.go` 的 `serve` 模式，可选启动网关
5. 不需要新增外部依赖之外的库
