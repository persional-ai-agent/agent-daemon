# 196 - Summary - CLI update bundle manifest 命令

## 本次变更

- `agentd update bundle` 现在新增 `manifest` 子命令。
- 新增 `agentd update bundle manifest -file <bundle.tar.gz|manifest.json> [-dest <dir>] [-json]`，用于读取并整理 bundle 的 manifest 元数据。
- 当同时传入 `-dest` 时，输出会额外附带目标目录的最近 backup 信息，便于在分发前同时确认“待安装 bundle 是什么”和“目标目录当前能否回滚”。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle manifest -file <tmp>/bundle.tgz -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小 bundle 分发清单入口，bundle 元数据与目标目录回滚点可被统一导出。
