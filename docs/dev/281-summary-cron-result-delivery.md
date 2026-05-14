# 281 总结：Cron 运行结果投递闭环

## 背景

继续按“先做功能”推进 Hermes Cron 高级能力补齐。此前 Cron 已能创建、管理、按 interval/one-shot/cron 表达式触发 Agent 任务，但运行结果只留在 `cron_runs` 中，用户无法在 Telegram/Discord/Slack/Yuanbao 等网关通道收到自动化任务结果。

## 完成内容

- `internal/store/cron_store.go`
  - `cron_jobs` 增加 `delivery_target` 与 `deliver_on`。
  - `cron_runs` 增加 `delivery_target`、`delivery_status`、`delivery_message_id`、`delivery_error`。
  - 新增兼容旧库的列迁移逻辑。
  - 新增 `SetRunDelivery` 保存每次运行的投递结果。
- `internal/tools/cronjob.go`
  - `create/update` 支持 `delivery_target`。
  - 支持 `deliver_on=always|success|failure` 控制成功/失败时是否投递。
- `internal/cronrunner/scheduler.go`
  - Agent 运行结束后按 job 配置投递最终结果。
  - 目标格式为 `platform:chat_id`，例如 `telegram:123`、`discord:channel_id`、`slack:channel_id`、`yuanbao:group:123`。
  - 若最终输出为安全路径下的 `MEDIA: /path`，优先使用支持媒体的 adapter 发送附件。
  - 投递成功、失败、跳过都会写回 run 记录。
- `internal/api/server.go` 与 Web
  - Web Cron 创建请求透传 `delivery_target` / `deliver_on`。
  - Cron 创建表单新增投递目标与投递时机选择。
- 文档同步
  - 更新 README、产品/开发总览、Frontend/TUI 文档与 Web README。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./internal/store ./internal/cronrunner ./internal/tools ./internal/api`
- `npm run test`（`web/`）
- `npm run build`（`web/`）

## 边界

当前投递复用运行中的 Gateway adapter 注册表，要求目标平台已连接；未连接或目标格式错误会记录为 `delivery_status=failed`，不会改变 Agent 运行本身的完成/失败状态。后续 `282` 已补齐链式上下文；脚本动作和更复杂的投递策略仍是后续 Cron 高级能力范围。
