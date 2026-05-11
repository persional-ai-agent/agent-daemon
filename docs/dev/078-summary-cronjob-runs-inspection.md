# 078 总结：cronjob 增加 runs/run_get（运行记录查询）

## 变更

`cronjob` 工具新增：

- `action=runs`：按 `job_id`（可选）列出最近运行记录
- `action=run_get`：按 `run_id` 获取单次运行详情（含 output/error）

用于对齐 Hermes cron 输出/历史查询体验，并便于排查定时任务问题。

