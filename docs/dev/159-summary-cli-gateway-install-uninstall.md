# 159-summary-cli-gateway-install-uninstall

## 背景

上一轮已经补齐了 `gateway run/start/stop/restart`，但网关管理仍缺少一个可交付给运维脚本或人工使用的“安装面”。当前阶段不打算直接接 systemd/launchd，因此本轮补的是最小本地脚本安装，而不是系统服务注册。

## 本次实现

- 新增 `agentd gateway install`
  - 在 `<workdir>/.agent-daemon/bin/` 生成：
    - `gateway-start.sh`
    - `gateway-stop.sh`
    - `gateway-restart.sh`
    - `gateway-status.sh`
  - 生成 `gateway-install.json` 记录 executable、config path、workdir、scripts
- 新增 `agentd gateway uninstall`
  - 删除上述脚本和 manifest
  - 支持可选 `-stop` 先停止当前网关进程
- 增强 `agentd gateway status`
  - 返回 `installed`
  - 返回 `install_dir`
  - 返回 `manifest_path`

## 设计取舍

### 1. 只做 repo 内可控安装

当前安装面只覆盖“本地脚本落盘”，不覆盖：

- systemd / launchd / Windows Service
- 开机自启注册
- PATH 注入
- 守护进程 supervisor

这样可以先把 Hermes CLI 中常见的 `install/uninstall` 操作补齐到最小可用，不引入平台相关复杂度。

### 2. 配置路径固化到脚本

如果调用 `gateway install -file <path>`，生成的脚本会把该 `-file` 参数固化进去，便于后续直接执行脚本而不用再次手填配置路径。

## 验证

- `go test ./...`
- `go run ./cmd/agentd gateway status -json`
- `tmpdir=$(mktemp -d) && go run ./cmd/agentd gateway install -workdir "$tmpdir" -json`
- `tmpdir=$(mktemp -d) && go run ./cmd/agentd gateway install -workdir "$tmpdir" -json && go run ./cmd/agentd gateway uninstall -workdir "$tmpdir" -json`

## 文档更新

- README 增加 `gateway install/uninstall` 示例
- 产品/开发总览更新为“已具备最小 gateway 脚本安装管理”

## 剩余差距

Gateway 主线剩余高价值缺口仍是：

- 原生平台 slash UI
- 审批按钮流
- token lock / 多实例协调
- system service 级安装管理
- 更多平台适配器
