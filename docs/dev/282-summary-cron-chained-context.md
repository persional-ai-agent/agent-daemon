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
