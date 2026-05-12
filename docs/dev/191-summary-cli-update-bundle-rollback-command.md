# 191 - Summary - CLI update bundle rollback 命令

## 本次变更

- `agentd update bundle` 现在新增 `rollback` 子命令。
- 新增 `agentd update bundle rollback -dest <dir> [-file <backup.tar.gz|manifest.json>] [-json]`，可把 `update bundle apply` 生成的 backup bundle 回滚应用到目标目录。
- 当未显式传 `-file` 时，会自动选择 `<dest>/.agent-daemon/release-backups/` 下最新一份 backup bundle；回滚前也会再次生成当前目标状态的 backup bundle，避免回滚本身不可逆。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle rollback -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小本地回滚能力，bundle 分发闭环从“打包/校验/解包/覆盖安装”扩展到“可回退”。
