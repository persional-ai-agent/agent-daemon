# 160-summary-cli-update-install-uninstall

## 背景

此前 `update` 只有 `check/apply`，能执行 git 检查与快进更新，但还缺一个可脚本化的“安装面”。在不引入完整安装器的前提下，本轮先补最小 `update install/uninstall`，与前面的 gateway 脚本安装思路保持一致。

## 本次实现

- 新增 `agentd update install`
  - 在仓库根目录 `.agent-daemon/bin/` 生成：
    - `update-check.sh`
    - `update-apply.sh`
  - 写入 `update-install.json` manifest
- 新增 `agentd update uninstall`
  - 删除上述脚本和 manifest
- 新脚本会固定切换到当前 git repo root，再执行 `agentd update -fetch` 或 `agentd update apply`

## 设计取舍

### 1. 仍然是 git checkout 级

本轮没有做二进制下载、版本选择、发布通道，也没有做自更新安装器。补的是“把当前已存在的 git update 流固化成可直接执行的脚本入口”。

### 2. 以 repo root 为安装锚点

`update` 本质只对 git checkout 生效，因此安装目录绑定到当前仓库：

- 安装目录：`<repo>/.agent-daemon/bin`
- manifest：`<repo>/.agent-daemon/bin/update-install.json`

这样脚本能稳定回到正确仓库执行更新，不依赖调用方所在目录。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update install -json`
- `tmpdir=$(mktemp -d) && cp -R .git "$tmpdir/.git"` 不适合当前轮次，因此采用当前仓库真实烟测安装/卸载
- `go run ./cmd/agentd update uninstall -json`

## 文档更新

- README 增加 `update install/uninstall` 示例
- 产品/开发总览收口为“已有最小 update 脚本安装；仍缺完整安装器级 update”

## 剩余差距

CLI 主线剩余缺口继续收敛为：

- 全屏 TUI
- 完整安装器级 update / release 管理
- Gateway 原生平台交互能力
