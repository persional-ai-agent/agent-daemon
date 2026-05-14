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
