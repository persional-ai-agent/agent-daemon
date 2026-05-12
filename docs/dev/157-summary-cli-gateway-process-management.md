# 157-summary-cli-gateway-process-management

## 背景

前几轮已经补齐了 `gateway setup`、`setup`、`setup wizard`、`update`、`version`，但 `agentd gateway` 仍只有配置与诊断入口，缺少 Hermes 常见的网关启动/停止管理面。虽然当前阶段不打算引入 systemd/launchd 安装器，但至少需要一个可脚本化、可后台运行的最小进程管理闭环。

## 本次实现

- 新增 `agentd gateway run`：前台运行网关，仅启动 Gateway Runner，不附带 HTTP server。
- 新增 `agentd gateway start`：后台拉起 `agentd gateway run`，日志写入 `<workdir>/.agent-daemon/gateway.log`。
- 新增 `agentd gateway stop`：读取 `<workdir>/.agent-daemon/gateway.pid`，发送 `SIGTERM` 并等待退出。
- 新增 `agentd gateway restart`：复用 stop/start 串联重启。
- 增强 `agentd gateway status`：返回 `running`、`pid`、`pid_path`、`log_path`，便于脚本与人工排障。

## 关键设计

### 1. 保持最小范围

本次只补“手工进程管理”，不实现：

- systemd / launchd / Windows Service 安装
- 自动守护、自动重启、健康探针
- 多实例协调与 token lock

这样可以先把 Hermes CLI 缺口收缩到“安装器级服务管理”和“平台交互能力”，不把复杂度提前引入到当前 Go 版。

### 2. 复用现有 Gateway Runner

`gateway run` 不新建独立网关子系统，而是直接复用现有：

- `mustBuildEngine`
- `buildGatewayAdapters`
- `gateway.NewRunner`

因此行为边界与 `serve` 内嵌网关保持一致，只是运行形态从“HTTP + Gateway 同进程”扩展为“Gateway-only 独立进程”。

### 3. PID / 日志约定

统一使用工作目录下的 `.agent-daemon`：

- PID：`<workdir>/.agent-daemon/gateway.pid`
- 日志：`<workdir>/.agent-daemon/gateway.log`

`gateway run` 启动成功后写入 pid，退出时只在 pid 归属当前进程时删除，避免误删别的实例 pidfile。

## 验证

- `go test ./...`
- `go run ./cmd/agentd gateway status -json`
- `go run ./cmd/agentd gateway stop -json`

说明：`gateway start/run` 依赖至少一个真实平台配置；在无有效平台凭证的开发环境中，本轮以编译与无副作用 CLI 验证为主，没有伪造外部平台联通测试。

## 文档同步

- README 增加 `gateway status/start/stop` 示例
- 产品/开发总览更新为“已具备最小 gateway 进程管理”

## 剩余差距

当前 Gateway 仍未对齐 Hermes 的完整体验，剩余主缺口包括：

- 原生平台 slash UI / 审批按钮流
- token lock 与更强的多实例协调
- system service / 安装器级启动管理
- 更多平台适配器与平台特定交互能力
