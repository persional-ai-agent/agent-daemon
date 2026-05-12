# 190 - Summary - CLI update bundle apply 命令

## 本次变更

- `agentd update bundle` 现在新增 `apply` 子命令。
- 新增 `agentd update bundle apply -file <bundle.tar.gz|manifest.json> -dest <dir> [-json]`，可把本地 bundle 覆盖安装到目标目录。
- 应用前会扫描目标目录中将被覆盖的文件，并自动在 `<dest>/.agent-daemon/release-backups/` 生成一份 backup bundle 与 manifest，便于后续回滚流程接入。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小本地覆盖安装能力，并开始沉淀可复用的 backup bundle，为后续回滚命令打基础。
