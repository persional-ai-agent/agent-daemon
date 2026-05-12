# 205 - Summary - CLI update bundle snapshots-restore-plan 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-restore-plan` 子命令。
- 新增 `agentd update bundle snapshots-restore-plan -dest <dir> [-file <snapshot.tar.gz|manifest.json>] [-json]`，用于在恢复手工 snapshot 前做 dry-run 预演。
- 未显式传 `-file` 时会自动选择最新手工 snapshot，并输出本次恢复将创建/覆盖多少文件。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- 修改目标目录后执行 `go run ./cmd/agentd update bundle snapshots-restore-plan -dest <tmp>/target -json`

## 结果

- update/release 链路现在对手工 restore point 同时具备 restore 与 restore-plan，恢复前也能先做风险评估。
