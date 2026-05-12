# 193 - Summary - CLI update bundle prune 命令

## 本次变更

- `agentd update bundle` 现在新增 `prune` 子命令。
- 新增 `agentd update bundle prune -dest <dir> [-keep N] [-json]`，用于清理 `<dest>/.agent-daemon/release-backups/` 下过旧的 backup bundle。
- 清理时会同时删除 `.tar.gz` 与对应 `.json` manifest，只保留最近 `N` 份备份，避免 backup 目录无限增长。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle prune -dest <tmp>/target -keep 0 -json`

## 结果

- update/release 链路现在具备最小 backup 生命周期管理能力，本地回滚点不再只能累积、不能清理。
