# 0014 gateway summary merged

## 模块

- `gateway`

## 类型

- `summary`

## 合并来源

- `0013-discord-summary-merged.md`
- `0017-feishu-summary-merged.md`
- `0021-gateway-summary-merged.md`
- `0045-spotify-summary-merged.md`
- `0060-yuanbao-summary-merged.md`

## 合并内容

### 来源：`0013-discord-summary-merged.md`

# 0013 discord summary merged

## 模块

- `discord`

## 类型

- `summary`

## 合并来源

- `0098-discord-admin-implemented.md`

## 合并内容

### 来源：`0098-discord-admin-implemented.md`

# 099 Summary - discord_admin 实现（Discord REST API）

## 变更

将 `discord_admin` 从占位工具升级为可用实现（需要 `DISCORD_BOT_TOKEN`）：

- `action=list_guilds`：列出 bot 所在 guild
- `action=list_channels guild_id=...`：列出 guild channels
- `action=create_text_channel guild_id=... name=... topic?=...`：创建文字频道
- `action=delete_channel channel_id=...`：删除频道

实现位置：

- `internal/tools/discord_admin.go`
- `internal/tools/builtin.go`：继续复用 `discordAdminParams()` 注册

### 来源：`0017-feishu-summary-merged.md`

# 0017 feishu summary merged

## 模块

- `feishu`

## 类型

- `summary`

## 合并来源

- `0100-feishu-tools-implemented.md`

## 合并内容

### 来源：`0100-feishu-tools-implemented.md`

# 101 Summary - 飞书（Feishu/Lark）doc/drive 工具实现（OpenAPI）

## 变更

将以下工具从占位升级为可用实现（需要 `FEISHU_APP_ID` + `FEISHU_APP_SECRET`；可选 `FEISHU_BASE_URL`，默认 `https://open.feishu.cn`）：

- `feishu_doc_read doc_token=...`：GET `/open-apis/docx/v1/documents/{doc_token}/raw_content`
- `feishu_drive_list_comments`：GET `/open-apis/drive/v1/files/{file_token}/comments`
- `feishu_drive_list_comment_replies`：GET `/open-apis/drive/v1/files/{file_token}/comments/{comment_id}/replies`
- `feishu_drive_reply_comment`：POST `/open-apis/drive/v1/files/{file_token}/comments/{comment_id}/replies`
- `feishu_drive_add_comment`：POST `/open-apis/drive/v1/files/{file_token}/new_comments`

鉴权：

- 使用 tenant access token：POST `/open-apis/auth/v3/tenant_access_token/internal/`（带缓存）

实现位置：

- `internal/tools/feishu.go`
- `internal/tools/builtin.go`：注册与 schema（沿用同名 params 函数）
- `internal/tools/toolsets.go`：toolset `feishu`

### 来源：`0021-gateway-summary-merged.md`

# 0021 gateway summary merged

## 模块

- `gateway`

## 类型

- `summary`

## 合并来源

- `0043-gateway-minimal.md`
- `0046-gateway-discord.md`
- `0049-gateway-slack.md`
- `0113-gateway-media-send-discord-slack.md`
- `0117-gateway-auto-deliver-media-final.md`
- `0122-gateway-queue-and-cancel.md`
- `0123-gateway-pairing.md`
- `0124-gateway-pairing-management.md`
- `0125-gateway-slow-response-hint.md`
- `0126-gateway-webhook-hooks.md`
- `0127-gateway-webhook-lifecycle.md`
- `0128-gateway-webhook-signing-retry.md`
- `0129-gateway-webhook-delivery-events.md`
- `0130-gateway-webhook-spool.md`
- `0131-gateway-webhook-event-id.md`
- `0132-gateway-webhook-spool-dedup.md`
- `0133-gateway-hook-spool-cli.md`
- `0134-gateway-hook-spool-replay-cli.md`
- `0135-gateway-hooks-ping-cli.md`
- `0136-gateway-hook-spool-rotation.md`
- `0137-gateway-hook-spool-rotated-cli.md`
- `0138-gateway-hook-spool-status-aggregate.md`
- `0139-gateway-hook-spool-replay-filters.md`
- `0140-gateway-hook-spool-export-prune.md`
- `0141-gateway-hook-spool-compact.md`
- `0142-gateway-hook-spool-stats-command.md`
- `0143-gateway-hook-spool-verify.md`
- `0144-gateway-hooks-doctor.md`
- `0145-gateway-hook-spool-import.md`
- `0146-gateway-hooks-doctor-and-verify.md`
- `0147-gateway-hook-spool-import-all.md`
- `0148-gateway-hooks-doctor-next-actions.md`
- `0149-gateway-hook-spool-import-filters.md`
- `0150-gateway-hooks-doctor-strict.md`
- `0160-gateway-single-instance-lock.md`
- `0161-gateway-token-lock.md`
- `0162-gateway-approval-text-commands.md`
- `0163-gateway-approval-status-command.md`
- `0164-gateway-approval-manage-commands.md`
- `0165-gateway-status-command.md`
- `0166-gateway-pending-command.md`
- `0167-gateway-telegram-approval-buttons.md`
- `0168-gateway-telegram-command-menu.md`
- `0169-gateway-discord-approval-buttons.md`
- `0170-gateway-slack-approval-buttons.md`
- `0171-gateway-yuanbao-approval-quick-replies.md`
- `0172-gateway-discord-slash-commands.md`
- `0173-gateway-discord-grant-revoke-slash.md`
- `0174-gateway-slack-slash-command-entrypoint.md`
- `0175-gateway-slack-generic-slash-forwarding.md`
- `0257-gateway-command-matrix-and-stale-lock-closure.md`

## 合并内容

### 来源：`0043-gateway-minimal.md`

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

### 来源：`0046-gateway-discord.md`

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

### 来源：`0049-gateway-slack.md`

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

### 来源：`0113-gateway-media-send-discord-slack.md`

# 114 - Summary: Gateway media delivery for Discord/Slack

## Goal

Align Hermes “MEDIA:” / attachment delivery experience so that `text_to_speech` (and other tools producing local artifacts) can be delivered through gateway adapters, not only as local files.

## What changed

- Implemented optional `platform.MediaSender` on gateway adapters:
  - Discord: `internal/gateway/platforms/discord.go` now supports sending a local file as an attachment with optional caption and reply reference.
  - Slack: `internal/gateway/platforms/slack.go` now supports uploading a local file (files.upload.v2 flow in slack-go) with optional initial comment and thread reply.
- `send_message` already supports `media_path` and `MEDIA:` prefix; with these adapter changes, `send_message` can now deliver local files on Discord/Slack when the gateway is connected.

## Usage

- Direct attachment:
  - `send_message(action="send", platform="discord", chat_id="<channel_id>", media_path="/tmp/a.mp3", message="caption")`
  - `send_message(action="send", platform="slack", chat_id="<channel_id>", media_path="/tmp/a.mp3", message="caption")`
- Hermes-style:
  - `send_message(..., message="MEDIA: /tmp/a.mp3")`

## Notes / limitations

- Slack upload uses `slack.UploadFileParameters` which requires a non-zero `FileSize`; we `stat()` the local file before upload.
- Discord attachment delivery requires the bot to have permissions to post messages and attach files in the target channel.
- Yuanbao media delivery is still pending (typically requires an upload/hosting flow rather than raw local file paths).

### 来源：`0117-gateway-auto-deliver-media-final.md`

# 118 - Summary: Gateway auto-delivery for final `MEDIA:` responses

## Goal

Align Hermes “artifact-first” UX: when the agent’s final response is a media artifact pointer (`MEDIA: <path>`), the gateway should deliver the file directly to the chat without requiring an explicit `send_message(media_path=...)` call.

## What changed

- `internal/gateway/runner.go` adds best-effort auto delivery:
  - If the final assistant content begins with `MEDIA:`, the runner extracts the path and tries to send it via the current adapter’s `platform.MediaSender`.
  - Safety checks:
    - Path must resolve under `engine.Workdir` or `/tmp`
    - File must exist and be a regular file
  - If delivery succeeds, the runner stops and does not send the literal `MEDIA:` text.
  - If delivery fails (no MediaSender / validation failed / send error), the runner falls back to sending the original final text.

## Notes

- This only triggers for the *final assistant content* (not streaming partials).
- Caption is currently empty; if a tool wants a caption it can still use `send_message(media_path=..., message=...)`.

### 来源：`0122-gateway-queue-and-cancel.md`

# 123 - Summary: Gateway per-session queue + `/cancel` slash command

## Goal

Improve gateway UX toward Hermes behavior by:

- preventing overlapping agent runs in the same chat/session
- allowing users to interrupt a running task with a minimal slash command

## What changed

- `internal/gateway/runner.go` now uses a per-session worker queue:
  - each `(platform, chat_type, chat_id)` session has a buffered queue (size 32)
  - events are processed sequentially to avoid concurrent runs in the same chat
  - when queue is full, the oldest event is dropped (best-effort backpressure)
- Minimal slash commands (only in gateway mode):
  - `/cancel` or `/stop`: cancels the currently running agent context for that session (if any)
  - `/queue`: reports current queue length
  - `/help`: prints supported commands

## Notes / limitations

- This is a minimal alignment point; Hermes has richer command routing and queue policies.
- Cancellation is cooperative (context cancel); tool calls already respect context where possible.

### 来源：`0123-gateway-pairing.md`

# 124 - Summary: Minimal gateway pairing via `/pair`

## Goal

Add a minimal “pairing” flow for gateway access so operators can bootstrap authorization without editing config files:

- users can send `/pair <code>` in chat to gain access
- paired user IDs persist across restarts (best-effort) in a local file

## What changed

- `internal/gateway/runner.go`:
  - Adds pairing store:
    - env: `AGENT_GATEWAY_PAIR_CODE` (shared pairing code)
    - file: `<workdir>/.agent-daemon/gateway_pairs.json`
  - Authorization logic:
    - normal allowlist (`gateway.<platform>.allowed_users`) still applies
    - if user is paired for that platform, access is granted even when allowlist is empty
  - Slash command:
    - `/pair <code>`: pairs the current user ID for the current platform when code matches
    - `/help` updated to include `/pair`

## Notes / limitations

- Pairing is platform-scoped (Telegram/Discord/Slack/Yuanbao user IDs are stored separately).
- This is a minimal alignment point; Hermes supports richer pairing/management flows.

### 来源：`0124-gateway-pairing-management.md`

# 125 - Summary: Gateway pairing management (`/unpair` + CLI revoke/list)

## Goal

Make the minimal pairing flow operable:

- allow users to unpair themselves from chat
- allow operators to list and revoke pairings via CLI without editing JSON manually

## What changed

- `internal/gateway/runner.go`:
  - Adds `/unpair` slash command to remove the current user from the pairing store.
  - Pairing data remains in `<workdir>/.agent-daemon/gateway_pairs.json`.
- `cmd/agentd/main.go`:
  - Adds `agentd gateway pairs list` to show current pairings.
  - Adds `agentd gateway pairs revoke -platform <p> -user <id>` to revoke a specific user id.

## Usage

- In chat:
  - `/pair <code>`
  - `/unpair`
- CLI:
  - `agentd gateway pairs list`
  - `agentd gateway pairs revoke -platform telegram -user <id>`

### 来源：`0125-gateway-slow-response-hint.md`

# 126 - Summary: Gateway slow-response hint message

## Goal

Align Hermes gateway UX by sending a one-time “still working” message when the agent hasn’t produced visible output for a while (useful for long tool runs or slow models).

## What changed

- `internal/gateway/runner.go`:
  - Adds a per-run slow-response notifier:
    - if no stream edits/messages were emitted for `N` seconds, sends a waiting message once and stops.
  - Env knobs:
    - `AGENT_GATEWAY_SLOW_RESPONSE_TIMEOUT_SECONDS` (default `120`; set `0` to disable)
    - `AGENT_GATEWAY_SLOW_RESPONSE_MESSAGE` (default: `任务有点复杂，正在努力处理中，请耐心等待...`)

## Notes

- This is best-effort and does not replace proper progress events; it only improves user feedback during quiet periods.

### 来源：`0126-gateway-webhook-hooks.md`

# 127 - Summary: Gateway webhook hooks (best-effort)

## Goal

Provide a minimal “hooks” mechanism for gateway observability / integrations (Hermes has hooks/delivery concepts). This lets external systems receive notifications when a gateway session completes a run.

## What changed

- `internal/gateway/runner.go`:
  - Adds optional webhook POST on completion:
    - env: `AGENT_GATEWAY_HOOK_URL`
    - env: `AGENT_GATEWAY_HOOK_TIMEOUT_SECONDS` (default 4)
  - Event emitted: `gateway.completed`
  - Payload is JSON:
    - `{ "type": "gateway.completed", "data": { ... } }`
    - `data` includes platform/chat/user/message/session_key/final/at

## Notes / limitations

- Best-effort fire-and-forget; hook failures do not affect chat delivery.
- Only completion events are emitted currently (can be extended to started/error/tool events later).

### 来源：`0127-gateway-webhook-lifecycle.md`

# 128 - Summary: Gateway webhook lifecycle events (started/failed + optional tool events)

## Goal

Extend the gateway webhook hooks beyond completion-only to provide minimal lifecycle observability, closer to Hermes “hooks/delivery” expectations.

## What changed

- `internal/gateway/runner.go`:
  - Always emits:
    - `gateway.started`
    - `gateway.completed`
    - `gateway.failed` (when run returns error)
  - Optional verbose tool events:
    - env: `AGENT_GATEWAY_HOOK_VERBOSE=true`
    - emits `gateway.tool_started` / `gateway.tool_finished` / `gateway.error`
  - Payloads are truncated to avoid oversized webhook bodies.

## Configuration

- `AGENT_GATEWAY_HOOK_URL` (required)
- `AGENT_GATEWAY_HOOK_TIMEOUT_SECONDS` (optional; default 4)
- `AGENT_GATEWAY_HOOK_VERBOSE=true` (optional)

### 来源：`0128-gateway-webhook-signing-retry.md`

# 129 - Summary: Gateway webhook signing + retries

## Goal

Harden gateway webhooks for production usage by adding:

- optional request signing (HMAC)
- best-effort retries with backoff

## What changed

- `internal/gateway/runner.go`:
  - Adds headers:
    - `X-Agent-Event`: event type
    - `X-Agent-Timestamp`: unix seconds
    - `X-Agent-Signature`: optional (when secret configured)
  - Optional signing:
    - env: `AGENT_GATEWAY_HOOK_SECRET`
    - signature: `hex(hmac_sha256(secret, ts + "." + body))`
  - Best-effort retries:
    - env: `AGENT_GATEWAY_HOOK_RETRIES` (default 2; total attempts = retries + 1)
    - env: `AGENT_GATEWAY_HOOK_BACKOFF_MS` (default 250; linear backoff per attempt)

## Notes

- Retries trigger on non-2xx responses or network errors.
- This remains fire-and-forget; hook failures do not affect gateway chat delivery.

### 来源：`0129-gateway-webhook-delivery-events.md`

# 130 - Summary: Gateway webhook delivery events (send/edit/media)

## Goal

Extend gateway hooks to cover “delivery” events so external systems can observe what was actually sent/edited, including media delivery.

## What changed

- `internal/gateway/runner.go`:
  - Adds delivery-level events when `AGENT_GATEWAY_HOOK_DELIVERY=true`:
    - `gateway.delivery.send` (text send)
    - `gateway.delivery.edit` (message edit)
    - `gateway.delivery.media` (media send via `platform.MediaSender`)
  - Delivery events include platform/chat/message id, success/error, and small metadata tags (`phase`, `turn`, `slash`).

## Configuration

- Requires `AGENT_GATEWAY_HOOK_URL`
- Enable delivery events: `AGENT_GATEWAY_HOOK_DELIVERY=true`

### 来源：`0130-gateway-webhook-spool.md`

# 131 - Summary: Gateway webhook spool (dead-letter + replay)

## Goal

Reduce webhook event loss on transient failures or restarts by adding an optional local spool:

- when hook delivery fails after retries, append the event to a JSONL spool file
- a background loop periodically retries sending spooled events and rewrites the file with remaining failures

## What changed

- `internal/gateway/runner.go`:
  - Env: `AGENT_GATEWAY_HOOK_SPOOL=true` enables spooling and replay loop.
  - Env: `AGENT_GATEWAY_HOOK_SPOOL_PATH` overrides spool path (default `<workdir>/.agent-daemon/gateway_hooks_spool.jsonl`)
  - Env: `AGENT_GATEWAY_HOOK_SPOOL_REPLAY_SECONDS` (default 10)
  - Env: `AGENT_GATEWAY_HOOK_SPOOL_MAX_LINES` (default 2000)

## Notes / limitations

- Best-effort: spool is local file, not an at-least-once durable queue with dedup/ordering guarantees.
- Replay is bounded per tick (caps work to avoid long blocks).

### 来源：`0131-gateway-webhook-event-id.md`

# 132 - Summary: Gateway webhook `event_id` for deduplication

## Goal

Make webhook consumers able to deduplicate and trace events reliably by adding a stable event identifier to every webhook envelope, including spooled events.

## What changed

- `internal/gateway/runner.go`:
  - Webhook envelope now includes:
    - `id` (UUID)
    - `type`
    - `at` (RFC3339Nano)
    - `data`
  - Adds header: `X-Agent-Event-Id`
  - Spool entries store the event id and replay preserves it.

## Notes

- Receivers should use `id` (or `X-Agent-Event-Id`) as the primary dedup key.

### 来源：`0132-gateway-webhook-spool-dedup.md`

# 133 - Summary: Webhook spool dedup by `event_id`

## Goal

Reduce spool growth and avoid re-sending duplicate webhook events by deduplicating using the webhook `event_id`.

## What changed

- `internal/gateway/runner.go`:
  - When spooling is enabled, keeps an in-memory “seen event_id” set (loaded from the spool file on startup).
  - Skips appending to spool if `event_id` has already been seen (default on).
  - During replay, also drops duplicate `event_id` entries while rewriting the spool file.

## Configuration

- `AGENT_GATEWAY_HOOK_SPOOL_DEDUP` (default `true`; set to `false` to disable)
- `AGENT_GATEWAY_HOOK_SPOOL_DEDUP_MAX` (default `5000`; max in-memory ids)

### 来源：`0133-gateway-hook-spool-cli.md`

# 134 - Summary: CLI for webhook spool inspection/clear

## Goal

Improve operability of gateway webhook spooling by adding CLI commands to inspect and clear the spool without editing files manually.

## What changed

- `cmd/agentd/main.go`:
  - Adds:
    - `agentd gateway hooks spool status [-workdir dir] [-path file]` (JSON output with size/count/mtime)
    - `agentd gateway hooks spool clear  [-workdir dir] [-path file]`

## Notes

- Replay is handled by the running gateway process when `AGENT_GATEWAY_HOOK_SPOOL=true`; the CLI currently focuses on inspection/clear.

### 来源：`0134-gateway-hook-spool-replay-cli.md`

# 135 - Summary: CLI `spool replay` for gateway webhooks

## Goal

Provide an operational “manual replay” for webhook spools so operators can force a retry without waiting for the gateway background replay loop.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool replay`:
    - Reads the JSONL spool file
    - Attempts to POST up to `-limit` events to the hook URL
    - Rewrites the spool file with remaining failures
  - Flags:
    - `-url` override (defaults to `AGENT_GATEWAY_HOOK_URL`)
    - `-secret` override (defaults to `AGENT_GATEWAY_HOOK_SECRET`)
    - `-limit` (default 200)
    - `-timeout` per-request seconds (default 4)

## Notes

- This is best-effort and does not replace a durable queue.

### 来源：`0135-gateway-hooks-ping-cli.md`

# 136 - Summary: CLI `gateway hooks ping` healthcheck

## Goal

Improve gateway webhook operability by providing a simple healthcheck command to validate:

- hook URL connectivity (HTTP POST)
- optional HMAC signing header generation (when secret configured)

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks ping`:
    - sends a `gateway.ping` event envelope to the configured hook URL
    - includes `X-Agent-Event`, `X-Agent-Event-Id`, `X-Agent-Timestamp`, and optional `X-Agent-Signature`
    - returns JSON with `success` and HTTP status code

## Usage

- `agentd gateway hooks ping` (uses env `AGENT_GATEWAY_HOOK_URL`)
- `agentd gateway hooks ping -url http://localhost:9000/hook -secret ...`

### 来源：`0136-gateway-hook-spool-rotation.md`

# 137 - Summary: Webhook spool rotation (size-based)

## Goal

Control disk usage of the webhook spool by rotating the JSONL spool file when it grows beyond a configured size.

## What changed

- `internal/gateway/runner.go`:
  - Adds rotation before appending and before replay:
    - `AGENT_GATEWAY_HOOK_SPOOL_MAX_BYTES` (default 5MB; set `0` to disable)
    - `AGENT_GATEWAY_HOOK_SPOOL_ROTATE_KEEP` (default 3; number of rotated files to keep)
  - Rotation renames:
    - `<spool>.YYYYMMDD_HHMMSS`
  - Best-effort deletes older rotated files beyond keep.

## Notes

- Rotation is a safety valve; it can drop events if spool grows too fast and the hook endpoint is consistently failing.

### 来源：`0137-gateway-hook-spool-rotated-cli.md`

# 138 - Summary: CLI for rotated spool files (`list` + `replay -all`)

## Goal

After enabling spool rotation, operators need to enumerate and replay rotated spool segments. Add CLI helpers to manage rotated spool files without manual file globs.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool list` to list rotated spool files plus the base spool file.
  - Extends `agentd gateway hooks spool replay` with `-all` to replay all spool segments (rotated files oldest-first, then base).

## Usage

- `agentd gateway hooks spool list`
- `agentd gateway hooks spool replay -all`

### 来源：`0138-gateway-hook-spool-status-aggregate.md`

# 139 - Summary: Aggregated spool status (`status -all`)

## Goal

Improve webhook spool observability by making `status` report backlog health across rotated files, not just the base spool file.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks spool status` adds `-all`.
  - `-all` aggregates over rotated files + base file and returns:
    - per-file stats (`exists`, `size_bytes`, `count`, `types`, `oldest_at`)
    - global totals (`total_count`, `total_size_bytes`, `types`, `oldest_at`, `oldest_age_seconds`)

## Usage

- `agentd gateway hooks spool status`
- `agentd gateway hooks spool status -all`

### 来源：`0139-gateway-hook-spool-replay-filters.md`

# 140 - Summary: Filtered webhook spool replay (`-type` / `-id`)

## Goal

Improve webhook spool replay operability by allowing targeted replay for specific event classes or a single event.

## What changed

- `cmd/agentd/main.go`:
  - Extends `agentd gateway hooks spool replay` with:
    - `-type <event_type>`: replay only matching event type
    - `-id <event_id>`: replay only matching event id
  - Non-matching entries are preserved in spool; only matching entries are attempted.

## Usage

- `agentd gateway hooks spool replay -type gateway.delivery.media`
- `agentd gateway hooks spool replay -id <event_id>`

### 来源：`0140-gateway-hook-spool-export-prune.md`

# 141 - Summary: Webhook spool `export` and `prune` commands

## Goal

Close the operational loop for webhook spool triage and cleanup by adding:

- targeted event export for incident analysis
- targeted event prune for controlled backlog cleanup

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool export`:
    - required: `-out <file>`
    - optional filters: `-type`, `-id`, `-before` (RFC3339/RFC3339Nano), `-all` (include rotated files)
  - Adds `agentd gateway hooks spool prune`:
    - optional filters: `-type`, `-id`, `-before`, `-all`
    - removes only matching entries; non-matching entries stay in spool

## Usage

- `agentd gateway hooks spool export -out /tmp/hook-export.jsonl -type gateway.delivery.media -all`
- `agentd gateway hooks spool prune -type gateway.delivery.media -before 2026-05-01T00:00:00Z -all`

### 来源：`0141-gateway-hook-spool-compact.md`

# 142 - Summary: Webhook spool `compact` command

## Goal

Provide a maintenance command to reduce spool size/noise by deduplicating and trimming entries while preserving the newest useful events.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool compact`
  - Behavior:
    - drops malformed lines
    - deduplicates by `event_id` (`hookSpoolEntry.ID`)
    - sorts by `created_at` (oldest -> newest)
    - keeps only the newest `-max-lines` entries per file
  - Supports `-all` to compact rotated spool files too.

## Usage

- `agentd gateway hooks spool compact`
- `agentd gateway hooks spool compact -all -max-lines 500`

### 来源：`0142-gateway-hook-spool-stats-command.md`

# 143 - Summary: Webhook spool `stats` command

## Goal

Add an explicit stats command for spool backlog inspection, complementary to `status`.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool stats`
  - Supports `-all` for aggregated stats across rotated files + base file.
  - Returns:
    - per-file count/size/type distribution/oldest timestamp
    - global totals and oldest event age

## Usage

- `agentd gateway hooks spool stats`
- `agentd gateway hooks spool stats -all`

### 来源：`0143-gateway-hook-spool-verify.md`

# 144 - Summary: Webhook spool `verify` command

## Goal

Add a quick integrity check for spool files so operators can detect malformed or incomplete lines before replay/export/prune operations.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool verify`
  - Supports `-all` to verify rotated files too.
  - Reports:
    - total/valid/invalid line counts
    - per-file invalid samples (up to 5)

### 来源：`0144-gateway-hooks-doctor.md`

# 145 - Summary: Gateway hooks `doctor` command

## Goal

Provide a single diagnostic command to check webhook-related configuration consistency and common risk conditions.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks doctor`
  - Reports:
    - resolved hook/spool settings
    - status (`ok`/`warn`/`error`)
    - issue list (invalid numeric settings, missing URL, oversized spool warning, etc.)

### 来源：`0145-gateway-hook-spool-import.md`

# 146 - Summary: Webhook spool `import` command

## Goal

Add a safe import workflow for webhook spool events so offline-exported JSONL events can be merged back for replay.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool import -in <file>`
  - Supports:
    - `-append=true|false` (default true)
    - `-path` to set target spool file
  - Deduplicates by `event_id` (`hookSpoolEntry.ID`) when appending.
  - Validates minimal shape (`type` and `body`) and skips malformed lines.

## Usage

- `agentd gateway hooks spool import -in /tmp/events.jsonl`
- `agentd gateway hooks spool import -in /tmp/events.jsonl -append=false`

### 来源：`0146-gateway-hooks-doctor-and-verify.md`

# 147 - Summary: Gateway hooks `doctor` + spool `verify`

## Goal

Provide first-class diagnostics for webhook operations:

- `hooks doctor` for config/env sanity checks
- `spool verify` for spool file integrity checks

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks doctor` reports resolved settings, status (`ok/warn/error`), and issue list.
  - `agentd gateway hooks spool verify` checks line-level validity and reports invalid samples, with `-all` support.

### 来源：`0147-gateway-hook-spool-import-all.md`

# 148 - Summary: `spool import -all` bulk import

## Goal

Support bulk import of multiple JSONL event files into target spool to simplify offline recovery workflows.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks spool import` adds `-all`.
  - `-all` scans candidate JSONL files (spool/hook-related naming) from input path context and imports in sorted order.
  - Keeps existing `event_id` dedup behavior when appending.

### 来源：`0148-gateway-hooks-doctor-next-actions.md`

# 149 - Summary: `hooks doctor` next actions

## Goal

Make diagnostics actionable by returning concrete remediation suggestions alongside detected issues.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks doctor` now outputs `next_actions` with suggested commands/config fixes for common warnings/errors.

### 来源：`0149-gateway-hook-spool-import-filters.md`

# 150 - Summary: `spool import` filter options

## Goal

Allow targeted import for spool recovery/migration instead of always importing all events.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks spool import` adds `-type` / `-id` / `-before`.
  - Import path now reuses existing spool filter semantics to select events by type, event id, and cutoff time.
  - Import result JSON adds `filter` payload for traceability.

### 来源：`0150-gateway-hooks-doctor-strict.md`

# 151 - Summary: `hooks doctor --strict`

## Goal

Make hook diagnostics composable in CI/script flows by returning non-zero exit code when health is not `ok`.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway hooks doctor` adds `-strict`.
  - When enabled, command exits with code `1` if computed status is not `ok`, while keeping JSON diagnosis output.

### 来源：`0160-gateway-single-instance-lock.md`

# 161-summary-gateway-single-instance-lock

## 背景

虽然前几轮已经补齐了 `gateway run/start/stop/restart/install/uninstall`，但 Gateway 仍缺少最小的多实例保护。同一个 `workdir` 下重复启动多个 gateway 进程，会导致同一平台消息被重复消费，因此本轮先补一个最小单实例锁，收口 token lock 的核心语义。

## 本次实现

- 新增 `gateway.lock` 文件，位置为 `<workdir>/.agent-daemon/gateway.lock`
- `agentd gateway run` 启动前会尝试获取非阻塞独占锁
- `agentd serve` 在启用 gateway 且存在可用 adapter 时，也会尝试获取同一把锁
- 如果锁已被其他进程持有：
  - `gateway run` 直接失败
  - `serve` 仅跳过 gateway 启动，不影响 HTTP 服务本身
- `agentd gateway status` 新增：
  - `locked`
  - `lock_pid`
  - `lock_path`
- `agentd gateway start` 在 fork 前会优先检查现有锁，避免无意义拉起子进程

## 设计取舍

### 1. 先补“同 workdir 单实例”

本轮不是完整分布式 token lock，只覆盖本机同一工作目录下的最小互斥。这样已经能消除最常见的重复消费问题，同时不需要引入外部存储或平台级协调。

### 2. `serve` 与 `gateway run` 共用一把锁

Hermes 语义上 Gateway 是同一类消费端，不应因为运行在 `serve` 内嵌模式还是独立 `gateway run` 模式而并发消费同一工作区消息。因此本轮将两条启动路径统一到同一锁文件。

## 验证

- `go test ./...`
- `go run ./cmd/agentd gateway status -json`
- 在当前环境中验证 `status` 输出新增 `locked/lock_pid/lock_path`

说明：由于烟测环境没有持续在线的真实平台凭证，本轮没有做双进程真实竞争测试；以锁路径写入、状态暴露、启动前检查和编译验证为主。

## 文档更新

- README 增加“Gateway 同一 workdir 单实例锁”说明
- 产品/开发总览从“完全缺 token lock”更新为“已具备最小单实例锁，仍缺更完整 token lock”

## 剩余差距

Gateway 主线剩余缺口继续收敛为：

- 原生平台 slash UI
- 审批按钮流
- 更完整 token lock / 跨实例协调
- 更多平台适配器

### 来源：`0161-gateway-token-lock.md`

# 162-summary-gateway-token-lock

## 背景

上一轮已经补了同 `workdir` 单实例锁，但这还不能阻止“两个不同工作区使用同一组平台凭证同时消费消息”。Hermes 的 `token lock` 本质是避免同一 bot/token 被多个进程并发消费，因此本轮继续补一个最小跨工作区凭证锁。

## 本次实现

- 新增基于平台凭证指纹的全局 `token lock`
- 锁文件路径：`$TMPDIR/agent-daemon-gateway-locks/<fingerprint>.lock`
- 指纹来源：
  - `telegram bot token`
  - `discord bot token`
  - `slack bot/app token`
  - `yuanbao token/app_id`
- `agentd gateway run` 启动时会同时获取：
  - `<workdir>/.agent-daemon/gateway.lock`
  - 凭证指纹 `token lock`
- `agentd serve` 内嵌 gateway 启动时也会走同样的双锁逻辑
- `agentd gateway start` 在 fork 之前会预检查 `token lock`
- `agentd gateway status` 新增：
  - `token_locked`
  - `token_lock_pid`
  - `token_lock_path`

## 设计取舍

### 1. 先做本机级跨工作区协调

本轮仍然不做分布式锁，也不接第三方存储。范围限定为“同一台机器上，不同工作区不能同时用同一套平台凭证消费”。

### 2. 用凭证指纹而不是 workdir

单实例锁解决的是“同一工作区重复启动”；`token lock` 解决的是“不同工作区复用同一平台身份”。两者语义不同，因此保留两把锁并行存在。

## 验证

- `go test ./...`
- `go run ./cmd/agentd gateway status -json`
- 验证输出新增 `token_locked/token_lock_pid/token_lock_path`

说明：当前环境没有真实在线平台凭证，本轮以锁路径计算、状态暴露、启动前预检查与编译验证为主，没有做真实双工作区竞争烟测。

## 文档更新

- README 说明从“同 workdir 单实例锁”更新为“同 workdir 锁 + 跨工作区 token lock”
- 产品/开发总览从“仍缺 token lock”收口为“已具备最小 token lock，仍缺更完整策略”

## 剩余差距

Gateway 主线剩余高价值缺口继续集中在：

- 原生平台 slash UI
- 审批按钮流
- 更完整 token lock 策略 / 分布式协调
- 更多平台适配器

### 来源：`0162-gateway-approval-text-commands.md`

# 163-summary-gateway-approval-text-commands

## 背景

目前 Gateway 已经具备 `pending_approval` 的底层能力，但用户仍需要模型继续走 `approval confirm` 工具链，缺少一个直接面向聊天平台用户的最小交互入口。Hermes 的理想形态是原生按钮流，本轮先补文本命令，把审批闭环推进到“用户可直接在聊天中处理”。

## 本次实现

- 新增 Gateway 文本命令：
  - `/approve <approval_id>`
  - `/deny <approval_id>`
- 当工具结果出现 `pending_approval` 时，Gateway 会主动发送提示：
  - `Pending approval: /approve <id> or /deny <id>`
- `sessionWorker` 会记住最近一次 `approval_id`
  - 因此 `/approve` 或 `/deny` 在省略参数时可复用最近待审批项
- `/help` 文案同步加入审批命令

## 设计取舍

### 1. 先补文本命令，不做按钮 UI

本轮目标是把审批能力真正落到 Gateway 用户可操作层，不依赖平台原生交互组件。按钮、卡片、交互回调属于下一层平台特定能力，暂不引入。

### 2. 直接复用已有 `approval confirm`

Gateway 命令本身不重新实现审批逻辑，而是直接调用现有 `approval` 工具：

- `action=confirm`
- `approval_id=<id>`
- `approve=true|false`

因此 CLI/HTTP/Gateway 三条路径复用同一套审批状态与执行语义。

## 验证

- `go test ./...`
- 代码路径验证：
  - Gateway slash 命令新增 `/approve` `/deny`
  - `pending_approval` 检测后会记录并提示 `approval_id`

说明：当前轮次没有真实平台在线会话，因此以编译和代码路径验证为主。

## 文档更新

- README 增加 Gateway 可直接处理 `pending_approval` 的说明
- 产品/开发总览从“缺审批按钮流”收口为“缺原生审批按钮流，但已有文本审批命令”

## 剩余差距

Gateway 主线仍未对齐的重点包括：

- 原生平台 slash UI / richer command UX
- 原生审批按钮流
- 更完整 token lock / 分布式协调
- 更多平台适配器

### 来源：`0163-gateway-approval-status-command.md`

# 164-summary-gateway-approval-status-command

## 背景

上一轮已经补齐了 `/approve` 和 `/deny`，但聊天平台用户仍缺少一个“先查看当前授权状态”的入口。现有 `approval` 工具已经支持 `status`，因此本轮把它直接透出到 Gateway 文本命令层。

## 本次实现

- 新增 Gateway 文本命令：
  - `/approvals`
  - `/approval`
- 命令会直接调用现有 `approval(action=status)` 工具
- 返回内容包括：
  - 当前是否存在 session 级授权
  - 当前 pattern 级授权列表
  - 到期时间（若有）
- `/help` 文案同步补上 `/approvals`

## 设计取舍

### 1. 直接复用现有审批状态工具

不新增单独的 Gateway 状态存储，也不写新的审批查询协议，而是把现有 `approval status` 结果翻译成面向聊天用户的文本输出。

### 2. 先做文本状态，不做平台卡片 UI

与 `/approve` `/deny` 一样，本轮先补文本闭环，避免为不同平台分别实现卡片、按钮或原生 action callback。

## 验证

- `go test ./...`
- 代码路径验证：
  - slash 命令新增 `/approvals` `/approval`
  - `/help` 已包含审批状态入口

## 文档更新

- README 增加 `/approvals` 说明
- 产品/开发总览更新为“文本审批闭环包含查看 + 批准/拒绝”

## 剩余差距

Gateway 主线仍未对齐的重点包括：

- 原生平台 slash UI / richer command UX
- 原生审批按钮流
- 更完整 token lock / 分布式协调
- 更多平台适配器

### 来源：`0164-gateway-approval-manage-commands.md`

# 165-summary-gateway-approval-manage-commands

## 背景

上一轮已经补齐了 `/approvals`、`/approve`、`/deny`，但文本审批闭环还差“直接在聊天里管理授权本身”。现有 `approval` 工具已经支持 `grant/revoke`，因此本轮继续把授权管理透出到 Gateway 命令层。

## 本次实现

- 新增 Gateway 文本命令：
  - `/grant [ttl_seconds]`
  - `/grant pattern <name> [ttl_seconds]`
  - `/revoke`
  - `/revoke pattern <name>`
- 命令直接复用 `approval` 工具：
  - `action=grant`
  - `action=revoke`
- `/help` 文案同步加入授权管理命令

## 设计取舍

### 1. 保持与现有 approval 工具语义一致

Gateway 不自己维护授权模型，而是把现有 session/pattern 两级授权语义原样透出，避免引入第二套权限状态机。

### 2. 先补文本管理，不做平台按钮/表单

本轮仍坚持最小可用范围，优先补文本命令闭环。按钮、卡片和原生 UI 交互仍属于后续平台特定实现。

## 验证

- `go test ./...`
- 代码路径验证：
  - slash 命令新增 `/grant` `/revoke`
  - `/help` 已包含授权管理入口

## 文档更新

- README 增加 `/grant` `/revoke` 说明
- 产品/开发总览更新为“文本审批闭环已包含查看 + 授权管理 + 批准/拒绝”

## 剩余差距

Gateway 主线仍未对齐的重点包括：

- 原生平台 slash UI / richer command UX
- 原生审批按钮流
- 更完整 token lock / 分布式协调
- 更多平台适配器

### 来源：`0165-gateway-status-command.md`

# 166-summary-gateway-status-command

## 背景

聊天侧已经有 `/queue`、`/approvals`、`/grant`、`/revoke`、`/approve`、`/deny`，但还缺少一个统一入口来快速查看当前 Gateway 会话概况。为减少用户在多条命令之间切换，本轮补一个最小 `/status` 文本命令。

## 本次实现

- 新增 Gateway 文本命令：
  - `/status`
- 返回的最小状态摘要包括：
  - `platform`
  - `session`
  - `queue`
  - `paired`
  - `running`
  - `last_approval_id`（若存在）
- `/help` 文案同步加入 `/status`

## 设计取舍

### 1. 先做会话级摘要

本轮不直接暴露完整 daemon/runtime 结构体，而是优先给聊天用户最常用的会话信息摘要。这样能保持命令输出简洁，避免把 CLI 级细节原样倾倒到聊天窗口。

### 2. 与 `/queue`、`/approvals` 互补

`/status` 负责总览；更细的审批明细仍由 `/approvals` 负责。避免单条消息过长，也便于后续升级成平台卡片 UI。

## 验证

- `go test ./...`
- 代码路径验证：
  - slash 命令新增 `/status`
  - `/help` 已包含 `/status`

## 文档更新

- README 增加 `/status` 说明
- 产品/开发总览更新为“Gateway 文本命令已具备最小状态总览 + 审批闭环”

## 剩余差距

Gateway 主线仍未对齐的重点包括：

- 原生平台 slash UI / richer command UX
- 原生审批按钮流
- 更完整 token lock / 分布式协调
- 更多平台适配器

### 来源：`0166-gateway-pending-command.md`

# 167-summary-gateway-pending-command

## 背景

聊天侧已经有 `/status`、`/approvals`、`/grant`、`/revoke`、`/approve`、`/deny`，但用户还缺少一个“专门查看最近待审批项详情”的命令。虽然 `/status` 会给出 `last_approval_id`，但不包含命令内容和原因，因此本轮补一个最小 `/pending`。

## 本次实现

- 新增 Gateway 文本命令：
  - `/pending`
- `sessionWorker` 现在会记住最近一次待审批项的：
  - `approval_id`
  - `command`
  - `reason`
- 当没有待审批项时返回：
  - `No pending approval.`
- `/help` 文案同步加入 `/pending`

## 设计取舍

### 1. 只保留最近一条待审批

当前实现先聚焦最常见场景：用户看到最近一条待审批项后立即处理。后续如有必要，再扩展成多条待审批列表。

### 2. 与 `/status` 和 `/approvals` 分层

- `/status`：会话总览
- `/approvals`：已生效授权状态
- `/pending`：最近待审批项详情

这样文本命令职责清晰，不会把不同维度信息塞到一条消息里。

## 验证

- `go test ./...`
- 代码路径验证：
  - `pending_approval` 结果会记录 `approval_id/command/reason`
  - slash 命令新增 `/pending`
  - `/help` 已包含 `/pending`

## 文档更新

- README 增加 `/pending` 说明
- 产品/开发总览更新为“文本状态/审批命令已覆盖总览、待审批、授权状态、授权管理与批准/拒绝”

## 剩余差距

Gateway 主线仍未对齐的重点包括：

- 原生平台 slash UI / richer command UX
- 原生审批按钮流
- 更完整 token lock / 分布式协调
- 更多平台适配器

### 来源：`0167-gateway-telegram-approval-buttons.md`

# 168 总结：Gateway Telegram 原生审批按钮最小闭环

## 1. 背景

文本审批闭环已经具备：

- `/pending`
- `/approvals`
- `/grant` / `/revoke`
- `/approve` / `/deny`

但对齐 Hermes 时，Gateway 仍缺“原生平台审批按钮流”。本轮先补一个最小可用版本：仅对 Telegram 增加原生 Approve / Deny 按钮，不扩展到其他平台，也不引入新的审批状态机。

## 2. 本轮实现

### 2.1 可携带 metadata 的文本发送扩展

- 在 `internal/platform/adapter.go` 增加可选接口 `RichTextSender`
- `internal/gateway/runner.go` 的 `sendText()` 会优先调用该扩展接口
- 非扩展平台保持原有 `Send()` 行为，不受影响

这样可以只给支持原生控件的平台附加按钮，而不改动统一 Gateway 主流程。

### 2.2 Telegram 按钮发送

在 `internal/gateway/platforms/telegram.go`：

- 新增 `SendText(..., meta)` 实现
- 当 metadata 中包含 `approval_id` 时，自动附加 Telegram inline keyboard：
  - `Approve`
  - `Deny`

对应 callback data 直接复用现有文本命令：

- `/approve <id>`
- `/deny <id>`

因此无需新增审批协议，也无需修改 `approval` 工具。

### 2.3 Telegram callback 回流到统一命令处理

Telegram update loop 现在会消费 `CallbackQuery`：

- 提取 `callback.Data`
- 转成统一 `gateway.MessageEvent`
- 继续走现有 slash command 分支

这意味着按钮点击最终仍落到：

- `confirmApproval()`
- 既有授权校验
- 既有审批工具调用链

实现上是“原生 UI + 统一命令内核”，复杂度最低。

### 2.4 `/pending` 响应补充 `approval_id` metadata

`/pending` 返回最近待审批项时，如果存在 `approval_id`，也会附带 metadata，这样 Telegram 侧不仅能看到详情，还能直接点按钮处理。

## 3. 结果

Gateway 审批链路现在分两层：

- 跨平台保底：文本命令 `/pending`、`/approve`、`/deny`
- Telegram 增强：原生 Approve / Deny inline buttons

这样可以把“原生审批按钮流”从完全缺失收敛为“Telegram 最小落地，其他平台后补”。

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

- Discord / Slack / Yuanbao 原生审批按钮
- 原生 slash UI
- 更完整 token lock / 分布式协调
- Gateway richer delivery / 更多平台

### 来源：`0168-gateway-telegram-command-menu.md`

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

### 来源：`0169-gateway-discord-approval-buttons.md`

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

### 来源：`0170-gateway-slack-approval-buttons.md`

# 171 总结：Gateway Slack 原生审批按钮最小闭环

## 1. 背景

在 Telegram 与 Discord 已补最小原生审批按钮后，Slack 侧仍只有文本审批命令：

- `/pending`
- `/approve`
- `/deny`

这会让 Slack 用户继续依赖手工输入命令，审批交互仍不够原生。

因此本轮补一个最小版本：仅增加 Slack block actions 审批按钮，不扩展为完整原生 slash command / modal 流。

## 2. 本轮实现

### 2.1 Slack 富文本发送支持审批按钮

在 `internal/gateway/platforms/slack.go`：

- 新增 `SendText(..., meta)` 实现
- 当 metadata 带有 `approval_id` 时，为消息附加一个 `ActionBlock`
- 其中包含两个按钮：
  - `Approve`
  - `Deny`

按钮 `action_id` 直接复用已有文本命令格式：

- `/approve <id>`
- `/deny <id>`

### 2.2 Socket Mode 交互事件回流统一命令处理

`SlackAdapter.Connect()` 现在除了 `events_api` 消息事件，还会处理 `interactive` 事件：

- 只接收 `block_actions`
- 取第一项 block action 的 `action_id`
- 若其内容是 slash 文本命令，则转成统一 `gateway.MessageEvent`
- 再复用既有 `sessionWorker.handleEvent()` slash command 分支

这样做保持了最小复杂度：

- 不新增审批状态机
- 不新增 Slack 专属确认协议
- 继续复用现有授权与 `approval` 工具链

### 2.3 交互确认

收到 `interactive` 事件后会先 `Ack`，避免 Slack 客户端报交互失败。

## 3. 结果

当前原生审批按钮覆盖变为：

- Telegram：最小原生命令菜单 + 原生审批按钮
- Discord：最小原生审批按钮
- Slack：最小原生审批按钮

审批原生化已经覆盖当前三大主流聊天平台适配器。

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

- Slack 原生命令注册 / 完整 slash command 清单
- Yuanbao 原生审批按钮
- 更完整原生 slash UI / modal / form 流
- 更完整 token lock / 分布式协调

### 来源：`0171-gateway-yuanbao-approval-quick-replies.md`

# 172 总结：Gateway Yuanbao 审批快捷回复最小闭环

## 1. 背景

在 Telegram、Discord、Slack 都已有最小原生审批交互后，Yuanbao 侧仍只有文本 slash 命令：

- `/pending`
- `/approve`
- `/deny`

但 Yuanbao 适配器当前没有稳定的原生按钮/命令菜单 API 可复用，因此不能照搬 Telegram/Discord/Slack 的按钮流。

本轮目标不是伪造按钮，而是补一个更符合 Yuanbao 现状的最小可用闭环：审批快捷回复。

## 2. 本轮实现

### 2.1 Yuanbao 文本快捷回复归一

在 `internal/gateway/runner.go` 新增 `normalizeGatewayCommand()`：

- 仅对 `yuanbao` 平台生效
- 将常见中文快捷回复归一到现有 slash command：
  - `批准` / `同意` / `通过` -> `/approve`
  - `拒绝` / `驳回` -> `/deny`
  - `状态` -> `/status`
  - `待审批` -> `/pending`
  - `审批` -> `/approvals`
  - `帮助` -> `/help`

这样可以继续复用原有命令内核和审批工具，不额外引入平台分支状态机。

### 2.2 待审批提示补充快捷回复文案

当工具返回 `pending_approval` 时，如果当前平台是 Yuanbao：

- 在原有 `Pending approval: /approve <id> or /deny <id>` 基础上
- 追加 `Quick reply: 批准 / 拒绝`

### 2.3 `/pending` 与 `/help` 增强

在 Yuanbao 平台：

- `/pending` 输出会额外提示 `quick_reply: 批准 / 拒绝`
- `/help` 会列出中文快捷回复别名：
  - `状态`
  - `待审批`
  - `审批`
  - `批准`
  - `拒绝`
  - `帮助`

## 3. 结果

Yuanbao 现在具备最小审批快捷回复闭环：

- 用户仍可继续使用原始 slash 命令
- 也可直接回复中文短语完成常用审批动作

这让 Yuanbao 从“只有机械 slash 命令”收敛为“具备最小自然语言快捷交互”，适配了当前平台能力边界。

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

- Yuanbao 原生审批按钮 / 命令菜单
- 更完整原生 slash UI
- 更完整 token lock / 分布式协调
- 更多平台交互组件

### 来源：`0172-gateway-discord-slash-commands.md`

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

### 来源：`0173-gateway-discord-grant-revoke-slash.md`

# 174 总结：Gateway Discord `grant` / `revoke` 原生 slash 参数面

## 1. 背景

上一轮已经补了 Discord 原生 slash 命令最小闭环，但当时仍缺一块明显空白：

- `grant`
- `revoke`

这两个命令在文本模式下已经完整可用，但 Discord 原生 slash UI 只覆盖了状态、审批查看和 approve/deny，授权管理仍要退回手工输入。

## 2. 本轮实现

### 2.1 注册 `grant` / `revoke` 原生命令

在 `internal/gateway/platforms/discord.go` 的命令注册表中新增：

- `grant`
  - `pattern`：可选字符串
  - `ttl`：可选整数
- `revoke`
  - `pattern`：可选字符串

这样 Discord 用户可以通过原生命令面板直接选择授权管理动作，而不必记忆文本格式。

### 2.2 渲染回既有文本命令

新增 `renderDiscordGrantRevoke()`，把 slash 参数翻译回现有命令串：

- `grant(ttl=300)` -> `/grant 300`
- `grant(pattern=delete, ttl=300)` -> `/grant pattern delete 300`
- `revoke(pattern=delete)` -> `/revoke pattern delete`
- `revoke()` -> `/revoke`

这样继续复用：

- `parseApprovalManageCommand()`
- `grantApproval()`
- `revokeApproval()`

不增加新的审批协议或分叉逻辑。

## 3. 结果

Discord 原生 slash command 现在不再只覆盖“查询/单次审批”，也覆盖了“授权管理”。

这让 Discord 侧的原生命令闭环更接近文本命令能力面。

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

- Slack / Yuanbao 原生命令菜单
- 更完整原生表单 / modal 流
- 更完整 token lock / 分布式协调

### 来源：`0174-gateway-slack-slash-command-entrypoint.md`

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

### 来源：`0175-gateway-slack-generic-slash-forwarding.md`

# 176 总结：Gateway Slack 通用 slash 命令转发与即时确认

## 1. 背景

上一轮已经补了 Slack slash 命令入口，但还存在两个明显问题：

1. 若 Slack app 配置的是通用入口（例如 `/agent`），用户输入 `status` 时会被错误拼成 `/agent status`，无法命中现有 Gateway 命令内核。
2. slash command 只做了底层 `Ack`，用户界面没有即时反馈。

因此本轮继续把 Slack slash 入口从“能接住事件”提升到“能稳定转发通用命令”。

## 2. 本轮实现

### 2.1 通用 slash 入口转发

在 `internal/gateway/platforms/slack.go`：

- `renderSlackSlashCommand()` 现在会区分两类入口：
  - **内置命令名直接配置**：如 `/status`、`/pending`
  - **通用代理入口**：如 `/agent`

规则如下：

- 如果 `Text` 本身已经是 slash 命令，直接透传
- 如果 command 名属于 Gateway 内置命令，则保留 `/<cmd> <text>` 形式
- 否则把 text 规范化为 `/<text>`，例如：
  - `/agent status` -> `/status`
  - `/agent grant 300` -> `/grant 300`

这样 Slack app 既可以逐个配置命令，也可以只配置一个通用入口。

### 2.2 即时确认

收到 Slack slash command 后，Socket Mode `Ack` 现在会回一个轻量 ephemeral payload：

- `Accepted. Check the next bot reply.`

这样用户在 Slack 客户端里会立刻看到命令已被接收，而不是只有静默 ACK。

## 3. 结果

Slack 侧现在的原生命令入口更接近真实可用：

- 支持直接 `/status`
- 也支持 `/agent status` 这类通用代理式入口
- 同时具备即时确认反馈

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

### 来源：`0257-gateway-command-matrix-and-stale-lock-closure.md`

# 257 总结：Gateway 命令矩阵与僵尸锁收口

本次围绕 Gateway “平台命令一致性 + 锁可靠性”做了一次性收口。

## 改动

- `cmd/agentd/main.go`
  - `gateway status` 增加锁状态字段：
    - `stale_lock`
    - `stale_token_lock`
  - `gateway start` 启动前自动清理僵尸锁（PID 不存活的 runtime/token lock）。
  - Slack manifest 命令补齐：
    - `/pair` `/unpair` `/cancel` `/queue` `/help`
    - 形成与其它平台一致的核心命令集合。

- 新增锁状态辅助逻辑：
  - `readGatewayLockState`
  - `cleanupStaleGatewayLock`

## 测试

- `cmd/agentd/main_test.go`
  - 核心命令跨平台一致性测试（Telegram/Discord/Slack/Yuanbao）。
  - 僵尸锁识别与清理测试。

验证：

- `go test ./cmd/agentd -count=1`
- `go test ./...`

### 来源：`0045-spotify-summary-merged.md`

# 0045 spotify summary merged

## 模块

- `spotify`

## 类型

- `summary`

## 合并来源

- `0099-spotify-tools-implemented.md`

## 合并内容

### 来源：`0099-spotify-tools-implemented.md`

# 100 Summary - Spotify 工具实现（Web API）

## 变更

将 Spotify 相关工具从占位升级为可用实现（需要 `SPOTIFY_ACCESS_TOKEN`）：

- `spotify_search`：`/v1/search`
- `spotify_devices`：`/v1/me/player/devices`
- `spotify_playback`：`action=get|play|pause`（`/v1/me/player`、`/play`、`/pause`）
- `spotify_queue`：`action=get|add`（`/v1/me/player/queue`）
- `spotify_playlists`：`/v1/me/playlists`
- `spotify_albums`：`/v1/me/albums`
- `spotify_library`：`/v1/me/tracks`

实现位置：

- `internal/tools/spotify.go`
- `internal/tools/builtin.go`：注册与 schema 更新

### 来源：`0060-yuanbao-summary-merged.md`

# 0060 yuanbao summary merged

## 模块

- `yuanbao`

## 类型

- `summary`

## 合并来源

- `0101-yuanbao-sticker-search.md`
- `0116-yuanbao-media-delivery-cos.md`

## 合并内容

### 来源：`0101-yuanbao-sticker-search.md`

# 102 Summary - Yuanbao 贴纸搜索工具实现（yb_search_sticker）

## 变更

- `yb_search_sticker`：基于内置贴纸目录（当前为子集）实现关键词搜索，返回 `sticker_id/name/description/package_id`
- `yb_send_dm` / `yb_send_sticker` / `yb_query_group_info` / `yb_query_group_members`：已接入 Go 版 Yuanbao gateway adapter（WebSocket + 手写 protobuf 编解码，最小可用）

实现位置：

- `internal/tools/yuanbao_stickers.go`
- `internal/tools/yuanbao_tools.go`
- `internal/tools/builtin.go`：更新注册与 schema
- `internal/gateway/platforms/yuanbao.go`：Yuanbao 平台适配器（当前偏 outbound/tool 驱动）
- `internal/gateway/platforms/yuanbao.go`：Yuanbao 平台适配器（已支持最小 inbound：TIMTextElem push -> MessageEvent）
- `internal/yuanbao/*`：sign-token + WS protobuf 编解码 + 最小 WS client

### 来源：`0116-yuanbao-media-delivery-cos.md`

# 117 - Summary: Yuanbao media delivery via COS upload (best-effort)

## Goal

Close the remaining “gateway media delivery” gap for Yuanbao by supporting `send_message(media_path=...)` / `MEDIA:` on the Yuanbao adapter.

Hermes Yuanbao 媒体投递依赖：调用 `genUploadInfo` 获取 COS 临时凭证 → PUT 上传 → 构造 `TIMImageElem` / `TIMFileElem` 发送。

## What changed

- 新增 Yuanbao COS 上传最小实现（Go 版 port）：
  - `internal/yuanbao/media.go`：`GetUploadInfo` + `UploadToCOS` + COS HMAC-SHA1 签名
  - `internal/yuanbao/proto.go`：补齐 `TIMImageElem` / `TIMFileElem` 所需的 MsgContent 编码
  - `internal/yuanbao/client.go`：新增 `SendC2CImage/SendGroupImage/SendC2CFile/SendGroupFile`
- `internal/gateway/platforms/yuanbao.go` 实现 `platform.MediaSender`：
  - 自动判断图片（`image/*`）→ `TIMImageElem`
  - 其他类型 → `TIMFileElem`
  - caption 作为单独的文本消息发送（Yuanbao 媒体 elem 不保证支持 caption 字段）

## Usage

- `send_message(action="send", platform="yuanbao", chat_id="direct:<uid>", media_path="/tmp/a.mp3", message="caption")`
- `send_message(action="send", platform="yuanbao", chat_id="group:<group_code>", message="MEDIA: /tmp/a.png")`

## Notes / limitations

- 依赖对外网络访问：需要能访问 `YUANBAO_API_DOMAIN` 以及 COS 上传域名；在受限网络环境下会失败并返回错误。
- 文件大小限制当前为 50MB（与 Hermes 默认一致）。
- `uuid` 使用 MD5 hex（Hermes 行为）；图片 `image_format` 做了最小映射（jpeg/gif/png/bmp，否则 255）。
