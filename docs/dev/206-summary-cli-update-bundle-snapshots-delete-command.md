# 206 - Summary - CLI update bundle snapshots-delete 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-delete` 子命令。
- 新增 `agentd update bundle snapshots-delete -dest <dir> [-file <snapshot.tar.gz|manifest.json>] [-json]`，用于定向删除指定手工 snapshot。
- 未显式传 `-file` 时会自动选择最新手工 snapshot，并同步删除对应 `.json` manifest。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle snapshots-delete -dest <tmp>/target -json`

## 结果

- update/release 链路现在既支持批量 `snapshots-prune`，也支持定向删除单个 restore point，手工 snapshot 生命周期更完整。
