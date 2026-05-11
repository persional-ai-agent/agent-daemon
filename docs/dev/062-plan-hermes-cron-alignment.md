# 062 计划：Hermes Cron 最小对齐（interval/one-shot）

## 目标（可验证）

- `cronjob` 工具可用：`create/list/get/pause/resume/remove/trigger`。
- 开启 `AGENT_CRON_ENABLED=true` 后，调度器会周期性扫描 due job 并触发独立 session 的 agent run。
- cron job 与 run 结果可持久化在 SQLite（与 `sessions.db` 同库）。
- 文档对齐矩阵更新：Cron 从“未覆盖”更新为“部分对齐”，并写明边界。

## 实施步骤

1. **存储**
   - 新增 `cron_jobs`、`cron_runs` 表，复用现有 SQLite 连接。
2. **调度器**
   - ticker 扫描 due jobs；按并发度执行；对 interval/once 计算 next_run_at。
3. **工具**
   - 新增内置工具 `cronjob`，action 压缩 schema。
4. **集成**
   - `serve` 与 `chat` 模式按配置启动 scheduler。
5. **文档**
   - 更新 `README.md`、`docs/overview-product*.md` 与 `docs/dev/README.md` 索引。
6. **测试**
   - schedule 解析与 cron store CRUD 单测（无网络/无端口依赖）。

## 不在本次范围

- cron 表达式执行
- 平台投递与 origin 捕获
- prompt threat scanning
- `no_agent` 脚本作业、context_from 链式作业

