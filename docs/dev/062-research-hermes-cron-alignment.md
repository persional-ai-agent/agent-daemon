# 062 调研：Hermes Cron 能力差异与最小对齐路径

## 背景

Hermes Agent 提供内置 Cron scheduler（`cron/` + `tools/cronjob_tools.py`），支持：

- job 存储（jobs.json）、输出归档、repeat/暂停/恢复/触发
- 多种 schedule：`every 30m`、一次性 duration、cron 表达式、时间戳
- 运行方式：触发时启动一个“新会话”的 agent run（或 `no_agent` 仅脚本）
- 可选投递：origin / local / 指定平台（Telegram/Discord/Slack…）与线程
- prompt 扫描与高危注入/外泄模式阻断（cron prompt 是高危入口）
- 可选链式上下文：把其他 job 的最近输出注入当前 job

本项目此前在对齐矩阵中标记 Cron 为“未覆盖”。

## 差异点（对齐目标）

对齐目标分两层：

1. **最小可用 Cron（本次优先）**
   - 作业存储 + interval/one-shot 调度
   - tool 入口：创建/列出/暂停/恢复/删除/触发
   - 运行：按 job prompt 启动独立 session 的 agent run，并保存 run 结果

2. **Hermes 完整 Cron（后续）**
   - cron 表达式计算
   - 平台投递（origin / 指定 chat/thread）
   - job 输出归档与链式上下文（context_from）
   - prompt threat scanning（注入/外泄/不可见字符）
   - `no_agent` 脚本型作业、toolset 限制与 workdir 继承

## 方案选择

考虑到本项目以 Go 实现、且已有 SQLite（`sessions.db`）：

- 先用 SQLite 承载 cron job / run 存储（避免另起 jobs.json 与一致性问题）。
- 先落地 interval/one-shot，cron expr 先存储但不执行（对齐路线明确可迭代）。
- 调度器以 goroutine + ticker 实现，支持并发度控制，避免与 Agent Loop 交织。

## 风险与约束

- 当前 sandbox 环境对网络/监听端口有限制，测试需要避开 `httptest` 监听。
- cron prompt 是“脱离用户实时监督”的入口，需要后续补 prompt scanning 与审批策略对齐。

