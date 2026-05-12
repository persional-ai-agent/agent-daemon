# 187 - Summary - CLI update bundle inspect 命令

## 本次变更

- `agentd update bundle` 现在支持子命令：默认 `build`，以及新增 `inspect`。
- 新增 `agentd update bundle inspect -file <bundle.tar.gz|manifest.json> [-json]`，用于读取本地 bundle / manifest，检查文件是否存在、manifest 是否匹配，以及归档 entry 数。
- README、产品文档、开发文档同步更新 bundle inspect 能力说明。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle inspect -file <tmp>/bundle.tgz -json`

## 结果

- 本地 release bundle 现在不仅能生成，也能被检查与校验，为后续分发/安装流程补上读取入口。
