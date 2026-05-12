# 197 - Summary - CLI update bundle plan 命令

## 本次变更

- `agentd update bundle` 现在新增 `plan` 子命令。
- 新增 `agentd update bundle plan -file <bundle.tar.gz|manifest.json> -dest <dir> [-json]`，用于在执行 `apply` 前做 dry-run 规划。
- 输出会区分将要创建和将要覆盖的文件数量，并给出是否需要生成 backup 的预估，便于在真正落盘前评估风险。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle plan -file <tmp>/bundle.tgz -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小 apply 前 dry-run 入口，可在分发脚本里先预估影响范围，再决定是否执行 apply。
