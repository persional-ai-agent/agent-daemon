# 199 - Summary - CLI update bundle snapshot 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshot` 子命令。
- 新增 `agentd update bundle snapshot -dest <dir> [-out file] [-json]`，用于主动把目标目录导出为一份本地 restore point。
- 默认会把快照写到 `<dest>/.agent-daemon/release-backups/`，并自动排除该备份目录自身，避免快照递归打包已有 backups。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`

## 结果

- update/release 链路现在支持在 apply/rollback 之外主动创建目标目录快照，手工 restore point 能力更完整。
