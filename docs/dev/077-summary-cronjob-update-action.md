# 077 总结：cronjob 补齐 update 动作

## 变更

`cronjob` 工具新增 `action=update`，支持更新：

- `name` / `prompt`
- `schedule`（interval/one-shot；cron expr 仍不执行）
- `repeat`
- `paused`

并落地到 SQLite（`cron_jobs` 表）存储层。

