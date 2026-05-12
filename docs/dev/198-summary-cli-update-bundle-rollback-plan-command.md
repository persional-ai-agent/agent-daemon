# 198 - Summary - CLI update bundle rollback-plan 命令

## 本次变更

- `agentd update bundle` 现在新增 `rollback-plan` 子命令。
- 新增 `agentd update bundle rollback-plan -dest <dir> [-file <backup.tar.gz|manifest.json>] [-json]`，用于在执行 `rollback` 前做 dry-run 预演。
- 未显式传 `-file` 时会自动选择最近一份 backup bundle，并输出本次回滚将创建/覆盖多少文件，帮助先评估回滚影响范围。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle rollback-plan -dest <tmp>/target -json`

## 结果

- update/release 链路现在在 apply 与 rollback 两侧都具备最小预演入口，执行前的可见性更完整。
