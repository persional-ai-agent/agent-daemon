# 182 - Summary - CLI update status 聚合命令

## 本次变更

- 新增 `agentd update status [-fetch] [-fetch-tags] [-limit N] [-repo path] [-json]`，统一返回 git upstream 状态、release tags 与 update 脚本安装状态。
- 复用既有 `gitUpdateStatus` / `gitReleaseInfo` 逻辑，并补充 `updateInstallStatus()` 读取 `update-install.json` 与已安装脚本。
- README、产品文档、开发文档同步更新 `update status` 能力说明。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update status -json`

## 结果

- `update` 现在有一个统一入口可用于运维查看，不再需要分别执行 `check`、`release`、`install` 结果拼装当前状态。
