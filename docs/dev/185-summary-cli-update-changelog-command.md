# 185 - Summary - CLI update changelog 命令

## 本次变更

- 新增 `agentd update changelog [-fetch-tags] [-limit N] [-repo path] [-json]`，输出最近 tag 到当前 `HEAD` 的提交摘要。
- 当仓库没有 tag 时，回退为直接列出最近提交，保证 release 管理面仍可给出本地变更视图。
- README、产品文档、开发文档同步更新 changelog 能力说明。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update changelog -json`

## 结果

- update/release 现在具备最小变更摘要入口，运维可直接查看当前版本相对最近 release 的提交列表。
