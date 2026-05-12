# 203 - Summary - CLI update bundle snapshots-status 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-status` 子命令。
- 新增 `agentd update bundle snapshots-status -dest <dir> [-limit N] [-json]`，用于聚合查看手工 snapshot 列表、诊断结果和最近可用 restore point。
- 输出包含 `snapshots`、`doctor`、`latest_snapshot_path`、`snapshot_ready`，避免手工快照运维需要多次调用分散命令。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle snapshots-status -dest <tmp>/target -json`

## 结果

- update/release 链路现在对手工 restore point 具备聚合状态入口，snapshot 创建、查看、诊断、清理已经形成最小闭环。
