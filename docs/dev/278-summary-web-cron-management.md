# 278 总结：Web Cron 管理面补齐

## 背景

延续“先做功能”的原则，在 Web Dashboard 已接入基础管理页后，继续补齐 Cron 任务管理能力。后端已有 `cronjob` 工具与 `CronStore`，本次不复制调度逻辑，只为 UI 增加薄 API 封装。

## 完成内容

- `internal/api/server.go`
  - 新增 `GET /v1/ui/cron/jobs`：列出 Cron jobs。
  - 新增 `POST /v1/ui/cron/jobs`：创建 Cron job。
  - 新增 `GET /v1/ui/cron/jobs/{job_id}`：查看单个 job。
  - 新增 `POST /v1/ui/cron/jobs/action`：执行 `pause/resume/trigger/remove/runs/run_get/update` 等操作。
  - 所有操作底层复用 `cronjob` 工具分发。
- `web/src/lib/api.ts`
  - 新增 Cron UI API client。
- `web/src/App.tsx`
  - 新增 `cron` 页面，支持创建、列表、详情、暂停、恢复、触发、删除与运行记录查看。
- 文档同步
  - 更新 Web README、Frontend/TUI 用户与开发文档、产品/开发总览。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./internal/api`
- `npm run test`
- `npm run build`

## 边界

本次补齐的是管理面和 API 封装。后续 `280` 已补齐 cron expression 调度执行，`281` 已补齐运行结果投递；链式上下文和更完整运行详情仍是后续功能。
