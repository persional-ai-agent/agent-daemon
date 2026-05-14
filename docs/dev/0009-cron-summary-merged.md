# 0009 cron summary merged

## 模块

- `cron`

## 类型

- `summary`

## 合并来源

- `0009-cron-summary-merged.md`
- `0010-cronjob-summary-merged.md`

## 合并内容

### 来源：`0009-cron-summary-merged.md`

# 0009 cron summary merged

## 模块

- `cron`

## 类型

- `summary`

## 合并来源

- `0280-cron-expression-scheduler.md`
- `0281-cron-result-delivery.md`
- `0282-cron-chained-context.md`
- `0283-cron-script-run-mode.md`

## 合并内容

### 来源：`0280-cron-expression-scheduler.md`

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

### 来源：`0281-cron-result-delivery.md`

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

### 来源：`0282-cron-chained-context.md`

# 282 总结：Cron 链式上下文模式

## 背景

继续按“先做功能”推进 Cron 自动化闭环。此前 Cron 已支持定时执行、表达式调度和运行结果投递，但每次运行默认使用独立会话，周期任务无法自然继承上一次运行的结论、状态和工具结果。

## 完成内容

- `internal/store/cron_store.go`
  - `cron_jobs` 增加 `context_mode`，默认 `isolated`。
  - 支持旧库自动迁移 `context_mode` 列。
- `internal/tools/cronjob.go`
  - `create/update` 支持 `context_mode=isolated|chained`。
  - 增加 `chain_context=true` 作为 `context_mode=chained` 的兼容别名。
- `internal/cronrunner/scheduler.go`
  - 默认 `isolated` 模式继续使用 `cron:<job_id>:<run_id>` 独立会话。
  - `chained` 模式使用稳定会话 `cron:<job_id>`。
  - 每次运行前从稳定会话加载历史消息，再交给 `Engine.Run()` 继续推理。
  - 每次 run 记录仍保存实际 `session_id`，方便查看该 job 的链式上下文入口。
- API / Web
  - `/v1/ui/cron/jobs` 与 `/v1/ui/cron/jobs/action` 透传 `context_mode` / `chain_context`。
  - Web Cron 创建表单新增 `isolated/chained` 模式选择。
- 文档同步
  - 更新 README、产品/开发总览、Frontend/TUI 文档与 Web README。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./internal/store ./internal/cronrunner ./internal/tools ./internal/api`
- `npm run test`（`web/`）
- `npm run build`（`web/`）

## 边界

链式上下文只改变 cron job 的会话复用与历史加载，不改变调度、投递、repeat 计数和错误状态语义。默认仍是 `isolated`，避免已有任务意外携带历史。长历史压缩继续复用 Engine 现有上下文压缩机制。

### 来源：`0283-cron-script-run-mode.md`

# 283 总结：Cron 脚本动作模式

## 背景

继续按“先做功能”推进 Cron 自动化。此前 Cron 任务只能运行 Agent 提示词（`prompt`），在纯运维/批处理场景中，用户仍需绕过 `cronjob` 自建外部调度。  
本次补齐 `run_mode=script`，让 Cron 任务可直接执行本地 shell 命令。

## 完成内容

- `internal/store/cron_store.go`
  - `cron_jobs` 增加：
    - `run_mode`（`agent|script`，默认 `agent`）
    - `script_command`
    - `script_cwd`
    - `script_timeout`
  - 兼容旧库自动迁移新增列。
  - `CreateJob/UpdateJob` 增加模式校验：
    - `agent` 模式要求 `prompt`。
    - `script` 模式要求 `script_command`。
- `internal/tools/cronjob.go`
  - `create/update` 支持：
    - `run_mode`
    - `script_command`
    - `script_cwd`
    - `script_timeout`
  - schema 同步新增字段说明。
- `internal/cronrunner/scheduler.go`
  - 新增执行分支：
    - `run_mode=agent`：保持原有 `Engine.Run`。
    - `run_mode=script`：调用 `tools.RunForeground` 执行命令，写入 `exit_code + output`。
  - 脚本模式仍复用现有 run 记录、重复调度、投递（`delivery_target`）逻辑。
- API / Web
  - `/v1/ui/cron/jobs` 与 `/v1/ui/cron/jobs/action` 支持脚本参数透传。
  - Web Cron 页面新增 `agent/script` 模式选择，脚本模式可填写命令、工作目录、超时秒数。
- 测试
  - `internal/store/cron_store_test.go` 增加脚本模式字段持久化校验。
  - `internal/tools/cronjob_tool_test.go` 增加脚本模式创建校验。
  - `internal/cronrunner/scheduler_test.go` 增加脚本执行分支校验。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./internal/store ./internal/cronrunner ./internal/tools ./internal/api`
- `npm run test`（`web/`）
- `npm run build`（`web/`）

## 边界

当前脚本模式基于本地 shell 前台执行，不包含多后端（docker/ssh 等）切换与脚本审批流编排；这些能力仍属于后续 Cron 高级扩展。

### 来源：`0010-cronjob-summary-merged.md`

# 0010 cronjob summary merged

## 模块

- `cronjob`

## 类型

- `summary`

## 合并来源

- `0076-cronjob-update-action.md`
- `0077-cronjob-runs-inspection.md`

## 合并内容

### 来源：`0076-cronjob-update-action.md`

# 077 总结：cronjob 补齐 update 动作

## 变更

`cronjob` 工具新增 `action=update`，支持更新：

- `name` / `prompt`
- `schedule`（interval/one-shot；cron expr 仍不执行）
- `repeat`
- `paused`

并落地到 SQLite（`cron_jobs` 表）存储层。

### 来源：`0077-cronjob-runs-inspection.md`

# 078 总结：cronjob 增加 runs/run_get（运行记录查询）

## 变更

`cronjob` 工具新增：

- `action=runs`：按 `job_id`（可选）列出最近运行记录
- `action=run_get`：按 `run_id` 获取单次运行详情（含 output/error）

用于对齐 Hermes cron 输出/历史查询体验，并便于排查定时任务问题。
