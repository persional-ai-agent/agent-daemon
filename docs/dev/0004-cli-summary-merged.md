# 0004 cli summary merged

## 模块

- `cli`

## 类型

- `summary`

## 合并来源

- `0005-cli-summary-merged.md`
- `0015-doctor-summary-merged.md`

## 合并内容

### 来源：`0005-cli-summary-merged.md`

# 0005 cli summary merged

## 模块

- `cli`

## 类型

- `summary`

## 合并来源

- `0053-cli-config-management.md`
- `0054-cli-model-management.md`
- `0055-cli-tools-inspection.md`
- `0056-cli-doctor.md`
- `0057-cli-gateway-management.md`
- `0059-cli-sessions.md`
- `0060-cli-session-show-stats.md`
- `0151-cli-gateway-setup.md`
- `0152-cli-setup-entrypoint.md`
- `0153-cli-setup-wizard.md`
- `0154-cli-update-command.md`
- `0155-cli-version-command.md`
- `0156-cli-gateway-process-management.md`
- `0157-cli-bootstrap-command.md`
- `0158-cli-gateway-install-uninstall.md`
- `0159-cli-update-install-uninstall.md`
- `0176-cli-gateway-slack-manifest-export.md`
- `0177-cli-gateway-discord-manifest-export.md`
- `0178-cli-gateway-telegram-manifest-export.md`
- `0179-cli-gateway-yuanbao-manifest-export.md`
- `0180-cli-update-release-command.md`
- `0181-cli-update-status-command.md`
- `0182-cli-update-install-script-expansion.md`
- `0183-cli-update-doctor-command.md`
- `0184-cli-update-changelog-command.md`
- `0185-cli-update-bundle-command.md`
- `0186-cli-update-bundle-inspect-command.md`
- `0187-cli-update-bundle-verify-command.md`
- `0188-cli-update-bundle-unpack-command.md`
- `0189-cli-update-bundle-apply-command.md`
- `0190-cli-update-bundle-rollback-command.md`
- `0191-cli-update-bundle-backups-command.md`
- `0192-cli-update-bundle-prune-command.md`
- `0193-cli-update-bundle-doctor-command.md`
- `0194-cli-update-bundle-status-command.md`
- `0195-cli-update-bundle-manifest-command.md`
- `0196-cli-update-bundle-plan-command.md`
- `0197-cli-update-bundle-rollback-plan-command.md`
- `0198-cli-update-bundle-snapshot-command.md`
- `0199-cli-update-bundle-snapshots-command.md`
- `0200-cli-update-bundle-snapshots-prune-command.md`
- `0201-cli-update-bundle-snapshots-doctor-command.md`
- `0202-cli-update-bundle-snapshots-status-command.md`
- `0203-cli-update-bundle-snapshots-restore-command.md`
- `0204-cli-update-bundle-snapshots-restore-plan-command.md`
- `0205-cli-update-bundle-snapshots-delete-command.md`
- `0234-cli-align-ui-contract-semantics.md`
- `0261-cli-tui-standalone-auto-mode.md`
- `0262-cli-tui-source-fallback-and-boot-message.md`
- `0274-cli-tui-stateful-command-surface.md`

## 合并内容

### 来源：`0053-cli-config-management.md`

# 054 总结：CLI 配置管理最小对齐

## 变更摘要

新增 `agentd config list|get|set`，补齐 Hermes 风格配置管理面的最小可用入口。

## 新增能力

```bash
agentd config list
agentd config get api.model
agentd config set api.model gpt-4o-mini
agentd config set provider.fallback anthropic
```

默认读写 `config/config.ini`，也支持：

- `AGENT_CONFIG_FILE=/path/to/config.ini`
- 子命令 `-file /path/to/config.ini`

配置优先级保持不变：环境变量 > 配置文件 > 内置默认值。

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/config/config.go` | `Load()` 增加 `AGENT_CONFIG_FILE` 查找 |
| `internal/config/manage.go` | 新增 INI 配置读写、列表与密钥脱敏函数 |
| `internal/config/config_test.go` | 增加配置管理与 `AGENT_CONFIG_FILE` 测试 |
| `cmd/agentd/main.go` | 新增 `config list|get|set` 子命令 |
| `README.md` | 增加配置管理示例 |
| `docs/overview-product.md` | 增加 CLI 配置管理能力说明 |
| `docs/overview-product-dev.md` | 增加配置管理模块与设计说明 |
| `docs/dev/README.md` | 增加 054 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/config`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过（需要沙箱外运行；默认沙箱禁止 `httptest` 监听本地端口）
- 临时配置文件手动验证：`config set/get/list -file /tmp/.../config.ini` 通过

### 来源：`0054-cli-model-management.md`

# 055 总结：CLI 模型管理最小对齐

## 变更摘要

新增 `agentd model show|providers|set`，补齐 Hermes `hermes model` 的最小本地模型切换体验。

## 新增能力

```bash
agentd model show
agentd model providers
agentd model set openai gpt-4o-mini
agentd model set anthropic:claude-3-5-haiku-latest
agentd model set -base-url https://api.openai.com/v1 codex gpt-5-codex
```

`model set` 默认写入 `config/config.ini`，也支持 `AGENT_CONFIG_FILE` 和 `-file`。环境变量仍优先于配置文件。

## 修改文件

| 文件 | 变更 |
|------|------|
| `cmd/agentd/main.go` | 新增 `model show|providers|set` 子命令与解析/写入 helper |
| `cmd/agentd/main_test.go` | 增加模型参数解析与 provider 专属配置键写入测试 |
| `internal/config/config.go` | 新增 `LoadFile`，支持 `model show -file` 读取指定配置 |
| `internal/config/config_test.go` | 增加 `LoadFile` 覆盖 |
| `README.md` | 增加模型切换示例 |
| `docs/overview-product.md` | 更新 CLI 管理面能力说明 |
| `docs/overview-product-dev.md` | 更新 CLI 配置/模型管理设计说明 |
| `docs/dev/README.md` | 增加 055 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/config ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 临时配置文件手动验证：`model set -file ... -base-url ... anthropic:claude-test`、`model show -file ...`、`config list -file ...` 均通过

### 来源：`0055-cli-tools-inspection.md`

# 056 总结：CLI 工具查看最小对齐

## 变更摘要

新增 `agentd tools list|show|schemas`，保留 `agentd tools` 原有列名行为，补齐 Hermes 工具管理入口中的最小工具查看能力。

## 新增能力

```bash
agentd tools
agentd tools list
agentd tools show terminal
agentd tools schemas
```

## 修改文件

| 文件 | 变更 |
|------|------|
| `cmd/agentd/main.go` | 新增 `runTools`、schema 查找与 JSON 输出 |
| `cmd/agentd/main_test.go` | 增加 `findToolSchema` 测试 |
| `README.md` | 增加工具查看示例 |
| `docs/overview-product.md` | 更新 CLI 管理面能力说明 |
| `docs/overview-product-dev.md` | 更新 CLI 工具查看设计说明 |
| `docs/dev/README.md` | 增加 056 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 手动验证：`tools list` 输出工具名，`tools show terminal` 输出单个 JSON schema，`tools schemas` 输出完整 schema 列表

### 来源：`0056-cli-doctor.md`

# 057 总结：CLI 本地诊断最小对齐

## 变更摘要

新增 `agentd doctor`，补齐 Hermes `doctor` 的最小本地诊断能力。

## 新增能力

```bash
agentd doctor
agentd doctor -json
```

检查项：

- 配置文件路径与环境变量优先级提示
- `workdir`
- `data_dir`
- provider/model 配置
- provider API key 是否为空
- MCP transport
- Gateway token 基础配置
- 内置工具注册数量

## 修改文件

| 文件 | 变更 |
|------|------|
| `cmd/agentd/main.go` | 新增 `doctor` 子命令与检查 helper |
| `cmd/agentd/main_test.go` | 增加 doctor 分支测试 |
| `README.md` | 增加诊断命令示例 |
| `docs/overview-product.md` | 更新 CLI 管理面能力说明并修正内置工具数量 |
| `docs/overview-product-dev.md` | 更新 CLI doctor 设计说明 |
| `docs/dev/README.md` | 增加 057 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 手动验证：`AGENT_DAEMON_HOME=/tmp/agentd-doctor-home go run ./cmd/agentd doctor -json` 输出 `ok/warn/error` JSON
- 手动验证：默认 `~/.agent-daemon` 在当前沙箱不可写时，`doctor` 返回 `data_dir` error 并以非零状态退出

### 来源：`0057-cli-gateway-management.md`

# 058 总结：CLI 网关管理最小对齐

## 变更摘要

新增 `agentd gateway status|platforms|enable|disable`，补齐 Hermes `gateway` 管理入口的最小配置能力。

## 新增能力

```bash
agentd gateway status
agentd gateway status -json
agentd gateway platforms
agentd gateway enable
agentd gateway disable
```

`enable/disable` 只写入 `gateway.enabled`；平台 token 继续通过 `agentd config set` 管理。

当前 `gateway platforms` 输出包含：Telegram、Discord、Slack、Yuanbao（Yuanbao 凭证来自 `YUANBAO_TOKEN` 或 `YUANBAO_APP_ID/YUANBAO_APP_SECRET`）。

## 修改文件

| 文件 | 变更 |
|------|------|
| `cmd/agentd/main.go` | 新增 gateway 子命令与状态 helper |
| `cmd/agentd/main_test.go` | 增加 gateway 状态测试 |
| `README.md` | 增加网关配置示例 |
| `docs/overview-product.md` | 更新 CLI 管理面能力说明 |
| `docs/overview-product-dev.md` | 更新 Gateway CLI 设计说明 |
| `docs/dev/README.md` | 增加 058 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 手动验证：临时配置文件执行 `gateway enable`、`gateway status`、`gateway platforms`、`gateway disable`、`gateway status -json` 均通过
- JSON 输出在没有已配置平台时保持 `configured_platforms: []`

### 来源：`0059-cli-sessions.md`

# 060 总结：CLI 会话列表与检索

## 变更摘要

新增 `agentd sessions list/search`，提供最小跨会话查看与检索入口。

## 新增能力

```bash
agentd sessions list
agentd sessions search hello
agentd sessions search -limit 50 -exclude session-id hello
```

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/store/session_store.go` | 增加 `ListRecentSessions` |
| `internal/store/session_store_test.go` | 增加最近会话列表测试 |
| `cmd/agentd/main.go` | 增加 `sessions list/search` CLI |
| `README.md` | 增加会话检索示例 |
| `docs/dev/README.md` | 增加 060 索引 |
| `docs/dev/060-*.md` | 新增 060 文档 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/store ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 手动验证：`sessions list -data-dir /tmp/agentd-sessions-home` 输出 JSON
- 手动验证：`sessions search -data-dir /tmp/agentd-sessions-home -limit 5 hello` 正常返回匹配消息

### 来源：`0060-cli-session-show-stats.md`

# 061 总结：CLI 会话详情查看与统计

## 变更摘要

新增 `agentd sessions show/stats`，补齐最小会话查看与统计能力，便于排障与外部系统读取会话元数据。

## 新增能力

```bash
go run ./cmd/agentd sessions show your-session-id
go run ./cmd/agentd sessions show -offset 200 -limit 200 your-session-id
go run ./cmd/agentd sessions stats your-session-id
```

## 修改文件

| 文件 | 变更 |
|------|------|
| `cmd/agentd/main.go` | 增加 `sessions show/stats` |
| `internal/store/session_store_test.go` | 增加 `LoadMessagesPage` / `SessionStats` 测试 |
| `README.md` | 增加示例 |
| `docs/dev/README.md` | 增加 061 索引 |
| `docs/dev/061-*.md` | 新增 061 文档 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过（本地）

### 来源：`0151-cli-gateway-setup.md`

# 152 - Summary: `gateway setup` minimal config writer

## Goal

Close the remaining CLI management gap by letting users write minimal gateway platform config without manually editing `config.ini`.

## What changed

- `cmd/agentd/main.go`:
  - `agentd gateway setup` adds per-platform config writing for `telegram` / `discord` / `slack` / `yuanbao`.
  - The command enables `gateway.enabled` and writes the minimal credential keys required by the selected platform.
  - Supports `-allowed-users`, `-file`, and `-json` for automation-friendly output.
- Documentation:
  - Refreshes product/dev parity docs to reflect existing minimal pairing/slash/queue/cancel/hooks support and the new setup entrypoint.

### 来源：`0152-cli-setup-entrypoint.md`

# 153 - Summary: `setup` unified bootstrap entrypoint

## Goal

Close the remaining CLI bootstrap gap by providing one non-interactive command that writes minimal provider and optional gateway config.

## What changed

- `cmd/agentd/main.go`:
  - Adds top-level `agentd setup`.
  - Supports writing provider/model/base-url/api-key/fallback-provider config in one step.
  - Optionally chains a gateway platform setup for `telegram` / `discord` / `slack` / `yuanbao`.
  - Reuses the same gateway config writer as `agentd gateway setup` to avoid config drift.
- Documentation:
  - Reframes the remaining gap from “no setup” to “no interactive setup wizard/update flow”.

### 来源：`0153-cli-setup-wizard.md`

# 154 - Summary: `setup wizard` interactive bootstrap

## Goal

Close the remaining human-facing bootstrap gap by adding an interactive terminal setup flow on top of the existing non-interactive `setup` command.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd setup wizard`.
  - Prompts for provider/model/base URL/API key/fallback provider and optional gateway platform credentials.
  - Reuses the same config application path as non-interactive `setup` and `gateway setup`.
- Documentation:
  - Updates parity docs to reflect that the remaining CLI gap is no longer “missing setup wizard”, but broader update/bootstrap completeness.

### 来源：`0154-cli-update-command.md`

# 155 - Summary: `update` minimal git-based flow

## Goal

Close the remaining CLI update gap for developer/git-checkout installs by adding a minimal command that can inspect and apply fast-forward updates.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd update` with `check` and `apply` modes.
  - `check` can optionally run `git fetch` and then reports branch, commit, upstream, ahead/behind counts, dirty state, and fast-forward eligibility.
  - `apply` runs `git fetch` + `git pull --ff-only` and reports the before/after commit IDs.
  - The implementation explicitly scopes itself to git checkouts in this minimal phase.
- Documentation:
  - Refreshes CLI parity docs to mark `update` as minimally available while keeping installer-level update flow as a remaining gap.

### 来源：`0155-cli-version-command.md`

# 156 - Summary: `version` command

## Goal

Close the remaining basic CLI maintenance gap by exposing a dedicated version command that can also report update status on git checkouts.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd version`.
  - Reports app version, release date, build commit, and Go runtime version.
  - Supports `-check-update` to reuse the minimal git-based update status check and `-json` for automation.
- Documentation:
  - Updates CLI parity docs so `version` and `update` are treated as a paired maintenance surface.

### 来源：`0156-cli-gateway-process-management.md`

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

### 来源：`0157-cli-bootstrap-command.md`

# 158-summary-cli-bootstrap-command

## 背景

虽然前几轮已经补齐 `setup`、`setup wizard`、`update`、`version` 和 `gateway` 管理面，但 CLI 仍缺一个最小“环境落盘/工作区骨架初始化”入口。文档里把这部分记成了安装器级 `bootstrap` 缺口，因此本轮先补一个不依赖外部安装器的最小版本。

## 本次实现

- 新增 `agentd bootstrap init`
  - 写入 `agent.workdir`
  - 写入 `agent.data_dir`
  - 创建工作目录下 `.agent-daemon/`
  - 创建数据目录、`processes/`、`MEMORY.md`、`USER.md`
- 新增 `agentd bootstrap status`
  - 检查 config/workdir/state_dir/data_dir/processes/MEMORY.md/USER.md 是否存在
  - 支持文本和 `-json` 输出
- 未传子命令时，`agentd bootstrap` 默认执行 `bootstrap init`

## 设计取舍

### 1. 不做安装器

本轮不是 system installer，不处理：

- shell profile 注入
- PATH 安装
- systemd / launchd 注册
- 二进制下载与升级

目标只是把“运行所需目录和基础文件”标准化，先收口当前仓库内可控的 bootstrap 差距。

### 2. 与运行时目录保持一致

初始化内容直接对齐现有运行时约定：

- `mustBuildEngine()` 依赖 `data_dir`
- `memory.Store` 依赖 `MEMORY.md`、`USER.md`
- `ProcessRegistry` 依赖 `processes/`
- Gateway 管理依赖 `<workdir>/.agent-daemon/`

因此 `bootstrap` 不是新体系，而是把已有隐式目录约定显式化。

## 验证

- `go test ./...`
- `go run ./cmd/agentd bootstrap status -json`
- `tmpdir=$(mktemp -d) && go run ./cmd/agentd bootstrap init -file "$tmpdir/config.ini" -workdir "$tmpdir/work" -data-dir "$tmpdir/data" -json`

## 文档更新

- README 增加 `bootstrap init/status` 示例
- 产品/开发总览从“缺 bootstrap”改为“已有最小 bootstrap，仍缺安装器级 update”

## 剩余差距

CLI/TUI 主线剩余缺口进一步收敛为：

- 全屏 TUI
- 安装器级 update
- 更完整的服务安装/守护管理
- Gateway 原生平台交互能力

### 来源：`0158-cli-gateway-install-uninstall.md`

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

### 来源：`0159-cli-update-install-uninstall.md`

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

### 来源：`0176-cli-gateway-slack-manifest-export.md`

# 177 总结：CLI 新增 Slack gateway manifest 导出命令

## 1. 背景

Slack 侧此前已经具备：

- 原生审批按钮
- slash 命令入口
- 通用 `/agent <cmd>` 转发

但仍有一个落地缺口：Slack slash command 与 app scopes 需要在 Slack app 后台手工配置，项目本身没有给出结构化导出结果，部署时容易漏项。

## 2. 本轮实现

### 2.1 新增 `agentd gateway manifest`

在 `cmd/agentd/main.go` 新增：

- `agentd gateway manifest -platform slack`
- `agentd gateway manifest -platform slack -command /agent`
- `agentd gateway manifest -platform slack -json`

当前仅支持 `slack` 平台。

### 2.2 导出内容

输出包含：

- `commands`：推荐 slash command 列表
- `app_manifest`：Slack app manifest 片段
- `next_actions`：后续配置提示
- `command_routes`：通用入口示例映射

manifest 中包含最小所需：

- `slash_commands`
- `commands` scope
- `chat:write`
- 历史消息读取 scope
- `socket_mode_enabled`
- `interactivity.is_enabled`

### 2.3 命令前缀可配置

通过 `-command /agent` 可指定通用 slash 入口前缀，导出的命令示例和路由映射会随之变化。

## 3. 结果

Slack Gateway 现在不只是“支持 slash command”，还具备最小安装配置导出能力，便于把代码能力落到真实 Slack app 配置中。

## 4. 验证

- `go test ./...`
- `go run ./cmd/agentd gateway manifest -platform slack -command /agent -json`

## 5. 文档同步

已更新：

- `README.md`
- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `docs/dev/README.md`

## 6. 仍未覆盖

本轮没有补：

- Slack app 的自动注册 / 自动应用 manifest
- 更完整原生 modal / form 流
- Yuanbao 原生命令菜单
- 更完整 token lock / 分布式协调

### 来源：`0177-cli-gateway-discord-manifest-export.md`

# 178 总结：CLI 新增 Discord gateway 命令清单导出命令

## 1. 背景

Slack 侧上一轮已经补了 `gateway manifest` 导出，但 Discord 侧仍缺少一个对外可见的配置清单出口：

- 代码里已经有原生 slash commands
- 运行时会自动 bulk overwrite 注册
- 但部署者看不到命令面、权限面和安装提示的结构化输出

因此本轮补一个最小的 Discord 导出能力，方便运维和部署前检查。

## 2. 本轮实现

### 2.1 `gateway manifest` 支持 `discord`

在 `cmd/agentd/main.go`：

- `agentd gateway manifest -platform discord -json`

现在 `manifest` 子命令支持：

- `slack`
- `discord`

### 2.2 复用现有命令定义

在 `internal/gateway/platforms/discord.go`：

- 将命令注册清单提升为导出函数 `DiscordApplicationCommands()`

CLI 导出直接复用这份定义，避免命令面出现两套来源。

### 2.3 导出内容

Discord 导出结果包含：

- `commands`：slash commands 与 option 描述
- `permissions`：建议 scopes
- `bot_permissions`：建议 bot 权限
- `install_url_hint`：OAuth2 安装 URL 模板
- `next_actions`：部署后续步骤

## 3. 结果

Gateway CLI 现在不只可导出 Slack manifest，也可以导出 Discord 命令清单，便于统一运维和部署检查。

## 4. 验证

- `go test ./...`
- `go run ./cmd/agentd gateway manifest -platform discord -json`

## 5. 文档同步

已更新：

- `README.md`
- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `docs/dev/README.md`

## 6. 仍未覆盖

本轮没有补：

- Discord 安装 URL 的自动生成（仍需外部填充 `APP_ID`）
- 更完整平台安装器 / 发布流
- Yuanbao 原生命令菜单
- 更完整 token lock / 分布式协调

### 来源：`0178-cli-gateway-telegram-manifest-export.md`

# 179 - Summary - CLI gateway Telegram manifest 导出

## 本次变更

- 新增 `agentd gateway manifest -platform telegram [-json]`，导出 Telegram `setMyCommands` 命令清单与 BotFather 可用的命令列表。
- `internal/gateway/platforms/telegram.go` 导出 `TelegramCommands()`，CLI 与运行时命令注册复用同一份定义，避免两套清单漂移。
- README、产品文档、开发文档同步更新 Telegram manifest 导出入口。

## 验证

- `go test ./...`
- `go run ./cmd/agentd gateway manifest -platform telegram -json`

## 结果

- Telegram 现在和 Slack / Discord 一样，具备最小命令清单导出能力。
- Gateway 平台对齐文档更新为“Telegram 已具备原生命令菜单 + 审批按钮 + manifest 导出”。

### 来源：`0179-cli-gateway-yuanbao-manifest-export.md`

# 180 - Summary - CLI gateway Yuanbao manifest 导出

## 本次变更

- 新增 `agentd gateway manifest -platform yuanbao [-json]`，导出 Yuanbao 所需环境变量、文本命令清单与快捷回复映射。
- 将 Gateway manifest 支持范围扩展为 `slack`、`discord`、`telegram`、`yuanbao` 四个平台。
- README、产品文档、开发文档同步更新 Yuanbao manifest 导出入口。

## 验证

- `go test ./...`
- `go run ./cmd/agentd gateway manifest -platform yuanbao -json`

## 结果

- Yuanbao 现在具备最小平台接入清单导出能力，便于按现有快捷回复闭环做落地配置。
- 多平台 Gateway manifest 导出能力已覆盖当前项目支持的四个主要平台。

### 来源：`0180-cli-update-release-command.md`

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

### 来源：`0181-cli-update-status-command.md`

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

### 来源：`0182-cli-update-install-script-expansion.md`

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

### 来源：`0183-cli-update-doctor-command.md`

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

### 来源：`0184-cli-update-changelog-command.md`

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

### 来源：`0185-cli-update-bundle-command.md`

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

### 来源：`0186-cli-update-bundle-inspect-command.md`

# 187 - Summary - CLI update bundle inspect 命令

## 本次变更

- `agentd update bundle` 现在支持子命令：默认 `build`，以及新增 `inspect`。
- 新增 `agentd update bundle inspect -file <bundle.tar.gz|manifest.json> [-json]`，用于读取本地 bundle / manifest，检查文件是否存在、manifest 是否匹配，以及归档 entry 数。
- README、产品文档、开发文档同步更新 bundle inspect 能力说明。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle inspect -file <tmp>/bundle.tgz -json`

## 结果

- 本地 release bundle 现在不仅能生成，也能被检查与校验，为后续分发/安装流程补上读取入口。

### 来源：`0187-cli-update-bundle-verify-command.md`

# 188 - Summary - CLI update bundle verify 命令

## 本次变更

- `agentd update bundle` 现在新增 `verify` 子命令。
- 新增 `agentd update bundle verify -file <bundle.tar.gz|manifest.json> [-strict] [-json]`，基于 inspect 结果输出 `status`、`issues`、`next_actions`。
- 支持 `-strict`，当 bundle 校验结果不是 `ok` 时返回非零退出码，便于 CI 或分发脚本接入。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle verify -file <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle verify -file /tmp/not-found.tgz -strict`

## 结果

- 本地 release bundle 现在具备可脚本化的 verify 入口，安装器级分发前的校验能力更完整。

### 来源：`0188-cli-update-bundle-unpack-command.md`

# 189 - Summary - CLI update bundle unpack 命令

## 本次变更

- `agentd update bundle` 现在新增 `unpack` 子命令。
- 新增 `agentd update bundle unpack -file <bundle.tar.gz> -dest <dir> [-json]`，可把本地 bundle 安全解包到目标目录。
- 解包时会校验归档路径，拒绝 `..` 或逃逸目标目录的 entry，避免路径穿越。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle unpack -file <tmp>/bundle.tgz -dest <tmp>/out -json`

## 结果

- update/release 链路现在具备最小本地解包能力，为后续 bundle 安装/回滚流程打基础。

### 来源：`0189-cli-update-bundle-apply-command.md`

# 190 - Summary - CLI update bundle apply 命令

## 本次变更

- `agentd update bundle` 现在新增 `apply` 子命令。
- 新增 `agentd update bundle apply -file <bundle.tar.gz|manifest.json> -dest <dir> [-json]`，可把本地 bundle 覆盖安装到目标目录。
- 应用前会扫描目标目录中将被覆盖的文件，并自动在 `<dest>/.agent-daemon/release-backups/` 生成一份 backup bundle 与 manifest，便于后续回滚流程接入。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小本地覆盖安装能力，并开始沉淀可复用的 backup bundle，为后续回滚命令打基础。

### 来源：`0190-cli-update-bundle-rollback-command.md`

# 191 - Summary - CLI update bundle rollback 命令

## 本次变更

- `agentd update bundle` 现在新增 `rollback` 子命令。
- 新增 `agentd update bundle rollback -dest <dir> [-file <backup.tar.gz|manifest.json>] [-json]`，可把 `update bundle apply` 生成的 backup bundle 回滚应用到目标目录。
- 当未显式传 `-file` 时，会自动选择 `<dest>/.agent-daemon/release-backups/` 下最新一份 backup bundle；回滚前也会再次生成当前目标状态的 backup bundle，避免回滚本身不可逆。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle rollback -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小本地回滚能力，bundle 分发闭环从“打包/校验/解包/覆盖安装”扩展到“可回退”。

### 来源：`0191-cli-update-bundle-backups-command.md`

# 192 - Summary - CLI update bundle backups 命令

## 本次变更

- `agentd update bundle` 现在新增 `backups` 子命令。
- 新增 `agentd update bundle backups -dest <dir> [-limit N] [-json]`，用于查看 `<dest>/.agent-daemon/release-backups/` 下最近的 backup bundle 列表。
- 输出会携带 backup bundle 路径、manifest 路径、生成时间、文件数、原始 source bundle 路径，便于在执行 `rollback` 前先查看可用回滚点。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle backups -dest <tmp>/target -json`

## 结果

- update/release 链路现在不仅能生成并消费 backup bundle，也能查询最近回滚点，最小本地运维面更完整。

### 来源：`0192-cli-update-bundle-prune-command.md`

# 193 - Summary - CLI update bundle prune 命令

## 本次变更

- `agentd update bundle` 现在新增 `prune` 子命令。
- 新增 `agentd update bundle prune -dest <dir> [-keep N] [-json]`，用于清理 `<dest>/.agent-daemon/release-backups/` 下过旧的 backup bundle。
- 清理时会同时删除 `.tar.gz` 与对应 `.json` manifest，只保留最近 `N` 份备份，避免 backup 目录无限增长。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle prune -dest <tmp>/target -keep 0 -json`

## 结果

- update/release 链路现在具备最小 backup 生命周期管理能力，本地回滚点不再只能累积、不能清理。

### 来源：`0193-cli-update-bundle-doctor-command.md`

# 194 - Summary - CLI update bundle doctor 命令

## 本次变更

- `agentd update bundle` 现在新增 `doctor` 子命令。
- 新增 `agentd update bundle doctor -dest <dir> [-strict] [-json]`，用于检查目标目录下 backup bundle 的数量、manifest 完整性与是否需要清理。
- 支持 `-strict`，当诊断结果不是 `ok` 时返回非零退出码，便于分发脚本或 CI 在执行 rollback 前先做健康检查。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle doctor -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小 bundle 运维诊断入口，build/apply/backups/prune/rollback 已能被统一检查。

### 来源：`0194-cli-update-bundle-status-command.md`

# 195 - Summary - CLI update bundle status 命令

## 本次变更

- `agentd update bundle` 现在新增 `status` 子命令。
- 新增 `agentd update bundle status [-file <bundle.tar.gz|manifest.json>] [-dest <dir>] [-json]`，用于聚合查看 bundle 校验状态、backup 列表、rollback 可用性与 backup doctor 结果。
- 当同时传入 `-file` 和 `-dest` 时，一次调用即可同时回答“bundle 能不能用”和“目标目录有没有可回滚点”。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle status -file <tmp>/bundle.tgz -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小 bundle 聚合状态入口，verify、doctor、backups、rollback readiness 已能统一查看。

### 来源：`0195-cli-update-bundle-manifest-command.md`

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

### 来源：`0196-cli-update-bundle-plan-command.md`

# 197 - Summary - CLI update bundle plan 命令

## 本次变更

- `agentd update bundle` 现在新增 `plan` 子命令。
- 新增 `agentd update bundle plan -file <bundle.tar.gz|manifest.json> -dest <dir> [-json]`，用于在执行 `apply` 前做 dry-run 规划。
- 输出会区分将要创建和将要覆盖的文件数量，并给出是否需要生成 backup 的预估，便于在真正落盘前评估风险。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle plan -file <tmp>/bundle.tgz -dest <tmp>/target -json`

## 结果

- update/release 链路现在具备最小 apply 前 dry-run 入口，可在分发脚本里先预估影响范围，再决定是否执行 apply。

### 来源：`0197-cli-update-bundle-rollback-plan-command.md`

# 198 - Summary - CLI update bundle rollback-plan 命令

## 本次变更

- `agentd update bundle` 现在新增 `rollback-plan` 子命令。
- 新增 `agentd update bundle rollback-plan -dest <dir> [-file <backup.tar.gz|manifest.json>] [-json]`，用于在执行 `rollback` 前做 dry-run 预演。
- 未显式传 `-file` 时会自动选择最近一份 backup bundle，并输出本次回滚将创建/覆盖多少文件，帮助先评估回滚影响范围。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle -out <tmp>/bundle.tgz -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle apply -file <tmp>/bundle.tgz -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle rollback-plan -dest <tmp>/target -json`

## 结果

- update/release 链路现在在 apply 与 rollback 两侧都具备最小预演入口，执行前的可见性更完整。

### 来源：`0198-cli-update-bundle-snapshot-command.md`

# 199 - Summary - CLI update bundle snapshot 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshot` 子命令。
- 新增 `agentd update bundle snapshot -dest <dir> [-out file] [-json]`，用于主动把目标目录导出为一份本地 restore point。
- 默认会把快照写到 `<dest>/.agent-daemon/release-backups/`，并自动排除该备份目录自身，避免快照递归打包已有 backups。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`

## 结果

- update/release 链路现在支持在 apply/rollback 之外主动创建目标目录快照，手工 restore point 能力更完整。

### 来源：`0199-cli-update-bundle-snapshots-command.md`

# 200 - Summary - CLI update bundle snapshots 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots` 子命令。
- 新增 `agentd update bundle snapshots -dest <dir> [-limit N] [-json]`，用于查看目标目录下手工创建的 snapshot 列表。
- 该命令只返回 `manual-snapshot` 类型的条目，和自动生成的 rollback backups 分开管理。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle snapshots -dest <tmp>/target -json`

## 结果

- update/release 链路现在既能创建 snapshot，也能查看现有 snapshot，手工 restore point 运维面更完整。

### 来源：`0200-cli-update-bundle-snapshots-prune-command.md`

# 201 - Summary - CLI update bundle snapshots-prune 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-prune` 子命令。
- 新增 `agentd update bundle snapshots-prune -dest <dir> [-keep N] [-json]`，用于清理目标目录下过旧的手工 snapshot。
- 该命令只处理 `manual-snapshot` 类型条目，不会删除自动生成的 rollback backups。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle snapshots-prune -dest <tmp>/target -keep 0 -json`

## 结果

- update/release 链路现在既能创建/查看 snapshot，也能单独清理手工 restore point，且不会误删回滚备份。

### 来源：`0201-cli-update-bundle-snapshots-doctor-command.md`

# 202 - Summary - CLI update bundle snapshots-doctor 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-doctor` 子命令。
- 新增 `agentd update bundle snapshots-doctor -dest <dir> [-strict] [-json]`，用于诊断目标目录下手工 snapshot 的健康状态。
- 该命令会检查手工 snapshot 数量、manifest 完整性，并在快照过多时建议执行 `snapshots-prune`。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshots-doctor -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle snapshots-doctor -dest <tmp>/empty -strict`

## 结果

- update/release 链路现在对手工 restore point 具备独立诊断能力，snapshot 创建、查看、清理、诊断形成闭环。

### 来源：`0202-cli-update-bundle-snapshots-status-command.md`

# 203 - Summary - CLI update bundle snapshots-status 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-status` 子命令。
- 新增 `agentd update bundle snapshots-status -dest <dir> [-limit N] [-json]`，用于聚合查看手工 snapshot 列表、诊断结果和最近可用 restore point。
- 输出包含 `snapshots`、`doctor`、`latest_snapshot_path`、`snapshot_ready`，避免手工快照运维需要多次调用分散命令。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle snapshots-status -dest <tmp>/target -json`

## 结果

- update/release 链路现在对手工 restore point 具备聚合状态入口，snapshot 创建、查看、诊断、清理已经形成最小闭环。

### 来源：`0203-cli-update-bundle-snapshots-restore-command.md`

# 204 - Summary - CLI update bundle snapshots-restore 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-restore` 子命令。
- 新增 `agentd update bundle snapshots-restore -dest <dir> [-file <snapshot.tar.gz|manifest.json>] [-json]`，用于把手工 snapshot 直接恢复到目标目录。
- 未显式传 `-file` 时会自动选择最新手工 snapshot；恢复时仍会生成 rollback backup，保证恢复过程可逆。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- 修改目标目录后执行 `go run ./cmd/agentd update bundle snapshots-restore -dest <tmp>/target -json`

## 结果

- update/release 链路现在对手工 restore point 具备直接恢复入口，snapshot 创建、查看、诊断、清理、恢复形成最小闭环。

### 来源：`0204-cli-update-bundle-snapshots-restore-plan-command.md`

# 205 - Summary - CLI update bundle snapshots-restore-plan 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-restore-plan` 子命令。
- 新增 `agentd update bundle snapshots-restore-plan -dest <dir> [-file <snapshot.tar.gz|manifest.json>] [-json]`，用于在恢复手工 snapshot 前做 dry-run 预演。
- 未显式传 `-file` 时会自动选择最新手工 snapshot，并输出本次恢复将创建/覆盖多少文件。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- 修改目标目录后执行 `go run ./cmd/agentd update bundle snapshots-restore-plan -dest <tmp>/target -json`

## 结果

- update/release 链路现在对手工 restore point 同时具备 restore 与 restore-plan，恢复前也能先做风险评估。

### 来源：`0205-cli-update-bundle-snapshots-delete-command.md`

# 206 - Summary - CLI update bundle snapshots-delete 命令

## 本次变更

- `agentd update bundle` 现在新增 `snapshots-delete` 子命令。
- 新增 `agentd update bundle snapshots-delete -dest <dir> [-file <snapshot.tar.gz|manifest.json>] [-json]`，用于定向删除指定手工 snapshot。
- 未显式传 `-file` 时会自动选择最新手工 snapshot，并同步删除对应 `.json` manifest。

## 验证

- `go test ./...`
- `go run ./cmd/agentd update bundle snapshot -dest <tmp>/target -json`
- `go run ./cmd/agentd update bundle snapshots-delete -dest <tmp>/target -json`

## 结果

- update/release 链路现在既支持批量 `snapshots-prune`，也支持定向删除单个 restore point，手工 snapshot 生命周期更完整。

### 来源：`0234-cli-align-ui-contract-semantics.md`

# CLI 语义对齐 UI 契约（Phase 19）

本轮将 `internal/cli`（`agentd chat/tui`）的命令输出语义与 `/v1/ui/*` 契约对齐，统一为结构化 `ok/error.code/error.message` 模式。

## 变更

- `internal/cli/chat.go` 新增统一输出助手 `printCLIEnvelope`：
  - 成功：`ok=true` + 业务字段
  - 失败：`ok=false` + `error.code` + `error.message`
  - 同步携带 `api_version` 与 `compat`
- 覆盖命令：
  - `/session`、`/tools`、`/sessions`、`/stats`、`/show`
  - `/reload`、`/clear`
  - 参数错误、能力不支持、未知命令等失败分支

## 结果

- API / ui-tui / CLI 三端错误语义一致，便于日志采集与自动化处理。

## 验证

- `go test ./internal/cli ./internal/api ./ui-tui`：通过

### 来源：`0261-cli-tui-standalone-auto-mode.md`

# 261-summary-cli-tui-standalone-auto-mode

本轮针对“CLI/TUI 差异项 1”做了可执行收口：`agentd tui` 不再固定走轻量内置循环，而是默认优先使用独立 `ui-tui`（完整命令面），并保留兼容回退。

## 变更

- `cmd/agentd/main.go`
  - `agentd tui` 新增参数：`-mode auto|standalone|lite`
  - 默认 `-mode auto`：
    - 优先启动独立 `ui-tui` 可执行文件
    - 若不可用，自动回退内置 lite TUI（原 `internal/cli/tui.go` 路径）
  - `-mode standalone`：强制独立 `ui-tui`，不可用即报错
  - `-mode lite`：强制内置 lite TUI
  - 新增 `resolveUITUIBinary()`：
    - 优先 `AGENT_UI_TUI_BIN`
    - 其次 `PATH` 中 `ui-tui`
    - 再次当前仓库 `ui-tui/tui.run`

- 测试
  - 新增 `cmd/agentd/main_tui_test.go`
    - 覆盖 `AGENT_UI_TUI_BIN` 分支
    - 覆盖本地 `ui-tui/tui.run` 候选路径分支

- 文档
  - `docs/frontend-tui-user.md`：增加 `agentd tui -mode` 使用说明
  - `docs/frontend-tui-dev.md`：增加运行模式与回退策略说明

## 验证

- `go test ./cmd/agentd -count=1`
- `go test ./...`

### 来源：`0262-cli-tui-source-fallback-and-boot-message.md`

# 262-summary-cli-tui-source-fallback-and-boot-message

本轮继续收口 CLI/TUI 差异项：提升 `agentd tui` 在“独立 ui-tui 不可执行”场景下的可用性，并补齐首条消息透传能力。

## 变更

- `cmd/agentd/main.go`
  - `agentd tui` 独立模式启动链路增强：
    - 先找二进制：`AGENT_UI_TUI_BIN` -> `PATH(ui-tui)` -> `./ui-tui/tui.run`
    - 若都不可用，新增源码回退：`go run ./ui-tui`（仓库根目录）
  - `agentd tui -message` 在独立 `ui-tui` 路径下生效：
    - 通过环境变量 `AGENT_UI_TUI_BOOT_MESSAGE` 透传给 `ui-tui`
  - 新增函数：
    - `buildUITUICommand`
    - `resolveUITUISourceDir`

- `ui-tui/main.go`
  - 启动时识别 `AGENT_UI_TUI_BOOT_MESSAGE`，自动发送首条对话。

- 测试
  - `cmd/agentd/main_tui_test.go`
    - 新增 `resolveUITUISourceDir` 分支测试。

- 文档
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`
  - `ui-tui/README.md`

## 验证

- `go test ./cmd/agentd -count=1`
- `go test ./ui-tui -count=1`
- `go test ./...`

### 来源：`0274-cli-tui-stateful-command-surface.md`

# 274 总结：CLI/TUI 状态化命令面补齐

## 背景

用户要求一次性完成 Hermes 差异清单中的第 1 项：CLI/TUI 体验补齐。当前不重写 Hermes 全量 TUI 前端，而是在现有 Go 入口内完成可验证的 CLI/TUI 核心命令闭环。

## 实现结果

- `internal/cli/chat.go` 引入 `chatState`，slash 命令可真实切换 `session_id`、重置/加载上下文，并保持 system prompt 与历史状态一致。
- 新增会话命令：`/new`、`/reset`、`/resume`、`/retry`、`/undo`、`/compress`、`/save`。
- 扩展管理命令：`/tools list|show|schemas`、`/toolsets list|show|resolve`、`/todo`、`/memory`、`/model`、`/status`、`/commands`。
- `internal/cli/tui.go` 的 lite TUI 事件输出扩展到 user、turn、model stream、assistant、tool、MCP、delegate、context compact、completed/error 等事件。
- 补充 CLI 单元测试覆盖新建/恢复、撤销/压缩、工具 schema 查看、重试等核心路径。

## 边界

- `/undo` 和 `/compress` 作用于当前进程内上下文，不回写删除 SQLite 历史，避免引入会话存储破坏性语义。
- `/retry` 会基于上一条 user 消息重新运行，并将新的上下文用于后续交互；持久化层会追加新的运行记录。
- 仍未复刻 Hermes 的完整高级 TUI 前端、模型选择器 UI、插件 slash 命令生态和全部快捷键。

## 验证

```bash
GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./...
```

结果：通过。

### 来源：`0015-doctor-summary-merged.md`

# 0015 doctor summary merged

## 模块

- `doctor`

## 类型

- `summary`

## 合并来源

- `0079-doctor-stub-tools.md`

## 合并内容

### 来源：`0079-doctor-stub-tools.md`

# 080 总结：doctor 增加 stub_tools 检查项

`agentd doctor` 新增 `stub_tools` 检查项：

- 当启用 `tools.enabled_toolsets` 时，若解析结果包含 browser/vision/tts 等 stub 工具，会给出 `warn` 提示（能力未实现）。
- 当未启用 toolsets 时，也会提示这些 stub 工具“仅接口对齐”。
