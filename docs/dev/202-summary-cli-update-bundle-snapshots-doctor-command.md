# 202 - Summary - CLI update bundle snapshots-doctor 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-doctor` 子命令。
- 新增 `agentd update bundle snapshots-doctor -dest <dir> [-strict] [-json]`，用于诊断目标目录下手工 snapshot 的健康状态。
- 该命令会检查手工 snapshot 数量、manifest 完整性，并在快照过多时建议执行 `snapshots-prune`。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshots-doctor -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle snapshots-doctor -dest <tmp>/empty -strict`

## 结果

- update/release 链路现在对手工 restore point 具备独立诊断能力，snapshot 创建、查看、清理、诊断形成闭环。
