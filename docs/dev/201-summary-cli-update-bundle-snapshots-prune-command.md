# 201 - Summary - CLI update bundle snapshots-prune 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-prune` 子命令。
- 新增 `agentd update bundle snapshots-prune -dest <dir> [-keep N] [-json]`，用于清理目标目录下过旧的手工 snapshot。
- 该命令只处理 `manual-snapshot` 类型条目，不会删除自动生成的 rollback backups。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle snapshots-prune -dest <tmp>/target -keep 0 -json`

## 结果

- update/release 链路现在既能创建/查看 snapshot，也能单独清理手工 restore point，且不会误删回滚备份。
