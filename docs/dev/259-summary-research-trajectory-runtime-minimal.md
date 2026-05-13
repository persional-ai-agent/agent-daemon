# 259 总结：Research/RL/Trajectory 最小运行时闭环

本次补齐了 Research/RL/Trajectory 的最小可用主链路，目标是“可批量执行任务并产出可压缩轨迹数据”。

## 新增能力

- `agentd research run`
  - 输入：`-tasks <jsonl>`
  - 每行任务结构：`{input, session_id?, id?, metadata?}`
  - 输出：trajectory `jsonl`（包含事件、结果、耗时、错误）
- `agentd research compress`
  - 输入 trajectory `jsonl`
  - 输出压缩后的 `compact jsonl.gz`（保留训练关键字段，裁剪长文本）
- `agentd research stats`
  - 统计 trajectory 文件的总量、成功数、失败数、平均耗时

## 实现

- `internal/research/batch.go`
  - `LoadTasks`
  - `RunBatch`
  - `CompressTrajectories`
  - `StatsTrajectories`
- `cmd/agentd/main.go`
  - 新增 `research` 子命令路由与 `run/compress/stats` 三个子命令

## 测试

- `internal/research/batch_test.go`
  - 任务加载
  - 轨迹压缩与统计

验证：

- `go test ./internal/research -count=1`
- `go test ./...`
