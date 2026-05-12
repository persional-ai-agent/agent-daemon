# 204 - Summary - CLI update bundle snapshots-restore 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-restore` 子命令。
- 新增 `agentd update bundle snapshots-restore -dest <dir> [-file <snapshot.tar.gz|manifest.json>] [-json]`，用于把手工 snapshot 直接恢复到目标目录。
- 未显式传 `-file` 时会自动选择最新手工 snapshot；恢复时仍会生成 rollback backup，保证恢复过程可逆。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- 修改目标目录后执行 `go run ./cmd/agentd update bundle snapshots-restore -dest <tmp>/target -json`

## 结果

- update/release 链路现在对手工 restore point 具备直接恢复入口，snapshot 创建、查看、诊断、清理、恢复形成最小闭环。
