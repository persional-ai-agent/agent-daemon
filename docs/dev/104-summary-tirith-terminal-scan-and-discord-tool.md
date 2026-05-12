# 104 Summary - Tirith 终端预扫描 + Discord 工具名对齐

## 变更

1. Tirith（Hermes parity）：
   - `terminal` 工具在执行前可选调用外部 `tirith` 二进制进行安全扫描（`allow/warn/block`）。
   - `warn/block` 以“可审批”方式走现有 approval 流程（不会绕过 hardline 阻断）。

2. Discord 工具名对齐：
   - 新增 Hermes 风格 `discord` 工具（与现有 `discord_admin` 并存），提供最小 server introspection：
     - `list_guilds`
     - `server_info`
     - `list_channels`
     - `fetch_channel`
     - `fetch_messages`
     - `send_message`
     - `react`

## 配置与环境变量

Tirith（默认：仅当系统可找到 `tirith` 时自动启用）：

- `TIRITH_ENABLED=true|false`：显式开关（默认：未设置时按是否存在二进制自动启用）
- `TIRITH_BIN=tirith`：二进制路径/命令名
- `TIRITH_TIMEOUT=5`：扫描超时（秒）
- `TIRITH_FAIL_OPEN=true|false`：tirith 不可用/异常时是否放行（默认 true）

Discord：

- `DISCORD_BOT_TOKEN`：Discord Bot Token

## 修改文件

- `internal/tools/tirith.go`
- `internal/tools/builtin.go`（`terminal` 增加 tirith pre-scan gate）
- `internal/tools/discord_tool.go`
- `internal/tools/builtin.go`（注册 `discord` 工具）
- `internal/tools/toolsets.go`（新增 `discord` toolset）
