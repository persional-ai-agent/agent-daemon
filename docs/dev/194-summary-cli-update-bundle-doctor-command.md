# 194 - Summary - CLI update bundle doctor 命令

## 本次变更

- `agentd update bundle` 现在新增 `doctor` 子命令。
- 新增 `agentd update bundle doctor -dest <dir> [-strict] [-json]`，用于检查目标目录下 backup bundle 的数量、manifest 完整性与是否需要清理。
- 支持 `-strict`，当诊断结果不是 `ok` 时返回非零退出码，便于分发脚本或 CI 在执行 rollback 前先做健康检查。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle doctor -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小 bundle 运维诊断入口，build/apply/backups/prune/rollback 已能被统一检查。
