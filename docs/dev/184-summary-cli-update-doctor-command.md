# 184 - Summary - CLI update doctor 诊断命令

## 本次变更

- 新增 `agentd update doctor [-fetch] [-fetch-tags] [-limit N] [-repo path] [-strict] [-json]`，基于 `update status` 聚合结果输出 `status`、`issues`、`next_actions`。
- 支持 `-strict`，当 update 状态不是 `ok` 时返回非零退出码，便于 CI 或自动化脚本接入。
- README、产品文档、开发文档同步更新 update doctor 能力说明。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update doctor -json`
- `go run ./cmd/agentd update doctor -strict`

## 结果

- update 管理面现在不只是提供原始状态，还能给出可执行诊断建议，最小运维闭环更完整。
