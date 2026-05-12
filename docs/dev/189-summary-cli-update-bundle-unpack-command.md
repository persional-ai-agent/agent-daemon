# 189 - Summary - CLI update bundle unpack 命令

## 本次变更

- `agentd update bundle` 现在新增 `unpack` 子命令。
- 新增 `agentd update bundle unpack -file <bundle.tar.gz> -dest <dir> [-json]`，可把本地 bundle 安全解包到目标目录。
- 解包时会校验归档路径，拒绝 `..` 或逃逸目标目录的 entry，避免路径穿越。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle unpack -file <tmp>/bundle.tgz -dest <tmp>/out -json`

## 结果

- update/release 链路现在具备最小本地解包能力，为后续 bundle 安装/回滚流程打基础。
