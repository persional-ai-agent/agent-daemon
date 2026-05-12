# 181 - Summary - CLI update release 命令

## 本次变更

- 新增 `agentd update release [-fetch-tags] [-limit N] [-json]`，用于查看当前提交对应 tag、本地最新 tag 与最近 release tags。
- `update` 子命令现在覆盖 `check`、`apply`、`release`、`install`、`uninstall`，补上最小 release 视图。
- README、产品文档、开发文档同步更新 update release 能力说明。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update release -json`

## 结果

- CLI 现在具备最小 release 元数据查询能力，安装器级 update/release 管理缺口进一步收敛到真正的分发与升级流程。
