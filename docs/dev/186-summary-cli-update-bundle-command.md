# 186 - Summary - CLI update bundle 命令

## 本次变更

- 新增 `agentd update bundle [-fetch-tags] [-repo path] [-out file] [-json]`，可把当前 git checkout 导出为本地 `tar.gz` release bundle。
- bundle 旁会写出同名 `.json` manifest，记录 commit、latest tag、文件数与生成时间，便于后续安装器级分发流程接入。
- 默认输出到 `<repo>/.agent-daemon/release/agent-daemon-<tag-or-commit>.tar.gz`，优先使用最近 tag，否则回退到短 commit。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -json`

## 结果

- update/release 管理面现在具备最小本地打包能力，距离完整安装器级分发流程又近一步。
