# 183 - Summary - CLI update 安装脚本扩展

## 本次变更

- `agentd update install` 现在额外生成 `update-status.sh` 与 `update-release.sh`，把新补齐的聚合状态与 release 查询能力纳入脚本安装面。
- `agentd update uninstall` 与 `updateInstallStatus()` 同步扩展，能正确移除并识别四个 update 脚本。
- README、产品文档、开发文档同步更新为“最小 update 脚本安装面已覆盖 status/check/release/apply”。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update install -json`
- `go run ./cmd/agentd update status -json`
- `go run ./cmd/agentd update uninstall -json`

## 结果

- update 安装面不再只覆盖 `check/apply`，而是具备完整的最小运维闭环入口。
