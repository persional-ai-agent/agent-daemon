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
