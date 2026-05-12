# 195 - Summary - CLI update bundle status 命令

## 本次变更

- `agentd update bundle` 现在新增 `status` 子命令。
- 新增 `agentd update bundle status [-file <bundle.tar.gz|manifest.json>] [-dest <dir>] [-json]`，用于聚合查看 bundle 校验状态、backup 列表、rollback 可用性与 backup doctor 结果。
- 当同时传入 `-file` 和 `-dest` 时，一次调用即可同时回答“bundle 能不能用”和“目标目录有没有可回滚点”。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle status -file <tmp>/bundle.tgz -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小 bundle 聚合状态入口，verify、doctor、backups、rollback readiness 已能统一查看。
