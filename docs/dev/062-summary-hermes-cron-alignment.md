# 062 总结：Hermes Cron 最小对齐（interval/one-shot）

## 完成情况

已补齐 Hermes Cron 域的最小可用能力：

- SQLite 内新增 `cron_jobs` / `cron_runs` 存储。
- 新增 `internal/cron` 调度器（ticker 扫描 due job，支持并发度控制）。
- 新增内置工具 `cronjob`（action 压缩 schema）用于管理作业。
- `serve` / `chat` 模式支持通过配置开启 cron scheduler。
- 文档更新：对齐矩阵将 Cron 从“未覆盖”调整为“部分对齐”，并写明边界与后续工作。

## 使用方式（最小）

- 开启：
  - `AGENT_CRON_ENABLED=true`
  - `AGENT_CRON_TICK_SECONDS=5`
  - `AGENT_CRON_MAX_CONCURRENCY=1`
- 创建作业：通过 `cronjob` tool 调用（适用于 agent 内部自举）。

## 边界与待补齐

- cron 表达式目前仅识别并存储，不执行；创建时会提示不支持。
- 未实现 Hermes 的平台投递、prompt threat scanning、脚本型作业与链式上下文。

