# 192 - Summary - CLI update bundle backups 命令

## 本次变更

- `agentd update bundle` 现在新增 `backups` 子命令。
- 新增 `agentd update bundle backups -dest <dir> [-limit N] [-json]`，用于查看 `<dest>/.agent-daemon/release-backups/` 下最近的 backup bundle 列表。
- 输出会携带 backup bundle 路径、manifest 路径、生成时间、文件数、原始 source bundle 路径，便于在执行 `rollback` 前先查看可用回滚点。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle backups -dest <tmp>/target -json`

## 结果

- update/release 链路现在不仅能生成并消费 backup bundle，也能查询最近回滚点，最小本地运维面更完整。
