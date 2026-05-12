# 188 - Summary - CLI update bundle verify 命令

## 本次变更

- `agentd update bundle` 现在新增 `verify` 子命令。
- 新增 `agentd update bundle verify -file <bundle.tar.gz|manifest.json> [-strict] [-json]`，基于 inspect 结果输出 `status`、`issues`、`next_actions`。
- 支持 `-strict`，当 bundle 校验结果不是 `ok` 时返回非零退出码，便于 CI 或分发脚本接入。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle verify -file <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle verify -file /tmp/not-found.tgz -strict`

## 结果

- 本地 release bundle 现在具备可脚本化的 verify 入口，安装器级分发前的校验能力更完整。
