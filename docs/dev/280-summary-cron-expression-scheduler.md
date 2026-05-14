# 280 总结：Cron 表达式调度执行

## 背景

继续按“先做功能”推进 Hermes 差异补齐。此前 `cronjob` 已支持 interval 与 one-shot，cron-like 表达式只能识别但不会执行。本次把 cron 表达式作为一个完整功能打通到解析、创建、更新和调度器推进。

## 完成内容

- `internal/cron/schedule.go`
  - 新增 5/6 字段 cron 表达式解析。
  - 支持 `*`、列表、范围、步进，例如 `*/15 9-17 * * 1-5`。
  - 支持 6 字段秒级表达式，例如 `*/10 * * * * *`。
  - 支持 day-of-week 的 `7` 作为 Sunday 别名。
  - 新增 `NextRun(expr, after)` 计算下一次触发时间。
- `internal/tools/cronjob.go`
  - `create/update` 不再拒绝 cron 表达式。
  - cron job 会写入 `schedule_kind=cron`、`schedule_expr` 与下一次 `next_run_at`。
- `internal/cronrunner/scheduler.go`
  - cron job 执行后按表达式计算下一次运行时间。
  - 无效表达式会暂停该 job，避免调度循环反复失败。
- `web/src/App.tsx`
  - Cron 页面 schedule 输入提示增加 cron 表达式示例。
- 文档同步
  - 更新 README、产品/开发总览与 Web Cron 管理面总结边界。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./internal/cron ./internal/cronrunner ./internal/tools`

## 边界

当前 cron 表达式实现覆盖数字型 5/6 字段、范围、列表与步进；未实现月份/星期英文别名、`L/W/#/?` 等扩展语法。后续 `281` 已补齐运行结果投递，链式上下文仍在后续 Cron 高级能力范围。
