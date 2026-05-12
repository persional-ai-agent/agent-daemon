# 200 - Summary - CLI update bundle snapshots 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots` 子命令。
- 新增 `agentd update bundle snapshots -dest <dir> [-limit N] [-json]`，用于查看目标目录下手工创建的 snapshot 列表。
- 该命令只返回 `manual-snapshot` 类型的条目，和自动生成的 rollback backups 分开管理。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle snapshots -dest <tmp>/target -json`

## 结果

- update/release 链路现在既能创建 snapshot，也能查看现有 snapshot，手工 restore point 运维面更完整。
